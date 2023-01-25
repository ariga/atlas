---
id: check-constraint
title: CHECK Constraint in MySQL
slug: /guides/mysql/check-constraint
---

### Introduction

A `CHECK` constraint is a very useful tool that can be used in MySQL to prevent invalid data from being inserted into a table.

It works by checking the values of certain columns in the table against certain conditions that you specify. If a row of data does not meet the conditions of the `CHECK` constraint, then it is rejected and not inserted into the table.

This can be a very helpful tool for preventing bad data from being inserted into your database, and it can save you a lot of time and hassle later on. In this article, we will show you how to use a `CHECK` constraint in MySQL.

### What is the `CHECK` constraint in MySQL?

MySQL 8.0.16 introduces the `CHECK` constraint, which you can use to protect the data in your tables. A `CHECK` constraint is an integrity constraint used to limit the value range for a column. It ensures that the inserted or updated value in a column must follow certain criteria.

### When do we need `CHECK` constraints?

The following are some examples of when you can use `CHECK` constraints:

- A column must be greater than or equal to a specified value.
- A date field must be in the format of MM/DD/YYYY.
- An employee’s joining date must not precede their DOB.
- The value of an employee’s salary must be a positive integer.

The `CHECK` constraint is commonly used along with the NOT NULL constraint for ensuring that a column contains only valid data and doesn’t contain any NULL values.

## Syntax

```sql
CHECK (expression)
```

The expression defines list of values that the column can have. You can have multiple expressions in a single `CHECK` constraint.

If you want to specify multiple expressions, you need to use the OR keyword between each expression. For example:

```sql
CHECK (expression_1 OR expression_2)
```

You can use the AND keyword between each expression to make the `CHECK` constraint more restrictive. For example:

```sql
CHECK (expression_1 AND expression_2)
```

The condition for an expression can be any valid MySQL expression. If the condition is not met, the insert or update operation will fail.

## Adding and removing the `CHECK` constraint in MySQL

#### To add a `CHECK` constraint to a table in MySQL, use the `ALTER TABLE` statement:

```sql title="Syntax"
ALTER TABLE table_name
ADD CONSTRAINT constraint_name CHECK (expression);
```

```sql title="Example"
CREATE TABLE example (
  id INT PRIMARY KEY,
  value INT
);

ALTER TABLE example
ADD CONSTRAINT positive_value CHECK (value > 0);

INSERT INTO example (id, value) VALUES (1, -1);  
-- This will trigger an "ERROR 3819 (HY000): Check constraint 'positive_value' is violated." error
```

#### You can also use the `MODIFY COLUMN` statement to add a `CHECK` constraint to a column in MySQL. The syntax is as follows:

```sql title="Syntax"
ALTER TABLE table_name
MODIFY COLUMN column_name data_type CHECK (expression);
```

```sql title="Example"
CREATE TABLE example (
  id INT PRIMARY KEY,
  value INT
);

ALTER TABLE example
MODIFY COLUMN value INT CHECK (value > 0);

INSERT INTO example (id, value) VALUES (1, -1);  
-- This will trigger an "ERROR 3819 (HY000): Check constraint 'example_chk_1' is violated." error
```

#### If you want to add a `CHECK` constraint to multiple columns, you can use syntax similar to this:

```sql title="Syntax"
ALTER TABLE table_name
ADD CONSTRAINT constraint_name
CHECK ((c1_expression) AND (c2_expression) AND (c3_expression));
```

```sql title="Example"
CREATE TABLE example (
  id INT PRIMARY KEY,
  value1 INT,
  value2 INT,
  value3 INT
);

ALTER TABLE example
ADD CONSTRAINT multi_condition CHECK ((value1 > 0) AND (value2 > 0) AND (value3 > 0));

INSERT INTO example (id, value1, value2, value3) VALUES (1, -1, 1, 1);  
-- This will trigger an "ERROR 3819 (HY000): Check constraint 'multi_condition' is violated." error
```

#### To remove a `CHECK` constraint, you can use the following statements:

```sql title="Syntax"
ALTER TABLE table_name
DROP CONSTRAINT constraint_name;
```

```sql title="Example"
CREATE TABLE example (
  id INT PRIMARY KEY,
  value INT,
  CHECK (value > 0)
);

ALTER TABLE example
DROP CONSTRAINT value;

INSERT INTO example (id, value) VALUES (1, -1);  
-- This will not trigger an error
```

### Benefits and drawbacks of using a `CHECK` constraint
Using a `CHECK` constraint comes with several benefits. A `CHECK` constraint can enforce the data to be valid according to the `CHECK` condition, protecting from poor data quality. This means that any values that do not comply with the `CHECK` condition fail the insertion or modification processes, thus making it easier to troubleshoot any eventual issues. It is worth noting that this comes at a certain cost in terms of performance, as MySQL needs to validate each value before applying it to the database.

### Common mistakes to avoid when using the `CHECK` constraint

When using the `CHECK` constraint in MySQL, it's important to be aware of a few common mistakes. 

1. Be aware of the impact of `CHECK` constraints on data manipulation: `CHECK` constraints can impact the way you manipulate data in the database. For example, you may need to modify multiple rows in order to satisfy a `CHECK` constraint, or you may need to temporarily disable a `CHECK` constraint in order to perform certain operations.
2. Older MySQL versions (8.0.15 and below) ignore `CHECK` constraints. Thus, one needs to keep in mind that the constraint is neither created nor evaluated even if it has been defined in a table definition while using older versions of MySQL. For example:

```sql title="MySQL 5.7"
CREATE TABLE example (
  id INT PRIMARY KEY,
  value INT,
  CHECK (value > 0)
);

INSERT INTO example (id, value) VALUES (1, -1);  
-- This will not trigger an error, even though the CHECK constraint specifies that value must be greater than 0
```

## Managing `CHECK` constraints is easy with Atlas​

Atlas is an open-source project which allows us to manage our database using a simple and easy-to-understand declarative syntax (similar to Terraform).

:::info
If you are just getting started, install the latest version of Atlas using the guide to [setting up Atlas](https://atlasgo.io/getting-started/).
:::

We will first use the `atlas schema inspect command` to get an [HCL](https://atlasgo.io/guides/ddl#hcl) representation of the table we created earlier (without any indexes) by using the Atlas CLI:

```console title="Terminal"
atlas schema inspect -u "mysql://root:@localhost:3306/check_constraint" > schema.hcl
```
```hcl title="schema.hcl"
schema "check_constraint" {
  charset = "utf8mb4"
  collate = "utf8mb4_0900_ai_ci"
}
```

There are no tables in our schema yet, so let’s create tables by adding the following table definitions to the schema.
To create a `CHECK` constraint that ensures a column contains only positive integers, you can use the following syntax:

```hcl title="schema.hcl"
table "users" {
  schema = schema.check_constraint
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

To create a `CHECK` constraint that ensures a column contains only values within a certain range, you can use the following syntax:

```hcl title="schema.hcl"
table "blog_posts" {
  schema = schema.check_constraint
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

To create a `CHECK` constraint that ensures a column contains a value that matches a specific pattern, you can use the following syntax:

```hcl title="schema.hcl"
table "friends" {
  schema = schema.check_constraint
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

Save the file and apply the schema changes on the database by using the following command:

```console title="Terminal"
atlas schema apply -u "mysql://root:password@localhost:3306/check_constraint" -f schema.hcl --dev-url docker://mysql/8/check_constraint
```

:::note
If you get `Error: pulling image: exit status 1` error, ensure that Docker Desktop is up and running.
:::

Atlas generates the necessary SQL statements to add the new index to the database schema. 

```console title="Output"
-- Planned Changes:
-- Create "users" table
CREATE TABLE `check_constraint`.`users` (`id` int NOT NULL, `value` int NULL, CONSTRAINT `user_id` CHECK (`value` > 0)) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci
-- Create "blog_posts" table
CREATE TABLE `check_constraint`.`blog_posts` (`id` int NOT NULL, `value` int NULL, CONSTRAINT `post_id` CHECK (`value` between 1 and 10)) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci
-- Create "friends" table
CREATE TABLE `check_constraint`.`friends` (`id` int NOT NULL, `email` varchar(255) NULL, CONSTRAINT `friend_id` CHECK (`email` like _utf8mb4'%@%.%')) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci
Use the arrow keys to navigate: ↓ ↑ → ← 
? Are you sure?: 
  ▸ Apply
    Abort
```

From the MySQL terminal, let’s verify that our tables are created successfully:

```sql
SHOW tables;
```
```console title="Output"
+----------------------------+
| Tables_in_check_constraint |
+----------------------------+
| users                      |
| blog_posts                 |
| friends                    |
+----------------------------+
```
```sql
SHOW CREATE TABLE users;
```
```console title="Output"
| users | CREATE TABLE `users` (
  `id` int NOT NULL,
  `value` int DEFAULT NULL,
  CONSTRAINT `user_id` CHECK ((`value` > 0))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci |
```

Amazing! Our tables with `CHECK` constraints were created!

### Wrapping up​
In this guide, we have demonstrated how to configure our columns to accept and store only desired sets of values using the `CHECK` constraint.

## Need More Help?​

[Join the Ariga Discord Server](https://discord.com/invite/zZ6sWVg6NT) for early access to features and the ability to provide exclusive feedback that improves your Database Management Tooling. 

[Sign up](https://atlasnewsletter.substack.com/) to our newsletter to stay up to date about [Atlas](https://atlasgo.io), our OSS database schema management tool, and our cloud platform [Atlas Cloud](https://atlasgo.cloud).
