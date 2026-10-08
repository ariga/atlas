# Testing MySQL Schemas and Migrations

Atlas runs tests on the dev database: `atlas schema test` checks the logic in the schema, and
`atlas migrate test` checks migration files against seeded data. The test syntax, every command, and
examples are in `../atlas/references/testing.md`; this file covers how to use them on MySQL. Testing
needs `atlas login`. If `atlas whoami` shows no login, ask the user to run `atlas login`; a user without
an account can start a free trial with it.

## Test Every Change

A schema or migration change is done when its tests pass:

1. After editing the schema, run `atlas schema validate --env <name>`.
2. Add or update a test case for what the change touches: a `test "schema"` case for a function,
   procedure, trigger, view, generated column, or check constraint, and a `test "migrate"` case for a
   migration that moves or converts data.
3. Run the new case alone with `--run <case>`, and fix the logic until it passes.
4. Run the whole suite: `atlas schema test --env <name>` and `atlas migrate test --env <name>`.
5. Lint the change, so the MySQL analyzers and custom rules check it (`references/migrations.md`).
6. Report the cases you added and that they pass. If the tests could not run, say why.

The env lists the test files next to its MySQL dev URL (`references/dev-database.md`):

```hcl
  test {
    schema {
      src = ["schema.test.hcl"]
    }
  }
```

Migration tests are listed the same way, under `test { migrate { src = [...] } }`. Without a project
file, pass the dev URL and the directory (https://atlasgo.io/cli-reference#atlas-migrate-test):

```bash
  atlas migrate test --dev-url docker://mysql/8/dev --dir file://migrations .
```

## Why It Matters More on MySQL

- MySQL commits DDL as it goes, so a migration file that fails halfway stays half applied
  (`references/migrations.md`, Transactions). A migration test applies the files to the dev database
  first, where a failing statement fails the test instead of production.
- Tests see the dev database's behavior. Run it on the target's version and character set
  (`references/dev-database.md`), or a test can pass on the dev database and fail on the target.
- MySQL has no template databases, which are PostgreSQL only; the dev database is reset between cases.
- On RDS, migrations that create IAM users fail on a stock dev image with `Error 1524`. Run the tests on
  the dev image with a mock of the plugin (`references/dev-database.md`, RDS IAM users on the dev
  database).

## Schema Tests

Each case starts from the desired schema, created on the dev database. What to cover on MySQL:

- Functions and procedures: normal inputs, edge cases, and error paths with `catch`
  (https://atlasgo.io/guides/testing/functions, https://atlasgo.io/guides/testing/procedures).
- Triggers: seed rows, change them, and check the side effects
  (https://atlasgo.io/guides/testing/triggers).
- Check constraints: `catch` a row that breaks the check. MySQL 8.0.15 and earlier ignore checks, so the
  test also shows whether the dev database enforces them.
- Generated columns: insert rows and compare the computed values.
- Views: seed rows and compare the view's output (https://atlasgo.io/guides/testing/views).

## Migration Tests

Each case starts from an empty database. Migrate to the version before the change, seed rows, migrate
to the version under test, and check the data (https://atlasgo.io/guides/testing/data-migrations).

Seeded rows expose the MySQL failures that an empty database hides. Write a migration test for each
migration that:

- Changes a column's type, where existing values may not fit the new one (`MY130`).
- Removes or changes enum or set values: rows that hold a removed value fail the change under strict
  `sql_mode`, and silently become `''` without it (`MY110`). Seed every value, migrate, and check that
  none was lost.
- Makes a column `NOT NULL` (`MF104`), or adds a `NOT NULL` column without a default (`MY101`): seed
  rows, migrate, and check the values MySQL filled in.
- Backfills or moves data: compare the rows after the migration.

In GitHub Actions, the `migrate/test` action runs the tests against a MySQL service container
(https://atlasgo.io/integrations/github-actions):

```yaml
      - uses: ariga/atlas-action/migrate/test@v1
        with:
          dir: file://migrations
          dev-url: mysql://root:pass@localhost:3306/dev
          run: 'example'
```
