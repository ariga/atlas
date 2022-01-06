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
  schema = schema.default
  column "id" {
    type = int
  }
  column "name" {
    type = varchar(255)
  }
  column "manager_id" {
    type = int
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

| Name        | Kind            | Type        | Description                                  |
|-------------|-----------------|-------------|----------------------------------------------|
| schema      | attribute       | reference   | References the  schema containing the table. |
| column      | resource (list) | column      | Describes a column in the table.             |
| primary_key | resource        | primary_key | Describes the table's primary key.           |
| foreign_key | resource (list) | foreign_key | Describes the table's foreign keys.          |
| index       | resource (list) | index       | Describes the table's indexes.               |

### Column

A column is a child resource of a `table`. 

#### Examples

```hcl

column "name" {
  type = text
  null = false
}

column "age" {
  type = integer
  default = 42
}

column "active" {
  type = tinyint(1)
  default = true
}
```

#### Properties

| Name    | Kind      | Type                     | Description                                                |
|---------|-----------|--------------------------|------------------------------------------------------------|
| null    | attribute | bool                     | Defines whether the column is nullable.                    |
| type    | attribute | *schemaspec.Type         | Defines the type of data that can be stored in the column. |
| default | attribute | *schemaspec.LiteralValue | Defines the default value of the column.                   |

### Column Types

The SQL dialects supported by Atlas (Postgres, MySQL, MariaDB, and SQLite) vary in
the types they support. At this point, the Atlas DDL does not attempt to abstract the
away the differences between various databases. This means, that the schema documents
are tied to a specific database engine and version. This may change in a future version
of Atlas as we plan to add support for "Virtual Types". This section lists the various
types that are supported in each database.

For a full list of supported column types, [click here](/sql/column_types).

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
