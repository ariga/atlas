# Snowflake Objects and Their Nuances

How Atlas models Snowflake objects, which ones it reads by default, and the changes Snowflake makes
only by dropping and creating the object again. The full attribute reference is
https://atlasgo.io/hcl/snowflake, with examples in https://atlasgo.io/atlas-schema/hcl. HCL details
are in `references/hcl.md`.

## What Atlas Reads, and the Switches

Every Snowflake command needs `atlas login`. Beyond that, an env's `mode` blocks decide what Atlas
inspects and plans:

| Objects | Read by default | Switch | Scope |
|---------|-----------------|--------|-------|
| Schemas, tables of every kind, constraints | Yes | `mode { tables = false }` turns tables off | Either |
| Views | Yes | `views` | Either |
| Functions and procedures | Yes | `funcs` | Either |
| File formats, stages, sequences, tasks, pipes | Yes | `objects` | Either |
| Account roles, database roles, users | No | `mode { roles = true }` | Database |
| Grants | No | `mode { permissions = true }` | Database |
| Tags and tag assignments | No | `mode "snowflake" { tags = true }` | Either |
| Masking, row access, projection, aggregation, and join policies, and their attachments | No | `mode "snowflake" { policies = true }` | Either |
| Secrets | No | `mode "snowflake" { secrets = true }` | Either |
| Alerts | No | `mode "snowflake" { alerts = true }` | Either |
| Network rules | No | `mode "snowflake" { network_rules = true }` | Either |
| External access integrations | No | `mode "snowflake" { external_access_integrations = true }` | Database |
| Warehouses | No | `mode "snowflake" { warehouses = true }` | Database |
| Resource monitors | No | `mode "snowflake" { resource_monitors = true }` | Database |
| External volumes | No | `mode "snowflake" { external_volumes = true }` | Database |

```hcl
env "prod" {
  schema {
    src = "file://schema.sf.hcl"
    mode "snowflake" {
      tags     = true
      policies = true
    }
  }
}
```

- A switch that is off leaves those objects out of inspection and plans without an error, and a desired
  state that declares them plans nothing for them. Turn the switch on before relying on a plan.
- The labeled `mode "snowflake"` block also takes the general switches (`tables`, `views`, `funcs`,
  `objects`, `roles`, `permissions`, `sensitive`).
- `schemas = false` skips schemas and everything in them, for an env that manages only account-level
  objects, such as warehouses or roles. Grants on schemas and tables need the schemas in scope.
- Policies need a Snowflake edition that supports them (Enterprise or higher). Atlas reads every policy
  kind in one query, so on an account without join or row access policies, `policies = true` fails
  every command with `query policies: 001003 (42000): SQL compilation error: syntax error ... unexpected 'JOIN'`.
  Leave `policies` off there.
- With an account-level switch on, every object of that kind on the account is in scope: scope the
  env with `include`, such as `"ETL_WH[type=warehouse]"`, or Atlas plans to drop the ones it does not
  declare.

## Not Modeled

Atlas does not inspect or plan the following. Create them outside Atlas, or in the dev `baseline` when
the schema depends on them (`references/dev-database.md`):

- Temporary tables, streams, and materialized views
- Indexes on hybrid tables, other than primary keys and unique constraints
- Ownership and future grants (`references/security.md`)
- Whether a task is started or suspended: Atlas creates tasks and never runs `ALTER TASK ... RESUME`,
  so a new task stays suspended until someone resumes it
- Users themselves: Atlas manages only their role grants
- Storage integrations and notification integrations: stages, pipes, and tasks name them as strings
  (`storage_integration`, `integration`, `error_integration`)

## Table Kinds

| Block | Creates | Kind fixed at creation |
|-------|---------|------------------------|
| `table` | `CREATE TABLE`, or `CREATE TRANSIENT TABLE` with `transient = true` | Permanent or transient |
| `dynamic_table` | `CREATE DYNAMIC TABLE ... AS <query>` | Yes |
| `hybrid_table` | `CREATE HYBRID TABLE`, dropped with `DROP TABLE` | Yes |
| `external_table` | `CREATE EXTERNAL TABLE` over a stage | Yes |
| `iceberg_table` | `CREATE ICEBERG TABLE` on an external volume | Yes |

Snowflake fixes a table's kind when it creates it, and Atlas does not plan a conversion. Changing
`transient`, or moving a `table` to `dynamic_table`, fails the plan:

```
cannot change permanent table to transient after creation
cannot change transient table to permanent after creation
cannot change table type to dynamic after creation
cannot change table type from iceberg after creation
```

Schemas work the same way: `cannot change permanent schema to transient after creation`. Write the
conversion by hand: create the new table, copy the rows, and swap the names.

## Changes Snowflake Makes Only by Recreating

Atlas plans a `DROP` and a `CREATE` for these. The data, the time travel history, and grants made outside
Atlas go with the dropped object:

| Object | Recreated when | Altered in place |
|--------|----------------|------------------|
| `dynamic_table` | The query or `refresh_mode` changes, or a column is added or dropped | `target_lag`, `warehouse`, `cluster_by`, comment |
| `external_table` | Its location, file format, pattern, partitioning, or refresh settings change | Columns, tags, and policies, through `ALTER TABLE` |
| `iceberg_table` | `external_volume`, `catalog`, `base_location`, `partition_by`, `storage_serialization_policy`, or `iceberg_version` changes | `cluster_by`, `target_file_size`, `catalog_sync`, `log_event_level`, `enable_data_compaction`, `enable_iceberg_merge_on_read`, `error_logging`, `path_layout`, `replace_invalid_characters`, comment, columns |
| `function`, `procedure` | The definition changes, such as the body, the handler, or an argument default; `secure` on a procedure | Comment, tags, `secure` on a function |
| `masking_policy` | `arg`, `return`, or `exempt_other_policies` changes | The body (`as`), comment, tags |
| `pipe` | Anything but the alterable settings changes | `error_integration`, comment, tags |
| `sequence` | `start` changes, or `order` turns on | `increment`, `order` turned off, comment |
| `stage` | `credentials` change | Every other setting |
| `network_rule` | `type` or `mode` changes | `value_list`, comment |
| `alert` | `schedule` is removed | `condition`, `action`, a changed `schedule`, `warehouse`, comment, tags |

- Dropping and creating a dynamic table refreshes it again from scratch, on its warehouse.
- Dropping a Snowflake-managed Iceberg table deletes its data. Review every `DROP ICEBERG TABLE` in a
  plan.
- An `iceberg_table` cannot set both `partition_by` and `cluster_by`:
  `iceberg table "events": partition_by and cluster_by cannot be specified together`.
- A file format's `type` cannot change: `cannot alter TYPE of file format "CSV_FORMAT" after creation`.
  Create a new format under another name and point the stages at it.

## Column Changes

What Snowflake alters in place: adding and dropping columns, `NOT NULL` on and off, comments, a type
change within its family that keeps the data (a longer `VARCHAR`, a different `NUMBER` precision at the
same scale), dropping a default, and tags and policies on the column.

What needs the column dropped and added again, losing its data:

- A type change across type families, such as `NUMBER` to `VARCHAR`, `BOOLEAN` to `VARCHAR`, or
  `TIMESTAMP_NTZ` to `DATE`.
- A shorter `VARCHAR`, a different `NUMBER` scale, a different timestamp kind (`NTZ`, `LTZ`, `TZ`) or
  precision, a different `TIME` precision, `FLOAT` to `DECFLOAT`, and any `INTERVAL` change.
- Setting a default on a column without one, or changing a default, except replacing one sequence
  default with another, which Atlas applies in place with `SET DEFAULT`.
- Adding, changing, or removing a generated expression (`as`).

How Atlas handles them depends on the terminal:

- Interactive (stdout is a TTY, and no `CI` variable): Atlas asks, for example
  `Column "status" type change from NUMBER(38) to VARCHAR(16777216) requires recreation`, with the
  choices `Abort` and `Drop and recreate column "status" (data loss)`. Recreating drops and re-adds the
  primary key, unique constraints, and foreign keys on the column.
- Non-interactive (the `CI` variable is set, or stdout is not a TTY): the diff fails with an error
  that ends with `Set 'allow_recreate' in the 'modify_column' block of the diff policy to plan it`.

### Recreating a column: `allow_recreate`

With the policy, Atlas plans the drop and add without asking, in a terminal and in CI
(https://atlasgo.io/hcl/config#diff.snowflake):

```hcl
env "dev" {
  diff "snowflake" {
    modify_column {
      allow_recreate = true
    }
  }
}
```

- The column's data is lost. For data you keep, write the change by hand instead: add a new column,
  backfill it, and drop the old one, each in its own migration file.
- `skip { drop_column = true }` does not stop a recreation.

Identity columns (`identity` in HCL, `AUTOINCREMENT` in SQL) can be dropped from a column, but not
added or changed:

```
adding identity to existing column "id" is not supported
changing identity properties on existing column "id" is not supported
```

Renames have no `renamed_from` attribute on Snowflake:

- A renamed column with the same type and attributes is offered as a rename in a terminal:
  `Did you rename "users" column from "name" to "full_name"`. In CI, or with no terminal, the plan
  drops the old column and adds the new one, and the data is lost. Lint reports the drop (`DS103`).
- A renamed table is always a drop and a create. Write the rename by hand in a migration file
  (`ALTER TABLE "OLD" RENAME TO "NEW"`, `ALTER TABLE ... RENAME COLUMN ... TO ...`), then update the
  schema to match.

## Constraints

- Primary keys, unique constraints, and foreign keys are declared on every table kind that takes them.
  Snowflake enforces them only on hybrid tables; on other tables they are metadata, and `NOT NULL` is the
  only enforced constraint.
- An unnamed `primary_key` or `unique` block gets a generated name, such as `PK_events_id` or
  `UQ_users_email`. Renaming a named constraint is `ALTER TABLE ... RENAME CONSTRAINT`; the
  `rename_constraint` skip policy keeps it out of plans.
- Constraints Snowflake names itself (`SYS_CONSTRAINT_...`) are matched by their columns, so identical
  anonymous constraints on two databases do not show up as changes.

## Views

- A view change is planned as `CREATE OR REPLACE [SECURE] VIEW`. Snowflake drops the view's grants on
  replace; Atlas grants the declared ones again when `permissions = true` (`references/security.md`).
- Inspection stores the body the way Snowflake reports it, with a trailing `;`, and the view's columns.
- `secure = true` makes the view `SECURE`. Changing it replaces the view.
- Views, dynamic tables, functions, procedures, tasks, and policies get their dependencies from their
  bodies. When a body references an object Atlas cannot resolve, add `depends_on`.

## Functions and Procedures

- `lang` is one of `SQL`, `JAVASCRIPT`, `PYTHON`, `JAVA`, or `SCALA`. Functions with the same name
  and different argument types are separate objects, dropped by signature:
  `DROP FUNCTION "add" (NUMBER, NUMBER)`.
- A body change drops and creates the routine, so for a moment it does not exist, and grants on it made
  outside Atlas are lost.
- `imports` blocks name staged files, and `secrets` blocks bind secrets to an alias the handler reads.
  A secret binding needs an external access integration that allows it; when the integration is
  managed outside Atlas, it must allow the dev database's copy too, such as
  `ALLOWED_AUTHENTICATION_SECRETS = ALL`.

## Tasks, Pipes, Alerts, Stages

- A task graph is `schedule` on the root, `after = [task.root]` on children, and
  `finalize = task.root` on the finalizer. `schedule` cannot be combined with `after` or `finalize`,
  and `overlap_policy` is valid only on a root. Atlas creates the root first and drops children and the
  finalizer before it. Snowflake requires the root task to be suspended while the graph is changed.
- A pipe's `as` is the `COPY INTO` statement. Unquoted names in it fold to uppercase, so a table declared
  as `table "PIPE_TARGET"` matches `COPY INTO PIPE_TARGET`.
- A stage's `credentials` are never read back: Atlas keeps the declared ones, and a change to them
  recreates the stage. Use `storage_integration` instead, which cannot be combined with `credentials`.
- File format options that hold Snowflake's defaults are left out of plans and inspection, so
  `SKIP_HEADER = 0` is no change.

## Tags and Policies

- A top-level `tag` block defines the tag; a `tag` block with `ref` and `value` inside an object assigns
  it (`references/hcl.md`). A tag defined in a schema is set on that schema with a separate
  `ALTER SCHEMA ... SET TAG` after the tag exists.
- Policy bodies are resolved by Snowflake when the policy is created, so the tables they read are created
  before them and dropped after them.
- A changed policy body is `ALTER ... POLICY ... SET BODY`. A changed signature (`arg` blocks or
  `return`) recreates the policy. Snowflake refuses to drop a policy that is still attached, so review
  the order of the detach and the drop in the plan before applying.
- Which policy kinds each object takes: tables, dynamic tables, and Iceberg tables take all five; hybrid
  tables take row access and masking; external tables take row access, join, and masking on columns;
  views take all five.

## Secrets

Snowflake never reports a secret's value, so:

- `migrate diff` refuses to write one into a migration file:
  `the SECRET_STRING of secret "S_GENERIC" is written only by an in-place apply`. Manage secrets in a
  declarative env with `atlas schema apply`.
- Inspection writes a placeholder (`"<sensitive>"`) for the value, and the placeholder is not a change
  on the next run. Applying an inspected file where the secret does not exist creates it with the
  literal value `<sensitive>`: replace the placeholder with an input variable first.
- Comparing the value is opt-in with `sensitive = ALLOW` in the `mode` block. Plans print it as
  `(sensitive)`. The stored value is never read back, so every plan sets it again, and the pre-apply
  lint check cannot settle: run such applies with `--skip-lint`.
- Pass the value from an input variable, never as a literal in the file (`references/hcl.md`). A
  secret type that carries a value (`PASSWORD`, `GENERIC_STRING`) with none declared fails with
  `secret "S" requires its SECRET_STRING, which inspection cannot read back`. A
  `CLOUD_PROVIDER_TOKEN` secret needs `api_authentication`, which inspection cannot read either.
- A change Snowflake cannot alter recreates the secret. In a migration file that would lose the value,
  and Atlas refuses with `recreating secret "S" to change ... would drop it and lose its SECRET_STRING`.
- An `OAUTH2` secret whose recreation would drop a refresh token Atlas cannot read back fails with
  `recreating secret "S" to change ... may drop a refresh token that cannot be read back`. Declare
  `oauth_refresh_token` to recreate it.
