# Spanner Migrations: Transactions, Long-Running Changes, and Migration Files

How Atlas runs migrations on a database without transactional DDL, how to keep Spanner's long-running
schema changes under control, and what to write by hand. Tests are in `references/testing.md`. The
Spanner driver, lint, custom rules, and tests need `atlas login`. If `atlas whoami` shows no login, ask
the user to run `atlas login`; a user without an account can start a free trial with it.

## Every File Starts with `-- atlas:txmode none`

Spanner cannot run DDL inside a transaction, and `migrate apply` wraps each file in one unless the file
opts out. `atlas migrate diff` writes the directive at the top of every file it generates:

```sql
-- atlas:txmode none

-- Create "Singers" table
CREATE TABLE `Singers` (
  `SingerId` STRING(36) NOT NULL,
  `Name` STRING(MAX) NOT NULL
) PRIMARY KEY (`SingerId`);
```

- `atlas migrate new` creates an empty file without it. Add it to every hand-written file, or the file
  fails on the target with
  `spanner: code = "FailedPrecondition", desc = "cannot execute DDL as part of a transaction"`.
- `atlas migrate lint` and `atlas migrate validate` accept a file without the directive.
  `atlas migrate test` fails on it, and so does applying it to a local emulator
  (`references/dev-database.md`).
- `atlas migrate apply --tx-mode none` turns transactions off for a whole run.
- `atlas schema clean` always runs in a transaction and fails on Spanner with the same error. Reset a
  database by applying an empty schema with `atlas schema apply`, or drop and recreate it with `gcloud`.

## How Atlas Runs Statements

Atlas sends one statement at a time, and Spanner runs each DDL statement as its own schema change,
a long-running operation on the server. Data statements in the file commit one by one. The Spanner
guide's run takes 13 to 17 seconds per statement against a real instance
(https://atlasgo.io/guides/spanner/automatic-migrations).

- A file or a declarative plan that fails halfway stays half applied. `atlas migrate status` shows a
  failed file as partially applied:

  ```text
  Migration Status: PENDING
    -- Current Version: 20261001000003 (1 statements applied)
    -- Next Version:    20261001000003 (2 statements left)
    -- Executed Files:  3 (last one partially)
    -- Pending Files:   1

  Last migration attempt had errors:
    -- SQL:   ALTER TABLE `Accounts` ALTER COLUMN `Nick` STRING(64) NOT NULL
    -- ERROR: rpc error: code = FailedPrecondition desc = Cannot specify a null value for column: Nick in table: Accounts referenced by key: {Int64(1)}
  ```

- The next `atlas migrate apply` continues from the statement that failed. Fix that statement and the
  ones after it, leave the applied ones unchanged, and run `atlas migrate hash`.
- `atlas migrate down` reverts applied files on Spanner, statement by statement. It computes the revert
  on the dev database and checks that a column it drops holds no data
  (https://atlasgo.io/versioned/down).
- `atlas schema apply` stops at the failed statement, and the next plan lists what is left.
- A deployment job that dies during a long statement, such as an index build, does not stop the schema
  change on Spanner. Check it with
  `gcloud spanner operations list --instance=<instance> --database=<database> --type=DATABASE_UPDATE_DDL`
  before running the file again; the rerun fails with a duplicate name if the statement finished. The
  `if_not_exists` diff policy writes `ADD COLUMN IF NOT EXISTS` and `CREATE INDEX IF NOT EXISTS`, which a
  rerun skips (https://atlasgo.io/versioned/diff#diff-policy):

  ```hcl
  env "prod" {
    diff {
      add_column {
        if_not_exists = true
      }
      add_index {
        if_not_exists = true
      }
    }
  }
  ```

## Long-Running Changes

Some statements make Spanner read every row of a table, and run from minutes to hours on a large one.
Reads and writes continue meanwhile, but conflicting schema changes wait
(https://docs.cloud.google.com/spanner/docs/schema-updates#updates-that-require-validation):

| Statement | What Spanner does |
|-----------|-------------------|
| `CREATE INDEX` on a populated table | Backfills the index |
| `ADD COLUMN` with a stored generated expression | Backfills the column, which cannot be read until done |
| `ALTER COLUMN ... NOT NULL`, a shorter `STRING(n)` or `BYTES(n)`, `BYTES` to `STRING` | Validates every row; rejects new writes that break the change at once |
| `ADD CONSTRAINT ... CHECK` or `FOREIGN KEY` | Validates every row; enforces the constraint on new writes at once |
| `allow_commit_timestamp = true` on an existing column | Validates that every value is in the past |

Google's guidance for these
(https://docs.cloud.google.com/spanner/docs/schema-updates-best-practices):

- Put no more than 10 of them in one batch, and add no more than 3 indexes that need a backfill per day.
- Spanner throttles a database that receives many schema changes in a short time.
- Create a table's indexes together with the table, while it is empty.

With Atlas, where every statement is its own schema change:

- Keep these statements in small files of their own, apart from other DDL and from data changes.
- Spread index builds over deployments: `atlas migrate apply <n>` applies the next `n` files.
- An initial schema of hundreds of statements takes as many schema changes. To load it as one batch,
  create the database with `gcloud spanner databases ddl update <database> --instance=<instance>
  --ddl-file=schema.sql`, then record the baseline migration as applied (Existing Databases, below).
- Watch and cancel a running change with `gcloud spanner operations list` and
  `gcloud spanner operations cancel`. If `migrate apply` is still waiting on the cancelled statement, the
  file fails there: fix it and apply again.

## What Lint Catches on Spanner

Spanner has no analyzers of its own, like the `MY` and `PG` checks of MySQL and PostgreSQL. The generic
ones run (https://atlasgo.io/lint/analyzers):

| Code | Reports |
|------|---------|
| `DS101` to `DS103` | Dropping a schema, a table, or a column that holds data |
| `BC101` to `BC104` | Renaming or dropping a table or column that clients use |
| `MF101` | A unique index on existing data, which fails on duplicates |
| `MF104` | A nullable column made `NOT NULL`, which fails on `NULL` rows |

Lint does not report a shorter `STRING(n)`, a foreign key or check added to a populated table, an index
backfill, or a recreated change stream. It does replay each file on the dev database, so a statement
the emulator rejects, such as a `DROP COLUMN` before the index that uses it, fails lint instead of the
deployment. Cover the rest with custom rules (`references/custom-rules.md`) and a `statement` rule
(https://atlasgo.io/lint/analyzers#statement-rules):

```hcl
lint {
  statement {
    deny "drop-change-stream" {
      match   = "(?is)^\\s*DROP\\s+CHANGE\\s+STREAM\\b"
      message = "DROP CHANGE STREAM deletes the stream's change records: change its options with ALTER CHANGE STREAM ... SET OPTIONS"
    }
  }
}
```

It reports each `DROP CHANGE STREAM` as a warning; set `error = true` in the `statement` block to fail
the run. A statement opts out of named checks with a directive on the line before it
(https://atlasgo.io/versioned/lint#nolint-directive):

```sql
-- atlas:nolint DS102 BC103
DROP TABLE `LegacyEvents`;
```

## Dependent Objects

Spanner refuses to drop a column that an index, a `STORING` list, a view, a foreign key, a check, a
generated column, or a row deletion policy uses
(https://docs.cloud.google.com/spanner/docs/schema-updates#supported-updates). When a change removes a
column together with the object that uses it, Atlas plans the `DROP COLUMN` first:

```sql
-- Drop column "Nick" from table "Users"
ALTER TABLE `Users` DROP COLUMN `Nick`;
-- Drop index "UsersByNick" from table "Users"
DROP INDEX `UsersByNick`;
```

```
Cannot drop column Nick from table Users because it is used by index UsersByNick.
Cannot drop column `Nick` on which there are dependent views: SingerNames.
```

`atlas migrate diff` writes the file without an error. `atlas migrate lint`, `atlas migrate validate`,
and `atlas schema apply` fail on it, because they replay it on the dev database. Move the `DROP INDEX`,
`ALTER INDEX ... DROP STORED COLUMN`, or `CREATE OR REPLACE VIEW` above the `DROP COLUMN` and run
`atlas migrate hash`, or remove the dependent object in one migration and the column in the next. In the
declarative workflow, apply the two steps one after the other.

## Changes to Write by Hand

- Change streams. Atlas recreates a stream for any change, which deletes its change records. Write the
  change instead, and keep the schema in step:

  ```sql
  -- atlas:txmode none

  ALTER CHANGE STREAM SingersBio SET FOR Singers(Name);
  ALTER CHANGE STREAM EventsStream SET OPTIONS (retention_period = '3d');
  ```

- Interleaved indexes and `NOT ENFORCED` foreign keys, which Atlas plans without those clauses
  (`references/objects.md`, What Atlas Leaves Out).
- Database options, which Atlas leaves out of plans:

  ```sql
  -- atlas:txmode none

  ALTER DATABASE `orders-db` SET OPTIONS (version_retention_period = '3d');
  ```

  `default_sequence_kind` cannot be changed once set, and `default_time_zone` only on an empty database.
- A type change Spanner cannot make in place: add the new column, backfill it, switch the application,
  then drop the old column in a later file.
- A primary key change, or adding or removing interleaving: create the new table, copy the rows, switch
  the application, then drop the old table in a later file.

## Renames

Atlas plans a renamed table as a drop and a create, which loses the rows (`DS102`), and Spanner HCL has
no `renamed_from`. Spanner renames tables, not columns.

1. Write the rename in a migration file. A synonym keeps the old name working for queries until the
   application moves to the new one; Atlas ignores synonyms. Lint reads the rename as a dropped table and
   fails on it, so after checking it, silence `DS102` and `BC103`:

   ```sql
   -- atlas:txmode none

   -- atlas:nolint DS102 BC103
   ALTER TABLE Legacy RENAME TO Accounts, ADD SYNONYM Legacy;
   ```

2. Rename the table in the schema. The next `migrate diff` reports no changes.
3. Once no query uses the old name, drop the synonym by hand: `ALTER TABLE Accounts DROP SYNONYM Legacy`.

In the declarative workflow, apply the rename on its own, before other changes: change only the table
name in the schema, run `atlas schema apply --env <name> --edit`, and replace the planned `CREATE TABLE`
and `DROP TABLE` with the `ALTER TABLE ... RENAME TO` statement (the `atlas` skill,
`references/declarative.md`, Editing a plan). Check the edited plan before approving it: a plan that
still drops the old table loses its rows.

A column cannot be renamed: add a new column, backfill it, and drop the old one in a later file.

## Data Changes

- Each `INSERT`, `UPDATE`, or `DELETE` in a migration file commits on its own. One DML statement makes
  up to 80,000 mutations, counting every column it writes and every index entry it changes, and a
  larger one fails with a `BadUsage` error (https://docs.cloud.google.com/spanner/docs/dml-tasks). An
  `UPDATE` over a large table exceeds it in production, while the emulator does not enforce the limit,
  so a migration test passes.
- Backfill large tables in batches with `atlas script loop`, which commits each batch on its own
  (https://atlasgo.io/scripts/loop). Spanner takes `@p1` placeholders, and `JSON_VALUE_ARRAY` expands a
  JSON list:

  ```hcl
  script "loop" "backfill_email_norm" {
    iterator "keyset" {
      cursor {
        id = int
      }
      batch {
        id = int
      }
      init {
        sql = "SELECT Id AS id FROM Users WHERE EmailNorm IS NULL ORDER BY Id LIMIT ${ramp.size}"
      }
      next {
        sql  = "SELECT Id AS id FROM Users WHERE EmailNorm IS NULL AND Id > @p1 ORDER BY Id LIMIT ${ramp.size}"
        args = [cursor.id]
      }
    }
    do {
      exec {
        sql  = "UPDATE Users SET EmailNorm = LOWER(TRIM(Email)) WHERE Id IN UNNEST(ARRAY(SELECT CAST(v AS INT64) FROM UNNEST(JSON_VALUE_ARRAY(@p1)) AS v))"
        args = [jsonencode(iterator.keyset.batch[*].id)]
      }
    }
    assert "none_left" {
      sql = "SELECT COUNT(*) = 0 FROM Users WHERE EmailNorm IS NULL"
    }
    policy {
      ramp {
        stage { size = 1000 }
      }
    }
  }
  ```

  Alias the cursor column to the name the `cursor` block uses (`Id AS id`). Run it with
  `atlas script loop --env <name> --file file://backfill.hcl --run backfill_email_norm`.
- Or run a single statement as Partitioned DML, outside Atlas:
  `gcloud spanner databases execute-sql <database> --instance=<instance> --enable-partitioned-dml --sql="UPDATE ..."`.
- Seed rows go in migration files as `INSERT` statements: Spanner HCL has no `data` block.

## Revisions Table

- Atlas keeps the revisions table in a named schema, `atlas_schema_revisions`, which it creates on the
  first `migrate apply`. `atlas schema inspect` shows it.
- A declarative plan on the same database drops it, unless the env excludes it:
  `exclude = ["atlas_schema_revisions"]`.

## Existing Databases

A database that has objects but no revisions table is refused until it gets a baseline, the first
migration recorded as applied without running (https://atlasgo.io/versioned/apply#existing-databases):

```
Error: sql/migrate: connected database is not clean: found schema "default". baseline version or allow-dirty is required
```

```bash
atlas migrate diff baseline \
  --dir "file://migrations" \
  --dev-url "docker://spanner/1.5.58" \
  --to "spanner://projects/<project>/instances/<instance>/databases/<database>"
```

Then pass its version once with `atlas migrate apply --baseline <version>`, or set `baseline` in the
env's `migration` block. The baseline holds only what Atlas plans: add identity columns, interleaved
indexes, `NOT ENFORCED` foreign keys, and database options to it by hand, so new databases get them
too (`references/objects.md`). Proto bundles cannot go in a migration file: create them with `gcloud`
before the first `migrate apply` on a new database.

## Large Changes

To spread a large change over several deployments, split it into several migration files and apply a few
at a time with `atlas migrate apply <n>`.
