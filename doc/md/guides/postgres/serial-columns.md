---
id: serial-columns
title: Serial Type Columns in PostgreSQL
slug: /guides/postgres/serial-columns
---

PostgreSQL allows creating columns of types `smallserial`, `serial`, and `bigserial`. These types are not
_actual_ types, but more like "macros" for creating non-nullable integer columns with sequences attached.

We can see this in action by creating a table with 3 "serial columns":

```sql
CREATE TABLE serials(
    c1 smallserial,
    c2 serial,
    c3 bigserial
);
```

```sql title="Serials Description"
 Column |   Type   | Nullable |            Default
--------+----------+----------+-------------------------------
 c1     | smallint | not null | nextval('t_c1_seq'::regclass)
 c2     | integer  | not null | nextval('t_c2_seq'::regclass)
 c3     | bigint   | not null | nextval('t_c3_seq'::regclass)
```

As you can see, each serial column was created as non-nullable integer with a default value set to the next sequence
value.

:::info
Note that `nextval` increments the sequence by 1 and returns its value. Thus, the first call to
`nextval('serials_c1_seq')` returns 1, the second returns 2, etc.
:::

### `ALTER COLUMN` type to serial

Sometimes it is necessary to change the column type from `integer` type to `serial`. However, as mentioned above, the
`serial` type is not a true type, and therefore, the following commands will fail:

```sql
CREATE TABLE t(
    c integer not null primary key
);

ALTER TABLE t ALTER COLUMN c TYPE serial;
// highlight-next-line-error-message
ERROR: type "serial" does not exist
```

We can achieve this by manually creating a [sequence](https://www.postgresql.org/docs/current/sql-createsequence.html)
owned by the column `c`, and setting the column `DEFAULT` value to the incremental counter of the sequence using the
[`nextval`](https://www.postgresql.org/docs/current/functions-sequence.html) function.

:::note
Note that it is recommended to follow the PostgreSQL naming format (i.e. `<table>_<column>_seq`)
when creating the sequence as some database tools know to detect such columns as "serial columns".
:::

```sql
-- Create the sequence.
CREATE SEQUENCE "public"."t_c_seq" OWNED BY "public"."t"."c";

-- Assign it to the table default value.
ALTER TABLE "public"."t" ALTER COLUMN "c" SET DEFAULT nextval('"public"."t_c_seq"');
```

### Update the sequence value

When a sequence is created, its value starts from 0 and the first call to `nextval` returns 1. Thus, in case the column
`c` from the example above already contains values, we may face a constraint error on insert when the sequence number
will reach to the minimum value of `c`. Let's see an example:

```sql
SELECT "c" FROM "t";
// highlight-start
 c
---
 2
 3
// highlight-end

-- Works!
INSERT INTO "t" DEFAULT VALUES;
-- Fails!
INSERT INTO "t" DEFAULT VALUES;
// highlight-next-line-error-message
ERROR:  duplicate key value violates unique constraint "t_pkey"
// highlight-next-line-error-message
DETAIL:  Key (c)=(2) already exists.
```

We can work around this by setting the sequence current value to the maximum value of `c`, so the following call to
`nextval` will return `MAX(c)+1`, the one after `MAX(c)+2`, and so on.

```sql
SELECT setval('"public"."t_c_seq"', (SELECT MAX("c") FROM "t"));
// highlight-start
 setval
--------
   3
// highlight-end

-- Works!
INSERT INTO "t" DEFAULT VALUES;
SELECT "c" FROM "t";
// highlight-start
 c
---
 2
 3
 4
// highlight-end
```

### Managing Serial Columns with Atlas

Atlas makes it easier to define and manipulate columns of `serial` types. Let's use the
[`atlas schema inspect`](../../reference.md#atlas-schema-inspect) command to get a representation
of the table we created above in the Atlas HCL format :

```console
atlas schema inspect -u "postgres://postgres:pass@:5432/test?sslmode=disable" > schema.hcl
```

```hcl title="schema.hcl"
table "t" {
  schema = schema.public
  column "c" {
    null = false
    type = serial
  }
  primary_key {
    columns = [column.c]
  }
}
schema "public" {
}
```

After inspecting the schema, we can modify it to demonstrate Atlas's capabilities in migration planning:

#### Change a column type from `serial` to `bigserial`

```hcl title="schema.hcl"
table "t" {
  schema = schema.public
  column "c" {
    null = false
    // highlight-start
    type = bigserial
    // highlight-end
  }
  primary_key {
    columns = [column.c]
  }
}
schema "public" {
}
```

Next, running `schema apply` will plan and execute the following changes:

```console
atlas schema apply -u "postgres://postgres:pass@:5432/test?sslmode=disable" -f schema.hcl

-- Planned Changes:
-- Modify "t" table
// highlight-next-line-info
ALTER TABLE "public"."t" ALTER COLUMN "c" TYPE bigint
✔ Apply
```

As you can see, Atlas detected that only the underlying integer type was changed as `serial` maps to `integer` and
`bigserial` maps to `bigint`.

#### Change a column type from `bigserial` to `bigint`

```hcl title="schema.hcl"
table "t" {
  schema = schema.public
  column "c" {
    null = false
    // highlight-start
    type = bigint
    // highlight-end
  }
  primary_key {
    columns = [column.c]
  }
}
schema "public" {
}
```

After changing column `c` to `bigint`, we can run `schema apply` and let Atlas plan and execute the new changes:

```console
atlas schema apply -u "postgres://postgres:pass@:5432/test?sslmode=disable" -f schema.hcl

-- Planned Changes:
-- Modify "t" table
// highlight-next-line-info
ALTER TABLE "public"."t" ALTER COLUMN "c" DROP DEFAULT
-- Drop sequence used by serial column "c"
// highlight-next-line-info
DROP SEQUENCE IF EXISTS "public"."t_c_seq"
✔ Apply
```

As you can see, Atlas dropped the `DEFAULT` value that was created by the `serial` type, and in addition removed
the sequence that was attached to it, as it is no longer used by the column.

#### Change a column type from `bigint` to `serial`

```hcl title="schema.hcl"
table "t" {
  schema = schema.public
  column "c" {
    null = false
    // highlight-start
    type = serial
    // highlight-end
  }
  primary_key {
    columns = [column.c]
  }
}
schema "public" {
}
```

Changing a column type from `bigint` to `serial` requires 3 changes:
1. Create a sequence named `t_c_seq` owned by `c`.
2. Set the `DEFAULT` value of `c` to `nextval('"public"."t_c_seq"')`.
3. Alter the column type, as `serial` maps to `integer` (!= `bigint`).

We call [`atlas schema apply`](../../reference.md#atlas-schema-apply) to plan and execute this three step process
with Atlas:

```console
atlas schema apply -u "postgres://postgres:pass@:5432/test?sslmode=disable" -f schema.hcl

-- Planned Changes:
-- Create sequence for serial column "c"
// highlight-next-line-info
CREATE SEQUENCE IF NOT EXISTS "public"."t_c_seq" OWNED BY "public"."t"."c"
-- Modify "t" table
// highlight-next-line-info
ALTER TABLE "public"."t" ALTER COLUMN "c" SET DEFAULT nextval('"public"."t_c_seq"'), ALTER COLUMN "c" TYPE integer
✔ Apply
```

## Need More Help?

[Join the Ariga Discord Server](https://discord.gg/zZ6sWVg6NT) for early access to features and the ability to provide
exclusive feedback that improves your Database Management Tooling.

