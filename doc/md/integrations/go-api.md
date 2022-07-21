---
title: Go API
id: go-api
slug: /integrations/go-api
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

## Inspection

Inspection is the one of Atlas's core capabilities. When we say "inspection" in the context of this project we mean the
process of connecting to an existing database, querying its metadata tables to understand the structure of the
different tables, types, extensions, etc.

Databases vary greatly in the API they provide users to understand a specific database's schema, but Atlas goes to great
lengths to abstract these differences and provide a unified API for inspecting databases. Consider the `Inspector`
interface in the [sql/schema](https://pkg.go.dev/ariga.io/atlas@v0.3.2/sql/schema#Inspector)
package:

```go
// Inspector is the interface implemented by the different database
// drivers for inspecting multiple tables.
type Inspector interface {
    // InspectSchema returns the schema description by its name. An empty name means the
    // "attached schema" (e.g. SCHEMA() in MySQL or CURRENT_SCHEMA() in PostgreSQL).
    // A NotExistError error is returned if the schema does not exists in the database.
    InspectSchema(ctx context.Context, name string, opts *InspectOptions) (*Schema, error)

    // InspectRealm returns the description of the connected database.
    InspectRealm(ctx context.Context, opts *InspectRealmOption) (*Realm, error)
}
```

As you can see, the `Inspector` interface provides methods for inspecting on different levels:

* `InspectSchema` - provides inspection capabilities for a single schema within a database server.
* `InspectRealm` - inspects the entire connected database server.

Each database driver (for example [MySQL](https://pkg.go.dev/ariga.io/atlas@master/sql/mysql#Driver),
[Postgres](https://pkg.go.dev/ariga.io/atlas@master/sql/postgres#Driver) or
[SQLite](https://pkg.go.dev/ariga.io/atlas@master/sql/sqlite#Driver)) implements this interface. Let's
see how we can use this interface by inspecting a "dummy" SQLite database.

```go
func TestInspect(t *testing.T) {
    // ... skipping driver creation
    ctx := context.Background()
    // Create an "example" table for Atlas to inspect.
    _, err = db.ExecContext(ctx, "create table example ( id int not null );")
    if err != nil {
        log.Fatalf("failed creating example table: %s", err)
    }
    // Open an atlas driver.
    driver, err := sqlite.Open(db)
    if err != nil {
        log.Fatalf("failed opening atlas driver: %s", err)
    }
    // Inspect the created table.
    sch, err := driver.InspectSchema(ctx, "main", &schema.InspectOptions{
        Tables: []string{"example"},
    })
    if err != nil {
        log.Fatalf("failed inspecting schema: %s", err)
    }
    tbl, ok := sch.Table("example")
    require.True(t, ok, "expected to find example table")
    require.EqualValues(t, "example", tbl.Name)
    id, ok := tbl.Column("id")
    require.True(t, ok, "expected to find id column")
    require.EqualValues(t, &schema.ColumnType{
        Type: &schema.IntegerType{T: "int"}, // An integer type, specifically "int".
        Null: false,                         // The column has NOT NULL set.
        Raw:  "INT",                         // The raw type inspected from the DB.
    }, id.Type)
}
```

In this example, we first created a table named "example" by executing a query directly
against the database. We next used the driver's `InspectSchema` method to inspect the schema
of the table we created. Finally, we made some assertions on the returned `schema.Table` instance
to verify that it was inspected correctly.
