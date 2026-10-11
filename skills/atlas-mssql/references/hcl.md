# Writing SQL Server Schemas in HCL

The rules of the HCL schema language that matter on SQL Server, and the places where HCL behaves
differently from T-SQL. The language: https://atlasgo.io/atlas-schema/hcl. Column types:
https://atlasgo.io/atlas-schema/hcl-types. Every SQL Server block and attribute:
https://atlasgo.io/hcl/mssql.

A SQL schema file (`file://schema.sql`) avoids the HCL-specific traps below, and is often the easier
choice for a team that already writes T-SQL. The rules on naming constraints still apply to it
(`references/objects.md`).

## Files

- Name SQL Server schema files `*.ms.hcl` and test files `*.test.hcl`, so editors apply the right
  schema (https://atlasgo.io/atlas-schema/hcl#file-naming-for-editor-support).
- A schema file targets one database engine: HCL does not abstract the differences between engines.
- Several `schema` blocks need a database-scoped URL and dev URL (`references/dev-database.md`).
  Two tables with the same name in different schemas take a second label, the schema:
  `table "sales" "orders" { schema = schema.sales }`, referenced as `table.sales.orders`.

## Columns

- A column without `null` is `NOT NULL`, the opposite of T-SQL. Write `null = true` for a nullable
  column.
- `default` takes a literal or an expression in `sql()`: `default = 0`,
  `default = sql("sysutcdatetime()")`, `default = sql("newsequentialid()")`. Atlas names the constraint
  `DF_<table>_<column>`. To choose the name, use a block:

  ```hcl
    column "created_at" {
      type = datetime2(7)
      default "df_users_created_at" {
        as = sql("sysutcdatetime()")
      }
    }
  ```

- A string `default` is written as a non-Unicode literal, `DEFAULT 'Tel Aviv'`. For text outside the
  database's code page, write the literal yourself: `default = sql("N'שלום'")`.
- An `identity` block with `seed` and `increment` makes an identity column.
- A computed column takes `as`, or an `as` block for `PERSISTED`, and still needs a `type`:

  ```hcl
    column "total_with_tax" {
      type = decimal(12,2)
      as {
        expr = "[total]*(1.17)"
        type = PERSISTED
      }
    }
  ```

- `collate = "Latin1_General_CS_AS"` sets a column's collation; inspection prints it only when it differs
  from the database's default.
- `renamed_from` plans `sp_rename` instead of dropping the old column and adding a new one
  (https://atlasgo.io/guides/destructive-change-policy):

  ```hcl
    column "email" {
      renamed_from = "legacy_email"
      type         = nvarchar(200)
    }
  ```

## Column Types

Write parameters inline: `decimal(10,2)`, `nvarchar(255)`, `datetime2(7)`
(https://atlasgo.io/atlas-schema/hcl-types):

| Types | HCL |
|-------|-----|
| Integers | `tinyint`, `smallint`, `int`, `bigint`, `bit` |
| Exact and floating point | `decimal(p,s)`, `numeric(p,s)`, `money`, `smallmoney`, `float(53)`, `real` |
| Strings | `char(n)`, `varchar(n)`, `nchar(n)`, `nvarchar(n)`, `varchar(-1)`, `nvarchar(-1)` |
| Binary | `binary(n)`, `varbinary(n)`, `varbinary(-1)`, `rowversion` |
| Date and time | `date`, `time(7)`, `datetime2(7)`, `datetimeoffset(7)`, `datetime`, `smalldatetime` |
| Other | `uniqueidentifier`, `xml`, `sql_variant`, `hierarchyid`, `geography`, `geometry`, `sysname` |
| Your own types | `type_alias.email` for columns; `type_table.id_list` for arguments |
| Anything else | `sql("...")` |

- `MAX` is written `-1`: `nvarchar(-1)` is `nvarchar(MAX)`, and inspection prints it that way.
  `sql("nvarchar(max)")` works too. `nvarchar(MAX)` fails with
  `Unknown column.type; There is no type named "MAX"`.
- Write the precision of `datetime2`, `time`, and `datetimeoffset`. In HCL, a bare `datetime2` is
  `datetime2(0)`, whole seconds, not T-SQL's default `datetime2(7)`. The same holds for `time` and
  `datetimeoffset`. Inspection prints `datetime2(7)`.
- Like T-SQL, a bare `varchar` or `nvarchar` is length 1, and `decimal` is `decimal(18)`. Always give
  the length.
- Inspection prints canonical names: `float` reads back as `float(53)`, `real` as `real(24)`, and
  `timestamp` as `rowversion`.

## Keys and Indexes

- An `index` without `nonclustered = true` is clustered, unlike `CREATE INDEX`. Next to a clustered
  primary key it fails with `Cannot create more than one clustered index on table 't'`. Write
  `nonclustered = true` on every secondary index:

  ```hcl
    index "ix_orders_customer" {
      columns      = [column.customer_id]
      include      = [column.total]
      where        = "([status]<>'archived')"
      nonclustered = true
    }
  ```

- A `primary_key` is clustered unless it sets `nonclustered = true`. An unlabeled one is named
  `PK_<table>`.
- A `unique` block is a unique constraint, and it is clustered by default too: next to a clustered
  primary key it fails with `Cannot add more than one clustered index for constraints on table 't'`.
  An `index` with `unique = true` is a unique index.

  ```hcl
    unique "uq_users_email" {
      columns      = [column.email]
      nonclustered = true
    }
  ```
- `check "ck_orders_total" { expr = "([total]>=(0))" }` takes its expression as a string. Always give
  checks a label: an unlabeled check is dropped and re-added on every plan.
- A foreign key's `on_delete` and `on_update` take `NO_ACTION`, `CASCADE`, `SET_NULL`, or
  `SET_DEFAULT`; `RESTRICT` fails on SQL Server.
- `type = COLUMNSTORE` makes a columnstore index (`references/objects.md`).

## Functions, Procedures, Triggers, and Views

`as` holds the body after the `AS` keyword. Inside a table, `column.<name>` refers to its own columns;
elsewhere use `table.<table>.column.<column>`:

```hcl
function "order_total" {
  schema = schema.dbo
  lang   = SQL
  arg "@id" {
    type = bigint
  }
  return = decimal(10,2)
  as     = <<-SQL
  BEGIN
    RETURN (SELECT total FROM dbo.orders WHERE id = @id);
  END
  SQL
}

function "big_orders" {
  schema = schema.dbo
  lang   = SQL
  arg "@min" {
    type = decimal(10,2)
  }
  return = sql("table")
  as     = "RETURN (SELECT id, total FROM dbo.orders WHERE total >= @min)"
}

procedure "touch_order" {
  schema = schema.dbo
  lang   = SQL
  arg "@id" {
    type = bigint
  }
  arg "@n" {
    type = int
    mode = OUT
  }
  as = <<-SQL
  BEGIN
    SET NOCOUNT ON;
    UPDATE dbo.orders SET updated_at = SYSUTCDATETIME() WHERE id = @id;
    SET @n = @@ROWCOUNT;
  END
  SQL
}

trigger "tr_orders_upd" {
  on = table.orders
  after {
    update = true
  }
  as = <<-SQL
  BEGIN
    SET NOCOUNT ON;
  END
  SQL
}

view "v_orders" {
  schema       = schema.dbo
  schema_bound = true
  column "id" {
    type = bigint
  }
  as         = "SELECT id, total FROM dbo.orders"
  depends_on = [table.orders]
}
```

- `<<-SQL` strips the common indentation of a heredoc.
- `lang` is `SQL`. Argument names keep their `@`. `mode = OUT` is an output parameter, and
  `readonly = true` a table-valued parameter.
- `schema_bound = true` is `WITH SCHEMABINDING`; `inline = true` on a scalar function needs SQL Server
  2019 or later.
- A view lists its columns, and `depends_on` orders it after the objects it reads.
- Leading `//` comments on views, functions, and procedures are kept only with
  `schema { mode "sqlserver" { header_comment = true } }` in the env.

## Sequences and Types

```hcl
sequence "seq_orders" {
  schema = schema.dbo
  type   = bigint
  start  = 1000
}

type_alias "email" {
  schema = schema.dbo
  type   = nvarchar(320)
  null   = true            # NOT NULL by default
}

type_table "id_list" {
  schema = schema.dbo
  column "id" {
    type = int
  }
  primary_key {
    columns = [column.id]
  }
}
```

## Roles, Users, and Grants

The rules are in `references/security.md`. A `role`, `user`, and `permission` block needs database
scope, and `roles = true` and `permissions = true` in the env's `schema { mode { ... } }` block.

## Seed Data

Rows are declared next to the table in a `data` block, and applied only when the env enables data sync
(https://atlasgo.io/guides/mssql/seed-data):

```hcl
data {
  table = table.currencies
  rows = [
    { code = "USD", name = "US Dollar" },
    { code = "EUR", name = "Euro" },
  ]
}
```

```hcl
env "local" {
  # ...
  data {
    mode = INSERT   # or UPSERT, or SYNC with max_rows
  }
}
```

- The table needs a primary key, and every row gives every primary-key column and every `NOT NULL`
  column without a default.
- A table whose key is an identity column cannot be seeded from HCL
  (`cannot set value for generated column "id"`); see `references/objects.md`.

## Input Variables

A schema file declares the values it expects, and `atlas.hcl` passes them through `data "hcl_schema"`
(https://atlasgo.io/atlas-schema/input-variables):

```hcl
variable "tenant" {
  type = string
}

schema "tenant" {
  name = var.tenant
}
```

```hcl
data "hcl_schema" "app" {
  path = "schema.ms.hcl"
  vars = {
    tenant = "acme"
  }
}

env "local" {
  src = data.hcl_schema.app.url
  url = getenv("DATABASE_URL")
}
```
