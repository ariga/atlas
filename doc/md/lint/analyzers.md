---
title: Migration Analyzers
slug: /lint/analyzers
---
The database is often the most critical component in software architectures. Being a stateful component, it cannot be
easily rebuilt, scaled-out or fixed by a restart. Outages that involve damage to data or simply unavailability of the database
are notoriously hard to manage and recover from, often taking long hours of careful work by a team's most senior
engineers.

As most outages happen directly as a result of a change to a system, Atlas provides users with means to verify the
safety of planned changes before they happen. The [`sqlcheck`](https://pkg.go.dev/ariga.io/atlas@master/sql/sqlcheck)
package provides interfaces for analyzing the contents of SQL files to generate insights on the safety of many kinds of
changes to database schemas. With this package developers may define an `Analyzer` that can be used to diagnose the impact
of SQL statements on the target database.

Using these interfaces, Atlas provides different `Analyzer` implementations that are useful for determining the
safety of migration scripts.

## Analyzers

Below are the `Analyzer` implementations currently supported by Atlas.

### Destructive Changes

Destructive changes are changes to a database schema that result in loss of data. For instance,
consider a statement such as:
```sql
ALTER TABLE `users` DROP COLUMN `email_address`;
```
This statement is considered destructive because whatever data is stored in the `email_address` column
will be deleted from disk, with no way to recover it. There are definitely situations where this type
of change is desired, but they are relatively rare. Using the `destructive` ([GoDoc](https://pkg.go.dev/ariga.io/atlas@master/sql/sqlcheck/destructive))
Analyzer, teams can detect this type of change and design workflows that prevent it from happening accidentally.

Running migration linting locally on in CI fails with exit code 1 in case destructive changes are detected. However,
users can disable this by configuring the `destructive` analyzer in the [`atlas.hcl`](../atlas-schema/projects#configure-migration-linting)
file:

```hcl title="atlas.hcl" {2-4}
lint {
  destructive {
    error = false
  }
}
```

### Data-dependent Changes

Data-dependent changes are changes to a database schema that _may_ succeed or fail, depending on the
data that is stored in the database. For instance, consider a statement such as:

```sql
ALTER TABLE `example`.`orders` ADD UNIQUE INDEX `idx_name` (`name`);
```
This statement is considered data-dependent because if the `orders` table
contains duplicate values on the name column we will not be able to add a uniqueness
constraint. Consider we added two records with the name `atlas` to the table:
```
mysql> create table orders ( name varchar(100) );
Query OK, 0 rows affected (0.11 sec)

mysql> insert into orders (name) values ("atlas");
Query OK, 1 row affected (0.06 sec)

mysql> insert into orders (name) values ("atlas");
Query OK, 1 row affected (0.01 sec)
```
Attempting to add a uniqueness constraint on the `name` column, will fail:
```sql
mysql> ALTER TABLE `example`.`orders` ADD UNIQUE INDEX `idx_name` (`name`);
// highlight-next-line-error-message
ERROR 1062 (23000): Duplicate entry 'atlas' for key 'orders.idx_name'
```
This type of change is tricky because a developer trying to simulate it locally
might succeed in performing it only to be surprised that their migration script
fails in production, breaking a deployment sequence or causing other unexpected
behavior. Using the `data_depend` ([GoDoc](https://pkg.go.dev/ariga.io/atlas@master/sql/sqlcheck/datadepend))
Analyzer, teams can detect this risk early and account for it in pre-deployment checks to a database.

By default, data-dependent changes are reported but not cause migration linting to fail. Users can change this by
configuring the `data_depend` analyzer in the [`atlas.hcl`](../atlas-schema/projects#configure-migration-linting) file:

```hcl title="atlas.hcl" {2-4}
lint {
  data_depend {
    error = true
  }
}
```

## Checks

The following schema change checks are provided by Atlas:

| **Check**                          | **Short Description**                                                       |
|------------------------------------|-----------------------------------------------------------------------------|
| [**DS1**](#destructive-changes)    | Destructive changes                                                         |
| [DS101](#DS101)                    | Schema was dropped                                                          |
| [DS102](#DS102)                    | Table was dropped                                                           |
| [DS103](#DS103)                    | Non-virtual column was dropped                                              |
| [**MF1**](#data-dependent-changes) | Changes that might fail                                                     |
| [MF101](#MF101)                    | Add unique index to existing column                                         |
| [MF102](#MF102)                    | Modifying non-unique index to unique                                        |
| [MF103](#MF103)                    | Adding a non-nullable column to an existing table                           |
| [MF104](#MF104)                    | Modifying a nullable column to non-nullable                                 |
| **CD1**                            | Constraint deletion changes                                                 |
| [CD101](#CD101)                    | Foreign-key constraint was dropped                                          |
| **MY**                             | MySQL and MariaDB specific checks                                           |
| [MY101](#MY101)                    | Adding a non-nullable column without a `DEFAULT` value to an existing table |
| [MY102](#MY102)                    | Adding a column with an inline `REFERENCES` clause has no actual effect     |
| **LT**                             | SQLite specific checks                                                      |
| [LT101](#LT101)                    | Modifying a nullable column to non-nullable without a `DEFAULT` value       |
| **AR**                             | Atlas cloud checks                                                          |
| [AR101](#AR101)                    | Creating table with non-optimal data alignment                              |

#### DS101 {#DS101}

Destructive change that is reported when a database schema was dropped. For example:

```sql
DROP SCHEMA test;
```

#### DS102 {#DS102}

Destructive change that is reported when a table schema was dropped. For example:

```sql
DROP TABLE test.t;
```

#### DS103 {#DS103}

Destructive change that is reported when a non-virtual column was dropped. For example:

```sql
ALTER TABLE t DROP COLUMN c;
```

#### MF101 {#MF101}

Adding a unique index to a table might fail in case one of the indexed columns contain duplicate entries. For example:

```sql
CREATE UNIQUE INDEX i ON t(c);
```

#### MF102 {#MF102}

Modifying a non-unique index to be unique might fail in case one of the indexed columns contain duplicate entries.

:::note
Since index modification is done with `DROP` and `CREATE`, this check will be reported only when analyzing changes
programmatically or when working with the [declarative workflow](../concepts/workflows.md#declarative-migrations).
:::

#### MF103 {#MF103}

Adding a non-nullable column to a table might fail in case the table is not empty. For example:

```sql
ALTER TABLE t ADD COLUMN c int NOT NULL;
```

#### MF104 {#MF104}

Modifying nullable column to non-nullable might fail in case it contains NULL values. For example:

```sql
ALTER TABLE t MODIFY COLUMN c int NOT NULL;
```

The solution, in this case, is to backfill `NULL` values with a default value:

```sql {1}
UPDATE t SET c = 0 WHERE c IS NULL;
ALTER TABLE t MODIFY COLUMN c int NOT NULL;
```

#### CD101 {#CD101}

Constraint deletion is reported when a foreign-key constraint was dropped. For example:

```sql
ALTER TABLE pets DROP CONSTRAINT owner_id;
```

#### MY101 {#MY101}

Adding a non-nullable column to a table without a `DEFAULT` value implicitly sets existing rows with the column
zero (default) value. For example:

```sql
ALTER TABLE t ADD COLUMN c int NOT NULL;
// highlight-next-line
-- Append column `c` to all existing rows with the value 0.
```

#### MY102 {#MY102}

Adding a column with an inline `REFERENCES` clause has no actual effect. Users should define a separate `FOREIGN KEY`
specification instead. For example:

```diff
-CREATE TABLE pets (owner_id int REFERENCES users(id));
+CREATE TABLE pets (owner_id int, FOREIGN KEY (owner_id) REFERENCES users(id));
```

#### LT101 {#LT101}

Modifying a nullable column to non-nullable without setting a `DEFAULT` might fail in case it contains `NULL` values.
The solution is one of the following:

1\. Set a `DEFAULT` value on the modified column:

```sql {2}
-- create "new_users" table
CREATE TABLE `new_users` (`a` int NOT NULL DEFAULT 1);
-- copy rows from old table "users" to new temporary table "new_users"
INSERT INTO `new_users` (`a`) SELECT IFNULL(`a`, 1) FROM `users`;
-- drop "users" table after copying rows
DROP TABLE `users`;
-- rename temporary table "new_users" to "users"
ALTER TABLE `new_users` RENAME TO `users`;
```

2\. Backfill `NULL` values with a default value:

```sql {1-2}
-- backfill previous rows
UPDATE `users` SET `a` = 1 WHERE `a` IS NULL;
-- disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- create "new_users" table
CREATE TABLE `new_users` (`a` int NOT NULL);
-- copy rows from old table "users" to new temporary table "new_users"
INSERT INTO `new_users` (`a`) SELECT `a` FROM `users`;
-- drop "users" table after copying rows
DROP TABLE `users`;
-- rename temporary table "new_users" to "users"
ALTER TABLE `new_users` RENAME TO `users`;
-- enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
```

#### AR101 {#AR101}

Creating a table with optimal data alignment may help minimize the amount of required disk space.
For example consider the next Postgres table on a 64-bit system:

```postgresql
CREATE TABLE accounts (
    id bigint PRIMARY KEY,
    premium boolean,
    balance integer,
    age     smallint
);
```
Each tuple in the table takes 24 bytes of successive memory without the header.
the `id` attribute takes 8 bytes, the `premium` takes 1 byte and 3 bytes of padding, the `balance` takes 4 bytes and the `age` takes 2 bytes,
and lastly 6 bytes of padding allocated for the end of the row.
In total 9 bytes of padding are allocated for each row.

Compared to same table with different ordering which only takes 16 bytes in memory with 1 byte of padding:

```postgresql
CREATE TABLE accounts (
    id bigint PRIMARY KEY,
    balance integer,
    age smallint,
    premium boolean
);
```
