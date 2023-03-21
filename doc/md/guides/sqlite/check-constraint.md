---
id: check-constraint
title: CHECK Constraints in SQLite
slug: /guides/sqlite/check-constraint
---

### Introduction

CHECK constraints are critical for maintaining data integrity in SQLite. By using CHECK constraint, we can define rules for valid data entry and prevent the insertion of incorrect, inconsistent, or unwanted data into a table. 

In this article, we'll explore the basics of CHECK constraints in SQLite, including how to define and use them, and some best practices for working with them. We will also see how we can manage CHECK constraints in a SQLite schema easily using [Atlas](https://atlasgo.io/), an open-source schema management tool.

### What is the `CHECK` constraint in SQLite?

A CHECK constraint allows you to specify a boolean expression that must result in `TRUE` in order to `INSERT` a row in a table. This expression can refer to one or more columns in the table, and can be used to enforce various kinds of data constraints, such as requiring a certain range of values or making sure that certain fields are not null. The constraint ensures that the inserted or updated values in a column must follow certain criteria.

### When do we need `CHECK` constraints?

Some examples of when you can use CHECK constraints are:

- A column must be greater than or equal to a specified value.
- Name should not exceed 20 characters
- An employee’s joining date must not precede their DOB.
- The value of an employee’s salary must be a positive integer.

### Syntax

SQLite users can define CHECK constraints either at the column level or the table level by including CHECK constraints in the table definition.

Usage:
```sql
CHECK (expression)
```

The condition for an expression can be any valid SQLite expression. If the condition is not met, the insert or update operation will fail.

### How to use `CHECK` constraints in a SQLite schema

To add a CHECK constraint to a table in SQLite:

```sql title=Syntax
CREATE TABLE table_name (
   column_name data_type CHECK (expression)
);
```

```sql title=Example
CREATE TABLE example (
id INTEGER PRIMARY KEY,
value INTEGER CHECK (value > 0)
);

INSERT INTO example (id, value) VALUES (1, -1);
-- This will trigger an "CHECK constraint failed" error
```

Here is an example on how to add CHECK constraint to multiple columns:

```sql title=Example
-- Create the example table with the specified CHECK constraints
CREATE TABLE example (
   id INTEGER PRIMARY KEY,
   age INTEGER,
   gender TEXT,
   CHECK (age >= 18 OR age IS NULL),
   CHECK (gender IN ('Male', 'Female', 'Other'))
);

-- Insert data that satisfies the CHECK constraints
INSERT INTO example (id, age, gender) VALUES (1, 25, 'Male');
INSERT INTO example (id, age, gender) VALUES (2, NULL, 'Other');

-- Try to insert data that violates the CHECK constraints
INSERT INTO example (id, age, gender) VALUES (3, 16, 'Male'); 
-- Runtime error: CHECK constraint failed: age >= 18 OR age IS NULL (19)
INSERT INTO example (id, age, gender) VALUES (4, 20, 'Non-binary'); 
-- Runtime error: CHECK constraint failed: gender IN ('Male', 'Female', 'Other') (19)
```

:::info
In SQLite, you can’t remove a CHECK constraint from a table. The only way to remove it is to drop and recreate the table without the constraint. You can temporarily disable CHECK constraints with `PRAGMA ignore_check_constraints = TRUE;`.

However, Dropping constraints is easy with Atlas. We’ll demonstrate this in the next section.
:::

### Things to remember before using `CHECK` constraints in a SQLite database:

- Performance impact: adding too many check constraints can negatively impact the performance of the database. It's important to balance the need for constraints with the performance impact they may have.
- Always make sure to use the correct expressions and operators when defining a check constraint.

:::info
CHECK constraints are only verified when the table is written, not when it is read. To learn more about attributes specified with `CREATE TABLE` command, visit the official documentation [here](https://www.sqlite.org/lang_createtable.html).
:::

## Managing `CHECK` constraints is easy with Atlas

Atlas is an open-source project which allows us to manage our database using a simple and easy-to-understand declarative syntax (similar to Terraform), as well as SQL.

:::info
If you are just getting started, install the latest version of Atlas using the guide to [set up Atlas](https://atlasgo.io/getting-started/).
:::

### Example: Easiest way to drop `CHECK` constraint from a table in SQLite

As of SQLite version 3.41.0, there is no built-in command to drop a CHECK constraint. Hence, it is not possible to remove a CHECK constraint from a table. Once a CHECK constraint has been added to a table, it cannot be dropped or altered. The only way to remove a CHECK constraint is to drop the entire table and recreate it without the constraint.

However, this is a complex operation and has chances of human error. In this example, we will show you how [Atlas](https://atlasgo.io/) magically does that for you.

Let’s begin by creating an example database that has a table with CHECK constraints in it with the following command:

```sql title=sqlite3
.open example.db

CREATE TABLE example (
   id INTEGER PRIMARY KEY,
   age INTEGER CHECK(age >= 18 OR age IS NULL),
   gender TEXT CHECK(gender IN ('Male', 'Female', 'Other'))
);
```

We will first use the `atlas schema inspect` command to get an [HCL](https://atlasgo.io/guides/ddl#hcl) representation of the `example` database using Atlas:

```console
atlas schema inspect --url "sqlite://example.db" > schema.hcl
```

```hcl title=schema.hcl {18-23}
table "example" {
  schema = schema.main
  column "id" {
    null = true
    type = integer
  }
  column "age" {
    null = true
    type = integer
  }
  column "gender" {
    null = true
    type = text
  }
  primary_key {
    columns = [column.id]
  }
  check {
    expr = "(age >= 18 OR age IS NULL)"
  }
  check {
    expr = "(gender IN ('Male', 'Female', 'Other'))"
  }
}
schema "main" {
}
```

Atlas presents our database in an easily readable HCL format. We can now remove a check constraint from our table simply, just by deleting the corresponding constraint definition from the HCL file.

```hcl title=schema.hcl
table "example" {
  schema = schema.main
  column "id" {
    null = true
    type = integer
  }
  column "age" {
    null = true
    type = integer
  }
  column "gender" {
    null = true
    type = text
  }
  primary_key {
    columns = [column.id]
  }
}
schema "main" {
}
```

After removing both the constraint definitions from the table and saving the changes, let’s apply the changes to the database by using `atlas schema apply` command:

```console
atlas schema apply --url "sqlite://example.db" --to "file://schema.hcl"
```

Dropping check constraints is not a straightforward process and requires complex workarounds. However, the Atlas simplified this process for us by automatically planning the removal of check constraints.

You can review the planned changes, and then press Enter while the `Apply` option is highlighted in order to apply the changes.

```console title=Output
-- Planned Changes:
-- Disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- Create "new_example" table
CREATE TABLE `new_example` (`id` integer NULL, `age` integer NULL, `gender` text NULL, PRIMARY KEY (`id`));
-- Copy rows from old table "example" to new temporary table "new_example"
INSERT INTO `new_example` (`id`, `age`, `gender`) SELECT `id`, `age`, `gender` FROM `example`;
-- Drop "example" table after copying rows
DROP TABLE `example`;
-- Rename temporary table "new_example" to "example"
ALTER TABLE `new_example` RENAME TO `example`;
-- Enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
Use the arrow keys to navigate: ↓ ↑ → ← 
? Are you sure?: 
  ▸ Apply
    Abort
```

To verify that our tables were created, run the following command to inspect our SQLite  database in [SQL](https://atlasgo.io/declarative/inspect#sql-format) format:

```console title=terminal
atlas schema inspect -u "sqlite://example.db" --format "{{ sql . }}"
```
```sql title=output
-- Create "example" table
CREATE TABLE `example` (`id` integer NULL, `age` integer NULL, `gender` text NULL, PRIMARY KEY (`id`));
```

Awesome, we can observe that the check constraints have been removed!

Alternatively, you can also verify the same from the SQLite terminal with the following command:

```sql title=sqlite3
.schema example
```

```sql title=output
CREATE TABLE IF NOT EXISTS "example" (`id` integer NULL, `age` integer NULL, `gender` text NULL, PRIMARY KEY (`id`));
```

### Wrapping up​​

In this guide, we have demonstrated how to configure our columns to accept and store only desired sets of values using the check constraint and manage check constraints in a SQLite schema easily using [Atlas](https://atlasgo.io/), an open-source schema management tool.

## Need More Help?​​

[Join the Atlas Discord Server](https://discord.com/invite/zZ6sWVg6NT) for early access to features and the ability to provide exclusive feedback to improve your database management.

[Sign up](https://www.getrevue.co/profile/ariga) for our newsletter to stay up to date about Atlas and our cloud platform, [Atlas Cloud](https://atlasgo.cloud/).