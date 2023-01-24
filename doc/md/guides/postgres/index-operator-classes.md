---
id: index-operator-classes
title: Index operator classes in PostgreSQL
slug: /guides/postgres/index-operator-classes
---

### What is an operator class?

An operator class identifies the operators to be used by the index for the indexed column. Operator classes can be specified for each column of an index in an index definition.

### Syntax

Here is how you can specify an operator class for a column in an index definition:

```sql
CREATE INDEX
    name
ON
    table (column opclass [ ( opclass_options ) ] [sort options] [, ...]);
```

### When do we need operator classes?

The main reason for having operator classes is that for some data types, there could be more than one meaningful index behavior. The operator class determines the basic sort ordering. In most cases, the default operator class is usually sufficient. Let’s see it in action.

### Example:

Let’s create a table which represents data of an ISP’s subscribers along with their email addresses and outstanding payments with the following command:

```sql
CREATE TABLE "internet_provider" (
  id SERIAL PRIMARY KEY,
  subscriber_name varchar(255),
  email_address varchar(255),
  payment_pending varchar(100),
  active_user varchar(255)
);
```

Here is how a portion of the table might look like after inserting values:

```sql
SELECT
        *
FROM
        internet_provider
```
```console title="Output"
 id | subscriber_name  |      email_address      | payment_pending | active_user
----+------------------+-------------------------+-----------------+-------------
  0 | Abel Warren      | havb@example.com        | 730             | false
  1 | Erick Valentine  | riuee@example.com       | 70              | false
  2 | Janice Payne     | dvtub193@example.com    | 67              | false
  3 | Gretchen Mason   | tmug.xfhq@example.com   | 767             | false
  4 | Lawanda Noble    | qpoy03@example.com      | 227             | true
  5 | Robbie Baird     | wdit@example.com        | 659             | true
  6 | Carla Compton    | qacaf.kznyx@example.com | 805             | false
  7 | Heath Stafford   | mehs271@example.com     | 29              | false
  8 | Kendra Stevenson | jcsvp57@example.com     | 810             | true
  9 | Brandie Chase    | abwf.dape@example.com   | 944             | false
.
.
.
   id   | subscriber_name  |      email_address      | payment_pending | active_user
--------+------------------+-------------------------+-----------------+-------------
 999999 | James Strong     | jxgi3@example.com       | 788             | true
 999998 | Virginia Ballard | rkdzo0@example.com      | 598             | false
 999997 | Shirley Bright   | wawuh02@example.com     | 619             | false
 999996 | Vicky Hull       | wobm.fcwsdq@example.com | 390             | false
 999995 | Juan Pittman     | tmuq.pyno@example.com   | 263             | false
 999994 | Gordon Hawkins   | litvbi.nral@example.com | 605             | true
 999993 | Demond Bright    | byvd41@example.com      | 221             | false
 999992 | Tara Lowe        | eiyul4@example.com      | 871             | false
 999991 | Kenny Daniel     | rbqt328@example.com     | 580             | true
 999990 | Alexandra Frank  | gfkw8@example.com       | 890             | true
(1000000 rows)
```

We do not have any indexes other than the primary index on the `id` column.

Now, let’s assume that we are not aware of the usage of operator classes in indexes just yet. We want to accelerate queries involving patterns matching expressions with a `LIKE` operator in order to search a name in the `subscriber_name` column. In this case, we would create an index on the column `subscriber_name` with the following command:

```sql
CREATE INDEX
    internet_provider_idx
ON
    internet_provider(subscriber_name);
```

```console title="Output"
CREATE INDEX
Time: 3911.261 ms (00:03.911)
```

Awesome! Our index is now created on the `subscriber_name` column. Now, suppose that we want to search for a subscriber whose registered name begins with “Shirley C”. We can create such a query with the use of a `WHERE` clause and a `LIKE` operator. Let’s check the performance and plan of this query with the following command:

```sql
EXPLAIN ANALYZE
SELECT
        *
FROM
        internet_provider
WHERE
        subscriber_name LIKE 'Shirley C%'
```

:::info
The EXPLAIN command is used for understanding the performance of a query. You can learn more about usage of the EXPLAIN command with the ANALYZE option [here](https://www.postgresql.org/docs/14/using-explain.html#USING-EXPLAIN-ANALYZE).
:::

```console title="Output"
                                                             QUERY PLAN
------------------------------------------------------------------------------------------------------------------------------------
 Gather  (cost=1000.00..15874.33 rows=100 width=44) (actual time=5.956..227.339 rows=78 loops=1)
   Workers Planned: 2
   Workers Launched: 2
   ->  Parallel Seq Scan on internet_provider  (cost=0.00..14864.33 rows=42 width=44) (actual time=10.923..211.886 rows=26 loops=3)
         Filter: ((subscriber_name)::text ~~ 'Shirley C%'::text)
         Rows Removed by Filter: 333307
 Planning Time: 4.955 ms
 Execution Time: 228.132 ms
(8 rows)

Time: 243.573 ms
```
Notice that the `internet_provider_idx` index that we created was not used in order to execute this query. Instead, the `Parallel Seq Scan` operation was performed. As a result, the total execution time and cost are still too high.

:::info
In a parallel sequential scan, the table's blocks will be divided into ranges and shared among the cooperating processes. Each worker process will complete the scanning of its given range of blocks before requesting an additional range of blocks. To learn more about Parallel Plans in PostgreSQL, visit the official documentation [here](https://www.postgresql.org/docs/current/parallel-plans.html).
:::

Now, you might be wondering why the index that we created was not being used in the execution of this query. This is when having knowledge about the usage of operator classes becomes important.

As mentioned earlier, an operator class identifies the operators to be used by the index for the indexed column. Let’s see this in action by specifying an operator class in our definition with the following commands:

```sql
DROP INDEX internet_provider_idx;

DROP INDEX
Time: 43.279 ms

CREATE INDEX
    internet_provider_idx
ON
    internet_provider(subscriber_name varchar_pattern_ops)
```

```console title="Output"
CREATE INDEX
Time: 2375.380 ms (00:02.375)
```

This time, we specified an operator class `varchar_pattern_ops` in our index definition. `varchar_pattern_ops` is a built-in operator class which supports B-tree indexes on the data-type `varchar`. Let’s check the performance and plan of the query we previously used with the following command:

```sql
EXPLAIN ANALYZE
SELECT
        *
FROM
        internet_provider
WHERE
        subscriber_name LIKE 'Shirley C%'
```
```console title="Output"
                                                                 QUERY PLAN
---------------------------------------------------------------------------------------------------------------------------------------------
 Index Scan using internet_provider_idx on internet_provider  (cost=0.42..8.45 rows=100 width=44) (actual time=0.025..0.131 rows=78 loops=1)
   Index Cond: (((subscriber_name)::text ~>=~ 'Shirley C'::text) AND ((subscriber_name)::text ~<~ 'Shirley D'::text))
   Filter: ((subscriber_name)::text ~~ 'Shirley C%'::text)
 Planning Time: 0.103 ms
 Execution Time: 0.153 ms
(5 rows)
```

Amazing! This time, Index Scan was performed using `internet_provider_idx`. As a result, the cost, planning time, and execution time for our query have reduced significantly, as we expected.

The previous index (without a specified operator class) could have been helpful while executing queries with WHERE clauses with operators such as  <, <=, >, or >=. Though, the same index could not be utilized when executing queries with WHERE clauses with a `LIKE` operator.

:::info
In our example, we saw that in some data types, there could be more than one meaningful index behavior, and we need to specify an operator class in the index definition to accelerate certain queries. An operator class is actually just a subset of a larger structure called an “operator family”. To learn more about Operator Classes and Operator Families, visit the official documentation [here](https://www.postgresql.org/docs/current/indexes-opclass.html).
:::

### Managing indexes with operator classes is easy with Atlas

Atlas is an open-source project which allows us to manage our database using a simple and easy-to-understand declarative syntax (similar to Terraform).

:::info
If you are just getting started, install the latest version of Atlas using the guide to [setting up Atlas](https://atlasgo.io/cli/getting-started/setting-up).
:::

#### Managing Operator Classes with Atlas

We will first use the `atlas schema inspect` command to get an [HCL](https://atlasgo.io/guides/ddl#hcl) representation of the table we created earlier (without any indexes other than primary index on `id` column) by using the Atlas CLI:

```console
atlas schema inspect -u "postgres://postgres:mysecretpassword@localhost:5432/internet_provider_db?sslmode=disable" > schema.hcl
```
```hcl title="schema.hcl"
table "internet_provider" {
  schema = schema.public
  column "id" {
    null = false
    type = serial
  }
  column "subscriber_name" {
    null = false
    type = varchar(255)
  }
  column "email_address" {
    null = false
    type = varchar(255)
  }
  column "payment_pending" {
    null = false
    type = varchar(100)
  }
  column "active_user" {
    null = false
    type = varchar(255)
  }
  primary_key {
    columns = [column.id]
  }
}
schema "public" {
}
```

Now, let's add the following index definition to the file:

```hcl title="schema.hcl"
  index "internet_provider_idx" {
    type = BTREE
    on {
      column = column.subscriber_name
      ops    = varchar_pattern_ops
    }
  }
```

Save and apply the schema changes on the database by using the `apply` command:

```console
atlas schema apply -u "postgres://postgres:mysecretpassword@localhost:5432/internet_provider_db?sslmode=disable" -f schema.hcl
```

Atlas generates the necessary SQL statements to add the new index to the database schema. Press Enter while the Apply option is highlighted to apply the changes:

```console
-- Planned Changes:
-- Create index "internet_provider_idx" to table: "internet_provider"
CREATE INDEX "internet_provider_idx" ON "public"."internet_provider" ("subscriber_name" varchar_pattern_ops)
Use the arrow keys to navigate: ↓ ↑ → ←
? Are you sure?:
  ▸ Apply
    Abort
```

To verify that our new index was created, open the database command line tool from the previous step and run:

```sql
\d internet_provider
```

```console title="Output"
                                        Table "public.internet_provider"
     Column      |          Type          | Collation | Nullable |                    Default
-----------------+------------------------+-----------+----------+-----------------------------------------------
 id              | integer                |           | not null | nextval('internet_provider_id_seq'::regclass)
 subscriber_name | character varying(255) |           | not null |
 email_address   | character varying(255) |           | not null |
 payment_pending | character varying(100) |           | not null |
 active_user     | character varying(255) |           | not null |
Indexes:
    "internet_provider_pkey" PRIMARY KEY, btree (id)
    "internet_provider_idx" btree (subscriber_name varchar_pattern_ops)
```

Amazing! Our new index `internet_provider_idx` with operator class `varchar_pattern_ops` on `subscriber_name` column is now created!

### Wrapping up

In this guide, we demonstrated how using indexes with an appropriate operator class becomes a very crucial skill in optimizing query performance with combinations of certain clauses and operators.

## Need More Help?​

[Join the Ariga Discord Server](https://discord.com/invite/zZ6sWVg6NT) for early access to features and the ability to provide exclusive feedback that improves your Database Management Tooling.

[Sign up](https://atlasnewsletter.substack.com/) to our newsletter to stay up to date about [Atlas](https://atlasgo.io/), our OSS database schema management tool, and our cloud platform [Atlas Cloud](https://atlasgo.cloud/).
