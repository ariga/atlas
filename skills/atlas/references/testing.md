# Testing Reference (`atlas schema test`, `atlas migrate test`, `atlas schema plan test`, `atlas script test`)

The Atlas testing framework runs test cases written in HCL against a dev database. The agent writes
the schema logic (functions, views, triggers, constraints, policies), data migrations, or scripts,
writes the tests next to them, and Atlas executes both and reports failures to fix. Testing requires
`atlas login`.

| Command | Test block | Starts from | Use for |
|---------|-----------|-------------|---------|
| `atlas schema test --env <name>` | `test "schema"` | The desired schema, created on the dev database | Functions, views, triggers, constraints, RLS, queries |
| `atlas migrate test --env <name>` | `test "migrate"` | An empty database, migrated to a chosen version | Data migrations and the migration files themselves |
| `atlas schema plan test --env <name>` | `test "plan"` | A given schema snapshot, then a plan file applied | Declarative plans before approval |
| `atlas script test --env <name>` | `test "script"` | A fresh database with no schema | Data Scripts in isolation (see `references/scripts.md`) |

All four accept `--run <regexp>` to select cases by name, `--var name=value` for input variables, and
optional paths to test files as trailing arguments.

## Configuration

Declare the test files on the env (or globally) so `--env` finds them:

```hcl
env "dev" {
  src = "file://schema.hcl"
  dev = "docker://postgres/17/dev?search_path=public"
  migration {
    dir = "file://migrations"
  }
  test {
    schema {
      src  = ["schema.test.hcl", "plan.test.hcl"]   # plan tests are declared here too
      vars = { seed_file = "seed.sql" }
    }
    migrate {
      src = ["migrate.test.hcl"]
    }
    script {
      src = ["scripts.test.hcl"]
    }
  }
}
```

Test files use the `.test.hcl` extension. `test "plan"` files are listed under `test.schema.src`,
or passed as an argument: `atlas schema plan test --env dev plan.test.hcl`. Each `test` block has two labels: the kind and the case name.
A case is a list of commands run in order; the first failing command aborts the case. The dev database
is reset between cases regardless of the result.

## Commands Available in Every Test Kind

| Command | Purpose |
|---------|---------|
| `exec { sql, format, output \| match }` | Runs SQL that must succeed. With `output` (exact, after trimming) or `match` (regexp) the result is compared, serialized as `csv` (default) or `table` |
| `catch { sql, error }` | Runs SQL that must fail; `error` matches the message |
| `assert { sql, error_message }` | SQL must return one row with one true value |
| `log { message }` | Prints a message to the test output |
| `external { program, working_dir, output \| match }` | Runs a program (a Go test, a seeding script) and checks its stdout |
| `script "exec" \| "query" \| "loop" { file, run, vars, output \| match \| error, as }` | Runs Data Scripts inside the case |
| `cleanup { sql }` | Runs after the case, whatever the result, in reverse order of definition |

Case-level arguments: `skip = <bool>` (can be an expression), `parallel = true` (schema tests only;
use for stateless cases), and `for_each` for table-driven cases.

## Schema Tests

Atlas creates the desired schema on the dev database, runs the case, and cleans up. On PostgreSQL a
template database can be cloned instead of rebuilt to keep setup fast for large schemas.

```hcl
# schema.test.hcl
test "schema" "postal_code_domain" {
  parallel = true
  exec {
    sql = "SELECT '12345'::us_postal_code"
  }
  catch {
    sql   = "SELECT 'hello'::us_postal_code"
    error = "invalid input value"
  }
}

test "schema" "order_total_trigger" {
  exec {
    sql = <<-SQL
      INSERT INTO orders (id, customer_id) VALUES (1, 10);
      INSERT INTO order_items (order_id, price, qty) VALUES (1, 5, 2), (1, 3, 1);
    SQL
  }
  assert {
    sql           = "SELECT total = 13 FROM orders WHERE id = 1"
    error_message = "trigger did not recompute orders.total"
  }
}

test "schema" "upper" {
  for_each = [
    { input = "hello", expected = "HELLO" },
    { input = "world", expected = "WORLD" },
  ]
  exec {
    sql    = "SELECT upper('${each.value.input}')"
    output = each.value.expected
  }
}
```

Input variables: declare `variable "name" { type = string }` in the test file, reference it as
`var.name`, and set it from `test { schema { vars = { ... } } }` in `atlas.hcl` or `--var`.

What to test in a schema: functions and procedures (inputs, edge cases, error paths with `catch`),
views (row counts and joins against seeded data), triggers (side effects after INSERT/UPDATE/DELETE),
check constraints and domains (`catch` on invalid values), row-level security (queries under a role),
and the application's own queries (`exec` with `output`).

## Migration Tests

Every case starts from the zero state of the migration directory. The pattern for testing a data
migration: migrate to the version before it, seed data, migrate to the tested version, assert.

```hcl
# migrate.test.hcl
test "migrate" "20260613061102_split_name" {
  migrate {
    to = "20260613061046"          # the version before the one under test
  }
  exec {
    sql = "INSERT INTO users (name) VALUES ('Ada Lovelace')"
  }
  migrate {
    to = "20260613061102"          # the version under test
  }
  exec {
    sql    = "SELECT first_name, last_name FROM users"
    output = "Ada,Lovelace"
  }
}
```

Migration tests cannot run in parallel. Use `skip` or `--run` to test only the latest migration during
development: `atlas migrate test --env dev --run 20260613061102`.

Write a migration test whenever a migration moves or transforms data (backfills, splits, merges,
type conversions), and whenever `migrate lint` reports a data-dependent change (`MF*`).

## Plan Tests

A declarative plan file records `From` and `To` fingerprints of the schema transition. A plan test
brings the dev database to the `From` state with a `schema` block, seeds data, applies the plan, and
asserts. The `schema` block state must match the plan's `From`, or the case fails.

```hcl
# plan.test.hcl
test "plan" "20260613061102" {
  schema {
    url = "file://snapshots/schema.v1.sql"   # or atlas://<repo>?tag=v1, or a data source
  }
  exec {
    sql = "INSERT INTO users (name) VALUES ('Ada Lovelace'), ('Grace Hopper')"
  }
  apply {
    url = "file://plans/20260613061102.plan.hcl"
  }
  exec {
    sql    = "SELECT first_name, last_name FROM users ORDER BY id"
    format = table
    output = <<-TAB
      first_name | last_name
      -----------+----------
       Ada       | Lovelace
       Grace     | Hopper
    TAB
  }
}
```

Run with `atlas schema plan test --env <name>` before approving the plan (`references/cloud.md`,
plan approvals).

## Script Tests

`test "script"` starts from a fresh database. Seed with `exec`, run the script with the `script`
command, and assert on what it prints or leaves behind. Details and the `as` block for privilege
testing are in `references/scripts.md`.

## Agent Workflow

1. Read `atlas.hcl` for the env's `test` block and the `dev` URL. Add the block if missing.
2. Write the logic (function, view, trigger, migration, or script) and a test file next to it.
3. Run the matching command with `--run` scoped to the new case:
   `atlas schema test --env dev --run order_total_trigger`.
4. Read the failure. The output shows the case, the command index, and the actual vs expected
   output or the SQL error.
5. Fix the logic (not the assertion) unless the assertion was wrong. Re-run.
6. Run the full suite once before reporting: `atlas schema test --env dev` and
   `atlas migrate test --env dev`.
7. Report which cases were added and that they pass.

## Error Handling

| Error | Action |
|-------|--------|
| `command requires 'atlas login'` or `available only to Atlas Pro users` | Run `atlas login`; report the skipped test run and continue |
| `no test files found` | Add `test { <kind> { src = [...] } }` to the env, or pass the file path as an argument |
| Output mismatch on whitespace | `output` is compared after trimming newlines only. Use `format = table` for aligned output, or `match` for a regexp |
| Plan test: `From` does not match | The `schema` block state differs from the plan's source state. Point `schema.url` at the exact pre-plan snapshot |
| Migration test hangs on a version | The `migrate { to }` version must exist in the directory; check `atlas migrate ls --env <name>` |

## Documentation

- [Schema testing](https://atlasgo.io/testing/schema)
- [Migration testing](https://atlasgo.io/testing/migrate)
- [Plan testing](https://atlasgo.io/testing/plan)
- [Data Scripts testing](https://atlasgo.io/scripts/testing)
- [Test block reference](https://atlasgo.io/hcl/testing)
- [Template databases for schema tests (PostgreSQL)](https://atlasgo.io/guides/postgres/schema-test-template-databases)
- [Testing guides: functions, views, triggers, procedures, domains, data migrations](https://atlasgo.io/guides/testing/functions)
