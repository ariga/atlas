# Writing Snowflake Schemas in HCL

The rules of the HCL schema language that matter on Snowflake, and the places where HCL behaves
differently from SQL. The language: https://atlasgo.io/atlas-schema/hcl. Every Snowflake block and
attribute: https://atlasgo.io/hcl/snowflake. Which objects need a `mode` switch before Atlas plans
them: `references/objects.md`.

Every block below needs `atlas login`. If `atlas whoami` shows no login, ask the user to run
`atlas login`; a user without an account can start a free trial with it.

## Files

- Name Snowflake schema files `*.sf.hcl` and test files `*.test.hcl`, so editors apply the right
  schema (https://atlasgo.io/atlas-schema/hcl#file-naming-for-editor-support).
- A database-scoped env covers every schema in the database, so a file with several `schema` blocks
  needs database scope (`references/dev-database.md`).

## Names

- Block labels are exact names, written quoted in the SQL Atlas generates. `table "users"` creates
  `"users"`, and `table "USERS"` creates `USERS`. Snowflake folds unquoted names to uppercase, so SQL
  written by hand, view and task bodies, and `as` expressions reach a lowercase object only through
  double quotes: `SELECT "id" FROM "users"`.
- Inspection writes names as Snowflake stores them. A SQL schema with `CREATE TABLE t1 (c NUMBER)` reads
  back as `table "T1"` with `column "C"`.
- `schema "public"` is a different schema from `schema "PUBLIC"`. Every new database has `PUBLIC`.
- Every table kind shares one namespace in a schema: a `table` and a `dynamic_table` with the same
  name in one schema fail with `conflict table name "..." in schema "..."`.
- Two objects with the same name in different schemas take the schema as a first label, and references
  include it:

  ```hcl
  table "PUBLIC" "users" {
    schema = schema.PUBLIC
    column "id" {
      type = NUMBER
    }
  }
  table "SALES" "users" {
    schema = schema.SALES
    column "id" {
      type = NUMBER
    }
  }
  ```

  A view on one of them uses `depends_on = [table.PUBLIC.users]`.

## Schemas

```hcl
schema "BRONZE" {
  comment                 = "landing zone"
  retention_time          = 1
  max_data_extension_time = 14
  managed_access          = true
  transient               = false
  default_ddl_collation   = "en-ci"
}
```

- `transient` is fixed at creation (`references/objects.md`). `managed_access = true` is
  `WITH MANAGED ACCESS`.
- Iceberg defaults for the schema's tables: `external_volume`, `catalog`, `iceberg_version_default`,
  `replace_invalid_characters`, `storage_serialization_policy`, `iceberg_merge_on_read_behavior`.
- Changing schema attributes needs database scope (`references/dev-database.md`).

## Columns

- A column without `null` is `NOT NULL`, the opposite of SQL. Write `null = true` for a nullable column.
- `default` takes a literal or an expression in `sql()`:

  ```hcl
  column "status" {
    type    = VARCHAR(20)
    default = "pending"
  }
  column "active" {
    type    = BOOLEAN
    default = true
  }
  column "created_at" {
    type    = TIMESTAMP_LTZ
    default = sql("CURRENT_TIMESTAMP()")
  }
  ```

  Setting or changing a default on an existing column needs the column recreated, except replacing
  one sequence default with another; see Column Changes in `references/objects.md`.
- An `identity` block is `AUTOINCREMENT`. `order` takes `ORDER` or `NOORDER` (the default):

  ```hcl
  column "id" {
    type = NUMBER
    identity {
      start     = 1
      increment = 1
    }
  }
  ```

- `as` makes a virtual column: `as = "\"id\" + 1"`. Write the expression the way Snowflake reports it,
  with quoted identifiers, so inspection reads it back unchanged.
- `collate = "en-ci"` sets a column's collation on regular and hybrid tables.
- `comment`, and `tag` and `policy` blocks, go on columns too (Tags and Policies below).

## Column Types

Types are written bare, with parameters inline: `NUMBER(10, 2)`, `VARCHAR(255)`, `TIMESTAMP_NTZ(6)`.

| Types | HCL |
|-------|-----|
| Numbers | `NUMBER`, `NUMERIC`, `DECIMAL`, `DEC` (precision and scale), `INT`, `INTEGER`, `BIGINT`, `SMALLINT`, `TINYINT`, `BYTEINT`, `FLOAT`, `FLOAT4`, `FLOAT8`, `REAL`, `DOUBLE`, `DECFLOAT` |
| Strings and binary | `VARCHAR`, `STRING`, `TEXT`, `CHAR`, `CHARACTER`, `BINARY`, `VARBINARY` (size) |
| Date and time | `DATE`, `TIME`, `TIMESTAMP`, `DATETIME`, `TIMESTAMP_NTZ`, `TIMESTAMP_LTZ`, `TIMESTAMP_TZ` (precision) |
| Other | `BOOLEAN`, `VARIANT`, `GEOMETRY`, `GEOGRAPHY`, `FILE`, `UUID` |
| Anything else | `sql("ARRAY(NUMBER)")`, `sql("OBJECT(a NUMBER, b VARCHAR)")`, `sql("MAP(VARCHAR, NUMBER)")`, `sql("VECTOR(INT, 8)")`, `sql("INTERVAL YEAR(4) TO MONTH")` |

Inspection prints the stored form: `NUMBER` reads back as `NUMBER(38)`, `TEXT` and `STRING` as
`VARCHAR(16777216)`, `BINARY` as `BINARY(8388608)`, and timestamps with precision `9`. In Iceberg tables
the defaults are `VARCHAR(134217728)` and precision `6`, and `NUMBER` needs an explicit precision, or
the schema fails with `iceberg table requires NUMBER to specify precision, e.g. NUMBER(38, 0)`. Both
spellings name the same type.

## Tables

```hcl
table "EVENTS" {
  schema                  = schema.BRONZE
  comment                 = "raw events"
  retention_time          = 1
  max_data_extension_time = 14
  change_tracking         = true
  cluster_by              = ["TO_DATE(\"OCCURRED_AT\")", "\"EVENT_TYPE\""]
  column "EVENT_ID" {
    type = NUMBER(38)
  }
  column "EVENT_TYPE" {
    type = VARCHAR(64)
  }
  column "OCCURRED_AT" {
    type = TIMESTAMP_NTZ(9)
  }
  primary_key {
    columns = [column.EVENT_ID]
  }
}
```

- `transient = true` is `CREATE TRANSIENT TABLE`, fixed at creation.
- `retention_time` is `DATA_RETENTION_TIME_IN_DAYS`. Atlas writes it on every `CREATE`, with the
  value the dev database reports when the schema does not set it. When the target's account or
  database default differs from the dev database's, declare `retention_time` on schemas and tables,
  or Atlas plans to set the dev database's value on the target.
- `cluster_by` takes expressions as strings, with quoted identifiers. Changing it is
  `ALTER TABLE ... CLUSTER BY (...)`; removing it is `DROP CLUSTERING KEY`.
- `change_tracking` and `enable_schema_evolution` turn the settings on. Leaving `change_tracking` out is
  no change: an incremental dynamic table turns it on for the tables it reads.
- Constraints: `primary_key`, `unique`, and `foreign_key` blocks, with `columns`, `ref_columns`, and
  `comment`. There are no `index` or `check` blocks, and no `on_delete` or `on_update`.

## Other Table Kinds

```hcl
dynamic_table "DAILY_EVENTS" {
  schema       = schema.SILVER
  target_lag   = "1 hour"   # or DOWNSTREAM
  refresh_mode = INCREMENTAL # AUTO, FULL
  initialize   = ON_CREATE  # ON_SCHEDULE; create-time only
  warehouse    = "TRANSFORM_WH"
  as           = "SELECT TO_DATE(\"OCCURRED_AT\") AS \"DAY\", COUNT(*) AS \"N\" FROM \"BRONZE\".\"EVENTS\" GROUP BY 1"
  depends_on   = [table.EVENTS]
  column "DAY" {}
  column "N" {}
}
```

- A `dynamic_table` takes its column types from the query; `column` blocks name the columns, and take
  `comment`, `null`, `tag`, and `policy`. Always set `warehouse` (Rule 8 in `SKILL.md`).
- A `hybrid_table` takes the same columns and constraints as a `table`, and needs a `primary_key`.
- An `external_table` needs a `location` block (`stage`, optional `path`), and takes `file_format` as a
  reference or a map, `pattern`, `partition_by`, `auto_refresh`, and `refresh_on_create`. Its columns are
  virtual, each with `as`:

  ```hcl
  external_table "EXT_CLAIM_FILES" {
    schema       = schema.PUBLIC
    partition_by = ["INGEST_DATE"]
    file_format = {
      FIELD_DELIMITER = "|"
      SKIP_HEADER     = 1
    }
    location {
      stage = stage.CLAIM_STAGE
      path  = "837/"
    }
    column "INGEST_DATE" {
      null = true
      type = DATE
      as   = "TO_DATE(SPLIT_PART(METADATA$FILENAME, '/', 3), 'YYYY-MM-DD')"
    }
  }
  ```

  Snowflake adds a `VALUE` column to every external table, holding each record of the file.
  Inspection leaves it out; do not declare it.

- An `iceberg_table` needs `external_volume` and `catalog` (`"SNOWFLAKE_MANAGED"` and `"SNOWFLAKE"` for
  Snowflake-managed tables), and takes `base_location`, `partition_by` or `cluster_by` (not both),
  `target_file_size`, `path_layout`, `iceberg_version`, and more. Change tracking is always on:
  `change_tracking = false` fails with
  `iceberg table "...": change tracking is always enabled and cannot be turned off`.

## Views

```hcl
view "ACTIVE_USERS" {
  schema     = schema.PUBLIC
  secure     = true
  comment    = "users seen this month"
  as         = "SELECT \"ID\" FROM \"PUBLIC\".\"USERS\" WHERE \"ACTIVE\""
  depends_on = [table.USERS]
}
```

`column` blocks are optional on a view; inspection writes them.

## Stages, Pipes, Tasks, and Other Schema Objects

```hcl
file_format "CSV_FORMAT" {
  schema = schema.BRONZE
  type   = CSV            # AVRO, JSON, ORC, PARQUET, XML
  options = {
    SKIP_HEADER = 1
  }
}

stage "LANDING" {
  schema              = schema.BRONZE
  url                 = "s3://bucket/landing/"
  storage_integration = "S3_INT"
  file_format         = file_format.CSV_FORMAT
  directory_enabled   = true
}

sequence "ORDER_SEQ" {
  schema    = schema.PUBLIC
  start     = 1
  increment = 1
  order     = true
}

pipe "LOAD_EVENTS" {
  schema      = schema.BRONZE
  as          = "COPY INTO \"BRONZE\".\"EVENTS\" FROM @\"BRONZE\".\"LANDING\""
  auto_ingest = true
}

task "ROLLUP" {
  schema    = schema.SILVER
  warehouse = "TRANSFORM_WH"
  schedule  = "60 MINUTE"
  as        = "INSERT INTO \"SILVER\".\"ROLLUP\" SELECT 1"
}

alert "ERROR_ALERT" {
  schema                           = schema.PUBLIC
  schedule                         = "1 MINUTE"
  warehouse                        = "TRANSFORM_WH"
  condition                        = "SELECT 1 FROM ERRORS"
  action                           = "INSERT INTO ALERT_LOG (MSG) VALUES ('failed')"
  suspend_alert_after_num_failures = 5
}

network_rule "OUTBOUND" {
  schema     = schema.PUBLIC
  type       = HOST_PORT # IPV4, ...
  mode       = EGRESS    # INGRESS, ...
  value_list = ["example.com:443"]
}
```

- A stage takes `credentials` (a map, with `url`) or `storage_integration`, never both.
- Options that hold Snowflake's defaults are left out of inspection and plans.
- A task takes `after = [task.ROOT]` or `finalize = task.ROOT` instead of `schedule`, `condition` for
  `WHEN`, and session parameters such as `user_task_timeout_ms`, `query_tag`, `timezone`, and
  `log_level`.
- Alerts need `alerts = true` and network rules `network_rules = true` in `mode "snowflake"`. An
  alert's `condition` and `action` are SQL, with Snowflake's case rules. A network rule's `type` and
  `mode` are fixed at creation: changing either recreates the rule.

## Functions and Procedures

`as` holds the body itself, never wrapped in `$$`. Overloads are separate blocks with the same label:

```hcl
function "ADD" {
  schema = schema.PUBLIC
  lang   = SQL
  arg "X" {
    type = NUMBER
  }
  arg "Y" {
    type    = NUMBER
    default = 10
  }
  return = NUMBER
  as     = "X + Y"
}

function "PY_DOUBLE" {
  schema          = schema.PUBLIC
  lang            = PYTHON
  runtime_version = "3.11"
  handler         = "double"
  arg "X" {
    type = NUMBER
  }
  return = NUMBER
  as     = <<-PY
    def double(x):
        return x * 2
  PY
}
```

- Table functions use a `return_table` block with `column` blocks instead of `return`.
- Functions take `secure`, `immutable`, `null_call` (`CALLED`, `RETURNS_NULL`), `execute_as`, `packages`,
  `imports` blocks (`stage`, `file`), `external_access_integrations`, and `secrets` blocks.
- Procedures take the same, plus `execute_as = RESTRICTED_CALLER`, `arg` `mode` (`IN`, `OUT`), and a
  `return` block with `type` and `null`.

## Tags and Policies

```hcl
tag "DATA_DOMAIN" {
  schema         = schema.PUBLIC
  allowed_values = ["customer", "finance"]
}

masking_policy "MASK_EMAIL" {
  schema = schema.PUBLIC
  arg "VAL" {
    type = VARCHAR
  }
  return = VARCHAR
  as     = "CASE WHEN CURRENT_ROLE() IN ('ANALYST') THEN VAL ELSE '***' END"
}

row_access_policy "BY_REGION" {
  schema = schema.PUBLIC
  arg "REGION" {
    type = VARCHAR
  }
  as = "REGION IN ('US', 'EU')"
}

table "CUSTOMERS" {
  schema = schema.PUBLIC
  column "EMAIL" {
    type = VARCHAR(255)
    policy "masking" {
      ref = masking_policy.MASK_EMAIL
    }
  }
  column "REGION" {
    type = VARCHAR(2)
  }
  policy "row_access" {
    ref     = row_access_policy.BY_REGION
    columns = ["REGION"]
  }
  tag {
    ref   = tag.DATA_DOMAIN
    value = "customer"
  }
}
```

- Tags and policies are planned only with `tags = true` and `policies = true` in `mode "snowflake"`.
- A `tag` assignment block goes on schemas, tables of every kind, columns, views, tasks, alerts,
  stages, pipes, functions, procedures, policies, warehouses, and external access integrations. `ref`
  is `tag.X`, `tag.SCHEMA.X` when the name repeats across schemas, or a string such as
  `"GOVERNANCE.PII"` for a tag outside the env's scope.
- A `policy` block's label is the kind. On a table: `row_access` (with `columns`), `aggregation` (with
  optional `entity_key`), and `join`. On a column: `masking` (with optional `using`) and `projection`.
  A `row_access` attachment without `columns` fails with
  `missing required attribute columns in block policy`.
- `projection_policy`, `aggregation_policy`, and `join_policy` blocks take `as` with the constraint,
  such as `as = "AGGREGATION_CONSTRAINT(MIN_GROUP_SIZE => 11)"`.

## Account-Level Blocks

These need database scope, their `mode "snowflake"` switch, and an `include` pattern per object:

```hcl
warehouse "TRANSFORM_WH" {
  size         = XSMALL
  auto_suspend = 60
  auto_resume  = true
  comment      = "dbt runs"
}

resource_monitor "MONTHLY_CAP" {
  credit_quota = 1000
  frequency    = MONTHLY
  trigger {
    threshold = 90
    action    = NOTIFY
  }
  trigger {
    threshold = 100
    action    = SUSPEND
  }
}
```

`external_volume` takes `storage_location "<PROVIDER>" "<name>"` blocks (`S3`, `S3GOV`, `GCS`, `AZURE`,
`S3COMPAT`). `external_access_integration` takes `allowed_network_rules` (at least one, or the plan
fails with `ALLOWED_NETWORK_RULES must name at least one network rule`), and either
`allowed_authentication_secrets` or `allow_all_authentication_secrets = true`.

## Roles, Users, and Grants

Covered in `references/security.md`.

## Seed Data

Snowflake has no `data` block. Seed rows with `INSERT` or `MERGE` statements in a migration file. A
file of DML statements only runs in one transaction and rolls back on failure
(`references/migrations.md`, Transactions and Partial Failures).

## Secrets and Input Variables

A secret's value comes in through an input variable, never as a literal in the file
(https://atlasgo.io/atlas-schema/input-variables):

```hcl
variable "api_key" {
  type = string
}

secret "API_KEY" {
  schema        = schema.PUBLIC
  type          = GENERIC_STRING
  secret_string = var.api_key
}
```

Secrets need `secrets = true` in `mode "snowflake"` and `atlas schema apply` (`references/objects.md`).
`atlas.hcl` passes the value through `data "hcl_schema"` with `vars`. A variable without a default
that gets no value fails with `missing value for required variable "api_key"`.
