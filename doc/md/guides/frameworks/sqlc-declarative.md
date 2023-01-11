---
id: sqlc-declarative
title: Declarative migrations for sqlc
slug: /guides/frameworks/sqlc-declarative
---

[sqlc](https://sqlc.dev/) is a tool that generates type-safe idiomatic Go code from SQL queries. It's like a transpiler,
where you provide the required queries along with the database schema, and sqlc generates code that implements type-safe
interfaces for these queries.

sqlc does not impose any requirements for handling migrations. While it has [support for
basic DDL commands](https://docs.sqlc.dev/en/latest/howto/ddl.html), it's not designed for handling the migration
process.

In this guide we will show you how Atlas can be used with sqlc in a declarative way, filling the gaps and providing a
complete solution for building applications with sqlc.

For this guide we will assume that you already have a project using sqlc. If you are new to sqlc or don't have a project yet,
[check the getting started guide](https://docs.sqlc.dev/en/latest/tutorials/getting-started-postgresql.html).

## Schema

To generate code, sqlc depends on the database schema and the queries to be generated against it. We need both the
schema and queries to ensure sqlc knows what types and queries it needs to generate.

As your application’s schema evolves, drift between the desired schema and database can cause many issues with sqlc,
since the most common time to detect these errors is during query execution.

In this guide, we will show how sqlc users can utilize Atlas’s declarative schema migration workflow to ensure their
database schemas are always in sync with their desired state.

## Desired state

Since both Atlas and sqlc can accept a single SQL file to describe the desired state of the database schema, this file
can be the source of truth (or desired state) for both tools, letting each tool handle its part of the job.

This is a great example of the power of
the [declarative concept](https://atlasgo.io/concepts/declarative-vs-versioned#declarative-migrations), where we declare
what we expect and let the tooling figure out how to reach the desired state.

## Migrating the initial schema

If you followed the sqlc tutorial, you may end up with a `schema.sql` file that looks like this:

```sql title="schema.sql"
CREATE TABLE authors
(
  id   BIGSERIAL PRIMARY KEY,
  name text NOT NULL,
  bio  text
);
```

And a `query.sql` file that looks like this:

```sql title="query.sql"
-- name: GetAuthor :one
SELECT *
FROM authors
WHERE id = $1
LIMIT 1;

-- name: ListAuthors :many
SELECT *
FROM authors
ORDER BY name;

-- name: CreateAuthor :one
INSERT INTO authors (name, bio)
VALUES ($1, $2)
RETURNING *;

-- name: DeleteAuthor :exec
DELETE
FROM authors
WHERE id = $1;
```

You may already have executed the `sqlc generate` command, if not, make sure to execute it.

The next step is migrating the database, for the purpose of this guide, we will assume that our application is backed by
a PostgreSQL database running in a local Docker container. You can start such a container by running:

```shell
docker run --rm -d --name atlas-sqlc -p 5432:5432 -e POSTGRES_PASSWORD=pass -e POSTGRES_DB=sqlc postgres
```

:::info
While in this example we are using Postgres, this is not a requirement by sqlc or Atlas. You can switch to any other
database as long the `schema.sql` is compatible.
:::

The first thing we do is initialize the database schema. This can be accomplished with a simple
command:

```shell
atlas schema apply \
--url "postgres://postgres:pass@localhost:5432/sqlc?sslmode=disable" \
--dev-url "docker://postgres" \
--to "file://schema.sql"
```

After executing the command, you should see the planned changes, similar to the example below:

```text {2-5}
-- Planned Changes:
-- Add new schema named "public"
CREATE SCHEMA "public";
-- Create "authors" table
CREATE TABLE "public"."authors" ("id" bigserial NOT NULL, "name" text NOT NULL, "bio" text NULL, PRIMARY KEY ("id"));
Use the arrow keys to navigate: ↓ ↑ → ←
? Are you sure?:
  ▸ Apply
    Abort
```

:::info
Atlas CLI will ask for approval during the command execution. If you want to skip this check, you can use the
flag `--auto-approve`.
:::

Let's break this command down: first we are telling Atlas to connect to the database using the `url` flag, then compare
to the desired state defined by the `to` flag and apply all the changes required to ensure we get to the desired state.

There is one more flag used on the command: `dev-url`. Atlas uses a temporary database to check, simulate and validate
the generated queries. For more information about the dev database, [read about it here](https://atlasgo.io/concepts/dev-database). Atlas has
support for running database containers, which is what we are using in this example.

After running the command above and confirming the changes, your database should be in sync with the `schema.sql` file.

## Evolving the schema

As your application evolves, it is very common to have the database schema evolve as well. Suppose that to support a new
feature in your application, you need to add a new `age` column. With Atlas, the process can be as simple as updating
the `schema.sql` to the desired schema, updating the `query.sql` file, executing the `sqlc generate` comamnd
and running `schema apply` again.

First, let's update our `schema.sql` file, adding the new column.

```sql title="schema.sql" {5-6}
CREATE TABLE authors
(
  id   BIGSERIAL PRIMARY KEY,
  name text NOT NULL,
  bio  text,
  age  integer
);
```

Let's add a new `GetAuthorsByAge` query and update our `CreateAuthor` query.

```sql title="query.sql" {7-11,19-20}
-- name: GetAuthor :one
SELECT *
FROM authors
WHERE id = $1
LIMIT 1;

-- name: GetAuthorsByAge :many
SELECT *
FROM authors
WHERE age = $1
ORDER BY age;

-- name: ListAuthors :many
SELECT *
FROM authors
ORDER BY name;

-- name: CreateAuthor :one
INSERT INTO authors (name, bio, age)
VALUES ($1, $2, $3)
RETURNING *;

-- name: DeleteAuthor :exec
DELETE
FROM authors
WHERE id = $1;
```

After changes to the `schema.sql` or `query.sql` file, we must run the sqlc generate command again:

```shell
sqlc generate
```

:::note
It's important to run `sqlc generate` even for changes that don't change the `query.sql` file, since sqlc
will replace the `*` with explicit column names, to ensure the query never returns unexpected data.
:::

As the last step, we run the `atlas schema apply` command to ensure the database (whether production or development) is
in sync with the desired state.

```shell
atlas schema apply \
--url "postgres://postgres:pass@localhost:5432/sqlc?sslmode=disable" \
--dev-url "docker://postgres" \
--to "file://schema.sql"
```

This time Atlas should show a different plan for executing these changes:

```text {2-3}
-- Planned Changes:
-- Modify "authors" table
ALTER TABLE "public"."authors" ADD COLUMN "age" integer NULL;
Use the arrow keys to navigate: ↓ ↑ → ←
? Are you sure?:
  ▸ Apply
    Abort
```

That’s it! After making changes to your schema, all you have to do is apply these changes again and Atlas
will handle the rest.

## Complete workflow

With the declarative strategy, one can visualize the complete workflow of using Atlas with sqlc as follows:

- Update the schema (`schema.sql`)
  - Optionally update the queries (`query.sql`)
- Run sqlc to generate the updated code
- Run Atlas during the development and migration process, referencing the `schema.sql` as the desired state.

## Wrapping up

In this guide we saw how Atlas can be used with sqlc in a declarative way, making the schema management process a
breeze. If you don’t like the approach of handling the migration in a declarative way, Atlas has support for [versioned
migrations](https://atlasgo.io/concepts/declarative-vs-versioned#versioned-migrations) as well.