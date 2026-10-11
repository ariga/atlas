---
name: atlas-spanner
description: "Google Cloud Spanner specifics for Atlas: the extended build and login, URLs and Application Default Credentials, GoogleSQL and PostgreSQL dialects, the emulator dev database and what it lacks, `-- atlas:txmode none` in every migration file, partial failures and long-running schema changes (index backfills, validations, Google's batch limits), what Spanner cannot change in place (primary keys, interleaving, column types, renames), objects Atlas leaves out or plans wrongly (identity columns, NOT ENFORCED foreign keys, interleaved indexes, change streams, database options), hotspot-free keys and bit-reversed sequences, named schemas, roles and grants, migration tests in place of schema tests, mutation limits and batched backfills, and custom lint rules. Use with the atlas skill whenever Atlas manages a Spanner database."
---

# Atlas for Spanner

What is specific to Google Cloud Spanner when Atlas manages the schema. The `atlas` skill covers the
workflows and commands; this skill covers the rules, edge cases, and answers that differ on Spanner. It
covers both dialects, GoogleSQL and PostgreSQL. Behavior was checked against the Spanner emulator
1.5.58 and Google's Spanner documentation.

## Files

This skill is `SKILL.md` plus seven reference files under `references/`, each served at
`https://atlasgo.io/skills/atlas-spanner/references/<name>`. Read a reference when the task needs it:

| File | Read it for |
|------|-------------|
| `references/dev-database.md` | Scope and named schemas, `docker://spanner` and `docker://spannerpg`, what the emulator lacks, your own dev database on a real instance, URLs, credentials, and IAM, running an emulator by hand |
| `references/objects.md` | What Atlas leaves out or plans wrongly, keys and hotspots, interleaving, type changes, indexes, foreign keys, checks and generated columns, sequences, views, change streams, TTL, named schemas, Enterprise features, the PostgreSQL dialect |
| `references/hcl.md` | Writing the schema in HCL: HCL or SQL, `STRING(MAX)`, generated columns and options, `interleave`, indexes, sequences, views, change streams, locality groups and placements, property graphs |
| `references/migrations.md` | `-- atlas:txmode none`, partial failures and recovery, long-running changes and Google's limits, lint on Spanner, dependent objects, changes to write by hand, renames, data changes and mutation limits, the revisions table, baselines |
| `references/security.md` | Database roles and grants, why they need a real Spanner dev database, `DEFINER` views, IAM bindings outside Atlas |
| `references/testing.md` | Testing every change with migration tests, testing a declarative schema, what to seed, matching errors, CI |
| `references/custom-rules.md` | Writing a custom lint rule in the HCL rule language: what Spanner objects expose, and tested rules for hotspot keys and indexes |

## Requires

The `atlas` skill, installed next to this one (`../atlas/SKILL.md`). If it is missing, download
https://atlasgo.io/skills/atlas/SKILL.md to `../atlas/SKILL.md` and each reference file listed in its
Files section to `../atlas/references/<name>`.

## Rules

1. The Spanner driver ships in the extended build of Atlas and needs a login. If a command fails with
   `The spanner driver is available only as custom build`, install that build:
   `curl -sSf https://atlasgo.sh | ATLAS_FLAVOR="spanner" sh`, or use the `arigaio/atlas:latest-extended`
   image. Check the login with `atlas whoami`. If it shows no login, ask the user to run `atlas login`:
   logged out, every command that connects to Spanner aborts with
   `The spanner driver is available only to Atlas Pro users`. A user without an account can start a free
   trial with `atlas login`.
2. URLs and scope (`references/dev-database.md`):
   - The URL is `spanner://projects/<project>/instances/<instance>/databases/<database>`. It carries no
     credentials: Atlas uses Application Default Credentials (`gcloud auth application-default login`, or
     the CI runner's service account).
   - The URL covers the whole database: the default schema (`schema "default"` in HCL, `public` in the
     PostgreSQL dialect) and every named schema. A named schema the desired state leaves out is planned
     for a drop: declare it, or limit the run with `schemas`, `--schema`, or `exclude`.
   - `migrate apply` keeps its revisions table in the named schema `atlas_schema_revisions`, which a
     declarative plan on the same database drops unless excluded.
3. The dev database is the emulator: `docker://spanner/<version>` for GoogleSQL, or
   `docker://spannerpg/<version>` for the PostgreSQL dialect. Pin a release, such as `1.5.58`: `latest` is
   whatever the machine pulled last. The emulator ignores roles and grants, treats placements as no-ops,
   loses some locality group and vector index options, and does not enforce mutation limits. For roles
   and grants, point `dev` at an empty database on a real Spanner instance. Never point `dev` at a real
   environment's database.
4. Every migration file starts with `-- atlas:txmode none`. `atlas migrate diff` writes it, but
   `atlas migrate new` does not, and a hand-written file without it fails on the target with
   `cannot execute DDL as part of a transaction`. `migrate lint` and `migrate validate` miss it;
   `migrate test` catches it. `atlas schema clean` and `atlas schema test` fail with the same error on
   Spanner: reset a database by applying an empty schema, and test with migration tests (Rule 12).
5. Spanner has no transactional DDL, and Atlas sends one statement at a time, each its own schema change.
   A file or a plan that fails halfway stays half applied. To recover, fix the failed statement and the
   ones after it and run `atlas migrate hash`, or undo the applied statements with `atlas migrate down`.
   Statements that backfill or validate every row (`CREATE INDEX` on a populated table, `NOT NULL`, a
   shorter `STRING(n)`, a new foreign key or check, a stored generated column) run from minutes to
   hours: keep them in small files of their own. Google advises at most 10 of them per batch and at
   most 3 new backfilled indexes a day. Set the `if_not_exists` diff policy, so a rerun after a dead job
   skips a statement Spanner finished (`references/migrations.md`).
6. Changes Spanner cannot make in place, each a new column or table, a copy, and a later drop
   (`references/objects.md`):
   - A primary key: planning fails with `the keys of a table cannot change after table creation`.
   - Interleaving is permanent: adding or removing it on an existing table plans no change at all.
   - A type change other than a new length or a conversion between `STRING` and `BYTES`: planning fails
     with `Cannot change type of column`.
   - A new `NOT NULL` column needs a `DEFAULT`, or planning fails with `Cannot add NOT NULL column`.
   - A renamed table is planned as a drop and a create: write `ALTER TABLE ... RENAME TO ...` by hand.
     Columns cannot be renamed.
7. Read every plan against the schema change: Atlas plans these without an error
   (`references/objects.md`, What Atlas Leaves Out):
   - Identity columns become a plain `INT64` without a generator, and a table without a primary key gets
     a plain `rowid` key. Declare keys, and generate `INT64` keys from a named bit-reversed sequence:
     `DEFAULT (GET_NEXT_SEQUENCE_VALUE(SEQUENCE OrderSeq))`.
   - `NOT ENFORCED` foreign keys become enforced, and interleaved indexes become global indexes. Write
     both by hand.
   - Database options and synonyms are left out. Write them by hand.
   - A change to a change stream drops and recreates it, which deletes its change records. Write
     `ALTER CHANGE STREAM` by hand, and flag `DROP CHANGE STREAM` with a `statement` lint rule.
   - Removing a column together with the index, `STORING` list, or view that uses it plans the
     `DROP COLUMN` first, which Spanner rejects. `migrate diff` writes the file anyway; `migrate lint`
     fails on it. Move the dependent change first and run `atlas migrate hash`
     (`references/migrations.md`, Dependent Objects).
   - A table interleaved with `INTERLEAVE IN`, without `PARENT`, breaks inspection, and a sequence in a
     named schema breaks planning.
8. Keys: never start a primary key or a frequently written index with a value that only grows, such as a
   timestamp, a commit timestamp, or an id the application increments. All new rows land on one split.
   Use a UUID version 4 (`GENERATE_UUID()`, or the `UUID` type with `NEW_UUID()`) or a bit-reversed
   sequence, and enforce it with the custom rules in `references/custom-rules.md`.
9. HCL (`references/hcl.md`): name files `*.sp.hcl`. Write `sql("STRING(MAX)")`, since `STRING(MAX)` is
   not an HCL type. Always set `type = IN_PARENT` in an `interleave` block: without it Atlas writes
   invalid DDL. A primary key with a descending part plans only from SQL: in HCL it plans
   `PRIMARY KEY ()`. There is no `data` block, `renamed_from`, or identity column. Declare vector indexes,
   locality groups, and placements in HCL: a SQL schema loses their options on the emulator.
10. Roles and grants (`references/security.md`): enable them with
    `mode { roles = true, permissions = true }`, and plan them on a dev database on a real Spanner
    instance. Declare every grantee as a `role`: Spanner has no database users, and the `user` block plans
    nothing. Which IAM principals may act as a role is an IAM binding outside Atlas.
11. Data changes: one DML statement makes up to 80,000 mutations, counting index entries, and the emulator
    does not enforce it. Backfill large tables in batches with `atlas script loop`, or with Partitioned
    DML outside Atlas. Seed rows go in migration files: there is no `data` block on Spanner
    (`references/migrations.md`, Data Changes).
12. Test every change before reporting it done, as a habit. Run `atlas schema validate`, lint the change
    (`atlas migrate lint`, or the plan of `atlas schema apply --dry-run`), then write `test "migrate"`
    cases: `atlas schema test` does not run on Spanner.
    Seed rows for each change that validates data: `NULL`s before `NOT NULL`, long values before a
    shorter `STRING(n)`, duplicates and two `NULL`s before a unique index, orphans before a foreign key.
    Run them with `atlas migrate test`. A declarative project tests through a throwaway migration
    directory (`references/testing.md`).
13. When no built-in analyzer covers a check, write a custom lint rule in the HCL rule language, and do
    the same for a convention the user states. Spanner has no analyzers of its own, so these rules carry
    the Spanner checks, such as hotspot keys (`references/custom-rules.md`).
14. PostgreSQL dialect (`references/objects.md`, PostgreSQL Dialect): plan on `docker://spannerpg/<version>`.
    A TTL breaks planning, and a `spanner.commit_timestamp` column breaks every plan after the first.
    `numeric`, `jsonb`, and `float4` columns cannot be keys or be indexed.

## Quick Answers

| Question | Answer |
|----------|--------|
| A local Spanner database to apply migrations to | `atlas tool docker --url "docker://spanner/1.5.58" --name my-spanner` prints its URL; `atlas tool docker kill --name my-spanner` stops it (`references/dev-database.md`) |
| `The spanner driver is available only as custom build` | Install the extended build: `curl -sSf https://atlasgo.sh \| ATLAS_FLAVOR="spanner" sh` (Rule 1) |
| `The spanner driver is available only to Atlas Pro users` | Ask the user to run `atlas login` (Rule 1) |
| `credentials: could not find default credentials` | Run `gcloud auth application-default login`, or give the runner a service account (`references/dev-database.md`) |
| `cannot execute DDL as part of a transaction` from `migrate apply` | Add `-- atlas:txmode none` at the top of the file and run `atlas migrate hash` (`references/migrations.md`) |
| `cannot execute DDL as part of a transaction` from `schema test` or `schema clean` | Neither runs on Spanner: write migration tests, and reset with an empty schema (Rule 4) |
| `the keys of a table cannot change after table creation` | Create a new table with the new key, copy the rows, then drop the old one (`references/objects.md`) |
| Adding or removing `INTERLEAVE IN PARENT` plans no change | Interleaving is permanent: create a new table and copy the rows (`references/objects.md`, Interleaving) |
| ``Cannot change type of column `X` from `INT64` to `STRING` `` | Add a new column, backfill it, and drop the old one later (`references/objects.md`) |
| `Cannot add NOT NULL column ... to existing table` | Give the new column a `DEFAULT` (`references/objects.md`) |
| `Cannot specify a null value for column` when a column becomes `NOT NULL` | Backfill the `NULL` rows first: `ALTER COLUMN` does not use the default (`references/objects.md`) |
| `Reducing the length of column ... is not allowed` | Shorten the longer values first; lint does not report this change (`references/objects.md`) |
| `Cannot drop column ... because it is used by index`, or `on which there are dependent views` | Move the `DROP INDEX` or the view change above the `DROP COLUMN` (`references/migrations.md`, Dependent Objects) |
| `Requested change to key column ... could not be made` | Interleaved children inherit the key column, so it cannot change (`references/objects.md`) |
| `Scan error on column index 3, name "on_delete_action"` | A table uses `INTERLEAVE IN` without `PARENT`: convert it with `ALTER TABLE ... SET INTERLEAVE IN PARENT ...` through `gcloud` (`references/objects.md`, Interleaving) |
| `Duplicate name in schema` or `Duplicate column name` on a rerun | The statement finished before the job died: set the `if_not_exists` policy, or remove the statement (`references/migrations.md`) |
| A plan drops `atlas_schema_revisions` or another named schema | The URL covers every schema: declare it, or exclude it (`references/dev-database.md`, Scope) |
| An identity column, a `NOT ENFORCED` foreign key, or an interleaved index is missing from the plan | Atlas leaves those clauses out: use a sequence, or write them by hand (`references/objects.md`) |
| A change stream is dropped and recreated | Write `ALTER CHANGE STREAM ... SET OPTIONS` or `SET FOR` by hand (`references/migrations.md`) |
| A renamed table shows up as a drop and a create | Write `ALTER TABLE Old RENAME TO New, ADD SYNONYM Old` by hand, after `-- atlas:nolint DS102 BC103`; in the declarative workflow, edit the plan with `schema apply --edit` (`references/migrations.md`, Renames) |
| `an error is expected, but none occurred` from a `catch` on a `SELECT` | `catch` does not see query errors on Spanner: check with `exec` and `output` over `INFORMATION_SCHEMA` (`references/testing.md`, Matching Errors) |
| `Drop sequence "X": ... Sequence not found` | The sequence is in a named schema: keep sequences in the default schema (`references/objects.md`) |
| `There is no type named "MAX"` | Write `sql("STRING(MAX)")` (`references/hcl.md`) |
| `Syntax error ... Expecting 'IN'` on an `INTERLEAVE` clause | Add `type = IN_PARENT` to the `interleave` block (`references/hcl.md`) |
| A plan creates `PRIMARY KEY ()` | An HCL primary key with `on` blocks: keep descending keys in a SQL schema (`references/hcl.md`) |
| `unknown block type data in spanner schema` | Seed rows with `INSERT` statements in a migration file (`references/migrations.md`) |
| `Table not found: INFORMATION_SCHEMA.TABLE_PRIVILEGES` | Roles and grants need a dev database on a real Spanner instance (`references/security.md`) |
| `modify index "X" must have at least one column to add or drop` | A vector index: exclude it once it exists (`references/objects.md`, Indexes) |
| `cannot add and drop a column at the same time` | Change one `STORING` column per migration (`references/objects.md`, Indexes) |
| ``View `V` is missing the SQL SECURITY clause`` | Add `SQL SECURITY INVOKER`, or `DEFINER` (`references/objects.md`, Views) |
| `Expecting 'INVOKER' but found 'DEFINER'` | Pin emulator 1.5.56 or later (`references/dev-database.md`) |
| `connected database is not clean: found schema "default". baseline version or allow-dirty is required` | Create a baseline migration and pass `--baseline` (`references/migrations.md`, Existing Databases) |
| `BadUsage` from a large `UPDATE` or `DELETE` | More than 80,000 mutations in one statement: batch it with `atlas script loop`, or use Partitioned DML (`references/migrations.md`) |
| `unsupported type` on the PostgreSQL dialect | A `spanner.commit_timestamp` column (`references/objects.md`, PostgreSQL Dialect) |
| `is not supported in Spanner PostgreSQL dialect` | Atlas cannot plan a PostgreSQL-dialect TTL: exclude the table (`references/objects.md`) |
| `Cannot reference PG.NUMERIC ... in the creation of index` | PostgreSQL-dialect `numeric`, `jsonb`, and `float4` columns cannot be indexed (`references/objects.md`) |
| A convention must hold on every table, such as hotspot-free keys | Write a custom lint rule (`references/custom-rules.md`) |

## Documentation

- Spanner HCL reference: https://atlasgo.io/hcl/spanner
- Spanner guides: https://atlasgo.io/guides/spanner/automatic-migrations,
  https://atlasgo.io/guides/spanner/proto-columns
- Dev database: https://atlasgo.io/concepts/dev-database
- Lint checks: https://atlasgo.io/lint/analyzers
- Spanner schema design: https://docs.cloud.google.com/spanner/docs/schema-design
- Spanner schema updates: https://docs.cloud.google.com/spanner/docs/schema-updates,
  https://docs.cloud.google.com/spanner/docs/schema-updates-best-practices
- Spanner DDL: https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language
- Guides for each database: https://atlasgo.io/databases
