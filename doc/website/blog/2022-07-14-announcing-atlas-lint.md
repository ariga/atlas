---
title: Announcing v0.5.0 with Migration Directory Linting
authors: rotemtam
tags: [atlas, lint, ci]
image: https://blog.ariga.io/uploads/images/posts/v0.5.0/atlas-lint.png
---

With the release of [v0.5.0](https://github.com/ariga/atlas/releases/tag/v0.5.0), we are
happy to announce a very significant milestone for the project. While this version includes some
cool features (such as multi-file schemas) and a [swath](https://github.com/ariga/atlas/compare/v0.4.2...v0.5.0)
of incremental improvements and bugfixes, there is one feature that we're particularly
excited about and want to share with you in this post.

As most outages happen directly as a result of a change to a system, Atlas provides users with the means to verify the
safety of planned changes before they happen. The [`sqlcheck`](https://pkg.go.dev/ariga.io/atlas@master/sql/sqlcheck)
package provides interfaces for analyzing the contents of SQL files to generate insights on the safety of many kinds of
changes to database schemas. With this package, developers may define an `Analyzer` that can be used to diagnose the impact
of SQL statements on the target database.

This functionality is exposed to CLI users via the `migrate lint` subcommand. By utilizing
the `sqlcheck` package, Atlas can now check your migration directory for common problems
and issues.

### `atlas migrate lint` in action

Recall that Atlas uses a [dev database](https://atlasgo.io/concepts/dev-database) to plan and
simulate schema changes. Let's start by spinning up a container that will serve as our
dev database:
```text
docker run --name atlas-db-dev -d -p 3307:3306 -e MYSQL_ROOT_PASSWORD=pass  mysql
```

Next let's create `schema.hcl`, the HCL file which will contain the desired state of
our database:

```hcl title=schema.hcl
schema "example" {
}
table "users" {
  schema = schema.example
  column "id" {
    type = int
  }
  column "name" {
    type = varchar(255)
  }
  primary_key {
    columns = [
      column.id
    ]
  }
}
```

To simplify the commands we need to type in this demo, let's create an Atlas
[project file](https://atlasgo.io/atlas-schema/projects) to define a local environment.
```hcl title=atlas.hcl
env "local" {
  src = "./schema.hcl"
  url = "mysql://root:pass@localhost:3306"
  dev = "mysql://root:pass@localhost:3307"
}
```
Next, let's plan the initial migration that creates the `users` table:
```text
atlas migrate diff --env local
```
Observe that the `migrations/` directory was created with an `.sql` file and
a file named `atlas.sum`:

```text
├── atlas.hcl
├── migrations
│   ├── 20220714090139.sql
│   └── atlas.sum
└── schema.hcl
```
This is the contents of our new migration script:
```sql
-- add new schema named "example"
CREATE DATABASE `example`;
-- create "users" table
CREATE TABLE `example`.`users` (`id` int NOT NULL, `name` varchar(255) NOT NULL, PRIMARY KEY (`id`)) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
```
Next, let's make a destructive change to the schema. Destructive changes are
changes to a database schema that result in loss of data, such as dropping a
column or table. Let's remove the `name` name column from our desired schema:
```hcl title=schema.hcl {8}
schema "example" {
}
table "users" {
  schema = schema.example
  column "id" {
    type = int
  }
  // Notice the "name" column is missing.
  primary_key {
    columns = [
      column.id
    ]
  }
}
```
Now, let's plan a migration to this new schema:
```text
atlas migrate diff --env local
```
Observe the new migration which Atlas planned for us:
```sql
-- modify "users" table
ALTER TABLE `example`.`users` DROP COLUMN `name`;
```

Finally, let's use `atlas migrate lint` to analyze this change and verify
it's safety:

```text
atlas migrate lint --env local --latest 1

Destructive changes detected in file 20220714090811.sql:

	L2: Dropping non-virtual column "name"
```
When we run the `lint` command, we need to instruct Atlas on how to decide
what set of migration files to analyze. Currently, two modes are supported.
* `--git-base <branchName>`: which selects the diff between the provided branch
and the current one as the changeset.
* `--latest <n>` which selects the latest `n` migration files as the changeset.

As expected, Atlas analyzed this change and detected a _destructive change_
to our database schema. In addition, Atlas users can analyze the migration
directory to automatically detect:
* Data-dependent changes
* Migration Directory integrity
* Backward-incompatible changes (coming soon)
* Drift between the desired and the migration directory (coming soon)
* .. and more

### Wrapping up

We started Atlas more than a year ago because we felt that the industry deserves
a better way to manage databases. A huge amount of progress has been made as part of the
DevOps movement on the fronts of managing compute, networking and configuration.
So much, in fact, that it always baffled us to see that the database,
the single most critical component of any software system, did not receive this level
of treatment.

Until today, the task of verifying the safety of migration scripts was reserved to
humans (preferably SQL savvy, and highly experienced). We believe that with this milestone
we are beginning to pave a road to a reality where teams can move as quickly and
safely with their databases as they can with their code.


Have questions? Feedback? Find our team [on our Discord server](https://discord.gg/zZ6sWVg6NT).
