---
title: "Announcing Atlas v0.3.2: multi-schema support"
date: "2022-02-01"
author: Ze'ev Manilovich
authorURL: "https://github.com/zeevmoney"
authorImageURL: "https://avatars.githubusercontent.com/u/7361100?v=4"
authorTwitter: zeevmoney
url: /announcing-atlas-v.0.3.2-multi-schema/
image: https://blog.ariga.io/uploads/images/posts/v0.3.2/multi-schema.png
---

Last week we released [v0.3.2](https://github.com/ariga/atlas/releases/tag/v0.3.2)
of the Atlas CLI.

[Atlas](https://atlasgo.io) is an open source tool that helps developers manage their database schemas. Atlas plans database
migrations for you based on your desired state. The two main commands are `inspect` and `apply`. The `inspect` command inspects your database
and the `apply` command runs a migration by providing an HCL document with your desired state.

The most notable change in this version is the ability to interact
with multiple schemas in both database inspection and migration (the `apply` command).

Some other interesting features include:
* `schema apply --dry-run` - running `schema apply` in dry-run mode connects to the target database
  and prints the SQL migration to bring the
  target database to the desired state without prompting the user to approve it.
* `schema fmt` - adds basic formatting capabilities to .hcl files.
* `schema diff` - Connects to two given databases, inspects them, calculates
  the difference in their schemas, and prints a plan of SQL statements needed to migrate the "from" database
  to the state of the "to" database.

In this post we will explore the topic of multi-schema support.  We will start our discussion
with a brief explanation of database schemas, next we'll present the difference between
how MySQL and PostgreSQL treat "schemas". We will then show how the existing `schema inspect`
and `schema apply` commands work with multi-schema support, and wrap up with some plans for future releases.

#### What is a database schema?

Within the context of relational (SQL) databases, a database schema is a logical unit within
a physical database instance (server/cluster) that forms a namespace of sorts.
Inside each schema you can describe the structure of the tables, relations, indexes and other attributes that belong to it.
In other words, the database schema is a "blueprint" of the data structure inside a logical
container (Note: in Oracle databases a schema [is linked to the user](https://docs.oracle.com/cd/B19306_01/server.102/b14196/schema.htm#ADMQS008),
so it carries a different meaning which is out of scope for this post). As you can guess from the
title of this post, many popular relational databases allow users to host _multiple_ (logical) schemas
on the same (physical) database.

#### Where are database schemas used in practice?

Why is this level of logical division necessary? Isn't it enough to be able
physically split data into different database instances? In my career, I've seen
multiple scenarios in which organizations opt to split a database into multiple schemas.

First, grouping different parts of your application into logical units makes it simpler
to reason about and govern. For instance, it is possible to create multiple user accounts in our database
and give each of them permission to access a subset of the schemas in the database. This way, each
user can only touch the parts of the database they need, preventing the practice of creating
an almighty super-user account that has no permission boundary.

An additional pattern I've seen used, is in applications with a multi-tenant architecture where each
tenant has its own schema with the same exact table structure (or some might have a
different structure since they use different versions of the application). This pattern is used
to create a stronger boundary between the different tenants (customers) preventing the scenario
where one tenant accidentally has access to another's data that is incidentally hosted on the
same machine.

Another useful feature of schemas is the ability to divide the same server into
different environments for different development states. For example, you can have
a "dev" and "staging" schema inside the same server.

#### What are the differences between schemas in MySQL and PostgreSQL?

A common source of confusion for developers (especially when switching teams or companies) is
the difference between the meaning of schemas in MySQL and PostgreSQL. Both are currently supported by Atlas, and have some
differences that should be clarified.

Looking at the MySQL [glossary](https://dev.mysql.com/doc/refman/8.0/en/glossary.html#glos_schema), it states:

> "In MySQL, physically, a schema is synonymous with a database. You can substitute the
keyword SCHEMA instead of DATABASE in MySQL SQL syntax, for example using CREATE SCHEMA
instead of CREATE DATABASE"

As we can see, MySQL doesn't distinguish between schemas and databases in the terminology,
but the underlying meaning is still the same - a logical boundary for resources and permissions.

To demonstrate this, open your favorite MySQL shell and run:

```shell
mysql> create schema atlas;
Query OK, 1 row affected (0.00 sec)
```
To create a table in our new schema, assuming we have the required permissions, we
can switch to the context of the schema that we just created, and create a table:
```sql
USE atlas;
CREATE table some_name (
    id int not null
);
```

Alternatively, we can prefix the schema, by running:
```sql
CREATE TABLE atlas.cli_versions
(
    id      bigint auto_increment primary key,
    version varchar(255) not null
);
```

This prefix is important since, as we said, schemas are logical boundaries (unlike database servers).
Therefore, we can create references between them using foreign keys from tables in
`SchemaA` to `SchemaB`. Let's demonstrate this by creating another schema with a table and
connect it to a table in the `atlas` schema:

```sql
CREATE SCHEMA atlantis;

CREATE TABLE atlantis.ui_versions
(
    id               bigint auto_increment
        primary key,
    version          varchar(255) not null,
    atlas_version_id bigint       null,
    constraint ui_versions_atlas_version_uindex
        unique (atlas_version_id)
);
```

Now let's link `atlantis` to `atlas`:

```sql
alter table atlantis.ui_versions
    add constraint ui_versions_cli_versions_id_fk
        foreign key (atlas_version_id) references atlas.cli_versions (id)
            on delete cascade;
```

That's it! We've created 2 tables in 2 different schemas with a reference between them.

##### How does PostgreSQL treat schemas?

When booting a fresh PostgreSQL server, we get a default logical schema called "public".
If you wish to split your database into logical units as we've shown with MySQL, you can
create a new schema:

```sql
CREATE SCHEMA atlas;
```

Contrary to MySQL, Postgres provides an additional level of abstraction: databases.
In Postgres, a single physical server can host multiple databases. Unlike schemas
(which are basically the same as in MySQL) - you can't reference a table from one
PostgreSQL database to another.

In Postgres, the following statement will create an entirely new database, where we can place different
schemas and tables with that may contain references between them:
```sql
create database releases;
```
When we run this statement, the database will be created with the default Postgres
metadata tables and the default `public` schema.

In Postgres, you can give permissions to an entire database(s), schema(s),
and/or table(s), and of course other objects in the Postgres schema.

Another distinction from MySQL is that in addition to sufficient permissions, a user must have the schema name
inside their [search_path](https://www.postgresql.org/docs/14/ddl-schemas.html#DDL-SCHEMAS-PATH) in
order to use it without a prefix.

To sum up, both MySQL and Postgres allow the creation of separate logical schemas within a physical database
server, schemas can refer to one another via foreign-keys. PostgreSQL supports an additional level of separation
by allowing users to create completely different _databases_ on the server.

#### Atlas multi-schema support

As we have shown, having multiple schemas in the same database is a common scenario with popular
relational databases. Previously, the Atlas CLI only supported inspecting or applying changes to a single
schema  (even though this has been long supported in the [Go API](https://atlasgo.io/go-api/intro)). With
this release, we have added support for inspecting and applying multiple schemas with a single `.hcl` file.

Next, let's demonstrate how we can use the Atlas CLI to inspect and manage a database with multiple schemas.

Start by [downloading and installing the latest version](https://atlasgo.io/cli/getting-started/setting-up#install-the-cli)
of the CLI. For the purpose of this demo, we will start with a fresh database of MySQL running in a local `docker`
container:

```shell
docker run --name atlas-db  -p 3306:3306 -e MYSQL_ROOT_PASSWORD=pass -e MYSQL_DATABASE=example mysql:8
```

By passing `example` in the `MYSQL_DATABASE` environment variable a new schema named
"example" is created. Let's verify this by using the `atlas schema inspect` command. In previous
versions of Atlas, users had to specify the schema name as part of the DSN for connecting to the
database, for example:
```shell
atlas schema inspect -u "mysql://root:pass@localhost:3306/example"
```
Starting with `v0.3.2`, users can omit the schema name from the DSN to instruct Atlas to inspect
the entire database. Let's try this:
```shell
$ atlas schema inspect -u "mysql://root:pass@localhost:3306/" > atlas.hcl
cat atlas.hcl
schema "example" {
  charset   = "utf8mb4"
  collation = "utf8mb4_0900_ai_ci"
}
```
Let's verify that this works correctly by editing the `atlas.hcl` that we have created above
and adding a new schema:

```hcl
schema "example" {
  charset   = "utf8mb4"
  collation = "utf8mb4_0900_ai_ci"
}
schema "example_2" {
  charset   = "utf8mb4"
  collation = "utf8mb4_0900_ai_ci"
}
```
Next, we will use the `schema apply` command to apply our changes to the database:

```shell
atlas schema apply -u "mysql://root:pass@localhost:3306/" -f atlas.hcl
```
Atlas plans a migration to add the new `DATABASE` (recall that in MySQL `DATABASE` and
`SCHEMA` are synonymous) to the server, when prompted to approve the migration we choose "Apply":
```shell
-- Planned Changes:
-- Add new schema named "example_2"
CREATE DATABASE `example_2`
✔ Apply
```

To verify that `schema inspect` works properly with multiple schemas, lets re-run:
```shell
atlas schema inspect -u "mysql://root:pass@localhost:3306/"
```
Observe that both schemas are inspected:
```hcl
schema "example" {
  charset   = "utf8mb4"
  collation = "utf8mb4_0900_ai_ci"
}
schema "example_2" {
  charset   = "utf8mb4"
  collation = "utf8mb4_0900_ai_ci"
}
```
To learn more about the different options for working with multiple schemas in `inspect`
and `apply` commands, consult the [CLI Reference Docs](https://atlasgo.io/cli-reference#atlas-schema-inspect).

#### What's next for multi-schema support?

I hope you agree that multi-schema support is a great improvement to the Atlas CLI, but there is
more to come in this area. In our previous [blogpost](https://atlasgo.io/blog/2022/01/19/atlas-v030) we have shared that
Atlas also has a [Management UI](https://atlasgo.io/ui/intro) (-w option in the CLI) and multi-schema
support is not present there yet - stay tuned for updates on multi-schema support for the UI in an upcoming release!

#### Getting involved with Atlas
* Follow the [Getting Started](https://atlasgo.io/cli/getting-started/setting-up) guide..
* Join our [Discord Server](https://discord.gg/zZ6sWVg6NT).
* Follow us [on Twitter](https://twitter.com/ariga_io).
