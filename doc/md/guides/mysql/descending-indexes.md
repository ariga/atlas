---
id: descending-indexes
title: Descending Indexes in MySQL
slug: /guides/mysql/descending-indexes
---

### What are the descending indexes?
In general, indexes with ascending or descending order help increase the performance of queries with the “ORDER BY” clause. Descending indexes are indexes where key values are stored in descending order.

#### When are descending indexes helpful?
In versions prior to MySQL 8.0, scanning an index in reverse order had a very high cost, which resulted in reduced performance for certain queries.
Since the release of MySQL version 8.0, users can now create descending indexes, which can be scanned in forward order, thus increasing efficiency of scanning for certain queries with the ORDER BY clause.

### Syntax
Here is how you can define descending indexes by using DESC in a table definition:

```sql
CREATE TABLE t (
  c1 INT, c2 INT,
  INDEX idx1 (c1, c2 DESC),
  INDEX idx2 (c1 DESC, c2),
  INDEX idx3 (c1 DESC, c2 DESC)
);
```
Or, you can add a descending index to an existing table with the following syntax:
```sql
CREATE INDEX
    index_name_idx
ON
    table_name(column_name DESC);
```
:::info
ASC or DESC specifier is used to specify whether index values are stored in ascending or descending order.

By default, the index values are stored in ascending order if no specifier is given (e.g. Column `c1` in the index `idx1` and column `c2` in the index `idx2` from the table definition above)`
:::

### Example
Let’s create a table which represents data of an ISP’s subscribers along with their email addresses and broadband data usage with the following command:

```sql
CREATE TABLE `telecom_data` (
  `id` bigint(8) unsigned NOT NULL auto_increment,
  `email_address` varchar(255) default NULL,
  `user_name` varchar(255) default NULL,
  `megabytes_used` bigint default NULL,
  PRIMARY KEY (`id`)
) AUTO_INCREMENT=1;
```
Here is how a portion of the table might look like after inserting values:

```sql
SELECT * FROM telecom_data
```
```console title="Output"
+----+---------------------------------------+--------------------+----------------+
| id | email_address                         | user_name          | megabytes_used |
+----+---------------------------------------+--------------------+----------------+
|  1 | erat.Etiam.vestibulum@inhendrerit.edu | Chancellor Puckett |           1359 |
|  2 | Cras.eu@nectempusmauris.edu           | Joel Nunez         |          83495 |
|  3 | Etiam@eleifendvitae.com               | Rogan Wright       |          16104 |
|  4 | gravida.mauris@ac.org                 | William Conrad     |          71934 |
|  5 | convallis@Nullatincidunt.ca           | Magee Ayers        |          62180 |
|  6 | tortor.Nunc@neque.edu                 | Demetrius Hanson   |          94045 |
|  7 | Curabitur@luctusaliquet.ca            | Adrian Mccall      |           3337 |
|  8 | purus@Phasellusdolor.org              | Evan Moreno        |          88898 |
|  9 | mattis.semper.dui@magnisdis.edu       | Theodore Russo     |            374 |
| 10 | enim@dolordolortempus.net             | Walter Knight      |          44358 |
.
.
.
| 98452 | volutpat.Nulla@tempor.ca                    | Jared Gardner  |          47607 |
| 98451 | id.erat@temporaugueac.edu                   | Lars Moreno    |          93351 |
| 98450 | est.tempor.bibendum@enimcondimentumeget.net | Len Harmon     |          71194 |
| 98449 | dis@idmagna.ca                              | Dylan Foreman  |          20388 |
| 98448 | eu.augue.porttitor@Aliquamerat.edu          | Samuel Lott    |          24077 |
| 98447 | erat@dolorNullasemper.edu                   | Graiden Peters |           7781 |
| 98446 | tincidunt.congue@nuncinterdum.org           | Ulric Atkins   |          31194 |
| 98445 | a@utnisia.co.uk                             | Cedric Larson  |          35201 |
| 98444 | gravida.non.sollicitudin@Curabitur.edu      | Cain Odonnell  |          85089 |
| 98443 | turpis@enimEtiam.org                        | Kasper Payne   |          72018 |
+-------+---------------------------------------------+----------------+----------------+
```
We do not have any indexes other than the primary index on the `id` column. Now, suppose we want information about all subscribers who have used less than 100 MB data, but in descending order. Let's query that data with the following command:

```sql
SELECT * FROM telecom_data WHERE megabytes_used < 100 ORDER BY megabytes_used DESC;
```
```console title="Output"
+-------+------------------------------------------------+-------------------+----------------+
| id    | email_address                                  | user_name         | megabytes_used |
+-------+------------------------------------------------+-------------------+----------------+
| 27383 | augue.Sed@malesuadaiderat.edu                  | Marsden Hurst     |             98 |
| 66371 | sem.elit@nislsem.ca                            | Finn Hayden       |             98 |
| 14929 | Nam.nulla.magna@aliquet.edu                    | Merritt Potts     |             97 |
| 18789 | aliquet.diam.Sed@acurnaUt.ca                   | Camden Cash       |             97 |
| 43361 | tincidunt.pede.ac@molestie.ca                  | Kareem Bond       |             96 |
| 85812 | aliquet@id.org                                 | Nash Crawford     |             90 |
| 86340 | ac@magnaPhasellus.ca                           | Cedric Patrick    |             89 |
| 93004 | Integer.tincidunt.aliquam@mauris.ca            | George Crosby     |             89 |
| 85158 | ante.Nunc.mauris@arcu.co.uk                    | Victor Willis     |             88 |
|  3036 | gravida.Praesent@cursusdiamat.ca               | Tiger Dale        |             87 |
|  9784 | nec.cursus@euismodenimEtiam.ca                 | Yuli Alford       |             86 |
| 43017 | Sed.nec@ornareegestas.net                      | Jesse Thornton    |             86 |
| 73056 | Donec.at.arcu@lectusa.org                      | Quinn Orr         |             86 |
.
.
.
| 19932 | ligula.tortor@cursus.co.uk                     | Thane Estrada     |              5 |
| 63591 | sapien.imperdiet@velit.edu                     | Adrian Zimmerman  |              5 |
| 69586 | rutrum.non@Nulla.com                           | Byron Daniels     |              5 |
| 58926 | mi.ac.mattis@Nullatinciduntneque.org           | Aladdin Gomez     |              4 |
|  4782 | nunc.sed.libero@Sed.edu                        | Preston Hernandez |              2 |
|  8134 | Sed@molestietellusAenean.net                   | Ali Horn          |              2 |
+-------+------------------------------------------------+-------------------+----------------+
```
Now, let's see how much the query cost, with the following command:

```sql
SHOW STATUS LIKE 'Last_query_cost';
```
```console title="Output"
+-----------------+---------------+
| Variable_name   | Value         |
+-----------------+---------------+
| Last_query_cost | 108034.649000 |
+-----------------+---------------+
```

Notice that the query cost 108034.65 units.

Let's check how MySQL resolves the query by running the following command and observing the EXTRA section:

:::info
You can learn more about information from `EXPLAIN EXTRA` at the official documentation [here](https://dev.mysql.com/doc/refman/8.0/en/explain-output.html#explain-extra-information).
:::


```sql
EXPLAIN SELECT * FROM telecom_data WHERE megabytes_used < 100 ORDER BY megabytes_used DESC\G
```
:::info
The client command `\G` is used in order to display the results vertically. To know more about client commands, visit the [documentation](https://dev.mysql.com/doc/refman/8.0/en/mysql-commands.html).
:::
```console title="Output"
*************************** 1. row ***************************
           id: 1
  select_type: SIMPLE
        table: telecom_data
   partitions: NULL
         type: ALL
possible_keys: NULL
          key: NULL
      key_len: NULL
          ref: NULL
         rows: 98104
     filtered: 33.33
        Extra: Using where; Using filesort
1 row in set, 1 warning (0.00 sec)
```
Observe that MySQL is using the `filesort` operation in order to resolve the query.

:::info
A `filesort` operation uses temporary disk files as necessary if the result set is too large to fit in memory. To learn more about how filesort is used to satisfy `ORDER BY` clause in MySQL, visit [here](https://dev.mysql.com/doc/refman/8.0/en/order-by-optimization.html#order-by-filesort)
:::

Now, let's try to optimize the query by using a descending index. Let's create a descending index on the `megabytes_used` column with the following command:

```sql
CREATE INDEX fastscan_idx ON telecom_data(megabytes_used DESC);
```
```console title="Output"
Query OK, 0 rows affected (0.25 sec)
Records: 0  Duplicates: 0  Warnings: 0
```
Now, let's run the following query again and check the cost:
```sql
SELECT * FROM telecom_data WHERE megabytes_used < 100 ORDER BY megabytes_used DESC;
```
```console title="Output"
+-------+------------------------------------------------+-------------------+----------------+
| id    | email_address                                  | user_name         | megabytes_used |
+-------+------------------------------------------------+-------------------+----------------+
| 27383 | augue.Sed@malesuadaiderat.edu                  | Marsden Hurst     |             98 |
| 66371 | sem.elit@nislsem.ca                            | Finn Hayden       |             98 |
| 14929 | Nam.nulla.magna@aliquet.edu                    | Merritt Potts     |             97 |
| 18789 | aliquet.diam.Sed@acurnaUt.ca                   | Camden Cash       |             97 |
| 43361 | tincidunt.pede.ac@molestie.ca                  | Kareem Bond       |             96 |
.
.
|  8134 | Sed@molestietellusAenean.net                   | Ali Horn          |              2 |
+-------+------------------------------------------------+-------------------+----------------+
```
```sql
SHOW STATUS LIKE 'Last_query_cost';
```
```console title="Output"
+-----------------+-----------+
| Variable_name   | Value     |
+-----------------+-----------+
| Last_query_cost | 52.009000 |
+-----------------+-----------+
```
(Note: The results will vary, ​​depending on the data that is stored in the database)

Amazing! Now our query cost only 52.01 units, compared to 108034.65 units earlier when the descending index was not used.

Let's check how MySQL resolves the query in this case:

```sql
EXPLAIN SELECT * FROM telecom_data WHERE megabytes_used < 100 ORDER BY megabytes_used DESC\G
```
```console title="Output"
*************************** 1. row ***************************
           id: 1
  select_type: SIMPLE
        table: telecom_data
   partitions: NULL
         type: range
possible_keys: fastscan_idx
          key: fastscan_idx
      key_len: 4
          ref: NULL
         rows: 115
     filtered: 100.00
        Extra: Using index condition
1 row in set, 1 warning (0.00 sec)
```

Observe that MySQL has used the index we created in order to resolve the query this time.

Additionally, you can check whether a descending index is being used in a query or not by checking `(reverse)` along the name of the index, while running the `EXPLAIN` command with `FORMAT=TREE` option. Here is an example:


```sql
EXPLAIN FORMAT=TREE SELECT * FROM telecom_data WHERE megabytes_used < 100 ORDER BY megabytes_used ASC\G
```
```console title="Output"
*************************** 1. row ***************************
EXPLAIN: -> Index range scan on telecom_data using fastscan_idx over (100 < megabytes_used < NULL) (reverse), with index condition: (telecom_data.megabytes_used < 100)  (cost=52.01 rows=115)
1 row in set (0.00 sec)
```

:::info
Descending indexes are supported only for the InnoDB storage engine. You can learn more about Descending Indexes and their limitations in MySQL from the [official documentation](https://dev.mysql.com/doc/refman/8.0/en/descending-indexes.html).
:::

We have seen that creating a descending index is a smart choice when using queries with `ORDER BY` clauses. Now, let's see how we can easily manage descending indexes in MySQL using Atlas.

### Managing Descending Indexes is easy with Atlas

Atlas is an open-source project which allows us to manage our database using a simple and easy-to-understand declarative syntax (similar to Terraform).

:::info
If you are just getting started, install the latest version of Atlas using the guide to [setting up Atlas](https://atlasgo.io/cli/getting-started/setting-up).
:::

#### Managing Descending Indexes with Atlas

We will first use the `atlas schema inspect` command to get an HCL representation of the table we created earlier (without any indexes) by using the Atlas CLI:

```console
atlas schema inspect -u "mysql://root:@localhost:3306/telecom_data" > schema.hcl
```
```hcl title="schema.hcl"
table "telecom_data" {
  schema = schema.telecom_data
  column "id" {
    null           = false
    type           = bigint
    unsigned       = true
    auto_increment = true
  }
  column "email_address" {
    null = true
    type = varchar(255)
  }
  column "user_name" {
    null = true
    type = varchar(255)
  }
  column "megabytes_used" {
    null = true
    type = bigint
  }
  primary_key {
    columns = [column.id]
  }
}
schema "telecom_data" {
  charset = "utf8mb4"
  collate = "utf8mb4_0900_ai_ci"
}
```
Now, lets add the following index definition to the file:
```hcl
  index "fastscan_idx" {
    on {
      desc   = true
      column = column.megabytes_used
    }
  }
```

Save the file and apply the schema changes on the database by using the following command:

```console
atlas schema apply --url "mysql://root:@localhost:3306/telecom_data" -f schema.hcl
```

Atlas generates the necessary SQL statements to add the new descending index to the database schema. Press Enter while the `Apply` option is highlighted to apply the changes:

```console title="Output"
-- Planned Changes:
-- Modify "telecom_data" table
ALTER TABLE `telecom_data`.`telecom_data` ADD INDEX `fastscan_idx` (`megabytes_used` DESC)
Use the arrow keys to navigate: ↓ ↑ → ←
? Are you sure?:
  ▸ Apply
    Abort
```

To verify that our new index was created, open the database command line tool from the previous step and run:

```sql
show index from telecom_data\G
```
```console title="Output"
*************************** 1. row ***************************
        Table: telecom_data
   Non_unique: 0
     Key_name: PRIMARY
.
.
.
*************************** 2. row ***************************
        Table: telecom_data
   Non_unique: 1
     Key_name: fastscan_idx
 Seq_in_index: 1
  Column_name: megabytes_used
    Collation: D
  Cardinality: 62544
     Sub_part: NULL
       Packed: NULL
         Null: YES
   Index_type: BTREE
      Comment:
Index_comment:
      Visible: YES
   Expression: NULL
2 rows in set (0.00 sec)
```
Amazing! Our new descending index is now created as seen on row no. 2!

### Wrapping up

In this guide, we demonstrated how to create and use descending indexes in MySQL to optimize queries with `ORDER BY` clause, and how we can use Atlas to easily manage descending indexes in MySQL database.

## Need More Help?​

[Join the Ariga Discord Server](https://discord.com/invite/zZ6sWVg6NT) for early access to features and the ability to provide exclusive feedback that improves your Database Management Tooling.

[Sign up](https://atlasnewsletter.substack.com/) to our newsletter to stay up to date about [Atlas](https://atlasgo.io), our OSS database schema management tool, and our cloud platform [Atlas Cloud](https://atlasgo.cloud).
