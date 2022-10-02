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

For GORM users, the current state can be thought of as the database schema that would have
been created by GORM's [AutoMigrate](https://gorm.io/docs/migration.html#Auto-Migration)
feature, if run on an empty database. 

The desired state can be provided to Atlas via an Atlas schema [HCL file](https://atlasgo.io/atlas-schema/sql-resources) 
or as a connection string to a database that contains the desired schema.

In this guide, we will show how Atlas can automatically plan schema migrations for
GORM users.

## Prerequisites

* An existing project with a GORM.
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

## Create an auto-migration script

To plan a migration to your desired state, we will first create a script that
populates an empty database with your current schema. Suppose you have models
such as:
```go
type User struct {
	gorm.Model
	Name string
}

type Product struct {
	gorm.Model
	Code  string
	Price uint
}
```
In a new directory in your project, create a new file named `main.go`:
```go title=main.go
package main

import (
	"flag"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
        conn := flag.StringVar("conn", "", "connection string to db")
	flag.Parse()
	if *conn == "" {
		log.Fatalln("conn flag required")
	}
	db, err := gorm.Open(mysql.Open(conn), &gorm.Config{})
	if err != nil {
		log.Fatalln(err)
	}
	// Replace `&Product{}, &User{}` with the models of your application.
	if err := db.AutoMigrate(&Product{}, &User{}); err != nil {
		log.Fatalln(err)
	}
}
```

Create a schema named `gorm` in our Dev Database to hold the desired state:
```text
docker exec atlas-db-dev mysql -ppass -e 'drop database if exists gorm; create database gorm'
```

To populate the `gorm` schema with the desired state run:
```text
go run main.go -conn 'root:pass@tcp(localhost:3306)/gorm'
```

## Use Atlas to plan an initial migration

We can now use Atlas's `migrate diff` command to calculate a migration from the 
current state, which can be thought of as the sum of all the migration scripts
in the `migrations` directory (currently an empty schema), to the desired schema
which exists in the Dev Database.

Run:

```text
atlas migrate diff --dir file://migrations --dev-url mysql://root:pass@:3306/dev --to mysql://root:pass@:3306/gorm
```

Observe that two new files were created under the `migrations` directory:

* `20221002070731.sql` (name will vary on your workstation) - a migration file containing SQL to create
  your database schemas:
  ```sql
  -- create "products" table
  CREATE TABLE `products` (`id` bigint unsigned NOT NULL AUTO_INCREMENT, `created_at` datetime(3) NULL, `updated_at` datetime(3) NULL, `deleted_at` datetime(3) NULL, `code` longtext NULL, `price` bigint unsigned NULL, PRIMARY KEY (`id`), INDEX `idx_products_deleted_at` (`deleted_at`)) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
  -- create "users" table
  CREATE TABLE `users` (`id` bigint unsigned NOT NULL AUTO_INCREMENT, `created_at` datetime(3) NULL, `updated_at` datetime(3) NULL, `deleted_at` datetime(3) NULL, `name` longtext NULL, PRIMARY KEY (`id`), INDEX `idx_users_deleted_at` (`deleted_at`)) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
  ```
* `atlas.sum` - To ensure migration history is correct while multiple developers work on the same project
  in parallel Atlas enforces [migration directory integrity](/concepts/migration-directory-integrity)
  using a file name `atlas.sum`. This file was created in your migrations directory
  which contains a hash sum of each file in your directory as well as a total sum:
  ```text
  h1:RrRV3cwyd/K1y74c0tUs04RQ1nRFTkA+g7JRb79PwBU=
  20221002070731.sql h1:je1k1wqknzZ72N2Hmg0MRyuXwHVtg9k7dtoCf33G4Ek=
  ```

:::info Support for other migration tools

By default, Atlas generates migration files in a format that is accepted by Atlas's migration
execution engine using the `migrate apply` ([docs](/versioned/apply)) command. Atlas can also generate migrations for other
popular migration tools such as golang-migrate, Flyway, Liquibase, and more! To learn more about custom formats, read
the docs [here](/versioned/diff#generate-migrations-with-custom-formats).

:::

## Evolving the schema

Next, let's see how we can generate additional migrations when we evolve our schema.
Start by adding an `email` field to our `User` model and add a struct tag telling GORM
we want this field to be unique:

```go
type User struct {
	gorm.Model
	Name  string
	// highlight-next-line-info
	Email string `gorm:"unique"`
}
```

Make sure our `gorm` schema is empty:

```text
docker exec atlas-db-dev mysql -ppass -e 'drop database if exists gorm; create database gorm'
```

Re-populate the schema with the new desired state:

```text
go run main.go -conn 'root:pass@tcp(localhost:3306)/gorm'
```

Use `migrate diff` to plan the next migration:

```text
atlas migrate diff --dir file://migrations --dev-url mysql://root:pass@localhost:3306/dev --to mysql://root:pass@localhost:3306/gorm
```

Observe a new migration file was created in the `migrations` directory:

```sql
-- modify "users" table
ALTER TABLE `users` ADD COLUMN `email` varchar(191) NULL, ADD UNIQUE INDEX `email` (`email`);
```

Amazing! Atlas automatically calculated the difference between our current state (the migrations
directory) and our desired state (our GORM schema) and planned a correct migration.

## Conclusion

In this guide we demonstrated how projects using GORM can use Atlas to automatically
plan schema migrations based only on their data model. To learn more about executing
these migrations against your production database, read the documentation for the 
[`migrate apply`](/versioned/apply) command.

Have questions? Feedback? Find our team [on our Discord server](https://discord.gg/zZ6sWVg6NT).
