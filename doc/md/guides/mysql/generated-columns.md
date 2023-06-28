---
id: generated-columns
title: Generated Columns in MySQL with Atlas
slug: /guides/mysql/generated-columns
---

**MySQL** is a popular open-source relational database. **Generated columns** are a feature of MySQL that allows you to
define tables with columns whose value is a function of the value stored in other columns; without requiring complex
expressions in `SELECT`, `INSERT` or `UPDATE` queries.

## What are Generated Columns?

Generated columns are columns that contain values calculated by expressions which can be dependent on other columns; in
a similar manner to formulas in a spreadsheet. There are two types of generated columns in MySQL: _Stored_ and _Virtual_.

### Stored Generated Columns

Stored generated columns are stored and evaluated when a row is inserted or updated. As a result, stored generated
columns use disk space in addition to CPU cycles during the execution of `INSERT` and `UPDATE` statements.

### Virtual Generated Columns

Virtual generated columns are not stored, and only evaluated when a row is read
_(after BEFORE [triggers](https://dev.mysql.com/doc/refman/5.7/en/trigger-syntax.html))_.
As a result, virtual generated columns take no storage at the cost of CPU cycles for `SELECT` statements.

### Limitations of Generated Columns

Generated column expressions must be deterministic which means that — given the same input — an expression must always
produce the same output. As a result, generated columns can not be used with stored variables, functions, procedures,
and subqueries; which could cause the output to be non-deterministic. Following this constraint, generated columns can
not be used to generate random values. On the other hand, a generated column may reference any non-generated
column _regardless_ of its position within the table row and any other generated column within the same table row,
as long as those columns are declared before the generated column.

## When to use Generated Columns?

Generated columns should be used whenever you want to create a column with a value that can be directly determined
from the values of other columns in the same row. In simpler words, for data that is dependent on other data. This
saves the developer from complex application code that is prone to errors on `SELECT`, `INSERT` and `UPDATE` statements.
It also ensures that data which must be consistent, stays consistent.

**MySQL Syntax for a Generated Column**

```sql
column_name data_type [GENERATED ALWAYS] AS (expr) [VIRTUAL | STORED]
    [NOT NULL | NULL] [UNIQUE [KEY]] [[PRIMARY] KEY] [COMMENT 'string']
```

### Using Stored Generated Columns

Stored generated columns should be used for data _(in a table)_ that is read more frequently than it is updated.
This saves CPU cycles while reading rows _(via `SELECT`)_. Stored generated columns should also be used when you
want to use the column in the table primary key or use it as a foreign key constraint. Alternatively, use stored
generated columns as a cache for complex conditions that are costly to calculate.

#### Example

The following example declares a stored generated column in a table that stores the base and height of a triangle in the `base` and `height` column, then computes its area in `area` _(when triangles are inserted or updated)_.

```sql
CREATE TABLE triangles (
    base   DOUBLE,
    height DOUBLE,
    area   DOUBLE AS (base * height * 1/2) STORED
);
```

### Using Virtual Generated Columns

Virtual generated columns should be used for computed data _(in a table)_ that is updated more frequently than it is read or computed data that is expensive to store on disk _(via `INSERT` or `UPDATE`)_. Since values are calculated on the fly, virtual generated columns are perfect for table columns that will have a new value for every `SELECT` statement. If you use the _InnoDB Storage Engine_, secondary indexes can be defined on virtual columns.

#### Example

The following example declares a virtual generated column in a table that stores the price and amount of products sold in the `price` and `quantity` column, then computes its `revenue` _(when products are read)_.

```sql
CREATE TABLE products (
    price    DOUBLE,
    quantity INT,
    revenue  DOUBLE AS (price * quantity) VIRTUAL
);
```

## Managing Generated Columns is easy with Atlas

Managing generated columns and database schemas in MySQL is confusing and error-prone. [Atlas](https://atlasgo.io) is an open-source tool that allows you to manage your database using a simple declarative syntax (similar to Terraform). Instead of creating complex SQL statements that break upon schema migration, we will implement generated columns using Atlas.

### Getting started with Atlas

Install the latest version of Atlas using the [Guide to Setting Up Atlas](/cli/getting-started/setting-up).

### Generated Column Syntax in Atlas

Use `as` in a column in a table to declare a MySQL generated column. For examples with other databases, read the [Atlas Generated Columns DDL](/atlas-schema/hcl.mdx#generated-columns).

```hcl
column "name" {
    type = data_type
    as {
        expr = expression
        type = [STORED | VIRTUAL]
    }
}
```

### Implementing Stored Generated Columns with Atlas

The following example declares a stored generated column in a table that stores the lengths of the sides of right-triangles in the `a` and `b` column, then computes the hypotenuse in `c` _(when triangles are inserted or updated)_.

```hcl
table "triangles"  {
    schema  = schema.example
    column "a"  {
        type  = numeric
    }
    column "b"  {
        type  = numeric
    }
    column "hypotenuse"  {
        type  = numeric
        as  {
            expr  = "SQRT(a * a + b * b)"
            type  = STORED
        }
    }
}
```

Guarantee the table is created by applying the schema to the database.

```
atlas schema apply -u "mysql://root:pass@localhost:3306/example" -f atlas.hcl
```

Approve the schema migration plan that Atlas creates for you _(if applicable)_.

```
-- Planned Changes:
-- Create "triangles" table
CREATE TABLE "example"."triangles" ("a" numeric NOT NULL, "b" numeric NOT NULL, "hypotenuse" numeric NOT NULL GENERATED ALWAYS AS (SQRT(a * a + b * b)) STORED)
Use the arrow keys to navigate: ↓ ↑ → ←
? Are you sure?:
> Apply
Abort
```

Insert triangles into the table.

```sql
INSERT INTO triangles (a, b) VALUES (1,1);
INSERT INTO triangles (a, b) VALUES (3,4);
INSERT INTO triangles (a, b) VALUES (6,8);
```

Select all the triangles in the table using `SELECT  *  FROM triangles` to receive a table with the following output.

| a    | b    | c                  |
| :--- | :--- | :----------------- |
| 1    | 1    | 1.4142135623730951 |
| 3    | 4    | 5                  |
| 6    | 8    | 10                 |

### Implementing Virtual Generated Columns with Atlas

The following example declares a virtual generated column in a TABLE that stores the first and last name of a person, and computes the full name of the person _(when people are selected)_.

```hcl
table "people"  {
    schema  = schema.example
    column "first_name"  {
        type  = varchar(255)
    }
    column "last_name"  {
        type  = varchar(255)
    }
    column "full_name"  {
        type  = varchar(255)
        as  {
            expr  = sql("CONCAT(first_name, ' ', last_name)")
            type  = VIRTUAL
        }
    }
}
```

Alternatively, use the default type of generated column _(VIRTUAL in MySQL)_.

```hcl
table "people"  {
    schema  = schema.example
    column "first_name"  {
        type  = varchar(255)
    }
    column "last_name"  {
        type  = varchar(255)
    }
    column "full_name"  {
        type  = varchar(255)
        as  = sql("CONCAT(first_name," ",last_name)")
    }
}
```

Approve the schema migration plan that Atlas creates for you _(if applicable)_.

```
-- Planned Changes:
-- Create "people" table
CREATE TABLE "example"."people" ("first_name" character varying(255) NOT NULL, "last_name" character varying(255) NOT NULL, "full_name" character varying(255) NOT NULL GENERATED ALWAYS AS (first_name + ' ' + last_name) VIRTUAL)
Use the arrow keys to navigate: ↓ ↑ → ←
? Are you sure?:
> Apply
Abort
```

Insert people into the table.

```sql
INSERT INTO people (first_name, last_name) VALUES ("Bob", "Bark");
INSERT INTO people (first_name, last_name) VALUES ("Kat", "Meow");
INSERT INTO people (first_name, last_name) VALUES ("Ty", "Garoar");
```

Select all the people in the table using `SELECT  *  FROM people` to receive a table with the following output.

| first_name | last_name | full_name |
| :--------- | :-------- | :-------- |
| Bob        | Bark      | Bob Bark  |
| Kat        | Meow      | Kat Meow  |
| Ty         | Garoar    | Ty Garoar |

## Need More Help?

[Join the Ariga Discord Server](https://discord.gg/zZ6sWVg6NT) for early access to features and the ability to provide
exclusive feedback that improves your Database Management Tooling.
