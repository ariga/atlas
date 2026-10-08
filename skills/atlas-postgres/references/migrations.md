# PostgreSQL Migrations: Lint, Locks, and Transactions

How to change a PostgreSQL schema without blocking traffic, and how Atlas runs migrations on PostgreSQL.
Tests are in `references/testing.md`. Lint, custom rules, the `not_null` policy, and seed data need
`atlas login`. If `atlas whoami` shows no login, ask the user to run `atlas login`; a user without an
account can start a free trial with it.

## What Lint Catches on PostgreSQL

| Code | Catches | Fix |
|------|---------|-----|
| `PG101`, `PG102` | Creating or dropping an index without `CONCURRENTLY` | Concurrent Indexes below |
| `PG103` | A concurrent index change in a file without `-- atlas:txmode none` | Concurrent Indexes below |
| `PG104`, `PG105` | Adding a primary key or unique constraint, which takes `ACCESS EXCLUSIVE` | Concurrent Indexes below |
| `PG108` | An index on a partitioned table, which blocks writes on every partition | Concurrent Indexes below |
| `PG109` | Adding an `EXCLUDE` constraint, which locks and scans the table | Concurrent Indexes below |
| `PG110` | Column order that wastes space to alignment padding | Reorder the columns of a new table |
| `PG301` | A column type or collation change, including a `char(n)` length change, that rewrites the table or an index | Rewrites and Scans below |
| `PG302` | A new column with a volatile default, which rewrites the table | Rewrites and Scans below |
| `PG303` | `SET NOT NULL` on an existing column, which scans the table under a lock | Lock-Safe NOT NULL below |
| `PG304` | A primary key on nullable columns, which sets them `NOT NULL` and scans the table | Rewrites and Scans below |
| `PG305`, `PG306` | A `CHECK` or foreign key added without `NOT VALID`, which scans the table | Rewrites and Scans below |
| `PG307` | `SET LOGGED` or `SET UNLOGGED`, which rewrites the table | Rewrites and Scans below |
| `PG308` | A trigger on an existing table, which blocks writes while it is added | Rewrites and Scans below |
| `PG309`, `PG310`, `PG311` | A stored generated column, an identity column, or a new access method, each rewriting the table | Rewrites and Scans below |
| `PG312` | Redefining a primary key | Rewrites and Scans below |
| `PG314` | Changing `REPLICA IDENTITY` to `FULL` or `NOTHING`, which risks logical replication | Check the replication setup first |
| `PG320` | Turning autovacuum off for a table, which lets dead rows accumulate | Keep autovacuum on |
| `TX101`, `TX201` | Statements that cannot share a transaction; `BEGIN`/`COMMIT` in a file Atlas already wraps | Transactions below |
| `MF104` | `SET NOT NULL` on a column that may hold `NULL` | Lock-Safe NOT NULL below |
| `SA101` | Dynamic SQL in a PL/pgSQL function, a possible SQL injection | Review how the function builds the statement |

Each check is tuned in the `lint` block (https://atlasgo.io/lint/analyzers#check-policy):

```hcl
lint {
  check "PG301" {
    error = true
  }
  check "PG304" {
    skip = true
  }
}
```

### Custom rules

When no analyzer above covers a check, or the user states a convention, such as row-level security on
every table or an index on every foreign key, write a custom rule in the HCL rule language. A rule can
check any attribute of any object Atlas inspects: `references/custom-rules.md`.

## Concurrent Indexes

`CREATE INDEX` takes a `SHARE` lock that blocks writes, and `DROP INDEX` takes `ACCESS EXCLUSIVE`. The
`concurrent_index` diff policy, off by default, makes Atlas plan both with `CONCURRENTLY`, and the files
it generates get the `atlas:txmode none` directive (https://atlasgo.io/versioned/diff#diff-policy):

```hcl
env "local" {
  diff {
    // By default, indexes are not created or dropped concurrently.
    concurrent_index {
      add  = true
      drop = true
    }
  }
}
```

A file written by hand needs the directive at the top, followed by an empty line
(https://atlasgo.io/versioned/apply#transaction-configuration):

```sql
-- atlas:txmode none

CREATE INDEX CONCURRENTLY name_idx ON users (name);
```

- Indexes that back a `UNIQUE`, primary key, or `EXCLUDE` constraint, and indexes on partitioned tables,
  cannot be dropped concurrently (`PG102`).
- An index on a partitioned table cannot be built concurrently (`PG108`). Under the policy, Atlas builds
  each partition's index concurrently, then the parent's, which attaches them.
- A new primary key or unique constraint locks the table (`PG104`, `PG105`). Build its index with
  `CREATE UNIQUE INDEX CONCURRENTLY`, check that the index is valid, and attach it in a later file with
  `ADD PRIMARY KEY USING INDEX` or `ADD CONSTRAINT ... UNIQUE USING INDEX`.
- In an HCL schema, add the `index` block first and apply it under the policy, then replace it with a
  `unique` block (https://atlasgo.io/atlas-schema/hcl#adding-unique-constraints-concurrently).
- An `EXCLUDE` constraint cannot use a pre-built index or `NOT VALID` (`PG109`).

## Rewrites and Scans

- A column type change rewrites the table under `ACCESS EXCLUSIVE` (`PG301`), except binary-coercible
  changes such as `varchar` to `text`, a longer or unlimited `varchar` length, and `numeric(p,s)` to
  `numeric`. Changing between `timestamp` and `timestamptz` rewrites unless the session time zone is UTC.
  Lint flags collation changes and `char(n)` length changes under `PG301` too.
- Volatile defaults (`PG302`), `SET LOGGED`/`UNLOGGED` (`PG307`), stored generated columns (`PG309`),
  identity columns (`PG310`), and a new access method (`PG311`) rewrite the table.
- A `CHECK` or foreign key added without `NOT VALID` scans the table under a lock (`PG305`, `PG306`). Add
  it `NOT VALID`, then validate it in a separate transaction. Inside one transaction, the lock the `ADD`
  takes is held while `VALIDATE` scans the table, so start the file with `-- atlas:txmode none`, or
  validate in a later migration file
  (https://atlasgo.io/guides/lock-safe-not-null#why-the-transaction-is-disabled):

  ```sql
  -- atlas:txmode none

  ALTER TABLE orders ADD CONSTRAINT fk_orders_users FOREIGN KEY (user_id) REFERENCES users(id) NOT VALID;

  -- Validate the constraint.
  ALTER TABLE orders VALIDATE CONSTRAINT fk_orders_users;
  ```

- A trigger on an existing table takes `SHARE ROW EXCLUSIVE` (`PG308`). Redefining a primary key holds
  `ACCESS EXCLUSIVE` while its index builds (`PG312`).

## Lock-Safe NOT NULL

A plain `SET NOT NULL` scans the table under `ACCESS EXCLUSIVE`. The `not_null` diff policy plans it in
steps that avoid the long lock (https://atlasgo.io/guides/lock-safe-not-null):

```hcl
diff "postgres" {
  not_null {
    check     = true
    lock_safe = true
  }
}
```

- `check = true` writes the migration as a txtar file with a pre-migration check that no row is `NULL`,
  which clears `MF104`. `lock_safe` requires `check`.
- On PostgreSQL 18, Atlas uses `NOT NULL ... NOT VALID`. On earlier versions it adds a temporary
  `CHECK ... NOT VALID`, validates it, sets `NOT NULL`, and drops the check. The `migration.sql` section
  starts with `-- atlas:txmode none`.
- Atlas falls back to a plain `SET NOT NULL` when a validated `CHECK (col IS NOT NULL)` already covers
  the column, or when another change in the same migration rewrites the table anyway.

## Transactions

- `atlas migrate apply` runs each file in its own transaction by default (`--tx-mode file`); `all` runs
  every pending file in one, and `none` runs statements without one. `atlas schema apply` supports
  `file` and `none` (https://atlasgo.io/versioned/apply#transaction-configuration).
- `-- atlas:txmode none` sets the mode for one file. Use it for statements that cannot run in a
  transaction, such as `CREATE INDEX CONCURRENTLY` (`TX101`), or creating a logical replication slot,
  which otherwise fails with
  `pq: cannot create logical replication slot in transaction that has performed writes`.
  Directives work only in Atlas-format migration directories, and need an empty line after them.
- Never put `BEGIN`/`COMMIT` in a file that Atlas wraps in a transaction (`TX201`): the run fails with
  `pq: unexpected transaction status idle`.
- A file without a transaction stops at the failed statement, and the next run resumes from it. When
  fixing the file, leave the statements before it unchanged, since they already ran, and run
  `atlas migrate hash` after editing.
- Migration files are SQL sent to the server, not psql scripts. Meta-commands such as `\set`, and
  `:'var'` variables, fail with `pq: syntax error at or near "\"`.
- `executing statement "..." from version "X": pq: ...` comes from replaying the migration directory,
  or the desired state (`from version "schema"`), on the dev database. The `pq:` part is PostgreSQL's
  error, and the version names the file that failed.
- `pq: current transaction is aborted, commands ignored until end of transaction block` follows an
  earlier failure in the same transaction. Find the first statement that failed.

## Revisions Table and Locks

- `atlas_schema_revisions` lives in the schema the URL's `search_path` names. With a database-scoped URL
  it gets a schema of its own, `atlas_schema_revisions`. `--revisions-schema`, or `revisions_schema` in
  the env's `migration` block, moves it; Atlas creates that schema if it is missing.
- After an existing project switches its URL between schema and database scope, Atlas fails with
  `ambiguous revision table` until `--revisions-schema` names the schema that holds the existing table.
- On Neon, `relation "atlas_schema_revisions.atlas_schema_revisions" does not exist` means the
  connection ignored `search_path`. Set `--revisions-schema public`.
- `migrate apply` and `schema apply` hold a PostgreSQL advisory lock for the run, so two deployments
  cannot apply at the same time. `--lock-timeout` waits 10 seconds by default; `--skip-lock` skips it.
  The lock needs a direct connection; see Connection Poolers below.
- `hook` blocks run SQL inside the migration transaction, for example to bound locks
  (https://atlasgo.io/versioned/apply#migration-hooks):

  ```hcl
  hook "sql" "timeout" {
    transaction {
      after_begin = [
        "SET statement_timeout TO '50ms'",
      ]
      before_commit = [
        // ...
      ]
    }
  }
  ```

  Hooks do not run for `txmode none` files, which include the files the `concurrent_index` policy
  writes.

## Connection Poolers

Never apply migrations through a connection pooler in transaction mode. The advisory lock that
`migrate apply` and `schema apply` hold is per session
(https://atlasgo.io/versioned/apply#controlling-advisory-locks). A transaction-mode pooler gives every
transaction a different server connection, so the lock is taken on one connection while the migration
runs on another, and nothing stops a second deployment. The providers say the same:

| Service | Give Atlas | Not | Where the provider says so |
|---------|------------|-----|----------------------------|
| Supabase | The direct connection, `db.[project-ref].supabase.co:5432`. From an IPv4-only network without the IPv4 add-on, the session pooler | The transaction pooler, port 6543 | Its connection guide lists the direct connection for migrations, and says transaction mode loses "session-level advisory locks": https://supabase.com/docs/guides/database/connecting-to-postgres#choose-a-connection-method and #transaction-mode-limitations |
| Neon | The direct connection string: the hostname without `-pooler` | The pooled string, whose hostname ends in `-pooler` | Its pooler runs PgBouncer in transaction mode, lists "Session-level advisory locks" as unsupported, and recommends a direct connection for schema migrations: https://neon.com/docs/connect/connection-pooling#when-to-use-pooled-vs-direct-connections |
| PgBouncer, or any pooler built on it | A direct connection, or a pool in session mode | A pool in transaction mode | PgBouncer's feature table: session-level advisory locks work in transaction pooling "Never" (https://www.pgbouncer.org/features.html) |

Through PgBouncer, Atlas also fails with `pq: unsupported startup parameter: search_path`, or with
`acquiring database lock: pq: unnamed prepared statement does not exist`. Both mean the URL goes
through the pooler: switch to the direct connection.

The application keeps its pooled URL; only Atlas needs the direct one. Store the direct URL as its own
secret for CI and deployments, and use it in the env that applies migrations.

## Existing Databases

A database that has objects but no revisions table is refused until it gets a baseline: the first
migration recorded as applied without running (https://atlasgo.io/versioned/apply#existing-databases).
Set `baseline` in the env's `migration` block, or pass `--baseline <version>` once. The GitHub action
`migrate/apply` has no baseline input; use the env attribute
(https://atlasgo.io/faq/migrate-baseline-github-action).

## Seed Data

Lookup tables and other reference rows can be declared with the schema and kept in sync on every apply
(https://atlasgo.io/guides/postgres/seed-data). `INSERT` adds missing rows, `UPSERT` also updates them,
and `SYNC` makes the table match exactly; `SYNC` requires `max_rows` and refuses tables larger than it.

## Large Changes

Nothing in Atlas batches a single huge plan for you. To spread a large change over several deployments,
split it into several migration files and apply a few at a time with `atlas migrate apply <n>`, or cap
each run with a `deny` rule in `check "migrate_apply"` (`atlas/references/versioned.md`). For a grant
that would expand to every table, see "One role on every table" in `references/security.md`.
