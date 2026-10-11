# SQL Server Migrations: Lint, Transactions, and Batches

How Atlas runs migrations on SQL Server, what lint catches, and what to write by hand. Tests are in
`references/testing.md`. Lint, custom rules, the `not_null` policy, and seed data need `atlas login`,
which SQL Server needs anyway.

## What Lint Catches on SQL Server

SQL Server has no analyzers of its own (no `MS*` codes). The generic ones apply
(https://atlasgo.io/lint/analyzers):

| Code | Catches |
|------|---------|
| `DS101`, `DS102`, `DS103` | Dropping a schema, a table, or a column, including the drop and add Atlas plans for identity and computed-column changes |
| `CD101`, `CD102`, `CD103` | Dropping a foreign key, a check, or a primary key |
| `MF101`, `MF102` | A unique index that may fail on duplicate rows |
| `MF103` | A `NOT NULL` column without a default added to an existing table |
| `MF104` | `NULL` to `NOT NULL` on a column that may hold `NULL` |
| `BC101`, `BC102`, `BC103`, `BC104` | Renames (`sp_rename`) and drops that break clients |
| `NM101` to `NM106` | Names that break the `naming` policy |
| `TX101` | Statements that cannot run in a transaction, mixed with others in one file |

What lint does not catch on SQL Server, so review the plan or test it with seeded rows:

- A narrower type, such as `varchar(50)` to `varchar(20)`, which fails on rows that do not fit with
  `String or binary data would be truncated`.
- Index builds and column changes that lock or rebuild a large table.
- `BEGIN TRANSACTION` and `COMMIT` inside a file (Transactions below).

Each check is tuned in the `lint` block (https://atlasgo.io/lint/analyzers#checks):

```hcl
lint {
  check "MF104" {
    error = true
  }
  check "CD102" {
    skip = true
  }
}
```

When no analyzer covers a check, or the user states a convention, write a custom rule in the HCL rule
language (`references/custom-rules.md`), or a `statement` rule that denies a pattern, such as
`TRUNCATE TABLE`.

## Transactions

- `atlas migrate apply` runs each file in its own transaction by default (`--tx-mode file`); `all` runs
  every pending file in one, and `none` runs statements without one. `atlas schema apply` supports
  `file` and `none` (https://atlasgo.io/versioned/apply#transaction-configuration).
- DDL is transactional on SQL Server: a file that fails is rolled back completely, and
  `atlas migrate status` shows it as pending. Fix the file, run `atlas migrate hash`, and apply again.
- Some statements cannot run in a transaction: `ALTER DATABASE`, `CREATE DATABASE`, `BACKUP`,
  `RESTORE`, `RECONFIGURE`, full-text catalogs and indexes, and index operations with
  `RESUMABLE = ON`. In a transaction they fail, for example with
  `mssql: ALTER DATABASE statement not allowed within multi-statement transaction.` Put them in their
  own file that starts with the directive, followed by an empty line:

  ```sql
  -- atlas:txmode none

  ALTER DATABASE CURRENT SET ALLOW_SNAPSHOT_ISOLATION ON;
  ```

  Atlas adds the directive itself to the files it generates for full-text indexes and catalogs and for
  natively compiled modules. Lint reports `TX101` when such a statement shares a file with other
  statements; alone in a file without the directive, lint reports nothing and the apply fails.
- A file without a transaction stops at the failed statement, and the next run resumes from it. Leave
  the statements before it unchanged, since they already ran, and run `atlas migrate hash`.
- Never write `BEGIN TRANSACTION`, `COMMIT`, or `ROLLBACK` in a file that Atlas wraps in a transaction.
  A bare `COMMIT;` fails with `mssql: Could not find stored procedure 'COMMIT'.`, and
  `COMMIT TRANSACTION` only closes the inner level of a nested transaction, while a `ROLLBACK` undoes
  Atlas's own transaction. Lint does not report them on SQL Server.

## Batches and `GO`

Atlas sends each statement of a migration file to SQL Server as its own batch. It splits the file on
`;` and on `GO`, except inside `BEGIN ... END`:

- A `CREATE PROCEDURE`, `FUNCTION`, `TRIGGER`, or `VIEW` must be followed by `GO`. Without it, the body
  runs to the next `GO` or the end of the file and swallows the next statement:
  `mssql: Incorrect syntax near the keyword 'VIEW'.`
- A variable does not survive from one statement to the next:
  `mssql: Must declare the scalar variable "@n".` For a script that declares variables, start the file
  with the `GO` delimiter, so only `GO` splits it:

  ```sql
  -- atlas:delimiter \nGO

  DECLARE @cutoff datetime2(7) = DATEADD(day, -90, SYSUTCDATETIME());
  DELETE FROM dbo.sessions WHERE last_seen < @cutoff;
  GO
  ```

- Atlas writes generated files that hold functions or procedures the same way: the directive at the
  top, and `GO` after every statement.
- Migration files are T-SQL sent to the server, not `sqlcmd` scripts. `:setvar`, `:r`, and `$(var)`
  fail with `mssql: Incorrect syntax near ':'.` Use input variables in `atlas.hcl`, or a template
  directory (https://atlasgo.io/guides/migration-dirs/template-directory).
- `executing statement "..." from version "X": mssql: ...` names the statement and the file that
  failed. The `mssql:` part is SQL Server's message.

## Lock-Safe Changes

Atlas has no online or lock-safe policy for SQL Server: it plans plain `CREATE INDEX`,
`ALTER TABLE ... ALTER COLUMN`, and `ADD CONSTRAINT`.

- Atlas changes an index by dropping and recreating it. For a large table, write the change by hand
  with `WITH (ONLINE = ON)` where the edition supports it, then run `atlas migrate hash`. An index
  build with `RESUMABLE = ON` needs `-- atlas:txmode none`.
- `NULL` to `NOT NULL` scans the table. The `not_null` diff policy writes the migration as a txtar file
  with a pre-migration check that no row is `NULL`, which clears `MF104` (https://atlasgo.io/hcl/config):

  ```hcl
  env "local" {
    diff "sqlserver" {
      not_null {
        check = true
      }
    }
  }
  ```

- Changing an identity's seed, making a column an identity, or changing a computed column drops the
  column and adds it again: lint reports it only as `DS103`. Write those by hand
  (`references/objects.md`).
- A partition function's input type or range side, or a partition scheme's filegroups, drop and
  recreate every table on them. Review those plans line by line.

## Revisions Table and Locks

- `atlas_schema_revisions` lives in the connected schema (`dbo`) at schema scope, and in a schema of its
  own, `atlas_schema_revisions`, at database scope. `--revisions-schema`, or `revisions_schema` in the
  env's `migration` block, moves it.
- After an existing project switches its URL between `mode=schema` and `mode=database`, Atlas stops
  with `We couldn't find a revision table in the connected schema` until `--revisions-schema` names
  the schema that holds the table, `atlas_schema_revisions` or the old schema.
- `migrate apply` and `schema apply` hold an application lock (`sp_getapplock`) on the database for the
  run, so two deployments cannot apply to it at the same time. `--lock-timeout` waits 10 seconds by
  default, then fails with `acquiring database lock: sql/schema: lock is held by other session`.
  `--skip-lock` skips it. The lock belongs to one database: deployments to other databases on the same
  server do not wait for it.
- `hook` blocks run SQL inside the migration transaction, such as `SET LOCK_TIMEOUT 5000` in
  `after_begin` (https://atlasgo.io/versioned/apply#migration-hooks). They do not run for
  `txmode none` files.

## Existing Databases

A database that has objects but no revisions table is refused until it gets a baseline: the first
migration recorded as applied without running (https://atlasgo.io/versioned/apply#existing-databases).
Set `baseline` in the env's `migration` block, or pass `--baseline <version>` once.

Generate the baseline from the database itself, logged in: its constraints keep the names SQL Server
generated, such as `PK__users__3213E83F6692D103`. Keep those names in the desired state, or rename
them in a later, reviewed migration (`references/objects.md`, Name Every Constraint).

## Seed Data

Lookup tables can be declared with the schema and kept in sync on every apply
(https://atlasgo.io/guides/mssql/seed-data). `INSERT` adds missing rows, `UPSERT` also updates them, and
`SYNC` makes the table match exactly; `SYNC` requires `max_rows`. Identity columns are left to the
database, and `preserve_ids` has no effect on SQL Server (`references/objects.md`).

## Large Changes

Nothing in Atlas batches a single huge plan for you. Split a large change into several migration
files and apply a few at a time with `atlas migrate apply <n>`, or cap each run with a `deny` rule in
`check "migrate_apply"` (`../atlas/references/versioned.md`). A data backfill on a large table belongs in
a Data Script loop (`../atlas/references/scripts.md`), not in a migration file.
