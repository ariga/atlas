# Databricks Dev Database and Connections

The dev database is a temporary database Atlas uses to load the desired state, replay migrations, and
compute plans (https://atlasgo.io/concepts/dev-database). Databricks does not run in Docker, so the dev
database is a catalog you provide, on the same workspace as the target.

## Install and Log In

The Databricks driver ships only in the extended Atlas binary
(https://atlasgo.io/guides/databricks/automatic-migrations):

```bash
curl -sSf https://atlasgo.sh | ATLAS_FLAVOR="databricks" sh
```

The Docker image is `arigaio/atlas:latest-extended`. The standard binary fails on a `databricks://` URL
with:

```
The databricks driver is available only as custom build.
```

The driver also needs a login. Logged out, every command that opens a Databricks URL fails with
`login is required to use the 'databricks' driver`. Run `atlas whoami`; if it shows no login, ask the
user to run `atlas login` (`atlas login --token` in CI). A user without an account can start a free
trial with `atlas login`. Because the whole driver is gated, inspection never skips objects silently.

## Connection URLs

```
databricks://<token>@<host>:443<warehouse-path>?catalog=<catalog>&schema=<schema>
```

| Part | Value |
|------|-------|
| `<token>` | A Databricks personal access token or a service principal's token |
| `<host>` | The workspace host, such as `dbc-xxxxxxxx-xxxx.cloud.databricks.com` |
| `<warehouse-path>` | The SQL warehouse's HTTP path, such as `/sql/1.0/warehouses/<id>` |
| `catalog` | The Unity Catalog catalog. Without it, the session uses the workspace's default catalog |
| `schema` | Optional. Sets schema scope (below) |

The host and HTTP path are on the warehouse's Connection details tab. Keep the token out of the
command line: build the URL in `atlas.hcl` from environment variables, and run commands with `--env`:

```hcl
locals {
  dbx = "databricks://${getenv("DATABRICKS_TOKEN")}@${getenv("DATABRICKS_HOST")}:443${getenv("DATABRICKS_WAREHOUSE")}"
}

env "prod" {
  url = "${local.dbx}?catalog=prod"
  dev = "${local.dbx}?catalog=atlas_dev"
  migration {
    dir     = "file://migrations"
    tx_mode = "none"
  }
  schema {
    src = "file://schema.dbx.hcl"
    mode {
      permissions = true
    }
  }
}
```

`tx_mode = "none"` is required on Databricks (Rule 4 in `SKILL.md`), and `permissions = true` plans
the `permission` blocks (`references/security.md`).

- The token goes in the user part of the URL, and the warehouse path follows the port directly:
  `@<host>:443/sql/1.0/warehouses/<id>`.
- Store the token as a CI secret, and pass it with `getenv`. Never write it into `atlas.hcl` or a
  command line.
- A token is tied to a principal. Atlas sees and changes only what that principal can read and alter,
  and objects it cannot see are missing from inspection. Use a principal that can read every object
  Atlas manages.
- Atlas reads every object through `information_schema` and the `system.information_schema` routine
  views, which Databricks provides for Unity Catalog catalogs. Point the URL at a Unity Catalog
  catalog.

## Scope

The `schema` parameter decides what Atlas covers (https://atlasgo.io/concepts/url#scope):

| URL | Scope | Atlas covers | DDL it writes |
|-----|-------|--------------|---------------|
| `databricks://...?catalog=main&schema=sales` | Schema | That schema only | Unqualified (`` `orders` ``) |
| `databricks://...?catalog=main` | Catalog | Every schema in the catalog | Qualified by schema (`` `sales`.`orders` ``) |

- The dev URL has the same scope as the target. Mixing them leads to wrong plans.
- No statement Atlas writes names a catalog, and inspection keeps only the schema of a qualified name.
  The same migration directory therefore applies to `dev`, `staging`, and `prod` catalogs; the URL
  picks the catalog.
- At catalog scope the desired state covers every schema in the catalog. A schema it does not declare
  is planned for `DROP SCHEMA ... CASCADE`, which drops everything in it. Declare `schema "default" {}`
  and every other schema the catalog keeps, list the managed ones with `schemas = [...]` on the env, or
  use schema scope. `information_schema` is never inspected.
- `exclude` and `include` patterns follow the scope. At schema scope, `"tmp_*"` matches tables. At
  catalog scope a pattern starts with the schema: `"default.tmp_*"`.
- An HCL schema with several `schema` blocks needs catalog scope on both URLs.

## The Dev Catalog

Create a catalog dedicated to Atlas, on the same workspace and warehouse as the target:

```sql
CREATE CATALOG IF NOT EXISTS atlas_dev;
```

A new catalog comes with a `default` schema. Atlas uses it as follows:

- Catalog scope, `dev = "${local.dbx}?catalog=atlas_dev"`: every schema in the catalog must be empty,
  and the `default` schema must exist. Atlas creates the schemas the desired state declares, and drops
  them again when it is done.
- Schema scope, `dev = "${local.dbx}?catalog=atlas_dev&schema=default"`: the named schema must hold no
  table, view, function, or procedure. Any empty schema works, even one in another catalog.
- The account in the dev URL must be able to create and drop schemas, tables, views, materialized
  views, and functions in that catalog.
- Atlas cleans up after itself, so several environments can share the dev catalog as long as they
  never run at the same time. Atlas takes no lock on Databricks, so nothing stops two runs from
  colliding there (`references/migrations.md`, Locks).
- Never point `dev` at a catalog that holds data. Atlas fails with `connected database is not clean`,
  or, if the catalog is empty, uses it as a sandbox and drops what it created.
- Each plan creates the desired objects on the dev catalog through the warehouse, so a plan takes
  seconds, not milliseconds. A materialized view in the desired state is created there on every plan.

### What normalization keeps

Atlas loads the desired state on the dev catalog and inspects it back, so the plan compares objects as
Databricks stores them. Databricks adds things of its own there, and Atlas removes them again:

- Table properties the desired state does not list, such as the `delta.*` defaults.
- A `location`: Atlas never creates a dev table at the target's path. It drops the clause before loading
  the desired state, and puts the declared one back after.
- The column list Databricks reports for a view that declares none, the schema's collation when the
  definition names none, and the determinism and SQL data access it derives for a routine that
  declares neither.

A SQL schema source (`file://schema.sql`) is different: Atlas runs it on the dev catalog and takes the
result as the desired state, so every property Databricks assigned there appears in the plan's
`CREATE TABLE`. Prefer an HCL schema, or list only the properties you own after inspecting.

### No `docker` or `dev` block

Atlas has no `docker` block and no `dev` block for Databricks, so there is no `baseline` to run on the
dev catalog first: `dev` is the URL and nothing else. An object the schema references but does not
manage, such as a function in another catalog, must already exist where the SQL names it, and the dev
catalog itself must stay empty.

## Errors

| Error | Cause | Fix |
|-------|-------|-----|
| `The databricks driver is available only as custom build.` | The standard binary | Install the extended binary with `ATLAS_FLAVOR="databricks"` |
| `login is required to use the 'databricks' driver` | No login | `atlas login`, or `atlas login --token` in CI |
| `connected database is not clean: found table "..." in schema "..."` | The dev catalog holds a table, often one left by an interrupted run, or `dev` points at a real catalog | Drop the leftover objects, or recreate the dedicated dev catalog |
| `connected database is not clean: found view "..." in schema "..."`, or `found function`, `found procedure` | The same, for a view or a routine | The same |
| `connected database is not clean: found table "..." in connected schema` | A schema-scoped dev URL names a schema that is not empty | Point `schema` at an empty schema |
| `connected database is not clean: default schema "default" not found` | The catalog-scoped dev catalog has no `default` schema | `CREATE SCHEMA atlas_dev.default;` |
| `connected database is not clean: found schema "..."` | `migrate apply` on a target that has schemas but no revisions table | Baseline the target (`references/migrations.md`, Existing Databases) |
| `databricks: schema "..." was not found` | A schema-scoped URL names a schema that does not exist | Create the schema, or fix the parameter |
