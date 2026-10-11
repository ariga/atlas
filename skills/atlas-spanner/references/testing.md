# Testing Spanner Schemas and Migrations

Atlas runs tests on the dev database. The test syntax, every command, and examples are in
`../atlas/references/testing.md`; this file covers how to use them on Spanner. Testing needs
`atlas login`. If `atlas whoami` shows no login, ask the user to run `atlas login`; a user without an
account can start a free trial with it.

## Use Migration Tests

`atlas schema test` does not run on Spanner: it creates the schema on the dev database inside a
transaction, and every case fails during setup:

```
-- FAIL: balance_check (0µs)
    schema.test.hcl:1:
        	Error:      	unexpected error during test setup: spanner: code = "FailedPrecondition", desc = "cannot execute DDL as part of a transaction"
```

`atlas migrate test` works. Write every test as a `test "migrate"` case: migrate to a version, seed rows,
and check the result. The test syntax is the same as in schema tests, plus a `migrate` block.

## Test Every Change

A schema or migration change is done when its tests pass:

1. After editing the schema, run `atlas schema validate --env <name>`.
2. Lint the change, so the analyzers and custom rules check it, and so a statement Spanner rejects
   fails here instead of in production: `atlas migrate lint --env <name> --latest 1`, or in the
   declarative workflow the plan of `atlas schema apply --env <name> --dry-run`
   (`references/migrations.md`).
3. Add or update a `test "migrate"` case for what the change touches: a check, a unique or
   `NULL_FILTERED` index, an interleaved delete, a foreign key, a generated column, a default, a
   sequence, or a view, and every migration that moves, converts, or backfills data.
4. Run the new case alone with `--run <case>`, and fix the logic until it passes.
5. Run the whole suite: `atlas migrate test --env <name>`.
6. Report the cases you added and that they pass. If the tests could not run, say why.

`migrate test` also catches a hand-written file without `-- atlas:txmode none`, which `migrate lint` and
`migrate validate` accept and `migrate apply` then fails on (`references/migrations.md`).

The env lists the test files next to its dev database (`references/dev-database.md`):

```hcl
env "dev" {
  dev = "docker://spanner/1.5.58"
  migration {
    dir = "file://migrations"
  }
  test {
    migrate {
      src = ["migrate.test.hcl"]
    }
  }
}
```

## Declarative Projects

A project without a migration directory tests its schema through a throwaway one: generate a single
migration from the schema, and run the cases against it. The test file takes the version as a variable:

```hcl
variable "version" {
  type    = string
  default = ""
}

env "dev" {
  src = "file://schema.sql"
  dev = "docker://spanner/1.5.58"
  test {
    migrate {
      src = ["schema.test.hcl"]
      vars = {
        version = var.version
      }
    }
  }
}
```

```hcl
# schema.test.hcl
variable "version" {
  type = string
}

test "migrate" "balance_check" {
  migrate {
    to = var.version
  }
  catch {
    sql   = "INSERT INTO Customers (CustomerId, Email, Balance) VALUES ('c1', 'a@x.io', -1)"
    error = "BalanceNonNegative"
  }
}
```

```bash
dir=$(mktemp -d)
atlas migrate diff --env dev --dir "file://$dir" snapshot
version=$(ls "$dir" | grep _snapshot.sql | cut -d_ -f1)
atlas migrate test --env dev --dir "file://$dir" --var version="$version"
rm -rf "$dir"
```

`--var` sets a variable of `atlas.hcl`, not of the test file: the test file gets it through `vars` in
the env's `test` block, as above. Without that, the run fails with
`missing value for required variable "version"`.

## Why It Matters More on Spanner

- Spanner runs each DDL statement as its own schema change, so a file or a plan that fails halfway
  stays half applied (`references/migrations.md`). A migration test fails on the dev database first.
- Constraints that Spanner checks against existing rows, such as `NOT NULL`, a shorter `STRING(n)`, a
  unique index, a foreign key, or a check, pass on an empty database. Only seeded rows show the failure.
- The emulator is not production. It does not enforce mutation limits or quotas, and it accepts some
  SQL that production rejects
  (https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/master/README.md). A data
  migration that passes its test can still exceed the 80,000 mutations one production DML statement
  allows (`references/migrations.md`, Data Changes).
- The emulator runs one read-write transaction or schema change at a time, so tests on one emulator
  run one after another.

## What to Write

Each case starts from an empty database. Migrate to the version before the change, seed rows, migrate to
the version under test, and check the data (https://atlasgo.io/guides/testing/data-migrations):

```hcl
test "migrate" "email_not_null" {
  migrate {
    to = "20261001000001"
  }
  exec {
    sql = "INSERT INTO Users (Id, Email) VALUES (1, NULL), (2, 'b@x.io')"
  }
  migrate {
    to = "20261001000002"
  }
  exec {
    sql    = "SELECT COUNT(*) FROM Users WHERE Email IS NULL"
    output = "0"
  }
}
```

Here version 2 backfills `Email` before it makes the column `NOT NULL`. Without the backfill, the test
fails where production would.

Write a migration test for each migration that:

- Makes a column `NOT NULL` (`MF104`): seed `NULL` rows.
- Shortens a `STRING(n)` or `BYTES(n)`: seed a value longer than the new length. Lint does not report
  this change.
- Adds a unique index (`MF101`): seed duplicates, and two rows with `NULL`. A plain `UNIQUE` index
  treats `NULL`s as equal and rejects the second row; a `UNIQUE NULL_FILTERED` index skips `NULL` rows
  (`references/objects.md`, Indexes).
- Adds a foreign key or a check: seed rows that break it.
- Backfills or moves data, such as the add, copy, and drop that replaces a type change: compare the rows
  after the migration.

And cases that exercise the schema:

```hcl
test "migrate" "interleaved_cascade" {
  migrate {
    to = "20261001000002"
  }
  exec {
    sql = "INSERT INTO Customers (CustomerId, Email) VALUES ('c1', 'a@x.io')"
  }
  exec {
    sql = "INSERT INTO Orders (CustomerId, Total) VALUES ('c1', 10), ('c1', 5)"
  }
  exec {
    sql    = "SELECT CAST(Total AS STRING) FROM CustomerTotals WHERE CustomerId = 'c1'"
    output = "15"
  }
  exec {
    sql = "DELETE FROM Customers WHERE CustomerId = 'c1'"
  }
  exec {
    sql    = "SELECT COUNT(*) FROM Orders"
    output = "0"
  }
}

test "migrate" "sequence_keys_are_spread" {
  migrate {
    to = "20261001000002"
  }
  exec {
    sql = "INSERT INTO Customers (CustomerId, Email) VALUES ('c1', 'a@x.io')"
  }
  exec {
    sql = "INSERT INTO Orders (CustomerId, Total) VALUES ('c1', 1), ('c1', 2)"
  }
  exec {
    sql    = "SELECT COUNTIF(OrderId > 1000000) FROM Orders"
    output = "2"
  }
}
```

A bit-reversed sequence hands out large, scattered values, not 1, 2, 3, so assert ranges or uniqueness,
never exact ids.

## Matching Errors

`catch` matches the error text of a statement that writes, such as an `INSERT`, `UPDATE`, or `DELETE`.
It does not see a failing `SELECT`: Spanner reports a query error only when the rows are read, so
`catch` fails with `an error is expected, but none occurred` even for a column that does not exist.
Check what a migration removed with `exec` and `output` over `INFORMATION_SCHEMA`:

```hcl
  exec {
    sql    = "SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = 'Users' AND COLUMN_NAME = 'Nick'"
    output = "0"
  }
```

Match the constraint or index name, or a stable phrase, not the whole message. The emulator reports:

| Violation | Error contains |
|-----------|----------------|
| Check constraint | ``Check constraint `P`.`VPositive` is violated for key {Int64(1)}`` |
| `NOT NULL` column left out of an `INSERT` | `does not specify a non-null value for NOT NULL column: Name` |
| `NOT NULL` column set to `NULL` | `Cannot specify a null value for column: Users.Status` |
| Unique index | `UNIQUE violation on index PByName` |
| Duplicate primary key | `due to previously existing row` |
| Interleaved child without a parent row | `Insert failed because key was not found in parent table` |
| Foreign key | ``Foreign key `FK_FP` constraint violation`` |
| Value longer than `STRING(n)` | `New value exceeds the maximum size limit for this column` |

Production words some errors differently, so name constraints and assert on the name.

## CI

Run tests where the extended build of Atlas is installed, with the install script and
`ATLAS_FLAVOR="spanner"`, or from the `arigaio/atlas:latest-extended` image. A `docker://spanner` dev URL
needs a Docker daemon. Without one, run the emulator as a service container,
`gcr.io/cloud-spanner-emulator/emulator:1.5.58` on ports 9010 and 9020, create an instance and an empty
database in it, and point `dev` at that database (`references/dev-database.md`, Connection URLs). Pin
the same emulator release locally and in CI.
