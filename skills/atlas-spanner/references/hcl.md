# Writing Spanner Schemas in HCL

The rules of the HCL schema language that matter on Spanner, and the places where HCL and SQL schemas
plan differently. The language: https://atlasgo.io/atlas-schema/hcl. Every Spanner block and attribute:
https://atlasgo.io/hcl/spanner. The Spanner driver needs `atlas login`. If `atlas whoami` shows no login,
ask the user to run `atlas login`; a user without an account can start a free trial with it.

## HCL or SQL

Pick one format per project. An env cannot mix them: a `src` list with an `.sp.hcl` file and a `.sql`
file fails with `expect one schema in env`, and a `composite_schema` fails with
`driver does not support changing connected schema`.

- Use SQL when a primary key has a descending part, such as `PRIMARY KEY (Kind, CreatedAt DESC)`: HCL
  cannot plan it (Keys and Interleaving, below).
- Use HCL for vector indexes, locality groups, and placements when the dev database is the emulator. A
  SQL schema loses their options, and placements entirely, because the emulator does not report them.

Both formats lose the same objects otherwise: `references/objects.md`, What Atlas Leaves Out.

## Files and Schemas

- Name Spanner schema files `*.sp.hcl`, test files `*.test.hcl`, and rule files `*.rule.hcl`. The Atlas
  editor plugins recognize `.sp.hcl` for Spanner
  (https://atlasgo.io/atlas-schema/hcl#file-naming-for-editor-support).
- The default schema is `schema "default" {}` in GoogleSQL and `schema "public" {}` in the PostgreSQL
  dialect. Each named schema (`CREATE SCHEMA sales`) gets its own block, and its tables set
  `schema = schema.sales`.
- The URL covers the whole database, so the file must declare every named schema the database has, or
  the plan drops the missing ones (`references/dev-database.md`, Scope).

## Columns

- A column without `null` is `NOT NULL`, the opposite of SQL. Write `null = true` for a nullable column.
- Types are written inline: `INT64`, `FLOAT64`, `FLOAT32`, `NUMERIC`, `BOOL`, `DATE`, `TIMESTAMP`, `JSON`,
  `UUID`, `TOKENLIST`, `STRING(36)`, and `BYTES(16)`.
- `STRING(MAX)` and `BYTES(MAX)` are not HCL types: `type = STRING(MAX)` fails with
  `There is no type named "MAX"`. Write `sql("STRING(MAX)")`. Inspection writes `STRING(2621440)` and
  `BYTES(10485760)`, which plan as `MAX` too.
- Arrays, vectors, and proto types take `sql()`: `sql("ARRAY<STRING(64)>")`,
  `sql("ARRAY<FLOAT32>(vector_length=>768)")`, `sql("examples.shipping.Order")`. Proto types need their
  bundle on the target first (https://atlasgo.io/guides/spanner/proto-columns).
- `default` takes a literal (`default = 0`, `default = false`) or an expression in `sql()`:

  ```hcl
    column "CustomerId" {
      type    = STRING(36)
      default = sql("GENERATE_UUID()")
    }
    column "CreatedAt" {
      type    = TIMESTAMP
      default = sql("CURRENT_TIMESTAMP()")
    }
  ```

- A generated column takes an `as` block. `type = STORED` stores the values; without `type` the column
  is computed on read; `type = HIDDEN` hides a `TOKENLIST` column from `SELECT *`:

  ```hcl
    column "EmailLower" {
      type = STRING(320)
      null = true
      as {
        expr = "LOWER(Email)"
        type = STORED
      }
    }
    column "Body_Tokens" {
      type = TOKENLIST
      null = true
      as {
        expr = "TOKENIZE_FULLTEXT(Body)"
        type = HIDDEN
      }
    }
  ```

  Changing the expression of a stored column is planned as a drop and an add of the column.
- An `options` block sets column options:

  ```hcl
    column "UpdatedAt" {
      type = TIMESTAMP
      null = true
      options {
        allow_commit_timestamp = true
      }
    }
  ```

  It also takes `locality_group = locality.cold` (Locality Groups and Placements, below).
- There is no `identity` block and no `renamed_from`: identity columns and renames are covered in
  `references/objects.md`.

## Keys and Interleaving

- `primary_key` takes `columns`:

  ```hcl
    primary_key {
      columns = [column.SingerId, column.AlbumId]
    }
  ```

  For a descending key part, inspection writes `on` blocks, and those do not plan. A new table gets
  `PRIMARY KEY ()`, a table that holds at most one row, and an existing table fails with
  `the keys of a table cannot change after table creation`. Keep a schema with descending key parts in
  SQL (HCL or SQL, above).
- An interleaved table takes an `interleave` block. Always set `type = IN_PARENT`:

  ```hcl
  table "Albums" {
    schema = schema.default
    # ...
    primary_key {
      columns = [column.SingerId, column.AlbumId]
    }
    interleave {
      parent    = table.Singers
      type      = IN_PARENT
      on_delete = CASCADE
    }
  }
  ```

  - Without `type`, Atlas plans ``INTERLEAVE `Singers` ``, which Spanner rejects with
    `Syntax error ... Expecting 'IN'`.
  - `type = IN` plans `INTERLEAVE IN`, and a database that holds such a table can no longer be
    inspected (`references/objects.md`, Interleaving).
  - `on_delete` takes `CASCADE` or `NO_ACTION`, the default.
  - Inspection adds `depends_on = [table.Singers]`.

## Indexes, Foreign Keys, and Checks

- An `index` takes `columns`, or `on` blocks with `column` and `desc`, plus `unique`, `null_filtered`
  (`NULL_FILTERED`), and `include` (the `STORING` columns):

  ```hcl
    index "AlbumsByTitle" {
      columns = [column.Title]
      include = [column.Price]
    }
    index "SingersByEmail" {
      unique        = true
      null_filtered = true
      columns       = [column.Email]
    }
    index "EventsByTime" {
      on {
        column = column.Kind
      }
      on {
        column = column.CreatedAt
        desc   = true
      }
    }
  ```

  Adding a column to `include` plans `ALTER INDEX ... ADD STORED COLUMN`.
- `interleave_in` is accepted and ignored: the index is planned as a global index
  (`references/objects.md`, Indexes).
- A search index takes `type = SEARCH` over `TOKENLIST` columns. A vector index takes `type = VECTOR`
  and its options, which only HCL keeps:

  ```hcl
    index "DocsEmbeddingIdx" {
      columns       = [column.Embedding]
      type          = VECTOR
      distance_type = COSINE
      tree_depth    = 2
      num_leaves    = 1000
      where         = "Embedding IS NOT NULL"
    }
  ```

  Inspection does not read the options back from the emulator, so once the index exists there, every
  later plan fails with `modify index "DocsEmbeddingIdx" must have at least one column to add or drop`.
  Exclude the index once it exists (`references/objects.md`, Indexes).
- `foreign_key` takes `columns`, `ref_columns`, and `on_delete` (`CASCADE` or `NO_ACTION`). There is no
  attribute for `NOT ENFORCED`:

  ```hcl
    foreign_key "FK_PaymentsCustomer" {
      columns     = [column.CustomerId]
      ref_columns = [table.Customers.column.CustomerId]
      on_delete   = CASCADE
    }
  ```

- `check` takes `expr`. A changed expression is planned as a drop and an add of the constraint, which
  checks every row again.

## Tables

- `row_deletion_policy` sets a TTL, in GoogleSQL only: in the PostgreSQL dialect a TTL breaks planning
  (`references/objects.md`, PostgreSQL Dialect).

  ```hcl
  table "Events" {
    schema              = schema.default
    row_deletion_policy = sql("OLDER_THAN(CreatedAt, INTERVAL 30 DAY)")
    # ...
  }
  ```

  A changed interval plans `ALTER TABLE ... REPLACE ROW DELETION POLICY`.
- `locality_group = locality.hot` puts the table in a locality group.

## Sequences

```hcl
sequence "OrderSeq" {
  kind               = bit_reversed_positive
  skip_range_min     = 1
  skip_range_max     = 1000
  start_with_counter = 5000
}

table "Orders" {
  schema = schema.default
  column "OrderId" {
    type    = INT64
    default = sql("GET_NEXT_SEQUENCE_VALUE(SEQUENCE OrderSeq)")
  }
  # ...
}
```

- A sequence has no `schema` attribute and lives in the default schema. A sequence in a named schema,
  written in SQL as `CREATE SEQUENCE sales.OrderSeq`, breaks planning with
  `Drop sequence "OrderSeq": ... Sequence not found`.
- Option changes plan `ALTER SEQUENCE ... SET OPTIONS`.

## Views

```hcl
view "EmployeeNames" {
  schema     = schema.default
  as         = "SELECT e.Id, e.Name FROM Employees AS e"
  security   = INVOKER
  depends_on = [table.Employees]
}
```

- `security` is required by Spanner: `INVOKER`, or `DEFINER` for a view that roles read without access
  to its tables (`references/security.md`). A change to `as` plans `CREATE OR REPLACE VIEW`.

## Change Streams

```hcl
stream "OrdersStream" {
  tables = [table.Orders]
  options = {
    retention_period   = "7d"
    value_capture_type = "NEW_ROW"
  }
}

stream "EverythingStream" {
  for_all = true
}

stream "DocsTitleStream" {
  for {
    table   = table.Docs
    columns = [table.Docs.column.Title, table.Docs.column.Price]
  }
}
```

A change to `options` is planned as `DROP CHANGE STREAM` and `CREATE CHANGE STREAM`, which deletes the
stream's change records: write `ALTER CHANGE STREAM` by hand (`references/migrations.md`).

## Locality Groups and Placements

```hcl
locality "cold" {
  storage = "hdd"
}

locality "recent" {
  storage                   = "ssd"
  ssd_to_hdd_spill_timespan = "30d"
}

placement "europe" {
  instance_partition = "euro-partition"
  default_leader     = "europe-west1"
}

table "Orders" {
  schema         = schema.default
  locality_group = locality.cold
  column "Location" {
    type          = sql("STRING(MAX)")
    placement_key = true
  }
  # ...
}
```

Locality groups need the Enterprise edition, and placements the Enterprise Plus edition. The emulator
does not report them back, so a SQL schema loses them on an emulator dev database, and plans against an
emulator target never converge; from HCL, Atlas plans them on the emulator too
(`references/dev-database.md`, What the Emulator Does Not Do).

## Property Graphs

Inspection writes graphs with explicit keys:

```hcl
property_graph "Social" {
  schema = schema.default
  node_table {
    table = table.Person
    key {
      columns = [table.Person.column.Id]
    }
  }
  edge_table {
    table = table.Knows
    key {
      columns = [table.Knows.column.Id]
    }
    source_key {
      columns     = [table.Knows.column.FromId]
      ref_columns = [table.Person.column.Id]
    }
    destination_key {
      columns     = [table.Knows.column.ToId]
      ref_columns = [table.Person.column.Id]
    }
  }
}
```

A change plans `CREATE OR REPLACE PROPERTY GRAPH`, before it drops a table the graph no longer names.
Spanner Graph is GoogleSQL only.

## Roles and Permissions

Covered in `references/security.md`.

## Seed Data

The `data` block is not supported on Spanner: a schema file with one fails with
`unknown block type data in spanner schema`. Insert seed rows with `INSERT` statements in a migration
file (`references/migrations.md`).

## Input Variables

Input variables work as in any Atlas HCL schema (https://atlasgo.io/atlas-schema/input-variables). A
variable without a default that gets no value fails the run.
