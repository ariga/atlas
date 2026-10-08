# Writing MySQL Schemas in HCL

The rules of the HCL schema language that matter on MySQL and MariaDB, and the places where HCL behaves
differently from SQL. The language: https://atlasgo.io/atlas-schema/hcl. Column types:
https://atlasgo.io/atlas-schema/hcl-types. Every MySQL block and attribute: https://atlasgo.io/hcl/mysql,
and for MariaDB: https://atlasgo.io/hcl/mariadb.

Several blocks below need `atlas login`. If `atlas whoami` shows no login, ask the user to run
`atlas login`; a user without an account can start a free trial with it.

## Files

- Name MySQL schema files `*.my.hcl`, MariaDB files `*.ma.hcl`, and test files `*.test.hcl`, so editors
  apply the right schema (https://atlasgo.io/atlas-schema/hcl#file-naming-for-editor-support).
- A `schema` block is a MySQL database, and every `table` sets `schema`. A file with several `schema`
  blocks needs server scope (`references/dev-database.md`).

## Columns

- A column without `null` is `NOT NULL`, the opposite of SQL. Write `null = true` for a nullable column.
- `default` takes a literal, or an expression in `sql()`, such as `default = sql("CURRENT_TIMESTAMP")`.
  `on_update` sets `ON UPDATE` and takes `sql()` too. Write timestamp defaults as expressions with the
  column's precision, not as quoted strings, which fail with `Error 1067: Invalid default value`:
  `default = sql("current_timestamp(6)")` and `on_update = sql("current_timestamp(6)")`.
- Write literal defaults as HCL literals (`default = 0`, `default = "active"`), and expressions in
  `sql()`, or plans can keep showing the default as changed. Expression defaults other than
  `CURRENT_TIMESTAMP` need MySQL 8.0.13 and parentheses around the expression, such as
  `default = sql("(uuid())")`.
- A renamed column or table is planned as a drop and an add, which loses its data, unless the rename is
  confirmed. In a non-interactive run, such as an agent or CI, declare it with `renamed_from`
  (https://atlasgo.io/guides/destructive-change-policy):

  ```hcl
    column "deprecated_legacy_email" {
      renamed_from = "legacy_email"
      type         = varchar(200)
    }
  ```

  A SQL schema does the same with a directive on the line before the column:
  `-- atlas:renamed_from legacy_email`.
- `auto_increment = true` on a column makes it `AUTO_INCREMENT`. On the table, `auto_increment` sets the
  start value (https://atlasgo.io/atlas-schema/hcl#auto-increment):

  ```hcl
  table "users" {
    schema = schema.public
    column "id" {
      null = false
      type = bigint
      auto_increment = true
    }
    primary_key  {
      columns = [column.id]
    }
    auto_increment = 100
  }
  ```

- `unsigned = true` applies to integer, decimal, and floating-point types.
- `charset` and `collate` set a column's character set and collation; tables and schemas take them
  too. A column or table without them takes its parent's defaults, which come from the dev server when
  nothing sets them (`references/dev-database.md`, Character Sets and Collations).
- `invisible = true` makes a column invisible to `SELECT *` (MySQL 8.0.23 and later).
- Generated columns take an expression in `as`, or an `as` block with the storage type. On MySQL, they
  are `VIRTUAL` by default (https://atlasgo.io/atlas-schema/hcl#generated-columns):

  ```hcl
    column "b" {
      type = int
      # In MySQL, generated columns are VIRTUAL by default.
      as = "a * 2"
    }
    column "c" {
      type = int
      as {
        expr = "a * b"
        type = STORED
      }
    }
  ```

  A generated column that is part of a primary key or foreign key must be `STORED`
  (https://atlasgo.io/guides/mysql/generated-columns). Adding a `STORED` one copies the table
  (`MY140`), so prefer `VIRTUAL` otherwise. Generated columns need a dev database to normalize the
  expression.

## Column Types

Write parameters inline: `varchar(255)`, `decimal(5,2)`, `datetime(6)`
(https://atlasgo.io/atlas-schema/hcl-types).

- `enum` and `set` are column types with their values inline, not blocks:

  ```hcl
    column "c1" {
      type = enum("a", "b")
    }
  ```

  and `type = set("a", "b")`. Value changes are covered in `references/objects.md`, Enums and Sets.
- `bool` and `boolean` map to `tinyint(1)`; inspection prints `bool`.
- Display widths are dropped: `bigint(8) unsigned` inspects as `type = bigint` with `unsigned = true`.
- Spatial columns take `srid`. Anything without an HCL type is written with `sql("...")`.

## Keys, Constraints, and Indexes

- A foreign key's `on_delete` and `on_update` take `NO_ACTION`, `RESTRICT`, `CASCADE`, `SET_NULL`, or
  `SET_DEFAULT`. MySQL has no deferrable constraints.
- MySQL creates an index for a foreign key's columns when none exists, and inspection shows it as an
  `index` block with the foreign key's name:

  ```hcl
    foreign_key "orders_ibfk_1" {
      columns     = [column.customer_id]
      ref_columns = [table.customers.column.id]
      on_update   = NO_ACTION
      on_delete   = NO_ACTION
    }
    index "orders_ibfk_1" {
      columns = [column.customer_id]
    }
  ```

- A `check` takes its SQL in `expr`. MySQL enforces checks from 8.0.16; earlier versions parse and
  ignore them (https://atlasgo.io/guides/mysql/check-constraint):

  ```hcl
    check "user_id" {
      expr = "value > 0"
    }
  ```

  `enforced = false` creates the check `NOT ENFORCED`, on MySQL only.
- A unique index is an `index` with `unique = true`. `type` takes `BTREE`, `HASH`, `FULLTEXT`, or
  `SPATIAL`, and a full-text index takes `parser`. MySQL indexes have no `where`, `include`, or `ops`.
- Functional indexes (MySQL 8.0.13 and later) use `on` blocks with `expr`
  (https://atlasgo.io/guides/mysql/functional-indexes; more examples, such as JSON values:
  https://atlasgo.io/faq/mysql-functional-indexes):

  ```hcl
    index "sci_math_avg_idx" {
      on {
        expr = "((`science` + `mathematics`) / 2)"
      }
    }
  ```

- An index on a `BLOB` or `TEXT` column needs a prefix length, or MySQL fails with
  `ERROR 1170 (42000): BLOB/TEXT column 'message' used in key specification without a key length`
  (https://atlasgo.io/guides/mysql/prefix-indexes):

  ```hcl
    index "message_idx" {
      on {
        column = column.message
        prefix = 30
      }
    }
  ```

- `desc = true` in an `on` block builds a descending index part (MySQL 8.0 and later, InnoDB)
  (https://atlasgo.io/guides/mysql/descending-indexes).

## Tables

- `engine` sets the storage engine, and takes `InnoDB`, `MyISAM`, other engine names, or a string
  (https://atlasgo.io/atlas-schema/hcl#table). MyISAM ignores foreign keys (`references/objects.md`).
  Changing the engine copies the table (`MY138`):

  ```hcl
  table "orders" {
    schema = schema.public
    engine = "MyRocks"
  }
  ```

- Partitions need `atlas login`. A table's `partition` block sets the `type` (`RANGE`, `RANGE_COLUMNS`,
  `LIST`, `LIST_COLUMNS`, `HASH`, `LINEAR_HASH`, `KEY`, or `LINEAR_KEY`), the key in `by` blocks or
  `columns`, and each named partition inside it. Bounds are strings, including `"MAXVALUE"`
  (https://atlasgo.io/atlas-schema/hcl#partitions):

  ```hcl
  table "range_orders" {
    schema = schema.public
    column "id" {
      null = false
      type = int
    }
    partition {
      type = RANGE
      by {
        expr = "(`id` + 1)"
      }
      partition "p0" {
        values_less_than = ["101"]
        comment          = "first range"
      }
      partition "pmax" {
        values_less_than = ["MAXVALUE"]
        comment          = "catch all range"
      }
    }
  }
  ```

- On MariaDB, `system_versioned = true` makes a table system-versioned; dropping it later deletes the
  history (`MY146`).

## Views, Functions, Procedures, and Triggers

These need `atlas login`.

- A view takes `as`, `depends_on`, `check_option` (`LOCAL` or `CASCADED`), and `security`
  (https://atlasgo.io/atlas-schema/hcl#view):

  ```hcl
  view "comedies" {
    schema = schema.public
    column "id" {
      type = int
    }
    column "name" {
      type = text
    }
    as           = "SELECT id, name FROM films WHERE kind = 'Comedy'"
    depends_on   = [table.films]
    check_option = CASCADED
    security     = INVOKER // DEFINER | INVOKER (MySQL/MariaDB only).
  }
  ```

- A function's arguments are named `arg` blocks, and it sets `deterministic`, `data_access`, and
  `security`; there is no `lang`. Without `deterministic = true`, the function is `NOT DETERMINISTIC`
  (https://atlasgo.io/atlas-schema/hcl#function):

  ```hcl
  function "add2" {
    schema = schema.public
    arg "a" {
      type = int
    }
    arg "b" {
      type = int
    }
    return        = int
    as            = "return a + b"
    deterministic = true     // NOT DETERMINISTIC | DETERMINISTIC
    data_access   = NO_SQL   // CONTAINS_SQL | NO_SQL | READS_SQL_DATA | MODIFIES_SQL_DATA
    security      = INVOKER  // DEFINER | INVOKER
  }
  ```

- A procedure's `arg` takes `mode` (`IN`, `OUT`, or `INOUT`), and `charset` or `collate`.
- A trigger has its body in `as`, one timing block (`before` or `after`) with one event, and `follows`
  or `precedes` to order triggers on the same event (https://atlasgo.io/atlas-schema/hcl#trigger):

  ```hcl
  trigger "after_orders_insert" {
    on = table.orders
    after {
      insert = true
    }
    as = <<-SQL
    BEGIN
        INSERT INTO orders_audit(order_id, changed_at, operation)
        VALUES (NEW.order_id, NOW(), 'INSERT');
    END
    SQL
  }
  ```

## Roles, Users, and Grants

Covered in `references/security.md`.

## Seed Data

Rows are declared next to the table in a `data` block, and applied only when the env enables data sync
(needs `atlas login`, https://atlasgo.io/guides/mysql/seed-data):

```hcl
data {
  table = table.countries
  rows = [
    { id = 1, code = "US", name = "United States" },
    { id = 2, code = "IL", name = "Israel" },
    { id = 3, code = "DE", name = "Germany" },
    { id = 4, code = "VN", name = "Vietnam" },
  ]
}
```

- The env's `data` block sets the `mode`: `INSERT`, `UPSERT`, or `SYNC`. `SYNC` also needs `max_rows`,
  and refuses a table with more rows than that.
- `AUTO_INCREMENT` primary-key values are left out of the generated `INSERT` statements unless the
  env's `data` block sets `preserve_ids = true`
  (https://atlasgo.io/atlas-schema/hcl#preserving-explicit-ids). Preserving a literal `0` also needs
  `NO_AUTO_VALUE_ON_ZERO` in the session's `sql_mode`.

## Input Variables

Input variables work as in any Atlas HCL schema (https://atlasgo.io/atlas-schema/input-variables). A
variable without a default that gets no value fails with
`missing value for required variable "tenant"`.
