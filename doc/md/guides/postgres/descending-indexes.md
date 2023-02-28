---
id: descending-indexes
title: Descending Indexes in PostgreSQL
slug: /guides/postgres/descending-indexes
---

### What are descending indexes?​
Descending indexes are indexes where key parts are stored in descending order. Descending indexes can be helpful in PostgreSQL when queries involve ordering the results in descending order and/or filtering out null values.

### When are descending indexes helpful?​
For example, if a query uses an `ORDER BY` clause to sort the results of a query in descending order, then a descending index can improve the performance of that query significantly. 

Similarly, if a query often filters out null values and uses an index to do that, a descending index with the `NULLS FIRST` option can help the index efficiently filter out null values. 

### Syntax​

Here is how you can create a descending index:

```sql
CREATE INDEX index_name ON table_name (column_name DESC);
```

Here is how you can create a descending index with `NULLS FIRST` option:

```sql
CREATE INDEX index_name ON table_name (column_name DESC NULLS FIRST);
```

To create a descending index with `NULLS LAST` option:

```sql
CREATE INDEX index_name ON table_name (column_name DESC NULLS LAST);
```

In general, `ASC` or `DESC` specifiers are used with `ORDER BY` clauses to specify whether index values are stored in ascending or descending order.

It is also worth mentioning that `NULLS FIRST` is used by default if it has not been specified in the command.

### Example​

Let’s create a table which represents data of an ISP’s subscribers along with their email addresses and broadband data usage with the following command:

```sql
CREATE TABLE telecom_data (
    id bigserial NOT NULL,
    email_address varchar(255),
    user_name varchar(255),
    megabytes_used bigint,
    PRIMARY KEY (id)
);
```

Here is how a portion of the table might look like after inserting values:

```sql
SELECT * FROM telecom_data
```
```console title=Output
 id |       email_address        |   user_name    | megabytes_used 
----+----------------------------+----------------+----------------
  1 | jason89@example.com        | michellewebb   |           6777
  2 | edwardstara@example.net    | joshuabautista |           5910
  3 | sarahbarrett@example.com   | melissaknight  |           8117
  4 | lorigonzalez@example.net   | michaelboyle   |           7627
  5 | melissajackson@example.net | jimmymarshall  |            105
  6 | smerritt@example.net       | andersontaylor |           1235
  7 | kevinatkinson@example.net  | mossjoseph     |           1782
  8 | campbellroy@example.org    | nicholas85     |           2801
  9 | gregory36@example.com      | iyoung         |           1781
 10 | michael26@example.com      | fcantrell      |           8024
.
.
.
(1000010 rows)
```

We do not have any indexes other than the primary index on the `id` column. Now, suppose we want information about the top 10 subscribers with maximum usage, but in descending order. Let's query that data with the following command:

```sql
SELECT * FROM telecom_data ORDER BY megabytes_used DESC LIMIT 10;
```

```console
   id   |      email_address       |     user_name      | megabytes_used 
--------+--------------------------+--------------------+----------------
  76887 | amanda00@example.com     | davidholland       |          10000
 106416 | pachecojacob@example.org | lisaperez          |          10000
  43078 | ydunn@example.com        | jamestracy         |           9999
  33363 | iwilliamson@example.org  | schroedernicole    |           9999
   4131 | james55@example.org      | lindseymolly       |           9999
  21796 | rchase@example.net       | amanda79           |           9999
  38158 | camposellen@example.net  | batesmarcus        |           9998
  41207 | ryan45@example.org       | oshaw              |           9997
  27160 | pjones@example.com       | lturner            |           9996
 111400 | josephspence@example.net | moraleschristopher |           9995
(10 rows)
```

Now, let's see how the query performed with the following command:

```sql
EXPLAIN ANALYZE
SELECT * FROM telecom_data ORDER BY megabytes_used DESC LIMIT 10;
```

```console title=Output
       QUERY PLAN                                            
-----------------------
 Limit  (cost=24157.84..24159.01 rows=10 width=48) (actual time=130.078..135.055 rows=10 loops=1)
   ->  Gather Merge  (cost=24157.84..121387.86 rows=833342 width=48) (actual time=130.076..135.052 rows=10 loops=1)
         Workers Planned: 2
         Workers Launched: 2
         ->  Sort  (cost=23157.82..24199.50 rows=416671 width=48) (actual time=122.863..122.865 rows=10 loops=3)
               Sort Key: megabytes_used DESC
               Sort Method: top-N heapsort  Memory: 27kB
               Worker 0:  Sort Method: top-N heapsort  Memory: 27kB
               Worker 1:  Sort Method: top-N heapsort  Memory: 27kB
               ->  Parallel Seq Scan on telecom_data  (cost=0.00..14153.71 rows=416671 width=48) (actual time=0.009..47.598 rows=333337 loops=3)
 Planning Time: 0.083 ms
 Execution Time: 135.082 ms
(12 rows)
```

:::info
The `EXPLAIN` command is used for understanding the performance of a query. You can learn more about usage of `EXPLAIN` command with `ANALYZE` option [here](https://www.postgresql.org/docs/14/using-explain.html#USING-EXPLAIN-ANALYZE).
:::

Notice that the total execution time for this operation is 135.082 ms.

Now, let's optimize the query by using a descending index. We will create a descending index on the `megabytes_used` column with the following command:

```sql
CREATE INDEX fastscan_idx ON telecom_data(megabytes_used DESC);
```

Now, let's run the following query again and check the cost:

```sql
EXPLAIN ANALYZE
SELECT * FROM telecom_data ORDER BY megabytes_used DESC LIMIT 10;
```
```console title=Output
                   QUERY PLAN                                                                  
------------------------------------------------------
 Limit  (cost=0.42..1.01 rows=10 width=48) (actual time=1.033..1.052 rows=10 loops=1)
   ->  Index Scan using fastscan_idx on telecom_data  (cost=0.42..58428.46 rows=1000010 width=48) (actual time=1.032..1.048 rows=10 loops=1)
 Planning Time: 1.938 ms
 Execution Time: 1.072 ms
(4 rows)
```

(Note: The results will vary, ​​depending on the data that is stored in the database)

Amazing! Now the total execution time is only 1.072ms, compared to 135.082 ms earlier. 

In the first query plan, a parallel sequential scan is performed on the entire table to filter and sort the data, which takes longer. In contrast, the second query plan uses an index scan to access the required data directly, and only scans a small portion of the table, which significantly reduces the execution time.

:::info
Descending indexes can increase the overhead of `INSERT`, `UPDATE` and `DELETE` operations, as the index must be updated to maintain the descending order. Hence, it must be used carefully. To learn more about creating indexes with the `ORDER BY` clause, visit the official documentation [here](https://www.postgresql.org/docs/current/indexes-ordering.html).
:::

We have seen that creating a descending index is a smart choice when using queries with `ORDER BY` clauses. Now, let's see how we can easily manage descending indexes in a PostgreSQL database using Atlas.

## Managing Descending Indexes is easy with Atlas​

Atlas is an open-source project which allows us to manage our database using a simple and easy-to-understand declarative syntax (similar to Terraform), as well as SQL.

:::info
If you are just getting started, install the latest version of Atlas using the guide to [set up Atlas](/getting-started/).
:::

#### Example
We will first use the `atlas schema inspect` command to get an HCL representation of the table we created earlier (without any indexes) by using the Atlas CLI:

```
atlas schema inspect -u "postgres://postgres:pass@localhost:5432/telecom_data?sslmode=disable" > schema.hcl
```
```hcl title=schema.hcl
schema "public" {}

table "telecom_data" {
  schema = schema.public
  column "id" {
    null = false
    type = bigserial
  }
  column "email_address" {
    null = true
    type = character_varying(255)
  }
  column "user_name" {
    null = true
    type = character_varying(255)
  }
  column "megabytes_used" {
    null = true
    type = bigint
  }
  primary_key {
    columns = [column.id]
  }
}
```

Now, let’s add the following index definition to the file:

```hcl title=schema.hcl {24-29}
schema "public" {}

table "telecom_data" {
  schema = schema.public
  column "id" {
    null = false
    type = bigserial
  }
  column "email_address" {
    null = true
    type = character_varying(255)
  }
  column "user_name" {
    null = true
    type = character_varying(255)
  }
  column "megabytes_used" {
    null = true
    type = bigint
  }
  primary_key {
    columns = [column.id]
  }
  index "fastscan_idx" {
    on {
      desc   = true
      column = column.megabytes_used
    }
  }
}
```

Save the file and apply the schema changes on the database by using the following command:

```console
atlas schema apply --url "postgres://postgres:pass@localhost:5432/telecom_data?sslmode=disable" --to "file://schema.hcl"
```

Atlas generates the necessary SQL statements to add the new descending index to the database schema. Press Enter while the `Apply` option is highlighted to apply the changes:

```console
-- Planned Changes:
-- Create index "fastscan_idx" to table: "telecom_data"
// highlight-next-line-info
CREATE INDEX "fastscan_idx" ON "public"."telecom_data" ("megabytes_used" DESC);
Use the arrow keys to navigate: ↓ ↑ → ← 
? Are you sure?: 
  ▸ Apply
    Abort
```

To verify that our new index was created, run the following command:

```
atlas schema inspect -u "postgres://postgres:pass@localhost:5432/telecom_data?sslmode=disable" | grep -A5 index
```

```hcl title=Output
  index "fastscan_idx" {
    on {
      desc   = true
      column = column.megabytes_used
    }
  }
```

Amazing! Our new descending index is now created!

## Wrapping up​
In this guide, we demonstrated how to create and use descending indexes in PostgreSQL to optimize queries with the `ORDER BY` clause, and how we can use Atlas to easily manage descending indexes in a PostgreSQL database.

## Need More Help?​​
[Join the Atlas Discord Server](https://discord.com/invite/zZ6sWVg6NT) for early access to features and the ability to provide exclusive feedback that improves your Database Management Tooling.

[Sign up](https://www.getrevue.co/profile/ariga) to our newsletter to stay up to date about [Atlas](https://atlasgo.io/), our OSS database schema management tool, and our cloud platform [Atlas Cloud](https://atlasgo.cloud/).