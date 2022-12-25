## Atlas: A modern tool for managing database schemas

[![Twitter](https://img.shields.io/twitter/url.svg?label=Follow%20%40ariga%2Fatlas&style=social&url=https%3A%2F%2Ftwitter.com%2Fariga_io)](https://twitter.com/ariga_io)
[![Discord](https://img.shields.io/discord/930720389120794674?label=discord&logo=discord&style=flat-square&logoColor=white)](https://discord.com/invite/zZ6sWVg6NT)

<img width="30%" align="right" style="display: block; margin:40px auto;"
src="https://atlasgo.io/uploads/images/gopher.png"/>

Atlas is a tool for managing and migrating database schemas using modern DevOps principles. It offers two workflows:

- **Declarative**: Similar to Terraform, Atlas compares the current state of the database with the desired state defined in
an [HCL] or SQL schema, and generates a migration plan to reach that state.
- **Versioned**: Unlike other tools, Atlas automatically plans schema migrations for you. Users can describe their desired
database schema in HCL or SQL and use Atlas CLI to plan, lint, and apply the necessary migrations.

## Quick installation

**macOS + Linux:**

```bash
curl -sSf https://atlasgo.sh | sh
```

**Homebrew:**

```bash
brew install ariga/tap/atlas
```

**Docker:**

```bash
docker pull arigaio/atlas
```

Click [here](https://atlasgo.io/getting-started#installation) to read instructions for other platforms.

## Getting started
Get started with Atlas by following the [Getting Started](https://atlasgo.io/getting-started/) docs.
This tutorial teaches you how to inspect a database, generate a migration plan and apply the migration to your database.

## Key features:

- **Schema management**: The `atlas schema` command offers various options for inspecting, diffing, comparing, and modifying
  database schemas.
- **Versioned migration**: The `atlas migrate` command provides a state-of-the-art experience for planning, linting, and
  applying migrations.
- **Terraform support**: Managing database changes as part of a Terraform deployment workflow.
- **SQL and [HCL] support**: Atlas supports both SQL and HCL for describing database schemas.
- **Multi-tenancy**: Atlas includes built-in support for multi-tenant database schemas.
- **Cloud integration**: Atlas integrates with standard cloud services and provides an easy way to read secrets from cloud
  providers such as AWS Secrets Manager and GCP Secret Manager.

## `schema inspect`
_**Easily inspect your database schema by providing a database URL and convert it to HCL, JSON or SQL.**_

Inspect a specific MySQL schema and get its representation in Atlas DDL syntax:
```shell
atlas schema inspect -u "mysql://root:pass@localhost:3306/example" > schema.hcl
```

<details><summary>Result</summary>

```hcl
table "users" {
  schema = schema.example
  column "id" {
    null = false
    type = int
  }
  ...
}
```
</details>

Inspect the entire MySQL database and get its JSON representation:
```shell
atlas schema inspect \
  --url "mysql://root:pass@localhost:3306/" \
  --log '{{ json . }}' | jq
```

<details><summary>Result</summary>

```json
{
  "schemas": [
    {
      "name": "example",
      "tables": [
        {
          "name": "users",
          "columns": [
            ...
          ]
        }
      ]
    }
  ]
}
```
</details>

Inspect a specific PostgreSQL schema and get its representation in SQL DDL syntax:
```shell
atlas schema inspect \
  --url "postgres://root:pass@:5432/test?search_path=public&sslmode=disable" \
  --log '{{ sql . }}'
```

<details><summary>Result</summary>

```sql
-- create "users" table
CREATE TABLE "users" ("id" integer NULL, ...);
-- create "posts" table
CREATE TABLE "posts" ("id" integer NULL, ...);
```
</details>

## `schema diff`
_**Compare two schema states and get a migration plan to transform one into the other. A state can be specified using a
database URL, HCL or SQL schema, or a migration directory.**_

Compare two MySQL schemas:
```shell
atlas schema diff \
  --from mysql://root:pass@:3306/db1 \
  --to mysql://root:pass@:3306/db2
```

<details><summary>Result</summary>

```sql
-- Drop "users" table
DROP TABLE `users`;
```
</details>

Compare a MySQL schema with a migration directory:
```shell
atlas schema diff \
  --from mysql://root:pass@:3306/db1 \
  --to file://migrations \
  --dev-url docker://mysql/8/db1
````

Compare a PostgreSQL schema with an Atlas schema in HCL format:
```shell
atlas schema diff \
  --from "postgres://postgres:pass@:5432/test?search_path=public&sslmode=disable" \
  --to file://schema.hcl \
  --dev-url "docker://postgres/15/test"
````

Compare an HCL schema with an SQL schema:
```shell
atlas schema diff \
  --from file://schema.sql \
  --to file://schema.hcl \ 
  --dev-url docker://postgres/15/test  
````

## `schema apply`
_**Generate a migration plan and apply it to the database to bring it to the desired state. The desired state can be
specified using a database URL, HCL or SQL schema, or a migration directory.**_

Update the database to the state defined in the HCL schema:
```shell
atlas schema apply \
  --url mysql://root:pass@:3306/db1 \
  --to file://schema.hcl \
  --dev-url docker://mysql/8/db1
```

<details><summary>Result</summary>

```shell
-- Planned Changes:
-- Modify "users" table
ALTER TABLE `db1`.`users` DROP COLUMN `d`, ADD COLUMN `c` int NOT NULL;
Use the arrow keys to navigate: ↓ ↑ → ← 
? Are you sure?: 
  ▸ Apply
    Abort
```
</details>

Update the database to the state defined in a specific version of the migration directory:
```shell
atlas schema apply \
  --url mysql://root:pass@:3306/db1 \
  --to "file://migrations?version=20221118091226" \
  --dev-url docker://mysql/8/db1
```

### Additional `schema` commands
Atlas offers additional commands to assist users managing their database schemas. These include `schema clean` and
`schema fmt`. For more information, see the versioned migration documentation at https://atlasgo.io/declarative/inspect.

## `migrate diff`
_**Write a new migration file to the migration directory that bring it to the desired state. The desired state can be
specified using a database URL, HCL or SQL schema, or a migration directory.**_

Create a migration file named `add_blog_posts` in the migration directory to bring the database to the state defined
in an HCL schema:
```shell
atlas migrate diff add_blog_posts \           
  --dir file://migrations \
  --to file://schema.hcl \
  --dev-url docker://mysql/8/test
```

Create a migration file named `add_blog_posts` in the migration directory to bring the database to the state defined
in an SQL schema:
```shell
atlas migrate diff add_blog_posts \           
  --dir file://migrations \
  --to file://schema.sql \
  --dev-url docker://mysql/8/test
```

Create a migration file named `add_blog_posts` in the migration directory to bring the database to the state defined
by another database:
```shell
atlas migrate diff add_blog_posts \           
  --dir file://migrations \
  --to mysql://root:pass@host:3306/db \
  --dev-url docker://mysql/8/test
```

## `migrate apply`
_**Apply all or part of pending migration files in the migration directory on the database.**_

Apply all pending migration files in the migration directory on a MySQL database:
```shell
atlas migrate apply \
  --url mysql://root:pass@:3306/db1 \
  --dir file://migrations
```

Apply in **dry run** mode the first the pending migration file in the migration directory on a PostgreSQL schema:
```shell
atlas migrate apply \
  --url "postgres://root:pass@:5432/test?search_path=public&sslmode=disable" \
  --dir file://migrations \
  --dry-run
```

### Additional `migrate` commands
Atlas offers additional commands to assist users managing their database migrations. These include `migrate lint`,
`migrate status`, and more. For more information, see the versioned migration documentation at https://atlasgo.io/versioned/diff.

### Supported databases
MySQL, MariaDB, PostgresSQL, SQLite, TiDB, CockroachDB

[HCL]: https://github.com/hashicorp/hcl