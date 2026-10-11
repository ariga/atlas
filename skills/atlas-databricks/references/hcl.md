# Writing Databricks Schemas in HCL

The rules of the HCL schema language that matter on Databricks, and the places where HCL behaves
differently from SQL. The language: https://atlasgo.io/atlas-schema/hcl. Column types:
https://atlasgo.io/atlas-schema/hcl-types. Every Databricks block and attribute:
https://atlasgo.io/hcl/databricks. The behavior behind each block is in `references/objects.md`.

## Files

- Name Databricks schema files `*.dbx.hcl` and test files `*.test.hcl`, so editors apply the right
  schema (https://atlasgo.io/atlas-schema/hcl#file-naming-for-editor-support).
- A schema file targets one database engine: HCL does not abstract the differences between engines.
- The top-level blocks are `schema`, `table`, `view`, `materialized`, `function`, `procedure`, and
  `permission`. There is no `index`, `trigger`, `sequence`, `role`, `user`, or `data` block.
- At catalog scope, declare every schema the catalog keeps, `schema "default" {}` included
  (`references/dev-database.md`, Scope).

## Schemas

The examples in this file build one schema, `retail_analytics`, taken from the tested examples of the
Databricks changelog (https://atlasgo.io/changelog/databricks-object-attribute-coverage).

```hcl
schema "default" {
}

schema "retail_analytics" {
  comment   = "curated retail data for the analytics team"
  collation = "UTF8_BINARY"
}
```

```sql
CREATE SCHEMA `retail_analytics` COMMENT 'curated retail data for the analytics team' DEFAULT COLLATION UTF8_BINARY;
```

## Columns

- A column without `null` is `NOT NULL`, the opposite of SQL. Write `null = true` for a nullable column.
- `default` takes a literal or an expression in `sql()`: `default = 1`, `default = "unknown"`,
  `default = sql("current_timestamp()")`. A default needs a Delta feature, which Atlas sets
  (`references/objects.md`, Defaults).
- A column takes at most one of `default`, `as`, and `identity`.
- `as` holds a generated column's expression, without `STORED` or `VIRTUAL`.
- `identity` takes `by_default` (false by default, `GENERATED ALWAYS`), `start`, and `increment` (both 1
  by default). The column must be `BIGINT`.
- `collation` sets a string column's collation, and `comment` its comment.
- `mask` sets a column mask: `function` references a `function` block, and `using_columns` lists the
  extra columns passed to it. The function takes the masked column first, then those columns, so it
  needs one argument more than `using_columns` lists. Atlas does not check the count; a mismatch
  fails when Databricks creates the table.
- There is no `renamed_from` on Databricks: it fails with
  `unknown attribute renamed_from in block of type column`. Renaming a column in the schema plans a
  drop and an add, so write renames by hand (`references/objects.md`, Renames).

The `orders` table in Tables below shows an identity, a collation, a default, and a generated column.

## Column Types

Types are written in upper case, as Databricks spells them:

| Types | HCL |
|-------|-----|
| Integers | `TINYINT` (`BYTE`), `SMALLINT` (`SHORT`), `INT` (`INTEGER`), `BIGINT` (`LONG`) |
| Exact and floating point | `DECIMAL(10,2)` (`DEC`, `NUMERIC`), `FLOAT` (`REAL`), `DOUBLE` |
| Strings and binary | `STRING`, `BINARY` |
| Date and time | `DATE`, `TIMESTAMP`, `TIMESTAMP_NTZ` |
| Boolean | `BOOLEAN` |
| Composite | `sql("ARRAY<STRING>")`, `sql("MAP<STRING, INT>")`, `sql("STRUCT<a:INT, b:DECIMAL(10,2)>")` |

- Aliases normalize on the dev catalog: `LONG` reads back as `BIGINT`, so prefer the canonical names.
- Write `DECIMAL` with its precision and scale, `DECIMAL(10,2)`, as inspection prints it.
- `VARIANT`, `INTERVAL`, `GEOGRAPHY`, `GEOMETRY`, and `OBJECT` have no HCL form that Atlas can write
  (`references/objects.md`, Types).

## Tables

| Attribute or block | Writes |
|--------------------|--------|
| `schema` (required) | The schema the table belongs to |
| `comment` | `COMMENT '...'` |
| `properties = { "key" = "value" }` | `TBLPROPERTIES ('key' = 'value')` |
| `location = "s3://..."` | `LOCATION '...'`, an external table |
| `provider = "parquet"` | `USING PARQUET`; Delta is the default |
| `collation = "UTF8_LCASE"` | `DEFAULT COLLATION UTF8_LCASE` |
| `depends_on = [...]` | An ordering dependency on other objects |
| `primary_key "name" { columns = [...] }` | `CONSTRAINT name PRIMARY KEY (...)`; the name is optional |
| `foreign_key "name" { columns, ref_columns }` | `CONSTRAINT name FOREIGN KEY (...) REFERENCES ...`, with no actions |
| `check "name" { expr = "..." }` | `ADD CONSTRAINT name CHECK (...)`; the name is required |
| `partition { columns = [...] }` | `PARTITIONED BY (...)` |
| `cluster { columns = [...] }` or `cluster { auto = true }` | `CLUSTER BY (...)` or `CLUSTER BY AUTO` |
| `row_filter { function, columns }` | `WITH ROW FILTER fn ON (...)` |

```hcl
table "orders" {
  schema   = schema.retail_analytics
  comment  = "one row per placed order"
  provider = "delta"
  properties = {
    "delta.enableDeletionVectors" = "true"
    "delta.enableChangeDataFeed"  = "true"
  }
  column "order_id" {
    null    = false
    type    = BIGINT
    comment = "surrogate key"
    identity {
      by_default = true
      start      = 1000
      increment  = 1
    }
  }
  column "customer_email" {
    null      = false
    type      = STRING
    collation = "UTF8_LCASE"
    comment   = "compared case-insensitively"
  }
  column "region" {
    null    = false
    type    = STRING
    default = "EU"
  }
  column "total_amount" {
    null    = false
    type    = DECIMAL(12,2)
    comment = "order total in EUR"
  }
  column "total_with_vat" {
    null = true
    type = DECIMAL(12,2)
    as   = "CAST(total_amount * 1.21 AS DECIMAL(12,2))"
  }
  column "placed_on" {
    null = false
    type = DATE
  }
  primary_key "pk_orders" {
    columns = [column.order_id]
  }
  check "region_is_known" {
    expr = "region IN ('EU', 'US', 'APAC')"
  }
  partition {
    columns = [column.placed_on]
  }
}
```

The SQL Atlas plans for it. The column default adds `delta.feature.allowColumnDefaults` to the
properties, and the check is a separate `ALTER TABLE`:

```sql
CREATE TABLE IF NOT EXISTS `retail_analytics`.`orders` (
  `order_id` BIGINT NOT NULL GENERATED BY DEFAULT AS IDENTITY (START WITH 1000 INCREMENT BY 1) COMMENT 'surrogate key',
  `customer_email` STRING COLLATE UTF8_LCASE NOT NULL COMMENT 'compared case-insensitively',
  `region` STRING NOT NULL DEFAULT 'EU',
  `total_amount` DECIMAL(12,2) NOT NULL COMMENT 'order total in EUR',
  `total_with_vat` DECIMAL(12,2) GENERATED ALWAYS AS (CAST(total_amount * 1.21 AS DECIMAL(12,2))),
  `placed_on` DATE NOT NULL,
  CONSTRAINT `pk_orders` PRIMARY KEY (`order_id`)
)
PARTITIONED BY (`placed_on`)
COMMENT 'one row per placed order'
TBLPROPERTIES ('delta.enableChangeDataFeed' = 'true', 'delta.enableDeletionVectors' = 'true', 'delta.feature.allowColumnDefaults' = 'supported');
ALTER TABLE `retail_analytics`.`orders` ADD CONSTRAINT `region_is_known` CHECK (region IN ('EU', 'US', 'APAC'));
```

- A foreign key names the referenced columns: `foreign_key "fk_orders_customer" { columns =
  [column.customer_id], ref_columns = [table.customers.column.id] }`. Databricks keeps it
  informational, and it takes no `on_delete` or `on_update`.
- `cluster { columns = [...] }` replaces `partition` for liquid clustering. `cluster { auto = false }`
  fails: omit the block instead.
- Inside a table, `column.<name>` refers to its own columns. Any other column is
  `table.<table>.column.<column>`.
- Two tables with the same name in different schemas take a second label, the schema:
  `table "sales" "orders" { schema = schema.sales }`. References then include it:
  `table.sales.orders.column.id`.

## Views

```hcl
view "recent_orders" {
  schema         = schema.retail_analytics
  comment        = "orders placed in the last 30 days"
  collation      = "UTF8_BINARY"
  schema_binding = BINDING
  properties = {
    "owner" = "analytics-team"
  }
  column "order_id" {
    null    = false
    type    = BIGINT
    comment = "carried over from the orders table"
  }
  column "region" {
    null = false
    type = STRING
  }
  column "total_amount" {
    null = false
    type = DECIMAL(12,2)
  }
  as         = "SELECT order_id, region, total_amount FROM retail_analytics.orders WHERE placed_on > date_sub(current_date(), 30)"
  depends_on = [table.orders]
}
```

- `as` is required. `schema_binding` takes `BINDING`, `COMPENSATION`, `TYPE_EVOLUTION`, or `EVOLUTION`.
- `column` blocks are optional, and list the view's column aliases. Each needs a `type` and takes
  `null` and `comment`; the type is not written into the DDL. A view column takes no `default` and no
  `mask`, and a view takes no `primary_key` or `expect`: those fail with
  `unknown attribute default in block of type column`, `unknown block type mask in block of type column`,
  and `unknown block type primary_key in block of type view`.
- SQL bodies can interpolate names, so a rename follows them: `FROM ${table.orders.name}`.

## Materialized Views

```hcl
materialized "revenue_by_region" {
  schema   = schema.retail_analytics
  comment  = "daily revenue rolled up per region"
  provider = "delta"
  properties = {
    "pipelines.autoOptimize.managed" = "true"
  }
  column "region" {
    null = false
    type = STRING
  }
  column "revenue_day" {
    null = false
    type = DATE
  }
  column "revenue" {
    null = true
    type = DECIMAL(18,2)
  }
  primary_key "pk_revenue_by_region" {
    columns = [column.region, column.revenue_day]
  }
  expect "revenue_is_not_negative" {
    expr         = "revenue >= 0"
    on_violation = FAIL_UPDATE
  }
  expect "region_is_known" {
    expr         = "region IN ('EU', 'US', 'APAC')"
    on_violation = DROP_ROW
  }
  cluster {
    columns = [column.region]
  }
  refresh {
    cron      = "0 0 2 * * ?"
    time_zone = "Europe/Amsterdam"
  }
  row_filter {
    function = function.filter_by_region
    columns  = [column.region]
  }
  as         = "SELECT region, placed_on AS revenue_day, CAST(SUM(total_amount) AS DECIMAL(18,2)) AS revenue FROM retail_analytics.orders GROUP BY region, placed_on"
  depends_on = [table.orders]
}
```

```sql
CREATE OR REPLACE MATERIALIZED VIEW `retail_analytics`.`revenue_by_region` (
  `region` STRING NOT NULL,
  `revenue_day` DATE NOT NULL,
  `revenue` DECIMAL(18,2),
  CONSTRAINT `revenue_is_not_negative` EXPECT (revenue >= 0) ON VIOLATION FAIL UPDATE,
  CONSTRAINT `region_is_known` EXPECT (region IN ('EU', 'US', 'APAC')) ON VIOLATION DROP ROW,
  CONSTRAINT `pk_revenue_by_region` PRIMARY KEY (`region`, `revenue_day`)
)
CLUSTER BY (`region`)
COMMENT 'daily revenue rolled up per region'
TBLPROPERTIES ('pipelines.autoOptimize.managed' = 'true')
SCHEDULE REFRESH CRON '0 0 2 * * ?' AT TIME ZONE 'Europe/Amsterdam'
WITH ROW FILTER `retail_analytics`.`filter_by_region` ON (`region`) AS SELECT region, placed_on AS revenue_day, CAST(SUM(total_amount) AS DECIMAL(18,2)) AS revenue FROM retail_analytics.orders GROUP BY region, placed_on;
```

- Columns are full definitions: `type`, `null`, `comment`, and `mask`, but no `default`.
- `expect` blocks need a name and `expr`; `on_violation` takes `DROP_ROW` or `FAIL_UPDATE`.
- `partition` and `cluster` are mutually exclusive. `provider`, `properties`, `collation`, `primary_key`,
  `foreign_key`, and `row_filter` work as on a table.
- `refresh` takes one of `every = "1 DAY"`, `cron = "0 0 8 * * ?"` with an optional
  `time_zone = "America/Los_Angeles"`, or `on_update = true` with an optional `at_most = "1 HOUR"`.
  `on_update = false` fails: omit the block instead.

## Functions

```hcl
function "filter_by_region" {
  schema          = schema.retail_analytics
  lang            = SQL
  comment         = "restricts rows to the region of the caller"
  deterministic   = false
  sql_data_access = READS_SQL_DATA
  arg "region" {
    type    = STRING
    comment = "the region column of the row"
  }
  return = BOOLEAN
  as     = "region = 'EU' OR is_account_group_member('admins')"
}

function "sales_rows" {
  schema = schema.retail_analytics
  lang   = SQL
  arg "min_id" {
    type = INT
  }
  return_table {
    column "id" {
      type = BIGINT
    }
    column "label" {
      type = STRING
    }
  }
  as = "SELECT CAST(min_id AS BIGINT), 'x'"
}

function "multiply_by" {
  schema = schema.retail_analytics
  lang   = PYTHON
  arg "a" {
    type = DOUBLE
  }
  arg "b" {
    type = DOUBLE
  }
  return = DOUBLE
  as     = <<-PY
    return a * b
  PY
}
```

- `lang` is `SQL` or `PYTHON`. A function takes one of `return` and `return_table`; with neither,
  planning fails with `missing return type for function "..."`.
- `sql_data_access` takes `CONTAINS_SQL` or `READS_SQL_DATA`.
- `<<-PY` strips the common indentation of a heredoc, which a Python body needs.

## Procedures

```hcl
procedure "archive_old_orders" {
  schema          = schema.retail_analytics
  lang            = SQL
  comment         = "moves orders older than the cutoff into cold storage"
  security        = INVOKER
  sql_data_access = MODIFIES_SQL_DATA
  arg "cutoff" {
    type    = DATE
    mode    = IN
    comment = "orders placed before this date are archived"
  }
  arg "archived_count" {
    type    = BIGINT
    mode    = OUT
    comment = "how many rows were archived"
  }
  arg "dry_run" {
    type    = BOOLEAN
    mode    = IN
    default = true
  }
  as = "BEGIN SET archived_count = (SELECT count(*) FROM retail_analytics.orders WHERE placed_on < cutoff); END"
}
```

- `lang` must be `SQL`. `security` takes `DEFINER` or `INVOKER`, and `sql_data_access` only
  `MODIFIES_SQL_DATA`. `deterministic` accepts only `false`.
- `mode` takes `IN` (the default), `OUT`, or `INOUT`. Only `IN` arguments take a `default`.

## Permissions

`permission` blocks need `mode { permissions = true }` on the env; the rules are in
`references/security.md`:

```hcl
permission {
  to         = "b21bc072-9110-4af2-81af-21a670c49233"
  for        = table.orders
  privileges = [SELECT, INSERT, UPDATE, DELETE, MODIFY, READ_METADATA]
}
```

```sql
GRANT DELETE, INSERT, MODIFY, READ METADATA, SELECT, UPDATE ON TABLE `retail_analytics`.`orders` TO `b21bc072-9110-4af2-81af-21a670c49233`;
```

`to` takes a principal's name, such as a group, or a service principal's application ID, as here.
`READ_METADATA` is written as `READ METADATA`, and the privileges are sorted.

## Seed Data

The Databricks HCL has no `data` block. Put seed rows in `INSERT` statements in a migration file, or
in Data Scripts (`references/objects.md`, Not Managed by Atlas).

## Input Variables

A schema file declares the values it expects, and `atlas.hcl` passes them through `data "hcl_schema"`
(https://atlasgo.io/atlas-schema/projects):

```hcl
variable "comment" {
  type    = string
  default = "curated retail data for the analytics team"
}

schema "retail_analytics" {
  comment   = var.comment
  collation = "UTF8_BINARY"
}
```

A schema's name cannot come from a variable. The `schema` block has no `name` attribute on
Databricks, and one fails with `unknown attribute name in block of type schema`. The name is the
block's label.
