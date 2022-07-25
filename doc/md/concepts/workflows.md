---
title: Declarative vs Versioned Workflows
id: workflows
slug: /concepts/declarative-vs-versioned
---

This section introduces two types of workflows that are supported by Atlas
to manage database schemas: _declarative_ and _versioned_ migrations. 

### Declarative Migrations

The declarative approach has become increasingly popular with engineers nowadays because it embodies
a convenient separation of concerns between application and infrastructure engineers.
Application engineers describe _what_ (the desired state) they need to happen, and
infrastructure engineers build tools that plan and execute ways to get to that state (_how_).
This division of labor allows for great efficiencies as it abstracts away the complicated
inner workings of infrastructure behind a simple, easy to understand API for the application
developers and allows for specialization and development of expertise to pay off for the
infra people.

With declarative migrations, the desired state of the database schema is given 
as input to the migration engine, which plans and executes a set of actions to
change the database to its desired state.

For example, suppose your application uses a small SQLite database to store its data.
In this database, you have a `users` table with this structure:
```hcl
schema "main" {}

table "users" {
  schema = schema.main
  column "id" {
    type = int
  }
  column "greeting" {
    type = text
  }
}
```
Now, suppose that you want to add a default value of `"shalom"` to the `greeting`
column. Many developers are not aware that it isn't possible to modify a column's
default value in an existing table in SQLite. Instead, the common practice is to
create a new table, copy the existing rows into the new table and drop the old one
after. Using the declarative approach, developers can change the default value for
the `greeting` column:

```hcl {10}
schema "main" {}

table "users" {
  schema = schema.main
  column "id" {
    type = int
  }
  column "greeting" {
    type = text
    default = "shalom"
  }
}
```
And have Atlas's engine devise a plan similar to this:
```sql
-- Planned Changes:
-- Create "new_users" table
CREATE TABLE `new_users` (`id` int NOT NULL, `greeting` text NOT NULL DEFAULT 'shalom')
-- Copy rows from old table "users" to new temporary table "new_users"
INSERT INTO `new_users` (`id`, `greeting`) SELECT `id`, IFNULL(`greeting`, 'shalom') AS `greeting` FROM `users`
-- Drop "users" table after copying rows
DROP TABLE `users`
-- Rename temporary table "new_users" to "users"
ALTER TABLE `new_users` RENAME TO `users`
```

### Versioned Migrations

As the database is one of the most critical components in any system, applying changes
to its schema is rightfully considered a dangerous operation. For this reason, many teams
prefer a more imperative approach where each change to the database schema is checked-in 
to source control and reviewed during code-review.  Each such change
is called a "migration", as it migrates the database schema from the previous version to 
the next. To support this kind of requirement, many popular database schema management
tools such as [Flyway](https://flywaydb.org/), [Liquibase](https://liquibase.org/) or 
[golang-migrate](https://github.com/golang-migrate/migrate) support a workflow that
is commonly called "versioned migrations".

In addition to the higher level of control which is provided by versioned migrations,
applications are often deployed to multiple remote environments at once. These environments,
are not controlled (or even accessible) by the development team. In such cases, declarative migrations, 
which rely on a network connection to the target database and on human
approval of migrations plans in real-time, are not a feasible strategy.

With versioned migrations (sometimes called "change-based migrations") instead of describing 
the desired state ("what the database should look like"), developers describe the changes themselves 
("how to reach the state"). Most of the time, this is done by creating a set of SQL files 
containing the statements needed. Each of the files is assigned a unique version and a
description of the changes. Tools like the ones mentioned earlier are then able to 
interpret the migration files and to apply (some of) them in the correct order to 
transition to the desired database structure.

The benefit of the versioned migrations approach is that it is explicit: engineers
know _exactly_ what queries are going to be run against the database when the time
comes to execute them.  Because changes are planned ahead of time, migration authors
can control precisely how to reach the desired schema.  If we consider a migration as 
a plan to get from state A to state B, oftentimes multiple paths exist, each with a
very different impact on the database. To demonstrate, consider an initial state which
contains a table with two columns:
```sql
CREATE TABLE users (
    id int,
    name varchar(255)
);
```
Suppose our desired state is:
```sql
CREATE TABLE users (
    id int,
    user_name varchar(255)
);
```
There are at least two ways get from the initial to the desired state:
* Drop the `name` column and create a new `user_name` column.
* Alter the name of the `name` column to `user_name`.

Depending on the context, either may be the desired outcome for the developer
planning the change. With versioned migrations, engineers have the ultimate confidence
of what change is going to happen which may not be known ahead of time in a _declarative_
approach.

### Migration Authoring

The downside of the _versioned migration_ approach is, of course, that it puts the 
burden of planning the migration on developers. This requires a certain level 
of expertise that is not always available to every engineer, as we demonstrated
in our example of setting a default value in a SQLite database above.

As part of the Atlas project we advocate for a third combined approach that we call 
"Versioned Migration Authoring". Versioned Migration Authoring is an attempt to combine
the simplicity and expressiveness of the declarative approach with the control and 
explicitness of versioned migrations. 

With versioned migration authoring, users still declare their desired state and use
the Atlas engine to plan a safe migration from the existing to the new state. 
However, instead of coupling planning and execution, plans are instead written 
into normal migration files which can be checked into source control, fine-tuned manually and 
reviewed in regular code review processes.

