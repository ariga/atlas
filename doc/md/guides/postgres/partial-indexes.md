---
id: partial-indexes
title: Partial Indexes in PostgreSQL
slug: /guides/postgres/partial-indexes
---

### Overview of Partial Indexes

#### What are Partial Indexes?

With PostgreSQL, users may create _partial indexes_, which are types of indexes that exist on a subset of a table, rather than the entire table itself. If used correctly, partial indexes improve performance and reduce costs, all while minimizing the amount of storage space they take up on the disk.

#### Why do we need them?

Let's demonstrate a case where partial indexes may be useful by contrasting them with a non-partial index. ​​If you have many records in an indexed table, the number of records the index needs to track also grows. If the index grows in size, the disk space needed to store the index itself increases as well.
In many tables, different records are not accessed with uniform frequency. A subset of a table's records might not be searched very frequently or not searched at all. Records take up precious space in your index whether they are queried or not, and are updated when a new entry is added to the field.

Partial indexes come into the picture to filter unsearched values and give you, as an engineer, a tool to index only what's important.

:::info
You can learn more about partial indexes in PostgreSQL [here](https://www.postgresql.org/docs/current/indexes-partial.html)
:::

#### Advantages of using Partial Indexes
In cases where we know ahead of time the access pattern to a table and can reduce the size of an index by making it partial:
1. Response time for SELECT operations is improved because the database searches through a smaller index.
2. On average, response time for UPDATE operations is also improved as the index is not going to get updated in all cases.
3. Index is smaller in size and can fit into RAM more easily.
4. Less space is required to store the index on disk.

#### Basic PostgreSQL syntax for using Partial Index

```sql
CREATE INDEX 
    index_name
ON 
    table_name(column_list)
WHERE 
    condition;
```

#### Example of Non-partial Index vs Partial Index in PostgreSQL

Let's see this in action by creating a table with the following command:

```sql
CREATE TABLE "vaccination_data" (
  id SERIAL PRIMARY KEY,
  country varchar(20),
  title varchar(10),
  names varchar(20),
  vaccinated varchar(3)
);
```

Here is how a portion of the table might look like after inserting values:

```sql
SELECT * FROM vaccination_data;
```
```console title="Output"
 id  |      country       | title |    names    | vaccinated 
-----+--------------------+-------+-------------+------------
   1 | Poland             | Mr.   | Teagan      | No
   2 | Ukraine            | Ms.   | Alden       | No
   3 | Ukraine            | Mr.   | Ima         | No
   4 | Colombia           | Mr.   | Lawrence    | Yes
   5 | Turkey             | Mrs.  | Keegan      | No
   6 | China              | Mrs.  | Kylan       | No
   7 | Netherlands        | Dr.   | Howard      | No
...
 289690 | Russian Federation | Mrs.  | Ray     | Yes
 289689 | Austria            | Dr.   | Lenore  | Yes
 289688 | Sweden             | Dr.   | Walker  | Yes
 289687 | Turkey             | Dr.   | Emerson | No
 289686 | Vietnam            | Dr.   | Addison | Yes

(289686 rows)
```

In the following example, suppose we want a list of doctors from India that have taken the vaccine. If we want to use normal index, we can create it on the “vaccinated” column with the following command:

```sql
CREATE INDEX vaccinated_idx ON vaccination_data(vaccinated);
```
```console title="Output"
CREATE INDEX
Time: 333.891 ms
```

Now, let's check the performance of querying data of doctors from India that have taken the vaccine with the following command:

```sql
EXPLAIN ANALYZE
SELECT  
        *
FROM    
        vaccination_data
WHERE
        vaccinated = 'Yes' AND country = 'India' AND title = 'Dr.';
```
```console title="Output"
QUERY PLAN                                                           
---------------------------------------------------------------------------
 Bitmap Heap Scan on vaccination_data  (cost=758.64..4053.40 rows=699 width=25) (actual time=4.142..16.212 rows=582 loops=1)
   Recheck Cond: ((vaccinated)::text = 'Yes'::text)
   Filter: (((country)::text = 'India'::text) AND ((title)::text = 'Dr.'::text))
   Rows Removed by Filter: 69334
   Heap Blocks: exact=1337
   ->  Bitmap Index Scan on vaccinated_idx  (cost=0.00..758.46 rows=69072 width=0) (actual time=3.940
..3.941 rows=69916 loops=1)
         Index Cond: ((vaccinated)::text = 'Yes'::text)
 Planning Time: 0.188 ms
 Execution Time: 16.292 ms
(9 rows)
```

:::info
The EXPLAIN command is used for understanding the performance of a query. You can learn more about usage of EXPLAIN command with ANALYZE option [here](https://www.postgresql.org/docs/14/using-explain.html#USING-EXPLAIN-ANALYZE)
:::

Notice that total Execution Time is 16.292ms. Also, let's check the index size with the following command:

```sql
SELECT pg_size_pretty(pg_relation_size('vaccinated_idx'));
```
```console title="Output"
 pg_size_pretty 
----------------
 1984 kB
(1 row)
```

Now, suppose we want to accelerate the same query using the partial index. Let's begin by dropping the existing index that we created earlier:

```sql
DROP INDEX vaccinated_idx;
```
```console title="Output"
DROP INDEX
Time: 7.183 ms
```

In the following command, we have created an index with a WHERE clause that precisely describes list of doctors from India that have taken the vaccine.

```sql
CREATE INDEX 
    vaccinated_idx
ON 
    vaccination_data(vaccinated)
WHERE 
    vaccinated = 'Yes' AND country = 'India' AND title = 'Dr.';
```
```console title="Output"
CREATE INDEX
Time: 94.567 ms
```

Notice that the partial index with the WHERE clause is created in 94.567ms, compared to the 333.891ms taken for the non-partial index on the 'vaccinated' column.
Let's check the performance of querying list of doctors from India that have taken the vaccine again, using the following command:

```sql
EXPLAIN ANALYZE
SELECT
        *
FROM    
        vaccination_data
WHERE
        vaccinated = 'Yes' AND country = 'India' AND title = 'Dr.';
```
```console title="Output"
QUERY PLAN                                                              
---------------------------------------------------------------------------
 Index Scan using vaccinated_idx on vaccination_data  (cost=0.15..1455.12 rows=699 width=25) (actual time=0.015..0.704 rows=582 loops=1)
 Planning Time: 0.442 ms
 Execution Time: 0.880 ms
(3 rows)
```

Observe that total execution time has dropped significantly and is now only 0.880ms, compared to 16.292ms achieved by using a non-partial index on the 'vaccinated' column. Once again, let's check the index size with the following command:

```sql
SELECT pg_size_pretty(pg_relation_size('vaccinated_idx'));
```
```console title="Output"
 pg_size_pretty 
----------------
 16 kB
(1 row)
```

As we can observe, the index size for the partial index takes significantly less space (16kb) compared to the non-partial index that we created earlier on the 'vaccinated' column (1984kb).

Here is a summary from our tests:

| Parameter                                    | Non-partial Index       | Partial Index           | Ratio of change(%) |
|:-------------------------------------------- |:----------------------- |:----------------------- |:------------------ |
| Estimated start-up cost                      | 758.64 arbitrary units  | 0.15 arbitrary units    | 99.9% reduced cost |
| Estimated total cost                         | 4053.40 arbitrary units | 1455.12 arbitrary units | 64.1% reduced cost |
| Time to create index                         | 333.891ms               | 94.567ms                | 71.6% less time    |
| Execution time for query with “WHERE” clause | 16.292ms                | 0.880ms                 | 94.5% less time    |
| Size of index                                | 1984kb                  | 16kb                    | 99.1% less space   |
(Note: The results will vary, ​​depending on the data that is stored in the database)

We have seen that creating a partial index is a better choice where only a small subset of the values stored in the database are accessed frequently. Now, let's see how we can easily manage partial indexes using Atlas.

### Managing Partial Indexes is easy with Atlas

Managing partial indexes and database schemas in PostgreSQL can be confusing and error-prone. Atlas is an open-source project which allows us to manage our database using a simple and easy-to-understand declarative syntax (similar to Terraform). We will now learn how to manage partial indexes using Atlas.

:::info
If you are just getting started, install the latest version of Atlas using the guide to [setting up Atlas](https://atlasgo.io/cli/getting-started/setting-up).
:::

#### Managing Partial Index in Atlas

We will first use the `atlas schema inspect` command to get an HCL representation of the table which we created earlier by using the Atlas CLI:

```console
atlas schema inspect -u "postgres://postgres:mysecretpassword@localhost:5432/vaccination_data?sslmode=disable" > schema.hcl
```
```hcl title="schema.hcl"
table "vaccination_data" {
  schema = schema.public
  column "id" {
    null = false
    type = serial
  }
  column "country" {
    null    = true
    type    = character_varying(20)
  }
  column "title" {
    null    = true
    type    = character_varying(10)
  }
  column "names" {
    null    = true
    type    = character_varying(20)
  }
  column "vaccinated" {
    null    = true
    type    = character_varying(3)
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
  index "vaccinated_idx" {
    columns = [column.vaccinated]
    where   = "(vaccinated::text = 'Yes'::text AND country::text = 'India'::text AND title::text = 'Dr.'::text)"
  }
```

Save and apply the schema changes on the database by using the following command:

```console
atlas schema apply -u "postgres://postgres:mysecretpassword@localhost:5432/vaccination_data?sslmode=disable" -f schema.hcl
```

Atlas generates the necessary SQL statements to add the new partial index to the database schema. Press Enter while the `Apply` option is highlighted to apply the changes:

```console
-- Planned Changes:
-- Create index "vaccinated_idx" to table: "vaccination_data"
CREATE INDEX "vaccinated_idx" ON "public"."vaccination_data" ("vaccinated") WHERE (vaccinated::text = 'Yes'::text AND country::text = 'India'::text AND title::text = 'Dr.'::text)
✔ Apply
  Abort
```

To verify that our new index was created, open the database command line tool from previous step and run:

```sql
SELECT
    indexname,
    indexdef
FROM
    pg_indexes
WHERE
    tablename = 'vaccination_data';
```
```console title="Output"
[ RECORD 1 ]
indexname | vaccinated_idx
indexdef  | CREATE INDEX vaccinated_idx ON public.vaccination_data USING btree (vaccinated) WHERE (((vaccinated)::text = 'Yes'::text) AND ((country)::text = 'India'::text) AND ((title)::text = 'Dr.'::text))
```

Amazing! Our new partial index is now created!

### Limitation of using Partial Index

Partial indexes are useful in cases where we know ahead of time that a table is most frequently queried with a certain `WHERE` clause.  As applications evolve, access patterns to the database also change. Consequently, we may find ourselves in a situation where our index no longer covers many queries, causing them to become resource consuming and slow.

### Conclusion

In this section, we learned about PostgreSQL partial indexes and how we can easily create partial indexes in our database by using Atlas.

## Need More Help?​

[Join the Ariga Discord Server](https://discord.gg/zZ6sWVg6NT) for early access to features and the ability to provide exclusive feedback that improves your Database Management Tooling.