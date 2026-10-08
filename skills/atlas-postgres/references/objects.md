# PostgreSQL Objects and Their Nuances

How Atlas models PostgreSQL objects, and the cases where its plan differs from what a hand-written
migration would do. The full attribute reference is https://atlasgo.io/hcl/postgres, with examples in
https://atlasgo.io/atlas-schema/hcl.

## What Needs a Login

Tables, indexes, check, unique, and exclusion constraints, foreign keys, enum types, and comments work
without a login. Everything else needs `atlas login`: views, materialized views, functions,
aggregates, procedures, triggers and event triggers, sequences, extensions, partitions, policies,
domain, composite, and range types, and foreign servers (https://atlasgo.io/features).

- Logged out, inspection skips those objects without an error, so a schema exported logged out
  silently leaves them out.
- A desired state that defines them fails with an error containing
  `are available to logged-in users only`.

If `atlas whoami` shows no login, ask the user to run `atlas login` before going further: logged out,
Atlas misses most of a PostgreSQL schema. A user without an account can start a free trial with
`atlas login`.

## Sequences, Serial, and Identity

- `smallserial`, `serial`, and `bigserial` stand for an integer column, a sequence, and a `nextval`
  default. Inspection prints `type = serial` and hides the implicit sequence, and so do identity
  columns. Write a `sequence` block only for a sequence the database did not create for a column
  (https://atlasgo.io/atlas-schema/hcl#sequence).
- A `sequence` block takes `type`, `start`, `increment`, `min_value`, `max_value`, `cache`, and `cycle`
  (https://atlasgo.io/hcl/postgres#sequence). `owner` ties it to a column, which is `OWNED BY`:

  ```hcl
  sequence "s3" {
    schema  = schema.public
    owner   = table.t2.column.id
    comment = "Sequence with column owner"
  }
  ```

- What Atlas plans when a column changes (https://atlasgo.io/guides/postgres/serial-columns):
  - `serial` to `bigserial`: only `ALTER COLUMN "c" TYPE bigint`. The sequence stays `AS integer` and
    stops at 2147483647, so add `ALTER SEQUENCE "public"."t_c_seq" AS bigint;` to the migration.
  - `bigserial` to `bigint`: drops the default and the sequence.
  - `bigint` to `serial`: creates an owned sequence and sets the default, and nothing more:

    ```sql
    CREATE SEQUENCE IF NOT EXISTS "public"."t_c_seq" OWNED BY "public"."t"."c";
    ALTER TABLE "public"."t" ALTER COLUMN "c" SET DEFAULT nextval('"public"."t_c_seq"'), ALTER COLUMN "c" TYPE integer;
    ```

    The new sequence starts at 1, so on a table with rows, inserts fail with a duplicate key error until
    the sequence is moved past the existing values. Run this after the migration, or add it to the
    migration file:

    ```sql
    SELECT setval('"public"."t_c_seq"', (SELECT MAX("c") FROM "t"));
    ```

- Grants on sequences take `SELECT`, `UPDATE`, and `USAGE`:

  ```hcl
  permission {
    to         = PUBLIC
    for        = sequence.s1
    privileges = [SELECT, UPDATE, USAGE]
  }
  ```

- A grant on the implicit sequence of a `serial` column changes how the column is managed: Atlas
  surfaces the sequence as an explicit object and demotes the column from `bigserial` to `bigint` with a
  `nextval(...)` default, so both the sequence and its grants are managed. A `serial` column without a
  grant is unchanged (https://atlasgo.io/atlas-schema/hcl#permissions).
- Identity columns need PostgreSQL 10 or later. Adding one to an existing table rewrites it (`PG310`):

  ```hcl
    column "id" {
      null = false
      type = int
      identity {
          generated = ALWAYS
          start = 10
          increment = 10
      }
    }
  ```

- An HCL `data` block cannot seed a table whose primary key is an identity column; seed it from SQL.
  When the rows come from SQL files or a database, Atlas leaves identity primary-key values out unless
  the env's `data` block in `atlas.hcl` sets `preserve_ids = true`, which applies to every managed table.
  With `GENERATED ALWAYS`, Atlas then adds `OVERRIDING SYSTEM VALUE`. `serial` values are always
  written.

## Enums

- `enum` takes `schema` and a list of `values`; a column uses it as `type = enum.status`.
- Adding a value plans `ALTER TYPE ... ADD VALUE`. A migration that adds a value and also uses it fails
  with `unsafe use of new value`: add the value in one migration file and use it in a later one.
- Removing or reordering values is not planned: make those changes in a hand-written migration. The
  plan for such a change can report no changes, or fail with
  `replacing or reordering enum ("<name>") value is not supported`, so never read "no changes" as done.
- Converting a nullable column to an enum that lacks some of the column's values sets those rows to
  `NULL`. Check the column's distinct values before the change.

## Domains, Composite, and Range Types

- A `domain` wraps a base type with `null`, `default`, and `check` blocks that refer to `VALUE`. In
  HCL, a domain without `null = true` is `NOT NULL`, like a column
  (https://atlasgo.io/atlas-schema/hcl#domain):

  ```hcl
  domain "username" {
    schema = schema.public
    type    = text
    null    = false
    default = "anonymous"
    check "username_length" {
      expr = "(length(VALUE) > 3)"
    }
  }
  ```

- A `composite` type needs at least one `field`; a column uses `type = composite.address`.
- Built-in range types such as `tsrange` need no block. A custom `range` takes a `subtype`, and Atlas
  hides the constructor functions and the multirange type PostgreSQL creates for it.

## Column Changes

- A type conversion that PostgreSQL cannot make implicitly, such as `text` to `integer` or `uuid`, needs
  a `USING` clause. When the plan has none, its `ALTER COLUMN ... TYPE` fails when applied. Add the
  conversion to the migration file, then run `atlas migrate hash`:

  ```sql
  ALTER TABLE "t" ALTER COLUMN "c" TYPE integer USING "c"::integer;
  ```

- Changes PostgreSQL cannot make in place, such as a function's return type, a domain's base type, or,
  before PostgreSQL 17, a generated column's expression, are planned as a drop and a create. Changing a
  range type that a column uses is refused. Review the plan for drops before applying.
- A type change that is not binary-coercible rewrites the table (`PG301`); see Rewrites and Scans in
  `references/migrations.md`.

## Extensions

Extensions belong to the database, so they need database scope; `references/dev-database.md` covers the
dev database, versions, and managed services. `schema` sets where the extension's objects install:

```hcl
extension "postgis" {
  schema  = schema.public
}
```

pgvector's extension is named `vector`. Its IVFFlat and HNSW indexes need `extension "vector"` and the
operator class as a string, such as `ops = "vector_l2_ops"`.

## Functions, Procedures, and Triggers

- A `function` needs `schema`, `lang`, and `as`, and returns one of `return`, `return_set`, or
  `return_table`. A trigger function uses `return = trigger`. `config_params` pins settings such as
  `search_path` for the function. A `procedure` takes the same blocks without a return.
- A PostgreSQL `trigger` calls a function with `execute`. Set `for = ROW` when the function reads `NEW`
  or `OLD`: PostgreSQL's default is `FOR EACH STATEMENT`, where both are null. A function the schema does
  not manage is called with `sql()`, and its arguments go in `args`
  (https://atlasgo.io/atlas-schema/hcl#trigger):

  ```hcl
  trigger "users_set_updated_at" {
    on = table.users
    before {
      update = true
    }
    for = ROW
    execute {
      function = sql("public.moddatetime")
      args     = ["updated_at"]
    }
  }
  ```

  A built-in function needs no schema: `function = sql("suppress_redundant_updates_trigger")`.
- On PostgreSQL, a trigger's `as` takes the action clause as raw SQL, in place of `execute`. A trigger
  cannot set both.
- An `event_trigger` belongs to the database, not to a schema. A schema-scoped dev URL silently drops
  event triggers from migrations; use a database-scoped one
  (https://atlasgo.io/faq/event-trigger-search-path).

## Views and Materialized Views

- A view takes `as`, and optionally `check_option` and `depends_on`. PostgreSQL view security goes in a
  `security` block with `invoker` and `barrier`.
- A `materialized` view takes `as` and its own `index` blocks. The `materialized { with_no_data = true }`
  diff policy creates materialized views `WITH NO DATA`.
- A materialized view that selects from a `postgres_fdw` foreign table connects to the remote server when
  it is created, so loading the schema on the dev database fails with `could not connect to server`.
  Stub the remote server in the dev container with `extra_hosts` and an `init` script
  (https://atlasgo.io/faq/mock-postgres-fdw-foreign-server).

## Partitions

The parent table declares `partition { type = RANGE, LIST, or HASH }` with its columns. Each child is a
top-level `partition` block with `schema`, `of`, and its bounds; a child without bounds is the default
partition (https://atlasgo.io/atlas-schema/hcl#partitions):

```hcl
partition "invoices_2025_06" {
  schema = schema.public
  of     = table.invoices
  range {
    from = ["'2025-06-01'"]
    to   = ["'2025-07-01'"]
  }
}
```

Bound values are SQL literals inside strings. An index on a partitioned table cannot be built
concurrently; see `references/migrations.md`.

- Atlas refuses to change a table's partition key, or to convert a table to partitioned or back: the
  error says `DROP and CREATE is required`. Make that change in a hand-written migration.
- Table inheritance (`INHERITS`) is not modeled: each child shows up as a standalone table.

## Indexes and Constraints

- An `index` takes `columns`, or `on` blocks with `column` or `expr`, plus `desc`, `ops` (the operator
  class), `include`, `where`, `type` (`BTREE`, `BRIN`, `HASH`, `GIN`, `GIST`, `SPGIST`), `unique`, and
  `nulls_distinct`. An operator class (https://atlasgo.io/guides/postgres/index-operator-classes):

  ```hcl
    index "internet_provider_idx" {
      type = BTREE
      on {
        column = column.subscriber_name
        ops    = varchar_pattern_ops
      }
    }
  ```

- Expression and partial indexes need a dev database to normalize: Atlas stores the expression the way
  PostgreSQL prints it, such as `(((science + mathematics) / 2))`, and a partial index's `where` comes
  back with casts such as `::text`. In SQL, wrap an index expression in parentheses.
- `INCLUDE` columns cannot be expressions.
- Exclusion constraints use an `exclude` block, with `type = GIST` for operators such as `&&`
  (`references/hcl.md`). PostgreSQL 18 temporal keys use `without_overlaps` on a primary key or unique
  constraint and `period` on a foreign key.
- `unlogged = true` on a table makes it unlogged; switching either way rewrites the table (`PG307`).

## Foreign Data

`server`, `foreign_table`, and `user_mapping` blocks manage foreign data wrappers, at database scope. A
`server` takes the wrapper (such as `extension.postgres_fdw`) and its `options`.

## Not Managed by Atlas

Atlas does not manage the following, and never plans changes to them. When the schema depends on one,
such as an operator class that an index uses, create it in the dev database's `baseline`
(`references/dev-database.md`):

- Publications and subscriptions
- Text search configurations
- Extended statistics (`CREATE STATISTICS`)
- Rules (`CREATE RULE`)
- Operators and operator classes
- Tablespaces
- A column's `STORAGE` and `COMPRESSION` settings
- Settings from `ALTER ROLE ... SET` and `ALTER DATABASE ... SET`
- Whether a trigger is enabled or disabled
