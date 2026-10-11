---
name: atlas-snowflake
description: "Snowflake specifics for Atlas: the extended build and login, URL scope (database vs schema), the dev database on the same account, identifier case, account and database roles, users and grants, table kinds (transient, dynamic, hybrid, external, Iceberg), stages, pipes, tasks, alerts, tags and policies, secrets, warehouses and other account-level objects, changes Snowflake makes only by recreating and the allow_recreate policy, no transactional DDL and no apply lock, TX101 and DS lint findings, schema and migration tests, and custom lint rules. Use with the atlas skill whenever Atlas manages a Snowflake database."
---

# Atlas for Snowflake

What is specific to Snowflake when Atlas manages the schema. The `atlas` skill covers the workflows
and commands; this skill covers the rules, edge cases, and answers that differ on Snowflake. Snowflake
is a managed service: there is no Docker dev database, and roles, users, warehouses, and integrations
live on the account, outside any database.

## Files

This skill is `SKILL.md` plus seven reference files under `references/`, each served at
`https://atlasgo.io/skills/atlas-snowflake/references/<name>`. Read a reference when the task needs it:

| File | Read it for |
|------|-------------|
| `references/dev-database.md` | Installing the extended build, URLs and their scope, the dev database on the same account, account-level objects during planning, authentication, connection errors |
| `references/security.md` | Account roles, database roles, users, `permission` blocks, `ref()` grants in composite schemas, excluding other teams' grants, what Atlas does not manage |
| `references/objects.md` | Which objects need a `mode` switch, table kinds, changes Snowflake makes only by recreating, column changes and renames, views, functions, tasks, pipes, stages, policies, tags, secrets, objects Atlas does not model |
| `references/hcl.md` | Writing the schema in HCL: identifier case, the `NOT NULL` default, types, defaults and identity, constraints, clustering and retention, each object block, tags and policy attachments, seed data, input variables |
| `references/migrations.md` | Lint codes on Snowflake, partial failures and recovery, no apply lock, the revisions table, recreations and dropped grants, `migrate down`, CI timing |
| `references/testing.md` | Testing every change: schema tests for functions, procedures, views, and dynamic tables, migration tests with seeded rows, CI |
| `references/custom-rules.md` | Writing a custom lint rule in the HCL rule language: what Snowflake objects expose, `statement` rules for what they do not, examples |

## Requires

The `atlas` skill, installed next to this one (`../atlas/SKILL.md`). If it is missing, download
https://atlasgo.io/skills/atlas/SKILL.md to `../atlas/SKILL.md` and each reference file listed in its
Files section to `../atlas/references/<name>`.

## Rules

1. Check the login first with `atlas whoami`. If it shows no login, ask the user to run `atlas login`:
   logged out, every command that opens a `snowflake://` URL fails with
   `login is required to use the 'snowflake' driver`. A user without an account can start a free trial
   with `atlas login`. The driver ships only in the extended build; the default binary fails with
   `sql/sqlclient: unknown driver "snowflake"` (`references/dev-database.md`).
2. Pick the URL scope from what the schema holds:
   - Schema scope, `snowflake://user:pass@account/DB/SCHEMA?warehouse=WH`: Atlas manages that schema
     and writes unqualified statements (`"users"`). Database scope,
     `snowflake://user:pass@account/DB?warehouse=WH`: Atlas covers every schema in the database and
     qualifies each statement (`"PUBLIC"."users"`).
   - Roles, database roles, users, grants, warehouses, resource monitors, external volumes, and
     external access integrations are read only at database scope. A schema-scoped URL leaves them out
     of inspection and plans without an error.
   - At database scope, a schema the desired state does not declare is planned for a drop, `PUBLIC`
     included: declare `schema "PUBLIC" {}`, or list the managed ones with `schemas`.
   - The dev URL takes the target's scope.
3. The dev database is a separate, empty database you create on the same account, dedicated to Atlas.
   At database scope it may hold only an empty `PUBLIC` schema. Never point `dev` at a real
   environment's database. Account-level objects have no per-database copy, so planning them runs
   against the shared account (`references/dev-database.md`).
4. Mind identifier case. HCL labels are exact, quoted names: `table "users"` creates `"users"`, not
   `USERS`. Unquoted names in SQL files, in `as` bodies, and in the URL's schema fold to uppercase.
   `include` and `exclude` patterns are case-sensitive. Pick one convention, usually uppercase, and keep
   it in HCL, SQL, and patterns.
5. With `roles = true` or an account-level mode on (`warehouses`, `resource_monitors`,
   `external_volumes`, `external_access_integrations`), Atlas plans to drop every such object on the
   account that the desired state does not declare. Scope each env with `include` patterns, such as
   `"an_*[type=role]"` and `"ETL_WH[type=warehouse]"`. Atlas skips Snowflake's system roles and the
   role it connects as.
6. Snowflake commits every DDL statement on its own, so a file that fails halfway stays half applied,
   and `--tx-mode` cannot prevent it. Atlas also takes no lock on Snowflake: two `migrate apply` runs
   against one database are not serialized, so serialize deployments in CI. Keep files small and test
   them first (`references/migrations.md`).
7. Changes Snowflake cannot make in place need care (`references/objects.md`):
   - Column type changes across types, shrinking a `VARCHAR`, changing a `NUMBER` scale or a
     timestamp kind, setting or changing a default (except replacing one sequence default with another),
     and generated expressions need the column dropped and re-added, which loses its data. In a
     terminal, Atlas asks first. When the `CI` variable is set or stdout is not a TTY, the diff fails.
     The `modify_column { allow_recreate = true }` diff policy plans the drop and add without asking:
     for data you keep, write the migration by hand.
   - Atlas plans a drop and a create for changes to a dynamic table's query or columns, an external
     table, non-alterable Iceberg settings, function and procedure bodies, pipe definitions, a
     sequence's `start` (or `order` turned on), and stage credentials. A renamed table, or a renamed
     column in CI, is a drop and an add. Review every `DROP` before applying.
8. Set `warehouse` on every `dynamic_table`. Without it, Atlas writes the warehouse of the connection
   that plans the change, which for `migrate diff` is the dev URL's.
9. Manage roles and grants in a declarative env of their own, never in a migration directory:
   replaying the directory on the dev database runs `CREATE ROLE` on the shared account and fails with
   `Object '...' already exists`. Atlas never creates or drops users; a `user` block manages role
   grants only (`references/security.md`).
10. Secrets are written only by `atlas schema apply`, with the value from an input variable.
    `migrate diff` refuses them with
    `the SECRET_STRING of secret "..." is written only by an in-place apply` (`references/objects.md`).
11. Test every change before reporting it done, as a habit. Run `atlas schema validate`, add a
    `test "schema"` case for each function, procedure, view, or dynamic table you add or change, and a
    `test "migrate"` case with seeded rows for each migration that moves or converts data. Run them
    with `atlas schema test` and `atlas migrate test` (`references/testing.md`).
12. When no built-in analyzer covers a check, write a custom lint rule in the HCL rule language, and do
    the same for a convention the user states. Snowflake-only attributes, such as `transient` or tags,
    are not exposed to rules; check those with `statement` rules on the generated SQL
    (`references/custom-rules.md`).

## Quick Answers

| Question | Answer |
|----------|--------|
| `login is required to use the 'snowflake' driver` | Run `atlas login` (Rule 1) |
| `sql/sqlclient: unknown driver "snowflake"`, or `The snowflake driver is available only as custom build.` | Install the extended build: `curl -sSf https://atlasgo.sh \| ATLAS_FLAVOR="snowflake" sh` (`references/dev-database.md`, Install and Log In) |
| Which dev URL to use | A second, empty database on the account, with the target's scope: `.../DEV_DB/PUBLIC` or `.../DEV_DB` (`references/dev-database.md`, The Dev Database) |
| `connected database is not clean: found schema "..."` | The database-scoped dev database holds a schema other than `PUBLIC`, often left by an interrupted run: drop it (`references/dev-database.md`) |
| `snowflake: schema "..." was not found` | The schema in the URL does not exist, or its case differs: unquoted names fold to uppercase (`references/dev-database.md`) |
| Roles, grants, or warehouses missing from inspection or plans | The URL is schema-scoped, or the `mode` switch is off (`references/objects.md`, What Atlas Reads, and the Switches) |
| A plan drops roles, warehouses, or integrations nobody declared | Scope the env with `include` patterns (`references/security.md`, Turn It On) |
| A plan drops `PUBLIC` or `atlas_schema_revisions` | Database scope covers every schema: declare `PUBLIC`, and list the managed schemas with `schemas` (`references/migrations.md`, Revisions Table) |
| `modify schema "..." is not allowed when migration plan is scoped to one schema` | A schema attribute (comment, retention, tags) changed under a schema-scoped plan: use database scope (`references/dev-database.md`) |
| `... requires recreation. Set 'allow_recreate' in the 'modify_column' block of the diff policy to plan it` | The change needs the column dropped and added, losing its data: set `allow_recreate = true` in `modify_column` of `diff "snowflake"`, or migrate the data by hand (`references/objects.md`, Column Changes) |
| A renamed column or table shows up as a drop and an add | Atlas asks about column renames in a terminal only, and never detects table renames: write `RENAME` by hand (`references/objects.md`, Column Changes) |
| `cannot change permanent table to transient after creation`, or `cannot change permanent schema to transient after creation` | Snowflake fixes the kind at creation: create a new object and move the data (`references/objects.md`, Table Kinds) |
| `adding identity to existing column "..." is not supported` | Snowflake cannot add an identity to an existing column: write the change by hand, such as a new table filled from the old one (`references/objects.md`, Column Changes) |
| `cannot alter TYPE of file format "..." after creation` | Create a new file format under another name and point the stages at it (`references/objects.md`) |
| `iceberg table requires NUMBER to specify precision, e.g. NUMBER(38, 0)` | Iceberg columns need an explicit precision: write `NUMBER(38, 0)`, not `NUMBER` (`references/hcl.md`, Column Types) |
| A dynamic table's migration names the dev warehouse | Set `warehouse` on the `dynamic_table` (`references/hcl.md`, Other Table Kinds) |
| `Object '...' already exists` when replaying the migration directory | Account-level `CREATE` statements in migration files: move them to a declarative env (`references/security.md`, Keep Roles out of Migration Directories) |
| `permission.privileges does not support ALL, list the privileges to grant instead` | List each privilege (`references/security.md`) |
| `Specification of grant option is not supported in a managed access schema` | Drop `grantable` for objects in a `managed_access` schema (`references/security.md`, Permissions) |
| `query policies: ... syntax error ... unexpected 'JOIN'` | The account's edition lacks join or row access policies: turn `policies` off (`references/objects.md`) |
| `create resource monitor "...": ... already exists` on every plan | Account-level objects collide on the dev account once they exist (`references/dev-database.md`, Account-Level Objects During Planning) |
| `permission.for reference to undeclared ... must be qualified as ref(...)` | Write `ref("<kind>.<schema>.<name>")`, inside a `composite_schema` (`references/security.md`) |
| `the SECRET_STRING of secret "..." is written only by an in-place apply` | Manage secrets with `atlas schema apply`, not migration files (`references/objects.md`, Secrets) |
| `secret "..." requires its ..., which inspection cannot read back` | The `secret` block has no value: pass it from an input variable (`references/objects.md`, Secrets) |
| `recreating secret "..." to change ... would drop it and lose its ...`, or `... may drop a refresh token that cannot be read back` | Apply the change with `atlas schema apply`, and declare the OAuth refresh token (`references/objects.md`, Secrets) |
| Lint reports `TX101` | The file has several statements Snowflake cannot run in one transaction: split it, or start it with `-- atlas:txmode none` (`references/migrations.md`) |
| Two deployments ran at the same time | Atlas takes no lock on Snowflake: serialize them in CI (`references/migrations.md`, No Apply Lock) |
| Grants on a view vanished after a deploy | A view change is `CREATE OR REPLACE VIEW`, which drops grants Atlas does not manage (`references/migrations.md`, Reviewing Plans) |
| Tasks created by Atlas never run | Atlas does not resume tasks: run `ALTER TASK ... RESUME` after deploying (`references/objects.md`, Not Modeled) |
| `sql/sqlclient: the "snowflake" driver does not support role switching` | A test `script` command uses `as { role = ... }`: use an `as` block with `user` and `password` (`references/testing.md`) |

## Documentation

- Snowflake HCL reference: https://atlasgo.io/hcl/snowflake
- Snowflake guides: https://atlasgo.io/guides/snowflake/automatic-migrations,
  https://atlasgo.io/guides/snowflake/security-declarative
- Snowflake with dbt: https://atlasgo.io/guides/dbt/snowflake
- URLs and scope: https://atlasgo.io/concepts/url
- Dev database: https://atlasgo.io/concepts/dev-database
- Lint checks: https://atlasgo.io/lint/analyzers
- Guides for each database: https://atlasgo.io/databases
