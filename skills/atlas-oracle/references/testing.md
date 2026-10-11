# Testing Oracle Schemas and Migrations

Atlas runs tests on the dev database: `atlas schema test` checks the logic in the schema, and
`atlas migrate test` checks migration files against seeded data. The test syntax, every command, and
examples are in `../atlas/references/testing.md`; this file covers how to use them on Oracle. Testing
needs `atlas login`. If `atlas whoami` shows no login, ask the user to run `atlas login`; a user without
an account can start a free trial with it.

## Test Every Change

A schema or migration change is done when its tests pass:

1. After editing the schema, run `atlas schema validate --env <name>`.
2. Add or update a test case for what the change touches: a `test "schema"` case for a function,
   procedure, trigger, view, virtual column, or check constraint, and a `test "migrate"` case for a
   migration that moves or converts data, or that creates PL/SQL.
3. Run the new case alone with `--run <case>`, and fix the logic until it passes.
4. Run the whole suite: `atlas schema test --env <name>` and `atlas migrate test --env <name>`.
5. Lint the change, so the analyzers and custom rules check it (`references/migrations.md`).
6. Report the cases you added and that they pass. If the tests could not run, say why.

The env lists the test files next to its dev database (`references/dev-database.md`):

```hcl
env "dev" {
  src = "file://schema.sql"
  dev = docker.oracle.dev.url
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

## Why It Matters More on Oracle

- Atlas reports PL/SQL that does not compile as applied in migration files, and leaves it out of plans
  without an error (`references/objects.md`). Only a test that calls the code, or counts invalid objects,
  finds it before a deployment.
- Oracle commits DDL as it goes, so a migration file that fails halfway stays half applied
  (`references/migrations.md`). A migration test fails on the dev database first.
- `''` is `NULL`, and identity, sequence, and number behaviors differ from other databases. Seeded rows
  show what an empty database hides.
- Tests see the dev database's behavior. Run it on the target's release and `MAX_STRING_SIZE`, or a test
  can pass on the dev database and fail on the target.
- A `docker://` dev URL starts a container for each run, in 20 to 50 seconds. For quick test loops, keep
  one Oracle container running and point `dev` at an empty user in it, which runs a schema test suite in
  seconds (`references/dev-database.md`, Your Own Dev Database).

## Schema Tests

Each case starts from the desired schema, created on the dev database:

```hcl
test "schema" "add_tax" {
  exec {
    sql    = "SELECT add_tax(100) FROM dual"
    output = "117"
  }
}

test "schema" "balance_check" {
  catch {
    sql   = "INSERT INTO accounts (id, balance) VALUES (1, -5)"
    error = "ORA-02290"
  }
}

test "schema" "credit_and_touch" {
  exec {
    sql = "INSERT INTO accounts (id, balance) VALUES (1, 10)"
  }
  exec {
    sql = "CALL credit_account(1, 5)"
  }
  exec {
    sql    = "SELECT balance FROM accounts WHERE id = 1"
    output = "15"
  }
  exec {
    sql    = "SELECT COUNT(*) FROM accounts WHERE updated_at IS NOT NULL"
    output = "1"
  }
}
```

- With an HCL schema, the dev database's schema is `PDBADMIN`, and a schema named after the target user
  fails with `mismatched schema names: "PDBADMIN" != "APP"`. Name the schema from a variable that
  defaults to the target user, and run `atlas schema test --env <name> --var schema_name=PDBADMIN`
  (`references/hcl.md`, Files and Schemas).
- Call functions from `DUAL`, and procedures with `CALL`. An anonymous block in `exec`, such as
  `BEGIN credit_account(1, 1); END;`, fails with `PLS-00103`, as it does in migration files
  (`references/migrations.md`).
- `catch` matches the error text, so match the code: `ORA-02290` for a check, `ORA-00001` for a unique
  key, `ORA-01400` for `NULL` in a `NOT NULL` column, `ORA-02291` for a missing parent row, and the
  `ORA-20000` to `ORA-20999` codes that `RAISE_APPLICATION_ERROR` uses in triggers.
- Call every function and procedure at least once. One that did not compile is missing from the dev
  database, and the call fails with `ORA-00904: "NO_SUCH_FN": invalid identifier`.
- Triggers: seed rows, make the change that fires the trigger, and check the side effect, as above
  (https://atlasgo.io/guides/testing/triggers). A row trigger that reads `:NEW` needs `for = ROW`.
- Views and virtual columns: seed rows and compare the output
  (https://atlasgo.io/guides/testing/views).
- Functions and procedures: normal inputs, edge cases, and error paths with `catch`
  (https://atlasgo.io/guides/testing/functions, https://atlasgo.io/guides/testing/procedures).

## Migration Tests

Each case starts from an empty database. Migrate to the version before the change, seed rows, migrate
to the version under test, and check the data (https://atlasgo.io/guides/testing/data-migrations):

```hcl
test "migrate" "nick_not_null" {
  migrate {
    to = "20261001000001"
  }
  exec {
    sql = "INSERT INTO customers (id, nick) VALUES (1, NULL)"
  }
  migrate {
    to = "20261001000002"
  }
  exec {
    sql    = "SELECT COUNT(*) FROM customers WHERE nick IS NULL"
    output = "0"
  }
}
```

Here the migration backfills with `UPDATE customers SET nick = '' WHERE nick IS NULL`, then sets
`NOT NULL`. On Oracle `''` is `NULL`, so the rows stay `NULL`, and the test fails where production would:
`ORA-02296: cannot enable (...) - null values found`.

Write a migration test for each migration that:

- Creates or replaces PL/SQL: migrate to the version, and check that nothing is invalid. Oracle creates
  code that does not compile as `INVALID`, and Atlas reports the file as applied:

  ```hcl
  test "migrate" "plsql_compiles" {
    migrate {
      to = "20261001000002"
    }
    exec {
      sql    = "SELECT COUNT(*) FROM user_objects WHERE status = 'INVALID'"
      output = "0"
    }
  }
  ```

  When it fails, `SELECT name, type, line, text FROM user_errors` shows the compilation errors.
- Makes a column `NOT NULL` (`MF104`), or adds a `NOT NULL` column without a default (`MF103`): seed
  rows with `NULL` and with `''`.
- Adds a unique constraint or index on existing data (`MF101`): seed duplicates.
- Changes a column's type, which needs an empty column (`ORA-01439`), or backfills or moves data:
  compare the rows after the migration.
- Copies rows into a `GENERATED ALWAYS` identity column, which rejects them with `ORA-32795`.

## CI

Run tests where the Oracle build of Atlas is installed, with the install script and
`ATLAS_FLAVOR="oracle"`, or from the `arigaio/atlas:latest-extended` image. Without Docker for
`docker://` URLs, run the Free image as a service container and point `dev` at an empty user in it, on
the same release as the target. Each Free container needs about 4 GB of disk.

A dev user you provide keeps what Oracle's recycle bin holds after each run. Purge it from time to time
as that user with `PURGE RECYCLEBIN`, or turn the recycle bin off for the dev database:
`ALTER SYSTEM SET recyclebin = OFF DEFERRED`.
