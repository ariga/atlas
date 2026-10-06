# Stage 2: Migration Workflow

Goal: the unit's workflow is set up, the user has seen a change go through it, and the project PR is
open. Continue on the stage 1 branch. The steps follow
https://atlasgo.io/guides/evaluation/setup-migrations and
https://atlasgo.io/guides/evaluation/developer-workflow.

## Roles and Permissions (optional)

Atlas leaves roles, users, and permissions (`GRANT` and `REVOKE`) out of inspection and management by
default. If the team wants to manage them as code, enable them with a `mode` block in the env's `schema`
block before the baseline, and repeat the stage 1 export so the desired state includes them
(https://atlasgo.io/atlas-schema/hcl#enabling-roles-and-permissions,
https://atlasgo.io/guides/security-as-code).

Roles and users belong to the database instance, not to one schema, so they cannot be managed in schema
scope. A unit that manages roles uses database scope: its URL and its dev URL name no schema (no
`search_path` in PostgreSQL), and its DDL is schema-qualified (stage 0.3).

The exported schema also refers to roles that exist on the target but not in a new dev database, such
as the provider roles of a managed database. Create them in the dev database's `baseline`, so Atlas can
load the schema there (https://atlasgo.io/concepts/dev-database#baseline-schema, and for managed
PostgreSQL, https://atlasgo.io/guides/postgres/security-declarative#dev-database-for-cloud-environments):

```hcl
docker "postgres" "dev" {
  image    = "postgres:15"        # the target's engine and version; no schema, so database scope
  baseline = <<-SQL
    CREATE ROLE rds_superuser;
    CREATE ROLE rds_iam;
    GRANT rds_superuser TO postgres;
  SQL
}

env "local" {
  url     = getenv("LOCAL_DATABASE_URL")   # database scope: no search_path
  dev     = docker.postgres.dev.url
  exclude = local.exclude
  schema {
    src = local.src
    mode {
      roles       = true                   # roles and users
      permissions = true                   # GRANT and REVOKE
    }
  }
}
```

- With `roles = true`, the desired state is authoritative for every role Atlas inspects on the instance:
  Atlas plans to drop any role the schema does not declare. Declare the roles Atlas must not touch with
  `external = true`: the role Atlas connects with, the admin, replication roles, provider roles such as
  `rds_superuser`, and other teams' roles.
- The dev database must be an instance of its own, such as the container above. On the target's
  instance, its roles would collide with the live ones.
- Declarative units that manage passwords also set `sensitive = ALLOW`
  (https://atlasgo.io/guides/postgres/security-declarative).
- Use the same `mode` block and dev database in every env of the unit, so local development, CI, and
  deployments plan roles and permissions the same way.

## Versioned

### 2.1 Generate the baseline

Add the migration directory to the `local` env:

```hcl
env "local" {
  # url, dev, exclude, and schema as in stage 1
  migration {
    dir = "file://migrations"
  }
}
```

Generate the baseline, the first migration: it captures the schema as it exists today, and every later
migration is a diff on top of it (https://atlasgo.io/versioned/import#generate-a-baseline-migration).
`migrate diff` compares the migration directory, empty so far, with the desired state from stage 1:

```bash
atlas migrate diff --env local baseline
```

This writes `migrations/<version>_baseline.sql` and `migrations/atlas.sum`. The version is the
timestamp at the start of the file name, such as `20250811074144` for `20250811074144_baseline.sql`.

If the user kept differences in stage 1, the baseline must describe the database as it is. Generate it
from an export of the database instead (`--to file://<export dir>`, exported as in stage 1.2 to a
temporary directory), then run `atlas migrate diff --env local <name>`: it writes the kept differences
as the second migration, reviewed like any other.

Verify the directory matches the desired state:

```bash
atlas migrate diff --env local
```

It prints `The migration directory is synced with the desired state, no changes to be made`.

### 2.2 Apply the baseline

How a database starts depends on whether it already has the schema
(https://atlasgo.io/versioned/import#apply-the-baseline-migration):

- New databases run the baseline in full, which creates the schema. Every new local database starts
  this way from now on, so the `local` env never sets a baseline. Show it on a new local database, such
  as a recreated compose database:

  ```bash
  atlas migrate apply --env local
  ```

  A local database that already has the schema, such as the one stage 1 exported, is either recreated
  this way or marked once with `atlas migrate apply --env local --baseline <version>`.
- Existing databases in real environments, such as staging and production, already have the schema and
  must not run the baseline. They start after it, in one of two ways, as the user prefers:
  - `baseline = "<version>"` in the `migration` block of that environment's env, which stages 4 and 5
    write. Every deployment through the env then starts after the baseline, whatever tool runs it.
  - A one-time `atlas migrate apply --env <env> --baseline <version>` on each database, before its first
    deployment. It writes Atlas's history table to the database, so the user runs it, not the agent.

Never set `baseline` in an env that creates new databases: there, Atlas skips the baseline file, and the
new database misses the schema it describes. Without a baseline, the first `migrate apply` on an
existing database stops with `connected database is not clean`.

### 2.3 Show the change workflow

Every schema change from now on is: edit the desired state, `atlas migrate diff --env local <name>`,
review the file, `atlas migrate lint --env local --latest 1`, then open a PR. Developers apply to their
local database with `atlas migrate apply --env local`.

Show what lint catches on a copy of the directory, so the repository is untouched. Pick a real table and
a column that no view, trigger, or index uses; otherwise the replay fails on the dependent object instead
of showing the finding. Say up front that the copy is thrown away:

```bash
tmp=$(mktemp -d) && cp -R migrations "$tmp/dir"
printf 'ALTER TABLE <table> DROP COLUMN <column>;\n' > "$tmp/dir/99999999999999_lint_demo.sql"
atlas migrate hash --env local --dir "file://$tmp/dir"
atlas migrate lint --env local --dir "file://$tmp/dir" --latest 1
rm -rf "$tmp"
```

Lint exits 1 with `DS103` (dropping a column) and `BC104` (clients using it will fail), plus a suggested
fix. Explain each finding in one sentence with its code (https://atlasgo.io/lint/analyzers): CI stops a
PR like this until someone approves it.

### Keeping an existing migration history (only on request)

The baseline is the default. Import a golang-migrate, goose, Flyway, Liquibase, or dbmate history
instead only when the user asks to keep it, and only when it covers exactly this unit's schemas
(https://atlasgo.io/versioned/import):

```bash
atlas migrate import --from "file://<old-dir>?format=flyway" --to "file://migrations"
```

Down and undo files are not imported, and comments that do not directly precede a statement are lost.
Fix the directory before anything else:

1. Order. Atlas runs files in file-name order, and imported versions are not zero-padded: `10_c.sql`
   sorts before `1_a.sql`, and a converted Flyway repeatable `10R_va.sql` sorts before `10_c.sql`.
   Compare `atlas migrate ls --env local` with the old tool's order (`flyway info`). If they differ,
   rename the files to zero-padded versions that keep the old order, and give each repeatable file the
   next free version after the last versioned file.
2. Repeatables run once. Flyway re-runs `R__` files when they change; after the import they are ordinary
   files. Tell the user about any statement that must run on every deploy.
3. Replace `${placeholders}` with their values, and add `-- atlas:txmode none` as the first line of files
   that ran outside a transaction (`CREATE INDEX CONCURRENTLY`).
4. `atlas migrate hash --env local`, then the same `atlas migrate diff --env local` check as above.

## Declarative

### 2.1 No migration directory

Declarative units have no migration files. `atlas.hcl` from stage 1 is complete: Atlas plans each change
against the target database when it is applied.

### 2.2 Show the change workflow

Every schema change from now on is: edit the desired state, then `atlas schema apply --env local`
against the local database. Atlas prints the planned statements, runs the lint analyzers on them, and
asks for approval (https://atlasgo.io/guides/evaluation/developer-workflow). `--dry-run` shows the plan
without applying it. On shared databases, changes go through a reviewed plan instead: CI creates it
with `atlas schema plan` on the PR (stage 3).

## Project PR

Before committing, look for secrets in the exported files and stop on any real one:

```bash
grep -rniE 'password|secret|token|api_?key' schema migrations
```

Open a pull request with the unit's directory and `.atlas-onboarding.json`, staged by path, titled
`Manage the <unit> schema with Atlas`. In the description, paste the stage 1 dry run, the stage 2 checks,
and the lint findings, and say that the current migration tool still deploys until stage 4. Merging is
the user's call.

## Check

1. Versioned: `atlas migrate validate --env local` passes, and `atlas migrate diff --env local` prints
   `The migration directory is synced with the desired state, no changes to be made`.
2. The project PR is open.

Tell the user how a schema change works from now on, in their workflow's terms. Link
https://atlasgo.io/guides/evaluation/developer-workflow.
