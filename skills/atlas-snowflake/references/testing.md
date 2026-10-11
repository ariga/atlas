# Testing Snowflake Schemas and Migrations

Atlas runs tests on the dev database: `atlas schema test` checks the logic in the schema, and
`atlas migrate test` checks migration files against seeded data. The test syntax, every command, and
examples are in `../atlas/references/testing.md`; this file covers how to use them on Snowflake.
Testing needs `atlas login`. If `atlas whoami` shows no login, ask the user to run `atlas login`; a
user without an account can start a free trial with it.

## Test Every Change

A schema or migration change is done when its tests pass:

1. After editing the schema, run `atlas schema validate --env <name>`.
2. Add or update a test case for what the change touches: a `test "schema"` case for a function,
   procedure, view, dynamic table, or policy, and a `test "migrate"` case for a migration that moves or
   converts data.
3. Run the new case alone with `--run <case>`, and fix the logic until it passes.
4. Run the whole suite: `atlas schema test --env <name>` and `atlas migrate test --env <name>`.
5. Lint the change (`references/migrations.md`).
6. Report the cases you added and that they pass. If the tests could not run, say why.

The env lists the test files next to its dev URL (`references/dev-database.md`):

```hcl
env "dev" {
  src = "file://schema.sf.hcl"
  dev = "snowflake://${getenv("SNOWFLAKE_USER")}:${urlescape(getenv("SNOWFLAKE_TOKEN"))}@${getenv("SNOWFLAKE_ACCOUNT")}/ANALYTICS_DEV/PUBLIC?warehouse=${getenv("SNOWFLAKE_WAREHOUSE")}"
  test {
    schema {
      src = ["schema.test.hcl"]
    }
    migrate {
      src = ["migrate.test.hcl"]
    }
  }
}
```

## Why It Matters More on Snowflake

- Snowflake commits DDL as it goes, so a migration file that fails halfway stays half applied on the
  target (`references/migrations.md`). A migration test runs the files on the dev database first, where a
  failing statement fails the test instead.
- A plan with `allow_recreate = true` drops and re-adds a column, losing its values
  (`references/objects.md`). A migration test that seeds the column and checks it after the migration
  shows whether it also needs a copy step.
- Snowflake enforces only `NOT NULL` on regular tables; primary keys, unique constraints, and foreign
  keys are enforced on hybrid tables only. A test that expects a duplicate key to fail on a regular table
  fails.
- Every case runs on the dev database on the account and restores it afterward, and Snowflake DDL is
  slow. Keep suites focused, and select cases with `--run` while iterating.
- Tests run as the role in the dev URL. Policies that depend on `CURRENT_ROLE()` return what that role
  sees. A `script` command (Data Scripts) can run as another principal only through a separate
  login: an `as` block with `user` and `password`, which keeps the dev URL's other parameters,
  `role=` included. `as { role = "..." }` fails on Snowflake with
  `sql/sqlclient: the "snowflake" driver does not support role switching`. For other checks, run
  the query from an `external` program that connects with that role's credentials.

## Schema Tests

Each case starts from the desired schema, created on the dev database. Names in SQL follow Snowflake's
case rules: quote lowercase names, and expect uppercase column names in output.

```hcl
test "schema" "add" {
  exec {
    sql    = "SELECT \"ADD\"(1, 2)"
    output = "3"
  }
  exec {
    sql    = "SELECT \"ADD\"(1)"
    output = "11"
  }
}

test "schema" "events_not_null" {
  catch {
    sql   = "INSERT INTO \"BRONZE\".\"EVENTS\" (\"EVENT_ID\") VALUES (NULL)"
    error = "NULL result in a non-nullable column"
  }
}
```

What to cover on Snowflake:

- SQL, JavaScript, and Python functions: normal inputs, edge cases, overloads, argument defaults, and
  `NULL` handling under `null_call` (https://atlasgo.io/guides/testing/functions).
- Procedures: `CALL` them, check the result and the rows they change
  (https://atlasgo.io/guides/testing/procedures).
- Views and secure views: seed rows and compare the output (https://atlasgo.io/guides/testing/views).
- Dynamic tables: seed the base tables, run `ALTER DYNAMIC TABLE "X" REFRESH`, and compare the rows. A
  refresh needs a warehouse, the dynamic table's `warehouse` or the dev URL's.
- Masking and row access policies: query the protected table as the dev role and compare what it returns.

## Migration Tests

Each case starts from an empty database. Migrate to the version before the change, seed rows, migrate to
the version under test, and check the data (https://atlasgo.io/guides/testing/data-migrations):

```hcl
test "migrate" "amount_cents" {
  migrate {
    to = "20261001000000"
  }
  exec {
    sql = "INSERT INTO \"ORDERS\" (\"ID\", \"AMOUNT\") VALUES (1, 9.5), (2, 12)"
  }
  migrate {
    to = "20261002000000"
  }
  exec {
    sql    = "SELECT \"ID\", \"AMOUNT_CENTS\" FROM \"ORDERS\" ORDER BY \"ID\""
    output = "1,950\n2,1200"
  }
}
```

Write a migration test for each migration that:

- Recreates a column by hand, or backfills a new one: seed values and compare them after.
- Changes a column type in place, such as a longer `VARCHAR` or a wider `NUMBER`.
- Recreates a table kind that holds data, such as an Iceberg table, or rebuilds a table to change
  `transient`: count the rows before and after.
- Makes a column `NOT NULL`: seed a `NULL` and check the migration's handling of it.

Without a project file, pass the dev URL and the directory
(https://atlasgo.io/cli-reference#atlas-migrate-test):

```bash
atlas migrate test --dev-url "snowflake://user:pass@account/ANALYTICS_DEV/PUBLIC?warehouse=WH" --dir file://migrations .
```
