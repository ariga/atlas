---
id: golang-migrate
title: Automatic migration planning for golang-migrate
slug: /guides/migration-tools/golang-migrate
---

## TL;DR
* [`golang-migrate`](https://github.com/golang-migrate/migrate) is a popular database migration
CLI tool and Go library that's widely used in the Go community.
* [Atlas](https://atlasgo.io) is an open-source tool for inspecting, planning, linting and
 executing schema changes to your database. 
* Developers using `golang-migrate` can use Atlas to automatically plan schema migrations
  for them, based on the desired state of their schema instead of crafting them by hand. 

## Automatic migration planning for golang-migrate

Atlas can automatically plan database schema migrations for developers using `golang-migrate`.
Atlas plans migrations by calculating the diff between the _current_ state of the database,
and it's _desired_ state. 

For golang-migrate users, the current state can be thought of as the sum of 
all _up_ migrations in a migration directory. The desired state can be provided to Atlas
via an Atlas schema [HCL file](https://atlasgo.io/atlas-schema/sql-resources) or as a
connection string to a database that contains the desired schema.

In this guide, we will show how Atlas can automatically plan schema migrations for
golang-migrate users. 

## Prerequisites

* An existing project with a `golang-migrate` migrations directory.
* Docker
* Atlas ([installation guide](https://atlasgo.io/getting-started/#installation))

## Dev database
To plan a migration from the current to the desired state, Atlas uses a [Dev Database](/concepts/dev-database),
which is usually provided by a locally running container with an empty database of the type
you work with (such as MySQL or PostgreSQL). 

To spin up a local MySQL database that will be used as a dev-database in our example, run:

```text
docker run --rm --name atlas-db-dev -d -p 3306:3306 -e MYSQL_DATABASE=dev -e MYSQL_ROOT_PASSWORD=pass mysql:8
```

As reference for the next steps, the URL for the Dev Database will be:
```text
mysql://root:pass@localhost:3306/dev
```

## Migration directory integrity

To ensure migration history is correct while multiple developers work on the same project
in parallel Atlas enforces [migration directory integrity](/concepts/migration-directory-integrity)
using a file name `atlas.sum`. 

For this example, we will assume your migrations are stored in a directory named `migrations`
in your current working directory:

To generate this file run:

```text
atlas migrate hash --dir file://migrations
```

Observe a new file named `atlas.sum` was created in your migrations directory
which contains a hash sum of each file in your directory as well as a total sum.
For example:
```text
h1:y6Zf8kAu98N0jAR+yemZ7zT91nUyECLWzxxR7GHJIAg=
1_init.down.sql h1:0zpQpoUZcacEatOD+DYXgYD1XvfWUC7EM+agXIRzKRU=
1_init.up.sql h1:kOM+4u8UsYvvjQMFYAo2hDv5rbx3Mdbh9GvhmbpS0Ig=
```

## Convert your migrations directory to an Atlas schema file

With Atlas, users can describe their desired schema using an [HCL-based configuration language](https://atlasgo.io/atlas-schema/sql-resources).
As a new user coming from an existing project, you may not want to learn this new language and
prefer that Atlas will generate a schema file that reflects your existing schema. 

:::info

If you want to read the desired state from a database instead of an Atlas HCL schema file, 
have a look [here](/guides/migration-tools/golang-migrate#alternative-use-an-existing-database-as-the-desired-state).

:::

Let's see how you can get your current schema from `golang-migrate` to the Atlas schema format.

1\. Open a MySQL shell inside our running container:
```text
docker exec -it atlas-db-dev mysql -ppass
```

2\. Create a new database named `migrate-current`:
```text
CREATE DATABASE `migrate-current`
```
The database is created successfully:
```text
Query OK, 1 row affected (0.01 sec)
```

3\. Close the shell and run `golang-migrate` to run all migrations on our new database:

```text
migrate -source file://migrations -database 'mysql://root:pass@tcp(localhost:3306)/migrate-current' up
```
All migrations are executed successfully:
```text
1/u init (35.601678ms)
```

4\. Next, use Atlas's `schema inspect` command to write a file named `schema.hcl` with an HCL
representation of your migration directory. Notice that we exclude the `schema_migrations` table
which contains `golang-migrate`'s revision history. 

```text
atlas schema inspect -u mysql://root:pass@localhost:3306/migrate-current --exclude '*.schema_migrations' > schema.hcl
```

Observe that a new file named `schema.hcl` was created:

```hcl
table "users" {
  schema = schema.migrate-current
  column "id" {
    null = false
    type = int
  }
  column "name" {
    null = true
    type = varchar(100)
  }
  primary_key {
    columns = [column.id]
  }
}
schema "migrate-current" {
  charset = "utf8mb4"
  collate = "utf8mb4_0900_ai_ci"
}
```

## Plan a new migration

Next, let's modify the desired state of our database by modifying the `schema.hcl` file 
to add a new table `blog_posts` that has a foreign key pointing to the existing `users`
table:

```hcl
table "blog_posts" {
  schema = schema.migrate-current
  column "id" {
    type = int
  }
  column "title" {
    type = varchar(255)
  }
  column "body" {
    type = text
  }
  column "author_id" {
    type = int
  }
  foreign_key "blog_author_fk" {
    columns = [column.author_id]
    ref_columns = [table.users.column.id]
  }
}
```

Now, let's use Atlas's `migrate diff` command to plan a migration from the current state
as it exists in the migrations directory to the desired state that is defined by the `schema.hcl`
file:

```shell
atlas migrate diff add_blog_posts \
  --dir "file://migrations?format=golang-migrate" \
  --dev-url "mysql://root:pass@localhost:3306/dev" \
  --to "file://schema.hcl" 
```

Notice that we used the `format` query parameter to specify that we're using `golang-migrate` as the directory format.

Hooray! Two new files were created in the migrations directory:
```text {5-6}
.
├── migrations
│ ├── 1_init.down.sql
│ ├── 1_init.up.sql
│ ├── 20220922123326_add_blog_posts.down.sql
│ ├── 20220922123326_add_blog_posts.up.sql
│ └── atlas.sum
└── schema.hcl
```
An up migration:
```sql
-- create "blog_posts" table
CREATE TABLE `blog_posts` (`id` int NOT NULL, `title` varchar(255) NOT NULL, `body` text NOT NULL, `author_id` int NOT NULL, INDEX `blog_author_fk` (`author_id`), CONSTRAINT `blog_author_fk` FOREIGN KEY (`author_id`) REFERENCES `users` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
```
And a down migration:
```sql
-- reverse: create "blog_posts" table
DROP TABLE `blog_posts`;
```

## Alternative: use an existing database as the desired state

In some cases, it is convenient to use the schema of an existing database as the desired
state for your project, instead of defining it in HCL. Atlas's `migrate diff` command can
plan a migration from your current migration directory state to an existing schema. Suppose such a 
database was available at `mysql://root:pass@some.db.io:3306/db`, a migration to the state
of that database could be planned by running:

```shell
atlas migrate diff migration_name \
  --dir "file://migrations?format=golang-migrate" \
  --dev-url "mysql://root:pass@localhost:3306/dev" \
  --to "mysql://root:pass@some.db.io:3306/db"
```

## Conclusion

We began our demo by explaining how to set up a dev-database and `atlas.sum` file for your project. 
Next, we showed how to use Atlas's `schema inspect` command to extract the current desired 
schema of your project from an existing migration directory. Finally, we showed how to 
automatically plan a schema migration by modifying the desired schema definition 
and using Atlas's `migrate diff` command.

Have questions? Feedback? Find our team [on our Discord server](https://discord.gg/zZ6sWVg6NT).
