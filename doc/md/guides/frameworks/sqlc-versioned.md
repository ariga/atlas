---
id: sqlc-versioned
title: Versioned migrations for sqlc
slug: /guides/frameworks/sqlc-versioned
---

In our [previous sqlc guide](https://atlasgo.io/guides/frameworks/sqlc-declarative) we saw how we can use Atlas to
handle the schema management process using
the [declarative workflow](https://atlasgo.io/concepts/declarative-vs-versioned#declarative-migrations). This works
great in many situations, but some teams prefer the imperative approach where schema changes are
explicitly checked in to source control and verified during code review
(see [this video](https://www.youtube.com/watch?v=FCeIjPb4AYs) to learn why).

To accommodate such cases, Atlas supports another kind of workflow for handling migrations,
called [versioned migrations](https://atlasgo.io/concepts/declarative-vs-versioned#versioned-migrations), where the
changes are stored as SQL files and replayed on the database as needed. This ensures that the code to be executed on the
database is known and approved beforehand and gives engineers ultimate control on exactly what code is going to be run
in production.

In this guide we will see how we can use sqlc with Atlas, using the versioned migrations workflow.

:::note
If you haven't already done so, it's helpful if have read the previous [sqlc guide](link) where we have explained a some concepts that we will expand on in this guide.
:::

## Versioned migrations

[Versioned migrations](https://atlasgo.io/concepts/declarative-vs-versioned#versioned-migrations) is a migration
workflow where whenever we want to change the database schema, we create a _migration file_, which is an SQL file containing SQL statements to upgrade the schema from the current version to the next. Many schema management tools, such as Flyway and Liquibase, support this strategy and Atlas supports
it as well.


Atlas's versioned migrations workflow works very well wit sqlc. In fact, in most cases, you can simply point [sqlc schema
configuration](https://docs.sqlc.dev/en/latest/reference/config.html#sql) to the migration directory, and things should just work.

In most cases, teams using versioned migrations author the migration files manually. While this is a common approach, there are obvious downsides to it. Writing migrations by hand can be both error-prone and time-consuming for developers.

## Migration Authoring

Atlas has support for a strategy that combines the simplicity of the declarative workflow with the control provided by
the versioned one. With this approach, which is called "[versioned migration authoring](https://atlasgo.io/versioned/diff)", we still define our desired
state in HCL or SQL but use Atlas to compute the required changes, storing them in the migration directory as SQL files.

This strategy brings the best of both worlds, we can have a single source of truth (our desired state) and have all the
authored SQL files that will be used to reach the desired state.

To learn more about this approach and see it in action, watch this [YouTube video](https://www.youtube.com/watch?v=L-UlkXtp3OY)

Now that we have a bit of understanding of all the concepts, let's see how we can use Atlas with SQL with
versioned migrations.

## Creating the initial migration directory

The first thing we should do is create the initial migration directory, let's assume our sqlc `schema.sql` looks like
this:

```sql title="schema.sql"
CREATE TABLE authors
(
    id   BIGSERIAL PRIMARY KEY,
    name text NOT NULL,
    bio  text
);
```

We have to generate a diff of our desired state and our current state (the migration directory, that right now doesn't
even exist), we can generate the diff with a simple command:

```shell
atlas migrate diff initial_schema \
  --dir "file://migrations" \
  --to "file://schema.sql" \
  --dev-url "docker://postgres?search_path=public"
```

A new `migrations` directory should be created with two files on it.

```text
.
├── 20230102211759_initial_schema.sql
└── atlas.sum
```

## Migrating the database

Now that we have our base migration directory, we can migrate our schema based on it, to do so we have to use a command
very similar to the one used on the previous guide.

```shell
atlas migrate apply --url "postgres://postgres:pass@localhost:5432/sqlc?sslmode=disable"
```

The command `migrate apply` connects to the given database [URL](https://atlasgo.io/concepts/url) provided in the `url`
flag, checks the current state of the database and applies any pending migration. In this guide, we are using a Postgres
database running in a local Docker container, you can start such a container by running:

```shell
docker run --rm -d --name atlas-sqlc -p 5432:5432 -e POSTGRES_PASSWORD=pass -e POSTGRES_DB=sqlc postgres
```

After running the `migrate apply` command, you should be able to see an output similar to this:

```text
Migrating to version 20230102211759 (1 migrations in total):

  -- migrating version 20230102211759
	-> CREATE TABLE "authors" ("id" bigserial NOT NULL, "name" text NOT NULL, "bio" text NULL, PRIMARY KEY ("id"));
  -- ok (3.282921ms)

  -------------------------
  -- 4.570005ms
  -- 1 migrations
  -- 1 sql statements
```

## Updating the schema

The process of changing our schema is quite easy and similar to the declarative way. First we update our `schema.sql`
file, let's add a new `age` column.

```sql title="schema" {5-6}
CREATE TABLE authors
(
    id   BIGSERIAL PRIMARY KEY,
    name text NOT NULL,
    bio  text,
    age  integer
);
```

Now we only have to update our `migrations` directory, we can use the same command from before, using "add_age_field" as
our migration name this time:

```shell
atlas migrate diff add_age_field \
  --dir "file://migrations" \
  --to "file://schema.sql" \
  --dev-url "docker://postgres?search_path=public"
```

If you look at the migration directory, a new SQL file should be created with the commands required for adding a new
column called `age`.

```text {3}
.
├── 20230102211759_initial_schema.sql
├── 20230102212813_add_age_field.sql
└── atlas.sum
```

For applying the migrations, we only have to execute `atlas migrate apply` again:

```shell
atlas migrate apply --url "postgres://postgres:pass@localhost:5432/sqlc?sslmode=disable"
```

If everything went correctly, you should see the output similar to the one below:

```text
Migrating to version 20230102212813 from 20230102211759 (1 migrations in total):

  -- migrating version 20230102212813
	-> ALTER TABLE "authors" ADD COLUMN "age" integer NULL;
  -- ok (1.565535ms)

  -------------------------
  -- 2.839369ms
  -- 1 migrations
  -- 1 sql statements
```

## Complete workflow

With the versioned strategy, we can visualize the complete workflow of using Atlas with sqlc as follows:

- Update the schema (`schema.sql`)
    - Optionally update the queries (`query.sql`)
- Run sqlc to generate the updated code
- Run `atlas migrate diff` to compute and store the required changes
- Run Atlas during the development and migration process, referencing the migration directory.

## Wrapping up

In this guide we saw how Atlas can be used with sqlc in a versioned way. We used Atlas to create migration files, while
still keeping a single source of truth for our database schema. The versioned strategy requires more steps, but allows
for more control as well.