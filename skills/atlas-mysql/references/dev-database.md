# MySQL Dev Database and Connections

The dev database is a temporary database Atlas uses to load the desired state, replay migrations, and
compute plans (https://atlasgo.io/concepts/dev-database). For MySQL it must match the target's version,
scope, and default character set and collation.

## Scope

In MySQL, a schema is a database. The URL's path names it, and decides what Atlas covers
(https://atlasgo.io/concepts/url#scope):

| URL | Scope | Atlas covers | DDL it writes |
|-----|-------|--------------|---------------|
| `mysql://localhost:3306/dev` | Schema | That database only | Unqualified (`users`) |
| `mysql://localhost:3306/` | Server | Every database on the server | Qualified (`app.users`) |

- The dev URL takes the target's scope: `docker://mysql/8/dev` for a schema-scoped target, and
  `docker://mysql/8` for a server-scoped one (https://atlasgo.io/concepts/dev-database). Mixing them
  leads to wrong plans.
- Users and roles belong to the server, not to a database: managing them needs server scope
  (`references/security.md`). So do several databases, and migrations with qualified names.
- At server scope, the desired state covers every database on the server: a database it does not
  declare is planned for `DROP DATABASE`. Limit the run to the databases it manages with `--schema`, or
  use schema scope.
- A plan made at schema scope has unqualified statements. Applying it through a server-scoped URL fails
  with `Error 1046: No database selected`.
- At schema scope, Atlas cannot change the database's own character set, collation, or comment: see
  Character Sets and Collations below.
- `--include` patterns follow the scope (https://atlasgo.io/faq/manage-single-table). At schema scope a
  pattern names the table, and at server scope it starts with the database:

  ```bash
  atlas schema apply \
    --url "mysql://root:pass@:3308/db" \
    --to "file://schema.sql" \
    --dev-url "docker://mysql/8/dev" \
    --include "products"
  ```

  At server scope, the same filter is `--include "db.products"` with `--dev-url "docker://mysql/8"`.

## Docker Dev URLs

The `docker://` driver starts a fresh container for each run (https://atlasgo.io/concepts/dev-database):

```bash
# When working on a single database schema.
--dev-url "docker://mysql/8/dev"

# When working on multiple database schemas.
--dev-url "docker://mysql/8"
```

Custom images:

```bash
# When working on a single database schema.
docker+mysql://org/image/dev
docker+mysql://user/image:tag/dev
# For local/official images, leave host empty or use "_".
docker+mysql:///local/dev
docker+mysql://_/mysql:latest/dev

# When working on multiple database schemas.
docker+mysql://local
docker+mysql://org/image
docker+mysql://user/image:tag
docker+mysql://_/mysql:latest
```

- The URL pins only the major version. To run the target's exact version, set `image` in a `docker`
  block, such as `image = "mysql:8.4"`. Lint reads the dev database's version for checks such as
  `MY142` (`references/migrations.md`).
- MariaDB has its own URLs. Run a MariaDB dev database for a MariaDB target: a MySQL dev database
  renders collations and defaults differently.

  ```bash
  # When working on a single database schema.
  --dev-url "docker://maria/latest/schema"

  # When working on multiple database schemas.
  --dev-url "docker://maria/latest"
  ```

- `docker://` needs a Docker client and a reachable daemon. Without one, such as in CI, run a MySQL
  service container and give its URL as the dev database, for example
  `mysql://root:pass@localhost:3306/dev` (https://atlasgo.io/faq/docker-in-docker).

### Named containers with `atlas tool docker`

`atlas tool docker` creates a named container from a `docker://` URL and prints its connection URL. It
also applies the baseline, and cleans the database if it is not empty, so never run it against a
container that holds data to keep (https://atlasgo.io/cli-reference#atlas-tool-docker). The container
runs until `atlas tool docker kill --name my-db`. Use it for a local database to apply migrations to
(https://atlasgo.io/getting-started#step-4-set-up-your-database):

```bash
# Spin up a local MySQL database
export DATABASE_URL=$(atlas tool docker --url "docker://mysql/8/myapp" --name my-db)
```

## The `docker` Block

A `docker "mysql"` or `docker "mariadb"` block configures the dev container: `image` (required),
`schema`, `baseline`, `init`, `command`, `platform`, `build`, and more
(https://atlasgo.io/hcl/config#docker.mysql). The `docker` and `dev` blocks need `atlas login`. If
`atlas whoami` shows no login, ask the user to run `atlas login`; a user without an account can start a
free trial with it.

`init` runs SQL once the server is up, such as global settings
(https://atlasgo.io/concepts/dev-database#docker-init-script):

```hcl
docker "mysql" "dev" {
  image = "mysql:8.4"
  init  = <<-SQL
    SET GLOBAL log_bin_trust_function_creators=true;
    SET GLOBAL restrict_fk_on_non_standard_key=false;
    SET GLOBAL sql_mode := REPLACE(@@sql_mode, 'NO_ZERO_DATE', '');
  SQL
}

env "dev" {
  src = "file://schema.my.hcl"
  dev = docker.mysql.dev.url
}
```

- `command` passes server startup flags, such as the character set (below).
- MySQL has no template databases; those are PostgreSQL only.

## Your Own Dev Database

A `dev "mysql"` block connects to an existing database instead of a container
(https://atlasgo.io/concepts/dev-database#providing-your-own-dev-database). The database must be empty
and dedicated to Atlas, on the same engine, version, and scope as the target. Never point `dev` at a
real environment's database: Atlas fails with `connected database is not clean`, or, if the database is
empty, uses it as a sandbox.

## Character Sets and Collations

Atlas normalizes the desired state on the dev database, so a schema, table, or column that sets no
character set or collation takes the dev server's defaults. A MySQL 8 dev database gives `utf8mb4` and
`utf8mb4_0900_ai_ci`, which then appear in every plan.

- Declare them on the schema, so the desired state does not depend on the dev server
  (https://atlasgo.io/atlas-schema/hcl#schema):

  ```hcl
  schema "market" {
    charset = "utf8mb4"
    collate = "utf8mb4_0900_ai_ci"
    comment = "A schema comment"
  }
  ```

- When the dev database's defaults differ from the target's, a schema-scoped plan tries to change the
  database's own attributes and fails with
  `modify schema "<name>" is not allowed when migration plan is scoped to one schema`. Give the dev
  database the target's settings, or use server scope for both URLs, which lets Atlas change the
  database's character set and collation (https://atlasgo.io/faq/modify-schema-error).
- A different server character set also causes drift that never converges: generated columns,
  expression defaults, functional indexes, checks, and views differ only by an introducer, such as
  `_latin1'-'` against `_utf8mb4'-'`. Start the dev server with the target's settings, through
  `command` (https://atlasgo.io/faq/mysql-charset-collation-drift):

  ```hcl
  docker "mysql" "dev" {
    image   = "mysql:8.4" # match the target version
    command = [
      "--character-set-server=latin1",
      "--collation-server=latin1_swedish_ci",
      "--skip-character-set-client-handshake",
    ]
  }
  ```

  Use `command`, not `init`: `--skip-character-set-client-handshake` is a startup flag. Read the values
  from the target on a fresh connection:

  ```sql
  SHOW VARIABLES LIKE 'character\_set\_server';
  SHOW VARIABLES LIKE 'collation\_server';
  ```

- Changing a table's character set rebuilds it (`MY136`), and a column's character set or collation
  copies the table (`MY148`); see `references/migrations.md`.

## Server Versions and Settings

The dev database behaves like the target only when it runs the same version and settings. Read them
on the target, and start the dev server with the same values through `command` in the `docker` block.

- Version. Features depend on it, and a newer dev server accepts statements the target rejects:

  | Feature | MySQL version |
  |---------|---------------|
  | Roles, descending indexes | 8.0 |
  | Functional indexes, expression defaults | 8.0.13 |
  | Enforced `CHECK` constraints (earlier versions parse and ignore them) | 8.0.16 |
  | Invisible columns | 8.0.23 |

  MySQL 5.7 has none of these. MySQL 8.4 disables the `mysql_native_password` plugin by default, and
  9.0 removes it, so a user created `IDENTIFIED WITH mysql_native_password` fails on a newer dev
  database than the target.
- `lower_case_table_names`. Linux servers, such as the dev container, default to case-sensitive table
  names, while Windows and macOS servers default to other values, and RDS takes it from the parameter
  group at creation. When the dev server and the target differ, plans keep showing table changes.
  MySQL fixes the value when it creates the data directory, which a fresh dev container does at
  startup, so pass it in `command`, such as `"--lower-case-table-names=1"`.
- `sql_mode`. Strict mode decides whether a bad value fails or is silently converted: a removed enum
  value becomes `''`, and a zero date is accepted. Run the dev server with the target's `sql_mode`,
  through `command` (`"--sql-mode=..."`) or `init`, like the `init` example above.
- `explicit_defaults_for_timestamp`. When it is off, the default before MySQL 8.0, the first
  `TIMESTAMP` column without a default becomes `NOT NULL`, with `DEFAULT CURRENT_TIMESTAMP` and
  `ON UPDATE CURRENT_TIMESTAMP`. Match the target's value, or give every `TIMESTAMP` column an explicit
  default.
- Read the target's values:

  ```sql
  SHOW VARIABLES LIKE 'version';
  SHOW VARIABLES LIKE 'lower_case_table_names';
  SHOW VARIABLES LIKE 'sql_mode';
  SHOW VARIABLES LIKE 'explicit_defaults_for_timestamp';
  ```

## Managed Providers

- RDS and Aurora MySQL with IAM authentication: the URL takes a token from the `aws_rds_token` data
  source, with `allowCleartextPasswords`
  (https://atlasgo.io/atlas-schema/projects#data-source-aws_rds_token):

  ```hcl
      url  = "mysql://iamuser:${urlescape(data.aws_rds_token.db)}@mydb.xxx.us-east-1.rds.amazonaws.com:3306/mydb?tls=true&allowCleartextPasswords=1"
  ```

  IAM users on the dev database are covered below.
- Cloud SQL for MySQL: the `gcp_cloudsql_token` data source signs in with IAM
  (https://atlasgo.io/atlas-schema/projects#data-source-gcp_cloudsql_token):

  ```hcl
    url = "mysql://${local.user}:${urlescape(data.gcp_cloudsql_token.db)}@${local.endpoint}/?allowCleartextPasswords=1&tls=skip-verify&parseTime=true"
  ```

- Azure Database for MySQL with Entra ID: the user name contains `@`, so build the URL with
  `urluserinfo` (https://atlasgo.io/atlas-schema/projects#data-source-azure_db_token):

  ```hcl
  data "azure_db_token" "db" {}

  env "azure" {
    url = urluserinfo(
      "mysql://myserver.mysql.database.azure.com:3306/?tls=true&allowCleartextPasswords=1",
      "app@contoso.onmicrosoft.com",
      data.azure_db_token.db,
    )
  }
  ```

- MariaDB connects with `maria://` URLs, such as `maria://user:pass@localhost:3306/schema`. This skill
  does not cover other MySQL-compatible engines, such as TiDB, PlanetScale, or SingleStore.

### RDS IAM users on the dev database

RDS creates IAM users with a plugin that exists only on RDS. A migration directory or a SQL schema with
such a statement, `CREATE USER 'app_iam' IDENTIFIED WITH AWSAuthenticationPlugin AS 'RDS';`, fails on a
stock MySQL image (https://atlasgo.io/faq/mock-mysql-rds-aws-auth-plugin):

```
Error: failed to run `atlas migrate diff`: Error: sql/migrate: read migration directory state:
sql/migrate: executing statement "CREATE USER 'app_iam' IDENTIFIED WITH AWSAuthenticationPlugin AS 'RDS';"
from version "20260602081826": Error 1524 (HY000): Plugin 'AWSAuthenticationPlugin' is not loaded
```

- Every command that replays the migrations or loads the schema on the dev database fails this way:
  `migrate diff`, `migrate lint`, `migrate test`, `schema plan`, and `schema apply`.
- A local database to apply the migrations to, such as one from `atlas tool docker --url docker://...`,
  fails on the same statement, since it runs the stock image too.
- Do not point `dev` at a real RDS instance to get the plugin: it is slow, needs network access and
  credentials, and the dev database stops being disposable.

Build a MySQL image with a no-op mock of the plugin instead. The mock implements just enough for the
`CREATE USER` statement to succeed, and rejects every login, so it is safe locally and in CI. Save the
FAQ's Dockerfile next to `atlas.hcl` as `aws-auth-mock.Dockerfile`. Then build it on demand from a
`docker` block, on the RDS instance's MySQL version:

```hcl
locals {
  // Change this to match your RDS engine version, e.g. "8.0".
  mysql_version = "8.4"
}

docker "mysql" "dev" {
  image = "mysql-aws-auth-mock:${local.mysql_version}"
  build {
    context    = "."
    dockerfile = "aws-auth-mock.Dockerfile"
    args = {
      MYSQL_VERSION = local.mysql_version
    }
  }
}
```

- Set `dev = docker.mysql.dev.url` in the env. When the target URL names a database, also set `schema`
  in the block to the same name, so both URLs share a scope.
- Keep `mysql_version` equal to the RDS engine version, such as `"8.0"`: the image is tagged with it,
  and lint reads the dev database's version.
- To replay the migrations on a local database, start it from the same image:
  `atlas tool docker --name my-db --env <env>` reads the env's dev URL and prints the URL of a named
  container to apply to (https://atlasgo.io/cli-reference#atlas-tool-docker).
- The target URL keeps real IAM authentication, with a token from `aws_rds_token`; the mock exists only
  in the dev and local databases.

## Connection URLs

- MySQL does not require TLS by default. `?tls=true` requires it; `?ssl-ca` sets a custom CA, and
  `?ssl-cert` with `?ssl-key` a client certificate (https://atlasgo.io/concepts/url#mysql).
- Percent-encode special characters in passwords, or build the URL with `urlescape`
  (https://atlasgo.io/faq/url-special-chars). An unescaped password fails with an error such as
  `Error: mysql: query system variables: dial tcp: lookup root:BnB+PjA:3309: no such host`.
- Atlas takes URLs: write `mysql://user:pass@host:3306/db`, not the `user:pass@tcp(host:3306)/db`
  form.
- Unix sockets use `mysql+unix://`:

  ```
  mysql+unix:///tmp/mysql.sock

  mysql+unix://user:pass@/tmp/mysql.sock

  mysql+unix://user@/tmp/mysql.sock?database=dbname
  ```

- Every user can read `information_schema`, but it shows only the objects the user has privileges on,
  and the privilege cannot be granted directly. Grant privileges on the target schema; inspecting views
  and triggers also needs `SHOW VIEW` and `TRIGGER`.
- On Windows PowerShell, `>` writes UTF-16, which fails with
  `Error 1064 (42000): You have an error in your SQL syntax` near `''`. Set
  `$PSDefaultParameterValues['Out-File:Encoding'] = 'utf8'` first
  (https://atlasgo.io/faq/schema-encoding).

## Errors

| Error | Cause | Fix |
|-------|-------|-----|
| `modify schema "<name>" is not allowed when migration plan is scoped to one schema` | The dev database's charset, collation, or comment differ from the target's | Match the dev database to the target, or use server scope for both URLs |
| `Error 1046: No database selected` | Unqualified statements applied through a server-scoped URL | Use the same scope the plan was made with |
| `cannot use HCL with more than 1 schema when dev-url is limited to schema "dev"` | An HCL schema with several databases against a schema-scoped dev URL | Use a server-scoped dev URL, `docker://mysql/8` |
| `unnecessary schema "X" specified in --dev-url for multi-schema mode` | A server-scoped target with a schema-scoped dev URL | Give both URLs server scope |
| `server does not allow insecure connections, client must use SSL/TLS` | The server requires TLS | Add `?tls=true` to the URL |
| `invalid port ":..." after host`, or `net/url: invalid userinfo` | A special character in the password breaks the URL | Percent-encode the password, or build the URL with `urlescape` |
| `Error 1049 (42000): Unknown database '<database_name>'` | The database in the URL does not exist, or its name is not escaped | Check that it exists, and quote or percent-encode the name (https://atlasgo.io/faq/mysql-error-1049-database-not-found) |
| `Error 1524 (HY000): Plugin 'AWSAuthenticationPlugin' is not loaded` | An RDS IAM user on a stock dev image | Build the dev image with a mock plugin |
| `connected database is not clean` | `dev` points at a database that is not empty | Use a Docker dev URL |
| `exec: "docker": executable file not found in $PATH` | No Docker client where Atlas runs | Use a separate dev database container |
