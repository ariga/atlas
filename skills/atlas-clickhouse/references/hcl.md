# Writing ClickHouse Schemas in HCL

The rules of the HCL schema language that matter on ClickHouse, and the places where HCL behaves
differently from SQL. The language: https://atlasgo.io/atlas-schema/hcl. Every ClickHouse block and
attribute: https://atlasgo.io/hcl/clickhouse. Expressions for engines, types, defaults, and TTLs:
https://atlasgo.io/guides/clickhouse/advanced-hcl-schema.

The ClickHouse driver needs `atlas login`. If `atlas whoami` shows no login, ask the user to run
`atlas login`; a user without an account can start a free trial with it.

## Files

- Name ClickHouse schema files `*.ch.hcl` and test files `*.test.hcl`, so editors apply the right
  schema (https://atlasgo.io/atlas-schema/hcl).
- A `schema` block is a ClickHouse database, and every `table` sets `schema`. A file with several
  `schema` blocks, or with users, roles, policies, profiles, named collections, or functions, needs
  server scope (`references/dev-database.md`).
- A SQL schema works too; the dev database parses it. Inspect to see the HCL Atlas reads back:
  `atlas schema inspect --env <name>`.

## Databases

```hcl
schema "app" {
  engine   = Atomic                       // Atomic | Lazy | sql("Replicated(...)")
  comment  = "Application data"
  settings = {
    disk = "default"
  }
}
```

Atlas plans `settings` changes, but not `engine` or `comment` changes (`references/objects.md`).
Database settings such as `disk` exist only on recent ClickHouse releases. ClickHouse 24.10 rejects
the example above with
``Database engine `Atomic` cannot have parameters, primary_key, order_by, sample_by, settings``: check that
the target's version accepts the setting before declaring it.

## Columns and Types

- Type names are ClickHouse's: `UInt64`, `Int32`, `String`, `Date`, `DateTime`, `DateTime64(3)`,
  `UUID`, `Bool`, `Float64`, `Decimal(10,2)`, `FixedString(16)`, `IPv4`, `JSON`, `Dynamic`. The only
  aliases Atlas accepts are `Float`, `Double`, `Tinyint`, `Smallint`, `Int`, `Bigint`, and `Object`,
  with that casing; they read back as `Float32`, `Float64`, `Int8`, `Int16`, `Int32`, `Int64`, and
  `JSON`.
- A type Atlas does not know, such as a SQL alias (`VARCHAR`, `TEXT`, `BOOLEAN`, `TIMESTAMP`), loads
  without an error and fails at plan time with
  `formatting type for column "<name>": unsupported type ...`. Write the ClickHouse type.
- Types read back normalized: `Decimal64(4)` as `Decimal(18, 4)`, a bare `Decimal` as `Decimal(10, 0)`,
  and `Dynamic` as `Dynamic(max_types=32)`. `DateTime` and `DateTime64` take a timezone:
  `sql("DateTime64(3, 'UTC')")`.
- Wrapped and nested types go in `sql()`: `sql("LowCardinality(String)")`, `sql("Array(Float32)")`,
  `sql("Map(String, String)")`, `sql("Tuple(u UInt64, a Array(String))")`, `sql("Nested(...)")`.
- A column is not nullable unless its type says so. Write the wrapper; `null = true` is optional with
  it, and fails without it:

  ```hcl
    column "deleted_at" {
      null = true
      type = sql("Nullable(DateTime)")
    }
  ```

  `null = true` with `type = DateTime` fails with an error such as:

  ```
  converting column "deleted_at" type: nullable column must have type wrapped with "Nullable()" (e.g., "Nullable(DateTime)")
  ```

  Atlas checks only the outer wrapper, so leave `null` unset on `sql("LowCardinality(Nullable(String))")`:
  `null = true` fails on it with the same error.
- `Enum` takes its values inline: `type = Enum("a", "b")` is written as `Enum('a', 'b')`, which
  ClickHouse stores and reads back as `Enum8('a' = 1, 'b' = 2)`. Explicit numbers go in `sql()`:
  `type = sql("Enum8('a' = 1, 'b' = 2, 'c' = -2)")`. Changing the number of an existing value cannot be
  applied in place (`references/objects.md`, Columns).
- `default` takes a literal or `sql()`. `default_kind` sets how it applies: `DEFAULT` (the default when
  `default` is set), `MATERIALIZED`, `EPHEMERAL`, or `ALIAS`:

  ```hcl
    column "created_at" {
      type         = DateTime
      default      = sql("now()")
      default_kind = MATERIALIZED
    }
  ```

  `MATERIALIZED`, `ALIAS`, and `EPHEMERAL` columns are left out of the row copy of a table rebuild and
  of seed data. A `default_kind` without `default` is ignored.
- `codec` takes an enum (`ZSTD`, `LZ4`, `LZ4HC`, `DoubleDelta`, `Gorilla`, `FPC`, `T64`, `NONE`, and
  the AES codecs), a string such as `"ZSTD(1)"`, or `sql("CODEC(Delta(8), LZ4)")`. `Delta`, `GCD`, and
  `DEFLATE_QPL` have no enum: write them as a string. Codecs read back with their default level, such as
  `ZSTD(1)` for `ZSTD`.
- `ttl` on a column takes a string or `sql()`: `ttl = sql("date + toIntervalMonth(1)")`.
- A rename is planned as a drop and an add, which loses the data, unless it is declared. In a
  non-interactive run, declare it with `renamed_from` on the column or the table; Atlas then writes
  `RENAME COLUMN` or `RENAME TABLE`. A SQL schema uses `-- atlas:renamed_from <old>` on the line before.

## Tables

```hcl
table "events" {
  schema = schema.app
  engine = MergeTree                      // or sql("ReplacingMergeTree(version)")
  ttl    = sql("event_date + toIntervalMonth(3)")
  settings = {
    index_granularity = 8192
  }
  column "tenant_id" {
    type = UInt32
  }
  column "event_date" {
    type = Date
  }
  column "name" {
    type  = sql("LowCardinality(String)")
    codec = ZSTD
  }
  primary_key {
    columns = [column.tenant_id, column.event_date]
  }
  sort {
    columns = [column.tenant_id, column.event_date, column.name]
  }
  partition {
    on {
      expr = "toYYYYMM(event_date)"
    }
  }
  index "name_bf" {
    type        = bloom_filter            // minmax | bloom_filter | "tokenbf_v1(...)" | sql("set(100)")
    granularity = 4
    on {
      expr = "name"
    }
  }
  projection "by_name" {
    as = "SELECT name, count() GROUP BY name"
  }
  check "valid_tenant" {
    expr = "tenant_id > 0"
  }
}
```

- `sort` is `ORDER BY`, and `primary_key` is `PRIMARY KEY`, which must be a prefix of the sort key.
  Without `sort`, the table sorts by its primary key. Each takes `columns`, or `on` blocks with a
  `column` or an `expr`.
- Declare `primary_key` on every MergeTree table. A table with `sort` and no `primary_key` is created
  with `PRIMARY KEY tuple()`, which has no primary index, unlike plain ClickHouse, where `ORDER BY`
  alone also sets the primary key. A table created outside Atlas with `ORDER BY` alone has that
  implicit primary key: inspection reads it back, and a schema that leaves it out fails to plan with
  `altering table "<name>": unsupported change type`.
- `partition` is `PARTITION BY`, with `columns` or `on` blocks.
- Changing `sort`, `primary_key`, `partition`, or `engine` rebuilds the table
  (`references/migrations.md`, Table Rebuilds). Pick them before the table holds data.
- `index` blocks are data-skipping indexes: `type` is required, and the expression goes in an `on`
  block, which takes `expr` only. The types are `minmax`, `set`, `tokenbf_v1`, `ngrambf_v1`,
  `bloom_filter`, `vector_similarity`, and `text`; only `minmax` and `bloom_filter` are enums. Any other
  type, such as `inverted`, fails with
  `converting index "<name>": unsupported index type "<type>"`. `granularity` defaults to 1, or
  100000000 for `vector_similarity` and `text` indexes.
- `check` takes `expr`; `assume = true` makes it `ASSUME` instead of `CHECK`. A change of kind recreates
  the constraint.
- A table-level `ttl`, `settings`, and `comment` are changed in place with `ALTER TABLE ... MODIFY`,
  except on the engines that are recreated for any change (`references/objects.md`, Table Engines).
- `settings` values, here and in databases, dictionaries, and `refresh` blocks, are written as follows:
  strings quoted, booleans as `1` or `0`, integers as is, and fractions with six decimals (`0.5` as
  `0.500000`). A list or a map fails with
  `converting settings for key "<key>": unsupported value type ...`.
- A `TimeSeries` table that names its target tables uses an `engine "TimeSeries"` block with `data`,
  `tags`, or `metrics` table references. The referenced tables are in the same database, and hold the
  columns the engine expects; Atlas adds the standard columns the block leaves out.

## Views and Materialized Views

```hcl
view "active_users" {
  schema   = schema.app
  as       = "SELECT id, name FROM users WHERE active = 1"
  security = INVOKER                      // DEFINER | INVOKER | NONE
}

materialized "events_daily_mv" {
  schema = schema.app
  to     = table.events_daily
  as     = "SELECT event_date, count() AS c FROM events GROUP BY event_date"
}

materialized "top_names" {
  schema = schema.app
  engine = MergeTree
  as     = "SELECT name, count() AS c FROM events GROUP BY name"
  column "name" {
    type = String
  }
  column "c" {
    type = UInt64
  }
  primary_key {
    columns = [column.name]
  }
  refresh {
    expr     = "EVERY 1 HOUR"
    settings = {
      refresh_retries = 3
    }
  }
}
```

- A materialized view sets either `to` or `engine`, not both. `populate` works only with `engine`.
  `to` takes a table, view, or materialized view reference.
- A materialized view's `sort` does not default to its `primary_key`, as a table's does.
- `definer` and `security` apply to both kinds. Declare `security` on normal views
  (`references/objects.md`, Views).
- Materialized views are recreated for any change but `refresh` (`references/objects.md`).
- At server scope, name each table with its database in `as`: `FROM app.events`
  (`references/dev-database.md`, Scope).

## Dictionaries, Functions, and Other Objects

```hcl
dictionary "users_dict" {
  schema = schema.app
  source = sql("CLICKHOUSE(TABLE users)")
  layout = sql("HASHED()")
  key "id" {
    type = UInt64
  }
  attribute "name" {
    type = String
  }
  lifetime {
    min = 10
    max = 100
  }
}

function "plus_one" {
  as = "(x) -> x + 1"
}
```

- Set `lifetime` on every dictionary that does not use the `DIRECT` layout: without it, the dictionary
  never reloads (`references/objects.md`, Dictionaries).
- A WASM function sets `lang = WASM`, `module`, `returns`, `source`, and `arg` blocks instead of `as`,
  and takes `abi`, `deterministic`, `hash`, and `settings`. A `wasm_module` block takes `code` and an
  optional `hash`, both as `sql()` expressions (https://atlasgo.io/guides/clickhouse/wasm).
- A `named_collection` holds `item` blocks with `value` and an optional `overridable`. Pass secret values
  through input variables.

## Roles, Users, and Grants

Covered in `references/security.md`.

## Seed Data

Rows are declared next to the table in a `data` block, and applied only when the env enables data sync
(https://atlasgo.io/guides/clickhouse/seed-data):

```hcl
data {
  table = table.countries
  rows = [
    { id = 1, code = "US", name = "United States" },
    { id = 2, code = "IL", name = "Israel" },
  ]
}
```

- The env's `data` block sets `mode` (`INSERT`, `UPSERT`, or `SYNC`) and `include`. `SYNC` needs
  `max_rows`, or fails with an error such as:

  ```
  syncing data between databases requires setting the "max_rows" configuration to prevent full table scans.
  ```

  A table with more rows than `max_rows` fails with
  `table "<name>" exceeds the configured max_rows limit of <n>; increase the limit or exclude this table`.
- Rows are matched by the table's primary key. A table without one fails with
  `table "<name>" has no primary key`, and a key with an expression, such as `toDate(ts)`, with
  `table "<name>" primary key part <i> is an expression`. MergeTree does not enforce the key's
  uniqueness, so keep seed keys unique.
- Updates are `ALTER TABLE ... UPDATE` mutations, and deletes are `DELETE FROM` on MergeTree tables and
  `ALTER TABLE ... DELETE` on other engines, each with `SETTINGS mutations_sync = 1`, so the statement
  waits for the change. Keep seed data to small lookup tables.

## Input Variables

Input variables work as in any Atlas HCL schema (https://atlasgo.io/atlas-schema/input-variables), and
carry every secret: passwords, named collection values, and credential settings.
