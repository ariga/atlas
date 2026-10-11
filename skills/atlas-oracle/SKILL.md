---
name: atlas-oracle
description: "Oracle Database specifics for Atlas: the Oracle build and login, dev database and URL scope (schema vs database, PDB services, the Free lite image and its tablespace), privileges for the Atlas user (SELECT_CATALOG_ROLE, DBMS_LOCK, quota), uppercase names and SYS_C constraint names, NUMBER and sequence pitfalls, identity columns, PL/SQL packages, compile errors and anonymous blocks, objects Atlas leaves out of plans (packages, partitions, materialized views, renames), partial migration failures (no transactional DDL), ORA-00054 and DDL_LOCK_TIMEOUT, ONLINE indexes and NOVALIDATE constraints, roles and grants, managed services (RDS, Autonomous Database), schema and migration tests, and custom lint rules. Use with the atlas skill whenever Atlas manages an Oracle database."
---

# Atlas for Oracle

What is specific to Oracle Database when Atlas manages the schema. The `atlas` skill covers the workflows
and commands; this skill covers the rules, edge cases, and answers that differ on Oracle. Behavior is
described for Oracle AI Database 26ai, release 23.26, the release of the Free image; where 19c or 21c
differ, the text says so.

## Files

This skill is `SKILL.md` plus seven reference files under `references/`, each served at
`https://atlasgo.io/skills/atlas-oracle/references/<name>`. Read a reference when the task needs it:

| File | Read it for |
|------|-------------|
| `references/dev-database.md` | The dev URL and its scope, the Free lite image and its tablespace, the `docker` block (`init` and `baseline`), your own dev database, releases and `MAX_STRING_SIZE`, privileges of the Atlas user, RDS, Autonomous Database, connection URLs |
| `references/security.md` | Roles in an env of their own, grants at schema scope, `READ` and column grants, `WITH ADMIN OPTION`, roles in definer's-rights code |
| `references/objects.md` | What Atlas manages and leaves out, names and `SYS_C` constraints, `NUMBER` and `''`, sequences and identity columns, constraint states, PL/SQL compile errors, materialized views |
| `references/hcl.md` | Writing the schema in HCL: uppercase labels, the schema name, types, `DEFAULT ON NULL`, identity and virtual columns, `NOVALIDATE` constraints, tablespaces, temporary tables, PL/SQL, sequences, seed data |
| `references/migrations.md` | Lint on Oracle, partial failures and recovery, PL/SQL and packages in migration files, `ORA-00054` and `DDL_LOCK_TIMEOUT`, `ONLINE` indexes, `NOVALIDATE`, renames, the revisions table, baselines |
| `references/testing.md` | Testing every change: schema tests, migration tests with seeded rows, checking for invalid PL/SQL, fast dev databases, CI |
| `references/custom-rules.md` | Writing a custom lint rule in the HCL rule language when no analyzer covers a check: what Oracle objects expose, and tested rules for foreign-key indexes, uppercase names, `CHAR` semantics, and release types |

## Requires

The `atlas` skill, installed next to this one (`../atlas/SKILL.md`). If it is missing, download
https://atlasgo.io/skills/atlas/SKILL.md to `../atlas/SKILL.md` and each reference file listed in its
Files section to `../atlas/references/<name>`.

## Rules

1. The Oracle driver ships in the Oracle build of Atlas and needs a login. If a command fails with
   `The oracle driver is available only as custom build`, install that build:
   `curl -sSf https://atlasgo.sh | ATLAS_FLAVOR="oracle" sh`, or use the `arigaio/atlas:latest-extended`
   image. Check the login with `atlas whoami`. If it shows no login, ask the user to run `atlas login`:
   logged out, every command that connects to Oracle aborts with
   `The oracle driver is available only to Atlas Pro users`. A user without an account can start a free
   trial with `atlas login`.
2. Pick the URL scope from what the schema holds:
   - In Oracle, a schema is a user. Schema scope, the default: Atlas manages the schema of the user in
     the URL and writes unqualified statements (`"ORDERS"`). Database scope, `?mode=database`: Atlas
     covers every schema in the PDB that Oracle does not maintain and qualifies each statement
     (`"APP"."ORDERS"`).
   - The URL's path is a PDB service, such as `FREEPDB1`, never the root container.
   - At database scope, a schema the desired state does not declare is planned for `DROP USER`, after
     its objects. Limit the run with `schemas` or `--schema`.
   - Database-scope plans can drop indexes and leave new ones out. Keep tables, indexes, code, and
     grants at schema scope, one env per schema, and use database scope for roles alone (Rule 11).
   - The dev URL takes the target's scope: `docker://oracle/free:latest-lite`, or the same with
     `?mode=database`.
3. The dev database is the Free image, where Atlas connects to `FREEPDB1` as `PDBADMIN`. The lite image
   has no `USERS` tablespace, and `JSON` and `VECTOR` columns fail in its `SYSTEM` tablespace with
   `ORA-43853`. Use a `docker` block with `database = "FREEPDB1"` and an `init` that creates `USERS` and
   makes it the default, plus any other tablespace the schema names (`references/dev-database.md`). Match
   the target's release and `MAX_STRING_SIZE`: a 26ai dev database accepts `BOOLEAN` and `JSON` columns
   that a 19c target rejects. Never point `dev` at a real environment's database.
4. Connect Atlas as the schema owner, never as an administrator. It needs the `CREATE` privileges for
   its object types, `SELECT_CATALOG_ROLE`, `EXECUTE ON SYS.DBMS_LOCK`, and a quota on its tablespace;
   each missing one fails with its own error (`references/dev-database.md`, Privileges of the Atlas User).
5. Names: Oracle folds unquoted names to uppercase, and Atlas quotes every name it writes. Write HCL
   labels in uppercase (`table "ORDERS"`): a lowercase label creates a lowercase name that every query
   must quote. At schema scope, the `schema` block's name is the URL's user. Name every primary key,
   unique constraint, foreign key, and check: for unnamed ones, Atlas copies the dev database's `SYS_C`
   names into the plan, which collide with names on the target (`ORA-02264`) and shift when the schema
   changes. Keep the `SYS_C` names inspection writes for existing constraints (`references/objects.md`).
6. Numbers and sequences:
   - Write integers as `NUMBER(p)`, such as `NUMBER(10)` or `NUMBER(19)`. `INTEGER` and `NUMBER(*,0)`
     are planned as plain `NUMBER`, which accepts decimals, and no later plan reports it.
   - Atlas does not plan a sequence's start, from SQL or HCL. Write `START WITH` in the migration file,
     or, in the declarative workflow, run `ALTER SEQUENCE ... RESTART START WITH n` once by hand. Never
     use `min_value` as the start: once the sequence is used, later plans restart it and repeat numbers.
   - Oracle treats `''` as `NULL`: a backfill of `''` leaves rows `NULL`.
7. Some changes never reach a plan, and the plan still reports success (`references/objects.md`):
   - A rename is planned as a drop and an add, which loses the data: Oracle has no `renamed_from`. Write
     renames by hand (`references/migrations.md`, Renames).
   - Packages, `PARTITION BY`, `INVISIBLE`, and checks of the form `col IS NOT NULL` are left out.
     Create packages, partitioned tables, and invisible columns in hand-written migration files.
   - Materialized views break planning, and existing ones inspect as tables: exclude them by name.
8. PL/SQL:
   - Code that does not compile on the dev database is left out of plans, and in a migration file Oracle
     creates it `INVALID` while Atlas reports it applied. An invalid trigger fails every statement on its
     table with `ORA-04098`. Test it (Rule 12).
   - In hand-written files, end each `CREATE` of PL/SQL with `/` on its own line, or `END;;`.
   - Anonymous blocks (`BEGIN`, `DECLARE`) fail with `PLS-00103`: use plain SQL, `CALL`, or a procedure
     the file creates, calls, and drops. SQL*Plus commands such as `SET` and `EXEC` fail too
     (`references/migrations.md`).
9. Oracle commits before and after every DDL statement, so a file or a plan that fails halfway stays half
   applied, and `--tx-mode` cannot prevent it. To recover, fix the failed statement and the ones after
   it and run `atlas migrate hash`, or undo the applied statements with `atlas migrate down`. Keep DDL
   and data changes in separate, small files (`references/migrations.md`).
10. Busy tables: `DDL_LOCK_TIMEOUT` is 0, so DDL on a table with an open transaction fails at once with
    `ORA-00054`. Set `ALTER SESSION SET DDL_LOCK_TIMEOUT = 30` at the top of the file or in a `hook`.
    Atlas plans `CREATE INDEX` without `ONLINE`, which blocks writes during the build: on a populated
    table, write `CREATE INDEX ... ONLINE` by hand. Add foreign keys and checks `ENABLE NOVALIDATE` and
    validate them in a later file; back a `NOVALIDATE` unique constraint with a non-unique index
    (`references/migrations.md`, Locks and Online Changes).
11. Roles and grants (`references/security.md`): manage roles in an env of their own at database scope,
    with `mode { schemas = false, objects = false, roles = true }`, and grants in the schema-scope env
    with `mode { permissions = true }`, naming roles as strings and creating them in the dev `init`.
    Roles are database-wide and the `role` block has no `external`, so set `drop_role` in the `skip`
    diff policy, or other teams' roles are planned for a drop. Users, system privileges, and schema
    privileges are not managed.
12. Test every change before reporting it done, as a habit. Run `atlas schema validate`, add a
    `test "schema"` case that calls each function and procedure (`SELECT ... FROM dual`, `CALL`) and
    exercises each trigger, view, and check, and a `test "migrate"` case with seeded rows for each
    migration that moves or converts data. For every migration that creates PL/SQL, assert that
    `SELECT COUNT(*) FROM user_objects WHERE status = 'INVALID'` returns 0. Run them with
    `atlas schema test` and `atlas migrate test` (`references/testing.md`).
13. When no built-in analyzer covers a check, write a custom lint rule in the HCL rule language, and do
    the same for a convention the user states, such as an index on every foreign key, uppercase names,
    or character length semantics. Oracle has no analyzers of its own, so these rules carry the Oracle
    checks (`references/custom-rules.md`).

## Quick Answers

| Question | Answer |
|----------|--------|
| A local Oracle database to apply migrations to | `atlas tool docker --url "docker://oracle/free:latest-lite" --name my-ora` prints its URL; `atlas tool docker kill --name my-ora` stops it (`references/dev-database.md`) |
| `The oracle driver is available only as custom build`, or `Error: timeout:` with that text | Install the Oracle build: `curl -sSf https://atlasgo.sh \| ATLAS_FLAVOR="oracle" sh` (Rule 1) |
| `The oracle driver is available only to Atlas Pro users` | Ask the user to run `atlas login` (Rule 1) |
| `ORA-43853: JSON type cannot be used in non-automatic segment space management tablespace "SYSTEM"` | Create a `USERS` tablespace in the dev `docker` block's `init` (`references/dev-database.md`) |
| `ORA-00959: tablespace '...' does not exist` on the dev database | Create the target's tablespaces in the dev `init` (`references/hcl.md`, Tables) |
| `the pluggable database (PDB) name is required for Oracle containers` | Add `database = "FREEPDB1"` to the `docker` block (`references/dev-database.md`) |
| `query server property: ORA-00942: table or view "SYS"."V_$INSTANCE" does not exist` | Grant `SELECT_CATALOG_ROLE` to the Atlas user (`references/dev-database.md`, Privileges of the Atlas User) |
| `PLS-00201: identifier 'DBMS_LOCK' must be declared` | Grant `EXECUTE ON SYS.DBMS_LOCK`, on the target and on a dev user you provide (`references/dev-database.md`) |
| `ORA-01950: ... insufficient quota on tablespace` | Give the user a quota: `ALTER USER APP QUOTA UNLIMITED ON USERS` (`references/dev-database.md`) |
| `mismatched HCL and database schemas: "APP" <> "app"` | At schema scope, the `schema` block's name is the URL's user, in uppercase (`references/hcl.md`) |
| `ORA-02264: name already used by an existing constraint`, or `the planned state does not match the desired state` | Name every constraint in the schema (`references/objects.md`, Constraint Names) |
| A plan drops other schemas (`DROP USER`) | Database scope covers every schema: limit it with `schemas`, or use schema scope (`references/dev-database.md`) |
| A database-scope plan drops indexes | Plan tables and indexes at schema scope (`references/dev-database.md`, Scope) |
| A column declared `INTEGER` accepts decimals | Atlas planned it as `NUMBER`: use `NUMBER(p)` (`references/objects.md`) |
| A sequence starts at 1 instead of its `START WITH` | Write `START WITH` in a migration file, or run `ALTER SEQUENCE ... RESTART START WITH n` once (`references/objects.md`) |
| A plan contains `ALTER SEQUENCE ... RESTART` | The schema sets `min_value` as a start: do not apply it. Drop `min_value` from the schema; the next plan is `ALTER SEQUENCE ... NOMINVALUE`, which keeps the current value (`references/objects.md`) |
| `mismatched schema names: "PDBADMIN" != "APP"` in `atlas schema test` | Name the HCL schema from a variable, and test with `--var schema_name=PDBADMIN` (`references/hcl.md`) |
| A rename shows up as a drop and an add | Rename by hand in a migration file (`references/migrations.md`, Renames) |
| Lint reports `DS103` for a `RENAME COLUMN` | Expected: add `-- atlas:nolint DS103 BC104`, plus `MF103` for a `NOT NULL` column, after checking (`references/migrations.md`) |
| A package, partitioning, or an `INVISIBLE` column is missing from the plan | Atlas leaves them out: write them in migration files (`references/objects.md`) |
| `ORA-12083: must use DROP MATERIALIZED VIEW` | A materialized view in the schema: exclude it, and create it by hand (`references/objects.md`) |
| A function, procedure, or trigger is missing from the plan | It does not compile on the dev database: fix it, and test it (`references/testing.md`) |
| `ORA-04098: trigger '...' is invalid and failed re-validation` after a migration | The trigger did not compile; check `user_errors`, and add the invalid-objects test (`references/testing.md`) |
| `PLS-00103: Encountered the symbol "end-of-file"` on a `BEGIN` or `DECLARE` block | Atlas cannot run anonymous blocks: use `CALL` or a temporary procedure (`references/migrations.md`) |
| `ORA-00922` on `SET ...`, or `ORA-00900` on `EXEC` | SQL*Plus commands: remove them, and use `CALL` (`references/migrations.md`) |
| A migration failed halfway, and `migrate status` shows `(last one partially)` | Fix the rest of the file and run `atlas migrate hash`, or `atlas migrate down` (`references/migrations.md`) |
| `ORA-00054: Failed to acquire a lock` (19c: `resource busy and acquire with NOWAIT specified`) | Set `DDL_LOCK_TIMEOUT`, or retry when traffic is low (`references/migrations.md`, Locks) |
| An index build blocks writes | Write `CREATE INDEX ... ONLINE` by hand (`references/migrations.md`) |
| `ORA-02299: cannot validate (...) - duplicate keys found` on a `NOVALIDATE` unique constraint | Back it with a non-unique index: `unique { index = index.X, validated = false }` (`references/hcl.md`) |
| `ORA-02296: cannot enable (...) - null values found` | Rows are `NULL`, or were backfilled with `''`, which is `NULL` (`references/objects.md`) |
| `ORA-00902: invalid datatype` on the target only | The dev database is a newer release than the target (`references/dev-database.md`) |
| `ORA-00910: specified length too long for its datatype` | `MAX_STRING_SIZE` differs between the dev database and the target (`references/dev-database.md`) |
| `ORA-43929: Collation cannot be specified if parameter MAX_STRING_SIZE=STANDARD is set` | Column collations need `MAX_STRING_SIZE = EXTENDED` (`references/hcl.md`) |
| `ORA-41900: missing COMMENT privilege` | Database scope on a 23ai or later dev image: grant `COMMENT ANY TABLE` to `PDBADMIN` in `init` (`references/dev-database.md`) |
| `ORA-01924: Role "X" not granted or does not exist` | Another account created the role: grant it to the Atlas user `WITH ADMIN OPTION` (`references/security.md`) |
| A plan drops roles the schema does not declare | Set `drop_role = true` in the `skip` diff policy (`references/security.md`) |
| `ORA-01917: user or role '...' does not exist` on the dev database | Create the roles that grants name in the dev `init` (`references/security.md`) |
| Lint does not flag blocking DDL on Oracle | Oracle has no analyzers of its own: add a `statement` rule and custom rules (`references/migrations.md`) |

## Documentation

- Oracle HCL reference: https://atlasgo.io/hcl/oracle
- Oracle guides: https://atlasgo.io/guides/oracle/automatic-migrations,
  https://atlasgo.io/guides/oracle/vector
- Security: https://atlasgo.io/guides/oracle/security-declarative,
  https://atlasgo.io/guides/oracle/security-versioned
- Dev database: https://atlasgo.io/concepts/dev-database
- Lint checks: https://atlasgo.io/lint/analyzers
- Oracle SQL Language Reference: https://docs.oracle.com/en/database/oracle/oracle-database/26/sqlrf/
- Guides for each database: https://atlasgo.io/databases
