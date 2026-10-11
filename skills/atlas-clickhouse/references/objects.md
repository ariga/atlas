# ClickHouse Objects and Their Nuances

How Atlas models ClickHouse objects, what each one needs, and the cases where its plan differs from what
a hand-written migration would do. The full attribute reference is https://atlasgo.io/hcl/clickhouse,
and the overview is https://atlasgo.io/guides/clickhouse/supported-objects. HCL syntax is in
`references/hcl.md`; rebuilds, skipped changes, and lint checks are in `references/migrations.md`.

## What Each Object Needs

The ClickHouse driver needs `atlas login` for everything. If `atlas whoami` shows no login, ask the user
to run `atlas login`; a user without an account can start a free trial with it. Beyond the login, each
object needs a scope, and some need a flag in the env's `schema.mode` block:

| Object | Scope | Inspected |
|--------|-------|-----------|
| Databases, tables, views, materialized views | Database or server | By default |
| Dictionaries | Database or server | By default, or with `mode "clickhouse" { dictionaries = true }` when `objects` is off |
| UDFs (lambda and WASM) | Server | By default |
| WASM modules | Server | With `mode "clickhouse" { wasm_modules = true }` |
| Users and roles | Server | With `mode { roles = true }` |
| Grants | Database or server | With `mode { permissions = true }` |
| Row policies | Server | With `mode "clickhouse" { policies = true }` |
| Settings profiles | Server | With `mode "clickhouse" { settings_profiles = true }` |
| Named collections | Server | With `mode "clickhouse" { named_collections = true }` |

A `mode "clickhouse"` block also accepts the generic flags, such as `roles`, `permissions`, and
`sensitive`. A database-scoped URL never inspects server-level objects, so a desired state that declares
them needs server scope (`references/dev-database.md`).

## Table Engines

`engine` takes `MergeTree`, `SharedMergeTree`, `Memory`, or `TimeSeries` as enums, and any other engine
as an expression: `engine = sql("ReplacingMergeTree(version)")`. The engine decides how Atlas applies a
change that ClickHouse cannot make with `ALTER`:

| Engine | A sort key, partition, primary key, or engine change |
|--------|-----------------------------------------------------|
| MergeTree family | Copies the rows into `<table>_tmp` and swaps it in with `EXCHANGE TABLES` |
| `Replicated*` with quoted arguments, such as an explicit ZooKeeper path | Drops and creates the table again, which loses its rows. Asked in a terminal, skipped without one |
| `TimeSeries` | Any change drops and creates it again, losing the data of its inner tables. Asked in a terminal, skipped without one |
| `Kafka`, `RabbitMQ`, `NATS`, `S3Queue`, `FileLog` | Any change drops and creates it again. These hold no rows |
| `Distributed`, `Merge`, `Buffer`, `Null`, `S3`, `URL`, `File`, `MySQL`, `PostgreSQL`, and other engines whose data lives elsewhere | Drops and creates it again, without copying rows, which would duplicate them at the source |
| `KeeperMap` | Drops and creates it again, without copying rows and without asking. Its rows live in Keeper and can be lost with the drop: change it by hand |

- A MergeTree table without a primary key is created with `PRIMARY KEY tuple()`. Without a `sort`
  block, the table sorts by its primary key.
- `index_granularity = 8192` is the MergeTree default; inspection prints it in `settings`.
- On ClickHouse Cloud, MergeTree and `Replicated*` engines and their `Shared` replacements compare as
  equal (`references/dev-database.md`, ClickHouse Cloud).
- In a database with the `Replicated` engine, statements carry no `ON CLUSTER`: the database replicates
  them.
- A table needs an engine and at least one column, or planning fails with
  `creating table "<name>": table engine is required` or `creating table "<name>": table has no columns`.
- Tables with the `Dictionary` engine, and `WindowView` and `LiveView` objects, are not managed: Atlas
  does not read the first back, and reads the others as plain views.

## Database Engines

A `schema` block takes `engine = Atomic`, `engine = Lazy`, or an expression:

```hcl
schema "app" {
  engine = sql("Replicated('/clickhouse/{cluster}/databases/{database}/{shard}', '{shard}', '{replica}')")
}
```

- Atlas does not plan changes to a database's engine or comment.
- Database `settings` are changed with `ALTER DATABASE ... MODIFY SETTING`. ClickHouse has no way to
  reset one, so removing a setting fails with
  `modifying schema "<name>": dropping setting "<key>" is not supported by ClickHouse`, and removing all
  of them with `modifying schema "<name>": dropping settings is not supported by ClickHouse`.
- On the `MySQL` and `DataLakeCatalog` database engines, a settings change drops and creates the
  database again. These keep no local data.
- `EXCHANGE TABLES` needs an `Atomic` or `Replicated` database. In other databases, a table rebuild uses
  `DROP TABLE` and `RENAME TABLE` instead.
- The `default` database is written as `CREATE DATABASE IF NOT EXISTS default` and
  `DROP DATABASE IF EXISTS default`.

## Columns

ClickHouse converts many column types in place with `MODIFY COLUMN`. A change it rejects is applied by
dropping the column and adding it again, which loses its values. In a terminal, Atlas asks first; without
one, it leaves the change out of the plan. The changes that need it:

- An `Enum` whose existing values change their numbers, also inside `Nullable` or `LowCardinality`:
  `Enum conversion changes value for column "<name>"`.
- A smaller `FixedString`, also inside `Nullable`, `LowCardinality`, or `Array`:
  `column "<name>" FixedString size decrease from <n> to <m> is not supported in-place`.
- A conversion ClickHouse does not support:
  `column "<name>" type conversion is not supported in-place`. `UUID` converts only to and from
  `String`; `IPv4` and `IPv6` only to and from `String`, and `IPv4` to and from `UInt32`; `Array`,
  `Map`, and `Tuple` only within their own kind or to `String`. `String` converts to `Array` or `Tuple`,
  but not to `Map`.

A type change on a column that a key, a skip index, or a projection uses is not detected in advance:
Atlas plans `MODIFY COLUMN`, and the plan fails when Atlas runs it on the dev database, with an error
that starts with `analyzing plan: executing statement: ALTER TABLE ... MODIFY COLUMN` and ends with the
server's reason:

- A sort or primary key column: `code: 524, message: ALTER of key column <name> from type <a> to type
  <b> is not safe because it can change the representation of primary key`.
- A skip index column: `code: 524, message: Trying to ALTER <name> column which is a part of index <index>`.
- A projection column: `code: 524, message: Cannot ALTER column: Cannot apply mutation because it breaks
  projection <projection>`.

Change a key column with a table rebuild (`references/migrations.md`, Table Rebuilds). For an index or
a projection, write a migration with `atlas migrate new`: drop it, `MODIFY COLUMN`, add it back, and
`MATERIALIZE` it.

When the recreate is chosen, Atlas drops each skip index on the column, drops the column, adds it at the
end of the table, adds the indexes back, and materializes them, each in its own `ALTER TABLE`. None of
these statements has a reverse, so `atlas migrate down` cannot undo them. Comment, codec, TTL, and
default changes on the same column are planned in place either way.

- Adding `Enum` values, with new numbers, is an in-place `MODIFY COLUMN`. So is removing one, with no
  prompt: Atlas does not check for rows that still hold the value. Update those rows first.
- The default kind of an `EPHEMERAL` column cannot change:
  `diffing column "<name>": cannot change default kind of ephemeral column`.
- New columns keep their declared position, with `FIRST` or `AFTER`.
- A type Atlas cannot format fails the plan with an error that starts with
  `formatting type for column "<name>": unsupported type` (`references/hcl.md`, Columns and Types).
- An expression default is compared by evaluating `SELECT <from> = <to>` on the dev database. A default
  that names another column cannot be evaluated this way, and is planned as changed when its text
  differs.

## Views

- A view is changed with `CREATE OR REPLACE VIEW`. When a change adds, drops, or modifies a column of a
  table the view selects, Atlas recreates the view too, so its column list stays current. It finds the
  tables by parsing the view's query; a query it cannot parse is not recreated.
- A view without `security` is compared with the server's `default_normal_view_sql_security` setting,
  and a materialized view defaults to `DEFINER`. When the dev server and the target disagree on the
  setting, plans keep showing a `security` change: declare `security` on each view.
- `definer` names the user or role the view runs as.

## Materialized Views

- A materialized view either writes into a table, `to = table.events_daily`, or stores rows itself with
  an `engine`, a `sort`, and a `partition`. `populate = true` fills it from existing rows, only with an
  engine.
- Any change except to its `refresh` drops and creates the view again. A view with its own `engine`
  loses the rows it stored; a view with `to` keeps them in the target table. Rows inserted into the
  source while the view is missing are not forwarded. Prefer `to` for views that hold data.
- A column change on a source table does not recreate a materialized view. Change the view's query in
  the same change when it needs the new column.
- A refreshable view's `refresh` block changes in place with `ALTER TABLE ... MODIFY REFRESH`, except
  `append`, which needs a recreate. `depends_on` in `refresh` writes `DEPENDS ON` with the refreshable
  views it names, in any database.

## Dictionaries

- A dictionary is created and changed with `CREATE OR REPLACE DICTIONARY`.
- Without a `lifetime` block, Atlas writes `LIFETIME(MIN 0 MAX 0)`, and the dictionary never reloads on
  its own. The `DIRECT` layout takes no lifetime.
- A literal attribute `default` is written as `DEFAULT '<value>'`, and a `sql()` default as
  `EXPRESSION <expr>`, which ClickHouse computes from the source row instead.
- Atlas does not apply a dictionary's `comment`, and does not plan a change to an attribute's `default`
  alone. Attributes are compared by position: renaming one, with the same type, plans nothing. Replace
  the dictionary by hand for these changes.
- Atlas orders dictionaries after the tables, views, and named collections their source reads, and drops
  a dictionary only after the tables and views that call it, such as a `dictGet` in a column default.
- A `source` with a password is a secret: see `references/security.md`, Secrets. Prefer a named
  collection for the credentials.

## UDFs

- Lambda UDFs are server-level, and have no `schema`: `function "plus_one" { as = "(x) -> x + 1" }`.
  Atlas changes one with `CREATE OR REPLACE FUNCTION`.
- In cluster mode, function DDL carries `ON CLUSTER`, unless both URLs set
  `has_user_defined_zookeeper_path=true` (`references/dev-database.md`).
- The `drop_func`, `add_func`, and `modify_func` skip policies apply to UDFs.

## WASM Modules and UDFs

WASM modules and WASM UDFs need ClickHouse 26.3 or later, with WebAssembly enabled in the server
configuration, on the target and on the dev database, and `wasm_modules = true` in `mode "clickhouse"`
(https://atlasgo.io/guides/clickhouse/wasm).

- A module's `code` is write-only: inspection returns only its `hash`, so Atlas detects a module change
  only by its hash. Without `hash`, the dev database computes one, used for comparison and never written.
- A module's `hash` is exactly `sql("reinterpretAsUInt256(unhex('<sha256 hex>'))")`; other forms fail
  with `wasm_module "<name>": unsupported hash expression`. A function's `hash` is the bare 64-character
  hex digest, or fails with `function "<name>": invalid hash "<hash>"`.
- `module` takes a `wasm_module.<name>` reference, or a string naming a module Atlas does not manage.
- Atlas creates or replaces a module before the functions that use it, and drops functions before the
  module they use. A function whose module is replaced in the same plan is dropped and created again.
- On ClickHouse Cloud, both fail: `WASM functions are not supported on ClickHouse Cloud` and
  `WASM modules are not supported on ClickHouse Cloud`.

## Named Collections

Named collections need `named_collections = true`, server scope, and a user allowed to manage them
(`named_collection_control`), on the target and the dev database
(https://atlasgo.io/guides/clickhouse/named-collections).

- A collection needs at least one `item`: `named collection "<name>" must have at least one item`.
- Values are secrets: see `references/security.md`, Secrets.
- `overridable` (`ALLOW` or `DENY`) needs ClickHouse 25.5:
  `item "<key>": overridable requires ClickHouse 25.5 or above`. Once set, it can only be removed by
  dropping and creating the collection again:
  `item "<key>": overridable can only be reset by dropping and re-creating the collection`.
- Atlas creates a collection before the tables and dictionaries that use it, and drops it after them.
- Inspection never reads named collections on ClickHouse Cloud, or below 23.10. There, a declared
  collection is planned for creation on every run, and an existing one is never changed or dropped.

## Normalized Definitions

ClickHouse rewrites expressions when it stores them: defaults, TTLs, sort keys, partition keys, and view
queries. Atlas compares them after loading the desired state on the dev database, and compares an
expression default by evaluating `SELECT <from> = <to>` there. Always plan with a dev URL, on the
target's version.

## Not Inspected

Inspection skips these. A declared one is planned for creation on every run, and an existing one is
never changed or dropped:

- Tables and views whose name contains `.inner_id`, or starts with `_tmp_replace_` or `_peerdb_`
  (ClickPipes).
- Skip indexes named `auto_minmax_index_*`, which ClickHouse creates for the
  `add_minmax_index_for_numeric_columns` and `add_minmax_index_for_temporal_columns` table settings.
  Set the setting; do not declare the indexes.
- Named collections on ClickHouse Cloud.
- Global and system grants, such as `GRANT ... ON *.*`, and the grants of the user Atlas connects as
  (`references/security.md`, Permissions).
- Users and roles that inspection filters (`references/security.md`, Which Users and Roles Atlas
  Manages).

## Not Planned

- A database's `engine` and `comment`.
- A `comment` on a role or user: the blocks have no such attribute.
- A user's `auth_type` change (`references/security.md`, Users and Roles).
- A dictionary's `comment`, and a change to an attribute's `default` alone.
- Constraint renames.
- A password change in a migration file, or a password removal (`references/security.md`, Passwords).
- Value changes of named collections, without the sensitive mode.
- The `skip` diff policy has no option for dictionaries, named collections, row policies, settings
  profiles, or WASM modules: to keep Atlas from dropping undeclared ones, turn off their `mode` flag
  (`references/migrations.md`, Diff Policies).
