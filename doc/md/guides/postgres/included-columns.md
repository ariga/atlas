---
id: included-columns
title: Indexes with Included Columns in PostgreSQL
slug: /guides/postgres/included-columns
---

With PostgreSQL, we can create *covering indexes* using the `INCLUDE` clause, which are types of indexes that specify a
list of columns to be included in the index as non-key columns. If used correctly, indexes with included columns improve
performance and reduce total costs.

### Basic PostgreSQL syntax for using `INCLUDE` clause with an index:

```sql
CREATE [UNIQUE] INDEX index_name
ON table_name(key_column_list)
INCLUDE(included_column_list);
```

### How do they work?

In PostgreSQL, a B-Tree index creates a multi-level tree structure where each level can be used as a doubly-linked list
of pages. Leaf pages are those at the lowest level of a tree, that point to rows of tables.

With *covering indexes*, records of the columns mentioned in the `INCLUDE` clause are included in the leaf pages of the
B-Tree as "payload" and are not part of the search key.

:::info
Each index is stored separately from the table's main data area, which in PostgreSQL this is known as the table's _heap_.
To learn more about the PostgreSQL B-tree index structure and *covering indexes*, visit the documentation:
1. [B-Tree structure](https://www.postgresql.org/docs/current/btree-implementation.html#BTREE-STRUCTURE).
2. [Covering indexes](https://www.postgresql.org/docs/current/indexes-index-only-scans.html).
:::

### When do we need them?

Let's demonstrate an example where an index with an `INCLUDE` clause may be useful, by contrasting it with a unique index without an `INCLUDE` clause.

First, create a table with the following command:

```sql
DROP TABLE IF EXISTS "bankdb";

CREATE TABLE "bankdb" (
  id SERIAL PRIMARY KEY,
  savings varchar(100),
  first_name varchar(255),
  last_name varchar(255),
  email varchar(255),
  bank varchar(34)
);
```

Here is how a portion of the table might look like after inserting values:

```console title="Output"
-[ RECORD 1 ]--------------------------------------
id         | 1
savings    | 28 497
first_name | Amena
last_name  | Gardner
email      | a_gardner@aol.edu
bank       | GE77159307648978112812
-[ RECORD 2 ]--------------------------------------
id         | 2
savings    | 71 279
first_name | Joan
last_name  | Kaufman
email      | k-joan3559@google.couk
bank       | DK8023212366607361

.
.
.
-[ RECORD 1499 ]------------------------------------
id         | 1499
savings    | 4 880
first_name | Ramona
last_name  | Wilkins
email      | r.wilkins@google.net
bank       | BA928132235277210873
-[ RECORD 1500 ]------------------------------------
id         | 1500
savings    | 69 873
first_name | Imani
last_name  | Noble
email      | imaninoble@hotmail.net
bank       | BG45LBAX41796917951361
```

Now, suppose we want to find the ID of a user by their email address. Let’s check the performance of the query with a WHERE clause without any index, with the following command:

```sql
EXPLAIN ANALYZE
SELECT
    first_name,
    last_name,
    email
FROM
    "bankdb"
WHERE
    email = 'd-abbott3425@google.edu';
```

```console title="Output"
QUERY PLAN
----------------
 Seq Scan on bankdb  (cost=0.00..38.75 rows=1 width=37) (actual time=0.180..0.181
rows=1 loops=1)
   Filter: ((email)::text = 'd-abbott3425@google.edu'::text)
   Rows Removed by Filter: 1499
 Planning Time: 0.053 ms
 Execution Time: 0.195 ms
(5 rows)

Time: 0.626 ms
```

Notice that the total cost is 38.75 units. If we want to use a unique index to accelerate the query, we can create it on the `email` column with the following command:

```sql
CREATE UNIQUE INDEX emails_idx
ON bankdb(email);
```
```console title="Output"
CREATE INDEX
Time: 10.316 ms
```

Now, let's check the performance of querying data of first and last names based on their email addresses, with the following command:

```sql
EXPLAIN ANALYZE
SELECT
    first_name,
    last_name,
    email
FROM
    "bankdb"
WHERE
    email = 'd-abbott3425@google.edu';
```
```console title="Output"
QUERY PLAN
----------------------------------
 Index Scan using emails_idx on bankdb  (cost=0.28..8.29 rows=1 width=37) (actual
time=0.200..0.203 rows=1 loops=1)
   Index Cond: ((email)::text = 'd-abbott3425@google.edu'::text)
 Planning Time: 0.081 ms
 Execution Time: 0.259 ms
(4 rows)

Time: 1.470 ms
```

Notice that the total cost is now 8.29 units. The performance of the query has improved by creating a primary key index on `email` column, compared to 38.75 units without using any index. The engine still has to fetch the `first_name` and `last_name` columns from the table (also known as "heap fetches").
Let's drop the existing index to demonstrate the next section in the article:

```sql
DROP INDEX emails_idx;
```
```console title="Output"
DROP INDEX
Time: 3.856 ms
```

Suppose we want to accelerate the same query using the `INCLUDE` clause.
In the following command, we will create an index with an `INCLUDE` clause that precisely covers `first_name` and `last_name` columns which are part of the query for which we are trying to improve performance.

```sql
CREATE UNIQUE INDEX emails_idx
ON bankdb(email)
INCLUDE(first_name,last_name);
```

```console title="Output"
CREATE INDEX
Time: 8.942 ms
```

Now, let's check the performance of querying data of first and last names based on their email addresses, with the following command:

```sql
EXPLAIN ANALYZE
SELECT
    first_name,
    last_name,
    email
FROM
    "bankdb"
WHERE
    email = 'd-abbott3425@google.edu';
```
```console title="Output"
QUERY PLAN
---------------------------------------
 Index Only Scan using emails_idx on bankdb  (cost=0.28..4.29 rows=1 width=37) (ac
tual time=0.228..0.231 rows=1 loops=1)
   Index Cond: (email = 'd-abbott3425@google.edu'::text)
   Heap Fetches: 0
 Planning Time: 0.233 ms
 Execution Time: 0.283 ms
(5 rows)
```
Notice that the total cost is now 4.29, which is significantly lower, compared to 8.29 which we got while using a unique index without the `INCLUDE` clause. We were able to reduce the total cost because the query only scanned the index in order to get the data. As a result, `heap fetches` is also zero, which means the query does not access any tables to retrieve the records.

:::info
You might be wondering why we didn’t just use `CREATE INDEX ON bankdb(email,first_name,last_name)` instead of using the `INCLUDE` clause. One of the advantages of using the `INCLUDE` clause is having fewer levels in a  B-tree. All `INCLUDE` columns are stored in the doubly linked list of the B-tree index.
:::

### Advantages of using Indexes with an INCLUDE clause:
1. The B-tree has fewer levels because they do not contain include columns
2. Greatly improves performance
3. Has the ability to return the contents of non-key columns without having to visit the index's table

### Limitation of using Indexes with included columns
- Expressions are not supported as included columns since they cannot be used in index-only scans.

## Managing indexes with included columns is easy with Atlas

Managing indexes and database schemas in PostgreSQL can be confusing and error-prone. Atlas is an open-source project which allows us to manage our database using a simple and easy-to-understand declarative syntax (similar to Terraform). We will now learn how to manage indexes with included columns using Atlas.

:::info
If you are just getting started, install the latest version of Atlas using the guide to [setting up Atlas](https://atlasgo.io/cli/getting-started/setting-up).
:::

### Managing Indexes with included columns in Atlas

We will first use the `atlas schema inspect` command to get an HCL representation of the table which we created earlier by using the Atlas CLI:

```console
atlas schema inspect -u "postgres://postgres:mysecretpassword@localhost:5432/bankdb?sslmode=disable" > schema.hcl
```
```hcl title="schema.hcl"
table "bankdb" {
  schema = schema.public
  column "id" {
    null = false
    type = serial
  }
  column "savings" {
    null    = true
    type    = varchar(100)
  }
  column "first_name" {
    null    = true
    type    = varchar(255)
  }
  column "last_name" {
    null    = true
    type    = varchar(255)
  }
  column "email" {
    null    = true
    type    = varchar(255)
  }
  column "bank" {
    null = true
    type = varchar(34)
  }
  primary_key {
    columns = [column.id]
  }
}
schema "public" {
}
```

Now, lets add the following index definition to the file:

```hcl
  index "emails_idx" {
    unique  = true
    columns = [column.email]
    include = [column.first_name, column.last_name]
  }
```

Save and apply the schema changes on the database by using the `apply` command:

```console
atlas schema apply -u "postgres://postgres:mysecretpassword@localhost:5432/bankdb?sslmode=disable" -f schema.hcl
```

Atlas generates the necessary SQL statements to add the new index to the database schema. Press Enter while the Apply option is highlighted to apply the changes:

```console title="Output"
-- Planned Changes:
-- Create index "emails_idx" to table: "bankdb"
CREATE UNIQUE INDEX "emails_idx" ON "public"."bankdb" ("email") INCLUDE ("first_name", "last_name")
Use the arrow keys to navigate: ↓ ↑ → ←
? Are you sure?:
  ▸ Apply
    Abort
```

To verify that our new index was created, open the database command line tool from the previous step and run:

```console
\d bankdb;
```

```console title="Output"
                                    Table "public.bankdb"
 Column     |          Type         |Collation| Nullable |              Default

------------+-----------------------+---------+----------+------------------------------------
 id         | integer               |         | not null | nextval('bankdb_id_seq'::regclass)
 savings    | character varying(100)|         |          | NULL::character varying
 first_name | character varying(255)|         |          | NULL::character varying
 last_name  | character varying(255)|         |          | NULL::character varying
 email      | character varying(255)|         |          | NULL::character varying
 bank       | character varying(34) |         |          |
Indexes:
    "bankdb_pkey" PRIMARY KEY, btree (id)
    "emails_idx" UNIQUE, btree (email) INCLUDE (first_name, last_name)
```
Amazing! Our new index with included columns is now created!

## Conclusion

In this section, we learned about PostgreSQL indexes with included columns and how we can easily create them in our database by using Atlas.

## Need More Help?

[Join the Ariga Discord Server](https://discord.com/invite/zZ6sWVg6NT) for early access to features and the ability to provide exclusive feedback that improves your Database Management Tooling. [Sign up](https://atlasnewsletter.substack.com/) to our newsletter to stay up to date about [Atlas](https://atlasgo.io), our OSS database schema management tool, and our cloud platform [Atlas Cloud](https://atlasgo.cloud).
