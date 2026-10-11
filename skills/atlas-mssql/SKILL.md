---
name: atlas-mssql
description: "SQL Server specifics for Atlas (also Azure SQL and MSSQL): login, dev database and URL scope (mode=schema vs mode=database, the database query parameter), compatibility level and collation, system-generated constraint names, clustered index defaults and datetime2 precision in HCL, GO batches and the delimiter directive, transactions and statements that cannot run in one, identity, computed and temporal columns, alias and table types, changes planned as a drop and an add, what Atlas leaves out (DENY, synonyms, row-level security, sparse columns), database roles, users and grants, schema and migration tests, and custom lint rules. Use with the atlas skill whenever Atlas manages a SQL Server or Azure SQL database."
---

# Atlas for SQL Server

What is specific to SQL Server when Atlas manages the schema. The `atlas` skill covers the workflows
and commands; this skill covers the rules, edge cases, and answers that differ on SQL Server. Behavior
is described for SQL Server 2022; where a compatibility level changes it, the text says so. Azure SQL
Database follows the same rules, and Fabric Data Warehouse has a guide of its own
(https://atlasgo.io/guides/azure-fabric).

## Files

This skill is `SKILL.md` plus seven reference files under `references/`, each served at
`https://atlasgo.io/skills/atlas-mssql/references/<name>`. Read a reference when the task needs it:

| File | Read it for |
|------|-------------|
| `references/dev-database.md` | Connection URLs, `mode=schema` and `mode=database`, Docker dev URLs, `atlas tool docker`, the `docker` block (`collate`, `baseline`), your own dev database, compatibility level and collation, Azure SQL |
| `references/security.md` | Database roles and users, users for logins, fixed database roles, `GRANT` and `REVOKE`, what is not managed (`DENY`, logins, row-level security) |
| `references/objects.md` | Constraint names, identity and sequences, column changes, alias and table types, functions, procedures, triggers and views, clustered indexes, temporal, memory-optimized and partitioned tables, what Atlas does not manage |
| `references/hcl.md` | Writing the schema in HCL: the `NOT NULL` default, defaults and their names, types (`-1` for `MAX`, `datetime2(7)`), `nonclustered`, routines, sequences and types, seed data |
| `references/migrations.md` | Lint on SQL Server, transactions, `GO` and the delimiter directive, `txmode none`, lock-safe changes, the revisions table and the lock, baselines |
| `references/testing.md` | Testing every change: T-SQL assertions, schema tests, permission tests with `as`, migration tests with seeded rows |
| `references/custom-rules.md` | Writing a custom lint rule in the HCL rule language when no analyzer covers a check: what SQL Server objects expose, and tested rules |

## Requires

The `atlas` skill, installed next to this one (`../atlas/SKILL.md`). If it is missing, download
https://atlasgo.io/skills/atlas/SKILL.md to `../atlas/SKILL.md` and each reference file listed in its
Files section to `../atlas/references/<name>`.

## Rules

1. Check the login first with `atlas whoami`. The SQL Server driver needs a login for every command:
   logged out, Atlas aborts with `The sqlserver driver is available only to Atlas Pro users` or
   `login is required to use the 'sqlserver' driver`. If it shows no login, ask the user to run
   `atlas login`; a user without an account can start a free trial with it.
2. Write URLs as `sqlserver://user:pass@host:1433?database=app&mode=schema`, and pick the scope from
   what the schema holds (`references/dev-database.md`):
   - The database goes in `?database=`. A database in the URL path is read as an instance name, and
     Atlas silently connects to the login's default database, usually `master`.
   - Schema scope, `mode=schema`, the default: Atlas manages the connecting user's default schema
     (`dbo`) and writes unqualified statements (`[users]`). Database scope, `mode=database`: Atlas
     covers every schema in that database, plus roles, users, and partition functions, and qualifies
     each statement (`[dbo].[users]`).
   - At database scope, a schema the desired state does not declare is planned for `DROP SCHEMA`, after
     its tables. Limit the run with `schemas` or `--schema`.
   - The dev URL takes the target's scope: `docker://sqlserver/2022-latest/dev?mode=schema`, or
     `?mode=database`.
3. The dev database runs the target's major version, compatibility level, and collation. Atlas changes
   views and triggers with `CREATE OR ALTER` only at level 130 and later, and a case-sensitive target
   needs a case-sensitive dev database: set `collate` in a `docker` block. Create what the schema
   references but does not manage in the block's `baseline`. Never point `dev` at a real environment's
   database.
4. Name every primary key, check, and default in a SQL schema with `CONSTRAINT <name>`, and give every
   HCL `check` block a label. SQL Server names unnamed constraints itself with a random suffix
   (`PK__users__3213E83F6692D103`), so every plan renames the primary key or drops and re-adds the
   check or default. Keep the generated names inspection writes for existing constraints
   (`references/objects.md`).
5. HCL differs from T-SQL in three places (`references/hcl.md`):
   - A column without `null = true` is `NOT NULL`, and so is a `type_alias`.
   - An `index` or `unique` block without `nonclustered = true` is clustered. Write
     `nonclustered = true` on every secondary index and unique constraint, or the plan fails with
     `Cannot create more than one clustered index`.
   - A bare `datetime2`, `time`, or `datetimeoffset` has precision 0. Write `datetime2(7)`. Write
     `MAX` as `-1`: `nvarchar(-1)`.
6. Migration files are T-SQL, not `sqlcmd` scripts (`references/migrations.md`):
   - End every `CREATE PROCEDURE`, `FUNCTION`, `TRIGGER`, and `VIEW` with `GO`, or its body swallows the
     next statement.
   - Each statement runs on its own, so a `DECLARE`d variable is gone in the next one. Start a file that
     uses variables with `-- atlas:delimiter \nGO` and an empty line, so only `GO` splits it.
   - `:setvar`, `:r`, and `$(var)` fail.
7. Atlas runs each migration file in a transaction, and a file that fails is rolled back completely.
   Never write `BEGIN TRANSACTION` or `COMMIT` in it. Statements that cannot run in a transaction, such
   as `ALTER DATABASE`, full-text indexes, and `RESUMABLE` index builds, go in a file of their own that
   starts with `-- atlas:txmode none`.
8. Some changes are planned as a drop and an add, which loses data; lint reports them only as
   `DS103` (`references/objects.md`):
   - An identity's `seed` or `increment`, or making an existing column an identity. To restart
     numbering, run `DBCC CHECKIDENT ('dbo.t', RESEED, n)` in a migration file and leave the schema's
     `seed` alone.
   - A computed column's expression.
   - A partition function's input type or range side drops and recreates every table on it.
   Write these by hand, and test them with seeded rows (Rule 11).
9. Atlas leaves out `DENY`, database-level permissions, logins, synonyms, security policies, dynamic
   data masking, Always Encrypted, `SPARSE` columns, data compression, and filegroups. A statement for
   them in a SQL schema is dropped without an error, and no drift is reported. Apply them in
   hand-written migration files (`references/objects.md`).
10. Managing roles and users needs `mode=database` on both URLs and `mode { roles = true }`; grants need
    `permissions = true` (`references/security.md`). Leave the fixed roles (`db_datareader`, ...) out
    of HCL and write membership in them in SQL. A `user` with `login_name` needs the login on the dev
    server. Set `drop_user` and `drop_role` in the `skip` diff policy if other teams own principals in
    the database.
11. Test every change before reporting it done, as a habit. Run `atlas schema validate`, add a
    `test "schema"` case for each function, procedure, trigger, view, computed column, or check you add
    or change, and a `test "migrate"` case with seeded rows for each migration that moves, converts, or
    narrows data. Cast `decimal`, `money`, and `uniqueidentifier` values to text in test queries, and
    write asserts as `SELECT IIF(<condition>, 1, 0)`. Run them with `atlas schema test` and
    `atlas migrate test` (`references/testing.md`).
12. SQL Server has no lint analyzers of its own, and lint does not report a narrower type that truncates
    rows. When no analyzer covers a check, write a custom lint rule in the HCL rule language, and do the
    same for a convention the user states, such as an index on every foreign key or `SCHEMABINDING` on
    every view (`references/custom-rules.md`).

## Quick Answers

| Question | Answer |
|----------|--------|
| A local SQL Server database to apply migrations to | `atlas tool docker --url "docker://sqlserver/2022-latest/myapp" --name my-db` prints its URL; `atlas tool docker kill --name my-db` stops it (`references/dev-database.md`) |
| `The sqlserver driver is available only to Atlas Pro users`, or `login is required to use the 'sqlserver' driver` | Ask the user to run `atlas login` (Rule 1) |
| Inspection prints an empty `schema "dbo" {}` for a database that has tables | The database is in the URL path; move it to `?database=` (Rule 2) |
| `cannot diff a schema "dbo" with a database connection`, or the reverse | Give the URL and the dev URL the same `mode` (`references/dev-database.md`) |
| `cannot use HCL with more than 1 schema when dev-url is limited to schema "dbo"` | Use `mode=database` on both URLs (`references/dev-database.md`) |
| `schema connection mode requires a default schema to be set for the user` | Give the user a `DEFAULT_SCHEMA`, or use `mode=database` (`references/dev-database.md`) |
| A plan drops other schemas (`DROP SCHEMA`) | Database scope covers every schema: limit it with `schemas` or `--schema` (`references/dev-database.md`) |
| Every plan renames a primary key (`sp_rename`), or drops and re-adds a check or default | Name the constraint (Rule 4) |
| `Cannot create more than one clustered index on table`, or `Cannot add more than one clustered index for constraints on table` | Add `nonclustered = true` to the HCL `index` or `unique` block (Rule 5) |
| `datetime2` values lost their fractional seconds | The HCL type is a bare `datetime2`: write `datetime2(7)` (`references/hcl.md`) |
| `There is no type named "MAX"` | Write `nvarchar(-1)` or `sql("nvarchar(max)")` (`references/hcl.md`) |
| A column became `NOT NULL` without asking for it | HCL columns are `NOT NULL` unless they set `null = true` (`references/hcl.md`) |
| `Incorrect syntax near the keyword 'VIEW'` (or another keyword) after a `CREATE PROCEDURE` | Add `GO` after the procedure (Rule 6) |
| `Must declare the scalar variable "@x"` | Start the file with `-- atlas:delimiter \nGO` (`references/migrations.md`) |
| `Incorrect syntax near ':'` | `sqlcmd` syntax such as `:setvar`: use plain T-SQL (`references/migrations.md`) |
| `ALTER DATABASE statement not allowed within multi-statement transaction` | Move it to a file with `-- atlas:txmode none` (Rule 7) |
| `Could not find stored procedure 'COMMIT'` | Remove `BEGIN TRANSACTION` and `COMMIT` from the file (Rule 7) |
| `String or binary data would be truncated` | A narrower column type: fix the rows first, and add a migration test (`references/objects.md`) |
| `Incorrect syntax near the keyword 'RESTRICT'` | SQL Server has no `RESTRICT`: use `NO_ACTION` (`references/hcl.md`) |
| A plan drops and re-adds an identity or computed column (`DS103`) | Data loss: write it by hand, or reseed with `DBCC CHECKIDENT` (Rule 8) |
| `Cannot drop type 'x' because it is being referenced by object` | A column uses the alias type: write the change by hand (`references/objects.md`) |
| `cannot set value for generated column "id"` in a `data` block | Identity columns cannot be seeded from HCL; seed them from SQL (`references/objects.md`) |
| `connected database is not clean` | `dev` points at a database that is not empty (`references/dev-database.md`) |
| `already exists` or `There is already an object named` on the dev database, on a later run | A user, login, or unmanaged object stayed on a dev database you provide: use a Docker dev database (`references/dev-database.md`) |
| `Invalid object name 'X'` while loading the schema | Identifier case on a case-sensitive dev database, or an object missing from the `baseline` (`references/dev-database.md`) |
| `User, group, or role 'db_...' already exists in the current database` | Remove the fixed-role `role` blocks from the HCL (`references/security.md`) |
| `unknown attribute external in block of type role` | SQL Server roles have no `external` (`references/security.md`) |
| `'x' is not a valid login or you do not have permission` | Create the login on the dev server, in the `baseline` (`references/security.md`) |
| A `DENY`, synonym, security policy, or `SPARSE` column is missing from the plan | Atlas does not manage it: write it in a migration file (Rule 9) |
| `acquiring database lock: sql/schema: lock is held by other session` | Another deployment holds the database's lock: wait, or raise `--lock-timeout` (`references/migrations.md`) |
| `We couldn't find a revision table in the connected schema` after changing `mode` | Set `--revisions-schema` to the schema that holds the table (`references/migrations.md`) |
| Test output shows `0x...` for numbers or GUIDs | Cast the column to `varchar` in the test query (`references/testing.md`) |
| `template attribute is supported only for postgres dev-databases` | SQL Server has no template databases: remove `template` (`references/testing.md`) |
| Lint does not flag a narrower column type | Add a migration test with seeded rows, and a custom rule if it is a convention (Rule 12) |

## Documentation

- SQL Server HCL reference: https://atlasgo.io/hcl/mssql
- SQL Server guide: https://atlasgo.io/guides/mssql
- Security: https://atlasgo.io/guides/mssql/security-declarative,
  https://atlasgo.io/guides/mssql/security-versioned
- Seed data: https://atlasgo.io/guides/mssql/seed-data
- Dev database: https://atlasgo.io/concepts/dev-database
- Lint checks: https://atlasgo.io/lint/analyzers
- Guides for each database: https://atlasgo.io/databases
