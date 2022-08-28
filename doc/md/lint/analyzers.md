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
```
mysql> ALTER TABLE `example`.`orders` ADD UNIQUE INDEX `idx_name` (`name`);
ERROR 1062 (23000): Duplicate entry 'atlas' for key 'orders.idx_name'
```
This type of change is tricky because a developer trying to simulate it locally
might succeed in performing it only to be surprised that their migration script
fails in production, breaking a deployment sequence or causing other unexpected
behavior. Using the `datadepend` ([GoDoc](https://pkg.go.dev/ariga.io/atlas@master/sql/sqlcheck/datadepend)) 
Analyzer, teams can detect this risk early and account for it in pre-deployment checks to a database. 

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
| **MY**                             | MySQL and MariaDB specific checks                                           |
| [MY101](#MY101)                    | Adding a non-nullable column without a `DEFAULT` value to an existing table |


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

#### MY101 {#MY101}

Adding a non-nullable column to a table without a `DEFAULT` value implicitly sets existing rows with the column
zero (default) value. For example:

```sql
ALTER TABLE t ADD COLUMN c int NOT NULL;
// highlight-next-line
-- Append column `c` to all existing rows with the value 0.
```