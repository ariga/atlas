# PostgreSQL Dev Database and Connections

The dev database is a temporary database Atlas uses to load the desired state, replay migrations, and
compute plans (https://atlasgo.io/concepts/dev-database). For PostgreSQL it must match the target's
version, extensions, and scope.

## Scope

The URL's `search_path` decides what Atlas covers (https://atlasgo.io/concepts/url#scope):

| URL | Scope | Atlas covers | DDL it writes |
|-----|-------|--------------|---------------|
| `postgres://localhost:5432/database?search_path=public` | Schema | That schema only | Unqualified (`users`) |
| `postgres://localhost:5432/database` | Database | Every schema | Qualified (`public.users`) |

- The dev URL has the same scope as the target. Mixing them leads to wrong plans.
- A schema-scoped Docker dev URL uses `search_path=public`, whatever schema the target names: `public`
  is created automatically in every new database, so it exists in the container, and schema-scoped DDL
  has no qualifiers, so it runs the same in any schema. A dev database you provide can use any empty
  schema.
- Extensions, event triggers, and foreign servers belong to the database, not to a schema, and roles
  and users belong to the cluster. A schema that declares them needs database scope: at schema scope,
  the `CREATE EXTENSION` line is ignored, and the migration uses the extension's objects without ever
  creating it.
- `exclude` and `include` patterns follow the scope. At schema scope, `"*[type=function]"` matches the
  schema's functions. At database scope, a pattern starts with the schema: `"public.*"` or `"*.audit_*"`.
- To manage several schemas, drop `search_path` and list them on the env: `schemas = ["public", "app"]`.
  An HCL schema with several `schema` blocks fails against a schema-scoped dev URL.
- At database scope, the desired state covers every schema in the database. A schema it does not
  declare, `public` included, is planned for `DROP SCHEMA "public" CASCADE`. Declare
  `schema "public" {}`, or use schema-scoped URLs.

## Docker Dev URLs

The `docker://` driver starts a fresh container for each run (https://atlasgo.io/concepts/dev-database).
Use the target's major version:

```bash
# When working on a single database schema, use the auto-created
# "public" schema as the search path.
--dev-url "docker://postgres/15/dev?search_path=public"

# When working on multiple database schemas.
--dev-url "docker://postgres/15/dev"
```

Images with PostGIS or pgvector, and custom images:

```bash
--dev-url "docker://postgis/latest/dev"
--dev-url "docker://pgvector/pg17/dev"

# When working on a single database schema.
docker+postgres://ghcr.io/namespace/image:tag/dev?search_path=public
# For local/official images, leave host empty or use "_".
docker+postgres://_/local/dev?search_path=public

# When working on multiple database schemas.
docker+postgres://org/image/dev
```

These images provide the extension, but the dev database Atlas creates in them starts without it: the
schema or the `baseline` still runs `CREATE EXTENSION`.

`docker://` needs a Docker client and a reachable daemon. Inside the `arigaio/atlas` image, or on a CI
runner without Docker, point `dev` at a separate PostgreSQL container instead
(https://atlasgo.io/faq/docker-in-docker):

```hcl
env "local" {
  url  = getenv("DATABASE_URL")
  src  = "file://schema"
  dev  = "postgres://user:pass@dev-db:5432/dev?sslmode=disable"
}
```

### Named containers with `atlas tool docker`

`atlas tool docker` creates a named container from a `docker://` URL and prints its connection URL. It
also applies the baseline, and cleans the database if it is not empty, so never run it against a
container that holds data to keep (https://atlasgo.io/cli-reference#atlas-tool-docker). The container
runs until `atlas tool docker kill`. Use it for a local database to apply migrations to
(https://atlasgo.io/getting-started#step-4-set-up-your-database):

```bash
# Spin up a local PostgreSQL database (scoped to the public schema)
export DATABASE_URL=$(atlas tool docker --url "docker://postgres/17/myapp?search_path=public" --name my-db)
```

Drop `search_path` for a database-scoped URL. `--env` reads the dev URL of an env in `atlas.hcl`
instead, such as a `docker` block with a `baseline` (The `docker` Block below):

```bash
atlas tool docker --name my-pg --env local
atlas tool docker kill --name my-pg
```

## The `docker` Block

A `docker` block configures the dev container, and its `baseline` creates what the schema depends on
but does not manage: extensions, schemas, provider roles, external functions
(https://atlasgo.io/concepts/dev-database#baseline-schema). The `docker` and `dev` blocks need
`atlas login`. If `atlas whoami` shows no login, ask the user to run `atlas login`; a user without an
account can start a free trial with it.

```hcl
docker "postgres" "dev" {
  image  = "postgres:15"
  schema = "public"
  baseline = <<SQL
   CREATE SCHEMA "auth";
   CREATE EXTENSION IF NOT EXISTS "uuid-ossp" SCHEMA "auth";
   CREATE TABLE "auth"."users" ("id" uuid NOT NULL DEFAULT auth.uuid_generate_v4(), PRIMARY KEY ("id"));
  SQL
}

env "local" {
  src = "file://schema.pg.hcl"
  dev = docker.postgres.dev.url
}
```

- `schema = "public"` makes the dev database schema-scoped. Leave it out for several schemas, or for
  migrations with qualified names.
- A longer script lives in a file: `baseline = file("baseline.sql")`.
- A `build` block builds a custom image, for example with an extension package installed
  (https://atlasgo.io/concepts/dev-database#docker-with-build-configurations).
- `command` passes server settings. For `ERROR: out of shared memory` on large schemas, raise
  `max_locks_per_transaction` above 128, which the official images already set
  (https://atlasgo.io/faq/out-of-shared-memory):

  ```hcl
  docker "postgres" "dev" {
    image = "postgres:16"
    // ... other configuration as needed
    command = [
      "-c", "max_locks_per_transaction=512",
    ]
  }
  ```

- `template = true` is used only by `atlas schema test`, where every test group starts from a fresh copy
  of the database (Template databases in `references/testing.md`). With a schema-scoped URL, such as a
  block that sets `schema`, it is skipped without an error.

## Your Own Dev Database

Without Docker, the `dev` block connects to an existing empty database, applies the `baseline`, and
restores the database to its original state on exit
(https://atlasgo.io/concepts/dev-database#providing-your-own-dev-database):

```hcl
dev "postgres" "rds" {
  url      = var.dev_url
  baseline = file("baseline.sql")
}

env "local" {
  src = "file://schema.pg.hcl"
  dev = dev.postgres.rds.url
}
```

The database must be empty and dedicated to Atlas, on the same engine, version, and extensions as the
target, and on the same scope. The account needs to create and drop objects in that scope. Environments
can share it only if they never run at the same time. Never point `dev` at a real environment's database:
Atlas fails with `connected database is not clean`, or, if the database is empty, uses it as a sandbox.

## Extensions

Managing extensions needs `atlas login`, and a database-scoped URL
(https://atlasgo.io/faq/postgres-extensions).

- Atlas writes the extension version installed in the dev image into the migration, so the target must
  offer that version (https://atlasgo.io/guides/postgres/rds-extensions):

  ```sql
  -- Create extension "aws_commons"
  CREATE EXTENSION "aws_commons" WITH SCHEMA "public" VERSION "1.2";
  ```

- Objects that `CREATE EXTENSION` creates belong to the extension; Atlas never manages them itself.
- To change an extension's version, Atlas loads the current state on the dev database and applies the
  plan there, so the dev image needs install scripts for both versions. Otherwise it fails with an error
  such as:

  ```
  Error: create extension "postgis_topology": pq: extension "postgis_topology" has no installation script nor update path for version "3.4.3"
  ```

  Use an image with the target's versions, or stop tracking versions
  (https://atlasgo.io/faq/extension-version-mismatch):

  ```hcl
  env "production" {
    url = "postgres://user:pass@host:5432/mydb"
    dev = "docker://postgres/16/dev"
    exclude = ["*[type=extension].version"]
  }
  ```

- To manage extensions in their own env, apart from the application schema
  (https://atlasgo.io/faq/manage-extension-only):

  ```hcl
  env "extensions" {
    src     = "file://extensions.pg.hcl"
    url     = getenv("DATABASE_URL")
    include = ["*[type=extension]"]
  }
  ```

  The application envs then create the same extensions in their dev `baseline`.
- PostGIS needs an image that provides it and `CREATE EXTENSION postgis`
  (https://atlasgo.io/faq/geometry-type). `pq: extension "postgis" is not available` means the dev
  image lacks it: use `docker://postgis/latest/dev`. `pq: type "geometry" does not exist` means the dev
  database does not have it installed, which the PostGIS image alone does not fix: declare the
  extension at the start of the schema, or create it in the dev `baseline`. With an ORM, load the
  extension's SQL before the models with `composite_schema`.
- Extensions that exist only on a managed service, such as `aws_s3` on RDS, have no public package. Mock
  them in the dev image with a control file and an SQL script; the RDS guide shows the image
  (https://atlasgo.io/guides/postgres/rds-extensions).

## Managed Providers

Provider admin roles and their grants are covered in `references/security.md`.

- RDS: creating the AWS extensions needs `rds_superuser`, so apply that migration with a privileged
  role. With IAM authentication, Atlas builds the URL from a token
  (https://atlasgo.io/guides/postgres/rds-extensions):

  ```hcl
  env "rds" {
    src = "file://schema.sql"
    url = "postgres://${var.username}:${urlescape(data.aws_rds_token.migrator)}@${var.endpoint}/${var.database}?sslmode=require"
    dev = docker.postgres.dev.url
  }
  ```

- Cloud SQL: the `gcp_cloudsql_token` data source signs in with IAM; use `sslmode=require`
  (https://atlasgo.io/atlas-schema/projects#data-source-gcp_cloudsql_token). Grants to the `postgres`
  admin need a dev database where `postgres` is an ordinary role (`references/security.md`).
- Supabase (https://atlasgo.io/guides/postgres/supabase):
  - Connect to the direct connection, `db.[project-ref].supabase.co:5432`. The session pooler works on
    IPv4-only networks such as CI runners. The transaction pooler on port 6543 does not work: the
    advisory lock that `migrate apply` holds is per session (Connection Poolers in
    `references/migrations.md`).

    ```bash
    export SUPABASE_URL="postgres://postgres:[PASSWORD]@db.[project-ref].supabase.co:5432/postgres?search_path=public&sslmode=require"
    ```

  - Supabase owns `auth`, `storage`, `realtime`, `graphql`, `extensions`, `vault`,
    `supabase_functions`, and `supabase_migrations`; manage only your own schemas.
  - With the stock `postgres` image, the dev `baseline` creates the `anon`, `authenticated`, and
    `service_role` roles and stubs of the `auth` objects the schema references. Keep the migration
    history out of the schema the Data API exposes:

    ```hcl
    docker "postgres" "dev" {
      image    = "postgres:17"
      schema   = "public"
      baseline = file("20260826120000_baseline.sql")
    }

    env "supabase" {
      url = getenv("SUPABASE_URL")
      dev = docker.postgres.dev.url
      schema {
        src = "file://schema.pg.hcl"
      }
      migration {
        dir              = "file://migrations"
        revisions_schema = "atlas"
      }
    }
    ```

  - Extensions such as `pg_graphql`, `pg_net`, or `pg_cron` need the `supabase/postgres` image instead,
    which already has the roles and the `auth` schema: drop them from the `baseline`, or it fails with
    `role "anon" already exists`.
  - With `permissions = true`, declare `USAGE` on `public` for `PUBLIC`, or Atlas plans to revoke it.
    Never grant to `service_role`.
- Neon: give Atlas the direct connection string, the hostname without `-pooler`. The pooled string runs
  through PgBouncer in transaction mode, where the advisory lock does not hold (Connection Poolers in
  `references/migrations.md`).
- Azure Database for PostgreSQL: use `sslmode=require`, allow extensions in the `azure.extensions`
  server parameter, and give the user as `user@server-name` where the server requires it.

## Connection URLs

- Local databases without TLS need `sslmode=disable`; managed services need `sslmode=require`
  (https://atlasgo.io/concepts/url#ssltls-mode).
- Percent-encode special characters in passwords, or build the URL with `urlescape`:
  `urlescape(getenv("DB_PASSWORD"))` (https://atlasgo.io/faq/url-special-chars).
- The user Atlas connects as must be able to read every object Atlas manages: inspecting with a
  low-privilege role loses the columns and views it cannot access.
- On PostgreSQL 15 and later, roles other than the database owner cannot create objects in `public`
  until they are granted `CREATE` on it. Without the grant, Atlas fails with
  `pq: permission denied for schema public`.
- In the URL, an empty host means localhost:
  `postgres://postgres:pass@:5432/demo?search_path=public&sslmode=disable`.
- On Windows PowerShell, `>` writes UTF-16, which Atlas cannot read. Set
  `$PSDefaultParameterValues['Out-File:Encoding'] = 'utf8'` first
  (https://atlasgo.io/faq/schema-encoding).

## Errors

| Error | Cause | Fix |
|-------|-------|-----|
| `connected database is not clean: found schema "auth"` | The dev database is not empty | Use a Docker dev URL, or create the object in the `baseline` |
| `connected database is not clean: found table "atlas_schema_revisions"` | A real database was used as the dev database | Point `dev` at a Docker dev URL |
| `cannot diff a schema with a database connection`, or `cannot diff a database connection with a schema "X"` | The URL and the dev URL have different scopes | Give both the same `search_path`, or remove it from both |
| `pq: no schema has been selected to create in` | Unqualified DDL on a connection with no schema to create it in | Add `search_path=public` to the URL |
| `pq: function uuid_generate_v4() does not exist` | The `uuid-ossp` extension is missing on the dev database | Create it in the schema or the dev `baseline`, or use the built-in `gen_random_uuid()` (PostgreSQL 13 and later) |
| `pq: SSL is not enabled on the server` | The URL asks for TLS, and the server has none | `sslmode=disable` for a local server |
| `invalid port ":..." after host` | A special character in the password breaks the URL | Percent-encode the password, or build the URL with `urlescape` |
| `exec: "docker": executable file not found in $PATH` | No Docker client where Atlas runs | Use a separate dev database container |
| `pq: extension "aws_commons" is not available` | The extension has no package in the dev image | Mock it in a custom image |
| `pq: extension "postgis" is not available` | The dev image has no PostGIS | Use a PostGIS image: `docker://postgis/latest/dev` |
| `pq: type "geometry" does not exist` | PostGIS is not installed in the dev database, even on the PostGIS image | `CREATE EXTENSION postgis` in the schema or the dev `baseline` |
| `ERROR: out of shared memory` | Too many locks for one transaction | Raise `max_locks_per_transaction` above 128 with `command` |
