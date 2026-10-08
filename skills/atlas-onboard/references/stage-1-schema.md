# Stage 1: Schema as Code

Goal: the unit's desired state lives in code and matches the database. This stage is the same for both
workflows. It writes `atlas.hcl` and the desired state, and changes nothing in a database. The steps
follow https://atlasgo.io/guides/evaluation/verify-atlas and
https://atlasgo.io/guides/evaluation/schema-as-code.

Create the branch `atlas-onboard/<unit>-1` (Rule 4) and work in the unit's directory.

## 1.1 Write `atlas.hcl`

```hcl
atlas {
  cloud {
    org = "acme"                                                 # the org atlas whoami printed
  }
}

locals {
  src = "file://schema"                                 # the desired state; an ORM loader in 1.3
  dev = "docker://postgres/15/dev?search_path=public"   # same engine version and scope as the target
  exclude = [
    "atlas_schema_revisions",                           # Atlas's history table, created by migrate apply
    "schema_migrations",                                # the current tool's history table (golang-migrate)
  ]
}

env "local" {
  url     = urlqueryset(getenv("LOCAL_DATABASE_URL"), "search_path", "identity")   # the database from stage 0
  dev     = local.dev
  exclude = local.exclude
  schema {
    src = local.src
  }
}
```

- The `atlas` block pins the org. Every command that reads this file (`--env`) aborts with
  `'atlas login' is required for organization acme as specified in atlas.hcl` unless someone is logged
  in to that org, which protects teammates and later sessions.
- `local` reads the local or development database from stage 0, and developers apply changes to it
  while they work. It derives the unit's scope from the database-scoped URL of the inventory:
  `urlqueryset(..., "search_path", "<schema>")` in PostgreSQL, `urlsetpath(..., "<database>")` in MySQL.
  Database-scoped units use the URL as is.
- `exclude` keeps tables that are not part of the application's schema out of every export, diff, and
  plan, so Atlas never proposes to drop them:
  - `atlas_schema_revisions`: Atlas's own history table, which `atlas migrate apply` creates in the
    database. Without this entry, `atlas schema apply --dry-run` on a database where migrations ran
    plans `drop "atlas_schema_revisions" table`.
  - The current migration tool's history table, while that tool still runs: `schema_migrations` for
    golang-migrate and dbmate, `flyway_schema_history`, `DATABASECHANGELOG` for Liquibase,
    `goose_db_version`, `alembic_version`, `django_migrations`. Leave it out when there is no such tool.
  - The unit's own patterns from stage 0.4, in the form that matches the URL scope (Gotchas in
    `SKILL.md`).

The dev database is a temporary, isolated database that Atlas uses as a sandbox: it loads the desired
state and the migrations there, because every expression, default, and statement must be accepted by a
real database of the same type and version (https://atlasgo.io/concepts/dev-database#introduction). The
`docker://` driver starts it as an ephemeral container and removes it afterward. It must match the
target's engine, version, and scope. For a schema-scoped PostgreSQL unit, its URL uses
`search_path=public` whatever the target schema is called: schema-scoped DDL has no schema qualifiers,
so it runs the same in any schema.

Docker dev URLs from https://atlasgo.io/concepts/dev-database#introduction. The dev database must use the
same engine and version as the target, so replace the version with the target's, such as
`docker://postgres/16/dev` for PostgreSQL 16:

| Engine | Schema scope | Database scope |
|--------|--------------|----------------|
| PostgreSQL | `docker://postgres/15/dev?search_path=public` | `docker://postgres/15/dev` |
| PostgreSQL with PostGIS | `docker://postgis/latest/dev?search_path=public` | `docker://postgis/latest/dev` |
| PostgreSQL with pgvector | `docker://pgvector/pg17/dev?search_path=public` | `docker://pgvector/pg17/dev` |
| PostgreSQL, custom image | `docker+postgres://ghcr.io/namespace/image:tag/dev?search_path=public` | `docker+postgres://ghcr.io/namespace/image:tag/dev` |
| MySQL | `docker://mysql/8/dev` | `docker://mysql/8` |
| MariaDB | `docker://maria/latest/schema` | `docker://maria/latest` |
| SQL Server | `docker://sqlserver/2022-latest/dev?mode=schema` | `docker://sqlserver/2022-latest/dev?mode=database` |
| ClickHouse | `docker://clickhouse/23.11/dev` | `docker://clickhouse/23.11` |
| Oracle | `docker://oracle/free:latest?mode=schema` | `docker://oracle/free:latest?mode=database` |
| CockroachDB | `docker://crdb/v25.1.1/dev?search_path=public` | `docker://crdb/v25.1.1/dev` |
| YugabyteDB | `docker://ysql/latest/dev?search_path=public` | `docker://ysql/latest/dev` |
| Aurora DSQL | `docker://dsql/16/postgres?search_path=public` | `docker://dsql/16` |
| Spanner | | `docker://spanner/latest`, or `docker://spannerpg/latest` for the PostgreSQL dialect |
| SQLite | `sqlite://dev?mode=memory` | |

Redshift, Snowflake, and Databricks do not run in Docker: point `dev` at a separate, empty database (a
catalog on Databricks) on the same service, as the page shows. `docker://` dev URLs need Docker, locally
and on CI runners.

The exported schema can depend on objects the unit does not manage, such as an extension, a function in
another schema, or a provider role. A new dev database does not have them, so Atlas cannot load the
schema there until they exist. Create them in the `baseline` of a `docker` block, which runs when the
container starts, and point the envs' `dev` at it
(https://atlasgo.io/concepts/dev-database#baseline-schema):

```hcl
docker "postgres" "dev" {
  image    = "postgres:15"
  schema   = "public"                  # schema scope; leave it out for a database-scoped unit
  baseline = <<-SQL
    CREATE EXTENSION IF NOT EXISTS pg_trgm;
  SQL
}

env "local" {
  url = urlqueryset(getenv("LOCAL_DATABASE_URL"), "search_path", "identity")
  dev = docker.postgres.dev.url
  # exclude and schema as above
}
```

A longer script can live in a file: `baseline = file("baseline.sql")`. For extensions such as PostGIS or
pgvector, use an image that ships them (`postgis/postgis`, `pgvector/pgvector`). Without Docker, a `dev`
block points at an empty database of your own and takes the same `baseline`
(https://atlasgo.io/concepts/dev-database#providing-your-own-dev-database).

### Remote Databases

When the user asks to read the schema from a remote environment and provides its connection, add an env
named after it (`staging`), and use `--env staging` where this stage says `--env local`. Use the
connection the user gives, and recommend a read-only role. Connecting to production is not recommended.
Stage 4 deploys through the same env, with a deploy role in CI instead of the read-only one.

Managed databases can sign in with a cloud identity instead of a password: Atlas requests a short-lived
token and builds the URL in `atlas.hcl`, so no password is stored
(https://atlasgo.io/atlas-schema/projects#data-sources). Each method needs the database user mapped to
the identity first; the linked sections list the steps.

AWS RDS with IAM authentication
(https://atlasgo.io/atlas-schema/projects#data-source-aws_rds_token):

```hcl
locals {
  db_user     = "atlas_readonly"
  db_endpoint = "app-db.abc123.us-east-1.rds.amazonaws.com:5432"
}

data "aws_rds_token" "staging" {
  region   = "us-east-1"
  endpoint = local.db_endpoint
  username = local.db_user
}

env "staging" {
  url = "postgres://${local.db_user}:${urlescape(data.aws_rds_token.staging)}@${local.db_endpoint}/app?search_path=identity"
  # dev, exclude, and schema as in local
}
```

Azure Database for PostgreSQL or MySQL with Microsoft Entra ID, using the credentials from `az login`,
a managed identity, or a workload identity
(https://atlasgo.io/atlas-schema/projects#data-source-azure_db_token):

```hcl
data "azure_db_token" "staging" {}

env "staging" {
  url = urluserinfo(
    "postgres://app-db.postgres.database.azure.com:5432/app?search_path=identity&sslmode=require",
    "atlas-readonly@contoso.onmicrosoft.com",
    data.azure_db_token.staging,
  )
}
```

Azure SQL (SQL Server) with Microsoft Entra ID: the `fedauth` parameter of the URL picks the sign-in
method (https://atlasgo.io/concepts/url, SQL Server tab):

```
azuresql://app-db.database.windows.net?fedauth=ActiveDirectoryDefault&database=app
```

GCP Cloud SQL with IAM authentication. MySQL needs `allowCleartextPasswords` and `tls` as shown;
PostgreSQL uses `sslmode=require` (https://atlasgo.io/atlas-schema/projects#data-source-gcp_cloudsql_token):

```hcl
data "gcp_cloudsql_token" "staging" {}   # locals as in the AWS example

env "staging" {
  url = "mysql://${local.db_user}:${urlescape(data.gcp_cloudsql_token.staging)}@${local.db_endpoint}/?allowCleartextPasswords=1&tls=skip-verify&parseTime=true"
}
```

A password kept in a secret store, such as AWS Secrets Manager or GCP Secret Manager
(https://atlasgo.io/atlas-schema/projects#data-source-runtimevar):

```hcl
data "runtimevar" "staging_password" {
  url = "awssecretsmanager://staging/atlas-readonly?region=us-east-1"
}

env "staging" {
  url = "postgres://atlas_readonly:${urlescape(data.runtimevar.staging_password)}@staging-db:5432/app?search_path=identity"
}
```

## 1.2 Export the Database to Code

For units without an ORM, the desired state is the database exported to code. Ask which language the
team will edit (decision point): SQL, which most teams already know, or HCL, which has editor support and
no ordering constraints (https://atlasgo.io/guides/evaluation/schema-as-code#what-should-you-choose).
Default to SQL.

```bash
# SQL: one file per object in schema/, with a main.sql entry point that imports every file
atlas schema inspect --env local --format '{{ sql . | split | write "schema" }}'

# HCL: one file; set local.src to "file://schema.hcl"
atlas schema inspect --env local > schema.hcl
```

Full reference: https://atlasgo.io/inspect/database-to-code.

## 1.3 Or Load the ORM Models

For units with an ORM, the desired state is the models. Add the `data "external_schema"` loader from
the ORM's guide (https://atlasgo.io/orms, `atlas/references/schema-sources.md`), and set `local.src` to
its URL. If the loader goes into the application's configuration (Django's `INSTALLED_APPS`), its
package must be in the requirements that production installs.

Objects the ORM cannot express (views, triggers, functions, row-level security) go in a SQL or HCL file
next to the models, combined with them through `composite_schema`
(https://atlasgo.io/guides/evaluation/schema-as-code#must-you-choose-only-one).

## 1.4 Verify Zero Diff

```bash
atlas schema apply --env local --dry-run
```

`--dry-run` only plans: it reads the database and changes nothing. When the code matches the database,
Atlas reports the schema is synced, with no changes to be made. Otherwise it prints the statements that
would make the database match the code, and each statement is a difference.

An export matches by construction; a difference there means the dev database differs from the target
(version, scope, or defaults). ORM models usually differ. Handle each kind of difference once, not each
object:

| Difference | Fix |
|------------|-----|
| Tables the ORM does not manage (unmanaged models, removed apps, other tools' tables) | Add them to `exclude`; never drop them |
| The same change on every table (collation, charset, owner) | The dev database's defaults differ from the target's: fix the dev database |
| Views, functions, triggers, extensions the ORM does not model | Add them to the `composite_schema` file from 1.3 |
| Types, defaults, index names | Ask whether the model or the database is right. Fix the model, or keep the difference: it becomes the first change in stage 2 |

## Check

`atlas schema apply --env local --dry-run` reports no changes, or only the differences the user chose
to keep. Stage 2 continues on this branch and opens the project PR.

Tell the user that the desired state now lives in the repository, and that both workflows start from it.
Link https://atlasgo.io/guides/evaluation/schema-as-code.
