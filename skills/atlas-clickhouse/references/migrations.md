# ClickHouse Migrations: Lint, Rebuilds, Clusters, and Transactions

How Atlas changes a ClickHouse schema that cannot be altered in place, what it leaves out of a plan, and
how it runs migrations on a database without transactions or locks. Tests are in
`references/testing.md`. The ClickHouse driver, lint, and tests need `atlas login`. If `atlas whoami`
shows no login, ask the user to run `atlas login`; a user without an account can start a free trial
with it.

## What Lint Catches on ClickHouse

The ClickHouse checks report changes whose cost grows with the size of the table, not with the size of
the change (https://atlasgo.io/lint/analyzers). They are reported under
`blocking table changes detected`:

| Code | Reports |
|------|---------|
| `CH301` | A column type change on an existing table: `Changing column "user_id" from "Int32" to "String" rewrites it in every existing part of table "events"`. It runs as a mutation |
| `CH302` | An `EXCHANGE TABLES` statement: `Rebuilding table "events" copies all its rows, and rows written to it after the copy are dropped with it` |
| `CH303` | `MATERIALIZE INDEX` or `MATERIALIZE PROJECTION`: `Materializing index "idx_user" builds it over every existing part of table "events"` |
| `DS101`, `DS102`, `DS103` | Dropping a database, a table, or a column. At server scope, `DS101` reports an undeclared `default` database |
| `MF104` | Making a `Nullable` column non-nullable when it may hold `NULL` |

- `CH301` also fires for a nullability change, such as `Nullable(Int64)` to `Int64`, next to `MF104`.
  It skips a table that the same file creates.
- Adding a column only touches metadata, and lint reports nothing.
- `CH303` fires for a changed index or projection too: Atlas drops it, adds it, and materializes it.
- `CH302` and `CH303` read the statements of the file, so they also report hand-written
  `EXCHANGE TABLES` and `MATERIALIZE` statements. `CH302` reports nothing for the rebuilds that use
  `DROP TABLE` and `RENAME TABLE` (Table Rebuilds below), which lose late writes the same way.
  Hand-written `ALTER TABLE ... UPDATE` or `DELETE` mutations have no check either.
- The `CH3*` checks are warnings: `atlas migrate lint` exits successfully with them. The other generic
  analyzers (`CD*`, `MF*`, `NM*`) run as on other databases.

Tune a check in the `lint` block, for example to fail on table rebuilds:

```hcl
lint {
  check "CH302" {
    error = true
  }
}
```

A statement can opt out of named checks with a directive, with the user's approval
(https://atlasgo.io/versioned/lint#nolint-directive):

```sql
-- atlas:nolint CH301
ALTER TABLE `events` MODIFY COLUMN `user_id` String;
```

### Custom rules

When no analyzer above covers a check, or the user states a convention, such as replicated engines or
a TTL on every table, write a custom rule in the HCL rule language. A rule can check any attribute of
any object Atlas inspects: `references/custom-rules.md`.

## Table Rebuilds

ClickHouse cannot alter a table's sort key, partition key, primary key, or engine. For a MergeTree
table, Atlas rebuilds it in four statements:

```sql
CREATE TABLE `events_tmp` (...) ENGINE = MergeTree PRIMARY KEY (`id`) ORDER BY (`id`, `name`) ...;
INSERT INTO `events_tmp` (`id`, `name`, `payload`) SELECT `id`, `name`, `payload` FROM `events`;
EXCHANGE TABLES `events` AND `events_tmp`;
DROP TABLE `events_tmp`;
```

- The copy lists every column but the `MATERIALIZED`, `ALIAS`, and `EPHEMERAL` ones and the columns the
  change adds, and reads a renamed column under its old name.
- Rows written to the table after the `INSERT ... SELECT` stay in the old table and are dropped with it
  (`CH302`). Stop writes to the table for the rebuild, or schedule it.
- The copy needs disk space for both tables, and time proportional to the table.
- `EXCHANGE TABLES` needs an `Atomic` or `Replicated` database, and ClickHouse 21.4. Elsewhere, Atlas
  drops the table and renames the copy, which leaves a moment without the table. In cluster mode,
  converting a table to a `Replicated*` engine also uses `DROP TABLE IF EXISTS` and `RENAME TABLE`,
  since the table may exist on one node only.
- Engines whose data lives elsewhere (`Distributed`, `Merge`, `Buffer`, `Null`, `S3`, `URL`, `MySQL`,
  and the like) and integration engines (`Kafka`, `RabbitMQ`, `NATS`, `S3Queue`, `FileLog`) are dropped
  with `SYNC` and created again instead. So is `KeeperMap`, whose rows can be lost
  (`references/objects.md`, Table Engines).
- For a table too large to copy, write the change by hand in a migration file, for example a new table
  filled by a materialized view, and swap it in later.

## Changes Skipped Without a Terminal

Some changes can be applied only by dropping data. In a terminal, Atlas asks first, with `Abort` as the
alternative. Without one, when `CI` is set or stdout is not a terminal, as in CI and most agent shells,
Atlas leaves these changes out of the plan, without an error or a warning:

- A column change ClickHouse rejects in place: an `Enum` whose existing values change their numbers, a
  smaller `FixedString`, or an unsupported conversion (`references/objects.md`, Columns). The prompt
  offers `Drop and recreate column "<name>" (data loss)`. A type change on a key, skip index, or
  projection column is not skipped: it fails the plan instead (`references/objects.md`, Columns).
- A sort key, partition, primary key, or engine change on a `Replicated*` table whose engine holds
  quoted arguments, such as an explicit ZooKeeper path:

  ```
  Table "events" must be recreated for this change, and its explicit ZooKeeper path cannot be shared with a replacement holding its rows
  ```

- Any change to a `TimeSeries` table:
  `Table "ts" must be recreated for this change, and the data of its inner tables is dropped with it`.

The rest of the plan still runs. A diff that holds only skipped changes reports
`The migration directory is synced with the desired state, no changes to be made`, or
`Schema is synced, no changes to be made`. After every diff, check each edit against the plan. When one
is missing, tell the user, and either write the migration by hand with `atlas migrate new`, or ask the
user to run the command in a terminal and choose the recreate. Choosing `Abort` fails with the reason,
such as `Enum conversion changes value for column "status"`, or
`Recreate the table manually to apply this change`.

## Mutations

A column type change is planned as `ALTER TABLE ... MODIFY COLUMN`, which ClickHouse runs as a mutation
that rewrites the column in every part (`CH301`). A new skip index or projection is followed by
`ALTER TABLE ... MATERIALIZE INDEX` or `MATERIALIZE PROJECTION`, so it covers existing parts (`CH303`).

- Several changes to one table share one `ALTER TABLE` statement. `RENAME COLUMN`, `MODIFY SETTING`,
  and `RESET SETTING` get their own statements. A changed `CHECK` is dropped and added again.
- Seed data updates and deletes carry `SETTINGS mutations_sync = 1`, so they wait for the mutation
  (`references/hcl.md`, Seed Data). Hand-written `ALTER TABLE ... UPDATE` or `DELETE` in a migration
  file do not: add `SETTINGS mutations_sync = 1` (or `2` on a cluster), or later statements and tests
  can see the old data.

## Diff Policies

The env's `diff` block shapes every plan (https://atlasgo.io/versioned/diff):

- `skip` leaves change kinds out of the plan: `drop_schema`, `drop_table`, `drop_column`, `drop_index`,
  `drop_view`, `modify_view` (views and materialized views), `drop_func`, `add_func`, `modify_func`,
  `drop_user`, `drop_role`, and the other generic kinds. It has no option for dictionaries, named
  collections, row policies, settings profiles, or WASM modules.
- `add_table`, `add_column`, and `add_index` take `if_not_exists = true`, and `drop_table` takes
  `if_exists = true`. `drop_table.cascade` and `materialized.with_no_data` have no effect on ClickHouse.
- The `diff "clickhouse"` block adds a `cluster` policy and `session_settings` policies:

```hcl
env "prod" {
  diff "clickhouse" {
    cluster {
      name = "{cluster}"
    }
    session_settings {
      match  = "app.*"
      values = {
        allow_nullable_key = 1
      }
    }
  }
}
```

- `cluster` writes ``ON CLUSTER `<name>` `` on planned DDL, with the name as given, plus `SYNC` drops,
  `SETTINGS alter_sync = 2`, and the cluster-mode rebuild into `Replicated*`. It does not cover
  `ALTER USER`, `ALTER ROLE`, user and role moves, the revisions table, or seed data quorum settings;
  `mode=cluster` on the URL covers them all (Cluster Mode below).
- `session_settings` adds `values` to the `SETTINGS` clause of each table the plan creates whose name
  matches `match`, a glob: on the table name in any database, or on `database.table` when it holds a
  dot. Values override settings of the same name. A new table without any settings, such as a
  non-MergeTree table, gets nothing. A bad glob fails with `syntax error in pattern`.

## Transactions

ClickHouse has no transactional DDL: every statement commits on its own. Atlas treats every migration
file as non-transactional, and cannot roll back a file, or a declarative plan, that fails halfway.

- `--tx-mode` does not change this.
- A failed file shows as partially applied in `atlas migrate status`. The next `atlas migrate apply`
  continues from the statement that failed: fix that statement and the ones after it, leave the applied
  ones unchanged, and run `atlas migrate hash`.
- To undo a partially applied file instead, run `atlas migrate down` before editing it, which reverts
  the applied statements one at a time (https://atlasgo.io/versioned/down). Once the file is edited
  and rehashed, `down` refuses it with
  `migration file with version "<version>" has been modified since it was applied`. A column recreate
  has no reverse statements (`references/objects.md`, Columns).
- `atlas migrate down` checks that a table is empty before dropping it, and that a column holds no
  values before dropping it, and stops with `check assertion "SELECT NOT EXISTS (...)" returned false`
  otherwise. The column check is `WHERE <column> IS NOT NULL`, and a column that is not `Nullable` is
  never `NULL` in ClickHouse: reverting an `ADD COLUMN` on a table with rows always fails it, even when
  the column holds only defaults. Confirm with the user that the values can go, then run
  `atlas migrate down --skip-checks`.
- When the schema and the revisions table disagree, `atlas migrate set <version>` updates the revisions
  table only.
- `schema apply` has the same limit: the statements before a failure stay applied.
- Keep files small, and test them before applying (`references/testing.md`).

## Locks

Atlas takes no lock on ClickHouse: `migrate apply` and `schema apply` do not wait for each other, and
`--lock-timeout`, `--lock-name`, and `--skip-lock` change nothing. Run one deployment per target at a
time, for example with a CI concurrency group.

## Cluster Mode

Add `mode=cluster` to the URL and the dev URL (https://atlasgo.io/guides/clickhouse/cluster-mode):

```hcl
env "prod" {
  url = "clickhouse://user:pass@ch-host:9000/app?mode=cluster"
  dev = "clickhouse://user:pass@ch-host:9000/app_dev?mode=cluster"
}
```

- Every DDL statement, `GRANT`, `REVOKE`, `ALTER USER`, and `ALTER ROLE` carries
  ``ON CLUSTER `{cluster}```. Exceptions: databases with the `Replicated` engine, `CREATE` and `DROP` of
  users and roles in the `replicated` storage, and UDFs when both URLs set
  `has_user_defined_zookeeper_path=true`.
- `DROP` statements carry `SYNC`, and `ALTER TABLE` statements carry `SETTINGS alter_sync = 2`, so each
  waits for every replica.
- Write the engine of each table yourself, such as `sql("ReplicatedMergeTree")` or
  `sql("Distributed('{cluster}', 'app', 'events_local')")`. Cluster mode does not change engines.
- The dev database must be on a cluster (`references/dev-database.md`, Clusters).

## Session Settings

A setting a statement needs, such as `allow_suspicious_low_cardinality_types`, can come from four places
(https://atlasgo.io/guides/clickhouse/session-settings):

- A URL parameter, which applies to the whole session: set it on the URL and the dev URL.
- A `SET` statement in a hand-written migration file, which applies to the statements after it.
- A settings profile assigned to the user Atlas connects as.
- The `session_settings` diff policy, for new tables only (Diff Policies above).

## Revisions Table

- `atlas_schema_revisions` lives in the database the URL names. With a server-scoped URL, it gets a
  database of its own, `atlas_schema_revisions`. `--revisions-schema`, or `revisions_schema` in the
  env's `migration` block, moves it.
- It is a MergeTree table ordered by `version`. In cluster mode it is a `ReplicatedMergeTree` created
  `ON CLUSTER`, and Atlas writes it with `insert_quorum='auto'` and `mutations_sync=2`. Turning on
  cluster mode later converts the existing table on the next run; turning it off does not convert it
  back.
- On a versioned database, exclude it from `schema diff` with `--exclude atlas_schema_revisions`.

## Existing Databases

A database that has tables but no revisions table is refused until it gets a baseline: the first
migration, recorded as applied without running (https://atlasgo.io/versioned/apply). Generate it with a
dev URL of the same scope:

```bash
atlas migrate diff baseline \
  --dir "file://migrations" \
  --dev-url "docker://clickhouse/24.10/app" \
  --to "clickhouse://user:pass@localhost:9000/app"
```

Then pass its version once, `atlas migrate apply --baseline "<version>"`, or set `baseline` in the env's
`migration` block.

## Seed Data

Seed data is declared in HCL, with the modes, `max_rows`, and primary key rules in
`references/hcl.md`, Seed Data.

## Large Changes

Nothing in Atlas batches a single huge plan for you. To spread a large change over several deployments,
split it into several migration files and apply a few at a time with `atlas migrate apply <n>`. For a
rebuild of a large table, see Table Rebuilds above.
