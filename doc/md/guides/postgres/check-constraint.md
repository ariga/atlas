---
id: check-constraint
title: Check Constraints in PostgreSQL
slug: /guides/postgres/check-constraint
---

### Introduction

In PostgreSQL, check constraints help prevent the insertion of incorrect, inconsistent, or unwanted data into a table. 

In this article, we'll explore the basics of check constraints in PostgreSQL, including how to define and use them, and some best practices for working with them. We will also see how we can manage check constraints in a PostgreSQL schema easily using [Atlas](https://atlasgo.io/), an open-source schema management tool.

### What is the `check` constraint in PostgreSQL?

A check constraint allows you to specify a Boolean expression that must result in `TRUE` in order to `INSERT` each row in a table. This expression can refer to one or more columns in the table, and can be used to enforce various kinds of data constraints, such as requiring a certain range of values or making sure that certain fields are not equal to range of values. The constraint ensures that the inserted or updated value in a column must follow certain criteria.

### When do we need check constraints?

Some examples of when you can use check constraints are:

- A column must be greater than or equal to a specified value.
- A date field must be in the format of MM/DD/YYYY.
- An employee’s joining date must not precede their DOB.
- The value of an employee’s salary must be a positive integer.

### Syntax

```sql
CHECK (expression)
```

The expression defines a list of values that the column can have. 

The condition for an expression can be any valid PostgreSQL expression. If the condition is not met, the insert or update operation will fail.

### How to use check constraint in a PostgreSQL schema

To add a CHECK constraint to an existing table in PostgreSQL, use the `ALTER TABLE` statement:

```sql title=Syntax
ALTER TABLE table_name
ADD CONSTRAINT constraint_name CHECK (expression);
```

```sql title=Example
CREATE TABLE example (
id INT PRIMARY KEY,
value INT
);

ALTER TABLE example
ADD CONSTRAINT positive_value CHECK (value > 0);

INSERT INTO example (id, value) VALUES (1, -1);
-- This will trigger an "ERROR: new row for relation "example" violates check constraint "positive_value"" error
```

Here is an example on how to add CHECK constraint to multiple columns:

```sql title=Example
CREATE TABLE example (
id INT PRIMARY KEY,
value1 INT,
value2 INT,
value3 INT
);

ALTER TABLE example
ADD CONSTRAINT multi_condition CHECK ((value1 > 0) AND (value2 > 0) AND (value3 > 0));

INSERT INTO example (id, value1, value2, value3) VALUES (1, -1, 1, 1);
-- This will trigger an "ERROR: new row for relation "example" violates check constraint "multi_condition"" error
```

To remove a CHECK constraint, you can use the following statements:

```sql title=Syntax
ALTER TABLE table_name
DROP CONSTRAINT constraint_name;
```

```sql title=Example
CREATE TABLE example (
id INT PRIMARY KEY,
value INT,
CONSTRAINT positive_value CHECK (value > 0)
);

ALTER TABLE example
DROP CONSTRAINT positive_value;

INSERT INTO example (id, value) VALUES (1, -1);
-- This will not trigger an error
```

### Things to remember before using check constraints in a PostgreSQL database:

- Performance impact: adding too many check constraints can negatively impact the performance of the database. It's important to balance the need for constraints with the performance impact they may have.
- Adding a check constraint to an existing table with data can cause problems if the existing data does not meet the constraint requirements. Make sure to check the data before adding a check constraint to an existing table.
- A check constraint does not apply to null values. If you want to prevent null values from being in the column, you must add a `NOT NULL` constraint to the column.
Always make sure to use the correct expressions and operators when defining a check constraint.

:::info
PostgreSQL assumes that CHECK constraints' conditions are immutable, meaning they are expected to always give the same result for the same input row. To learn more about using check constraints in a PostgreSQL database, visit the official documentation [here](https://www.postgresql.org/docs/current/ddl-constraints.html#DDL-CONSTRAINTS-CHECK-CONSTRAINTS).
:::

## Managing check constraints is easy with Atlas

Atlas is an open-source project which allows us to manage our database using a simple and easy-to-understand declarative syntax (similar to Terraform), as well as SQL.

:::info
If you are just getting started, install the latest version of Atlas using the guide to [set up Atlas](https://atlasgo.io/getting-started/).
:::

We will first use the `atlas schema inspect` command to get an [HCL](https://atlasgo.io/guides/ddl#hcl) representation of an empty database named `check_constraint` using Atlas:

```console title=Terminal
atlas schema inspect -u "postgres://postgres:pass@localhost:5432/check_constraint?sslmode=disable" > schema.hcl
```

```hcl title=schema.hcl
schema "public" {}
```

There are no tables in our schema yet, so let’s create tables by adding the following table definitions to the schema. To create a check constraint that ensures a column contains only positive integers, you can use the following syntax:

```hcl title=schema.hcl
schema "public" {}

table "users" {
  schema = schema.public
  column "id" {
    null = false
    type = int
  }
  column "value" {
    null = true
    type = int
  }
  check "user_id" {
    expr = "value > 0"
  }
}
```

To create a check constraint that ensures a column contains only values within a certain range, you can use the following syntax:

```hcl title=schema.hcl {18-31}
schema "public" {}

table "users" {
  schema = schema.public
  column "id" {
    null = false
    type = int
  }
  column "value" {
    null = true
    type = int
  }
  check "user_id" {
    expr = "value > 0"
  }
}

table "blog_posts" {
  schema = schema.public
  column "id" {
    null = false
    type = int
  }
  column "value" {
    null = true
    type = int
  }
  check "post_id" {
    expr = "value BETWEEN 1 AND 10"
  }
}
```

To create a check constraint that ensures a column contains a value that matches a specific pattern, you can use the following syntax:

```hcl title=schema.hcl {33-46}
schema "public" {}

table "users" {
  schema = schema.public
  column "id" {
    null = false
    type = int
  }
  column "value" {
    null = true
    type = int
  }
  check "user_id" {
    expr = "value > 0"
  }
}

table "blog_posts" {
  schema = schema.public
  column "id" {
    null = false
    type = int
  }
  column "value" {
    null = true
    type = int
  }
  check "post_id" {
    expr = "value BETWEEN 1 AND 10"
  }
}

table "friends" {
  schema = schema.public
  column "id" {
    null = false
    type = int
  }
  column "email" {
    null = true
    type = varchar(255)
  }
  check "friend_id" {
    expr = "email LIKE '%@%.%'"
  }
}
```

We have added three table definitions in our HCL schema file. Now, it is time to apply the changes to the database `check_constraint`. In order to do that, save the file and apply the schema changes on the database by using the following command:

```console title=Terminal
atlas schema apply --url "postgres://postgres:pass@localhost:5432/check_constraint?sslmode=disable" --to "file://schema.hcl"
```

Atlas generates the necessary SQL statements to add the new tables we defined in our HCL file, to the database schema. Press Enter while the `Apply` option is highlighted to apply the changes:

```console title=Output
-- Planned Changes:
-- Create "users" table
CREATE TABLE "public"."users" ("id" integer NOT NULL, "value" integer NULL, CONSTRAINT "user_id" CHECK (value > 0));
-- Create "blog_posts" table
CREATE TABLE "public"."blog_posts" ("id" integer NOT NULL, "value" integer NULL, CONSTRAINT "post_id" CHECK (value BETWEEN 1 AND 10));
-- Create "friends" table
CREATE TABLE "public"."friends" ("id" integer NOT NULL, "email" character varying(255) NULL, CONSTRAINT "friend_id" CHECK (email LIKE '%@%.%'));
Use the arrow keys to navigate: ↓ ↑ → ← 
? Are you sure?: 
  ▸ Apply
    Abort
```

To verify that our tables were created, run the following command to inspect our PostgreSQL database in SQL format:

```console title=Terminal
atlas schema inspect -u "postgres://postgres:pass@localhost:5432/check_constraint?sslmode=disable" --format "{{ sql . }}"
```

```console title=Output
-- Add new schema named "public"
CREATE SCHEMA IF NOT EXISTS "public";
-- Create "blog_posts" table
CREATE TABLE "public"."blog_posts" ("id" integer NOT NULL, "value" integer NULL, CONSTRAINT "post_id" CHECK ((value >= 1) AND (value <= 10)));
-- Create "friends" table
CREATE TABLE "public"."friends" ("id" integer NOT NULL, "email" character varying(255) NULL, CONSTRAINT "friend_id" CHECK ((email)::text ~~ '%@%.%'::text));
-- Create "users" table
CREATE TABLE "public"."users" ("id" integer NOT NULL, "value" integer NULL, CONSTRAINT "user_id" CHECK (value > 0));
```

Awesome, we can observe that our tables have been created successfully with the check constraints we added!

Alternatively, you can also verify the same from the PostgreSQL terminal with the following command:

```console title=Terminal
\c check_constraint
```
```console title=Output
You are now connected to database "check_constraint" as user "<username>".
```

```console title=Terminal
check_constraint=# \d
```
```console title=Output
           List of relations
 Schema |    Name    | Type  |  Owner   
--------+------------+-------+----------
 public | blog_posts | table | postgres
 public | friends    | table | postgres
 public | users      | table | postgres
(3 rows)
```

### Wrapping up​​

In this guide, we have demonstrated how to configure our columns to accept and store only desired sets of values using the check constraint and manage check constraints in a PostgreSQL schema easily using [Atlas](https://atlasgo.io/), an open-source schema management tool.

## Need More Help?​​
[Join the Atlas Discord Server](https://discord.com/invite/zZ6sWVg6NT) for early access to features and the ability to provide exclusive feedback to improve your database management.

[Sign up](https://www.getrevue.co/profile/ariga) for our newsletter to stay up to date about Atlas and our cloud platform, [Atlas Cloud](https://atlasgo.cloud/).