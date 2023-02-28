---
id: functional-indexes
title: Functional Indexes in SQLite
slug: /guides/sqlite/functional-indexes
---

### What are functional key parts?​​
A functional index is an index in a database that is based on the result of a function applied to one or more columns of a table. Functional key parts can index expression values. Hence, functional key parts enable indexing values that are not stored directly in the table itself.

### When are functional indexes helpful?​
Functional indexes are helpful in SQLite databases when the query retrieves data based on the result of a function. It can be useful when the function requires high computational power to execute.

By creating an index based on the result of the function used in the query, the database can quickly find the matching rows based on the function output, rather than having to perform a full table scan and do the necessary computation. This can lead to significant improvements in query performance, especially in large databases with complex queries.

Some common use cases for functional indexes in SQLite include case-insensitive searching, date calculations, and full-text search. However, SQLite has some limitations with functional indexes, such as the function used in the index must be deterministic and the function must always return the same result for the same input.

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

### Example​​
Let’s create a table containing a list of students and the marks they received in each subject with the following command:

```sql
CREATE TABLE scorecard (
    id INTEGER PRIMARY KEY,
    name TEXT,
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
+----+-------------------+---------+-------------+----------+----------------+------+
| id |       name        | science | mathematics | language | social_science | arts |
+----+-------------------+---------+-------------+----------+----------------+------+
| 1  | Geoffrey Franklin | 84      | 72          | 24       | 92             | 47   |
| 2  | Emily Wood        | 47      | 9           | 87       | 63             | 65   |
| 3  | Monica Castillo   | 59      | 23          | 46       | 74             | 69   |
| 4  | John Long         | 40      | 37          | 47       | 35             | 75   |
| 5  | Audrey Stark      | 36      | 52          | 90       | 74             | 31   |
.
.
| 1000001 | Frank Johnson   | 9       | 95          | 83       | 55             | 77   |
+---------+-----------------+---------+-------------+----------+----------------+------+
```

:::info
You can also beautify tables in SQLite like shown above, by using the command `.mode table`
:::

We do not have any indexes other than the primary index on the `id` column. Now, suppose we want information about the top ten students who scored the best average in `math` and `science` combined. Let's query that data with the following command:

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
+--------+-----------------+---------------+
|   id   |      name       | average_score |
+--------+-----------------+---------------+
| 21583  | Tammy Daniels   | 100           |
| 26175  | Tammy Gibson    | 100           |
| 28559  | Anthony Carroll | 100           |
| 32260  | Lauren Jackson  | 100           |
| 60446  | Ryan Allison    | 100           |
| 92806  | Michelle Lewis  | 100           |
| 107326 | Gary Jones      | 100           |
| 117537 | Joanne Cobb     | 100           |
| 124260 | Adam Lucas      | 100           |
| 128316 | Michael Kelly   | 100           |
+--------+-----------------+---------------+
```

Now, let's see how the query performs with the following command:

```sql
EXPLAIN QUERY PLAN
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
|--SCAN scorecard
`--USE TEMP B-TREE FOR ORDER BY
```

:::info
The `EXPLAIN QUERY PLAN` command is used to obtain a high-level description of the strategy or plan that SQLite uses to implement a specific SQL query. To learn more about it, visit the official documentation [here](https://www.sqlite.org/eqp.html).
:::

Without an index, the query planner has to scan the entire table to find the rows that match the query condition, and then sort the result according to the `ORDER BY` clause. This can be a very time-consuming operation, especially for large tables. Additionally, when the planner has to use a temporary B-tree to perform the sort, it has to perform an added operation, which can slow down the query even further.

As we are making use of columns `science` and `mathematics`, let’s try to optimize the query performance by indexing these columns with the following command:

```sql
CREATE INDEX
   normal_index
ON
   scorecard (science, mathematics);
```

Awesome, our index is now created! Now, let's run the following query again and check the plan, with the same command:

```sql
EXPLAIN QUERY PLAN
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
|--SCAN scorecard
`--USE TEMP B-TREE FOR ORDER BY
```

There is no change in query plan, because our query hasn’t used the index we have created. This is where having knowledge about functional indexes becomes essential. Now, let's try to optimize the query by creating a functional index with the expression `((science + mathematics)/2)` from our query, with the following command:

```sql
CREATE INDEX
   functional_idx
ON
   scorecard((science + mathematics)/2);
```

Let's check again the plan with the same command:

```sql
EXPLAIN QUERY PLAN
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
`--SCAN scorecard USING INDEX functional_idx
```

Awesome! Our query has made use of the index in order to retrieve the results. 

After creating the index, the query performance is expected to improve significantly, because the query planner will use the index to perform the search instead of scanning the entire table. This results in fewer disk reads and a reduction in the number of rows that need to be processed, which leads to a faster query execution time.

It is worth noting that while functional indexes can improve performance for certain queries, they can also introduce overhead for insert and update operations, since the index needs to be updated every time the table is modified. This can slow down write-centric workloads,, so they need to be used with caution.

:::info
To learn more about creating indexes with expressions in SQLite, visit the official documentation [here](https://www.sqlite.org/expridx.html)
:::

## Managing Functional Indexes is easy with Atlas​​

Atlas is an open-source project which allows us to manage our database using a simple and easy-to-understand declarative syntax (similar to Terraform), as well as SQL.

:::info
If you are just getting started, install the latest version of Atlas using the guide to [set up Atlas](/getting-started/).
:::

#### Example
We will first use the atlas schema inspect command to get an HCL representation of the table we created earlier (without any indexes) by using the Atlas CLI:

```console
atlas schema inspect -u "sqlite://scorecard.db" > schema.hcl
```
```hcl title=schema.hcl
table "scorecard" {
  schema = schema.main
  column "id" {
    null = true
    type = integer
  }
  column "name" {
    null = true
    type = text
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
schema "main" {
}
```

Now, let's add the following index definition to the file:
```hcl title=schema.hcl {34-38}
table "scorecard" {
  schema = schema.main
  column "id" {
    null = true
    type = integer
  }
  column "name" {
    null = true
    type = text
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
      expr = "((science + mathematics)/2)"
    }
  }
}
schema "main" {
}
```

Save the file and apply the schema changes on the database by using the following command:

```console
atlas schema apply --url "sqlite://scorecard.db" --to "file://schema.sql"
```

Atlas generates the necessary SQL statements to add the new functional index to the database schema. Press Enter while the `Apply` option is highlighted to apply the changes:

```console
-- Planned Changes:
-- Create index "functional_idx" to table: "scorecard"
// highlight-next-line-info
CREATE INDEX `functional_idx` ON `scorecard` (((science + mathematics)/2));
Use the arrow keys to navigate: ↓ ↑ → ← 
? Are you sure?: 
  ▸ Apply
    Abort
```

To verify that our new index was created, run the following command:

```console
atlas schema inspect -u "sqlite://scorecard.db" | grep -A4 index
```

```hcl title=Output
  index "functional_idx" {
    on {
      expr = "((science + mathematics)/2)"
    }
  }
```

Amazing! Our new index `sci_math_avg_idx` with the expression `(science + mathematics) / 2` is now created!

## Wrapping up​​
In this guide, we demonstrated how using functional indexes with an appropriate expression becomes a very crucial skill in optimizing query performance with combinations of certain expressions, functions and/or conditions.

## Need More Help?​​
[Join the Atlas Discord Server](https://discord.com/invite/zZ6sWVg6NT) for early access to features and the ability to provide exclusive feedback that improves your Database Management Tooling.

[Sign up](https://www.getrevue.co/profile/ariga) to our newsletter to stay up to date about [Atlas](https://atlasgo.io/), our OSS database schema management tool, and our cloud platform [Atlas Cloud](https://atlasgo.cloud/).
