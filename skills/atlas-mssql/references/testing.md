# Testing SQL Server Schemas and Migrations

Atlas runs tests on the dev database: `atlas schema test` checks the logic in the schema, and
`atlas migrate test` checks migration files against seeded data. The test syntax and every command are
in `../atlas/references/testing.md`; this file covers how to use them on SQL Server. Testing needs
`atlas login`, which SQL Server needs anyway.

## Test Every Change

A schema or migration change is done when its tests pass:

1. After editing the schema, run `atlas schema validate --env <name>`.
2. Add or update a test case for what the change touches: a `test "schema"` case for a function,
   procedure, trigger, view, computed column, or check, and a `test "migrate"` case for a migration that
   moves, converts, or narrows data.
3. Run the new case alone with `--run <case>`, and fix the logic until it passes.
4. Run the whole suite: `atlas schema test --env <name>` and `atlas migrate test --env <name>`.
5. Lint the change, so the analyzers and custom rules check it (`references/migrations.md`).
6. Report the cases you added and that they pass. If the tests could not run, say why.

The env lists the test files (https://atlasgo.io/testing/schema):

```hcl
env "dev" {
  src = "file://schema.sql"
  dev = "docker://sqlserver/2022-latest/dev?mode=schema"
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

## Writing Assertions in T-SQL

- `exec` with `output` compares the query result as text. SQL Server `decimal`, `numeric`, `money`, and
  `uniqueidentifier` values print as hex bytes, such as `0x31302e3530` for `10.50`. Cast them in the
  query: `CAST(total AS varchar(20))`, `CAST(id AS char(36))`.
- `bit` prints as `true` or `false`, and `datetime2` without its fractional seconds. Use
  `CONVERT(varchar(30), created_at, 121)` to compare the full value.
- `assert` needs a single value that reads as a boolean. T-SQL has no boolean expressions in `SELECT`,
  so write `SELECT IIF(<condition>, 1, 0)` or `CASE WHEN ... THEN 1 ELSE 0 END`.
- `catch` matches SQL Server's message, which starts with `mssql:`, such as
  `mssql: The INSERT statement conflicted with the CHECK constraint "ck_orders_total"`. Match a stable
  part of it: `error = "CHECK constraint \"ck_orders_total\""`.
- Statements in one `exec` block run as separate batches, so a variable from one statement is gone in
  the next. Keep `DECLARE` and its use in one `BEGIN ... END` block, or use a function.

## Schema Tests

Each case starts from the desired schema, created on the dev database:

```hcl
test "schema" "order_total" {
  exec {
    sql = "INSERT INTO dbo.orders (total) VALUES (10.50)"
  }
  exec {
    sql    = "SELECT CAST(dbo.with_tax(100) AS varchar(20))"
    output = "117.00"
  }
  assert {
    sql = "SELECT IIF(dbo.with_tax(0) = 0, 1, 0)"
  }
  catch {
    sql   = "INSERT INTO dbo.orders (total) VALUES (-1)"
    error = "CHECK constraint \"ck_orders_total\""
  }
}
```

What to cover on SQL Server:

- Functions and procedures: normal inputs, edge cases, and error paths with `catch`. Call a procedure
  with `EXEC dbo.proc_name @arg = 1` (https://atlasgo.io/guides/testing/procedures).
- Triggers: seed rows, change them, and check the side effects. A trigger runs once per statement, so
  test a multi-row `UPDATE`: a trigger written for one row (`SELECT @id = id FROM inserted`) passes a
  single-row test and loses rows on a bulk one (https://atlasgo.io/guides/testing/triggers).
- Views and computed columns: seed rows and compare the output (https://atlasgo.io/guides/testing/views).
- Checks and alias types: `catch` an invalid value.

### Permissions

Grants are tested as a less privileged principal. A `script` case with `as { role = "<user>" }` runs
Atlas Data Scripts under `EXECUTE AS USER`, so the user must be a database user, such as one created
`WITHOUT LOGIN`. The scripts live in their own file:

```hcl
# scripts.hcl
script "query" "count_orders" {
  query "q" {
    sql    = "SELECT count(*) FROM [dbo].[orders]"
    format = CSV
  }
}

script "exec" "purge_orders" {
  exec {
    sql = "DELETE FROM [dbo].[orders]"
  }
}
```

```hcl
# schema.test.hcl
test "schema" "reader_cannot_delete" {
  exec {
    sql = <<-SQL
      INSERT INTO [dbo].[orders] ([total]) VALUES (10);
      DROP USER IF EXISTS [reader];
      CREATE USER [reader] WITHOUT LOGIN WITH DEFAULT_SCHEMA = [dbo];
      GRANT SELECT ON [dbo].[orders] TO [reader];
    SQL
  }
  cleanup {
    sql = "DROP USER IF EXISTS [reader]"
  }
  script "query" {
    file   = "scripts.hcl"
    run    = "^count_orders$"
    as     { role = "reader" }
    output = "1"
  }
  script "exec" {
    file  = "scripts.hcl"
    run   = "^purge_orders$"
    as    { role = "reader" }
    error = "permission was denied"
  }
}
```

- An `as` block with `user` and `password` opens a separate connection as a SQL login instead; create
  the login and its user in the test, and drop both in `cleanup`.
- Users belong to the database, not to the schema the test restores. Drop them in `cleanup`, and drop
  them `IF EXISTS` before creating them, or the next run on a dev database you provide fails with
  `User, group, or role 'reader' already exists in the current database`.

### Template databases

Template databases are not available on SQL Server: `template = true` fails with
`template attribute is supported only for postgres dev-databases`. Each test case restores the dev
database when it ends. Set `parallel = true` only on stateless cases, without `INSERT` or `CREATE`
(https://atlasgo.io/testing/schema): parallel cases share the dev database and see each other's rows.

## Migration Tests

Each case starts from an empty database. Migrate to the version before the change, seed rows, migrate
to the version under test, and check the data (https://atlasgo.io/guides/testing/data-migrations):

```hcl
test "migrate" "split_names" {
  migrate {
    to = "20240101000000"
  }
  exec {
    sql = "INSERT INTO users (id, full_name) VALUES (1, N'Ada Lovelace'), (2, N'Plato')"
  }
  migrate {
    to = "20240102000000"
  }
  exec {
    sql    = "SELECT id, first_name, last_name FROM users ORDER BY id"
    format = table
    output = <<TAB
 id | first_name | last_name 
----+------------+-----------
 1  | Ada        | Lovelace  
 2  | Plato      |           
TAB
  }
}
```

Seeded rows expose the SQL Server problems that an empty database hides. Write a migration test for:

- A narrower type or a shorter length: seed a value that does not fit, and expect
  `String or binary data would be truncated` in a `catch`, or fix the data first.
- `NULL` to `NOT NULL`: seed a `NULL`, and check the backfill.
- A change Atlas plans as a drop and an add of a column (identity, computed): seed rows, and check the
  values after the migration.
- A backfill or any other data change: compare the rows after the migration, as above.
