---
title: Dev Database
id: dev-database
slug: /concepts/dev-database
---
## Introduction

Atlas uses the concept of "Dev Database" to provide extra safety and correctness to the migration process.
a development database (a twin environment) to validate schemas, simulate migrations and calculate the state of the
migration directory by replaying the historical changes. Let's go over a few examples to explain the benefits of using a
dev/twin database. For a one-time use Atlas can spin up an ephemeral local docker container for you with a special
[docker driver](concepts/url.mdx).

:::info

Atlas cleans up after itself! You can use the same instance of a "Dev Database" for multiple environments, as long 
as they are not accessed concurrently.

:::

## Validation

Suppose we want to the add the following `CHECK` constraint to the table below:

```hcl title="test.hcl" {6-8}
table "t" {
  schema = schema.test
  column "c" {
    type = int
  }
  check "ck" {
    expr = "c <> d"
  }
}
```

After running [`schema apply`](reference.md#atlas-schema-apply), we get the following error because the `CHECK`
constraint is invalid, as column `d` does not exist.

```shell
$ atlas schema apply --url "mysql://root:pass@:3308/test" -f test.hcl
```
```text
-- Planned Changes:
-- Modify "t" table
ALTER TABLE `test`.`t` ADD CONSTRAINT `ck` CHECK (c <> d), DROP COLUMN `c1`, ADD COLUMN `c` int NOT NULL
✔ Apply
Error: modify "t" table: Error 1054: Unknown column 'd' in 'check constraint ck expression'
exit status 1
```

Atlas cannot predict such errors without applying the schema file on the database, because some cases require parsing
and compiling SQL expressions, traverse their AST and validate them. This is already implemented by the database engine.

Migration failures can leave the database in a broken state. Some databases, like MySQL, do not support transactional
migrations due to [implicit COMMIT](https://dev.mysql.com/doc/refman/8.0/en/implicit-commit.html). However, this can be
avoided using the `--dev-url` option. Passing this to `schema apply` will first create and validate the desired state
(the HCL schema file) on temporary named-databases (schemas), and only then continue to `apply` the changes if it passed
successfully.

```shell
$ atlas schema apply --url "mysql://root:pass@:3308/test" -f test.hcl
  --dev-url="mysql://root:pass@:3308/test"
```
```text
Error: create "t" table: Error 3820: Check constraint 'ck' refers to non-existing column 'd'.
exit status 1
```

## Diffing

Atlas adopts the declarative approach for maintaining the schemas desired state, but provides two ways to manage and
apply changes on the database: `schema apply` and `migrate diff`. In both commands, Atlas compares the "current", and the
"desired" states and suggests a migration plan to migrate the "current" state to the "desired" state. For example, the
"current" state can be an inspected database or a migration directory, and the "desired" state can be an inspected
database, or an HCL file.

Schemas that are written in HCL files are defined in natural form by humans. However, databases store schemas in
normal form (also known as canonical form). Therefore, when Atlas compares two different forms it may suggest incorrect
or unnecessary schema changes, and using the `--dev-url` option can solve this (see the above section for more
in-depth example).

Let's see it in action, by adding an index-expression to our schema.

```hcl title="test.hcl" {6-10}
table "t" {
  schema = schema.test
  column "c" {
    type = varchar(32)
  }
  index "i" {
    on {
      expr = "upper(concat('c', c))"
    }
  }
}
```

```shell
$ atlas schema apply --url "mysql://root:pass@:3308/test" -f test.hcl
```
```text
-- Planned Changes:
-- Modify "t" table
ALTER TABLE `test`.`t` ADD INDEX `i` ((upper(concat('c', c))))
✔ Apply
```

We added a new index-expression to our schema, but using `schema inspect` will show our index in its normal form.

```shell
$ atlas schema inspect --url "mysql://root:pass@:3308/test"
```
```hcl {7-11}
table "t" {
  schema = schema.test
  column "c" {
    null = false
    type = varchar(32)
  }
  index "i" {
    on {
      expr = "upper(concat(_utf8mb4'c',`c`))"
    }
  }
}
```

Therefore, running `schema apply` again will suggest unnecessary schema changes.
```shell
$ atlas schema apply --url "mysql://root:pass@:3308/test" -f test.hcl
```
```text
-- Planned Changes:
-- Modify "t" table
ALTER TABLE `test`.`t` DROP INDEX `i`
-- Modify "t" table
ALTER TABLE `test`.`t` ADD INDEX `i` ((upper(concat('c', c))))
✔ Abort
```

Similarly to the previous example, we will use the `--dev-url` option to solve this.

```shell
$ atlas schema apply --url "mysql://root:pass@:3308/test" -f test.hcl
  --dev-url="mysql://root:pass@:3307/test"
```
```text
Schema is synced, no changes to be made
```

Hooray! Our desired schema is synced and no changes have to be made.

:::info

Atlas cleans up after itself! You can use the same instance of a "Dev Database" for multiple environments, as long
as they are not accessed concurrently.

:::
