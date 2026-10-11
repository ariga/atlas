# Databricks Migrations: Lint, Transactions, and Recovery

How Atlas runs migrations on a database that commits every DDL statement on its own and takes no
locks, and what lint checks on Databricks. Tests are in `references/testing.md`. Lint and custom rules
need `atlas login`, as does the Databricks driver itself. If `atlas whoami` shows no login, ask the user
to run `atlas login`; a user without an account can start a free trial with it.

## What Lint Catches on Databricks

Databricks has no driver-specific analyzers. Lint runs these (https://atlasgo.io/lint/analyzers):

| Code | Reports |
|------|---------|
| `DS101` | Dropping a schema. Atlas drops schemas with `CASCADE`, so every object in it goes too |
| `DS102`, `DS103` | Dropping a table, or a column |
| `BC103`, `BC104` | Dropping a table or a column that clients may still read |
| `CD101`, `CD102`, `CD103` | Dropping a foreign key, a check, or a primary key |
| `NM101` to `NM106` | Names that break the `naming` policy |
| `TX102` | An `atlas:txmode` directive attached to a statement instead of the file header, which takes no effect |

What lint does not check on Databricks:

- No data-dependent checks (`MF*`) run. Nothing reports `SET NOT NULL` on a column that holds `NULL`, or a
  type change that existing values may not fit. Cover those with migration tests on seeded rows
  (`references/testing.md`).
- `TX101`, the multi-statement file that cannot run in a transaction, never fires on Databricks: the
  driver reports no statement as the cause, and the check needs one.
- Plans that change a key report `CD101` or `CD103`, because Atlas drops and re-adds the key
  (`references/objects.md`, Keys and Constraints). Review them, then accept them with
  `-- atlas:nolint CD101` on the statement, with the user's approval.
- The rename checks (`BC101`, `BC102`) never fire. Atlas plans no renames on Databricks, and lint
  reads a hand-written `RENAME COLUMN` or `RENAME TO` as a drop and an add, so it reports `DS102` or
  `DS103` and `BC103` or `BC104` on it. The destructive check is an error, not a warning, so a
  hand-written rename fails `migrate lint` until it is accepted. Confirm the statement is a rename,
  then accept it with `-- atlas:nolint` on the statement, with the user's approval
  (`references/objects.md`, Renames).

Statement rules fill gaps for Databricks-specific risks, such as a rewrite of a materialized view,
which refreshes it in full (https://atlasgo.io/lint/analyzers#statement-rules):

```hcl
lint {
  statement {
    deny "mv-rewrite" {
      match   = "(?i)CREATE OR REPLACE MATERIALIZED VIEW"
      message = "rewriting a materialized view refreshes it in full: confirm the change"
    }
  }
}
```

Each check is tuned in the `lint` block (https://atlasgo.io/lint/analyzers#destructive-changes):

```hcl
lint {
  destructive {
    error = true
  }
}
```

### Custom rules

When no analyzer covers a check, or the user states a convention, such as a comment on every table or
no Python functions, write a custom rule in the HCL rule language: `references/custom-rules.md`.

## Transactions

Databricks runs no statement in a transaction, and its SQL driver cannot open one. Atlas plans every
Databricks change as non-transactional, and each statement commits on its own
(https://atlasgo.io/versioned/apply#transaction-configuration):

- `atlas migrate apply` and `atlas schema apply` default to `--tx-mode file`, open a transaction per
  file or plan before running it, and fail on Databricks with
  an error that starts with `sql/sqlclient: starting transaction:`. Run both with
  `--tx-mode none`, or set it once in the env's `migration` block, which both commands read:

  ```hcl
  env "prod" {
    url = "${local.dbx}?catalog=prod"
    migration {
      dir     = "file://migrations"
      tx_mode = "none"
    }
  }
  ```

- `--tx-mode all` fails the same way. The `tx_mode` setting does not reach `atlas migrate test`,
  which runs each file in a transaction unless the file itself starts with `-- atlas:txmode none`
  followed by a blank line. `migrate diff` does not add the directive, so add it to the top of
  every file and run `atlas migrate hash`. Lint reports `TX102` when a directive sits on a later
  statement, where it takes no effect.
- `atlas migrate set` also opens a transaction and fails on Databricks. To record a version by hand,
  baseline the target instead (Existing Databases below).
- A file or a plan that fails halfway stays half applied. The statements before the failure remain,
  and Atlas records how many statements of the file ran, so `atlas migrate status` shows the file as
  partially applied (https://atlasgo.io/versioned/troubleshoot).
- The next `atlas migrate apply` continues from the statement that failed. Fix that statement and the
  ones after it, leave the applied ones unchanged, and run `atlas migrate hash`. When data caused the
  failure, fix the data and add the fix to the file.
- If the connection drops between a statement and its entry in the revisions table, Atlas resumes one
  statement early. Revert that statement by hand before the next run
  (https://atlasgo.io/versioned/troubleshoot#connection-loss-failures).
- `atlas migrate down` reverts statement by statement on Databricks, without a transaction, and
  updates the revisions table after each, so a run that fails midway is re-run to continue
  (https://atlasgo.io/versioned/down). Reverting a dropped table or column recreates it empty. A file
  that drops a generated or identity column has no reverse, so `migrate down` cannot revert it.
- `schema apply` has the same limit: a plan that fails midway leaves its first statements applied, and
  the next `schema apply` plans the rest.
- Atlas scans `BEGIN ... END` blocks as one statement, so a procedure body with semicolons needs no
  delimiter directive. String literals use backslash escapes.
- Keep files small. A file with one logical change leaves little to recover after a failure.

## Locks

Databricks has no advisory locks, and Atlas takes none on it. `--lock-timeout` and `--lock-name`
change nothing, and two `migrate apply` or `schema apply` runs against one target can interleave
their statements and their revisions table writes.

- Run one deployment per target at a time: one CI concurrency group per catalog, or one deployment
  job that applies to each target in order.
- Do not let two environments share a dev catalog at the same time (`references/dev-database.md`).

## Revisions Table

- With a catalog-scoped URL, `atlas_schema_revisions` gets a schema of its own, also named
  `atlas_schema_revisions`, in the target catalog. With a schema-scoped URL, it lives in that schema.
  `--revisions-schema`, or `revisions_schema` in the env's `migration` block, moves it.
- At catalog scope, the revisions schema shows up in `schema inspect` and `schema diff` of a versioned
  target, and a declarative plan would drop it: add `--exclude "atlas_schema_revisions"`.
- After an existing project switches its URL from catalog scope to schema scope, Atlas finds no table
  in the connected schema but one in `atlas_schema_revisions`, and fails with
  `ambiguous revision table`. Set `--revisions-schema atlas_schema_revisions` to keep using it.

## Existing Databases

A target that has objects but no revisions table is refused until it gets a baseline: the first
migration, recorded as applied without running (https://atlasgo.io/versioned/apply#existing-databases).
The refusal reads:

```
sql/migrate: connected database is not clean: found table "users" in schema "default". baseline version or allow-dirty is required
```

At catalog scope, an empty schema other than `default` is refused the same way, with
`found schema "..."`.

1. Inspect the target into code: `atlas schema inspect --env prod --format '{{ sql . | split | write "src" }}'`,
   or into HCL, and drop the `delta.*` properties the team does not set (`references/objects.md`).
2. Generate the baseline: `atlas migrate diff --env prod baseline --to file://src`, and add
   `-- atlas:txmode none` to the top of the file, then `atlas migrate hash`.
3. Record it once per existing target: `atlas migrate apply --env prod --baseline <version> --tx-mode none`,
   or set `baseline` in the env's `migration` block.

When `--dev-url` points at a catalog that already has the revisions table, `migrate diff` fails with an
error that starts with `connected database is not clean`: point it at the dedicated dev catalog.

## Large Changes

Nothing in Atlas batches a single huge plan for you. To spread a large change over several deployments,
split it into several migration files and apply a few at a time with `atlas migrate apply <n>`. For a
change Databricks cannot make in place, such as repartitioning a table, write the new table, the data
copy, and the swap as separate files, and test them first.
