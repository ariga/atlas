# Testing Redshift Schemas and Migrations

Atlas runs tests on the dev database: `atlas schema test` checks the logic in the schema, and
`atlas migrate test` checks migration files against seeded data. The test syntax and every command are
in `../atlas/references/testing.md`; this file covers how to use them on Redshift. Testing needs
`atlas login`. If `atlas whoami` shows no login, ask the user to run `atlas login`; a user without an
account can start a free trial with it.

## Test Every Change

A schema or migration change is done when its tests pass:

1. After editing the schema, run `atlas schema validate --env <name>`.
2. Add or update a test case for what the change touches: a `test "schema"` case for a function,
   procedure, view, or materialized view, and a `test "migrate"` case for a migration that rebuilds a
   column, changes a type, or moves data.
3. Run the new case alone with `--run <case>`, and fix the logic until it passes.
4. Run the whole suite: `atlas schema test --env <name>` and `atlas migrate test --env <name>`.
5. Lint the change, so the analyzers and custom rules check it (`references/migrations.md`).
6. Report the cases you added and that they pass. If the tests could not run, say why.

The env lists the test files next to its dev URL (`references/dev-database.md`):

```hcl
env "dev" {
  src = "file://schema.rs.hcl"
  dev = getenv("REDSHIFT_DEV_URL")
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

## Why It Matters on Redshift

- The dev database is a real Redshift database, not a container. Tests run against it, so they see
  Redshift's real behavior: encodings, type limits, and the statements it refuses in a transaction.
  They also take real time and, on Serverless, real compute. Keep cases focused.
- Each case runs on the dev database, which Atlas restores afterwards. Do not share one dev database
  between test runs that can overlap.
- A test that creates users, groups, or roles creates them on the dev database's cluster, since
  principals are cluster-wide. Run such tests on a dev database with a cluster or workgroup of its own,
  and drop what the test creates in `cleanup`.
- Redshift does not enforce primary keys, foreign keys, or unique constraints, so tests cannot rely on
  them to reject rows. Check uniqueness with a query instead.

## Schema Tests

Each case starts from the desired schema, created on the dev database. What to cover on Redshift:

- SQL functions: normal inputs, edge cases, and `NULL` inputs
  (https://atlasgo.io/guides/testing/functions).
- Stored procedures: `CALL` them with seeded rows, and check the rows they change, and errors with
  `catch` (https://atlasgo.io/guides/testing/procedures).
- Views and late-binding views: seed the tables and compare the view's output
  (https://atlasgo.io/guides/testing/views).
- Materialized views: seed the tables, run `REFRESH MATERIALIZED VIEW`, and compare the output.

```hcl
test "schema" "add_numbers" {
  exec {
    sql    = "SELECT add_numbers(1, 2)"
    output = "3"
  }
}

test "schema" "daily_counts" {
  exec {
    sql = "INSERT INTO events (user_id, kind) VALUES (1, 'a'), (1, 'b'), (2, 'a')"
  }
  exec {
    sql = "REFRESH MATERIALIZED VIEW daily_counts"
  }
  exec {
    sql    = "SELECT n FROM daily_counts WHERE user_id = 1"
    output = "2"
  }
}
```

## Migration Tests

Each case starts from an empty database. Migrate to the version before the change, seed rows, migrate
to the version under test, and check the data (https://atlasgo.io/guides/testing/data-migrations). Here,
`20260102000000` is a hand-written change of `orders.amount` from `integer` to `bigint` (add a column,
copy, drop, rename):

```hcl
test "migrate" "widen_amount" {
  migrate {
    to = "20260101000000"
  }
  exec {
    sql = "INSERT INTO orders (id, amount) VALUES (1, 10), (2, 2147483647)"
  }
  migrate {
    to = "20260102000000"
  }
  exec {
    sql    = "SELECT max(amount) FROM orders"
    output = "2147483647"
  }
}
```

Seeded rows expose the Redshift failures that an empty database hides. Write a migration test for each
migration that:

- Rebuilds a column with `allow_recreate`: seed rows, migrate, and check what the column holds after.
  The rebuild drops the values, so the test shows whether the migration also needs a copy step.
- Changes a type by hand (add, copy, drop, rename): check that every value was copied.
- Runs under `-- atlas:txmode none`: a statement that fails on real data leaves the earlier ones
  applied, and the test catches it on the dev database first.
- Backfills or moves data: compare the rows after the migration.

Without a project file, pass the dev URL and the directory:

```bash
atlas migrate test --dev-url "$REDSHIFT_DEV_URL" --dir file://migrations .
```
