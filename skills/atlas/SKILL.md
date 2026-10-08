---
name: atlas
description: "Database schema management, migrations, data scripts, and Atlas Cloud operations with the Atlas CLI. Use when: generating or applying migrations, diffing, linting, validating, or testing schemas, working with atlas.hcl, schema.hcl, or ORM schemas (GORM, Drizzle, SQLAlchemy, Django, Ent, Sequelize, TypeORM, Prisma), running data backfills, purges, or reports with atlas script, or answering questions about Atlas Cloud: which databases are synced, pending, or failed, recent or failed deployments, what is waiting for deployment, registry repos, and pending plan approvals."
allowed-tools: Bash(atlas version) Bash(atlas whoami) Bash(atlas cloud repo list:*) Bash(atlas cloud repo describe:*) Bash(atlas cloud database list:*) Bash(atlas cloud database describe:*) Bash(atlas cloud migration list:*) Bash(atlas cloud migration describe:*) Bash(atlas schema inspect:*) Bash(atlas schema validate:*) Bash(atlas schema diff:*) Bash(atlas schema lint:*) Bash(atlas schema plan list:*) Bash(atlas migrate status:*) Bash(atlas migrate lint:*) Bash(atlas migrate diff:*) Bash(atlas migrate hash:*) Bash(atlas migrate ls:*) Bash(atlas migrate validate:*) Bash(atlas migrate drift:*) Bash(atlas migrate test:*) Bash(atlas schema test:*) Bash(atlas schema plan test:*) Bash(atlas script test:*)
---

# Atlas: Schema Migrations, Data Scripts, and Cloud Operations

## Files

This skill is `SKILL.md` plus nine reference files under `references/`, each served at
`https://atlasgo.io/skills/atlas/references/<name>`: `schema-sources.md`, `versioned.md`, `declarative.md`,
`cicd.md`, `lint.md`, `testing.md`, `drift.md`, `cloud.md`, and `scripts.md`. Read a reference only when
the task needs it; "Choosing a Workflow" below says which.

## Security

Never hardcode credentials. Use environment variables in `atlas.hcl`:

```hcl
env "prod" {
  url = getenv("DATABASE_URL")
}
```

Always run commands with `--env <name>` so database URLs stay in `atlas.hcl` and never enter the
conversation. Read `atlas.hcl` first to learn the environment names.

## Quick Reference

Use `--help` on any command for full docs and examples: `atlas migrate diff --help`.

```bash
# Setup
atlas version                                        # Installed version
atlas whoami                                         # Login status and org
atlas login                                          # Needed by lint, test, drift, plan, scripts, schema lint, and cloud

# Schema
atlas schema inspect --env <name>                    # Inspect current schema
atlas schema validate --env <name>                   # Validate schema syntax/semantics
atlas schema diff --env <name> --from env://url --to file://schema.hcl   # Live database vs desired schema
atlas schema lint --env <name>                       # Check schema policies
atlas schema test --env <name>                       # Test schema logic

# Declarative workflow
atlas schema plan --env <name>                       # Pre-plan changes for review
atlas schema apply --env <name> --dry-run            # Preview changes
atlas schema apply --env <name>                      # Apply schema changes

# Versioned workflow
atlas migrate diff --env <name> "migration_name"     # Generate migration
atlas migrate lint --env <name> --latest 1           # Lint the newest migration
atlas migrate test --env <name>                      # Test migrations
atlas migrate apply --env <name> --dry-run           # Preview changes
atlas migrate apply --env <name>                     # Apply migration
atlas migrate status --env <name>                    # Live database vs migration directory
atlas migrate down --env <name> --dry-run            # Preview reverting the last migration
atlas migrate hash --env <name>                      # Recompute atlas.sum after manual edits
atlas migrate new --env <name> "name"                # Empty migration file for hand-written SQL
atlas migrate checkpoint --env <name>                # Squash history into a checkpoint file
atlas migrate rebase --env <name> <version>          # Rebase a migration onto newer files
atlas migrate push --env <name>                      # Push the directory to the Atlas Registry

# Linting and testing (see references/lint.md, references/testing.md)
atlas migrate lint --env ci                          # CI: new files vs the registry; --git-base master only without a registry
atlas schema lint --env <name>                       # Lint the whole schema against policy
atlas schema test --env <name> --run <case>          # Schema tests: functions, views, triggers
atlas schema plan test --env <name>                  # Test a declarative plan file

# Drift (see references/drift.md)
atlas migrate drift --env <name>                     # Database vs migration history, exit 1 on drift (login)
atlas schema diff --from <url> --to <url> --dev-url <dev>   # Any two states

# Atlas Cloud (see references/cloud.md)
atlas cloud repo list                                # Registry repos with synced/failed/pending counts
atlas cloud database list --env-name <env>           # Every tracked database and its status
atlas cloud migration list --status FAILED           # Failed deployments
atlas schema plan list --env <name> --pending        # Plans waiting for approval

# Data Scripts (see references/scripts.md)
atlas script exec  --env <name> --run '^name$'       # Transactional mutation
atlas script query --env <name> --run '^name$' -q    # Read or report
atlas script loop  --env <name> --run '^name$'       # Batched backfill or purge
atlas script test  --env <name>                      # Test scripts on the dev database
```

## Choosing a Workflow

```
What is the request?
├─ A schema change
│  ├─ Project has migrations/ dir or a migration block in atlas.hcl?
│  │  ├─ Yes → Versioned: migrate diff → lint → test → apply   (references/versioned.md)
│  │  └─ No  → Declarative: schema apply --dry-run → apply     (references/declarative.md)
│  ├─ Change must be reviewed and approved before it runs?
│  │  └─ schema plan (declarative) or a PR with migrate lint (versioned)
│  ├─ Iterating on a local database?
│  │  └─ schema apply --auto-approve for fast edit-apply cycles
│  └─ Not sure → Read atlas.hcl first
├─ CI/CD: lint on PRs, push to the registry, deploy from it
│  └─ references/cicd.md
├─ A data change (backfill, purge, report, invariant check)
│  └─ Data Scripts: references/scripts.md
├─ A lint failure, a lint policy, or a custom rule
│  └─ references/lint.md
├─ Tests for functions, views, triggers, data migrations, or plans
│  └─ references/testing.md
├─ "Has this database drifted?" or a drift check for deploys
│  └─ references/drift.md
├─ A question about deployments, database status, or the registry
│  └─ Atlas Cloud: references/cloud.md
└─ ORM or schema source setup
   └─ references/schema-sources.md
```

`atlas schema apply` applies schema changes directly to a database without migration files. Use it for
fast iteration during development: edit the schema, run `schema apply`, see the result.

## Example

Versioned project (a `migration` block in `atlas.hcl`):

```
User: Add an email column to the users table

Agent steps:
1. atlas schema inspect --env dev           # understand current state
2. Edit schema source file                  # add email column
3. atlas schema validate --env dev          # verify syntax
4. atlas migrate diff --env dev "add_users_email"  # generate migration
5. atlas migrate lint --env dev --latest 1  # check for issues
6. atlas migrate apply --env dev --dry-run  # preview before applying
```

Declarative project (a `schema` block, no migration directory):

```
Agent steps:
1. Edit schema source file                  # add email column
2. atlas schema validate --env dev          # verify syntax
3. atlas schema apply --env dev --dry-run   # read the planned SQL to the user
4. atlas schema apply --env dev             # apply (or --auto-approve on a local database)
```

## Core Concepts

### Configuration File (atlas.hcl)

Always read the project's `atlas.hcl` first. It contains the environment configurations:

```hcl
env "<name>" {
  url = getenv("DATABASE_URL")
  dev = "docker://postgres/17/dev?search_path=public"

  migration {
    dir = "file://migrations"       # or "atlas://<repo>" for the Atlas Registry
  }

  schema {
    src = "file://schema.hcl"
  }

  script {
    src = "file://scripts"          # Data Scripts source
  }
}
```

### Dev Database

Atlas uses a temporary dev database to process and validate schemas. The URL scope must match the
target: schema-scoped when the project manages one schema, database-scoped when it manages several
schemas, extensions, or event triggers.

```bash
# Schema-scoped (single schema, most common)
--dev-url "docker://mysql/8/dev"
--dev-url "docker://postgres/17/dev?search_path=public"
--dev-url "sqlite://dev?mode=memory"
--dev-url "docker://sqlserver/2022-latest/dev?mode=schema"

# Database-scoped (multiple schemas, extensions, or event triggers)
--dev-url "docker://mysql/8"
--dev-url "docker://postgres/17/dev"
--dev-url "docker://sqlserver/2022-latest/dev?mode=database"
```

Using the wrong scope causes errors such as
`modify schema "public" is not allowed when migration plan is scoped to one schema`, or silently drops database-level
objects (extensions, event triggers) from migrations. For PostGIS or pgvector schemas, use
`docker://postgis/latest/dev` or `docker://pgvector/pg17/dev`.

If the schema depends on extensions or external objects, use a `docker` block with a `baseline`:

```hcl
docker "postgres" "dev" {
  image  = "postgres:17"
  schema = "public"
  baseline = <<SQL
   CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
  SQL
}

env "local" {
  src = "file://schema.hcl"
  dev = docker.postgres.dev.url
}
```

## Workflows

### 1. Schema Inspection

Start with an overview before diving into details. The default output is HCL. Use
`--format "{{ json . }}"` for JSON or `--format "{{ sql . }}"` for SQL.

```bash
# List tables (overview first, JSON output)
atlas schema inspect --env <name> --format "{{ json . }}" | jq ".schemas[].tables[].name"

# Full SQL schema
atlas schema inspect --env <name> --format "{{ sql . }}"

# Filter with --include/--exclude (useful for large schemas)
atlas schema inspect --env <name> --include "users_*"           # Only matching tables (needs atlas login)
atlas schema inspect --env <name> --exclude "*_backup"          # Skip matching tables
atlas schema inspect --env <name> --exclude "*.*[type=trigger]" # Skip triggers (database scope: "*.*.*[type=trigger]")

# Open a visual ERD in the browser (requires atlas login)
atlas schema inspect --env <name> -w
```

### 2. Schema Comparison (Diff)

```bash
# Live database vs desired schema. --from and --to are always required; --env supplies the dev database.
# env://url reads the env's url so the database URL never appears on the command line.
# On a versioned database add --exclude atlas_schema_revisions, or the revisions table shows up as a drop.
atlas schema diff --env <name> --from env://url --to file://schema.hcl --exclude atlas_schema_revisions

# Compare specific sources
atlas schema diff --env <name> --from file://migrations --to file://schema.hcl
```

### 3. Migration Generation

```bash
# Generate a migration from the schema diff
atlas migrate diff --env <name> "add_users_table"

# With explicit parameters
atlas migrate diff \
  --dir file://migrations \
  --dev-url docker://postgres/17/dev \
  --to file://schema.hcl \
  "add_users_table"
```

### 4. Schema Validation

Validate schema definitions after every edit and before generating migrations:

```bash
atlas schema validate --env <name>
atlas schema validate --dev-url docker://postgres/17/dev --url file://schema.hcl
```

If valid, the command exits successfully. If invalid, it prints the error (unresolved references,
syntax issues, unsupported attributes).

### 5. Migration Linting

```bash
atlas migrate lint --env <name> --latest 1    # Lint the latest migration
atlas migrate lint --env ci                   # CI: new files vs the registry (no registry: --git-base master)
atlas schema lint --env <name>                # Check the whole schema against policy
```

The analyzers catch destructive changes (`DS*`), data-dependent changes (`MF*`),
backward-incompatible changes (`BC*`), table locks and rewrites (`MY*`, `PG*`), naming and ownership
policy, and custom rules written in HCL. Policy lives in the `lint` block of `atlas.hcl`.

Fixing lint issues:
- Unapplied migrations: edit the file, then run `atlas migrate hash --env <name>`.
- Applied migrations: create a corrective migration. Never edit an applied migration.
- `-- atlas:nolint <code>` only with the user's approval.

Codes, policy configuration, and custom rules: `references/lint.md`.

### 6. Testing

```bash
atlas schema test --env <name>                # Functions, views, triggers, constraints, queries
atlas migrate test --env <name>               # Data migrations between versions
atlas schema plan test --env <name>           # Declarative plan files
atlas script test --env <name>                # Data Scripts
```

Tests are HCL cases (`test "schema" "<name>" { exec, catch, assert, ... }`) run against the dev
database. Write a test with every function, view, trigger, and data migration you add. Block syntax,
env configuration, and the fix loop: `references/testing.md`.

### 7. Applying Migrations

```bash
atlas migrate apply --env <name> --dry-run    # Always preview first
atlas migrate apply --env <name>              # Apply
atlas migrate status --env <name>             # Verify
```

Before applying to a shared or production environment, check Atlas Cloud state first (failed or
in-flight deployments on the same targets). See the pre-deployment checklist in `references/cloud.md`.

### 8. Declarative Plan and Approval

In the declarative workflow, `atlas schema plan` saves a migration plan to the Atlas Registry so it
can be reviewed and approved before `atlas schema apply` runs it:

```bash
atlas schema plan --env <name>                        # Plan, lint, review, approve and push
atlas schema plan --env <name> --pending              # Push for someone else to approve
atlas schema plan list --env <name> --pending         # Plans waiting for approval
atlas schema plan approve --url "atlas://<repo>/plans/<name>"
atlas schema apply --env <name>                       # Applies the approved plan as-is
```

Plan editing, `push`, `pull`, `lint`, `validate`, the review policy, and `schema push`:
`references/declarative.md`. Migration directory formats, the diff policy, baselines, `down`,
`checkpoint`, `rebase`, pre-execution checks, and multi-tenant apply: `references/versioned.md`.

### 9. CI/CD

Lint (or plan) on pull requests, push the directory or schema to the Atlas Registry on merge, deploy
from `atlas://<repo>` with drift and pre-execution checks. Workflows for GitHub Actions and the
equivalents for GitLab, CircleCI, Bitbucket, Azure DevOps, Kubernetes, and Terraform:
`references/cicd.md`.

## Standard Workflow

1. `atlas schema inspect --env <name>`: understand the current state (skip on an empty database)
2. Edit schema files
3. `atlas schema validate --env <name>`: check syntax
4. `atlas migrate diff --env <name> "change_name"`: generate the migration
5. `atlas migrate lint --env <name> --latest 1`: validate (requires login)
6. `atlas migrate test --env <name>`: test (requires login)
7. If issues: edit the migration, then `atlas migrate hash`
8. `atlas migrate apply --env <name> --dry-run`, then apply

Run `atlas whoami` before inspecting, diffing, or baselining an existing database, and ask the user to
log in if it fails. Logged out, `atlas schema inspect` and `atlas schema diff` skip views, functions,
procedures, triggers, sequences, extensions, and other objects and still exit 0, so their output looks
complete when it is not.

Other commands that need a login fail with `command requires 'atlas login'` or `available only to Atlas
Pro users`. Tell the user that step was skipped and why, and continue with the remaining steps.

## Drift Detection

Drift is a difference between a database and its source of truth. Four tools, one per question:

| Question | Tool |
|----------|------|
| Does this database match its migration history, now or on a schedule? | `atlas migrate drift --env <name>` (exit 1 on drift, cron-friendly; requires `atlas login`) |
| Block a deploy if the target drifted | `check "migrate_apply" { drift { on_error = FAIL } }` in `atlas.hcl` |
| How do two states differ (declarative, or database vs database)? | `atlas schema diff --from <url> --to <url>` (both flags required) |
| Agent-based continuous monitoring with alerts | Schema Monitoring in Atlas Cloud |

Commands, flags, `exclude` patterns, and the reporting workflow: `references/drift.md`.

## Atlas Cloud

`atlas cloud` commands report what Atlas Cloud knows: registry repos, every tracked database with its
sync status (`SYNCED`, `PENDING`, `FAILED`) and current version, and every deployment event
(`PASSED`, `FAILED`, `NO_ACTION`, `DRY_RUN`). Use them to answer operational questions:

| Question | Command |
|----------|---------|
| Summarize our database infrastructure | `atlas cloud repo list`, then `atlas cloud database list` per env |
| Which deployments failed? | `atlas cloud migration list --status FAILED` |
| What happened in a deployment? | `atlas cloud migration describe --id <id>` |
| Which databases are waiting for a deployment? | `atlas cloud database list` and filter `Status=PENDING` |
| Which plans are waiting for approval? | `atlas schema plan list --env <name> --pending` |
| What version is production on? | `atlas cloud database list --env-name prod` |

Full command reference, report templates, and decision rules: `references/cloud.md`.

## Data Scripts

`atlas script` runs data operations written as HCL: `exec` for transactional mutations, `query` for
reads and reports, and `loop` for batched backfills and purges, with `condition` guards, `assert`
checks, `expect_rows`, output masking, and tests. Use a script for data work and a migration for schema
work. Full reference: `references/scripts.md`.

## Schema Sources

For HCL schemas, ORM integrations (GORM, Drizzle, SQLAlchemy, Django, Ent, Sequelize, TypeORM),
composite schemas, and dev-database dialect URLs, see `references/schema-sources.md`.

## Onboarding an Existing Project

For a guided onboarding of a whole project or team (inventory, splitting a large database into units,
CI, and deployment, each stage verified), the user can run the `atlas-onboard` skill
(https://atlasgo.io/guides/ai-tools/agent-skills#onboarding-skill).

To start managing an existing database with versioned migrations, log in first: a logged-out inspect
leaves views, functions, triggers, and other objects out of the baseline without an error. Then pin the
org in `atlas.hcl`, so any later command that reads it (`--env`) aborts unless someone is logged in to
that org. Commands that take `--url` directly do not read it, so run `atlas whoami` before them:

```hcl
atlas {
  cloud {
    org = "<org printed by atlas whoami>"
  }
}
```

```bash
# 1. Export the current schema to code
atlas schema inspect --env <name> --format '{{ sql . | split | write "src" }}'

# 2. Generate a baseline migration from the exported schema
atlas migrate diff --env <name> "baseline" --to "file://src"

# 3. Record the baseline as applied on each existing database (the version from the filename).
#    Deploy envs can set baseline = "<version>" in their migration block instead.
#    Preview first; --dry-run writes nothing:
atlas migrate apply --env <name> --baseline '<version>' --dry-run
#    Then, only after the user approves the named target, record it (this writes the revisions table):
atlas migrate apply --env <name> --baseline '<version>'
```

The baseline migration captures the current state without executing it on existing databases. On new
databases, it runs in full to create the initial schema.

## Troubleshooting

```bash
atlas version                                 # Check installation
atlas whoami                                  # Check login and org
atlas migrate hash --env <name>               # Repair migration integrity after manual edits
```

- `command requires 'atlas login'` or `available only to Atlas Pro users`: the command needs a
  login. Run `atlas login`, or `atlas login --token "$ATLAS_TOKEN"` in CI. Report the skipped step and
  continue.
- `'atlas login' is required for organization <org> as specified in atlas.hcl`: the project pins its
  org. Ask the user to run `atlas login` with an account in that org.
- Missing driver error: ensure `--url` or `--dev-url` is correctly specified.
- `add`, `drop`, or `modify schema "<name>" is not allowed when migration plan is scoped to one schema`:
  the dev URL scope does not match the target. See Dev Database above.

## Key Rules

1. Read `atlas.hcl` first and use the environment names from it.
2. Never hardcode credentials. Use `getenv()` and `--env`.
3. Run `atlas schema validate` after schema edits.
4. Always lint before applying migrations.
5. Always dry-run before applying.
6. Run `atlas migrate hash` after editing migration files.
7. Never edit an applied migration. Create a corrective migration instead.
8. Never ignore lint errors. Fix them or get explicit user approval.
9. Run `atlas login` once per machine. Linting, testing, drift, `schema plan`, the review policy,
    scripts, ERD, `migrate checkpoint`, `schema inspect --include`, and Atlas Cloud need it, and fail
    without it: report the skipped step and continue.
    Inspecting and diffing a database do not fail without it; they skip objects. Log in before either.
10. Before deploying to shared environments, check Atlas Cloud for `FAILED` or `PENDING` targets
    and recent failed deployments (`references/cloud.md`).
11. Use Data Scripts, not ad-hoc SQL, for data changes: guard with `condition`, assert the outcome,
    and mask PII in reports (`references/scripts.md`).
12. Report cloud state and live database state separately. `atlas cloud` says what Atlas Cloud
    knows; `atlas migrate status` says what is in the database.

## Documentation

- [CLI Reference](https://atlasgo.io/cli-reference)
- [Versioned Migrations](https://atlasgo.io/versioned/diff)
- [Declarative Workflow](https://atlasgo.io/declarative/apply)
- [Declarative Plan and Approval](https://atlasgo.io/declarative/plan)
- [Versioned CI/CD Setup](https://atlasgo.io/versioned/setup-cicd)
- [Declarative CI/CD Setup](https://atlasgo.io/declarative/setup-cicd)
- [GitHub Actions](https://atlasgo.io/integrations/github-actions)
- [Migration Linting](https://atlasgo.io/versioned/lint)
- [Lint Analyzers](https://atlasgo.io/lint/analyzers)
- [Custom Lint Rules](https://atlasgo.io/lint/rules)
- [Schema Testing](https://atlasgo.io/testing/schema)
- [Migration Testing](https://atlasgo.io/testing/migrate)
- [Drift Detection](https://atlasgo.io/versioned/drift-detection)
- [Schema Monitoring](https://atlasgo.io/monitoring)
- [Data Scripts](https://atlasgo.io/scripts)
- [Atlas Cloud Deployments](https://atlasgo.io/cloud/deployment)
- [Onboard Existing Database](https://atlasgo.io/versioned/import)
- [ORM Integrations](https://atlasgo.io/orms)
- [Dev Database](https://atlasgo.io/concepts/dev-database)
