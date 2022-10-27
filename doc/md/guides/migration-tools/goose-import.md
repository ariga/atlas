---
id: goose-import
title: Importing a Goose project to Atlas
slug: /guides/migration-tools/goose-import
---

## TL;DR
* [`pressly/goose`](https://github.com/pressly/goose) is a popular database migration
  CLI tool and Go library that's widely used in the Go community.
* [Atlas](https://atlasgo.io) is an open-source tool for inspecting, planning, linting and
  executing schema changes to your database.
* Developers using `goose` that want to start managing their database schema using Atlas
  can follow this guide to import an existing project to Atlas. 

## Prerequisites

* [Install Goose](https://github.com/pressly/goose#install)
* An existing project with a `goose` migrations directory.
* Docker
* Atlas ([installation guide](https://atlasgo.io/getting-started/#installation))

## Convert the migration files

The first step in importing a project to Atlas is to convert the existing
migration files from the Goose [SQL format](https://github.com/pressly/goose#sql-migrations)
to the Atlas format. 

To automate this process Atlas supports the `atlas migrate import` command. To read 
more about this command, [read the docs](/versioned/import).

Suppose your migrations are located in a directory named `goose` and you would like to
convert them and store them in a new directory named `atlas`. For this example, let's 
assume we have a simple Goose project with only two files:
```text
.
├── 20221027094633_init.sql
└── 20221027094642_new.sql
```

Run:

```text
atlas migrate import --from file://goose?format=goose --to file://atlas
```

Observe that a new directory named `atlas` was created with 3 files:
```text
.
├── 20221027094633_init.sql
├── 20221027094642_new.sql
└── atlas.sum
```

A few things to note about the new directory:
* When converting the migration files, Atlas ignores the `down` migrations as those are not
  supported by Atlas.
* Atlas created a file named `atlas.sum`. To ensure migration history is correct while multiple developers work on the same project
  in parallel, Atlas enforces [migration directory integrity](/concepts/migration-directory-integrity) using
  this file.
* Comments not directly preceding a SQL statement are dropped as well.

## Set the baseline on the target database

Like many other database schema management tools, Atlas uses a metadata table
on the target database to keep track of which migrations were already applied.
In the case where we start using Atlas on an existing database, we must somehow
inform Atlas that all migrations up to a certain version were already applied.

To illustrate this, let's try to run Atlas's `migrate apply` command on a database
that is currently managed by Goose using the migration directory that we just
converted:

```text
atlas migrate apply --dir file://atlas --url mysql://root:pass@localhost:3306/dev
```
Atlas returns an error:
```text
Error: sql/migrate: connected database is not clean: found table "atlas_schema_revisions" in schema "dev". baseline version or allow-dirty is required
```
To fix this, we use the `--baseline` flag to tell Atlas that the database is already at
a certain version:

```text
atlas migrate apply --dir file://atlas --url mysql://root:pass@localhost:3306/dev  --baseline 20221027094642
```

Atlas reports that there's nothing new to run:

```text
No migration files to execute
```

That's better! Next, let's verify that Atlas is aware of what migrations 
were already applied by using the `migrate status` command:

```text
atlas migrate status --dir file://atlas --url mysql://root:pass@localhost:3306/dev
```
Atlas reports:
```text
Migration Status: OK
  -- Current Version: 20221027094642
  -- Next Version:    Already at latest version
  -- Executed Files:  1
  -- Pending Files:   0
```
Great! We have successfully imported our existing Goose project to Atlas.

## Wrapping up

In this guide we have demonstrated how to take an existing project that is
managed by [`pressly/goose`](https://github.com/pressly/goose), a popular
schema migration tool to be managed by Atlas. 

Have questions? Feedback? Find our team [on our Discord server](https://discord.gg/zZ6sWVg6NT).
