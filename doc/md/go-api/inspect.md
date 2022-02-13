---
title: Inspecting Schemas 
slug: /go-api/inspect
---
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
see how we can use this interface by inspecting a "dummy" SQLite database. Continuing on the example
in the [previous](intro.md) section: 

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