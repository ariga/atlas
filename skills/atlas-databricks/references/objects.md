# Databricks Objects and Their Nuances

How Atlas models Databricks objects, the cases where its plan differs from what a hand-written
migration would do, and the changes it refuses to plan. The full attribute reference is
https://atlasgo.io/hcl/databricks, and the HCL for each object is in `references/hcl.md`.

## What Needs a Login

Everything. The Databricks driver itself needs `atlas login`, so a logged-out command fails with
`login is required to use the 'databricks' driver` instead of skipping objects. If `atlas whoami` shows
no login, ask the user to run `atlas login`; a user without an account can start a free trial with it.

## Table Properties

Databricks writes `delta.*` properties onto every table it creates, such as `delta.minReaderVersion`,
`delta.enableDeletionVectors`, and `delta.feature.*` flags. Atlas handles them in three ways:

- A property the desired state lists is compared, and a different value plans
  `ALTER TABLE ... SET TBLPROPERTIES ('name' = 'value')`.
- A property the target holds and the desired state does not list is ignored. Removing a property from
  the schema therefore plans nothing; write `ALTER TABLE ... UNSET TBLPROPERTIES ('name')` in a migration
  file by hand.
- Some properties never appear in inspection, as Databricks assigns them per object: `delta.rowTracking*`,
  `delta.columnMapping.maxColumnId`, `io.unitycatalog.*`, and `databricks.*`. The clustering properties
  (`clusteringColumns`, `delta.feature.clustering`) are reported as the `cluster` block instead, and
  each `delta.constraints.<name>` property as a `check` block.

`atlas schema inspect` prints every property the table holds, Databricks' defaults included. When
turning the output into the desired state, keep only the properties the team sets on purpose, such as
`delta.enableChangeDataFeed` or a team property. A listed property is enforced: when the target holds
another value, the plan sets it back.

A view's properties are different: they come from its definition, so removing one from the schema plans
`ALTER VIEW ... UNSET TBLPROPERTIES`.

## Managed and External Tables

- A table without `location` is a managed Delta table. `location = "s3://..."` writes the `LOCATION`
  clause and makes the table external. Declare `location` only for external tables.
- When both the desired state and the target declare a location and the paths differ, Atlas plans
  `ALTER TABLE ... SET LOCATION '...'`. Adding or removing `location` on an existing table plans
  nothing.
- `provider` sets the data source, written as `USING <provider>`. Delta is the default and is never
  written; inspection prints `provider = "delta"` for every table. Databricks accepts only Delta for
  managed tables, so another provider, such as `"parquet"`, needs a `location`.
- The provider is set on creation only. A change fails with
  `databricks: alter table "...": the data source of a table is set on creation only`.
- On the dev catalog, Atlas creates every table without its location, so the dev table never points
  at the target's storage (`references/dev-database.md`, What normalization keeps).

## Partitioning and Clustering

- `partition { columns = [...] }` writes `PARTITIONED BY`. It is set on creation only: a change fails
  with `databricks: alter table "...": the PARTITIONED BY clause of a table is set on creation only`.
  Repartitioning means a new table and a data copy, in a hand-written migration.
- `cluster { columns = [...] }` writes liquid clustering, `CLUSTER BY (...)`, and `cluster { auto = true }`
  writes `CLUSTER BY AUTO`. Clustering changes in place: `ALTER TABLE ... CLUSTER BY (...)`, and removing
  the block plans `CLUSTER BY NONE`.
- Partitioning and clustering are alternatives on Databricks. A `materialized` block that declares
  both fails validation; a `table` that declares both is rejected by Databricks when it is created.
  Declare one per table.

## Keys and Constraints

- Primary and foreign keys are informational: Databricks enforces neither. Atlas writes foreign keys
  without referential actions, and the `foreign_key` block takes no `on_delete` or `on_update`.
- A primary key may be unnamed. Atlas then writes `PRIMARY KEY (...)` without `CONSTRAINT`, Databricks
  names it, and later plans keep that name.
- Neither key can be altered. A changed primary key plans `DROP PRIMARY KEY IF EXISTS` and
  `ADD PRIMARY KEY`; a changed foreign key plans `DROP CONSTRAINT` and `ADD CONSTRAINT` under the same
  name. Lint reports the drop (`CD101`, `CD103`), which is expected for these plans.
- Databricks has no `RENAME CONSTRAINT`. Renaming a named primary key, on the same columns, fails at
  planning with `databricks: alter table "...": databricks: unsupported modify table change ...`.
  Write `DROP PRIMARY KEY` and `ADD CONSTRAINT <name> PRIMARY KEY (...)` in a hand-written migration.
- `CHECK` constraints are stored in the table's `delta.constraints.<name>` properties. A new table gets
  its checks in `ALTER TABLE ... ADD CONSTRAINT ... CHECK (...)` statements right after
  `CREATE TABLE`.
- Atlas matches checks by name. Changing the expression of a check under the same name fails at
  planning with the same `unsupported modify table change` error. Give the changed check a new name:
  Atlas then drops the old constraint and adds the new one in one plan.
- The Databricks HCL has no `index` block.

## Generated and Identity Columns

- `as = "quantity * 2"` writes `GENERATED ALWAYS AS (quantity * 2)`. Databricks stores the expression in a
  form of its own; Atlas normalizes both sides before comparing them.
- An `identity` block writes `GENERATED ALWAYS AS IDENTITY (START WITH 1 INCREMENT BY 1)`, or
  `GENERATED BY DEFAULT` with `by_default = true`, which lets inserts set the column. `start` and
  `increment` default to 1.
- An identity column must be `BIGINT`: another type fails with `identity column "..." must be BIGINT`,
  and `increment = 0` with `identity column "..." takes a nonzero increment`.
- A column takes one of `default`, `as`, and `identity`.
- Both clauses exist only at table creation. Adding a generated or identity column to an existing table
  fails with `add column "...": a generated column is set on table creation only`, and changing the
  clause fails with `modify column "...": the generated clause of a column is set on creation only`.
  Dropping such a column works, but the statement has no reverse, so the file cannot be reverted with
  `atlas migrate down`.

## Defaults

- A column with a `default` needs the Delta feature `allowColumnDefaults`. Atlas adds
  `'delta.feature.allowColumnDefaults' = 'supported'` to a new table's `TBLPROPERTIES`, and before
  adding a column with a default to an existing table it plans
  `ALTER TABLE ... SET TBLPROPERTIES ('delta.feature.allowColumnDefaults' = 'supported')`.
- Delta takes neither a default nor `NOT NULL` in `ADD COLUMN`. A new column with either is added
  nullable and plain, then altered: `ALTER COLUMN ... SET DEFAULT ...` and `ALTER COLUMN ... SET NOT NULL`.
- Numbers and booleans are written as they are, and strings and dates are quoted: `default = "unknown"`
  writes `DEFAULT 'unknown'`. Expressions go in `sql()`: `default = sql("current_timestamp()")`.

## Column Changes

Databricks accepts one `ALTER COLUMN` alternative per statement, so a column whose nullability and
comment both change takes two statements. Each kind of change:

| Change | Statement Atlas plans |
|--------|----------------------|
| Nullability | `ALTER COLUMN c DROP NOT NULL` or `SET NOT NULL` |
| Default | `ALTER COLUMN c SET DEFAULT ...` or `DROP DEFAULT` |
| Comment | `ALTER COLUMN c COMMENT '...'` |
| Mask | `ALTER COLUMN c SET MASK ...` or `DROP MASK` |
| Type or collation | `ALTER COLUMN c TYPE <type> [COLLATE <collation>]` |
| Drop | `DROP COLUMN IF EXISTS c` |

- A drop first plans `ALTER TABLE ... SET TBLPROPERTIES ('delta.columnMapping.mode' = 'name')`, which
  Delta requires for it. Expect that statement in every such plan.
- A renamed column is planned as a drop and an add, and its data is lost (Renames below).
- Atlas plans a type change as `ALTER COLUMN ... TYPE` and leaves it to Databricks to accept it. Delta
  accepts only some type changes on an existing column. When Databricks rejects one, add a new column,
  backfill it, and drop the old one, in a hand-written migration. Test type changes with seeded rows
  first (`references/testing.md`).
- Removing a column's collation plans `TYPE STRING COLLATE <table default>`: Databricks keeps a
  collation on every string column, so Atlas sets the one the column inherits. A collation on a
  non-string column fails with `modify column "...": only a string column takes a collation`.

## Renames

Atlas has no rename hints on Databricks. The HCL has no `renamed_from` attribute, so one on a column
or a table fails with `unknown attribute renamed_from in block of type column` (or `table`), and a SQL
schema's `-- atlas:renamed_from` directive has no effect. A renamed column is planned as
`DROP COLUMN` and `ADD COLUMN`, and a renamed table as `DROP TABLE` and `CREATE TABLE`: the data is
lost. Lint reports the drop (`DS102` or `DS103`).

Write the rename in a migration file by hand, and rename the object in the schema to match:

```sql
-- atlas:txmode none

ALTER TABLE `default`.`users` SET TBLPROPERTIES ('delta.columnMapping.mode' = 'name');
ALTER TABLE `default`.`users` RENAME COLUMN `email` TO `customer_email`;
```

- Create the file with `atlas migrate new rename_email --edit`, run `atlas migrate hash`, then run
  `atlas migrate diff` and confirm it reports no changes.
- A table rename is ``ALTER TABLE `default`.`users` RENAME TO `customers` ``.
- In a declarative workflow, run the rename on the target first, then `atlas schema apply`.
- Test the file with seeded rows before applying it (`references/testing.md`).

## Collations

- `collation` on a schema writes `DEFAULT COLLATION`, and changing it plans
  `ALTER SCHEMA ... DEFAULT COLLATION ...`. Inspection prints the schema's collation, such as
  `collation = "UTF8_BINARY"`.
- A schema that declares no collation keeps whatever it has, which may come from its catalog. Naming
  the default, `collation = "UTF8_BINARY"`, is how a schema asks for it.
- A table or a column without `collation` inherits it from its schema or table. Atlas compares the
  collation each object ends up with, so declaring the inherited one is not a change.
- A table that inherits from its schema follows a schema collation change: Atlas alters the schema and
  each such table in one plan, since Databricks fixes a table's collation when it is created.

## Types

Atlas models the numeric, string, binary, boolean, date, and timestamp types (including
`TIMESTAMP_NTZ`), `DECIMAL(p,s)`, and the composite `ARRAY<...>`, `MAP<..., ...>`, and `STRUCT<...>`.

`VARIANT`, `OBJECT`, `INTERVAL`, `GEOGRAPHY`, `GEOMETRY`, and type names Atlas does not know are
inspected as they are, so inspecting a schema that holds them works. Atlas cannot write them:
creating such a column fails with `databricks: unsupported type: "VARIANT"`, and a plan that compares a
table holding one fails as well. Keep those tables out of Atlas with `exclude`, for example
`exclude = ["default.raw_events"]` at catalog scope, and manage them by hand.

## Views and Materialized Views

Atlas creates both with `CREATE OR REPLACE`, and changes them in place when the Databricks `ALTER`
grammar covers the change. Otherwise it rewrites them:

| Change | Plain view | Materialized view |
|--------|-----------|-------------------|
| Query (`as`) | `ALTER VIEW ... AS ...` | Rewrite |
| Properties | `ALTER VIEW ... SET` or `UNSET TBLPROPERTIES` | Rewrite |
| Schema binding (`schema_binding`) | `ALTER VIEW ... WITH SCHEMA ...` | Not a clause of a materialized view |
| Refresh schedule (`refresh`) | Not a clause of a plain view | `ALTER MATERIALIZED VIEW ... ADD`, `ALTER`, or `DROP SCHEDULE` |
| Row filter | No `row_filter` block | `SET ROW FILTER` or `DROP ROW FILTER` in place |
| Column comment or mask | Rewrite | `ALTER MATERIALIZED VIEW ... ALTER COLUMN ...` in place |
| Comment, collation, columns, keys, expectations, partitioning, clustering, provider | Rewrite | Rewrite |

- Rewriting a materialized view re-provisions the pipeline behind it and refreshes it in full. On a
  large view, review the plan for `CREATE OR REPLACE MATERIALIZED VIEW` before applying it.
- A rewrite drops the view's grants; Atlas grants them again in the same plan
  (`references/security.md`, Rewrites keep grants).
- A plain view lists only column aliases, each with an optional comment. A materialized view lists full
  column definitions, with expectations (`expect`), a primary key, and foreign keys.
- Databricks binds a plain view `WITH SCHEMA COMPENSATION` by default and writes it back, so a migration
  generated with a dev database shows that clause even when the schema declares no `schema_binding`.
- Databricks rewrites expectation expressions: `priority BETWEEN 1 AND 5` is stored as
  `between(priority, 1, 5)`, and `1.0` as `1.0BD`. Atlas compares the normalized forms, so plans stay
  stable, and migrations show Databricks' form.
- A refresh interval reads back in a unit Databricks picks (`"1 DAY"` as `"1 DAYS"`). Atlas compares
  intervals as durations, so the difference is not a change.
- Inspection skips the tables Databricks keeps for pipelines: names starting with `event_log_` and
  `__materialization_mat_`.

## Functions and Procedures

- Functions take `lang = SQL` or `lang = PYTHON`. A SQL function's `as` is the expression after
  `RETURN`; a Python body is written between `$$` markers as it is. `return_table` declares a table
  function (`RETURNS TABLE (...)`).
- Databricks identifies a function by name, so every change, a new signature included, plans
  `CREATE OR REPLACE FUNCTION` with the whole definition, followed by its grants.
- Databricks derives determinism and SQL data access for a routine that declares neither. Atlas
  compares them only when the schema declares them (`deterministic`, `sql_data_access`).
- A procedure is always `LANGUAGE SQL`. Atlas writes `SQL SECURITY INVOKER` unless `security = DEFINER`,
  and its body goes in `as`, such as `"BEGIN SET value = value * 2; END"`.
- Procedure clauses Databricks rejects fail before planning: `LANGUAGE PYTHON is not a clause of CREATE
  PROCEDURE`, `DETERMINISTIC is not a clause of CREATE PROCEDURE` (only `deterministic = false` is
  accepted), and `parameter "..." of mode OUT takes no default`.
- Masks and row filters reference functions; Atlas creates the function before the table or view
  that uses it (`references/security.md`, Column Masks and Row Filters).

## Schemas

- Atlas writes `CREATE SCHEMA` with `COMMENT` and `DEFAULT COLLATION`, and `IF NOT EXISTS` for
  `default`, which every catalog already has.
- A comment change plans `COMMENT ON SCHEMA ... IS '...'`.
- Databricks gives the `default` schema of a new catalog the comment `Default schema (auto-created)`.
  Atlas hides it, so `schema "default" {}` matches a fresh catalog.
- Dropping a schema plans `DROP SCHEMA ... CASCADE`, which drops every object in it.

## Statistics

`atlas schema stats inspect --env <name>` reports each table's size in bytes from `DESCRIBE DETAIL`, as
Prometheus metrics.

## Not Managed by Atlas

Atlas has no blocks for catalogs, volumes, tags, streaming tables, external locations, storage
credentials, connections, models, or principals, and does not inspect owners. The HCL has no `data`
block, so seed rows go in migration files or Data Scripts. Manage these objects outside Atlas, and keep
any of them that inspection reports out of plans with `exclude`.
