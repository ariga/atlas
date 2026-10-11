# SQL Server Dev Database and Connections

The dev database is a temporary database Atlas uses to load the desired state, replay migrations, and
compute plans (https://atlasgo.io/concepts/dev-database). For SQL Server it must match the target's
version, compatibility level, collation, and scope.

## Connection URLs

```
sqlserver://user:pass@host:1433?database=app&mode=schema
```

- The database goes in the `database` query parameter. The URL path is not the database: the driver
  reads it as an instance name, and `sqlserver://user:pass@host:1433/app` silently connects to the
  login's default database, usually `master`. Always write `?database=<name>`.
- Azure SQL Database and Managed Instance with Microsoft Entra ID use the `azuresql://` scheme and a
  `fedauth` method (Azure SQL and Fabric below).
- Percent-encode special characters in passwords, or build the URL with `urlescape`:
  `urlescape(getenv("DB_PASSWORD"))` (https://atlasgo.io/faq/url-special-chars).
- The user Atlas connects as must be able to read every object Atlas manages, and create and alter
  them on the target. `db_owner` on the database covers both.

## Scope

`mode` decides what Atlas covers (https://atlasgo.io/concepts/url):

| URL | Scope | Atlas covers | DDL it writes |
|-----|-------|--------------|---------------|
| `?database=app` or `?database=app&mode=schema` | Schema | The connecting user's default schema (`dbo` for `sa`) | Unqualified (`[users]`) |
| `?database=app&mode=database` | Database | Every schema in that one database, plus roles, users, partition functions and schemes, and full-text catalogs | Qualified (`[dbo].[users]`) |

- `mode=schema` is the default. Any other value fails with
  `sql/mssql: connection mode must be schema or database, received "<value>"`.
- A URL always names one database. Atlas never manages other databases, logins, server roles, or
  database settings (`ALTER DATABASE`).
- Schema scope uses the connecting user's default schema; no URL parameter picks another one. A user
  without a default schema fails with
  `sql/mssql: schema connection mode requires a default schema to be set for the user`. To manage a
  schema other than `dbo`, use `mode=database` and limit it with `schemas = ["sales"]` on the env or
  `--schema sales`, or connect as a user whose `DEFAULT_SCHEMA` is that schema.
- The dev URL takes the target's scope. Mixed scopes fail with
  `cannot diff a schema "dbo" with a database connection` (or the reverse).
- An HCL schema with several `schema` blocks needs a database-scoped dev URL. Against a schema-scoped
  one it fails with `cannot use HCL with more than 1 schema when dev-url is limited to schema "dbo"`.
- At database scope, the desired state covers every schema in the database: a schema it does not
  declare is planned for `DROP TABLE` and `DROP SCHEMA`. Declare it, or limit the run with `schemas` or
  `--schema`. Atlas never creates or drops `dbo`.
- `exclude` and `include` patterns follow the scope: `"*[type=view]"` at schema scope, `"dbo.*"` or
  `"*.audit_*"` at database scope.

## Docker Dev URLs

The `docker://` driver starts a fresh container for each run (https://atlasgo.io/concepts/dev-database).
The tag is a `mcr.microsoft.com/mssql/server` tag; use the target's major version:

```bash
# A schema-scoped dev database ("dbo" of a new database named "dev").
--dev-url "docker://sqlserver/2022-latest/dev?mode=schema"

# A database-scoped dev database: several schemas, roles, users, and permissions.
--dev-url "docker://sqlserver/2022-latest/dev?mode=database"

# Another image.
--dev-url "docker+sqlserver://mcr.microsoft.com/azure-sql-edge:1.0.7?mode=schema"
```

- Name a database in the path (`/dev`): Atlas creates it. Without one, the dev database is `master`.
- The SQL Server images are `linux/amd64`. On an ARM machine, Docker runs them under emulation and
  they start slowly. A `docker` block raises the start-up wait with `timeout` (one minute by default)
  and sets `platform`.
- `docker://` needs a Docker client and a reachable daemon. Inside the `arigaio/atlas` image, or on a CI
  runner without Docker, point `dev` at a separate SQL Server container instead
  (https://atlasgo.io/faq/docker-in-docker):

  ```hcl
  env "ci" {
    url = getenv("DATABASE_URL")
    src = "file://schema.sql"
    dev = getenv("DEV_URL")    # an empty database on a SQL Server service container
  }
  ```

### Named containers with `atlas tool docker`

`atlas tool docker` creates a named container from a `docker://` URL and prints its connection URL. It
cleans the database if it is not empty, so never run it against a container that holds data to keep
(https://atlasgo.io/cli-reference#atlas-tool-docker). The container runs until `atlas tool docker kill`:

```bash
export DATABASE_URL=$(atlas tool docker --url "docker://sqlserver/2022-latest/myapp" --name my-db)
atlas tool docker kill --name my-db
```

`--env <name>` reads the dev URL of an env in `atlas.hcl` instead, such as a `docker` block below.

## The `docker` Block

A `docker` block configures the dev container. The `docker` and `dev` blocks need `atlas login`, which
SQL Server needs anyway:

```hcl
docker "sqlserver" "dev" {
  image    = "mcr.microsoft.com/mssql/server:2022-latest"
  database = "dev"
  collate  = "SQL_Latin1_General_CP1_CS_AS"   # the target's collation
  mode     = "schema"                         # or "database"
  baseline = <<-SQL
    CREATE TABLE ext_lookup (id int NOT NULL CONSTRAINT pk_ext_lookup PRIMARY KEY);
  SQL
}

env "local" {
  src = "file://schema.sql"
  dev = docker.sqlserver.dev.url
}
```

- `database` names the dev database. `collate` sets its collation and needs `database`; on `master` it
  fails with `docker/sqlserver: can not modify collate of the master database`.
- `baseline` creates what the schema references but other teams own, such as a shared lookup table. A
  baseline object is part of the desired state, so it must also exist on every target. Do not
  `exclude` a table the schema references: `exclude` also removes the foreign keys that point at it,
  without an error.
- `init` runs a script on `master` when the container starts (logins, server settings); `baseline` runs
  in the dev database. A longer script lives in a file: `baseline = file("baseline.sql")`.

## Your Own Dev Database

Without Docker, a `dev` block connects to an existing empty database, applies the `baseline`, and
restores the database to its original state on exit
(https://atlasgo.io/concepts/dev-database#providing-your-own-dev-database):

```hcl
dev "sqlserver" "shared" {
  url      = getenv("DEV_URL")    # sqlserver://...?database=atlas_dev&mode=schema
  baseline = file("baseline.sql")
}

env "local" {
  src = "file://schema.sql"
  dev = dev.sqlserver.shared.url
}
```

- Use a separate, empty database that only Atlas uses, on the target's version, compatibility level,
  and collation. One server can host it next to other databases: `?database=atlas_dev`.
- Never point `dev` at a real environment's database: Atlas fails with
  `connected database is not clean: found table "orders" in connected schema`, or, if the database is
  empty, uses it as a sandbox.
- Atlas restores only the objects it manages. A schema-scoped run leaves database-level principals
  behind (a `CREATE USER` in the desired state), every run leaves objects Atlas does not manage (a
  `CREATE SYNONYM`) and server-level objects (a `CREATE LOGIN` in the baseline). The next run then fails
  with `already exists` or `There is already an object named 'x'`: guard the baseline with
  `IF SUSER_ID('x') IS NULL CREATE LOGIN ...`, keep unmanaged objects out of the desired state, or use a
  Docker dev database, which starts empty.

## Versions, Compatibility Level, and Collation

- Atlas gates features by the database's compatibility level, not the server version: a database at
  level 100 on SQL Server 2022 is treated as SQL Server 2008. Below 130 (2016), views and triggers are
  changed with `DROP` and `CREATE` instead of `CREATE OR ALTER`, and `DROP ... IF EXISTS` is not used.
  `inline = true` on a scalar function needs level 150 (2019). The lowest supported level is 100.
  Give the dev database the target's level, in the `baseline` or `init`:
  `ALTER DATABASE CURRENT SET COMPATIBILITY_LEVEL = 140`.
- Match the target's collation. On a case-sensitive dev database, a view that writes `Orders` for the
  table `orders` fails to load with `mssql: Invalid object name 'Orders'`, and on a case-insensitive
  one, the same mistake loads and then fails on a case-sensitive target.
- A column's `collate` is part of the desired state only when it differs from the database's default
  collation. Atlas never changes a database's collation.

## Azure SQL and Fabric

- Azure SQL Database and Managed Instance with Microsoft Entra ID use the `azuresql://` scheme. As with
  `sqlserver://`, `mode` sets the scope (https://atlasgo.io/concepts/url):

  ```
  azuresql://<instance>.database.windows.net?fedauth=ActiveDirectoryDefault&database=master
  ```

  `ActiveDirectoryDefault` uses the environment's Azure credentials, such as the Azure CLI. With SQL
  authentication, use `sqlserver://`. The dev database is still a Docker container on the matching
  SQL Server version.
- Fabric Data Warehouse cannot run in Docker: the dev URL is a separate, empty warehouse on
  `azuresql://`. Atlas adds `-- atlas:txmode none` to every migration file and `NOT ENFORCED` to keys,
  and a primary key sets `nonclustered = true` (https://atlasgo.io/guides/azure-fabric).

## Errors

| Error | Cause | Fix |
|-------|-------|-----|
| `The sqlserver driver is available only to Atlas Pro users`, or `login is required to use the 'sqlserver' driver` | Not logged in | `atlas login` |
| Inspection prints an empty `schema "dbo" {}` for a database with tables | The database is in the URL path, so Atlas connected to `master` | Move it to `?database=<name>` |
| `mssql: login error: Cannot open database "x" that was requested by the login` | The database does not exist, or the login has no user in it | Create the database, or fix `database` |
| `mssql: login error: Login failed for user 'x'` | Wrong user or password, or a special character breaks the URL | Percent-encode the password, or use `urlescape` |
| `cannot diff a schema "dbo" with a database connection` | The URL and the dev URL have different `mode` | Give both the same `mode` |
| `cannot use HCL with more than 1 schema when dev-url is limited to schema "dbo"` | Several `schema` blocks with a schema-scoped dev URL | Use `mode=database` on both URLs |
| `connected database is not clean: found table "x" in connected schema` | `dev` points at a database that is not empty | Use a Docker dev URL, or an empty database |
| `already exists` from the dev database on a second run | A user, role, or login stayed from an earlier run on a dev database you provide | Guard the baseline, or use a Docker dev database |
| `mssql: Invalid object name 'X'` while loading the schema | Identifier case on a case-sensitive dev database, or an object the baseline should create | Fix the case, match the collation, or add the object to the `baseline` |
| `docker/sqlserver: can not modify collate of the master database` | `collate` in a `docker` block without `database` | Set `database = "dev"` |
| `timeout: ... unable to open tcp connection` from a `docker://` URL | The container did not accept connections in time, or Docker runs elsewhere | Raise `?timeout=`, or use a separate dev container |
