# Snowflake Migrations: Lint, Partial Failures, and Deployments

How Atlas runs migrations on Snowflake, which commits DDL as it goes and has no lock Atlas can hold, and
what lint checks there. Tests are in `references/testing.md`. Every Snowflake command needs
`atlas login`; if `atlas whoami` shows no login, ask the user to run `atlas login`, and a user without an
account can start a free trial with it.

## What Lint Catches on Snowflake

Snowflake has no engine-specific analyzers. These run on it (https://atlasgo.io/lint/analyzers):

| Code | Reports | Fix |
|------|---------|-----|
| `DS101`, `DS102`, `DS103` | Dropping a schema, a table, or a column. A table recreated by a plan shows up as a drop too | Confirm the drop, or add a pre-migration check that the table is empty, as lint suggests |
| `CD101`, `CD103` | Dropping a foreign key or a primary key | Confirm it is intended |
| `TX101` | A file with several statements that cannot share a transaction: on Snowflake, anything but `INSERT`, `UPDATE`, `DELETE`, `SELECT`, `MERGE`, `TRUNCATE`, and `COPY INTO` | Transactions below |
| `TX102` | An `atlas:txmode` directive with no empty line after it, which Atlas then ignores | Add the empty line |
| `NM101` to `NM106` | Names that break the `naming` policy, when configured | Rename, or adjust the policy |
| No code: `statement rule violations detected` | SQL that matches a `deny` rule of the `statement` analyzer, when configured | `references/custom-rules.md` |

There are no `MF` (data-dependent) or `BC` (backward-incompatible) checks on Snowflake. Write a custom
rule or a `statement` rule for what the user needs checked (`references/custom-rules.md`).

`TX101` fires on most multi-statement Snowflake files, since nearly every DDL statement commits on its
own. It skips the first migration file. The analyzer's remedies are to split the statements into
separate files, or to start the file with `-- atlas:txmode none` and an empty line. The directive
silences the check; it does not make the file atomic.

Each check is tuned in the `lint` block (https://atlasgo.io/lint/analyzers#check-policy):

```hcl
lint {
  destructive {
    error = true
  }
  check "TX101" {
    skip = true
  }
}
```

## Transactions and Partial Failures

- Snowflake commits the open transaction before and after every DDL statement. `atlas migrate apply`
  still runs each file under `--tx-mode file` by default, but only a file of DML statements rolls back
  on failure. A file with DDL that fails halfway stays half applied, whatever `--tx-mode` says.
- Atlas records each statement as it runs. After a failure, `atlas migrate status` shows the file as
  partially applied, and the next `migrate apply` continues from the statement that failed.
- To recover, fix the failed statement and the ones after it, leave the statements that already ran
  unchanged, and run `atlas migrate hash`. Editing a statement that already ran fails with
  `sql/migrate: history changed: statement 1 from file "..." changed`.
- Keep migration files small: one table change, or one object, per file, so a failure leaves little to
  untangle. Run them on the dev database first with `atlas migrate test`.
- `atlas migrate down` works on Snowflake: it plans the reverse statements and checks them on the dev
  database. A hybrid table is dropped with `DROP TABLE`, which is the form Snowflake accepts.

## No Apply Lock

Atlas takes no lock on Snowflake: `migrate apply` and `schema apply` do not wait for each other, and
`--lock-timeout` has no effect. Two deployments to one database at the same time can both run the same
pending files. Serialize them in the deployment system, for example one concurrency group per target
database in CI.

## Revisions Table

- With a schema-scoped URL, `atlas_schema_revisions` is created in the connected schema.
- With a database-scoped URL, Atlas creates a schema named `atlas_schema_revisions` in the target
  database and puts the table there. `--revisions-schema`, or `revisions_schema` in the env's `migration`
  block, moves it.
- After an existing project switches its URL from database scope to schema scope, Atlas fails with
  `ambiguous revision table` until `--revisions-schema` names the schema that holds the existing table.
- At database scope, `atlas schema diff` and `schema apply` against the live database see the
  revisions schema as one the desired state does not declare, and plan to drop it. List the managed
  schemas on the env with `schemas`, or add `--exclude atlas_schema_revisions`
  (https://atlasgo.io/guides/dbt/snowflake).

## Reviewing Plans

Snowflake makes many changes by dropping and creating the object (`references/objects.md`), and some
Snowflake statements replace an object as a side effect:

- A dropped and created dynamic, external, or Iceberg table loses its time travel history, and an
  Iceberg table managed by Snowflake loses its data. Lint reports the `DROP` (`DS102`).
- A view change is `CREATE OR REPLACE VIEW`, which drops every grant on the view. With
  `permissions = true`, Atlas re-grants the declared ones in the same plan; grants made outside Atlas are
  lost. A function or procedure body change is a drop and a create, with the same effect on its grants.
- A column change that needs the column recreated is prompted for in a terminal and fails the diff
  in CI, unless `allow_recreate = true` plans it (`references/objects.md`, Column Changes). Either way
  the column's data is lost. For data you keep, write it by hand:

  ```sql
  ALTER TABLE "ORDERS" ADD COLUMN "AMOUNT_V2" NUMBER(12, 2) NULL;
  UPDATE "ORDERS" SET "AMOUNT_V2" = "AMOUNT";
  ```

  Then drop the old column and rename the new one in a later file, after the application reads it.
- A dynamic table without `warehouse` gets the planning connection's warehouse written into the
  `CREATE` (`references/objects.md`).

## Diff Policy

- `add_table { if_not_exists = true }`, `drop_table { if_exists = true }`, and
  `add_column { if_not_exists = true }` write the `IF [NOT] EXISTS` clause on tables of every kind and
  on columns. `add_index` has no effect: Snowflake has no indexes
  (https://atlasgo.io/versioned/diff#diff-policy):

  ```hcl
  diff {
    add_column {
      if_not_exists = true
    }
  }
  ```

- The `skip` block keeps change kinds out of plans, such as `drop_schema`, `drop_table`, `drop_column`,
  `drop_view`, `drop_func`, `rename_constraint`, and `drop_role`.
- Drops of other objects, such as stages, tasks, policies, secrets, warehouses, and integrations, have
  no `skip` option: keep them out of the env with `include` and `exclude` patterns.
- `modify_column { allow_recreate = true }` plans the column changes Snowflake cannot alter as a drop
  and an add (`references/objects.md`, Column Changes).

## Planning Time

Every `migrate diff`, `migrate lint`, `schema plan`, and `schema apply` loads the desired state, the
migration directory, or both on the dev database, and Snowflake DDL is slow: a lint of a few statements
can take about a minute (https://atlasgo.io/guides/dbt/snowflake). In CI, budget for it, lint only new
files with `--latest`, and keep `migrate checkpoint` files to shorten long directories.

## Existing Databases

A database that has objects but no revisions table is refused until it gets a baseline: the first
migration recorded as applied without running (https://atlasgo.io/versioned/apply#existing-databases).
Export the current schema with `atlas schema inspect` at the scope the project will use, then follow the
baseline steps in the `atlas` skill.

## Seed Data

Snowflake has no `data` block for seed rows: write them as `INSERT` or `MERGE` statements in a migration
file (`references/hcl.md`, Seed Data).
