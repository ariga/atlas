# Redshift Migrations: Lint, Locks, and Transactions

How Atlas runs migrations on Redshift, a database with no advisory locks and with statements that
cannot run in a transaction. Tests are in `references/testing.md`; the column changes Redshift cannot
make in place are in `references/objects.md`. Every Redshift command needs `atlas login`. If
`atlas whoami` shows no login, ask the user to run `atlas login`; a user without an account can start
a free trial with it.

## What Lint Catches on Redshift

Redshift has no driver-specific analyzers (no `PG*` or `MY*` codes). Lint runs the general ones
(https://atlasgo.io/lint/analyzers):

| Code | Catches | Fix |
|------|---------|-----|
| `DS101`, `DS102`, `DS103` | Dropping a schema, a table, or a column | Confirm the drop, or add a pre-migration check that the object is empty |
| `TX101` | A file with several statements, some of which cannot run in a transaction, and no `-- atlas:txmode none` | Split the file, or add the directive (Transactions below) |
| `TX102` | An `atlas:txmode` directive attached to a statement instead of the file, so it is ignored | Put an empty line after the directive |
| `NM101` to `NM106` | Names that break the `naming` policy | Rename, or adjust the policy |

The constraint-drop checks (`CD1*`) never fire: Atlas does not model constraints on Redshift. On the
schema itself, `atlas schema lint` runs the `naming` checks. Each check is tuned in the `lint` block:

```hcl
lint {
  destructive {
    error = true
  }
  naming {
    match   = "^[a-z_]+$"
    message = "must be lowercase"
  }
}
```

A rebuilt column (`allow_recreate`) is a `DROP COLUMN` followed by an `ADD COLUMN`; read lint's
destructive findings on such files before applying them.

### Custom rules

Most Redshift conventions need a custom rule, such as an explicit `encode` on every column, a
distribution key on fact tables, or no `INTERLEAVED` sort keys. A rule can check any attribute Atlas
inspects: `references/custom-rules.md`.

## Transactions

- `atlas migrate apply` runs each file in its own transaction by default (`--tx-mode file`); `all` runs
  every pending file in one, and `none` runs statements without one
  (https://atlasgo.io/versioned/apply#transaction-configuration).
- Redshift rejects these statements inside a transaction block:

  | Statements |
  |------------|
  | `ALTER TABLE ... ALTER COLUMN ... TYPE`, `ALTER TABLE ... ALTER SORTKEY NONE`, `ALTER TABLE ... APPEND FROM` |
  | `CREATE DATABASE`, `DROP DATABASE`, `CREATE TABLESPACE`, `DROP TABLESPACE` |
  | `CREATE EXTERNAL TABLE`, `ALTER EXTERNAL TABLE`, `DROP EXTERNAL TABLE`, `RENAME EXTERNAL TABLE`, and `GRANT` or `REVOKE` `ON EXTERNAL SCHEMA` or `ON EXTERNAL TABLE` |
  | `GRANT USAGE ON DATASHARE`, `REVOKE USAGE ON DATASHARE`, `ALTER DATASHARE ... SET PUBLICACCESSIBLE` |
  | `CREATE LIBRARY`, `DROP LIBRARY`, `VACUUM`, `COPY`, `UNLOAD`, `REINDEX DATABASE`, `REBUILDCAT`, `INDEXCAT` |

- When a plan holds one of them, Atlas marks it non-transactional: `atlas migrate diff` starts the file
  with `-- atlas:txmode none`, and `atlas schema apply` runs the plan without a transaction:

  ```sql
  -- atlas:txmode none

  -- Modify "t1" table
  ALTER TABLE "t1" ADD COLUMN "c3" character varying(100) NOT NULL DEFAULT 'unknown' ENCODE LZO;
  -- Modify "t1" table
  ALTER TABLE "t1" ALTER COLUMN "c1" TYPE character varying(200);
  ```

- The directive covers the whole file, so every statement in it runs on its own. A file that fails
  midway stays half applied, and the next run resumes from the failed statement. Keep these changes in
  small files: move the `ALTER COLUMN ... TYPE` or the datashare grant into its own migration with
  `atlas migrate new` when it shares a file with other changes.
- A hand-written file needs the directive at the top, followed by an empty line:

  ```sql
  -- atlas:txmode none

  ALTER TABLE "t1" ALTER COLUMN "c1" TYPE character varying(200);
  ```

  Without it, the statement fails inside the file's transaction, such as
  `pq: ALTER TABLE ALTER COLUMN cannot run inside a transaction block (25001)`, and lint reports
  nothing for a file that holds only that statement (`TX101` needs several). Add the directive, run
  `atlas migrate hash`, and apply again. Directives work only in Atlas-format migration directories.
- Never put `BEGIN`/`COMMIT` in a migration file: Atlas runs each file in a transaction unless it sets
  `txmode none`.
- When fixing a failed non-transactional file, leave the statements that already ran unchanged, and run
  `atlas migrate hash` after editing.

### The Data API

Through `redshift+http://`, Atlas sends statements as Data API calls instead of over one session:

- A transactional file is buffered and sent as one batch when the transaction commits. Redshift runs
  the batch in one transaction, so a failure rolls the whole file back and reports
  `query failed: <Redshift error>`. The URL's `timeout` bounds the whole batch: raise it for long files
  (`references/dev-database.md`, Connection URLs).
- A file with `txmode none` sends one statement at a time, and each statement runs in its own session.
  Session state does not carry over: a `SET search_path`, a `SET` of another parameter, or a temporary
  table created by one statement is gone in the next. Qualify names, and use regular tables for staging
  data in such files.

## Locks

Redshift has no advisory locks, so Atlas takes no lock for `migrate apply` or `schema apply`, and
`--lock-timeout` and `--skip-lock` change nothing. Two runs against the same database can apply at the
same time. Serialize deployments in CI, for example with one concurrency group per target database.
Atlas locks only the revisions table (`LOCK TABLE`) while it writes a revision.

## Revisions Table

- `atlas_schema_revisions` lives in the schema the URL is bound to: the `search_path` of a
  `redshift://` URL, or the user's current schema for a `redshift+http://` URL in `mode=schema`. With a
  database-scoped URL it gets a schema of its own, `atlas_schema_revisions`. `--revisions-schema`, or
  `revisions_schema` in the env's `migration` block, moves it.
- After an existing project switches its URL between schema and database scope, Atlas fails with
  `ambiguous revision table` until `--revisions-schema` names the schema that holds the existing table.
- On a versioned database, add `--exclude atlas_schema_revisions` when diffing the live database
  against the desired state, or the revisions table shows up as a drop.

## Plans Atlas Refuses

| Error | Cause | Fix |
|-------|-------|-----|
| `Column "c" default change requires recreating the column.` (also `nullability` and `type`) | Redshift cannot alter it | `allow_recreate`, or a hand-written data migration (`references/objects.md`) |
| `adding non-nullable column "c" requires a default value in Redshift` | Redshift rejects it even on an empty table | Add a `default`, or `null = true` |
| `adding identity column "c" is not supported after the table is created` | Identity only in `CREATE TABLE` | Rebuild the table by hand |
| `modify column identity is not supported in Redshift` | Seed, step, or generation changed | Rebuild the table by hand |
| `modify sort key to INTERLEAVED is not supported in Redshift` | Interleaved keys only in `CREATE TABLE` | Rebuild the table by hand |
| `cannot modify view "v" to materialized view` | The kind changed | Drop in one migration, create in the next |
| `cannot create table "t" in external schema "s"` | External schemas hold no local objects | Create it in a local schema |
| `"dev-url" not set` | `migrate diff` needs a dev database | Set `dev` on the env |

## Existing Databases

A database that has objects but no revisions table is refused until it gets a baseline: the first
migration recorded as applied without running (https://atlasgo.io/versioned/apply#existing-databases).
Inspect the database at the target's scope, write the baseline file from it, and set `baseline` in the
env's `migration` block, or pass `--baseline <version>` once.

## Large Changes

Redshift rebuilds can take a long time on large tables. To spread a large change over several
deployments, split it into several migration files and apply a few at a time with
`atlas migrate apply <n>` (`../atlas/references/versioned.md`).
