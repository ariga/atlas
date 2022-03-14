---
title: Introduction 
slug: /go-api/intro
---
In addition to using Atlas as a CLI tool, all of Atlas's core-engine capabilities are available as
a [Go module](https://pkg.go.dev/ariga.io/atlas) that you can use programmatically. This guide provides high-level
documentation on how to use Atlas from within Go programs.

## Installation

To install Atlas, use:

```shell
go get ariga.io/atlas@latest
```

This installs the latest release of Atlas. If you would like to get the most recent version from the `master` branch,
use:

```shell
go get ariga.io/atlas@master
```

## Drivers

Atlas currently supports three core capabilities for working with SQL schemas.

* "Inspection" - Connecting to a database and understanding its schema.
* "Diff" - Compares two schemas and producing a set of changes needed to reconcile the target schema to the source
  schema.
* "Apply" - creates concrete set of SQL queries to migrate the target database.

The implementation details for these capabilities vary greatly between the different SQL databases. Atlas currently has
three supported drivers:

* MySQL (+MariaDB, TiDB)
* PostgreSQL
* SQLite

Atlas drivers build on top of the standard library [`database/sql`](https://pkg.go.dev/database/sql)
package. To initialize the different drivers, we need to initialize a `sql.DB` and pass it to the Atlas driver
constructor. For example:

```go
package main

import (
	"database/sql"
	"log"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlite"
)

func Test(t *testing.T) {
	// Open a "connection" to sqlite.
	db, err := sql.Open("sqlite3", "file:example.db?cache=shared&_fk=1&mode=memory")
	if err != nil {
		log.Fatalf("failed opening db: %s", err)
	}
	// Open an atlas driver.
	driver, err := sqlite.Open(db)
	if err != nil {
		log.Fatalf("failed opening atlas driver: %s", err)
	}
	// ... do stuff with the driver
}
```
