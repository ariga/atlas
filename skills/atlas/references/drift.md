# Drift Detection Reference

Schema drift is a difference between what a database actually contains and what its source of truth
says it should contain: the migration directory at the applied version, or the desired schema in
the declarative workflow. Drift comes from manual hotfixes, out-of-band tools, and partially failed
deployments. Atlas detects it in four places. Pick the one that matches the question.

| Question | Tool | Docs |
|----------|------|------|
| Does this database match its migration history, now or on a schedule (cron, CI job)? | `atlas migrate drift` (requires `atlas login`) | https://atlasgo.io/versioned/drift-detection#migrate-drift |
| Block a deployment if the target has drifted | `check "migrate_apply" { drift {} }` in `atlas.hcl` | https://atlasgo.io/versioned/drift-detection#pre-apply-check |
| How does this database differ from the desired schema (declarative) or from another database? | `atlas schema diff` | https://atlasgo.io/declarative/diff |
| Agent-based continuous monitoring of every database, alerts, and the diff in a UI | Schema Monitoring in Atlas Cloud | https://atlasgo.io/monitoring/drift-detection |
| Compliance framing (SOC 2, change management) | Drift detection guide | https://atlasgo.io/guides/drift-detection |

The declarative workflow is self-correcting: `atlas schema apply` plans from the live state every time,
so drift is folded into the next apply. The versioned workflow assumes the database is exactly where the
last migration left it, which is why the checks below exist.

## `atlas migrate drift` (versioned, on demand or scheduled)

Requires `atlas login`. Reads the revisions table of the connected database, resolves the state the migration directory defines
at the last applied version, inspects the database, and diffs the two. Files after the applied version
are pending, not drift, and are reported as ignored. The command never changes the database, and it
takes the same advisory lock as `migrate apply`, so an in-flight deployment is not reported as drift.

```bash
atlas migrate drift --env prod                                   # registry mode: dir = atlas://<repo>
atlas migrate drift --url "$DATABASE_URL" --dir file://migrations --dev-url docker://postgres/17/dev
atlas migrate drift --env prod --format '{{ json . }}'
```

Expected state comes from the Atlas Registry when `migration.dir` is an `atlas://` URL (or
`migration.repo.name` is set). Otherwise pass `--dev-url` and Atlas computes it from the local directory.

```
Drift Status: OK
  -- Current Version: 20260423120000
  -- Expected State:  atlas://my-app (registry)
  -- Pending Files:   0
```

Exit codes: `0` no drift, `1` drift found (the report lists the objects and the SQL that would
reconcile them). This is the background check: run it from a cron job, a scheduled CI workflow, or a
Kubernetes CronJob and it fails on drift without parsing output, so the same command serves an ad hoc
question and continuous checking. On an env with
`for_each` (multi-tenant), one report is printed per target and the run fails if any target drifted.

| Flag | Env attribute | Notes |
|------|---------------|-------|
| `--url` | `url` | Required |
| `--dev-url` | `dev` | Required for a local directory |
| `--dir` | `migration.dir` | Defaults to `file://migrations` |
| `--exclude` | `migration.exclude`, else `exclude` | Objects to ignore, see Excluding Objects |
| `--format` | `format.migrate.drift` | Go template for the report |
| `--no-cache` | | Bypass the expected-state cache |
| `--lock-timeout`, `--lock-name`, `--skip-lock` | `migration.*` | Advisory lock shared with `migrate apply` |

## Pre-Apply Check (versioned, at deploy time)

Runs at the start of every `atlas migrate apply` for the env and aborts before any file runs if the
target has drifted from the expected state at its current revision. Requires the directory to be
pushed to the Atlas Registry and at least one applied revision (a fresh database skips the check).

The registry stores the expected state only for untagged pushes, such as the `latest` push that the
`migrate/push` action makes by default. If the applied version was pushed only with a tag
(`atlas migrate push <repo>:<tag>`), the check prints `no state found for version ...` and is skipped,
even with `on_error = FAIL`. Push the version untagged as well.

```hcl
env "prod" {
  url = getenv("DATABASE_URL")
  migration {
    dir = "atlas://my-app"
  }
  check "migrate_apply" {
    drift {
      on_error = FAIL            # or CONTINUE: report and keep deploying
      exclude  = ["audit_log", "monitoring_*"]
    }
  }
}
```

Rollout on an existing environment: start with `on_error = CONTINUE`, read the reported diff, add a
migration for unintentional drift, add `exclude` patterns for intentional objects, then switch to `FAIL`.

## Excluding Objects

Objects that intentionally live outside the migration scope (manually installed extensions, audit or
sidecar tables owned by another service, schemas owned by another team) would be reported on every
run. Exclude them with glob patterns. The pattern format follows the URL scope of `env.url`:

```hcl
# Schema scope (URL names one schema): patterns match objects in that schema
exclude = ["audit_log", "monitoring_*", "*[type=policy|function]"]

# Database scope (URL covers the database): qualify with the schema
exclude = ["audit.*", "public.monitoring_*", "*.*[type=trigger]"]
```

The revisions table and its schema are excluded automatically by `migrate drift` and the pre-apply
check. `schema diff` does not exclude them: pass `--exclude atlas_schema_revisions`.

## `atlas schema diff` (any workflow, ad hoc)

Compares any two states: a live database, a migration directory, an HCL or SQL file, or an ORM. Use it
to answer "what would it take to make A look like B", including between two databases.

```bash
atlas schema diff --env <name> --from env://url --to file://schema.hcl --exclude atlas_schema_revisions
atlas schema diff --from "$PROD_URL" --to "$STAGING_URL" --dev-url docker://postgres/17/dev
atlas schema diff --from file://migrations --to file://schema.hcl --dev-url docker://postgres/17/dev
```

The output is the SQL that turns `--from` into `--to`; "Schemas are synced, no changes to be made" means
no drift. `--from env://url` reads the env's `url` so the database URL is never typed. `--to` is the
env's schema source (`schema.src` in `atlas.hcl`). On a versioned database add
`--exclude atlas_schema_revisions`, or the revisions table is reported as a table to drop.

## Schema Monitoring (Atlas Cloud, agent-based)

Atlas Cloud Schema Monitoring inspects databases on a schedule through a lightweight agent and reports
drift against a deployed migration target or against another database's snapshot. It shows the diff as
ERD, HCL, and SQL, and notifies Slack or a webhook when drift appears. It is set up in the Atlas Cloud
UI, not from the CLI: https://atlasgo.io/monitoring/drift-detection. Use it when the user wants an agent
watching many databases with alerts, rather than a scheduled `migrate drift` job they operate themselves.

`atlas cloud database list` (see `references/cloud.md`) reports deployment status (`SYNCED`,
`PENDING`, `FAILED`), which is a different question from drift: a database can be `SYNCED` at its
version and still have drifted objects.

## Agent Workflow: "Has this database drifted?"

1. Read `atlas.hcl`: versioned (`migration` block) or declarative (`schema` block)?
2. Versioned: `atlas migrate drift --env <name>`. Report the status, the current version, the pending
   file count, and the reconciling SQL if drift was found.
3. Declarative: `atlas schema apply --env <name> --dry-run` and report the SQL Atlas would apply
   ("Schema is synced, no changes to be made" means no drift). Without a login the lint section of that
   output is redacted ("Run atlas login ... to view diagnostics"); say so rather than reporting "no issues".
   `atlas schema diff --env <name> --from env://url --to <schema.src>` is the same answer as plain SQL.
4. If drift is real, propose either a corrective migration (`atlas migrate diff` after fixing the
   schema source) or, for intentional objects, an `exclude` entry. Do not run `atlas schema apply`
   or `atlas migrate apply` to "fix" drift without the user's approval.
5. If the user wants this checked continuously, suggest `atlas migrate drift` on a schedule (cron or
   CI) for versioned projects, the pre-apply check for deploys, and Schema Monitoring for agent-based
   monitoring with alerts.

## Documentation

- [Drift detection for versioned migrations](https://atlasgo.io/versioned/drift-detection)
- [Schema diff](https://atlasgo.io/declarative/diff)
- [Schema Monitoring drift detection](https://atlasgo.io/monitoring/drift-detection)
- [Drift detection and compliance](https://atlasgo.io/guides/drift-detection)
- [Pre-execution checks in migrate apply](https://atlasgo.io/versioned/apply#pre-execution-checks)
