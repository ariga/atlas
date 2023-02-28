---
id: functional-indexes
title: Functional Indexes in PostgreSQL
slug: /guides/postgres/functional-indexes
---

### What are functional key parts?​​
A functional index is an index in a database that is based on the result of a function applied to one or more columns in a table. Functional key parts can index expression values. Hence, functional key parts enable indexing values that are not stored directly in the table itself.

### When are functional indexes helpful?​
Functional indexes are helpful in a PostgreSQL database when the query retrieves data based on the result of a function. This is particularly useful when the function requires high computational power to execute. 

By creating an index based on the result of the function used in the query, the database can quickly find the matching rows based on the function output, rather than having to perform a full table scan and doing the necessary computation. This can lead to significant improvements in query performance, for example in large databases with complex queries. 

Some common use cases for functional indexes in PostgreSQL include case-insensitive searching, date calculations, and full-text search.

### Syntax​​

Here is how you can define functional indexes in a table:

```sql
CREATE INDEX functional_idx ON table_name ((expression));
```

**Here are some examples:**

Index using a mathematical function:

```sql
CREATE INDEX functional_idx ON table_name ((column1 + column2));
```

Index using a string function:

```sql
CREATE INDEX functional_idx ON table_name (lower(column1));
```

:::info
Expressions must be enclosed within parentheses in order to create a functional index. If you do not enclose expressions within parentheses in the index definition, you will get a syntax error.
:::

### Example​​

Let’s create a table containing a list of students and the marks they received in each subject with the following command:

```sql
CREATE TABLE scorecard (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255),
    science INTEGER,
    mathematics INTEGER,
    language INTEGER,
    social_science INTEGER,
    arts INTEGER
);
```

Here is what a portion of the table might look like after inserting values:

```sql
SELECT * FROM scorecard
```

```console title=Output
 id |       name        | science | mathematics | language | social_science | arts 
----+-------------------+---------+-------------+----------+----------------+------
  1 | Ronald Bradley    |       9 |          71 |       60 |             11 |   58
  2 | Dale Ellis        |      22 |          96 |       20 |             25 |   63
  3 | Jeremy Gray       |      84 |          60 |       46 |             43 |   71
  4 | Jacqueline Porter |      34 |          44 |       81 |             57 |   94
  5 | Amanda Cohen      |      36 |          61 |       30 |             36 |   94
.
.
 1000005 | Laura Blair        |      72 |          32 |        2 |             52 |   38
(1000005 rows)
```

We do not have any indexes other than the primary index on the `id` column. Now, suppose we want information about the top ten students who scored the best average in math and science combined. Let's query that data with the following command:

```sql
SELECT
   id, name, ((science + mathematics)/2) as average_score
FROM
   scorecard
ORDER BY
   average_score DESC
LIMIT 10;
```

```console title=Output
   id   |       name       | average_score 
--------+------------------+---------------
  95401 | Jacob Day        |           100
 105523 | Patrick Welch    |           100
  19727 | Thomas Lopez     |           100
  31179 | Jennifer Gibson  |           100
 100572 | Tara Morris      |           100
   9744 | Bob Edwards      |           100
   8519 | Andres Hernandez |            99
  16250 | Ana Kelly        |            99
  24575 | Allen Jenkins    |            99
 116991 | Jeffery Miller   |            97
(10 rows)
```

Now, let's see how the query performs with the following command:

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

```console title=Output
      QUERY PLAN                                                                  
-----------------------
Limit  (cost=24762.12..24763.29 rows=10 width=22) (actual time=159.121..161.760 rows=10 loops=1)
   ->  Gather Merge  (cost=24762.12..121991.68 rows=833338 width=22) (actual time=159.119..161.757 rows=10 loops=1)
         Workers Planned: 2
         Workers Launched: 2
         ->  Sort  (cost=23762.10..24803.77 rows=416669 width=22) (actual time=153.364..153.366 rows=10 loops=3)
               Sort Key: (((science + mathematics) / 2)) DESC
               Sort Method: top-N heapsort  Memory: 26kB
               Worker 0:  Sort Method: top-N heapsort  Memory: 26kB
               Worker 1:  Sort Method: top-N heapsort  Memory: 26kB
               ->  Parallel Seq Scan on scorecard  (cost=0.00..14758.03 rows=416669 width=22) (actual time=0.015..81.038 rows=333335 loops=3)
 Planning Time: 0.105 ms
 Execution Time: 161.802 ms
(12 rows)
```

:::info
The EXPLAIN command is used for understanding the performance of a query. You can learn more about usage of the EXPLAIN command with the ANALYZE option [here](https://www.postgresql.org/docs/14/using-explain.html#USING-EXPLAIN-ANALYZE).
:::

The overall plan and the execution time suggests that there is scope for optimization here. The planner uses the sort operation when it is unable to utilize any index. 

As we are making use of columns `science` and `mathematics`, let’s try to optimize the query performance by indexing these columns with the following command:

```sql
CREATE INDEX
   normal_index
ON
   scorecard (science, mathematics);
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

```console title=Output
 QUERY PLAN                                                                  
------------
 Limit  (cost=24762.12..24763.29 rows=10 width=22) (actual time=155.803..158.791 rows=10 loops=1)
   ->  Gather Merge  (cost=24762.12..121991.68 rows=833338 width=22) (actual time=155.801..158.787 rows=10 loops=1)
         Workers Planned: 2
         Workers Launched: 2
         ->  Sort  (cost=23762.10..24803.77 rows=416669 width=22) (actual time=149.608..149.609 rows=10 loops=3)
               Sort Key: (((science + mathematics) / 2)) DESC
               Sort Method: top-N heapsort  Memory: 26kB
               Worker 0:  Sort Method: top-N heapsort  Memory: 26kB
               Worker 1:  Sort Method: top-N heapsort  Memory: 26kB
               ->  Parallel Seq Scan on scorecard  (cost=0.00..14758.03 rows=416669 width=22) (actual time=0.014..78.088 rows=333335 loops=3)
 Planning Time: 0.379 ms
 Execution Time: 158.813 ms
(12 rows)
```

There is no significant change in `Execution Time` and the cost is still the same, because our query hasn’t used the index we have created. This is where having knowledge about functional indexes becomes essential. Now, let's try to optimize the query by creating a functional index with the expression `((science + mathematics)/2)` from our query, with the following command:

```sql
CREATE INDEX
   functional_idx
ON
   scorecard((science + mathematics)/2);
```

```console title=Output
ERROR:  syntax error at or near "/"
LINE 4:    scorecard((science + mathematics)/2);
                                            ^
```

Oops, that didn’t work! As we mentioned in the syntax section above, expressions must be enclosed within parentheses in order to create a functional index. Let’s try this again with the correct syntax:

```sql
CREATE INDEX
   functional_idx
ON
   scorecard(((science + mathematics)/2));
```

Let's check again how much our previous query cost, with the same command:

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

```console title=Output
         QUERY PLAN                     
---------------------------
 Limit  (cost=0.42..1.00 rows=10 width=22) (actual time=0.060..0.072 rows=10 loops=1)
   ->  Index Scan Backward using functional_idx on scorecard  (cost=0.42..57448.53 rows=1000005 width=22) (actual time=0.058..0.068 rows=10 loops=1)
 Planning Time: 0.344 ms
 Execution Time: 0.086 ms
(4 rows)
```

The execution time has reduced down to just 0.086ms! By using the index we created, the query can perform the search much more efficiently as it can directly look up the matching rows using the indexed values rather than having to scan the entire table.

It is worth noting that while functional indexes can improve performance for certain queries, they can also introduce overhead for insert and update operations, since the index needs to be updated every time the table is modified. This can slow down write-centric workloads, so they need to be used with caution.

:::info
All functions and operators used in an index definition must be “immutable”. An immutable function or operator always returns the same output when given the same inputs, regardless of any external factors, such as the current time, state of other tables in the database, or other environmental variables. This is a requirement for functions and operators used in the definition of functional indexes. To learn more about creating indexes in a PostgreSQL database, visit the official documentation [here](https://www.postgresql.org/docs/current/sql-createindex.html).
:::

## Managing Functional Indexes is easy with Atlas​​

Atlas is an open-source project which allows us to manage our database using a simple and easy-to-understand declarative syntax (similar to Terraform), as well as SQL.

:::info
If you are just getting started, install the latest version of Atlas using the guide to [set up Atlas](/getting-started/).
:::

#### Example
We will first use the `atlas schema inspect` command to get an HCL representation of the table we created earlier (without any indexes) by using the Atlas CLI:

```
atlas schema inspect -u "postgres://postgres:pass@localhost:5432/scorecard?sslmode=disable" > schema.hcl
```

```hcl title=schema.hcl
table "scorecard" {
  schema = schema.public
  column "id" {
    null = false
    type = serial
  }
  column "name" {
    null = true
    type = character_varying(255)
  }
  column "science" {
    null = true
    type = integer
  }
  column "mathematics" {
    null = true
    type = integer
  }
  column "language" {
    null = true
    type = integer
  }
  column "social_science" {
    null = true
    type = integer
  }
  column "arts" {
    null = true
    type = integer
  }
  primary_key {
    columns = [column.id]
  }
}
schema "public" {
}
```

Now, let's add the following index definition to the file:

```hcl title=schema.hcl {34-38}
table "scorecard" {
  schema = schema.public
  column "id" {
    null = false
    type = serial
  }
  column "name" {
    null = true
    type = character_varying(255)
  }
  column "science" {
    null = true
    type = integer
  }
  column "mathematics" {
    null = true
    type = integer
  }
  column "language" {
    null = true
    type = integer
  }
  column "social_science" {
    null = true
    type = integer
  }
  column "arts" {
    null = true
    type = integer
  }
  primary_key {
    columns = [column.id]
  }
  index "functional_idx" {
    on {
      expr = "(((science + mathematics) / 2))"
    }
  }
}
schema "public" {
}
```

Save the file and apply the schema changes on the database by using the following command:

```
atlas schema apply --url "postgres://postgres:pass@localhost:5432/scorecard?sslmode=disable" --to "file://schema.hcl"
```

Atlas generates the necessary SQL statements to add the new functional index to the database schema. Press Enter while the `Apply` option is highlighted to apply the changes:

```console
-- Planned Changes:
-- Create index "functional_idx" to table: "scorecard"
// highlight-next-line-info
CREATE INDEX "functional_idx" ON "public"."scorecard" ((((science + mathematics) / 2)));
Use the arrow keys to navigate: ↓ ↑ → ← 
? Are you sure?: 
  ▸ Apply
    Abort
```

To verify that our new index was created, run the following command:

```
atlas schema inspect -u "postgres://postgres:pass@localhost:5432/scorecard?sslmode=disable" | grep -A4 index
```

```hcl title=Output
  index "functional_idx" {
    on {
      expr = "(((science + mathematics) / 2))"
    }
  }
```

Amazing! Our new index sci_math_avg_idx with the expression `(science + mathematics) / 2` is now created!

## Wrapping up​​
In this guide, we demonstrated how using functional indexes with an appropriate expression becomes a very crucial skill in optimizing query performance with combinations of certain expressions, functions and/or conditions.

## Need More Help?​​
[Join the Atlas Discord Server](https://discord.com/invite/zZ6sWVg6NT) for early access to features and the ability to provide exclusive feedback that improves your Database Management Tooling.

[Sign up](https://www.getrevue.co/profile/ariga) to our newsletter to stay up to date about [Atlas](https://atlasgo.io/), our OSS database schema management tool, and our cloud platform [Atlas Cloud](https://atlasgo.cloud/).