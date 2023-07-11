---
id: beego
title: Automatic migration planning for Beego
slug: /guides/orms/beego
---

## TL;DR
* [Beego](https://github.com/beego/beego) is an open-source web framework that's widely used in the Go community.
* [Atlas](https://atlasgo.io) is an open-source tool for inspecting, planning, linting and
  executing schema changes to your database.
* Developers using Beego can use Atlas to automatically plan schema migrations
  for them, based on the desired state of their schema instead of crafting them by hand.

## Automatic migration planning for Beego

Beego is a popular web framework widely used in the Go community. Among many other features,
Beego provides an ORM that allows work with popular databases such as MySQL, PostgreSQL, SQLite3, and more.

Beego allows developers to manage their database schemas using its [syncdb](https://github.com/beego/beedoc/blob/master/en-US/mvc/model/cmd.md#database-schema-generation)
feature, which is usually sufficient during development and in many simple cases. 

However, at some point, teams need more control and decide to employ 
the [versioned migrations](/concepts/declarative-vs-versioned#versioned-migrations) 
methodology. Once this happens, the responsibility for planning migration scripts and making
sure they are in line with what Beego expects at runtime is moved to developers.

Atlas can automatically plan database schema migrations for developers using Beego.
Atlas plans migrations by calculating the diff between the _current_ state of the database,
and its _desired_ state.

In the context of versioned migrations, the current state can be thought of as the database schema that would have
been created by applying all previous migration scripts.

The desired schema of your application can be provided to Atlas via an [External Schema Datasource](/atlas-schema/projects#data-source-external_schema)
which is any program that can output a SQL schema definition to stdout.

To use Atlas with Beego, users can utilize the [Beego Atlas Provider](https://github.com/ariga/atlas-provider-beego),
a small Go program that can be used to load the schema of a Beego project into Atlas.

In this guide, we will show how Atlas can be used to automatically plan schema migrations for
Beego users.

## Prerequisites

* A local [Beego](https://github.com/beego/beego) project - the project have a `go.mod` file describing it.

The Beego Atlas Provider works by creating a temporary Go program, compiling and running
it to extract the schema of your Beego project. Therefore, you will need to have Go installed
on your machine.

## Using the Beego Atlas Provider 

In this guide, we will use the [Beego Atlas Provider](https://github.com/ariga/atlas-provider-beego)
to automatically plan schema migrations for a Beego project.

### Installation

Install Atlas from macOS or Linux by running:
```bash
curl -sSf https://atlasgo.sh | sh
```
See [atlasgo.io](https://atlasgo.io/getting-started#installation) for more installation options.

Install the provider by running:
```bash
go get -u ariga.io/atlas-provider-beego
``` 

### Standalone vs Go Program mode

The Atlas Beego Provider can be used in two modes:
* **Standalone** - If your application contains a package that registers all of its Beego models 
  during initialization (using an `func init()` function), you can use the provider directly to load your Beego schema into Atlas.
* **Go Program** - In other cases, you can use the provider as a library directly in a Go program to load your Beego schema into Atlas.

### Standalone mode

In your project directory, create a new file named `atlas.hcl` with the following contents:

```hcl
data "external_schema" "beego" {
  program = [
    "go",
    "run",
    "-mod=mod",
    "ariga.io/atlas-provider-beego",
    "load",
    "--path", "./path/to/models",
    "--dialect", "mysql", // | postgres | sqlite
  ]
}

env "beego" {
  src = data.external_schema.beego.url
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

Be sure to replace `./path/to/models` with the path to the package that contains the registration
of your Beego models. It should look something like this:

```go
func init() {
    orm.RegisterModel(new(User), ...)
}
```

### Go Program mode

In other cases, you can use the provider as a library directly in a Go program to load your Beego schema into Atlas.
This can happen, for example, if your models are registered in multiple packages, or if you want to use the provider
as part of a larger program.

Create a new program named `loader/main.go` with the following contents:

```go
package main

import (
	"io"
    "os"
	
    "ariga.io/atlas-provider-beego/beegoschema"
    "github.com/<yourorg>/<yourrepo>/path/to/models"
    "github.com/beego/beego/v2/client/orm"
)

func main() {
    stmts, err := beegoschema.New("mysql").Load()
    if err != nil {
        fmt.Fprintf(os.Stderr, "failed to load beego schema: %v\n", err)
        os.Exit(1)
    }
    io.WriteString(os.Stdout, stmts)
}

func init() {
    orm.RegisterModel(new(models.HotdogType), new(models.Stand), new(models.HotdogStock))
}
```
  
Be sure to replace `github.com/<yourorg>/<yourrepo>/path/to/models` with the import path to your beego models.
In addition, replace the model types (e.g `models.HotdogType`) with the types of your beego models.

Next, in your project directory, create a new file named `atlas.hcl` with the following contents:

```hcl
data "external_schema" "beego" {
  program = [
    "go",
    "run",
    "-mod=mod",
    "./loader",
  ]
}

env "beego" {
  src = data.external_schema.beego.url
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
from its latest revision to the current application schema.

Suppose we have the following Beego models in our `models` package:

```go
package models

import "github.com/beego/beego/v2/client/orm"

type HotdogType struct {
	Id          int            `orm:"auto;pk"`
	Name        string         `orm:"unique"`
	Description string         `orm:"type(text)"`
	Price       float64        `orm:"digits(10);decimals(2);index"`
	Inventory   []*HotdogStock `orm:"reverse(many)"`
}

type Stand struct {
	Id          int            `orm:"auto;pk"`
	Name        string         `orm:"unique;index"`
	Address     string         `orm:"type(text)"`
	Description string         `orm:"type(text)"`
	Inventory   []*HotdogStock `orm:"reverse(many)"`
}

type HotdogStock struct {
	Id       int         `orm:"auto;pk"`
	Quantity int         `orm:"default(0)"`
	Hotdog   *HotdogType `orm:"rel(fk);on_delete(cascade);index"`
	Stand    *Stand      `orm:"rel(fk);on_delete(cascade);index"`
}

func init() {
  orm.RegisterModel(new(HotdogType), new(Stand), new(HotdogStock))
}
```

Using the [Standalone mode](#standalone-mode) configuration file for the provider,
we can generate a migration file by running this command:

```bash
atlas migrate diff --env beego 
```

Running this command will generate files similar to this in the `migrations` directory:

```
migrations
|-- 20230627123246.sql
`-- atlas.sum

0 directories, 2 files
```

Examining the contents of `20230625161420.sql`:

```sql
-- Create "hotdog_stock" table
CREATE TABLE `hotdog_stock` (
  `id` int NOT NULL AUTO_INCREMENT,
  `quantity` int NOT NULL DEFAULT 0,
  `hotdog_id` int NOT NULL,
  `stand_id` int NOT NULL,
  PRIMARY KEY (`id`),
  INDEX `hotdog_stock_hotdog_id` (`hotdog_id`),
  INDEX `hotdog_stock_stand_id` (`stand_id`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
-- Create "hotdog_type" table
CREATE TABLE `hotdog_type` (
  `id` int NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL DEFAULT "",
  `description` longtext NOT NULL,
  `price` decimal(10,2) NOT NULL DEFAULT 0.00,
  PRIMARY KEY (`id`),
  INDEX `hotdog_type_price` (`price`),
  UNIQUE INDEX `name` (`name`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
-- Create "stand" table
CREATE TABLE `stand` (
  `id` int NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL DEFAULT "",
  `address` longtext NOT NULL,
  `description` longtext NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `name` (`name`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
```

Amazing, Atlas automatically generated a migration file that will create the `hotdog_stock`,
`hotdog_type`, and `stand` tables in our database!

Next, alter the `models.HotdogType` struct to modify the type of the `Price` field and 
drop the index on it:

```diff
type HotdogType struct {
    Id          int            `orm:"auto;pk"`
    Name        string         `orm:"unique"`
    Description string         `orm:"type(text)"`
-   Price       float64        `orm:"digits(10);decimals(2);index"`
+   Price       float64
    Inventory   []*HotdogStock `orm:"reverse(many)"`
}
```

Re-run this command: 

```shell
atlas migrate diff --env beego 
```

Observe a new migration file is generated:

```
-- Modify "hotdog_type" table
ALTER TABLE `hotdog_type` MODIFY COLUMN `price` double NOT NULL DEFAULT 0, DROP INDEX `hotdog_type_price`;
```

## Conclusion

In this guide we demonstrated how projects using Beego can use Atlas to automatically
plan schema migrations based only on their data model. To learn more about executing
these migrations against your production database, read the documentation for the 
[`migrate apply`](/versioned/apply) command.

Have questions? Feedback? Find our team [on our Discord server](https://discord.gg/zZ6sWVg6NT).
