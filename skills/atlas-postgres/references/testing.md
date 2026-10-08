# Testing PostgreSQL Schemas and Migrations

Atlas runs tests on the dev database: `atlas schema test` checks the logic in the schema, and
`atlas migrate test` checks migration files against seeded data. The test syntax and every command are
in `../atlas/references/testing.md`; this file covers how to use them on PostgreSQL. Testing needs
`atlas login`. If `atlas whoami` shows no login, ask the user to run `atlas login`; a user without an
account can start a free trial with it.

## Test Every Change

A schema or migration change is done when its tests pass:

1. After editing the schema, run `atlas schema validate --env <name>`.
2. Add or update a test case for what the change touches: a `test "schema"` case for a function,
   trigger, view, domain, check constraint, or policy, and a `test "migrate"` case for a migration that
   moves or converts data.
3. Run the new case alone with `--run <case>`, and fix the logic until it passes.
4. Run the whole suite: `atlas schema test --env <name>` and `atlas migrate test --env <name>`.
5. Lint the change, so the PostgreSQL analyzers and custom rules check it (`references/migrations.md`).
6. Report the cases you added and that they pass. If the tests could not run, say why.

The env lists the test files (https://atlasgo.io/guides/testing/triggers):

```hcl
env "dev" {
  src = "file://schema.hcl"
  dev = "docker://postgres/15/dev?search_path=public"
  # Test configuration for local development.
  test {
    schema {
      src = ["schema.test.hcl"]
    }
  }
}
```

Migration tests are listed the same way: `test { migrate { src = ["migrate.test.hcl"] } }`.

## Schema Tests

Each case starts from the desired schema, created on the dev database. A trigger test seeds rows,
makes the change that fires the trigger, and checks the side effect
(https://atlasgo.io/guides/testing/triggers):

```hcl
test "schema" "trigger" {
  # Seed data
  exec {
    sql = "INSERT INTO products (id, price) VALUES (1, 15.00), (2, 22.99), (3, 13.50);"
  }
  # Verify products_audit table empty
  exec {
    sql = "SELECT COUNT(*) FROM products_audit"
    output = "0"
  }
  exec {
    sql = "UPDATE products SET price = 19.99 WHERE id = 1;"
  }
  exec {
    sql = "SELECT product_id, old_price, new_price FROM products_audit"
    format = table
    output = <<TAB
 product_id | old_price | new_price
------------+-----------+-----------
 1          | 15        | 19.99
TAB
  }
}
```

What to cover on PostgreSQL:

- PL/pgSQL functions and procedures: normal inputs, edge cases, and error paths with `catch`
  (https://atlasgo.io/guides/testing/functions, https://atlasgo.io/guides/testing/procedures).
- Triggers: seed rows, change them, and check the side effects, as above. A trigger declared without
  `for = ROW` runs once per statement, where `NEW` and `OLD` are null; a test catches it.
- Domains and check constraints: `catch` an invalid value (https://atlasgo.io/guides/testing/domains).
- Views: seed rows and compare the view's output (https://atlasgo.io/guides/testing/views).
- Row-level security policies: test them as a non-privileged role, below.

### Row-level security

The Docker dev database connects as `postgres`, a superuser that bypasses every policy, so a test that
queries as `postgres` passes whatever the policy says. Create a non-privileged role in the test, run
the queries as that role, and drop it in `cleanup`. In this example, a Go test connects with the
role's URL, built by `urluserinfo` (https://atlasgo.io/faq/testing-rls):

```hcl
variable "working_dir" {
  type = string
}

test "schema" "tenant_isolation" {
  # Create a non-privileged user for testing.
  exec {
    sql = <<-SQL
      CREATE ROLE testing_user LOGIN PASSWORD 'pass';
      GRANT USAGE ON SCHEMA public TO testing_user;
      GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO testing_user;
    SQL
  }
  # Drop the user.
  cleanup {
    sql = <<-SQL
      DROP OWNED BY testing_user;
      DROP ROLE testing_user;
    SQL
  }
  # Test the RLS policy using Go (can be any language).
  external {
    program = [
      "go", "test",
      "-run", "Test_TenantIsolation",
      "--dev-url", urluserinfo(self.dev_url, "testing_user", "pass"),
    ]
    working_dir = var.working_dir
  }
}
```

### Template databases

Template databases make `atlas schema test` fast; `atlas migrate test` does not use them. With
`template = true`, every test group starts from a fresh copy of the dev database, and role changes are
reverted (https://atlasgo.io/guides/postgres/schema-test-template-databases):

```hcl
docker "postgres" "dev" {
  image    = "postgres:18"
  template = true
  baseline = <<-SQL
    CREATE ROLE app_owner;
    CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
  SQL
}
```

They need a database-scoped dev URL and plain PostgreSQL (not CockroachDB, YugabyteDB, or Aurora DSQL).
With a schema-scoped dev URL, `template = true` is skipped without an error. A dev database you provide
must be dedicated to Atlas and allowed to create databases and roles.

### pgTAP

pgTAP runs inside Atlas tests: install it in the dev image, run `SELECT no_plan();` first, and match
`^ok` in the assertions (https://atlasgo.io/guides/postgres/pgtap-with-atlas). Prefer custom lint rules
for structural checks, such as "every table has a primary key" (`references/custom-rules.md`).

## Migration Tests

Each case starts from an empty database. Migrate to the version before the change, seed rows, migrate
to the version under test, and check the data (https://atlasgo.io/guides/testing/data-migrations):

```hcl
test "migrate" "check_latest_post" {
  migrate {
    to = "20240807192632"
  }
  exec {
    sql = <<-SQL
      INSERT INTO users (id, email) VALUES (1, 'user1@example.com'), (2, 'user2@example.com');
      INSERT INTO posts (id, title, created_at, user_id) VALUES (1, 'My First Post', '2024-01-23 00:51:54', 1), (2, 'Another Interesting Post', '2024-02-24 02:14:09', 2);
    SQL
  }
  migrate {
    to = "20240807192934"
  }
  exec {
    sql = "select * from users"
    format = table
    output = <<TAB
 id |       email       |   latest_post_ts
----+-------------------+---------------------
1  | user1@example.com | 2024-01-23 00:51:54
2  | user2@example.com | 2024-02-24 02:14:09
TAB
  }
  log {
    message = "Data migrated successfully"
  }
}
```

Seeded rows expose the PostgreSQL problems that an empty database hides. Write a migration test for
each migration that Rule 9 in `SKILL.md` flags:

- A type conversion that needs `USING`, such as `text` to `integer`: seed values, migrate, and check
  the converted values.
- A populated column converted to `serial`: seed rows, migrate, then insert a row without an id.
  Without the `setval`, the insert fails with a duplicate key.
- A backfill or any other data change: compare the rows after the migration, as above.
