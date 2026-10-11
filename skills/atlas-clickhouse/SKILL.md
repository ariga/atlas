---
name: atlas-clickhouse
description: "ClickHouse specifics for Atlas: the login-only driver, dev database and URL scope (database vs server), Docker dev images and version checks, cluster mode and ON CLUSTER, table engines, sort keys and table rebuilds, plans that skip destructive changes without a terminal, mutations and CH lint checks, materialized views, dictionaries, UDFs and WASM, named collections, users, roles, grants, row policies and settings profiles, secret settings and the sensitive mode, no transactions and no lock, ClickHouse Cloud, schema and migration tests, and custom lint rules. Use with the atlas skill whenever Atlas manages a ClickHouse database."
---

# Atlas for ClickHouse

What is specific to ClickHouse when Atlas manages the schema. The `atlas` skill covers the workflows and
commands; this skill covers the rules, edge cases, and answers that differ on ClickHouse. Clusters and
ClickHouse Cloud are covered where they differ.

## Files

This skill is `SKILL.md` plus seven reference files under `references/`, each served at
`https://atlasgo.io/skills/atlas-clickhouse/references/<name>`. Read a reference when the task needs it:

| File | Read it for |
|------|-------------|
| `references/dev-database.md` | URL scope, URL parameters, Docker dev URLs, the `docker` and `dev` blocks, version checks, cluster and Cloud dev databases, connection errors |
| `references/security.md` | Users, roles, grants, row policies, settings profiles, access storages, passwords, secret settings and the sensitive mode |
| `references/objects.md` | What each object needs (scope, `mode` flags), engines, column conversions, materialized views, dictionaries, UDFs, WASM, named collections, what Atlas skips or does not plan |
| `references/hcl.md` | Writing the schema in HCL: `Nullable`, types, defaults and `default_kind`, codecs, TTL, `sort`, `partition`, `primary_key`, indexes, projections, views, seed data |
| `references/migrations.md` | `CH3*` lint findings, table rebuilds, skipped changes, mutations, diff policies, no transactions, no lock, cluster mode DDL, the revisions table |
| `references/testing.md` | Testing every change: schema tests, migration tests with seeded rows, mutations and merges in tests |
| `references/custom-rules.md` | Writing a custom lint rule in the HCL rule language when no analyzer covers a check: what each ClickHouse object exposes, examples |

## Requires

The `atlas` skill, installed next to this one (`../atlas/SKILL.md`). If it is missing, download
https://atlasgo.io/skills/atlas/SKILL.md to `../atlas/SKILL.md` and each reference file listed in its
Files section to `../atlas/references/<name>`.

## Rules

1. Check the login first with `atlas whoami`. If it shows no login, ask the user to run `atlas login`:
   logged out, the ClickHouse driver does not work, and every command that connects fails with
   `login is required to use the 'clickhouse' driver`. A user without an account can start a free
   trial with `atlas login`.
2. Pick the URL scope from what the schema holds:
   - Database scope, a URL with a database in its path (`clickhouse://user:pass@host:9000/app`): Atlas
     manages that database's tables, views, materialized views, and dictionaries, and writes unqualified
     statements. Server scope, no database in the path: Atlas covers every database except the system
     ones and qualifies each statement (`app.users`).
   - Users, roles, row policies, settings profiles, named collections, UDFs, and WASM modules belong to
     the server: they need server scope (`references/objects.md`).
   - At server scope, a database the desired state does not declare is planned for `DROP DATABASE`,
     `default` included. Declare `schema "default" {}`, or limit the run with `--schema`. Queries in
     views, materialized views, and dictionaries name each table with its database (`FROM app.events`).
   - The dev URL takes the target's scope: `docker://clickhouse/24.10/dev`, or
     `docker://clickhouse/24.10`.
3. The dev database runs the target's major version, or commands fail with
   `target connection is running version 25.x but dev database is running version 24.x`. Give it the
   settings the schema relies on, as URL parameters on both URLs. A clustered target needs a clustered
   dev database, which `docker://` cannot start (`references/dev-database.md`, Clusters). Never point
   `dev` at a database that holds data: give Atlas an empty database of its own.
4. On a cluster, add `mode=cluster` to both URLs: Atlas then writes ``ON CLUSTER `{cluster}```, `SYNC`
   drops, and `SETTINGS alter_sync = 2`, and keeps the revisions table in a `ReplicatedMergeTree`
   (`references/migrations.md`, Cluster Mode).
5. Without a terminal, Atlas leaves out of the plan, without an error, the changes it can apply only by
   dropping data: a column change ClickHouse cannot make in place, any change to a `TimeSeries` table,
   and a sort key, partition, primary key, or engine change on a `Replicated*` table whose engine holds
   an explicit ZooKeeper path. This is the case in CI and in most agent shells. After every diff, check
   that each edit appears in the plan; when one is missing, write the migration by hand or ask the user
   to run the command in a terminal, where Atlas asks before dropping (`references/migrations.md`).
6. Review every `CH3*` lint finding before applying to a large table: a sort key, partition, primary
   key, or engine change rebuilds the table by copying its rows into `<table>_tmp` and swapping it with
   `EXCHANGE TABLES`, and rows written after the copy are lost (`CH302`). A column type change rewrites
   every part (`CH301`), and a new or changed index or projection is materialized over existing data
   (`CH303`). The checks are warnings: lint passes with them unless the `lint` block makes them errors.
7. ClickHouse has no transactional DDL, and Atlas takes no lock on it. Each statement commits on its own,
   so a file that fails halfway stays half applied, and two deployments to the same target can run at
   once. Keep files small, test them (Rule 10), and run one deployment per target at a time.
8. Never write secrets into the schema: pass them through input variables. Without `sensitive = ALLOW`
   in the env's `schema.mode`, Atlas never writes a secret setting (such as `kafka_sasl_password`), a
   user password, or a named collection value: a new user is created with no password, and anyone can
   log in as it. Migration files never hold secret settings or passwords;
   a credential inside an engine's arguments or a dictionary source is written, under
   `-- atlas:sensitive` (`references/security.md`, Secrets).
9. Declare a nullable column with its wrapper, `type = sql("Nullable(Date)")`. `null = true` on its own
   fails with `nullable column must have type wrapped with "Nullable()"` (`references/hcl.md`).
10. Test every change before reporting it done, as a habit. Run `atlas schema validate`, add a
    `test "schema"` case for each materialized view, dictionary, UDF, check, or row policy you add or
    change, and a `test "migrate"` case with seeded rows for each migration that rebuilds a table or
    converts a column. Run them with `atlas schema test` and `atlas migrate test`
    (`references/testing.md`).
11. When no built-in analyzer covers a check, write a custom lint rule in the HCL rule language, and do
    the same for a convention the user states, such as replicated engines, a TTL on every table, or a
    codec on every column. Lint enforces it on every change (`references/custom-rules.md`).

## Quick Answers

| Question | Answer |
|----------|--------|
| A local ClickHouse database to apply migrations to | `atlas tool docker --url "docker://clickhouse/24.10/myapp" --name my-db` prints its URL; `atlas tool docker kill --name my-db` stops it (`references/dev-database.md`) |
| `login is required to use the 'clickhouse' driver` | Run `atlas login` (Rule 1) |
| `failed to check compatibility with dev database: target connection is running version 24.x but dev database is running version 23.x` | Run the dev image on the target's major version (`references/dev-database.md`, Versions) |
| `dev database must also be in a cluster configuration` | The target has a cluster; use a dev database on a cluster (`references/dev-database.md`, Clusters) |
| `target and dev databases disagree on has_user_defined_zookeeper_path, set it on both URLs` | Set the parameter on both URLs (`references/dev-database.md`, Connection URLs) |
| `clickhouse: unknown mode "..."` | The only `mode` URL value is `cluster` (`references/dev-database.md`, Connection URLs) |
| `connected database is not clean: found ...` | `dev` points at a database that holds objects (`references/dev-database.md`, Your Own Dev Database) |
| `dev database is not clean (found ...). Add a schema to the URL to limit the scope of the connection` | A server-scoped dev URL with a database-scoped desired state: give the dev URL the same scope (`references/dev-database.md`) |
| `code: 81, message: Database <name> does not exist` | The URL's path names a database that does not exist: create it, or use server scope (`references/dev-database.md`) |
| A plan drops the `default` database | Server scope covers every database: declare `schema "default" {}` (`references/dev-database.md`, Scope) |
| A view or dictionary fails at server scope because a table does not exist in `default` | Name the table with its database in the query, `FROM app.events` (`references/dev-database.md`, Scope) |
| A column type or sort key change is missing from the plan, or `no changes to be made` | Atlas skipped a destructive rebuild without a terminal (`references/migrations.md`, Changes Skipped Without a Terminal) |
| `nullable column must have type wrapped with "Nullable()"` | Write `type = sql("Nullable(T)")`; on `LowCardinality(Nullable(T))`, leave `null` unset (`references/hcl.md`) |
| `formatting type for column "...": unsupported type` | Atlas does not know the type, often a SQL alias such as `VARCHAR`: write the ClickHouse type (`references/hcl.md`, Columns and Types) |
| `converting index "...": unsupported index type` | Use a supported skip index type (`references/hcl.md`, Tables) |
| Lint reports `CH301`, `CH302`, or `CH303` | A mutation, a table copy, or an index or projection materialized over existing parts (`references/migrations.md`) |
| A migration failed halfway | ClickHouse kept the statements before the failure: fix the rest and run `atlas migrate hash`, or undo them with `atlas migrate down` (`references/migrations.md`, Transactions) |
| A materialized view lost its rows after a change | Any change but its refresh drops and recreates it; keep rows in a `to` table (`references/objects.md`, Materialized Views) |
| `... must be recreated for this change, which drops its secrets. Enable the sensitive mode to plan it:` | The recreate would lose a secret setting, declared or only on the server: plan it with `sensitive = ALLOW` (`references/security.md`, Secrets) |
| `..., as a deferred plan does not hold them. Recreate it manually to apply this change` | A migration file cannot recreate an object that holds secrets: do it by hand (`references/security.md`, Secrets) |
| Sensitive mode reports the same dictionary, table, or named collection change on every run | The server reads secrets back masked, so they never match (`references/security.md`, Secrets) |
| `modify schema "..." is not allowed when migration plan is scoped to one schema` | A database `settings` change in a migration file needs server scope (`references/dev-database.md`, Scope) |
| Undeclared users or roles are planned for a drop | Declare them, or skip drops with `drop_user` and `drop_role` (`references/security.md`) |
| A user named `default...` or `root...` never shows up | Inspection skips them, and the connecting user (`references/security.md`) |
| A user's `auth_type` change is missing from the plan | Atlas does not plan it: run `ALTER USER ... IDENTIFIED WITH` by hand (`references/security.md`, Users and Roles) |
| `changing password for user "..." should be handled manually` | Migration files never set passwords: rotate it with `schema apply` and `sensitive = ALLOW`, or by hand (`references/security.md`, Passwords) |
| `unsupported permission.for type "dictionary"` | Grants on dictionaries are not managed: grant on the database, or by hand (`references/security.md`, Permissions) |
| Named collections, row policies, settings profiles, or WASM modules are missing from inspection | Enable them in `mode "clickhouse"` (`references/objects.md`) |
| A declared object is planned for creation on every run | Inspection skips it: a name starting with `_tmp_replace_` or `_peerdb_`, a named collection on Cloud, or a global grant (`references/objects.md`, Not Inspected) |
| A view shows a `security` change on every plan | The dev server's `default_normal_view_sql_security` differs from the target's; declare `security` (`references/objects.md`, Views) |
| `WASM functions are not supported on ClickHouse Cloud` | WASM modules and WASM UDFs are self-hosted only (`references/objects.md`, WASM Modules and UDFs) |
| A Docker dev database rejects `SharedMergeTree` or `Shared` | Write OSS engines; Cloud converts them, and Atlas treats them as equal (`references/dev-database.md`, ClickHouse Cloud) |
| Two deployments ran at the same time | Atlas takes no lock on ClickHouse: serialize deployments (`references/migrations.md`, Locks) |
| `driver: bad connection` from an `exec` in `atlas schema test` or `atlas migrate test` | `exec` can only read on ClickHouse: run `INSERT`, DDL, and `SYSTEM` statements through a `script` command (`references/testing.md`, Seed Rows with a Script) |
| `mismatched schema names: "dev" != "app"` from `atlas schema test` | Name the dev database after the `schema` block, `docker://clickhouse/24.10/app`, or use server scope (`references/testing.md`, Test Every Change) |
| `altering table "...": unsupported change type` | The table has a primary key the schema leaves out: declare its `primary_key` (`references/hcl.md`, Tables) |
| `analyzing plan: executing statement: ALTER TABLE ... MODIFY COLUMN ...: code: 524` | The column is in a key, a skip index, or a projection: rebuild the table, or drop and re-add the index or projection by hand (`references/objects.md`, Columns) |
| `check assertion "SELECT NOT EXISTS (...)" returned false` from `atlas migrate down` | The table or column holds data; a non-`Nullable` column always does. Confirm with the user, then `--skip-checks` (`references/migrations.md`, Transactions) |
| A user Atlas created logs in without a password | Without `sensitive = ALLOW`, passwords are never written (`references/security.md`, Passwords) |

## Documentation

- ClickHouse HCL reference: https://atlasgo.io/hcl/clickhouse
- Getting started: https://atlasgo.io/guides/clickhouse
- Dev database: https://atlasgo.io/concepts/dev-database
- Clusters: https://atlasgo.io/guides/clickhouse/cluster-mode
- ClickHouse Cloud: https://atlasgo.io/guides/clickhouse/clickhouse-cloud-atlas
- Security: https://atlasgo.io/guides/clickhouse/security-declarative,
  https://atlasgo.io/guides/clickhouse/security-versioned
- Lint checks: https://atlasgo.io/lint/analyzers
- Guides for each database: https://atlasgo.io/databases
