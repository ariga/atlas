# MySQL Migrations: Lint, Locks, and Transactions

How to change a MySQL schema without blocking traffic, and how Atlas runs migrations on a database that
commits DDL as it goes. Tests are in `references/testing.md`. Lint, custom rules, and seed data need
`atlas login`. If `atlas whoami` shows no login, ask the user to run `atlas login`; a user without an
account can start a free trial with it.

## What Lint Catches on MySQL

Most MySQL checks report a change that makes MySQL copy or rebuild the table, or block writes while it
runs (https://atlasgo.io/lint/analyzers):

| Code | Reports |
|------|---------|
| `MY101` | A `NOT NULL` column without a `DEFAULT` added to an existing table: MySQL fills existing rows with the type's zero value |
| `MY102` | A column added with an inline `REFERENCES` clause, which MySQL ignores |
| `MY110` to `MY113` | Removing, reordering, or inserting enum values anywhere but at the end, or crossing a storage-size boundary: the table is copied |
| `MY120` to `MY123` | The same for `SET` columns |
| `MY130` | A column type change, which copies the table |
| `MY131` | A new foreign key, which blocks writes while `foreign_key_checks` is on |
| `MY132`, `MY133`, `MY137` | Adding, dropping, or modifying a primary key, which rebuilds or copies the table |
| `MY134`, `MY135` | A new `FULLTEXT` or `SPATIAL` index, which blocks writes |
| `MY136` | A table character set change, which rebuilds the table |
| `MY138` | A storage engine change, which copies the table and blocks writes |
| `MY139` | Partitioning a table or removing its partitioning, which copies it and blocks writes |
| `MY140` | A new `STORED` generated column, which copies the table |
| `MY141` | A new `AUTO_INCREMENT` column, which rebuilds the table |
| `MY142` | A column added before existing ones, which is not instant on MySQL 8.0.12 to 8.0.28 and MariaDB 10.3; lint checks the dev database's version |
| `MY143` | A generated column change, which copies the table |
| `MY144`, `MY145` | Adding, modifying, or enforcing a `CHECK`, which validates every row |
| `MY146` | Dropping system versioning on MariaDB, which deletes the history |
| `MY147` | A column nullability change, which rebuilds the table |
| `MY148` | A column character set or collation change, which copies the table |
| `DS102`, `DS103` | Dropping a table or column that holds data |
| `MF104` | Making a column `NOT NULL` when it may hold `NULL` |
| `TX101` | Statements that cannot share a transaction |

What the docs give as fixes:

- `MY110` to `MY123`: add new enum and set values at the end of the list. Inserting a value in the
  middle copies the whole table, however large.
- `MY138`: a storage engine change is often an unintended edit of the `engine` attribute.
- `MY140`: add the generated column as `VIRTUAL`. `MY141`: add the column without `AUTO_INCREMENT`.
  `MY142`: add the column last, or upgrade.
- `MY143`: for a `VIRTUAL` column on a table without partitions, add a new column and drop the old one.
- `MY144`: add the check `NOT ENFORCED`, on MySQL only; MariaDB has no such state.
- `MY146`: archive the history first, with `FOR SYSTEM_TIME ALL`.
- `MF104`: backfill the `NULL` rows first, and guard the migration with a check
  (https://atlasgo.io/lint/analyzers#MF104):

  ```sql
  -- atlas:assert MF104
  SELECT NOT EXISTS (SELECT 1 FROM `users` WHERE `name` IS NULL) AS `not_null`;
  ```

- `TX101`: split the statements into separate files, or start the file with `-- atlas:txmode none`.

Each check is tuned in the `lint` block, for example to fail on an enum insert that copies the table:

```hcl
lint {
  // Enum insert that copies the table.
  check "MY112"    { error = true }
}
```

A statement can opt out of named checks with a directive
(https://atlasgo.io/versioned/lint#nolint-directive):

```sql
-- atlas:nolint DS103 MY101
ALTER TABLE `t1` DROP COLUMN `c1`, ADD COLUMN `d1` varchar(255) NOT NULL;
```

### Custom rules

When no analyzer above covers a check, or the user states a convention, such as an allowed character
set or deterministic functions, write a custom rule in the HCL rule language. A rule can check any
attribute of any object Atlas inspects: `references/custom-rules.md`.

## Table Copies

MySQL runs each `ALTER` in place, instantly, or by copying the table. A copy needs disk space for both
copies, and many of the changes above also block writes while they run. Lint reports them before they
reach production, so review every `MY1*` finding on a large table. Lint reads the dev database's
version for some checks, so run the dev database on the target's version
(`references/dev-database.md`).

- Atlas adds no `ALGORITHM=` or `LOCK=` clause to the statements it plans; MySQL picks the algorithm.
  To make sure a change never copies the table, write the statement by hand in a migration file with
  `ALGORITHM=INSTANT`, or `ALGORITHM=INPLACE, LOCK=NONE`: MySQL then fails instead of copying.
- A change that only works as a copy, on a table too large to lock, needs an online schema change tool
  such as gh-ost or pt-online-schema-change, run outside Atlas. Keep the change in its migration file,
  and once the tool finishes, record the file as applied with `atlas migrate set <version>`, which
  updates the revisions table without running SQL.

## Transactions

MySQL and MariaDB have no transactional DDL: each DDL statement commits on its own. Atlas cannot roll
back a migration file, or a declarative plan, that fails halfway. The statements before the failure
stay applied, and the schema and the revisions table can end up out of step
(https://atlasgo.io/versioned/troubleshoot#a-word-on-transactions).

- `--tx-mode` does not change this: `all` cannot make several files atomic, because every DDL statement
  commits the open transaction. Data statements that run before a DDL statement in the same file are
  committed with it (https://atlasgo.io/versioned/apply#transaction-configuration).
- Atlas records each statement it applies, so a failed file shows as partially applied in
  `atlas migrate status`
  (https://atlasgo.io/versioned/troubleshoot):

  ```text
  Migration Status: PENDING
    -- Current Version: 2 (2 statements applied)
    -- Next Version:    2 (1 statements left)
    -- Executed Files:  2 (last one partially)
    -- Pending Files:   2
  ```

- The next `atlas migrate apply` continues from the statement that failed. Fix that statement and the
  ones after it, leave the applied ones unchanged, and run `atlas migrate hash` after editing. When data
  caused the failure, fix the data, and add the fix to the file.
- To undo a partially applied file instead, run `atlas migrate down`. On MySQL it reverts the applied
  statements one at a time and updates the revisions table after each, so a run that fails midway is
  re-run to continue (https://atlasgo.io/versioned/down).
- If the connection drops between a statement and its entry in the revisions table, Atlas resumes one
  statement early. Revert that statement by hand before the next run
  (https://atlasgo.io/versioned/troubleshoot#connection-loss-failures).
- When the schema and the revisions table disagree, `atlas migrate set <version>` updates the revisions
  table only; it runs and reverts no SQL (https://atlasgo.io/faq/revisions-table-delete-latest).
- `schema apply` has the same limit: a plan that fails midway leaves the statements before the failure
  applied, and the next `schema apply` plans the rest.
- Nothing rolls back, so a file with fewer statements leaves less to recover after a failure. Test
  migrations before applying them (`references/testing.md`).
- A stored program contains semicolons, so a file that creates one redefines the statement delimiter,
  with the MySQL client's `DELIMITER` command or the `-- atlas:delimiter` directive
  (https://atlasgo.io/versioned/new#custom-statements-delimiter):

  ```sql
  DELIMITER //
  CREATE PROCEDURE dorepeat(p1 INT)
      BEGIN
      SET @x = 0;
      REPEAT SET @x = @x + 1; UNTIL @x > p1 END REPEAT;
  END
  //
  DELIMITER ;
  CALL dorepeat(100)
  ```

## Locks

- `migrate apply` and `schema apply` take `GET_LOCK` on MySQL and MariaDB, so two deployments cannot
  apply at the same time. `--lock-timeout` waits 10 seconds by default; `--skip-lock` skips the lock
  (https://atlasgo.io/versioned/apply#controlling-advisory-locks).
- The lock is per server, and its name defaults to `atlas_migrate_execute`. Targets that are separate
  databases on the same server wait for each other. `--lock-name`, or `lock_name` in the env's
  `migration` block, scopes the lock per target; it needs `atlas login`.

## Revisions Table

- `atlas_schema_revisions` lives in the database the URL names. With a server-scoped URL, it gets a
  database of its own, `atlas_schema_revisions`. `--revisions-schema`, or `revisions_schema` in the
  env's `migration` block, moves it.
- At server scope, a pattern that names the revisions table includes its database, such as
  `myschema.atlas_schema_revisions`.
- Unqualified statements from a schema-scoped plan, applied through a server-scoped URL, fail with
  `Error 1046: No database selected` (`references/dev-database.md`, Scope).

## Existing Databases

A database that has tables but no revisions table is refused until it gets a baseline: the first
migration, recorded as applied without running (https://atlasgo.io/versioned/apply#existing-databases):

```bash
atlas migrate diff my_baseline \
  --dir "file://migrations" \
  --dev-url "docker://mysql/8/my_schema" \
  --to "mysql://root:pass@localhost:3306/my_schema"
```

Then pass its version once: `atlas migrate apply --baseline "20220811074144"`, or set `baseline` in the
env's `migration` block. The GitHub action `migrate/apply` has no baseline input; use the env attribute
(https://atlasgo.io/faq/migrate-baseline-github-action). When `--dev-url` points at a database that
already has the revisions table, `migrate diff` fails with an error that starts with
`connected database is not clean` and ends with `baseline version or allow-dirty is required`: point
`--dev-url` at a dev database instead (https://atlasgo.io/faq/migrate-diff-clean).

## Seed Data

Seed data is declared in HCL, with the modes and `preserve_ids` in `references/hcl.md`, Seed Data.

## Large Changes

Nothing in Atlas batches a single huge plan for you. To spread a large change over several deployments,
split it into several migration files and apply a few at a time with `atlas migrate apply <n>`.
