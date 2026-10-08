# Writing PostgreSQL Schemas in HCL

The rules of the HCL schema language that matter on PostgreSQL, and the places where HCL behaves
differently from SQL. The language: https://atlasgo.io/atlas-schema/hcl. Column types:
https://atlasgo.io/atlas-schema/hcl-types. Every PostgreSQL block and attribute:
https://atlasgo.io/hcl/postgres.

Several blocks below need `atlas login`. If `atlas whoami` shows no login, ask the user to run
`atlas login`; a user without an account can start a free trial with it.

## Files

- Name PostgreSQL schema files `*.pg.hcl` and test files `*.test.hcl`, so editors apply the right
  schema (https://atlasgo.io/atlas-schema/hcl#file-naming-for-editor-support).
- A schema file targets one database engine: HCL does not abstract the differences between engines.
- One file can hold several `schema` blocks.

## Columns

- A column without `null` is `NOT NULL`, the opposite of SQL. Write `null = true` for a nullable column.
- `default` takes a literal (a bool, a number, or a string, which becomes a quoted SQL literal) or an
  expression in `sql()`: `default = sql("now()")`, `default = sql("gen_random_uuid()")`,
  `default = sql("CURRENT_DATE")`. Qualify functions that come from an extension:
  `default = sql("public.uuid_generate_v4()")`.
- `renamed_from` plans a rename instead of dropping the old column and adding a new one
  (https://atlasgo.io/guides/destructive-change-policy):

  ```hcl
    column "deprecated_legacy_email" {
      renamed_from = "legacy_email"
      type         = varchar(200)
    }
  ```

  A SQL schema does the same with a directive on the line before the column:
  `-- atlas:renamed_from legacy_email`.
- Generated columns take an expression in `as`, or an `as` block with the storage type
  (https://atlasgo.io/atlas-schema/hcl#generated-columns):

  ```hcl
    column "b" {
      type = int
      # In PostgreSQL, generated columns are STORED by default.
      as = "a * 2"
    }
    column "c" {
      type = int
      as {
        expr = "a * 3"
        type = STORED
      }
    }
  ```

  In HCL, a generated column without `type` is `STORED` on every PostgreSQL version. PostgreSQL 18's
  `VIRTUAL` default applies only to SQL sources; in HCL, write `type = VIRTUAL`. The expression cannot
  reference another generated column. Generated columns need a dev database to normalize the
  expression. Changing the expression: Column Changes in `references/objects.md`.
- `collate = "es_ES"` sets a column's collation.

## Column Types

Write parameters inline: `numeric(5,2)`, `varchar(255)`, `timestamp(4)`, `bit(2)`. Multi-word types use
underscores (https://atlasgo.io/atlas-schema/hcl-types):

| Types | HCL |
|-------|-----|
| Integers | `smallint`, `integer` (or `int`), `bigint`, `serial`, `bigserial` |
| Exact and floating point | `numeric`, `numeric(5,2)`, `decimal`, `real`, `double_precision`, `float(p)` |
| Strings | `text`, `varchar`, `varchar(255)`, `char`, `char(5)` |
| Date and time | `date`, `time`, `timetz`, `timestamp`, `timestamptz`, `timestamp(4)`, `interval` |
| Bits and binary | `bit`, `bit(2)`, `bit_varying`, `bit_varying(1)`, `bytea`, `boolean` |
| Documents and identifiers | `json`, `jsonb`, `uuid`, `xml`, `money` |
| Network | `inet`, `cidr`, `macaddr`, `macaddr8` |
| Geometric | `point`, `line`, `lseg`, `box`, `path`, `polygon`, `circle` |
| Ranges | `int4range`, `int8range`, `numrange`, `tsrange`, `tstzrange`, `daterange`, and their multiranges |
| Text search | `tsvector`, `tsquery` |
| Your own types | `enum.status`, `domain.username`, `composite.address`, `range.price_range` |
| Anything else | `sql("...")` |

- Arrays have no HCL type; write them with `sql()`:

  ```hcl
    column "c1" {
      type = sql("int[]")
    }
    column "c4" {
      type = sql("varchar(255)[]")
    }
  ```

- Types from extensions are written with `sql()` too, and the extension is declared in the schema:
  `type = sql("vector(1536)")` with `extension "vector"`. PostGIS columns inspect as
  `type = sql("public.geometry(Point,4326)")`.
- Inspection prints the canonical name: `int` reads back as `integer`, and `varchar(255)` as
  `character_varying(255)`. Both spellings name the same type.

## References and Names

- Inside a table, `column.<name>` refers to its own columns. Any other column is
  `table.<table>.column.<column>`, even in another schema, and top-level blocks such as triggers always
  use the full path.
- Two tables with the same name in different schemas take a second label, the schema
  (https://atlasgo.io/atlas-schema/hcl#table-qualification):

  ```hcl
  schema "a" {}
  schema "b" {}

  table "a" "users" {
    schema = schema.a
    // .. columns
  }
  table "b" "users" {
    schema = schema.b
    // .. columns
  }
  ```

  References then include the qualifier: `ref_columns = [table.public.users.column.id]`.
- SQL bodies can interpolate names, so a rename follows them: `FROM ${table.users.name} AS u`.
- Constraint names are block labels and may contain spaces: `check "positive price"`. A `check` takes
  its SQL in `expr`, as a string.
- HCL strings escape backslashes, so a regular expression doubles them:
  `expr = "((VALUE ~ '^\\d{5}$'::text) OR (VALUE ~ '^\\d{5}-\\d{4}$'::text))"`.

## Keys and Constraints

- A foreign key's `on_delete` and `on_update` take `NO_ACTION`, `RESTRICT`, `CASCADE`, `SET_NULL`, or
  `SET_DEFAULT`, and `deferrable` takes `INITIALLY_IMMEDIATE` or `INITIALLY_DEFERRED`
  (https://atlasgo.io/hcl/postgres). A self-reference uses the table's own column:

  ```hcl
    foreign_key "manager_fk" {
      columns = [column.manager_id]
      ref_columns = [column.id]
      on_delete = CASCADE
      on_update = NO_ACTION
    }
  ```

- PostgreSQL 18 temporal keys need `atlas login`. `without_overlaps` on a primary key or unique
  constraint applies `WITHOUT OVERLAPS` to the key's last column, backed by a GiST index. A foreign key
  that references one sets `period`:

  ```hcl
    foreign_key "child_parent_fkey" {
      columns     = [column.parent_id, column.validity]
      ref_columns = [table.parent.column.id, table.parent.column.validity]
      period      = true
    }
  ```

- A `unique` block takes `include`. An `exclude` block gives each `on` part its operator as a string,
  such as `op = "&&"`. Set `type = GIST` on the block for `&&`: the default index type is `BTREE`.
- `check`, `foreign_key`, `unique`, and `exclude` blocks take a `comment`.

## Indexes

- An `index` takes either `columns` or `on` blocks, never both.
- Index settings go in `storage_params`, for example `fillfactor = 50` and `buffering = ON` on a GiST
  index, or `deduplicate_items = false` on a B-tree.
- pgvector indexes (need `atlas login` and `extension "vector"` in the schema) name the type as a string
  (https://atlasgo.io/atlas-schema/hcl#index-storage-parameters):

  ```hcl
  index "index_name" {
    type = "HNSW"
    on {
      column = column.embedding
      ops    = "vector_l2_ops"
    }
    storage_params {
      m = 16
      ef_construction = 64
    }
  }
  ```

  `type = "IVFFlat"` takes `lists = 100` in `storage_params` instead.

## Table Settings

- A table's `storage_params` block sets options such as `fillfactor` and the autovacuum settings
  (https://atlasgo.io/hcl/postgres#table.storage_params). Lint flags turning autovacuum off (`PG320`).
- `replica_identity = FULL`, or `replica_identity = index.idx_account_id` for `USING INDEX`, sets the
  table's replica identity (https://atlasgo.io/changelog/postgres-replica-identity). Lint flags `FULL`
  and `NOTHING` (`PG314`).
- `access_method` sets the table access method (https://atlasgo.io/hcl/postgres#table.access_method).
  A new access method rewrites the table (`PG311`).
- `collation` and `cast` blocks define collations and casts (https://atlasgo.io/hcl/postgres#collation,
  https://atlasgo.io/hcl/postgres#cast).

## Partitions

A partition key can be an expression: one `by` block per part, each with `column` or `expr`
(https://atlasgo.io/atlas-schema/hcl#partitions):

```hcl
  partition {
    type = RANGE
    by {
      column = column.x
    }
    by {
      expr = "floor(y)"
    }
  }
```

List partitions take `in = ["'de'", "'fr'", "'it'"]`, and hash partitions `modulus` and `remainder`. A
`partition` block built with `for_each` has no label; it sets `name = each.value.name` instead.

## Functions, Procedures, and Aggregates

- `as` holds the body itself, never wrapped in `$$`. `lang` is a bare identifier: `SQL` or `PLpgSQL`.
- A SQL-standard body goes in a heredoc. `<<-SQL` strips the common indentation; with `<<SQL`, the body
  and the closing `SQL` start at column 0:

  ```hcl
    as = <<-SQL
     BEGIN ATOMIC
      SELECT v;
     END
    SQL
  ```

- Arguments may be unnamed and read as `$1`. Function attributes such as volatility, `strict`,
  `leakproof`, and `security` are set directly (https://atlasgo.io/atlas-schema/hcl#function):

  ```hcl
  function "sql_body2" {
    schema = schema.public
    lang   = SQL
    arg {
      type = integer
    }
    return     = integer
    as         = "RETURN $1"
    volatility = IMMUTABLE // STABLE | VOLATILE
    leakproof  = true      // NOT LEAKPROOF | LEAKPROOF
    strict     = true      // (CALLED | RETURNS NULL) ON NULL INPUT
    security   = INVOKER   // DEFINER | INVOKER
  }
  ```

- A procedure's `arg` can take a `default`.
- Aggregates need `atlas login`, like functions:

  ```hcl
  aggregate "sum_of_squares" {
    schema = schema.public
    arg {
      type = double_precision
    }
    state_type = double_precision
    state_func = function.sum_squares_sfunc
  }
  ```

## Event Triggers, Policies, and Repeated Blocks

- An `event_trigger` has no `schema`, and its `execute` is an attribute, not a block as in `trigger`
  (https://atlasgo.io/atlas-schema/hcl#event-trigger):

  ```hcl
  event_trigger "record_table_creation" {
    on      = ddl_command_start
    tags    = ["CREATE TABLE"]
    execute = function.record_table_creation
  }
  ```

- A `policy` takes `to` as a list, `[PUBLIC]` or role names as strings, and `using` and `check` as SQL
  strings (https://atlasgo.io/atlas-schema/hcl#row-level-security-policy):

  ```hcl
  policy "restrict_sales_rep_updates" {
    on      = table.orders
    as      = RESTRICTIVE
    for     = UPDATE
    to      = ["custom_role"]
    check   = "(sales_rep_id = (CURRENT_USER)::integer)"
    comment = "This is a restrictive policy"
  }
  ```

- `for_each` with `each.value` repeats a trigger, policy, or permission over a list of tables or roles,
  for example `for_each = [table.users, table.orders, table.payments]` with `on = each.value`.

## Roles, Users, and Grants

The rules are in `references/security.md`; the HCL details:

- `role "reader" {}` is a complete role. A role takes `login` (false by default), `inherit` (true by
  default), `superuser`, `create_db`, `create_role`, `replication`, `bypass_rls`, `conn_limit` (`-1` for
  no limit), and `member_of`. A `user` is a role with `login = true`. A `membership` block sets the
  options of one membership (`references/security.md`).
- A password comes in through an input variable, never as a literal in the file
  (https://atlasgo.io/atlas-schema/hcl#sensitive-information):

  ```hcl
  variable "db_password" {
    type = string
  }

  user "app_user" {
    password = var.db_password
  }
  ```

- A grant can target one column, of a table or a view:

  ```hcl
  permission {
    to         = "developer"
    for        = table.users.column.id
    privileges = [UPDATE]
  }
  ```

- `privileges = [ALL]` grants everything the object supports. A schema supports only `USAGE` and
  `CREATE`.

## Seed Data

Rows are declared next to the table in a `data` block, and applied only when the env enables data sync
(needs `atlas login`, https://atlasgo.io/atlas-schema/hcl#data):

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

- The table needs a primary key, and every row gives every primary-key column and every `NOT NULL`
  column without a default. Generated columns are always left out.
- A `data` block cannot seed a table whose primary key is an identity column; seed it from SQL
  (Sequences in `references/objects.md`).
- The env's `data` block sets the `mode` and the tables to sync. `include` follows the URL's scope:

  ```hcl
  env "database-scope" {
    ...
    data {
      mode     = SYNC
      max_rows = 100
      include  = ["public.countries", "tenant*.settings"]
    }
  }
  ```

- Columns whose default changes on every write, such as `now()`, would show a diff on every run. Leave
  them out of the comparison with `skip_diff = ["*.updated_at", "*.created_at"]` in the env's `data`
  block. Those patterns work at schema scope; at database scope, add the schema:
  `skip_diff = ["*.*.updated_at", "*.*.created_at"]`. A pattern with the wrong number of parts is
  ignored without an error.

## Input Variables

A schema file declares the values it expects (https://atlasgo.io/atlas-schema/input-variables):

```hcl
variable "comment" {
  type    = string // | int | bool | list(string) | etc.
  default = "default value"
}
```

`atlas.hcl` passes them through `data "hcl_schema"`
(https://atlasgo.io/atlas-schema/projects):

```hcl
data "hcl_schema" "app" {
  path = "schema.hcl"
  vars = {
    // Variables are passed as input values to "schema.hcl".
    tenant = "ariga"
  }
}

env "local" {
  src = data.hcl_schema.app.url
  url = getenv("DATABASE_URL")
}
```

- A variable without a default that gets no value fails with
  `missing value for required variable "tenant"`.
- `validation` blocks check a value with `condition` and `error_message`. A variable can have several;
  they also check defaults:

  ```hcl
  variable "env" {
    type = string
    validation {
      condition     = contains(["dev", "staging", "prod"], var.env)
      error_message = "Environment must be dev, staging, or prod."
    }
  }
  ```

- A schema's name can come from a variable, while references keep using the label, `schema.tenant`:

  ```hcl
  schema "tenant" {
    // Reference to the input variable.
    name = var.tenant
  }
  ```
