---
id: gorm
title: Automatic migration planning for GORM
slug: /guides/orms/gorm
---

## TL;DR
* [GORM](https://gorm.io) is an ORM library that's widely used in the Go community.
* [Atlas](https://atlasgo.io) is an open-source tool for inspecting, planning, linting and
  executing schema changes to your database.
* Developers using GORM can use Atlas to automatically plan schema migrations
  for them, based on the desired state of their schema instead of crafting them by hand.

## Automatic migration planning for GORM

GORM is a popular ORM widely used in the Go community. GORM allows users to 
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

The desired schema of your application can be provided to Atlas via an [External Schema Datasource](/atlas-schema/projects#data-source-external_schema)
which is any program that can output a SQL schema definition to stdout.

To use Atlas with GORM, users can utilize the [GORM Atlas Provider](https://github.com/ariga/atlas-provider-gorm),
a small Go program that can be used to load the schema of a GORM project into Atlas.

In this guide, we will show how Atlas can be used to automatically plan schema migrations for
GORM users.

## Prerequisites

* A local [GORM](https://gorm.io) project.

If you don't have a GORM project handy, you can use [go-admin-team/go-admin](https://github.com/go-admin-team/go-admin)
as a starting point:

```
git clone git@github.com:go-admin-team/go-admin.git
```

## Using the Atlas GORM Provider 

In this guide, we will use the [GORM Atlas Provider](https://github.com/ariga/atlas-provider-gorm)
to automatically plan schema migrations for a GORM project.

### Installation

Install Atlas from macOS or Linux by running:
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
    "--dialect", "mysql", // | postgres | sqlite | sqlserver
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

#### Pinning Go dependencies

Next, to prevent the Go Modules system from dropping this dependency from our `go.mod` file, let's
follow its [official recommendation](https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module)
for tracking dependencies of tools and add a file named `tools.go` with the following contents:

```go title="tools.go"
//go:build tools
package main

import _ "ariga.io/atlas-provider-gorm/gormschema"
```
Alternatively, you can simply add a blank import to the `models.go` file we created
above.

Finally, to tidy things up, run:

```text
go mod tidy
```

### Go Program mode

If your GORM models are spread across multiple packages, or do not embed `gorm.Model` or contain `gorm` struct tags,
you can use the provider as a library in your Go program to load your GORM schema into Atlas.

Create a new program named `loader/main.go` with the following contents:

```go
package main

import (
	"io"
    "os"

    "ariga.io/atlas-provider-gorm/gormschema"
    _ "ariga.io/atlas-provider-gorm/recordriver"
    "github.com/<yourorg>/<yourrepo>/path/to/models"
)

func main() {
    stmts, err := gormschema.New("mysql").Load(&models.User{}, &models.Pet{})
    if err != nil {
        fmt.Fprintf(os.Stderr, "failed to load gorm schema: %v\n", err)
        os.Exit(1)
    }
    io.WriteString(os.Stdout, stmts)
}
```
  
Be sure to replace `github.com/<yourorg>/<yourrepo>/path/to/models` with the import path to your GORM models.
In addition, replace the model types (e.g `models.User`) with the types of your GORM models.

Next, in your project directory, create a new file named `atlas.hcl` with the following contents:

```hcl
data "external_schema" "gorm" {
  program = [
    "go",
    "run",
    "-mod=mod",
    "./loader",
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

## Usage

Atlas supports a [versioned migrations](/concepts/declarative-vs-versioned#versioned-migrations)
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

Using the [Standalone mode](#standalone-mode) configuration file for the Atlas GORM Provider,
we can generate a migration file by running this command:

```bash
atlas migrate diff --env gorm 
```

Will generate files similar to this in the `migrations` directory:

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

Amazing, Atlas automatically generated a migration file that will create the `pets` and `users` tables
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

## Conclusion

In this guide we demonstrated how projects using GORM can use Atlas to automatically
plan schema migrations based only on their data model. To learn more about executing
these migrations against your production database, read the documentation for the 
[`migrate apply`](/versioned/apply) command.

Have questions? Feedback? Find our team [on our Discord server](https://discord.gg/zZ6sWVg6NT).
