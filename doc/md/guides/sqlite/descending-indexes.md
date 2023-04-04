---
id: descending-indexes
title: Descending Indexes in SQLite
slug: /guides/sqlite/descending-indexes
---

### What are descending indexes?​
Descending indexes are indexes where key values are stored in descending order. Descending indexes can be helpful in SQLite when queries involve ordering the results in descending order. 

### When are descending indexes helpful?​
For example, if a query uses an `ORDER BY` clause to sort the results of a query in descending order, then a descending index can improve the performance of that query significantly.

### Syntax​
Here is how you can create a descending index:

```sql
CREATE INDEX index_name ON table_name(column_name DESC);
```

### Example​
Let’s create a table which represents data of an ISP’s subscribers along with their email addresses and broadband data usage with the following command:

```sql
CREATE TABLE telecom_data (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 email_address varchar(255),
 user_name varchar(255),
 megabytes_used bigint
);
```

Here is how a portion of the table might look like after inserting values:

```sql
SELECT * FROM telecom_data
```
```console title=Output
+----+-----------------------------+--------------------+----------------+
| id |        email_address        |     user_name      | megabytes_used |
+----+-----------------------------+--------------------+----------------+
| 1  | richard85@example.org       | robertasanchez     | 3629           |
| 2  | christinahansen@example.org | dawnmcdonald       | 8182           |
| 3  | zlynch@example.org          | dtorres            | 4768           |
| 4  | erica21@example.net         | zbrown             | 9130           |
| 5  | lynchanthony@example.com    | osmith             | 3464           |
| 6  | reillycaroline@example.org  | tcarrillo          | 7004           |
| 7  | romanamy@example.com        | wmitchell          | 8836           |
| 8  | maryperez@example.com       | hernandezjohnathan | 5847           |
| 9  | qherman@example.net         | scottmonroe        | 1957           |
| 10 | james55@example.com         | caitlin78          | 8135           |
.
.
.
| 1000001 | huynhadam@example.com       | robertgaines   | 2558           |
+---------+-----------------------------+----------------+----------------+
```

:::info
You can also beautify tables in SQLite like shown above, by using the command `.mode table`
:::

We do not have any indexes other than the primary index on the `id` column. Now, suppose we want information about the top 10 subscribers with maximum usage, but in descending order. Let's query that data with the following command:

```sql
SELECT * FROM telecom_data ORDER BY megabytes_used DESC LIMIT 10;
```
```console title=Output
+--------+-----------------------------+------------------+----------------+
|   id   |        email_address        |    user_name     | megabytes_used |
+--------+-----------------------------+------------------+----------------+
| 68549  | clarkealexandra@example.net | uvega            | 10000          |
| 85006  | rjacobs@example.org         | alexandra15      |  9999          |
| 95969  | nashkathy@example.com       | fmendoza         |  9998          |
| 106820 | ramirezshelly@example.org   | hjohnson         |  9998          |
| 117508 | bautistacharles@example.net | ufox             |  9997          |
| 142507 | christine64@example.net     | lawrencerobinson |  9996          |
| 143542 | gary28@example.net          | bgoodman         |  9995          |
| 151621 | gilescatherine@example.net  | wendyroberts     |  9994          |
| 155916 | ohorn@example.net           | btucker          |  9991          |
| 157456 | imurphy@example.org         | douglashensley   |  9990          |
+--------+-----------------------------+------------------+----------------+
```

Now, let's see the query plan with the following command:

```sql
EXPLAIN QUERY PLAN
SELECT * FROM telecom_data ORDER BY megabytes_used DESC LIMIT 10;
```

```console title=Output
QUERY PLAN
--SCAN telecom_data
--USE TEMP B-TREE FOR ORDER BY
```

:::info
The `EXPLAIN QUERY PLAN` command is used to obtain a high-level description of the strategy or plan that SQLite uses to implement a specific SQL query. To learn more about it, visit the official documentation [here](https://www.sqlite.org/eqp.html).
:::

Without an index, the query planner has to scan the entire table to find the rows that match the query condition, and then sort the result according to the `ORDER BY` clause. This can be a very time-consuming operation, especially for large tables. Additionally, when the planner has to use a temporary B-tree to perform the sort, it has to perform an added operation, which can slow down the query even further.

Now, let's optimize the query by using a descending index. We will create a descending index on the `megabytes_used` column with the following command:

```sql
CREATE INDEX fastscan_idx ON telecom_data(megabytes_used DESC);
```

Now, let's run the following query again and check the plan:

```sql
EXPLAIN QUERY PLAN
SELECT * FROM telecom_data ORDER BY megabytes_used DESC LIMIT 10;
```

```console title=Output
QUERY PLAN
`--SCAN telecom_data USING INDEX fastscan_idx
```

Awesome! Our query has made use of the index in order to retrieve the results. 

After creating the index, the query performance is expected to improve significantly, because the query planner will use the index to perform the search instead of scanning the entire table. This results in fewer disk reads and reduces the number of rows that need to be processed, which leads to a faster query execution time.

:::info
To learn more about creating indexes in SQLite, visit the official documentation [here](https://www.sqlite.org/lang_createindex.html)
:::

We have seen that creating a descending index is a smart choice when using queries with `ORDER BY` clauses. Now, let's see how we can easily manage descending indexes in an SQLite database using Atlas.

## Managing Descending Indexes is easy with Atlas​

Atlas is an open-source project which allows us to manage our database using a simple and easy-to-understand declarative syntax (similar to Terraform), as well as SQL.

:::info
If you are just getting started, install the latest version of Atlas using the guide to [set up Atlas](/getting-started/).
:::

#### Example
We will first use the `atlas schema inspect` command to get an HCL representation of the table we created earlier (without any indexes) by using the Atlas CLI:

```console
atlas schema inspect -u "sqlite://telecom_data.db" > schema.hcl
```
```hcl title="schema.hcl"
schema "main" {}

table "telecom_data" {
  schema = schema.main
  column "id" {
    null           = true
    type           = integer
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
```

Now, let’s add the following index definition to the file:

```hcl title="schema.hcl" {25-30}
schema "main" {}

table "telecom_data" {
  schema = schema.main
  column "id" {
    null           = true
    type           = integer
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
atlas schema apply --url "sqlite://telecom_data.db" --to "file://schema.hcl"
```

Atlas generates the necessary SQL statements to add the new descending index to the database schema. Press Enter while the `Apply` option is highlighted to apply the changes:

```console
-- Planned Changes:
-- Create index "fastscan_idx" to table: "telecom_data"
// highlight-next-line-info
CREATE INDEX `fastscan_idx` ON `telecom_data` (`megabytes_used` DESC);
Use the arrow keys to navigate: ↓ ↑ → ← 
? Are you sure?: 
  ▸ Apply
    Abort
```

To verify that our new index was created, run the following command:

```console
atlas schema inspect -u "sqlite://telecom_data.db" | grep -A5 index
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

## Wrapping up
In this guide, we demonstrated how to create and use descending indexes in SQLite to optimize queries with the ORDER BY clause, and how we can use Atlas to easily manage descending indexes in a SQLite database.

## Need More Help?
[Join the Atlas Discord Server](https://discord.com/invite/zZ6sWVg6NT) for early access to features and the ability to provide exclusive feedback that improves your Database Management Tooling.

[Sign up](https://www.getrevue.co/profile/ariga) to our newsletter to stay up to date about [Atlas](https://atlasgo.io/), our OSS database schema management tool, and our cloud platform [Atlas Cloud](https://atlasgo.cloud/).
