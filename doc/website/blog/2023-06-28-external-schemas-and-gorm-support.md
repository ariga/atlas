---
title: "Announcing External Schemas and magical GORM support"
authors: rotemtam
tags: [schema, migration, gorm, migration, automatic-planning]
---
:::info TL;DR

You can now import the desired database schema from any ORM or other tool into Atlas, 
and use it to automatically plan migrations for you.

[See an example](#demo-time)

:::

## Introduction

Today, I'm happy to share with you one of the most exciting features we've added to Atlas since its inception:
"External Schemas".

Atlas is a modern tool for managing your database schema. It allows you to inspect, plan, lint and execute schema changes
to your database. It is designed to be used by developers, DBAs and DevOps engineers alike.

## Schema-as-Code

Atlas is built around the concept of database "Schema-as-Code", which means that you define the desired 
schema of your database in a declarative way, and Atlas takes care of planning and executing the necessary
migrations to get your database to the desired state.  The goal of this approach is to let organizations
build a single source of truth for complex data topologies, and to make it easy to collaborate on schema changes.

## Schema Loaders

To achieve this goal, Atlas provides support for "Schema Loaders" which are different mechanisms for loading
the desired state of your database schema into Atlas.  Until today, Atlas supported a few ways to load your schema:
* Using [Atlas DDL](/atlas-schema/sql-resources) - an HCL based configuration language for defining database schemas.
* Using [Plain SQL](/blog/2023/01/05/atlas-v090) - a simple way to define your schema using plain SQL files (CREATE TABLE statements, etc.)
* From an existing database - Atlas can connect to your database and load the schema from it.
* The [Ent](https://entgo.io) ORM - Atlas can load the schema of your Ent project.

Today, we are adding support for "External Schemas", which means that you can now import the desired database schema
from any ORM or other tool into Atlas, and use it to automatically plan migrations and execute them for you.

## How do External Schemas work?

External Schemas are implemented using a new type of [Datasource](/atlas-schema/projects#data-sources)
called `external_schema`.  The `external_schema` data source enables the import of an SQL schema 
from an external program into Atlas' desired state. With this data source, users have the flexibility 
to represent the desired state of the database schema in any language.

To use an `external_schema`, create a file named `atlas.hcl` with the following content:

```hcl 
data "external_schema" "example" {
  program = [
    "echo",
    "create table users (name text)",
  ]
}

env "local" {
  src = data.external_schema.example.url
  dev = "sqlite://file?mode=memory&_fk=1"
}
```

In this dummy example, we use the `echo` command to generate a simple SQL schema.  In a real-world scenario,
you would use a program that understands your ORM or tool of choice to generate the desired schema.  Some ORMs
support this out-of-the-box, such as Laravel's Eloquent's `schema:dump` command, while others require
some simple integrations work to extract the schema from.  

In the next section we will present the [GORM Atlas Provider](https://github.com/ariga/atlas-provider-gorm) and
how it can be used to seamlessly integrate a GORM based project with Atlas.

## Demo Time

[GORM](https://gorm.io) is a popular ORM widely used in the Go community. GORM allows users to
manage their database schemas using its [AutoMigrate](https://gorm.io/docs/migration.html#Auto-Migration)
feature, which is usually sufficient during development and in many simple cases.

However, at some point, teams need more control and decide to employ
the [versioned migrations](/concepts/declarative-vs-versioned#versioned-migrations)
methodology. Once this happens, the responsibility for planning migration scripts and making
sure they are in line with what GORM expects at runtime is moved to developers.

Atlas can automatically plan database schema migrations for developers using GORM.
Atlas plans migrations by calculating the diff between the _current_ state of the database,
and its _desired_ state.

In the context of versioned migrations, the current state can be thought of as the database schema that would have
been created by applying all previous migration scripts.

### Installation

If you haven't already, install Atlas from macOS or Linux by running:
```bash
curl -sSf https://atlasgo.sh | sh
```
See [atlasgo.io](https://atlasgo.io/getting-started#installation) for more installation options.

Install the provider by running:
```bash
go get -u ariga.io/atlas-provider-gorm
``` 

### Standalone vs Go Program mode

The Atlas GORM Provider can be used in two modes:
* **Standalone** - If all of your GORM models exist in a single package, and either embed `gorm.Model` or contain `gorm` struct tags,
  you can use the provider directly to load your GORM schema into Atlas.
* **Go Program** - If your GORM models are spread across multiple packages, or do not embed `gorm.Model` or contain `gorm` struct tags,
  you can use the provider as a library in your Go program to load your GORM schema into Atlas.

### Standalone mode

If all of your GORM models exist in a single package, and either embed `gorm.Model` or contain `gorm` struct tags,
you can use the provider directly to load your GORM schema into Atlas.

In your project directory, create a new file named `atlas.hcl` with the following contents:

```hcl
data "external_schema" "gorm" {
  program = [
    "go",
    "run",
    "-mod=mod",
    "ariga.io/atlas-provider-gorm",
    "load",
    "--path", "./path/to/models",
    "--dialect", "mysql", // | postgres | sqlite
  ]
}

env "gorm" {
  src = data.external_schema.gorm.url
  dev = "docker://mysql/8/dev"
  migration {
    dir = "file://migrations"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}
```

In this example, we use the `go run` command to run the `atlas-provider-gorm` program and load the schema from the
`./path/to/models` directory.  The `atlas-provider-gorm` program will scan the directory for GORM models and generate
the desired schema for them.  The `--dialect` flag is used to specify the database dialect that the schema should be
generated for.  The `atlas-provider-gorm` program supports the following dialects: `mysql`, `postgres`, and `sqlite`.

For the sake of brevity, we will not review the *Go program mode* in this post, but you can find more information about
it in the [GORM Guide](/guides/orms/gorm).

#### External schemas in action

Atlas supports a [versioned migrations](https://atlasgo.io/concepts/declarative-vs-versioned#versioned-migrations)
workflow, where each change to the database is versioned and recorded in a migration file. You can use the
`atlas migrate diff` command to automatically generate a migration file that will migrate the database
from its latest revision to the current GORM schema.

Suppose we have the following GORM models in our `models` package:

```go
package models

import "gorm.io/gorm"

type User struct {  
	gorm.Model
    Name string
    Pets []Pet
}

type Pet struct {
    gorm.Model
    Name   string
    User   User
    UserID uint
}
```
We can now generate a migration file by running this command:

```bash
atlas migrate diff --env gorm 
```

Observe that files similar to this were created in the `migrations` directory:

```
migrations
|-- 20230627123246.sql
`-- atlas.sum

0 directories, 2 files
```

Examining the contents of `20230625161420.sql`:

```sql
-- Create "users" table
CREATE TABLE `users` (
 `id` bigint unsigned NOT NULL AUTO_INCREMENT,
 `created_at` datetime(3) NULL,
 `updated_at` datetime(3) NULL,
 `deleted_at` datetime(3) NULL,
 `name` longtext NULL,
 PRIMARY KEY (`id`),
 INDEX `idx_users_deleted_at` (`deleted_at`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
-- Create "pets" table
CREATE TABLE `pets` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) NULL,
  `updated_at` datetime(3) NULL,
  `deleted_at` datetime(3) NULL,
  `name` longtext NULL,
  `user_id` bigint unsigned NULL,
  PRIMARY KEY (`id`),
  INDEX `fk_users_pets` (`user_id`),
  INDEX `idx_pets_deleted_at` (`deleted_at`),
  CONSTRAINT `fk_users_pets` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;

```

Amazing! Atlas automatically generated a migration file that will create the `pets` and `users` tables
in our database!

Next, alter the `models.Pet` struct to add a `Nickname` field:

```diff
type Pet struct {
	gorm.Model
	Name     string
+	Nickname string
	User     User
	UserID   uint
}
```

Re-run this command:

```shell
atlas migrate diff --env gorm 
```

Observe a new migration file is generated:

```
-- Modify "pets" table
ALTER TABLE `pets` ADD COLUMN `nickname` longtext NULL;
```


### Conclusion

In this post, we have presented External Schemas and how they can be used to automatically generate database schema
directly from your ORM models. We have also demonstrated how to use the GORM Atlas Provider to automatically plan
migrations for your GORM models.

We believe that this is a huge step forward in making Atlas more accessible to developers who are already using
ORMs in their projects. We hope that you will find this feature useful and we look forward to hearing your feedback.

#### How can we make Atlas better?

We would love to hear from you [on our Discord server](https://discord.gg/zZ6sWVg6NT) :heart:.
