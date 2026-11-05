---
name: atlas-lakebase
description: "Databricks Lakebase specifics for Atlas: the databricks_lakebase_token data source and the Databricks credentials it uses (a CLI profile, or a service principal with the workspace-access entitlement), the endpoint host instead of the -pooler host, the Postgres role in the URL, password roles, the objects Lakebase manages in every database (__db_system, three grant_* functions, three on_create_* event triggers) and how to exclude them at schema and database scope, the first migrate apply, drift checks, object ownership, and Lakebase errors. Use with the atlas and atlas-postgres skills whenever Atlas manages a Databricks Lakebase database."
---

# Atlas for Databricks Lakebase

What is specific to Databricks Lakebase when Atlas manages the schema. Lakebase is managed PostgreSQL,
and Atlas connects to it with the `postgres` driver: the `atlas-postgres` skill covers the schema work
(URL scope, the dev database, roles and grants, objects, lint, and tests), and the `atlas` skill covers
the workflows. This skill covers connecting to Lakebase and the objects Lakebase manages. Guide:
https://atlasgo.io/guides/postgres/databricks-lakebase.

## Requires

The `atlas` and `atlas-postgres` skills, installed next to this one. Read `../atlas-postgres/SKILL.md`
and `../atlas/SKILL.md` before you start, and follow them for everything this skill does not cover:
this skill only adds what differs on Lakebase. If one is missing, download
https://atlasgo.io/skills/<skill>/SKILL.md to `../<skill>/SKILL.md`, and each reference file listed in
its Files section to `../<skill>/references/<name>`.

## Rules

1. Check the login first with `atlas whoami`. If it shows no login, ask the user to run `atlas login`:
   the `databricks_lakebase_token` data source needs it. A user without an account can start a free
   trial with `atlas login`. An Atlas release older than the data source fails every env in the file
   with `missing data source handler for "databricks_lakebase_token"`: run `atlas version`, and install
   the latest release.
2. Connect with a token from the data source. `endpoint` is the endpoint path, and the URL user is the
   Postgres role of the identity the token is issued to: the email of a Databricks user, or the
   application ID of a service principal. `urluserinfo` sets both and escapes the `@`:

   ```hcl
   locals {
     host = "ep-example.database.us-west-2.cloud.databricks.com"
     user = "user@example.com"
   }

   data "databricks_lakebase_token" "db" {
     endpoint = "projects/my-app/branches/production/endpoints/primary"
   }

   env "lakebase" {
     url = urluserinfo(
       "postgres://${local.host}:5432/databricks_postgres?sslmode=require&search_path=public",
       local.user,
       data.databricks_lakebase_token.db,
     )
     dev = "docker://postgres/17/dev?search_path=public"
     // Functions that Lakebase manages in "public" (Rule 5).
     exclude = [
       "grant_usage_on_new_schema[type=function]",
       "grant_all_on_new_sequences[type=function]",
       "grant_select_on_new_objects[type=function]",
     ]
     schema {
       src = "file://schema.sql"
     }
   }
   ```

   - The host is the endpoint's host, never the `-pooler` host, which fails with
     `OAuth authentication is not supported with pooled connections`. This prints each endpoint path
     on a branch, followed by its host:

     ```bash
     databricks postgres list-endpoints projects/<project>/branches/<branch> -o json \
       | jq -r '.[] | "\(.name) \(.status.hosts.host)"'
     ```

   - `databricks_postgres` is the database Lakebase creates on every branch.
   - A token expires one hour after it is issued. Atlas generates a new one each time it loads
     `atlas.hcl`, so each command starts with a valid token.
3. The data source finds Databricks credentials with unified authentication; the URL user only names
   the Postgres role:
   - On a developer machine: `databricks auth login --host <workspace-url> --profile lakebase`, then
     `profile = "lakebase"` in the data source, or `DATABRICKS_CONFIG_PROFILE=lakebase`. That profile
     gets its token from the Databricks CLI, so `databricks` must be on the `PATH` wherever Atlas runs.
   - In CI: a service principal, read from `DATABRICKS_HOST`, `DATABRICKS_CLIENT_ID`, and
     `DATABRICKS_CLIENT_SECRET` (from the CI secret store). It needs an OAuth secret, the
     `workspace-access` entitlement, and an OAuth role on the branch; the URL user is its application
     ID. Give the secret only to the jobs that apply migrations, from a scope that only the default
     branch can read, such as a protected environment, and grant the role only the privileges those
     migrations need.
   - A native password role works without Databricks credentials. Lakebase disables password
     connections in new projects: enable them, create the role, and pass `getenv("LAKEBASE_USER")` and
     `getenv("LAKEBASE_PASSWORD")` to `urluserinfo`. Prefer tokens where the identity can use them.
4. The dev database runs the project's Postgres version, which this prints:
   `databricks postgres get-project projects/<project> -o json | jq .status.pg_version`. Use
   `docker://postgres/<version>/dev`, with the target's scope (`atlas-postgres`, Rule 2).
5. Every Lakebase database has a `__db_system` schema, three functions in `public`
   (`grant_usage_on_new_schema`, `grant_all_on_new_sequences`, `grant_select_on_new_objects`), and three
   event triggers that call them (`on_create_schema`, `on_create_sequence`, `on_create_table_or_view`).
   The env excludes them, or Atlas plans to drop them:
   - At schema scope (`search_path=public`), exclude the functions by exact name, as in Rule 2.
     Without it, `schema apply` plans to drop them and fails with `2BP01`. Never use a `grant_*`
     pattern: it also hides the user's own functions from every plan, without a warning.
   - At database scope (several schemas, no `search_path` in either URL), qualify the functions with
     `public.`, and exclude `__db_system` and the event triggers by name. Without the event triggers,
     planning fails with `function public.grant_usage_on_new_schema() does not exist (42883)`:

     ```hcl
     exclude = [
       "__db_system",
       "on_create_schema",
       "on_create_sequence",
       "on_create_table_or_view",
       "public.grant_usage_on_new_schema[type=function]",
       "public.grant_all_on_new_sequences[type=function]",
       "public.grant_select_on_new_objects[type=function]",
     ]
     ```

   - The functions of the `databricks_auth` extension belong to the extension and need no exclude.
6. The versioned workflow runs only the migration files, so `migrate apply` does not need the exclude.
   Keep it in the env anyway: drift checks read it, and report the objects Lakebase manages as drift
   without it. A `drift.exclude` list in `check "migrate_apply"`, and `migration.exclude` or
   `--exclude` for `migrate drift`, replace `exclude` instead of adding to it: repeat the Rule 5
   patterns there. At database scope, the first `migrate apply` on a new database fails with
   `connected database is not clean: found schema "__db_system"`: run that first apply with
   `--allow-dirty` (https://atlasgo.io/versioned/apply#allow-dirty).
7. The role that runs the migrations owns the objects they create, and only the owner or a member of
   its role can alter them. Keep deploying as the same role, or grant that role to the identity that
   replaces it.
8. Schema changes, lint, and tests follow `atlas-postgres` and `atlas`: validate the schema, test the
   change with `atlas schema test` and `atlas migrate test`, and lint it before reporting it done.

## Quick Answers

| Question | Answer |
|----------|--------|
| `missing data source handler for "databricks_lakebase_token"` | The installed Atlas predates the data source: install the latest release (Rule 1) |
| `default auth: cannot configure default credentials` | No Databricks credentials: `databricks auth login` with `profile`, or the service principal variables (Rule 3) |
| `default auth: databricks CLI not found` | The profile gets its token from the Databricks CLI: install it where Atlas runs (Rule 3) |
| `Endpoint name expects 'projects/{project_id}/branches/{branch_id}/endpoints/{endpoint_id}' format` | `endpoint` must be the full endpoint path (Rule 2) |
| `OAuth authentication is not supported with pooled connections` | Use the endpoint host, not the `-pooler` host (Rule 2) |
| `This API is disabled for users without the workspace-access entitlement` | Turn on Workspace access for the service principal (Rule 3) |
| A valid token fails to log in | The URL user must be the Postgres role of the identity the token is issued to (Rule 2) |
| `cannot drop function grant_... because other objects depend on it (2BP01)` | Exclude the three functions by exact name (Rule 5) |
| `create event trigger "on_create_schema": pq: function public.grant_usage_on_new_schema() does not exist (42883)` | At database scope, also exclude the event triggers and `__db_system` (Rule 5) |
| `connected database is not clean: found schema "__db_system"` | The first `migrate apply` on a new database at database scope: run it once with `--allow-dirty` (Rule 6). An existing database takes `baseline` instead (https://atlasgo.io/versioned/apply#existing-databases) |
| Drift reports the `grant_*` functions, the `on_create_*` event triggers, or `__db_system` | Keep the Rule 5 excludes in the env the drift check reads, and repeat them in its `drift.exclude` or `migration.exclude` list if it sets one: those replace `exclude` (Rule 6) |
| `must be owner of table ...` | Another role created the table: deploy as that role, or grant it to this one (Rule 7) |

## Documentation

- Databricks Lakebase guide: https://atlasgo.io/guides/postgres/databricks-lakebase
- The data source: https://atlasgo.io/atlas-schema/projects#data-source-databricks_lakebase_token
- Excluding objects from drift checks: https://atlasgo.io/versioned/drift-detection#excluding-objects
- Lakebase Postgres roles: https://docs.databricks.com/aws/en/oltp/projects/postgres-roles
