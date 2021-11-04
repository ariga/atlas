---
id: sql-hcl
title: SQL Syntax
---

The [sqlspec](https://pkg.go.dev/ariga.io/atlas@master/sql/sqlspec) package defines
the resource types used to describe an SQL database schema. Used with the
[Atlas HCL](ddl#hcl) syntax, it is easy to compose documents describing the desired
state of a SQL database.  

## Resource Types

### Schema

```hcl
schema "default" {
  
}
```

### Table

A `table` describes a table in a SQL database. 

#### Full Example:
```hcl
table "users" {
  schema = "default"
  column "id" {
    type = "int"
  }
  column "name" {
    type = "string"
  }
  column "manager_id" {
    type = "int"
  }
  primary_key {
    columns = [
        table.users.column.id
    ]
  }
  index "idx_name" {
    columns = [
      table.users.column.name
    ]
    unique = true
  }
  foreign_key "manager_fk" {
    columns = [table.users.column.manager_id]
    ref_columns = [table.users.column.id]
    on_delete = "CASCADE"
    on_update = "NO ACTION"
  }
}
```

#### Properties

| Name        | Kind            | Type        | Description                                             |
|-------------|-----------------|-------------|---------------------------------------------------------|
| schema      | attribute       | string      | References the name of the schema containing the table. |
| column      | resource (list) | column      | Describes a column in the table.                        |
| primary_key | resource        | primary_key | Describes the table's primary key.                      |
| foreign_key | resource (list) | foreign_key | Describes the table's foreign keys.                     |
| index       | resource (list) | index       | Describes the table's indexes.                          |

### Column

A column is a child resource of a `table`. 

#### Examples

```hcl

column "name" {
  type = "string"
  null = false
}

column "age" {
  type = "int"
  default = 42
}

column "active" {
  type = "bool"
  default = true
}
```

#### Properties

| Name    | Kind      | Type                     | Description                                                |
|---------|-----------|--------------------------|------------------------------------------------------------|
| null    | attribute | bool                     | Defines whether the column is nullable.                    |
| type    | attribute | string                   | Defines the type of data that can be stored in the column. |
| default | attribute | *schemaspec.LiteralValue | Defines the default value of the column.                   |

#### Virtual Types

Since RDBMS engines vary in their support for different column
types, Atlas supports the notion of "virtual types". To make schemas more
portable, users can select these types for their columns that will be
projected to a database-specific type in runtime. The types are backed by
the [sqlspec.Type](https://pkg.go.dev/ariga.io/atlas@master/sql/sqlspec#Type)
in the Atlas codebase:
```go
const (
	TypeInt     Type = "int"
	TypeInt8    Type = "int8"
	TypeInt16   Type = "int16"
	TypeInt64   Type = "int64"
	TypeUint    Type = "uint"
	TypeUint8   Type = "uint8"
	TypeUint16  Type = "uint16"
	TypeUint64  Type = "uint64"
	TypeString  Type = "string"
	TypeBinary  Type = "binary"
	TypeEnum    Type = "enum"
	TypeBoolean Type = "boolean"
	TypeDecimal Type = "decimal"
	TypeFloat   Type = "float"
	TypeTime    Type = "time"
)
```

##### Integer Types

| Type    | MySQL             | Postgres | SQLite  |
|---------|-------------------|----------|---------|
| int     | INT               | INTEGER  | INTEGER |
| int8    | TINYINT           | X        | INTEGER |
| int16   | SMALLINT          | SMALLINT | INTEGER |
| int64   | BIGINT            | BIGINT   | INTEGER |
| uint    | INT UNSIGNED      | X        | X       |
| uint8   | TINYINT UNSIGNED  | X        | X       |
| uint16  | SMALLINT UNSIGNED | X        | X       |
| uint64  | BIGINT UNSIGNED   | X        | X       |

##### String Types

##### Binary Types

##### Other Types

### Primary Key 

A `primary_key` is a child resource of a `table`, it defines the table's
primary key. 

#### Example 

```hcl
primary_key {
  columns = [table.users.column.id]
}
```

#### Properties

| Name    | Kind      | Type                     | Description                                                    |
|---------|-----------|--------------------------|----------------------------------------------------------------|
| columns | resource  | reference (list)         | A list of references to columns that comprise the primary key. |

### Foreign Key

Foreign keys are child resources of a `table`, they define some columns in the table
as references to columns in other tables. 

#### Example
```hcl
foreign_key "manager_fk" {
  columns = [table.users.column.manager_id]
  ref_columns = [table.users.column.id]
  on_delete = "CASCADE"
  on_update = "NO ACTION"
}
```

#### Properties

| Name        | Kind      | Type                   | Description                               |
|-------------|-----------|------------------------|-------------------------------------------|
| columns     | attribute | reference (list)       | The columns that reference other columns. |
| ref_columns | attribute | reference (list)       | The referenced columns.                   |
| on_update   | attribute | schema.ReferenceOption | Defines what to do on update.             |
| on_delete   | attribute | schema.ReferenceOption | Defines what to do on delete.             |

### Index

Indexes are child resources of a `table`, they define an index on the table.

```hcl
index "idx_name" {
    columns = [
      table.users.column.name
    ]
    unique = true
}
```

| Name      | Kind      | Type                   | Description                                                  |
|-----------|-----------|------------------------|--------------------------------------------------------------|
| columns   | attribute | reference (list)       | The columns that comprise the index.                         |
| unique    | attribute | boolean                | Defines whether a uniqueness constraint is set on the index. |

 
