# Testing ClickHouse Schemas and Migrations

Atlas runs tests on the dev database: `atlas schema test` checks the logic in the schema, and
`atlas migrate test` checks migration files against seeded data. The test syntax, every command, and
examples are in `../atlas/references/testing.md`; this file covers how to use them on ClickHouse.
Testing needs `atlas login`, as the ClickHouse driver does. If `atlas whoami` shows no login, ask the
user to run `atlas login`; a user without an account can start a free trial with it.

## Test Every Change

A schema or migration change is done when its tests pass:

1. After editing the schema, run `atlas schema validate --env <name>`.
2. Check that the plan holds every edit: Atlas leaves destructive rebuilds out of the plan without a
   terminal (`references/migrations.md`, Changes Skipped Without a Terminal).
3. Add or update a test case for what the change touches: a `test "schema"` case for a materialized
   view, dictionary, UDF, check, default expression, or row policy, and a `test "migrate"` case for a
   migration that rebuilds a table, converts a column, or moves data.
4. Run the new case alone with `--run <case>`, and fix the logic until it passes.
5. Run the whole suite: `atlas schema test --env <name>` and `atlas migrate test --env <name>`.
6. Lint the change, so the `CH3*` checks and custom rules see it (`references/migrations.md`).
7. Report the cases you added and that they pass. If the tests could not run, say why.

The env lists the test files next to its ClickHouse dev URL (`references/dev-database.md`):

```hcl
env "dev" {
  src = "file://schema.ch.hcl"
  dev = "docker://clickhouse/24.10/app"
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

- At database scope, the dev database takes the name of the schema's `schema` block, here `app`.
  Another name fails every schema test with `mismatched schema names: "dev" != "app"`.
- UDFs, users, roles, and other server-level objects are left out of a database-scoped desired state
  without an error, so a test that calls a UDF fails with ``Function with name `plus_one` does not
  exist``. Test them with a server-scoped dev URL, `docker://clickhouse/24.10`, and qualify every table
  in the test statements: `app.events`.

Without a project file, pass the dev URL and the directory:
`atlas migrate test --dev-url docker://clickhouse/24.10/dev --dir file://migrations .`

## Seed Rows with a Script

On ClickHouse, an `exec` command in a test can only read: an `INSERT` or a DDL statement in `exec`
fails with `driver: bad connection`, and can run twice before it fails. Assertions with `SELECT` work.
Seed rows with a `script` command instead, which runs the `exec` scripts of a script file
(`../atlas/references/scripts.md`):

```hcl
# seed.hcl
script "exec" "seed_events" {
  exec {
    sql = "INSERT INTO events (tenant_id, event_date, name) SELECT 1, today(), toString(number) FROM numbers(1000)"
  }
}
```

In the test, `file` is relative to the test file, and `run` picks the script by name:

```hcl
script "exec" {
  file = "seed.hcl"
  run  = "seed_events"
}
```

A `catch` command runs its statement and expects it to fail, so `catch` works for a violating
`INSERT`.

## Why It Matters More on ClickHouse

- ClickHouse commits each statement on its own, so a file that fails halfway stays half applied
  (`references/migrations.md`, Transactions). A migration test applies the files to the dev database
  first, where a failing statement fails the test instead of production.
- Rebuilds copy rows with `INSERT ... SELECT`, and column conversions run as mutations. An empty database
  hides a value that does not fit the new type, or a column the copy leaves out.
- Tests see the dev database's behavior. Run it on the target's version and settings, and with the
  server configuration the schema needs, such as WASM support or `named_collection_control`, mounted
  through `volumes` in a `docker` block.
- A test against a clustered schema needs a clustered dev database.

## ClickHouse Behavior in Tests

- Mutations run in the background. A hand-written `ALTER TABLE ... UPDATE` or `DELETE` in a test script
  or a migration needs `SETTINGS mutations_sync = 1` before an assertion can see its result.
- `ReplacingMergeTree`, `SummingMergeTree`, and other merging engines combine rows only when parts merge.
  Assert on the final state with `FINAL`, or run `OPTIMIZE TABLE ... FINAL` in a script first.
- A materialized view processes the rows inserted after it exists. Seed the source table with a script,
  after the schema is created, and read the target table or the view.
- A dictionary loads its source on first use. Insert the source rows and run
  `SYSTEM RELOAD DICTIONARY <name>` in one script, then call `dictGet` in an `exec`.
- The dev database connects as `default`, which row policies may not cover. Test a policy as a user the
  policy names.

## Schema Tests

Each case starts from the desired schema, created on the dev database. With `seed.hcl` holding a
`script "exec" "two_events"` that inserts two rows on `2026-01-01`:

```hcl
test "schema" "daily_counts" {
  script "exec" {
    file = "seed.hcl"
    run  = "two_events"
  }
  exec {
    sql    = "SELECT c FROM events_daily WHERE event_date = '2026-01-01'"
    output = "2"
  }
}

# UDFs need a server-scoped dev URL (Test Every Change, above).
test "schema" "plus_one" {
  exec {
    sql    = "SELECT plus_one(41)"
    output = "42"
  }
}

test "schema" "valid_tenant" {
  catch {
    sql   = "INSERT INTO events (tenant_id, event_date, name) VALUES (0, '2026-01-01', 'a')"
    error = "valid_tenant"
  }
}
```

What to cover: materialized views (seed the source, check the target), dictionaries (`dictGet` after a
reload), UDFs (inputs and edge cases), `CHECK` constraints (`catch` a violating row), `MATERIALIZED` and
`ALIAS` columns (insert rows and compare the computed values), and the queries the application runs.

## Migration Tests

Each case starts from an empty database. Migrate to the version before the change, seed rows, migrate to
the version under test, and check the data (https://atlasgo.io/testing/migrate):

```hcl
test "migrate" "20260601120000_resort_events" {
  migrate {
    to = "20260601110000"
  }
  script "exec" {
    file = "seed.hcl"
    run  = "seed_events"
  }
  migrate {
    to = "20260601120000"
  }
  exec {
    sql    = "SELECT count() FROM events"
    output = "1000"
  }
}
```

Write a migration test for each migration that:

- Rebuilds a table (`CH302`): seed rows, migrate, and check the row count and the copied columns.
- Changes a column's type (`CH301`), or makes a `Nullable` column non-nullable (`MF104`): seed values at
  the edges of the new type, and `NULL` rows.
- Changes `Enum` values, or a hand-written column recreation: seed every value and check none was lost.
- Backfills or moves data with a mutation: compare the rows after the migration.
