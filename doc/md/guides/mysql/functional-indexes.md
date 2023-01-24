---
id: functional-indexes
title: Functional Indexes in MySQL
slug: /guides/mysql/functional-indexes
---

### What are functional key parts?​

In contrast to a "normal" index which indexes column values, functional key parts can index expression values. Hence, functional key parts enable indexing values which are not stored directly in the table itself. MySQL 8.0.13 and higher have added support for functional key parts.

### When are functional indexes helpful?

In cases where MySQL users may want to optimize their queries containing expressions, indexing column values will not achieve the desired result. In such cases, users should create a functional index for the expressions in their query in order to optimize the performance of the query.

### Syntax​

Here is how you can define functional indexes in a table:

```sql
CREATE TABLE table_name (column_1 INT, column_2 INT, INDEX functional_index ((ABS(column_1))));
```
```sql
CREATE INDEX functional_index ON table_name ((column_1 + column_2));
```
```sql
CREATE INDEX functional_index ON table_name ((column_1 + column_2), (column_1 - column_2), column_1);
```
```sql
ALTER TABLE table_name ADD INDEX ((column_1 * 90) DESC);
```

:::info
Expressions must be enclosed within parentheses in order to create a functional index. If you do not enclose expressions within parentheses in the index definition, you will get the following error:

`ERROR 1064 (42000): You have an error in your SQL syntax`
:::

### Example​

Let’s create a table containing a list of students and the marks they received in each subject with the following command:

```sql
CREATE TABLE `scorecard` (
  `id` mediumint(8) unsigned NOT NULL auto_increment,
  `name` varchar(255),
  `science` mediumint,
  `mathematics` mediumint,
  `language` mediumint,
  `social_science` mediumint,
  `arts` mediumint,
  PRIMARY KEY (`id`)
) AUTO_INCREMENT=1;
```

Here is what a portion of the table might look like after inserting values:

```sql
SELECT * FROM scorecard
```
```console title="Output"
+----+------------------+---------+-------------+----------+----------------+------+
| id | name             | science | mathematics | language | social_science | arts |
+----+------------------+---------+-------------+----------+----------------+------+
|  1 | Abel Warren      |      73 |          73 |       73 |             73 |   73 |
|  2 | Erick Valentine  |      90 |           7 |       91 |              8 |   57 |
|  3 | Janice Payne     |      49 |           6 |       91 |             48 |   77 |
|  4 | Gretchen Mason   |      98 |          76 |       67 |             46 |   11 |
|  5 | Lawanda Noble    |      85 |          22 |        7 |             44 |   33 |
|  6 | Robbie Baird     |      98 |          66 |        2 |             69 |   67 |
|  7 | Carla Compton    |      22 |          80 |       69 |             27 |   54 |
|  8 | Heath Stafford   |       7 |           2 |       80 |             75 |   89 |
|  9 | Kendra Stevenson |      74 |          81 |       15 |             21 |    1 |
| 10 | Brandie Chase    |      94 |          94 |       92 |             92 |   93 |
.
.
| 1000000 | James Strong     |      47 |          78 |       36 |             67 |   73 |
```

We do not have any indexes other than the primary index on the `id` column. Now, suppose we want information about the top ten students who scored the best average in math and science combined. Let's query that data with the following command:

```sql
SELECT
    id, name, ((science + mathematics)/2) as average_score
FROM
    scorecard
ORDER BY
    average_score DESC
LIMIT 10
```

```console title="Output"
+-------+------------------+-------------------+
| id    | name             | average_score     |
+-------+------------------+-------------------+
| 31593 | Rusty Mercer     | 99.0000           |
| 24451 | Sara Fowler      | 99.0000           |
|  8190 | Cecil Glass      | 99.0000           |
| 47193 | Nelson Horn      | 99.0000           |
|  7032 | Calvin Freeman   | 99.0000           |
| 99198 | Mathew Rowland   | 99.0000           |
| 50054 | Josephine Forbes | 99.0000           |
| 52280 | Krystal Yoder    | 99.0000           |
| 12639 | Maribel Carroll  | 99.0000           |
| 53245 | Ramona Fischer   | 99.0000           |
+-------+------------------+-------------------+
10 rows in set (0.47 sec)
```

Now, let's see how much the query cost with the following command:

```sql
EXPLAIN FORMAT=JSON
SELECT
    id, name, ((science + mathematics)/2) as average_score
FROM
    scorecard
ORDER BY
    average_score DESC
LIMIT 10;
```

```console title="Output"
| {
  "query_block": {
    "select_id": 1,
    "cost_info": {
      "query_cost": "100567.25"
    },
    "ordering_operation": {
      "using_filesort": true,
      "table": {
        "table_name": "scorecard",
        "access_type": "ALL",
        "rows_examined_per_scan": 997100,
        "rows_produced_per_join": 997100,
        "filtered": "100.00",
        "cost_info": {
          "read_cost": "857.25",
          "eval_cost": "99710.00",
          "prefix_cost": "100567.25",
          "data_read_per_join": "996M"
        },
        "used_columns": [
          "id",
          "name",
          "science",
          "mathematics"
        ]
      }
    }
  }
} |
```

The cost is huge (100567.25) because MySQL uses the filesort operation when it is unable to utilize an index. As we are making use of columns `science` and `mathematics`, let’s try to optimize the query performance by indexing these columns with the following command:

```sql
CREATE INDEX
    normal_index
ON
    scorecard (science, mathematics);
```

```console title="Output"
Query OK, 0 rows affected (1.40 sec)
Records: 0  Duplicates: 0  Warnings: 0
```

Awesome, our index is now created! Let's check again how much our previous query cost, with the same command:

```sql
EXPLAIN ANALYZE
SELECT
    id, name, ((science + mathematics)/2) as average_score
FROM
    scorecard
ORDER BY
    average_score DESC
LIMIT 10;
```

```console title="Output"
| -> Limit: 10 row(s)  (cost=100448.95 rows=10) (actual time=552.143..552.145 rows=10 loops=1)
    -> Sort: average_score DESC, limit input to 10 row(s) per chunk  (cost=100448.95 rows=995917) (actual time=552.142..552.143 rows=10 loops=1)
        -> Table scan on scorecard  (cost=100448.95 rows=995917) (actual time=0.076..263.811 rows=1000000 loops=1)
 |
```

:::info
You can learn more about EXPLAIN EXTRA in the official documentation [here](https://dev.mysql.com/doc/refman/8.0/en/explain-output.html#explain-extra-information).
:::

The cost is still the same, our query hasn’t used the index we created which can be seen in the output of the following command (key = NULL):

```sql
EXPLAIN
SELECT
    id, name, ((science + mathematics)/2) as average_score
FROM
    scorecard
ORDER BY
    average_score DESC
LIMIT 10\G;
```

```console title="Output"
*************************** 1. row ***************************
           id: 1
  select_type: SIMPLE
        table: scorecard
   partitions: NULL
         type: ALL
possible_keys: NULL
          key: NULL
      key_len: NULL
          ref: NULL
         rows: 997269
     filtered: 100.00
        Extra: Using filesort
1 row in set, 1 warning (0.00 sec)
```

:::info
A filesort operation uses temporary disk files as necessary if the result set is too large to fit in memory. To learn more about how filesort is used to satisfy the ORDER BY clause in MySQL,  click [here](https://dev.mysql.com/doc/refman/8.0/en/order-by-optimization.html#order-by-filesort).
:::

This is where having knowledge about functional indexes becomes essential. Now, let's try to optimize the query by creating a functional index with the expression `((science + mathematics)/2)` from our query, with the following command:

```sql
CREATE INDEX
    functional_idx
ON
    scorecard((science + mathamatics)/2);
```

```console title="Output"
ERROR 1064 (42000): You have an error in your SQL syntax; check the manual that corresponds to your MySQL server version for the right syntax to use near '/2)' at line 4
```

Oops, that didn’t work! As we mentioned in the [syntax](#syntax) section above, expressions must be enclosed within parentheses in order to create a functional index. Let’s try this again with the correct syntax:

```sql
CREATE INDEX
    functional_idx
ON
    scorecard(((science + mathamatics)/2));
```

```console title="Output"
Query OK, 0 rows affected (2.13 sec)
Records: 0  Duplicates: 0  Warnings: 0
```

Let's check again how much our previous query cost, with the same command:

```sql
EXPLAIN
SELECT
    id, name, ((science + mathematics)/2) as average_score
FROM
    scorecard
ORDER BY
    average_score DESC
LIMIT 10;
```

```console title="Output"
| -> Limit: 10 row(s)  (cost=0.01 rows=10) (actual time=0.149..0.160 rows=10 loops=1)
    -> Index scan on scorecard using functional_idx (reverse)  (cost=0.01 rows=10) (actual time=0.148..0.157 rows=10 loops=1)
|
```

The cost has reduced down to 0.01! The index scan has used the index "functional_idx" in order to execute the query. This is a significant improvement in performance compared to the scenarios above with an index on a column (unused) or no index at all (both of which cost 100000+ units).

### Limitations
1. Functional indexes cannot be defined as primary keys.
2. Any conditions, functions or expressions in the query other than the ones defined in the functional index will not make use of the functional index. Thus, one needs to create a functional index with the exact conditions, functions or expressions which are used by the query.

## Managing Functional Indexes is easy with Atlas​
Atlas is an open-source project which allows us to manage our database using a simple and easy-to-understand declarative syntax.

:::info
If you are just getting started, install the latest version of Atlas using the guide to [setting up Atlas](https://atlasgo.io/cli/getting-started/setting-up).
:::

We will first use the `atlas schema inspect` command to get an [HCL](https://atlasgo.io/guides/ddl#hcl) representation of the table we created earlier (without any indexes other than primary index on id column) by using the Atlas CLI:

```console
atlas schema inspect -u "mysql://root:password@localhost:3306/scorecard" > schema.hcl
```

```hcl title="schema.hcl"
table "scorecard" {
  schema = schema.scorecard
  column "id" {
    type           = int
    auto_increment = true
  }
  column "name" {
    type = varchar(255)
  }
  column "science" {
    type = mediumint
  }
  column "mathematics" {
    type = mediumint
  }
  column "language" {
    type = mediumint
  }
  column "social_science" {
    type = mediumint
  }
  column "arts" {
    type = mediumint
  }
  primary_key {
    columns = [column.id]
  }
}
schema "scorecard" {
  charset = "utf8mb4"
  collate = "utf8mb4_0900_ai_ci"
}
```

Now, let's add the following index definition to the file:

```hcl
  index "sci_math_avg_idx" {
    on {
      expr = "((`science` + `mathematics`) / 2)"
    }
  }
```

Save and apply the schema changes on the database by using the apply command:

```console
atlas schema apply -u "mysql://root:password@localhost:3306/scorecard" -f schema.hcl --dev-url docker://mysql/8/scorecard
```

:::info
If you get `Error: pulling image: exit status 1` error, ensure that Docker Desktop is up and running.
:::

Atlas generates the necessary SQL statements to add the new index to the database schema.
Press `Enter` key while the Apply option is highlighted to apply the changes:

```console
-- Planned Changes:
-- Modify "scorecard" table
ALTER TABLE `scorecard`.`scorecard` ADD INDEX `sci_math_avg_idx` (((`science` + `mathematics`) / 2))
Use the arrow keys to navigate: ↓ ↑ → ←
? Are you sure?:
  ▸ Apply
    Abort
```

To verify that our new index was created, open the database command line tool from the previous step and run:

```sql
SHOW INDEXES FROM scorecard;
```

```console title="Output"
*************************** 1. row ***************************
        Table: scorecard
   Non_unique: 0
     Key_name: PRIMARY
 Seq_in_index: 1
  Column_name: id
    Collation: A
  Cardinality: 997438
     Sub_part: NULL
       Packed: NULL
         Null:
   Index_type: BTREE
      Comment:
Index_comment:
      Visible: YES
   Expression: NULL
*************************** 2. row ***************************
        Table: scorecard
   Non_unique: 1
     Key_name: sci_math_avg_idx
 Seq_in_index: 1
  Column_name: NULL
    Collation: A
  Cardinality: 190
     Sub_part: NULL
       Packed: NULL
         Null: YES
   Index_type: BTREE
      Comment:
Index_comment:
      Visible: YES
   Expression: ((`science` + `mathematics`) / 2)
2 rows in set (0.00 sec)
```

Amazing! Our new index sci_math_avg_idx with the expression “(`science` + `mathematics`) / 2” is now created!

## Wrapping up​
In this guide, we demonstrated how using functional indexes with an appropriate expression becomes a very crucial skill in optimizing query performance with combinations of certain expressions, functions and/or conditions.

# Need More Help?​​
[Join the Ariga Discord Server](https://discord.com/invite/zZ6sWVg6NT) for early access to features and the ability to provide exclusive feedback that improves your Database Management Tooling.

[Sign up](https://www.getrevue.co/profile/ariga) to our newsletter to stay up to date about [Atlas](https://atlasgo.io/), our OSS database schema management tool, and our cloud platform [Atlas Cloud](https://atlasgo.cloud/).
