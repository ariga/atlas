---
id: partial-indexes
title: Partial Indexes in SQLite
slug: /guides/sqlite/partial-indexes
---

### Overview of Partial Indexes

#### What are Partial Indexes?

With SQLite, users may create _partial indexes_, which are types of indexes that exist on a subset of a table, rather than the entire table itself. If used correctly, partial indexes improve performance and reduce costs, all while minimizing the amount of storage space they take up on the disk.

#### Why do we need them?

Let's demonstrate a case where partial indexes may be useful by contrasting them with a non-partial index. ​​If you have many records in an indexed table, the number of records the index needs to track also grows. If the index grows in size, the disk space needed to store the index itself increases as well.
In many tables, different records are not accessed with uniform frequency. A subset of a table's records might not be searched very frequently or not searched at all. Records take up precious space in your index whether they are queried or not, and are updated when a new entry is added to the field.

Partial indexes come into the picture to filter unsearched values and give you, as an engineer, a tool to index only what's important.

#### Advantages of using Partial Indexes
1. Partial indexes have index entries only for a defined subset of rows, compared to ordinary indexes which have exactly one index entry for every row in the table.
2. When used wisely, partial indexes result in smaller database files with improved query and write performance.

#### Basic SQLite syntax for using Partial Indexes

```sql
CREATE INDEX
    index_name
ON
    table_name(column_list)
WHERE
    expression;
```

#### Example of Non-partial Indexes vs Partial Indexes in SQLite

Let's see this in action by creating a table with the following command:

```sql
CREATE TABLE vaccination_data (
  id INTEGER NOT NULL,
  country varchar(100) default NULL,
  title TEXT default NULL,
  names varchar(255) default NULL,
  vaccinated varchar(255) default NULL,
  PRIMARY KEY (id)
);
```

Here is how a portion of the table might look like after inserting values:

```sql
SELECT * FROM vaccination_data;
```
```console title="Output"
+----+--------------------+-------+----------------+------------+
| id |      country       | title |     names      | vaccinated |
+----+--------------------+-------+----------------+------------+
| 1  | Poland             | Er.   | Travis Freeman | No         |
| 2  | Australia          | Mr.   | Hu Dodson      | No         |
| 3  | Vietnam            | Ms.   | Amery Herman   | No         |
| 4  | Peru               | Mr.   | Brynne Mann    | Yes        |
| 5  | Chile              | Er.   | Nora Mitchell  | No         |
| 6  | Brazil             | Er.   | Tanner Oneal   | No         |
| 7  | Vietnam            | Mr.   | Ora Conway     | Yes        |
| 8  | United Kingdom     | Er.   | Quinn Waters   | No         |
| 9  | Russian Federation | Er.   | Xyla Holloway  | No         |
| 10 | Norway             | Mr.   | Macy Sullivan  | No         |
.
.
.
| 576000 | Ukraine   | Dr.   | Kuame Gay         | Yes        |
+--------+-----------+-------+-------------------+------------+
```

:::info
You can also beautify tables in SQLite like shown above, by using the command `.mode table`
:::

In the following example, suppose we want a list of doctors from India that have taken the vaccine. If we want to use a non-partial index, we can create it on the "vaccinated" column with the following command:

```sql
CREATE INDEX
    vaccinated_idx
ON
    vaccination_data(vaccinated);
```

Now, let's check the size of the index that we created, with the following command:

```sql
SELECT NAME, sum(pgsize) AS size FROM dbstat GROUP BY NAME ORDER BY size DESC;
```
```console title="Output"
+------------------+----------+
|       name       |   size   |
+------------------+----------+
| vaccination_data | 22224896 |
| vaccinated_idx   | 6492160  |
| sqlite_schema    | 4096     |
+------------------+----------+
```
Notice that the total size of our index vaccinated_idx is 6492160 bytes (~6 MB).

:::info
The DBSTAT virtual table is a read-only eponymous virtual table that returns information about the amount of disk space used to store the content of an SQLite database. To know more about DBSTAT, visit the official documentation page [here](https://www.sqlite.org/dbstat.html)
:::

Now, suppose we want to accelerate the same query using the partial index. Let's begin by dropping the existing index that we created earlier:

```sql
DROP INDEX vaccinated_idx;
```

In the following command, we will create an index with a `WHERE` clause that precisely describes the list of doctors from India that have taken the vaccine.

```sql
CREATE INDEX
    vaccinated_idx
ON
    vaccination_data(vaccinated)
WHERE
    vaccinated = 'Yes' AND country = 'India' AND title = 'Dr';
```
Let’s verify if the index we created is being used in the query with a `WHERE` clause by running the following command:

```sql
EXPLAIN QUERY PLAN
SELECT
        *
FROM
        vaccination_data
WHERE
        vaccinated = 'Yes' AND country = 'India' AND title = 'Dr';
```

```console title="Output"
QUERY PLAN
`--SEARCH vaccination_data USING INDEX vaccinated_idx
```

We confirmed that the index vaccinated_idx is being used while running the query above. Let's check the size of the index that we created again, with the following command:

```sql
SELECT NAME, sum(pgsize) AS size FROM dbstat GROUP BY NAME ORDER BY size DESC;
```
```console title="Output"
+------------------+----------+
|       name       |   size   |
+------------------+----------+
| vaccination_data | 22224896 |
| vaccinated_idx   | 4096     |
| sqlite_schema    | 4096     |
+------------------+----------+
```
(Note: The results will vary, ​​depending on the data that is stored in the database)

Notice that the total size of our index vaccinated_idx is just 4096 bytes. In our example, the index size for the partial index took significantly less space (4 KB) compared to the non-partial index that we created earlier on the 'vaccinated' column (~6 MB).

:::info
Learn more about partial indexes in SQLite from official documentation [here](https://www.sqlite.org/partialindex.html)
:::

We have seen that creating a partial index is a better choice where only a small subset of the values stored in the database are accessed frequently. Now, let's see how we can easily manage partial indexes using Atlas.

### Managing Partial Indexes is easy with Atlas

Managing partial indexes and database schemas in SQLite can be confusing and error-prone. Atlas is an open-source project which allows us to manage our database using a simple and easy-to-understand declarative syntax (similar to Terraform). We will now learn how to manage partial indexes using Atlas.

:::info
If you are just getting started, install the latest version of Atlas using the guide to [setting up Atlas](https://atlasgo.io/cli/getting-started/setting-up).
:::

#### Managing Partial Indexes in Atlas

We will first use the `atlas schema inspect` command to get an HCL representation of the table which we created earlier by using the Atlas CLI:

```console
atlas schema inspect -u "sqlite://vaccination_data.db" > schema.hcl
```
```hcl title="schema.hcl"
table "vaccination_data" {
  schema = schema.main
  column "id" {
    null = false
    type = integer
  }
  column "country" {
    null    = true
    type    = varchar(100)
  }
  column "title" {
    null    = true
    type    = text
  }
  column "names" {
    null    = true
    type    = varchar(255)
  }
  column "vaccinated" {
    null    = true
    type    = varchar(255)
  }
  primary_key {
    columns = [column.id]
  }
}
schema "main" {
}
```

Now, lets add the following index definition to the file:

```hcl
  index "vaccinated_idx" {
    columns = [column.vaccinated]
    where   = "(vaccinated = 'Yes' AND country = 'India' AND title = 'Dr.')"
  }
```

Save the changes in the `schema.hcl` file and apply the changes on the database by using the following command:

```console
atlas schema apply -u "sqlite://vaccination_data.db" -f schema.hcl
```

Atlas generates the necessary SQL statements to add the new partial index to the database schema. Press Enter while the `Apply` option is highlighted to apply the changes:

```console
-- Planned Changes:
-- Create index "vaccinated_idx" to table: "vaccination_data"
CREATE INDEX `vaccinated_idx` ON `vaccination_data` (`vaccinated`) WHERE (vaccinated = 'Yes' AND country = 'India' AND title = 'Dr.')
✔ Apply
  Abort
```

To verify that our new index was created, open the database command line tool and run:

```sql
.index
```
```console title="Output"
vaccinated_idx
```

Amazing! Our new partial index is now created!

### Limitation of using Partial Indexes

Partial indexes are useful in cases where we know ahead of time that a table is most frequently queried with a certain WHERE clause. As applications evolve, access patterns to the database also change. Consequently, we may find ourselves in a situation where our index no longer covers many queries, causing them to become resource-consuming and slow.

### Conclusion

In this section, we learned about SQLite partial indexes and how we can easily create partial indexes in our database by using Atlas.

## Need More Help?​

[Join the Ariga Discord Server](https://discord.com/invite/zZ6sWVg6NT) for early access to features and the ability to provide exclusive feedback that improves your Database Management Tooling.

[Sign up](https://atlasnewsletter.substack.com/) to our newsletter to stay up to date about [Atlas](https://atlasgo.io), our OSS database schema management tool, and our cloud platform [Atlas Cloud](https://atlasgo.cloud).
