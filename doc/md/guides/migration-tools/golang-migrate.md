---
id: golang-migrate
title: Automatic migration planning for golang-migrate
slug: /guides/migration-tools/golang-migrate
---

### TL;DR

* [`golang-migrate`](https://github.com/golang-migrate/migrate) is a popular database migration
  CLI tool and Go library that's widely used in the Go community.
* [Atlas](https://atlasgo.io) is an open-source tool for inspecting, planning, linting and
  executing schema changes to your database.
* Developers using `golang-migrate` can use Atlas to automatically plan schema migrations
  for them, based on the desired state of their schema instead of crafting them by hand.

### Automatic migration planning for golang-migrate

Atlas can automatically plan database schema migrations for developers using `golang-migrate`.
Atlas plans migrations by calculating the diff between the _current_ state of the database,
and it's _desired_ state.

For golang-migrate users, the current state can be thought of as the sum of
all _up_ migrations in a migration directory. The desired state can be provided to Atlas
via an a Atlas schema [HCL file](https://atlasgo.io/atlas-schema/hcl), a plain SQL file, or as a
connection string to a database that contains the desired schema.

In this guide, we will show how Atlas can automatically plan schema migrations for
golang-migrate users.

### Prerequisites

* An existing project with a `golang-migrate` migrations directory.
* Atlas ([installation guide](https://atlasgo.io/getting-started/#installation))
* Docker

### Step 1: Create a project configuration file

For this example, let's assume we have a simple `golang-migrate` project with only two files
in a directory named `migrations`:

```sql title=migrations/1_init.up.sql
create table t1
(
    c1 int
);
```

```sql title=migrations/1_init.down.sql
drop table t1;
```

To get started, create a project configuration file named `atlas.hcl` in the parent directory
of your migration directory. This file will tell Atlas where to find your migrations
and configure some basic settings.

```hcl title=atlas.hcl
env "local" {
  src = "file://schema.sql"
  dev = "docker://mysql/8/dev"
  migration {
    dir    = "file://migrations"
    format = golang-migrate
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}
```

This configuration defines an environment named `local`, that we can reference in many Atlas
commands using the `--env local` flag. Here is a breakdown of the configuration:

* `src` - Defines the desired state of the database. In this example, we use a plain SQL file
  named `schema.sql` that contains the desired state of the database. This file does not exist
  yet but we will create it in one of the following steps.
* `dev` - Atlas requires an empty database to normalize your database schema and perform
  different calculations. In this example, we provide the `docker://` driver which tells Atlas
  to spin up a local ephemeral MySQL 8 container for us to use when needed.
* `migration` - This section tells Atlas where to find your migrations and what format they are
  in. In this example, we use the `golang-migrate` format.
* `format` - This section tells Atlas how to format the output of various commands.

### Step 2: Create a migration directory integrity file

To ensure migration history is correct while multiple developers work on the same project
in parallel Atlas enforces [migration directory integrity](/concepts/migration-directory-integrity)
using a file name `atlas.sum`.

To generate this file run:

```text
atlas migrate hash --env local
```

Observe a new file named `atlas.sum` was created in your migrations directory
which contains a hash sum of each file in your directory as well as a total sum.
For example:

```text
h1:Hfk//Tj4BzMV4ZQI038FkU+zXVOky1aV8VUjj4i7/nU=
1_init.down.sql h1:zPo0X07ddhhsI7Ulxuxj/0BLqvKNK9zuUPIe4cJB3gQ=
1_init.up.sql h1:Y/8CG91XwFMRALh8DHwQ8HRQEcUOWpmM9JtH3+IZ5cM=
```

### Step 3: Create a schema file for the desired state

Automatic migration planning works by diffing the current state of the database (which is calculated
by replaying all `up` migrations on an empty database) and the desired state of the database. The desired
state of the database can be provided in many ways, but in this tutorial we will use a plain SQL file.

To extract the current state of the database as a SQL file, we will use the `schema inspect` command:

```text
atlas schema inspect --env local --url "file://migrations?format=golang-migrate" --format "{{ sql . \"  \" }}" > schema.sql
```

After running this command, you should see a new file named `schema.sql` in your project directory
that contains the current state of your database:

```sql
-- Create "t1" table
CREATE TABLE `t1`
(
    `c1` int NULL
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
```

### Step 4: Plan a new migration

Next, let's modify the desired state of our database by modifying the `schema.sql` file
to add some new columns to our table:

```sql title=schema.sql {3,5}
-- Create "t1" table 
CREATE TABLE `t1` (
  `id` int PRIMARY KEY AUTO_INCREMENT,
  `c1` int NULL,
  `c2` text NULL
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
```

Next, let's run the `atlas migrate diff` command to automatically generate a new migration
that will bring the current state of the database to the desired state:

```text
atlas migrate diff --env local new_columns
```

Hooray! Two new files were created in the migrations directory:

```text {6-7}
.
├── atlas.hcl
├── migrations
│ ├── 1_init.down.sql
│ ├── 1_init.up.sql
│ ├── 20230801091329_new_columns.down.sql
│ ├── 20230801091329_new_columns.up.sql
│ └── atlas.sum
└── schema.sql

1 directory, 7 files
```

An up migration:

```sql
-- modify "t1" table
ALTER TABLE `t1` ADD COLUMN `id` int NOT NULL AUTO_INCREMENT, ADD COLUMN `c2` text NULL, ADD PRIMARY KEY (`id`);
```

And a down migration:

```sql
-- reverse: modify "t1" table
ALTER TABLE `t1` DROP PRIMARY KEY, DROP COLUMN `c2`, DROP COLUMN `id`;
```

## Wrapping up

In this guide, we showed how to use Atlas to automatically plan schema migrations for
`golang-migrate`:

* We started by creating a project configuration file that tells Atlas where to find our migrations and how to format
  the output.
* Next, we created a migration directory integrity file to ensure migration history is correct while multiple developers
  work on the same project in parallel.
* Next, we created a schema file for the desired state of the database.
* Finally, we used the `atlas migrate diff` command to automatically generate a new migration that will bring the
  current state of the database to the desired state. 

Have questions? Feedback? Find our team [on our Discord server](https://discord.gg/zZ6sWVg6NT).
