# SQL Server Objects and Their Nuances

How Atlas models SQL Server objects, and the cases where its plan differs from what a hand-written
migration would do. The full attribute reference is https://atlasgo.io/hcl/mssql.

## Login

The whole SQL Server driver needs `atlas login`, for every object kind. Logged out, Atlas does not
skip objects: every command fails with `The sqlserver driver is available only to Atlas Pro users`
or `login is required to use the 'sqlserver' driver`. If `atlas whoami` shows no login, ask the user
to run `atlas login`; a user without an account can start a free trial with it.

## Name Every Constraint

SQL Server names an unnamed constraint itself, with a random suffix, such as
`PK__users__3213E83F6692D103` or `CK__parents__qty__534D60F1`. Atlas loads the desired state on the
dev database, so an unnamed constraint gets a new name on every run, and the plan never settles:

- An unnamed `PRIMARY KEY` in a SQL schema is planned as an
  `EXECUTE sp_rename @objtype = N'INDEX'` in every plan.
- An unnamed `CHECK` or `DEFAULT` in a SQL schema, and an unlabeled `check` block in HCL, are dropped
  and added again in every plan, and lint reports the drop (`CD102`).

Name them in SQL with `CONSTRAINT <name>`:

```sql
CREATE TABLE users (
  id int IDENTITY(1,1) NOT NULL CONSTRAINT pk_users PRIMARY KEY,
  email nvarchar(255) NOT NULL CONSTRAINT uq_users_email UNIQUE,
  created_at datetime2(7) NOT NULL CONSTRAINT df_users_created_at DEFAULT SYSUTCDATETIME(),
  qty int NOT NULL CONSTRAINT ck_users_qty CHECK (qty >= 0)
);
```

- In HCL, an unlabeled `primary_key` is named `PK_<table>`, and a `default` attribute is named
  `DF_<table>_<column>`; both are stable. Give `check` blocks a label: `check "ck_users_qty" { ... }`.
- Unnamed `UNIQUE` constraints and foreign keys are matched by their columns and stay stable, but
  naming them keeps plans and errors readable.
- An existing database already holds generated names. Keep the inspected names in the desired state,
  or expect a one-time rename (primary keys) or drop and add (checks, defaults) to the chosen names.

## Identity, Sequences, and Defaults

- An `identity` block sets `seed` and `increment` on a column; an empty block is `IDENTITY (1, 1)`.
- Changing an identity's seed or increment, or making an existing column an identity, drops the column
  and adds it again, which loses its values; lint reports `DS103`. To restart numbering, run
  `DBCC CHECKIDENT ('dbo.orders', RESEED, 1000)` in a hand-written migration and leave the schema's
  `seed` unchanged: inspection reads the original seed, so no drift follows.
- An HCL `data` block cannot set an identity column: a row that gives one fails with
  `cannot set value for generated column "id"`, and `preserve_ids` has no effect on SQL Server. Give a
  lookup table a plain key, as the seed-data guide does (https://atlasgo.io/guides/mssql/seed-data), or
  seed an identity table with `SET IDENTITY_INSERT` in a hand-written migration.
- A `sequence` takes `type` (`bigint` by default), `start`, `increment`, `min_value`, `max_value`,
  `cache` (`0` for `NO CACHE`), and `cycle`. A column uses it through its default:
  `default = sql("NEXT VALUE FOR [seq_orders]")`. Changing a sequence's `type` drops and recreates it.
- Defaults are constraints. Atlas drops and re-adds a column's default around an `ALTER COLUMN`, so
  type and nullability changes need no hand edit.

## Column Changes

- Atlas changes type, length, nullability, and collation in place with `ALTER COLUMN`. A narrower
  type passes lint and fails when applied if a row does not fit:
  `mssql: String or binary data would be truncated in table 'app.dbo.a', column 'code'`. Check the
  data, or add a `test "migrate"` case with seeded rows (`references/testing.md`).
- Changing a computed column's expression or `PERSISTED` drops the column and adds it again.
  Converting a regular column to a computed one, or back, fails with
  `changing column "c" to computed column is not supported (drop and add is required)`: write it by
  hand.
- `NULL` to `NOT NULL` on a column that may hold `NULL` is reported as `MF104`; the `not_null` diff
  policy adds a pre-migration check (`references/migrations.md`, Lock-Safe Changes).
- Atlas drops and re-adds the primary key when one of its columns changes type.

## User-Defined Types

- A `type_alias` block is `CREATE TYPE ... FROM`. In HCL, an alias type is `NOT NULL` unless it sets
  `null = true` (`references/hcl.md`, Sequences and Types).
- A `type_table` is a table type for table-valued parameters (`readonly = true` on the `arg`).
- Changing an alias type drops and recreates it, with the functions, procedures, and table types that
  use it. A table column that uses it blocks the drop:
  `Cannot drop type 'email' because it is being referenced by object 'u'`. Write that change by hand:
  alter the columns to the base type, drop and recreate the type, and alter the columns back.

## Functions, Procedures, Triggers, and Views

- `as` holds the body after the `AS` keyword. A scalar function or multi-statement table-valued
  function includes its `BEGIN ... END`; an inline table-valued function is
  `as = "RETURN (SELECT ...)"` with `return = sql("table")`.
- Argument names keep their `@`: `arg "@id" { type = bigint }`. An output parameter is `mode = OUT`;
  `INOUT` is not accepted.
- Changing a function's arguments or return type drops and recreates it. Changing its body, or a
  procedure's arguments or body, is an `ALTER`, which keeps its grants.
- Views and triggers are changed with `CREATE OR ALTER` at compatibility level 130 (SQL Server 2016) and
  later, and dropped and recreated below it.
- `schema_bound = true` adds `WITH SCHEMABINDING`. An indexed view is a `schema_bound` view with an
  `index` block: Atlas creates the first one `UNIQUE CLUSTERED`.
- `execute_as` sets `EXECUTE AS` on functions, procedures, and triggers: `sql("OWNER")`, `sql("SELF")`,
  or a user name.
- A trigger takes `on`, one of `after` or `instead_of` with `insert`, `update`, and `delete`, and `as`.
  Only DML triggers are managed; `FOR` inspects as `AFTER`.
- Modules created `WITH ENCRYPTION` are skipped by inspection: Atlas cannot read their definitions.

## Indexes and Keys

- In HCL, an `index` without `nonclustered = true` is clustered, unlike `CREATE INDEX` in SQL. A
  table has one clustered index, usually its primary key, so a second one fails with
  `Cannot create more than one clustered index on table 't'`. Write `nonclustered = true` on every
  secondary index. Inspection prints it.
- A `primary_key` and a `unique` block are clustered by default too; `nonclustered = true` changes that.
  A clustered `unique` next to a clustered primary key fails with
  `Cannot add more than one clustered index for constraints on table 't'`.
- An index takes `columns`, or `on` blocks with `column` and `desc`, plus `unique`, `where` (a filtered
  index), `include`, and `fill_factor`. Index parts cannot be expressions: index a computed column.
- `type = COLUMNSTORE` is a clustered columnstore index, or nonclustered with `nonclustered = true` and
  `columns`. `type = SPATIAL` and `type = XML` (a primary XML index) are also supported.
- A change to an index other than `fill_factor` drops and recreates it. Atlas has no online index
  option: for a large table, write the change by hand with `WITH (ONLINE = ON)` where the edition
  supports it.
- A foreign key's `on_delete` and `on_update` take `NO_ACTION`, `CASCADE`, `SET_NULL`, or `SET_DEFAULT`.
  `RESTRICT` is not valid on SQL Server and fails with `Incorrect syntax near the keyword 'RESTRICT'`.
  Changing a foreign key drops and adds it.

## Temporal Tables

A system-versioned table declares its period columns, the period, and the history table
(https://atlasgo.io/atlas-schema/hcl#system-versioned-tables). The period columns are `datetime2`,
`NOT NULL`, and generated always as the row start and row end:

```hcl
table "t1" {
  schema = schema.dbo
  column "c1" {
    type = int
    null = false
  }
  column "c3" {
    type = datetime2(7)
    null = false
    generated_always {
      as = ROW_START
    }
  }
  column "c4" {
    type = datetime2(7)
    null = false
    generated_always {
      as = ROW_END
    }
  }
  primary_key {
    columns = [column.c1]
  }
  period "system_time" {
    type  = SYSTEM_TIME
    start = column.c3
    end   = column.c4
  }
  system_versioned {
    history_table  = "dbo.t1_History"
    retention      = 3
    retention_unit = MONTH
  }
}
```

- History tables are not inspected as tables of their own.
- Dropping a system-versioned table asks whether to drop its history table too. In a non-interactive
  run, such as an agent or CI, the plan turns versioning off and drops both.

## Memory-Optimized, Graph, Partitioned, and Full-Text Tables

- `memory_optimized = true` creates the table `WITH (MEMORY_OPTIMIZED = ON)`. Atlas cannot change a
  table between disk-based and memory-optimized; that needs a hand-written migration.
- `graph = NODE` or `EDGE` creates graph tables, with `connection` blocks for edge constraints.
- `partition_function` and `partition_scheme` are database-level objects (`mode=database`), and a
  table uses them with a `partition` block that sets `scheme` and `column`.
  Changing a partition function's boundary values is planned as `SPLIT` and `MERGE RANGE`. Changing
  its input type or `range` side drops and recreates the function and its schemes, and drops and
  recreates every table on them, losing their rows: review that plan, and write the change by hand.
- `fulltext_catalog` and a table's `fulltext` block manage full-text search. They need a dev image with
  full-text search installed, and SQL Server cannot create them inside a transaction
  (https://atlasgo.io/atlas-schema/hcl#full-text-index).

## Not Managed by Atlas

Atlas does not inspect or plan the following. Manage them in hand-written migration files, and expect
no drift report for them:

- Synonyms, security policies (row-level security), dynamic data masking, Always Encrypted. A
  `CREATE SYNONYM` in a SQL schema never reaches the target, and stays behind on a dev database you
  provide, where the next run fails with `There is already an object named 'x'`
- `SPARSE`, `ROWGUIDCOL`, and `FILESTREAM` columns: a `SPARSE` column in a SQL schema is created
  without `SPARSE`, without an error
- Data compression, filegroups for tables and indexes (`ON [fg]`), `TEXTIMAGE_ON`, `LOCK_ESCALATION`
- Index options other than `fill_factor` and the spatial-index options, such as `IGNORE_DUP_KEY` or
  `DATA_COMPRESSION`
- Statistics, DDL and database triggers, Service Broker, external tables, XML schema collections, CLR
  assemblies and types, and the `vector` type
- Logins, server roles, `DENY`, and database-level permissions (`references/security.md`)
- Database settings (`ALTER DATABASE`), such as the collation or compatibility level
