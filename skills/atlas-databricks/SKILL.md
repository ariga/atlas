---
name: atlas-databricks
description: "Databricks specifics for Atlas: the extended binary and login, connection URLs and scope (catalog vs schema), the dev catalog (no Docker), Delta table properties, managed and external tables, partitioning and liquid clustering, generated and identity columns, defaults and column mapping, renames, informational keys and checks, collations, column masks and row filters, views and materialized views, SQL and Python functions, procedures, Unity Catalog grants, no transactions and no locks, partial failures, schema and migration tests, and custom lint rules. Use with the atlas skill whenever Atlas manages a Databricks database."
---

# Atlas for Databricks

What is specific to Databricks when Atlas manages the schema. The `atlas` skill covers the workflows and
commands; this skill covers the rules, edge cases, and answers that differ on Databricks. Atlas connects
through a SQL warehouse and manages Unity Catalog objects: schemas, Delta tables, views, materialized
views, functions, procedures, and grants.

## Files

This skill is `SKILL.md` plus seven reference files under `references/`, each served at
`https://atlasgo.io/skills/atlas-databricks/references/<name>`. Read a reference when the task needs it:

| File | Read it for |
|------|-------------|
| `references/dev-database.md` | Installing the extended binary, connection URLs and their scope, the dev catalog and its clean checks, what normalization keeps |
| `references/security.md` | Grants on schemas, tables, views, and routines, Unity Catalog principals, privileges, grants on objects the schema does not declare, masks and row filters |
| `references/objects.md` | Delta properties, managed and external tables, partitioning and clustering, keys and checks, generated and identity columns, defaults, column changes, renames, collations, types, views, materialized views, routines, what Atlas does not manage |
| `references/hcl.md` | Writing the schema in HCL: the `NOT NULL` default, types, every Databricks block and attribute, views, materialized views, functions, procedures, permissions |
| `references/migrations.md` | Lint codes on Databricks, `--tx-mode none` and the `atlas:txmode none` directive, partial failures and recovery, no locks, the revisions table, baselines |
| `references/testing.md` | Testing every change: why `atlas schema test` does not run on Databricks, migration tests with seeded rows that also cover functions, views, masks, and checks, CI |
| `references/custom-rules.md` | Writing a custom lint rule in the HCL rule language when no analyzer covers a check: predicates, operators, what each Databricks object exposes, examples |

## Requires

The `atlas` skill, installed next to this one (`../atlas/SKILL.md`). If it is missing, download
https://atlasgo.io/skills/atlas/SKILL.md to `../atlas/SKILL.md` and each reference file listed in its
Files section to `../atlas/references/<name>`.

## Rules

1. Check the login first with `atlas whoami`. If it shows no login, ask the user to run `atlas login`:
   logged out, every command that opens a Databricks URL fails, so nothing is skipped silently. The
   driver ships only in the extended binary (`ATLAS_FLAVOR="databricks"` for the install script, or
   the `arigaio/atlas:latest-extended` image). A user without an account can start a free trial with
   `atlas login`.
2. Pick the URL scope from what the schema holds:
   - Schema scope, a URL with `schema`
     (`databricks://<token>@<host>:443<warehouse-path>?catalog=<catalog>&schema=<schema>`): Atlas
     manages that schema and writes unqualified statements. Catalog scope, no `schema`: Atlas covers
     every schema of the catalog and qualifies each statement (`` `default`.`users` ``).
   - Atlas has no catalog level: no statement names a catalog, so the same migrations apply to any
     catalog. Give each environment its own catalog through the `catalog` parameter.
   - At catalog scope, a schema the desired state does not declare, `default` included, is planned for
     `DROP SCHEMA ... CASCADE`. Declare `schema "default" {}`, or use schema scope.
   - The dev URL takes the target's scope: `?catalog=atlas_dev&schema=default`, or `?catalog=atlas_dev`.
3. There is no Docker dev database. Use a dedicated, empty catalog on the same workspace. At catalog
   scope it must keep its `default` schema, and every schema in it must be empty. Never point `dev` at
   a catalog that holds data (`references/dev-database.md`).
4. Databricks runs no statement in a transaction, and its SQL driver cannot open one. `schema apply`
   and `migrate apply` default to `--tx-mode file` and fail with an error that starts with
   `starting transaction:` before running anything. Run both with `--tx-mode none`, or set
   `tx_mode = "none"` in the env's `migration` block. Start every migration file with
   `-- atlas:txmode none` and a blank line, or `atlas migrate test` fails the same way. Each statement
   commits on its own, so a file or plan that fails halfway stays half applied; keep files small and
   test them first (`references/migrations.md`).
5. Atlas takes no lock on Databricks: `--lock-timeout` has no effect, and two deployments to one target
   can run at the same time. Serialize deployments in CI, with one concurrency group per target.
6. Databricks writes `delta.*` properties onto every table. Atlas compares only the properties the
   desired state lists, and removing a property from the schema plans nothing: write
   `ALTER TABLE ... UNSET TBLPROPERTIES` by hand. After `schema inspect`, keep only the properties the
   team owns (`references/objects.md`, Table Properties).
7. Some clauses exist only at table creation, and Atlas refuses to plan a change to them with an error
   ending in `is set on creation only`: identity and generated columns, `partition`, and `provider`.
   Changing them means a new table and a hand-written data copy. Clustering, location, comments, and
   properties change in place (`references/objects.md`).
8. Atlas plans no renames on Databricks. The HCL has no `renamed_from`, so a renamed column or table is
   planned as a drop and an add, and its data is lost. Write renames by hand in a migration file. Other
   column changes expand into several statements: a drop first sets
   `'delta.columnMapping.mode' = 'name'`, and a new column with a default or `NOT NULL` is added plain,
   then altered (`references/objects.md`, Renames).
9. Foreign keys are informational: Databricks enforces none, and Atlas writes no `ON DELETE` or
   `ON UPDATE`. A changed key is dropped and re-added. A check changed under the same name, or a
   renamed primary key, fails at planning: give the check a new name, and rename the key by hand
   (`references/objects.md`, Keys and Constraints).
10. Review every plan that touches a materialized view. A change to its query, columns, comment, or
    storage clauses rewrites it with `CREATE OR REPLACE MATERIALIZED VIEW`; only its refresh schedule,
    row filter, and column comments and masks change in place. A plain view's query changes in place
    with `ALTER VIEW ... AS` (`references/objects.md`).
11. Grants are planned only with `mode { permissions = true }` in the env's `schema` block. Grantees are
    Unity Catalog principals that already exist; Atlas creates none. Once the mode is on, a grant the
    desired state does not declare on a managed object is revoked (`references/security.md`).
12. Test every change before reporting it done, as a habit. Run `atlas schema validate`, then
    `atlas migrate test`: `atlas schema test` does not run on Databricks, so write `test "migrate"`
    cases for everything. Add one for each function, procedure, mask, row filter, view, or check you
    add or change, and one with seeded rows for each migration that moves, converts, or drops data
    (`references/testing.md`).
13. When no built-in analyzer covers a check, write a custom lint rule in the HCL rule language, and do
    the same for a convention the user states, such as a comment on every table or no Python functions.
    Lint enforces it on every change (`references/custom-rules.md`).

## Quick Answers

| Question | Answer |
|----------|--------|
| `The databricks driver is available only as custom build.` | Install the extended binary with `ATLAS_FLAVOR="databricks"` (Rule 1) |
| `login is required to use the 'databricks' driver` | Run `atlas login`, or `atlas login --token` in CI (Rule 1) |
| Which URL to give Atlas | `databricks://<token>@<host>:443/sql/1.0/warehouses/<id>?catalog=<catalog>`, with `&schema=<schema>` for schema scope (`references/dev-database.md`, Connection URLs) |
| A dev database for Databricks | A dedicated empty catalog: `CREATE CATALOG IF NOT EXISTS atlas_dev;` (`references/dev-database.md`) |
| `connected database is not clean: found table "..." in schema "..."` | The dev catalog is not empty, or `dev` points at a real catalog (`references/dev-database.md`, Errors) |
| `connected database is not clean: default schema "default" not found` | The dev catalog lost its `default` schema: create it again (`references/dev-database.md`, Errors) |
| `connected database is not clean: ... baseline version or allow-dirty is required` | `migrate apply` on a target that has objects but no revisions table: baseline it (`references/migrations.md`, Existing Databases) |
| `databricks: schema "..." was not found` | The URL's `schema` parameter names a schema that does not exist in that catalog (`references/dev-database.md`, Errors) |
| `starting transaction: ...` from `schema apply` or `migrate apply` | Add `--tx-mode none`, or `tx_mode = "none"` in the `migration` block (Rule 4) |
| The same error from `atlas migrate test` | The file has no `-- atlas:txmode none` header: add it to every file, then `atlas migrate hash` (`references/testing.md`) |
| The same error from every case of `atlas schema test` | Schema tests do not run on Databricks: cover the change with `test "migrate"` cases (`references/testing.md`, Schema Tests Do Not Run) |
| A plan drops schemas with `DROP SCHEMA ... CASCADE` | Catalog scope covers every schema: declare them, limit the run with `--schema`, or use schema scope (Rule 2) |
| A table property was removed from the schema, and the plan has no change | Expected: write `ALTER TABLE ... UNSET TBLPROPERTIES` by hand (Rule 6) |
| The migration's `CREATE TABLE` lists every `delta.*` property | The desired state lists them: a SQL source, or an inspected HCL file. Trim them (`references/objects.md`, Table Properties) |
| `the PARTITIONED BY clause of a table is set on creation only` | Partitioning cannot change in place: create a new table and copy the data (Rule 7) |
| `a generated column is set on table creation only`, or `the generated clause of a column is set on creation only` | Identity and generated columns exist only at table creation: use a new table, or a plain column (Rule 7) |
| `unsupported type: "VARIANT"` | Atlas inspects `VARIANT`, `INTERVAL`, `GEOGRAPHY`, `GEOMETRY`, and `OBJECT` columns but cannot create or alter them (`references/objects.md`, Types) |
| A rename shows up as a drop and an add, or `unknown attribute renamed_from in block of type column` | Databricks has no `renamed_from`: write the rename in a migration file (`references/objects.md`, Renames) |
| `unsupported modify table change` after editing a check or renaming a primary key | Give the changed check a new name; rename the key in a hand-written migration (`references/objects.md`, Keys and Constraints) |
| Lint reports `CD101` or `CD103` on a key that only changed | Atlas drops and re-adds a modified foreign or primary key (Rule 9) |
| Lint reports `DS103` on a hand-written `RENAME COLUMN` | Lint reads the rename as a drop and an add (`references/migrations.md`, What Lint Catches on Databricks) |
| `unknown attribute default in block of type column`, or `unknown block type primary_key in block of type view` | A plain view's columns take only `type`, `null`, and `comment`; masks, keys, and expectations belong to `materialized` (`references/hcl.md`, Views) |
| Grants never show up in inspection or plans | Turn on `mode { permissions = true }` (`references/security.md`, Turn It On) |
| `unknown attribute grantable in block of type permission` | Databricks has no `WITH GRANT OPTION`: remove `grantable` (`references/security.md`) |
| `cluster auto: must be true, or omit the block`, or `refresh on_update: must be true, or omit the block` | Omit the block instead of setting it to `false` (`references/hcl.md`) |
| A migration failed halfway, and `migrate status` shows it as partially applied | The statements before the failure stayed applied: fix the rest and run `atlas migrate hash` (`references/migrations.md`, Transactions) |
| `ambiguous revision table` after changing the URL's scope | Set `--revisions-schema` to the schema that holds the existing table (`references/migrations.md`, Revisions Table) |
| `LANGUAGE PYTHON is not a clause of CREATE PROCEDURE`, or `DETERMINISTIC is not a clause of CREATE PROCEDURE` | Procedures are SQL only, and take `deterministic = false` at most (`references/objects.md`, Functions and Procedures) |

## Documentation

- Databricks HCL reference: https://atlasgo.io/hcl/databricks
- Dev database: https://atlasgo.io/concepts/dev-database
- Connection URLs and scope: https://atlasgo.io/concepts/url
- Databricks guide: https://atlasgo.io/guides/databricks/automatic-migrations
- Lint checks: https://atlasgo.io/lint/analyzers
- Guides for each database: https://atlasgo.io/databases
