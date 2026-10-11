# Writing Redshift Schemas in HCL

The rules of the HCL schema language that matter on Redshift, and the places where HCL behaves
differently from SQL. The language: https://atlasgo.io/atlas-schema/hcl. Column types:
https://atlasgo.io/atlas-schema/hcl-types. Every Redshift block and attribute:
https://atlasgo.io/hcl/redshift.

Every Redshift command needs `atlas login`. If `atlas whoami` shows no login, ask the user to run
`atlas login`; a user without an account can start a free trial with it.

## Files

- Name Redshift schema files `*.rs.hcl` and test files `*.test.hcl`, so editors apply the Redshift
  schema (https://atlasgo.io/atlas-schema/hcl#file-naming-for-editor-support).
- A block or attribute the Redshift schema does not define fails with `unknown block type ...` or
  `unknown attribute ...`, so do not copy PostgreSQL HCL (indexes, keys, sequences, triggers) into a
  Redshift file.
- One file can hold several `schema` blocks, which needs database scope (`references/dev-database.md`).
- Two tables with the same name in different schemas take a second label, the schema:
  `table "sales" "orders" { schema = schema.sales }`, referenced as `table.sales.orders`.

## Columns

```hcl
schema "public" {
  comment = "Standard public schema"
}

table "events" {
  schema  = schema.public
  comment = "Raw events"
  column "id" {
    type = bigint
    identity {
      generated = ALWAYS
      seed      = 1
      step      = 1
    }
  }
  column "user_id" {
    type   = integer
    encode = AZ64
  }
  column "kind" {
    type    = varchar(64)
    default = "unknown"
  }
  column "payload" {
    null = true
    type = super
  }
  column "created_at" {
    type    = timestamp
    default = sql("getdate()")
  }
}
```

- A column without `null` is `NOT NULL`, the opposite of SQL. Write `null = true` for a nullable
  column. A new `NOT NULL` column on an existing table needs a `default` (`references/objects.md`).
- A column takes `type`, `null`, `default`, `comment`, `encode`, and an `identity` block.
- `default` takes a literal (a bool, a number, or a string, which becomes a quoted SQL literal) or an
  expression in `sql()`: `default = 0`, `default = "unknown"`, `default = sql("1 + 1")`. A column with
  an identity has no default.
- `encode` takes `RAW`, `AZ64`, `BYTEDICT`, `DELTA`, `DELTA32K`, `LZO`, `MOSTLY8`, `MOSTLY16`,
  `MOSTLY32`, `RUNLENGTH`, `TEXT255`, `TEXT32K`, or `ZSTD`. Without it, Redshift picks one, and
  inspection writes it back (`references/objects.md`).
- `identity` takes `generated` (`ALWAYS` or `BY_DEFAULT`, the default), `seed`, and `step` (both 1 by
  default). `identity {}` is a `BY_DEFAULT` identity starting at 1.
- The Redshift schema has no `renamed_from`. A renamed table is always planned as a drop and a create.
  A renamed column is planned as a drop and an add unless the rename is confirmed in an interactive
  run. Both lose data. In a non-interactive run, such as an agent or CI, write the
  `ALTER TABLE ... RENAME` in a hand-written migration.

## Column Types

Write parameters inline: `varchar(255)`, `numeric(10,2)`, `varbyte(10000)`. Multi-word types use
underscores:

| Types | HCL | Inspected as |
|-------|-----|--------------|
| Integers | `smallint`, `integer`, `int`, `bigint`, `int2`, `int4`, `int8` | `smallint`, `integer`, `bigint` |
| Exact | `numeric`, `numeric(10,2)`, `decimal`, `decimal(10,2)` | `numeric(18)` without a precision |
| Floating point | `real`, `float4`, `float`, `float8`, `double_precision` | `real`, `double_precision` |
| Fixed strings | `char`, `char(10)`, `character(10)`, `nchar(10)` | `character(1)` without a length |
| Variable strings | `varchar`, `varchar(10)`, `character_varying(10)`, `nvarchar(10)`, `text` | `character_varying(256)` without a length; `text` is `character_varying(256)` |
| Date and time | `date`, `time`, `timetz`, `timestamp`, `timestamptz` | the same |
| Intervals | `sql("interval year to month")`, `sql("interval day to second")` | the same |
| Boolean | `boolean`, `bool` | `boolean` |
| Binary | `varbyte`, `varbyte(10000)`, `varbinary`, `binary_varying` | `binary_varying(64000)` without a length |
| Semi-structured, sketches, spatial | `super`, `hllsketch`, `geometry`, `geography` | the same |

The inspected form is what a plan compares against, so declare explicit lengths, or plan with a dev
URL, which normalizes both sides.

## Tables

```hcl
table "events" {
  schema = schema.public
  column "user_id" {
    type = integer
  }
  column "created_at" {
    type = timestamp
  }
  distribution {
    style  = KEY
    column = column.user_id
  }
  sort {
    style   = COMPOUND
    columns = [column.created_at]
  }
}
```

- A table takes `schema`, `comment`, `column`, `distribution`, and `sort`; nothing else.
- `distribution` takes `style` (`AUTO`, `EVEN`, `ALL`, `KEY`) and `column`, which only `KEY` accepts:
  `parsing <table>.distribution: column only allowed with "KEY" style`.
- `sort` takes `style` (`AUTO`, `COMPOUND`, `INTERLEAVED`) and `columns`. `COMPOUND` and `INTERLEAVED`
  need columns: `parsing <table>.sort: missing columns for "COMPOUND" sort`.
- Without the blocks, the table gets `DISTSTYLE AUTO` and `SORTKEY AUTO`, and inspection writes both
  blocks back (`references/objects.md`, Distribution, Sort Keys, and Encodings).

## Views and Materialized Views

```hcl
view "active_users" {
  schema     = schema.public
  comment    = "Users seen in the last 30 days"
  as         = "SELECT id, name FROM public.users WHERE last_seen > dateadd(day, -30, getdate())"
  depends_on = [table.users]
}

materialized "daily_counts" {
  schema       = schema.public
  as           = "SELECT user_id, count(*) AS n FROM public.events GROUP BY user_id"
  depends_on   = [table.events]
  auto_refresh = true
  distribution {
    style  = KEY
    column = column.user_id
  }
  sort {
    style   = COMPOUND
    columns = [column.user_id]
  }
  column "user_id" {
    null = true
    type = integer
  }
  column "n" {
    null = true
    type = bigint
  }
}
```

- `as` holds the query; `depends_on` orders the view after the objects it reads.
- A late-binding view keeps `WITH NO SCHEMA BINDING` at the end of `as`, with schema-qualified table
  names.
- A view's `column` blocks take `type`, `null`, and `comment`; inspection writes them.
- A materialized view's `distribution` takes `EVEN` (the default), `ALL`, or `KEY`, and its `sort` takes
  only `COMPOUND`. A `distribution` or `sort` column must be one of the view's `column` blocks.

## Functions and Procedures

```hcl
function "add_numbers" {
  schema = schema.public
  lang   = SQL
  arg "a" {
    type = integer
  }
  arg "b" {
    type = integer
  }
  return     = integer
  as         = "SELECT $1 + $2"
  volatility = IMMUTABLE
}

procedure "process_data" {
  schema   = schema.public
  lang     = PLpgSQL
  security = DEFINER
  arg "input" {
    type = integer
  }
  arg "output" {
    type = integer
    mode = OUT
  }
  as = "BEGIN output := input * 2; END;"
}
```

- `as` holds the body itself, never wrapped in `$$`. SQL function bodies refer to arguments as `$1`,
  `$2`.
- A function takes `schema`, `lang`, `as`, `return`, `volatility` (`VOLATILE`, `STABLE`, `IMMUTABLE`),
  `arg`, and `depends_on`. A procedure takes `schema`, `lang`, `as`, `security` (`INVOKER`, `DEFINER`),
  `arg` with `mode` (`IN`, `OUT`, `INOUT`), and `depends_on`.
- Overloads are blocks with the same label and different `arg` types.

## External Schemas

The `external` block's label selects the source, and each source takes its own options:

| Source | Options |
|--------|---------|
| `data_catalog` | `database`, `region`, `iam_role`, `catalog_role` |
| `hive_metastore` | `database`, `uri`, `port`, `iam_role` |
| `postgres` | `database`, `schema`, `uri`, `port`, `iam_role`, `secret_arn` |
| `mysql` | `database`, `uri`, `port`, `iam_role`, `secret_arn` |
| `redshift` | `database`, `schema` |
| `kinesis` | `iam_role`, `region` |
| `msk` | `uri`, `cluster_arn`, `iam_role`, `region`, `authentication`, `authentication_arn`, `secret_arn` |
| `kafka` | `uri`, `iam_role`, `region`, `authentication`, `authentication_arn`, `secret_arn` |

```hcl
schema "spectrum" {
  external "data_catalog" {
    database = "spectrumdb"
    region   = "us-east-1"
    iam_role = "arn:aws:iam::123456789012:role/spectrum"
  }
}
```

`authentication` takes `NONE`, `IAM`, or `MTLS`. A schema holds one `external` block; a second fails
with `redshift: schema "s" contains multiple external blocks`.

## Datashares

Needs `mode "redshift" { datashares = true }` on the env (`references/objects.md`, Datashares):

```hcl
datashare "sales_share" {
  public_accessible = false
  namespaces        = ["00000000-0000-0000-0000-000000000000"]
  accounts          = ["000000000000"]
  schema {
    for     = schema.sales
    objects = [table.orders, view.summary, materialized.rollup, function.calc]
  }
  schema {
    for                = schema.events
    include_new_except = [table.audit]
  }
}
```

- `objects`, `include_new`, and `include_new_except` are mutually exclusive in one `schema` block.
- `namespaces` or `accounts` is required.
- `for` and `objects` take references, or strings for schemas and objects the state does not define.
  A schema the state does not define takes its objects as strings too, or fails with
  `an object of schema "..." must be shared by name: its definition is not in this context`.
  `include_new_except` needs the schema's definition in the state, or fails with
  `include_new_except requires the definition of schema "..." in this context`.
- An object reference must point into the block's `for` schema, or it fails with
  `object "..." is not defined in the shared schema "..."`.

## Roles, Users, and Grants

The rules are in `references/security.md`; the HCL details:

- `role` takes `name`, `comment`, `group`, `external`, `member_of`, and `membership` blocks.
- `user` takes the same, plus `password`, `valid_until`, `superuser`, `create_db`, `conn_limit`, and
  `params`, a map of session parameters.
- `permission` takes `to`, `for`, `privileges`, and `grantable`. `for_each` with `each.value` repeats
  one block over a list of objects.
- A password comes in through an input variable, never as a literal
  (https://atlasgo.io/atlas-schema/hcl#sensitive-information):

  ```hcl
  variable "etl_password" {
    type = string
  }

  user "etl" {
    password = var.etl_password
  }
  ```

## Input Variables

A schema file declares the values it expects, and `atlas.hcl` passes them through `data "hcl_schema"`
(https://atlasgo.io/atlas-schema/input-variables):

```hcl
data "hcl_schema" "app" {
  path = "schema.rs.hcl"
  vars = {
    etl_password = getenv("ETL_PASSWORD")
  }
}

env "prod" {
  url = getenv("REDSHIFT_URL")
  dev = getenv("REDSHIFT_DEV_URL")
  src = data.hcl_schema.app.url
}
```

A variable without a default that gets no value fails with `missing value for required variable "..."`.
`getenv` of an unset environment variable returns an empty string, which counts as a value: the
variable is silently empty, and a password becomes `''`. Check that the environment variable is set.
