---
title: Programmatic inspection of databases in Go using Atlas 
date: "2022-02-09"
author: Rotem Tamir 
authorURL: "https://github.com/rotemtam"
authorImageURL: "https://s.gravatar.com/avatar/36b3739951a27d2e37251867b7d44b1a?s=80"
authorTwitter: _rtam 
url: /programmatic-inspection-of-databases-in-go-using-atlas/
image: https://release.ariga.io/images/assets/inspector-carbon.png
---
Database inspection is the process of connecting to a database to extract metadata about the way data is structured
inside it. In this post, we will present some use cases for inspecting a database, demonstrate why it is a non-trivial
problem to solve, and finally show how it can be solved using [Atlas](https://atlasgo.io), an open-source package (and
command-line tool) written in [Go](https://go.dev) that we are maintaining at Ariga.

As an infrastructure engineer, I have wished many times to have a simple way to programmatically inspect a database.
Database schema inspection can be useful for many purposes. For instance, you might use it to create visualizations of
data topologies, or use it to find table columns that are no longer in use and can be deprecated. Perhaps you would like
to automatically generate resources from this schema (such as documentation or GraphQL schemas), or to use to locate
fields that might carry personally-identifiable information for compliance purposes. Whatever your use case may be,
having a robust way to get the schema of your database is the foundation for many kinds of infrastructure applications.

When we started working on the core engine for Atlas, we quickly discovered that there wasn't any established tool or
package that could parse the information schema of popular databases and return a data structure representing it. Why is
this the case? After all, most databases provide some command-line tool to perform inspection. For example,
`psql`, the standard CLI for Postgres, supports the `\d` command to describe a table:

```text
postgres=# \d users;
                       Table "public.users"
 Column |          Type          | Collation | Nullable | Default
--------+------------------------+-----------+----------+---------
 id     | integer                |           | not null |
 name   | character varying(255) |           |          |
Indexes:
    "users_pkey" PRIMARY KEY, btree (id)
```

So what makes inspection a non-trivial problem to solve? In this post, I will discuss two aspects that I think are
interesting. The first is the variance in how databases expose schema metadata and the second is the complexity of the
data model that is required to represent a database schema.

#### How databases expose schema metadata

Most of the SQL that we use in day-to-day applications is pretty standard. However, when it comes to exposing schema
metadata, database engines vary greatly in the way they work. The way to retrieve information about things like
available schemas and tables, column types and their default values and many other aspects of the database schema looks
completely different in each database engine. For instance, consider this query
([source](https://github.com/ariga/atlas/blob/2e0886e03c5862c54247f41f906f60d64f9c7eaf/sql/postgres/inspect.go#L728))
which can be used to get the metadata about table columns from a Postgres database:

```sql
SELECT t1.table_name,
       t1.column_name,
       t1.data_type,
       t1.is_nullable,
       t1.column_default,
       t1.character_maximum_length,
       t1.numeric_precision,
       t1.datetime_precision,
       t1.numeric_scale,
       t1.character_set_name,
       t1.collation_name,
       t1.udt_name,
       t1.is_identity,
       t1.identity_start,
       t1.identity_increment,
       t1.identity_generation,
       col_description(to_regclass("table_schema" || '.' || "table_name")::oid, "ordinal_position") AS comment,
       t2.typtype,
       t2.oid
FROM "information_schema"."columns" AS t1
         LEFT JOIN pg_catalog.pg_type AS t2
                   ON t1.udt_name = t2.typname
WHERE table_schema = $1
  AND table_name IN (%s)
ORDER BY t1.table_name, t1.ordinal_position
```

As you can see, while it's definitely possible to get the needed metadata, information about the schema is stored in
multiple tables in a way that isn't particularly well documented, and often requires delving into the actual source code
to understand fully. Here's a query to get similar information from
MySQL ([source](https://github.com/ariga/atlas/blob/2e0886e03c5862c54247f41f906f60d64f9c7eaf/sql/mysql/inspect.go#L631)):

```sql
SELECT `TABLE_NAME`,
       `COLUMN_NAME`,
       `COLUMN_TYPE`,
       `COLUMN_COMMENT`,
       `IS_NULLABLE`,
       `COLUMN_KEY`,
       `COLUMN_DEFAULT`,
       `EXTRA`,
       `CHARACTER_SET_NAME`,
       `COLLATION_NAME`
FROM `INFORMATION_SCHEMA`.`COLUMNS`
WHERE `TABLE_SCHEMA` = ?
  AND `TABLE_NAME` IN (%s)
ORDER BY `ORDINAL_POSITION`
```

While this query is much shorter, you can see that it's completely different from the one we ran to inspect Postgres
column metadata. This demonstrates just one way in inspecting Postgres is difference from inspecting MySQL.

#### Mapping database schemas into a useful data structure

To be a solid foundation for building infrastructure, inspection must produce a useful data structure that can be
traversed and analyzed to provide insights, in other words, a graph representing the data topology. As mentioned
above, such graphs can be used to create ERD (entity-relation diagram) charts, such as the schema visualizations
on the [Atlas Management UI](https://atlasgo.io/ui/intro):

![Schema ERD open](https://atlasgo.io/uploads/images/docs/schema-erd-open.png)

Let's consider some aspects of database schemas that such a data structure should capture:

* Databases are split into logical schemas.
* Schemas contain tables, and may have attributes (such as default collation).
* Tables contain columns, indexes and constraints.
* Columns are complex entities that have types, that may be standard to the database engine (and version) or
  custom data types that are defined by the user. In addition,  Columns may have attributes, such as default 
  values, that may be a literal or an expression (it is important to be  able to discern between `now()` and `"now()"`).
* Indexes contain references to columns of the table they are defined on.
* Foreign Keys contain references to column in other tables, that may reside in other schemas.
* ...and much, much more!

To capture any one of these aspects boils down to figuring out the correct query for the specific database engine you
are working with. To be able to provide developers with a data structure that captures all of it, and to do it well
across different versions of multiple database engines we've learned, is not an easy task. This is a perfect opportunity
for an infrastructure project: a problem that is annoyingly complex to solve and that if solved well, becomes a
foundation for many kinds of applications. This was one of our motivations for
creating [Atlas](https://atlasgo.io) ([GitHub](https://github.com/ariga/atlas)) - an open-source project that we
maintain here at [Ariga](https://ariga.io).

Using Atlas, database schemas can be inspected to product Go structs representing a graph of the database
schema topology. Notice the many cyclic references that make it hard to print (but very ergonomic to travere :-)):

```go
&schema.Realm{
    Schemas: {
        &schema.Schema{
            Name:   "test",
            Tables: {
                &schema.Table{
                    Name:    "users",
                    Schema:  &schema.Schema{(CYCLIC REFERENCE)},
                    Columns: {
                        &schema.Column{
                            Name: "id",
                            Type: &schema.ColumnType{
                                Type: &schema.IntegerType{
                                    T:        "int",
                                    Unsigned: false,
                                },
                                Null: false,
                            },
                        },
                    },
                    PrimaryKey: &schema.Index{
                        Unique: false,
                        Table:  &schema.Table{(CYCLIC REFERENCE)},
                        Attrs:  nil,
                        Parts:  {
                            &schema.IndexPart{
                                SeqNo: 0,
                                Desc:  false,
                                C:     &schema.Column{(CYCLIC REFERENCE)},
                            },
                        },
                    },
                },
                &schema.Table{
                    Name:    "posts",
                    Schema:  &schema.Schema{(CYCLIC REFERENCE)},
                    Columns: {
                        &schema.Column{
                            Name: "id",
                            Type: &schema.ColumnType{
                                Type: &schema.IntegerType{
                                    T:        "int",
                                    Unsigned: false,
                                },
                                Null: false,
                            },
                        },
                        &schema.Column{
                            Name: "author_id",
                            Type: &schema.ColumnType{
                                Type: &schema.IntegerType{
                                    T:        "int",
                                    Unsigned: false,
                                },
                                Null: true,
                            },
                        },
                    },
                    PrimaryKey: &schema.Index{
                        Unique: false,
                        Table:  &schema.Table{(CYCLIC REFERENCE)},
                        Parts:  {
                            &schema.IndexPart{
                                SeqNo: 0,
                                Desc:  false,
                                C:     &schema.Column{(CYCLIC REFERENCE)},
                            },
                        },
                    },
                    ForeignKeys: {
                        &schema.ForeignKey{
                            Symbol:  "owner_id",
                            Table:   &schema.Table{(CYCLIC REFERENCE)},
                            Columns: {
                                &schema.Column{(CYCLIC REFERENCE)},
                            },
                            RefTable:   &schema.Table{(CYCLIC REFERENCE)},
                            RefColumns: {
                                &schema.Column{(CYCLIC REFERENCE)},
                            },
                            OnDelete: "SET NULL",
                        },
                    },
                },
            },
        },
    },
}
```

#### Inspecting databases in Go using Atlas

While Atlas is commonly used as a [CLI tool](https://atlasgo.io/cli/getting-started/setting-up), all of Atlas's
core-engine capabilities are available as a [Go module](https://pkg.go.dev/ariga.io/atlas) that you can use
programmatically. Let's get started with database inspection in Go:

To install Atlas, use:

```shell
go get ariga.io/atlas@master
```

#### Drivers

Atlas currently supports three core capabilities for working with SQL schemas.

* "Inspection" - Connecting to a database and understanding its schema.
* "Plan" - Compares two schemas and produces a set of changes needed to reconcile the target schema to the source
  schema.
* "Apply" - creates concrete set of SQL queries to migrate the target database.

In this post we will dive into the inspection with Atlas. The way inspection is done varies greatly between the
different SQL databases. Atlas currently has four supported drivers:

* MySQL
* MariaDB
* PostgreSQL
* SQLite

Atlas drivers are built on top of the standard library [`database/sql`](https://pkg.go.dev/database/sql)
package. To initialize the different drivers, we need to initialize a `sql.DB` and pass it to the Atlas driver
constructor. For example:

```go
package tutorial

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

#### Inspection

As we mentioned above, inspection is one of Atlas's core capabilities. Consider the `Inspector`
interface in the [sql/schema](https://pkg.go.dev/ariga.io/atlas@master/sql/schema#Inspector)
package:

```go
package schema

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
[SQLite](https://pkg.go.dev/ariga.io/atlas@master/sql/sqlite#Driver)) implements this interface. Let's see how we can
use this interface by inspecting a "dummy" SQLite database. Continuing on the example from above:

```go
package tutorial

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

The full source-code for this example is available in
the [atlas-examples repo](https://github.com/ariga/atlas-examples/blob/fb7fef80ca0ad635f056c40a0a1ea223ccf0a9c0/inspect_test.go#L15)
.

And voila! In this example, we first created a table named "example" by executing a query directly against the database.
Next, we used the driver's `InspectSchema` method to inspect the schema of the table we created. Finally, we made some
assertions on the returned `schema.Table` instance to verify that it was inspected correctly.

#### Inspecting using the CLI

If you don't want to write any code and just want to get a document representing your database schema, you can always
use the Atlas CLI to do it for you. To get
started, [head over to the docs](https://atlasgo.io/cli/getting-started/setting-up).

#### Wrapping up

In this post we presented the Go API of Atlas, which we initially built around our use case of building a new database
migration tool, as part of
the [Operational Data Graph Platform](https://blog.ariga.io/data-access-should-be-an-infrastructure-problem/)
that we are creating here at Ariga. As we mentioned in the beginning of this post, there are a lot of cool things you
can build if you have proper database inspection, which raises the question, what will **you** build with it?

#### Getting involved with Atlas

* Follow the [Getting Started](https://atlasgo.io/cli/getting-started/setting-up) guide.
* Join our [Discord Server](https://discord.gg/zZ6sWVg6NT).
* Follow us [on Twitter](https://twitter.com/ariga_io). 
