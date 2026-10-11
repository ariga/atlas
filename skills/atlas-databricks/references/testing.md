# Testing Databricks Schemas and Migrations

Atlas runs tests on the dev database. On Databricks only `atlas migrate test` runs: `atlas schema test`
fails on every case, because its setup opens a transaction on the dev catalog and the Databricks
driver has none. The test syntax and every command are in `../atlas/references/testing.md`; this file
covers how to use them on Databricks. Testing needs `atlas login`. If `atlas whoami` shows no login,
ask the user to run `atlas login`; a user without an account can start a free trial with it.

## Test Every Change

A schema or migration change is done when its tests pass:

1. After editing the schema, run `atlas schema validate --env <name>`.
2. Generate the migration, add `-- atlas:txmode none` and a blank line at the top of the file, and run
   `atlas migrate hash`. A file without the header fails under `migrate test` with
   `sql/sqlclient: starting transaction: databricks: driver error: not implemented`.
3. Add or update a `test "migrate"` case for what the change touches: a function, procedure, mask, row
   filter, view, materialized view, check, or generated column, and seeded rows for a migration that
   moves, converts, or drops data.
4. Run the new case alone with `--run <case>`, and fix the logic until it passes.
5. Run the whole suite: `atlas migrate test --env <name>`.
6. Lint the change, so the analyzers and custom rules check it (`references/migrations.md`).
7. Report the cases you added and that they pass. If the tests could not run, say why.

The env lists the test files next to its dev catalog (`references/dev-database.md`):

```hcl
env "dev" {
  src = "file://schema.dbx.hcl"
  dev = "${local.dbx}?catalog=atlas_dev"
  migration {
    dir     = "file://migrations"
    tx_mode = "none"
  }
  test {
    migrate {
      src = ["migrate.test.hcl"]
    }
  }
}
```

## Why It Matters More on Databricks

- Databricks commits DDL as it goes, so a migration file that fails halfway stays half applied
  (`references/migrations.md`, Transactions). A migration test runs the files on the dev catalog first,
  where a failing statement fails the test instead of production.
- Lint runs no data-dependent checks on Databricks. A `SET NOT NULL` over rows that hold `NULL`, or a
  type change existing values do not fit, shows up only when the migration runs on data. Seeded rows in
  a migration test are the only check before production.
- The dev catalog is the real engine, reached through the warehouse. Tests see Databricks' own
  behavior, and each statement takes a round trip, so a suite takes minutes. Keep cases focused.
- Every case runs as the principal of the dev URL's token. Functions that read the session, such as
  `current_user()` and `is_account_group_member()`, return what that principal gets.

## Schema Tests Do Not Run

`atlas schema test` creates the desired schema on the dev catalog inside a transaction, and the
Databricks driver rejects the transaction before the first statement, so every case reports
`unexpected error during test setup: sql/sqlclient: starting transaction: databricks: driver error:
not implemented`. There is no flag that changes this. Put what a schema test would check into a
migration test instead: migrate to the version that creates the object, then query it.

## Migration Tests

Each case starts from an empty dev catalog. Migrate to a version, seed rows, migrate further, and check
the data (https://atlasgo.io/guides/testing/data-migrations). The same shape covers the logic in the
schema, with the version that creates the object as the target:

```hcl
test "migrate" "add_numbers" {
  migrate {
    to = "20260101000000"
  }
  for_each = [
    { x = 1, y = 2, expected = "3" },
    { x = 5, y = 10, expected = "15" },
  ]
  exec {
    sql    = "SELECT `default`.`add_numbers`(${each.value.x}, ${each.value.y})"
    output = each.value.expected
  }
}

test "migrate" "positive_amount" {
  migrate {
    to = "20260101000000"
  }
  exec {
    sql = "INSERT INTO `default`.`orders` (order_id, customer_id, amount) VALUES (1, 1, 10.00)"
  }
  catch {
    sql   = "INSERT INTO `default`.`orders` (order_id, customer_id, amount) VALUES (2, 1, -1.00)"
    error = "positive_amount"
  }
}

test "migrate" "rename_email" {
  migrate {
    to = "20260101000000"
  }
  exec {
    sql = "INSERT INTO `default`.`users` (id, email) VALUES (1, 'a@example.com')"
  }
  migrate {
    to = "20260102000000"
  }
  exec {
    sql    = "SELECT customer_email FROM `default`.`users` WHERE id = 1"
    output = "a@example.com"
  }
}
```

- `rename_email` covers a hand-written rename, version `20260102000000`: Atlas plans no renames on
  Databricks (`references/objects.md`, Renames).
- Qualify names with their schema in catalog-scoped test SQL, as the session's current schema may not
  be the one under test.
- Keep one statement per `exec` block.
- SQL and Python functions: normal inputs, edge cases, and `NULL`. Atlas does not check a Python body,
  so a test is what runs it.
- Procedures: call them with `CALL`, and check their side effects.
- Checks: `catch` a row that breaks the check.
- Generated and identity columns: insert rows and compare the computed values.
- Views: seed rows and compare the view's output. A materialized view takes longer to create than a
  view, so keep those cases few.
- Masks and row filters: query as the dev principal, and assert what that principal should see.

Write a migration test for each migration that:

- Sets `NOT NULL` on a column, or adds a `NOT NULL` column (added plain, then altered): seed `NULL` rows
  and check the migration's handling of them.
- Changes a column's type: seed values at the edges of the old type and check them after the change,
  or that the migration fails before production does.
- Renames a column, in a hand-written file (`references/objects.md`, Renames), or drops one: seed rows
  and check that the renamed column keeps its data, as `rename_email` above does.
- Recreates a table to change its partitioning, provider, or a generated or identity column: compare the
  row counts and values of the old and the new table.
- Backfills or moves data: compare the rows after the migration.

## CI

Run the tests in CI with the same env, the token in a secret, and `--env`:

```bash
atlas migrate test --env dev
```

CI runs share the dev catalog, and Atlas takes no lock on Databricks. Run one test job at a time per
dev catalog, or give each job its own catalog.
