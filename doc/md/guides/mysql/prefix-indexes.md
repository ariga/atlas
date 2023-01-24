---
id: prefix-indexes
title: Prefix Indexes in MySQL
slug: /guides/mysql/prefix-indexes
---

### What are prefix indexes?

With MySQL, users may create secondary indexes for string columns which use the first N characters of the values stored in the column, instead of using the entire value. If used correctly, prefix indexes improve performance and reduce costs, all while minimizing the amount of storage space they take up on the disk.
When do we need them?

Let’s assume you have a lengthy column. If you have many records in an indexed table, the number of records the index needs to track also grows. If the index grows in size, the disk space needed to store the index itself increases as well. In many tables, lengthy records are not accessed with uniform search queries. One might prefer to use the `LIKE` operator to fetch content from a lengthy column, instead of writing the full value itself in the query.

Some data types (such as `BLOB` and `TEXT`) are not allowed to be indexed (unless the prefix value is specified). Also, The maximum length of the indexed value is 767 bytes. If the indexed value exceeds this length, it will be truncated.

In such cases, the prefix index can become useful to filter unsearched parts of values and give you, as an engineer, a tool to index only the parts of the values which are important.

### Syntax

Here is how you can create a prefix index:

```sql
CREATE INDEX
    index_name
ON
    table_name(column_name(length));
```

Here is how you can define a prefix index in a table definition:

```sql
CREATE TABLE
    table_name(column_name BLOB, index_name(column_name(length)));
```

Remember that the `length` is interpreted as the number of characters in non-binary string types such as `CHAR`, `VARCHAR` and `TEXT`. For binary string types such as `BINARY`, `VARBINARY` and `BLOB`, the `length` is interpreted as the number of bytes in the string.

Also, while indexing `BLOB` and `TEXT` types, you *must* specify the length

## Examples

### 1. Using prefix indexes with a TEXT type column

Let's see this in action by creating a table with the following command:

```sql
CREATE TABLE `comm_database` (
    `id` mediumint(8) unsigned NOT NULL auto_increment,
    `sender` varchar(255),
    `receiver` varchar(255),
    `sender_address` varchar(255),
    `email` varchar(255),
    `message` TEXT,
    `suspicious` BOOLEAN,
    `sent_date` varchar(255),
    PRIMARY KEY (`id`)
);
```

Here is how a portion of the table might look like after inserting values:

```console title="Output"
+----+------------------+-----------------+-------------------------------+---------------------------+-------------------------------------------------------------------------------------------------------------------------------------------------------+------------+-------------------------+
| id | sender           | receiver        | sender_address                | email                     | message                                                                                                                                               | suspicious | sent_date               |
+----+------------------+-----------------+-------------------------------+---------------------------+-------------------------------------------------------------------------------------------------------------------------------------------------------+------------+-------------------------+
|  1 | Abel Warren      | Latoya Spencer  | 575 North Rocky Fabien Avenue | xufh.vuho@example.com     | Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed nec dapibu                                                                               |          0 | 2008-09-24 04:57:35.552 |
|  2 | Erick Valentine  | Casey Leonard   | 243 West Green Nobel Parkway  | kmku.dmfn@example.com     | cus. Maecenas vitae odio sagittis lectus auctor varius. In ultricies facilisis sem, vel scelerisque nulla gravida vitae. Quisque volutpat com         |          0 | 2008-08-21 11:24:52.16  |
|  3 | Janice Payne     | Lakeisha Montes | 36 North Rocky Nobel St.      | kevr7@example.com         | Pellentesque porttitor dolor rhoncus ex tempus pulvinar. Aenean mattis malesuada tincidunt.                                                           |          0 | 2008-09-30 23:12:21.056 |
|  4 | Gretchen Mason   | Colleen Proctor | 241 North Green Second Way    | bymd1@example.com         | In hac habitasse platea dictumst. Nam gravida sem vitae hendrerit mattis. Maecenas vestibulum ante sed interdum accumsan.                             |          1 | 2008-01-04 07:04:50.56  |
|  5 | Lawanda Noble    | Alice Velazquez | 255 West White Hague Avenue   | xfvh.icqt@example.com     | lentesque sed dictum diam, non sodales ligula. Etiam lacinia augue et mattis sodales. Cr                                                              |          0 | 2008-05-07 06:03:08.736 |
|  6 | Robbie Baird     | Bernard Wilkins | 508 South Rocky Milton Blvd.  | monlt963@example.com      | mollis dui. Morbi neque mauris, convallis id enim eget, efficitur tincidunt nisi. Vestibulum congue nisi vel egestas condimentu                       |          0 | 2008-06-28 07:58:27.904 |
|  7 | Carla Compton    | Howard Randall  | 448 South Rocky Fabien Street | ffnjia@example.com        | aretra mauris elementum, efficitur nibh. In urna risus, scelerisque ut fringilla quis, suscipit nec dolor. Proin leo l                                |          0 | 2008-11-24 04:10:54.912 |
|  8 | Heath Stafford   | Hector Dickson  | 266 South Rocky New Road      | xlxnq6@example.com        | a vehicula tellus tempor. Suspendisse potenti. In                                                                                                     |          0 | 2008-09-28 13:23:49.44  |
|  9 | Kendra Stevenson | Vernon Rodgers  | 451 East White Milton Blvd.   | hnguyso.qcbat@example.com | arcu. Integer fringilla vulputate lectus, et faucibus mauris fermentum et. Quisque malesuada justo quam, vehicu                                       |          1 | 2008-09-15 23:07:41.504 |
| 10 | Brandie Chase    | Mike Finley     | 188 North Rocky Old Way       | xhqx@example.com          | mi. Donec non vehicula tortor. Nunc ut semper nibh, in accumsan metus. Cras eu pellentesque lacus, sit amet                                           |          0 | 2008-07-27 01:54:37.248 |
```

We do not have any indexes other than the primary index on the `id` column. Now, suppose we want information about senders and receivers of a message with a particular beginning. Let's query that data with the following command:

```sql
SELECT
    sender,
    sender_address,
    receiver,
    email
FROM
    comm_database
WHERE
    message LIKE 'Lorem ipsum dolor sit ame%';
```

```console title="Output"
+-----------------+-----------------------------------+-----------------+-------------------------------+
| sender          | sender_address                    | receiver        | email                         |
+-----------------+-----------------------------------+-----------------+-------------------------------+
| Bethany Velez   | 340 South Green Hague Blvd.       | Yvette Odonnell | ohhr2@example.com             |
| Clarence Simon  | 42 East Green Nobel Road          | Jay Mayo        | gonfy.qvwqum@example.com      |
| Vicky Bishop    | 64 West White Second Drive        | Kevin Petty     | wheo6@example.com             |
| Jami Lam        | 545 North Rocky Milton St.        | Justin Cameron  | dsec4@example.com             |
| Ernest Lynch    | 183 West Green Old Road           | Franklin Medina | knoph5@example.com            |
| Sam Freeman     | 324 West Green Fabien Road        | Jose Shelton    | ewqa@example.com              |
| Derek Copeland  | 88 South Rocky Cowley Blvd.       | Nichole Powell  | tzkr3@example.com             |
| Lynn Lynch      | 72 South Rocky New Drive          | Carolyn Levine  | rrew@example.com              |
| Olivia Meyer    | 384 South Green Hague Freeway     | Angie Whitaker  | yvft01@example.com            |
| Shaun Harper    | 886 North Rocky Hague Parkway     | Gregory Irwin   | hhro6@example.com             |
| Darrin Trujillo | 88 North White Nobel Freeway      | Gail Blevins    | ddwsw@example.com             |
| Stanley Mata    | 758 South Green Cowley Blvd.      | Alma Cole       | ksxx.viza@example.com         |
| Cory Howard     | 517 West Rocky Milton Boulevard   | Eugene Ortiz    | faut.hkvn@example.com         |
| Joyce Owen      | 24 North Green Hague Way          | Leah Wilkerson  | jupv.wypo@example.com         |
| Kristina Beard  | 68 East White New Street          | Thomas Faulkner | mrixv@example.com             |
| Connie Glass    | 874 South White Clarendon Freeway | Sharon Lamb     | qgesbi.whcqendylg@example.com |
+-----------------+-----------------------------------+-----------------+-------------------------------+
16 rows in set (0.62 sec)
```

Now, let's see how much the query cost by running  the following command:

```sql
SHOW STATUS LIKE 'Last_query_cost';
```

```console title="Output"
+-----------------+---------------+
| Variable_name   | Value         |
+-----------------+---------------+
| Last_query_cost | 108430.088807 |
+-----------------+---------------+
1 row in set (0.01 sec)
```

Notice that the query cost 108430.088807 units.

Now, suppose we want to optimize this query but we do not know about prefix indexes yet. In this case, we will make an index on column `message` with the following command:

```sql
CREATE INDEX
    message_idx
ON
    comm_database(message);
```

```console title="Output"
ERROR 1170 (42000): BLOB/TEXT column 'message' used in key specification without a key length
```

That didn’t work! As mentioned in the [Syntax](#syntax) section above and the error message, you must specify length when indexing `BLOB` and `TEXT` type columns. Let’s specify the length and try to check the performance for the same query again:

```sql
CREATE INDEX
    message_idx
ON
    comm_database(message(30));
```

```console title="Output"
Query OK, 0 rows affected (3.55 sec)
Records: 0  Duplicates: 0  Warnings: 0
```

```sql
SELECT
    sender,
    sender_address,
    receiver,
    email
FROM
    comm_database
WHERE
    message LIKE 'Lorem ipsum dolor sit ame%';
.
.
.
SHOW STATUS LIKE 'Last_query_cost';
```

```console title="Output"
+-----------------+-----------+
| Variable_name   | Value     |
+-----------------+-----------+
| Last_query_cost | 16.461134 |
+-----------------+-----------+
```

Awesome! Now our query performs far better and has a significantly lower cost, (16.46 units now as compared to 108430.08 units before creating the index)
Additionally, we can also confirm by using `EXPLAIN` with the query that the index we created is indeed being used:

```console title="Output"
+----+-------------+--------------+------------+-------+---------------------+---------------------+---------+------+------+----------+-------------+
| id | select_type | table        | partitions | type  | possible_keys       | key                 | key_len | ref  | rows | filtered | Extra       |
+----+-------------+--------------+------------+-------+---------------------+---------------------+---------+------+------+----------+-------------+
|  1 | SIMPLE      | comm_database| NULL       | range | message_idx         | message_idx         | 123     | NULL |   16 |   100.00 | Using where |
+----+-------------+--------------+------------+-------+---------------------+---------------------+---------+------+------+----------+-------------+
```

In the above example we saw how we can improve performance for a `TEXT` type column by specifying appropriate `length` while creating the index.

Now in the next example, let’s see how we can optimize space used by the index with  a prefix index.

### 2. Using prefix indexes to optimize space usage and increase query performance

Using the same table, let’s assume that we want to fetch data of senders by using their email address. Here is a sample query and its cost without any index:

```sql
SELECT
    id,
    sender,
    sender_address
FROM
    comm_database
WHERE
    email = 'dsec4@example.com';
```

```console title="Output"
+--------+----------+----------------------------+
| id     | sender   | sender_address             |
+--------+----------+----------------------------+
| 210829 | Jami Lam | 545 North Rocky Milton St. |
+--------+----------+----------------------------+
```

```console title="Cost"
+-----------------+---------------+
| Variable_name   | Value         |
+-----------------+---------------+
| Last_query_cost | 108453.391718 |
+-----------------+---------------+
```

To optimize this query, usually we would create the following index:

```sql
CREATE INDEX
    email_idx
ON
    comm_database(email);
```

```console title="Output"
Query OK, 0 rows affected (1.57 sec)
Records: 0  Duplicates: 0  Warnings: 0
```

The cost for the query after creating this index is as follows:

```console title="Cost"
+-----------------+----------+
| Variable_name   | Value    |
+-----------------+----------+
| Last_query_cost | 0.870951 |
+-----------------+----------+
```

The storage space used by the index is as following:

```sql
SELECT
    stat_value
FROM
    mysql.innodb_index_stats
WHERE
    index_name = 'email_idx' AND stat_name= 'size';
```

```console title="Output"
+------------+
| stat_value |
+------------+
|       2085 |
+------------+
```

:::info
The `innodb_page_size` variable specifies the size of the pages used by the InnoDB storage engine, and the `stat_value` column contains statistics about the index, such as the number of pages used by the index. When `stat_name` is 'size', the `stat_value` column contains the size of the index in terms of number of pages.

For example, if the `innodb_page_size` system variable is set to 16 KB and the `stat_value` column contains the value 10, this means that the index uses 10 pages, or 160KB of disk space.
:::

For our example, keeping `innodb_page_size` as 16KB in mind, our index uses ~32MB disk space. Now, we can further improve the query performance as well as reduce the storage used by the index by using a prefix index on the email column. Let’s create a prefix index with the following command:

```sql
ALTER TABLE
    comm_database
DROP INDEX
    email_idx,
ADD INDEX
    email_prefix_idx (email(5));
```

```console title="Output"
Query OK, 0 rows affected (1.59 sec)
Records: 0  Duplicates: 0  Warnings: 0
```

The cost for the query after creating this index is as follows:

```console title="Cost"
+-----------------+----------+
| Variable_name   | Value    |
+-----------------+----------+
| Last_query_cost | 0.832643 |
+-----------------+----------+
```

Our cost has reduced by 5% by using the prefix index.

And the storage space used by the index is as follows:

```sql
SELECT
    stat_value
FROM
    mysql.innodb_index_stats
WHERE
    index_name = 'email_prefix_idx' AND stat_name= 'size';
```

```console title="Output"
+------------+
| stat_value |
+------------+
|       1123 |
+------------+
```

Based on the calculation using the `innodb_page_size` of 16KB, the index now occupies 17.55MB of storage space, a reduction of 46% from the previous size of 32.58MB.

## Managing Prefix Indexes is easy with Atlas​

Atlas is an open-source project which allows us to manage our database using a simple and easy-to-understand declarative syntax (similar to Terraform).

:::info
If you are just getting started, install the latest version of Atlas using the guide to [setting up Atlas](https://atlasgo.io/getting-started/).
:::

We will first use the `atlas schema inspect` command to get an [HCL](https://atlasgo.io/guides/ddl#hcl) representation of the table we created earlier (without any indexes) by using the Atlas CLI:

```console title="Terminal"
atlas schema inspect -u "mysql://root:@localhost:3306/comm_database" > schema.hcl
```

```hcl title="schema.hcl"
table "comm_database" {
  schema = schema.comm_database
  column "id" {
    null           = false
    type           = mediumint
    unsigned       = true
    auto_increment = true
  }
  column "sender" {
    null = true
    type = varchar(255)
  }
  column "receiver" {
    null = true
    type = varchar(255)
  }
  column "sender_address" {
    null = true
    type = varchar(255)
  }
  column "email" {
    null = true
    type = varchar(255)
  }
  column "message" {
    null = true
    type = text
  }
  column "suspicious" {
    null = true
    type = bool
  }
  column "sent_date" {
    null = true
    type = varchar(255)
  }
  primary_key {
    columns = [column.id]
  }
}
schema "comm_database" {
  charset = "utf8mb4"
  collate = "utf8mb4_0900_ai_ci"
}
```

Now, lets add the two following index definitions to the file:

```hcl
  index "message_idx" {
    on {
      column = column.message
      prefix = 30
    }
  }
  index "email_prefix_idx" {
    on {
      column = column.email
      prefix = 5
    }
  }
```

Save the file and apply the schema changes on the database by using the following command:

```console title="Terminal"
atlas schema apply -u "mysql://root:password@localhost:3306/comm_database" -f schema.hcl --dev-url docker://mysql/8/comm_database
```

:::info
If you get `Error: pulling image: exit status 1` error, ensure that Docker Desktop is up and running.
:::

Atlas generates the necessary SQL statements to add the new index to the database schema.

```console
-- Planned Changes:
-- Modify "comm_database" table
ALTER TABLE `comm_database`.`comm_database` ADD INDEX `message_idx` (`message` (30)), ADD INDEX `email_prefix_idx` (`email` (5))
Use the arrow keys to navigate: ↓ ↑ → ←
? Are you sure?:
  ▸ Apply
    Abort
```

To verify that our new index was created, run the following command:

```console title="Terminal"
atlas schema inspect -u "mysql://root:password@localhost:3306/comm_database" | grep -A5 index
```

```console title="Output"
  index "message_idx" {
    on {
      column = column.message
      prefix = 30
    }
  }
  index "email_prefix_idx" {
    on {
      column = column.email
      prefix = 5
    }
  }
```

Amazing! Our new indexes `message_idx` and `email_prefix_idx` are now created!

### Wrapping up​

In this guide, we have demonstrated that creating a prefix index is inevitable when improving performance for queries accessing `TEXT` or `BLOB` type columns. More importantly, it is a smart choice when trying to improve performance for queries accessing `CHAR`, `VARCHAR`, `BINARY` and `VARBINARY` type columns, if used wisely.

## Need More Help?​​
[Join](https://discord.com/invite/zZ6sWVg6NT) the Ariga Discord Server for early access to features and the ability to provide exclusive feedback that improves your Database Management Tooling.
[Sign up](https://www.getrevue.co/profile/ariga) to our newsletter to stay up to date about [Atlas](https://atlasgo.io/), our OSS database schema management tool, and our cloud platform [Atlas Cloud](https://atlasgo.cloud/).
