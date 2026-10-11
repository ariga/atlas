# Deploy Reference: `migrate/apply`, `migrate/down`, `migrate/set`

The versioned deploy steps. They change a real database, so the agent writes and reviews these
workflows but never triggers a deploy, approves one, or reruns one for the user. Production deploys go
in their own pull request, when the user asks for them.

## `migrate/apply`

Runs `atlas migrate apply` against the target and records the deployment in Atlas Cloud.

```yaml
name: Deploy Migrations
on:
  workflow_dispatch:
    inputs:
      tag:
        description: Commit SHA of the pushed version
        required: true
jobs:
  deploy:
    runs-on: ubuntu-latest
    environment: production                  # required reviewers; deployment branches: the default branch
    steps:
      - uses: ariga/setup-atlas@v0
        with:
          cloud-token: ${{ secrets.ATLAS_CLOUD_TOKEN }}
      - uses: ariga/atlas-action/migrate/apply@v1
        id: apply
        with:
          dir: atlas://app?tag=${{ inputs.tag }}
          url: ${{ secrets.DATABASE_URL }}
```

- Deploy from the registry (`atlas://<repo>?tag=<commit>`): it holds exactly what was reviewed and
  pushed on merge, and needs no checkout. `dir: file://migrations` deploys the checkout instead.
- `dir` takes a migration directory URL with its scheme. A path without one fails with
  `missing scheme for dir url`, and a schema file fails with `"<path>" is not a dir`: a desired schema
  deploys with `schema/apply` (`references/declarative.md`).
- `env` works too: an env whose `migration { dir = "atlas://app" }` and `url = getenv("DATABASE_URL")`
  keeps the inputs short, and carries the checks below.
- A deploy from the registry (`atlas://`) is recorded in Atlas Cloud with its status
  (`atlas cloud migration list`, `atlas/references/cloud.md`). A deploy from a `file://` directory is
  recorded only when the env's `migration` block sets `report = true` and a `repo { name = "..." }`
  (https://atlasgo.io/cloud/deployment). A recorded deploy sends each applied file's contents to Atlas
  Cloud: mark statements that carry secrets with `-- atlas:sensitive`
  (https://atlasgo.io/atlas-schema/sql#sensitive-values).
- Preview and test databases: `skip_report = true` in the env's `migration` block keeps their deploys
  out of the deployment history (https://atlasgo.io/cloud/deployment#skipping-cloud-reporting).
- A directory kept in S3 deploys through a `blob_dir` data source in the env, which needs the login
  (https://atlasgo.io/faq/s3-dirs).
- `dry-run: true` prints the pending SQL without running it; pre-execution checks still run.
- `to-version` or `amount` deploys part of the pending files.

### Outputs and Several Targets

An env with a `for_each` over databases (one database per tenant) runs one apply per target. The
`runs` output is a JSON array with each target's `current`, `target`, `applied_count`,
`pending_count`, and `pending_files`, set only when every target succeeded; the scalar outputs hold the
last target's values. The first failed target fails the step and sets `error`; targets applied before
it stay applied.

### Checks on the Deploy Env

Put the safety checks in the env the deploy uses, not in the workflow:

```hcl
env "prod" {
  url = getenv("DATABASE_URL")
  migration {
    dir      = "atlas://app"
    baseline = "20240115120000"             # only for databases that existed before Atlas
  }
  check "migrate_apply" {
    drift {
      on_error = FAIL                       # stop the deploy if the database drifted
    }
  }
}
```

- The drift check needs the login and reads the expected state from the registry, which keeps it only
  for untagged pushes (`migrate/push` makes one by default with `latest`); a fresh database skips the
  check. `on_error = FAIL` aborts before any file runs; `CONTINUE` reports and deploys. On a database
  that may already have drifted, start with `CONTINUE`, read the drift in the deploy log, then switch
  to `FAIL`. For the full difference, run `migrate/drift` (`references/drift.md`) against the same
  target.
- Pre-execution `deny` rules (batch size, peak hours) and migration hooks:
  `atlas/references/versioned.md`.
- The first deploy to a database that already has objects aborts with
  `connected database is not clean`. Set `baseline` to the version of the baseline file
  (https://atlasgo.io/versioned/apply#existing-databases). Use `allow-dirty` only when the user confirms
  the existing objects stay outside the directory.
- Engines without transactional DDL (ClickHouse, Databricks) need `tx-mode: none`, or
  `migration { tx_mode = none }` in the env.
- Deploys to one database take turns: `migrate/apply` holds a lock while it runs, and another deploy
  waits for it, then fails after `lock_timeout` (`10s` by default). Raise `lock_timeout` in the env's
  `migration` block, or run one deploy per database at a time. Apps that manage their own schemas in
  one database wait on each other unless each env sets its own `lock_name`. Set `skip_lock = true` only
  where something else guarantees a single deploy
  (https://atlasgo.io/versioned/apply#controlling-advisory-locks).
- Out-of-order files and `exec-order`: `references/rebase.md`.

### Promotion

Production advances only to a version a lower environment already runs. In the production env, a
`cloud_databases` data source reads that environment's current version from Atlas Cloud, and
`migration { to_version = ... }` pins production to it; or a pre-execution `deny` check blocks a higher
version. Gate the production job with a GitHub environment that has required reviewers
(`environment: production`). The lower environment's deploys must be recorded, from `atlas://` or with
`report = true`, or `cloud_databases` has no version to read. Guide:
https://atlasgo.io/guides/environment-promotion.

## `migrate/down`

Reverts applied migrations: `amount` files, or back `to-version`, or back `to-tag` (a registry tag).
Atlas plans the revert on the dev database (`dev-url` or the env's `dev`) from the directory, so pass
the registry directory the database was deployed from.

| Plan status | Step |
|-------------|------|
| Applied right away | Passes: outputs `current`, `target`, `planned_count`, `reverted_count` |
| Waiting for approval in Atlas Cloud | Without `wait-timeout`: fails with `plan approval pending, review here: <url>`; rerun after approval. With `wait-timeout`: polls every `wait-interval`, then applies, or warns `plan has not been approved in configured waiting period, exiting` and fails |
| Rejected | Fails with `plan rejected, review here: <url>` |

```yaml
- uses: ariga/atlas-action/migrate/down@v1
  with:
    dir: atlas://app
    url: ${{ secrets.DATABASE_URL }}
    dev-url: docker://postgres/17/dev?search_path=public   # DATABASE_URL names the same schema
    to-version: "20240301000000"
    wait-timeout: 30m
    wait-interval: 30s                      # each check inspects the target again
```

Prefer rolling forward with a new migration. Before a revert drops anything, Atlas runs pre-migration
checks that stop it if the objects to drop hold data (https://atlasgo.io/versioned/down). On engines
without transactional DDL, Atlas reverts one statement at a time, so a revert that failed midway can be
rerun and continues where it stopped.

## `migrate/set`

Sets the revisions table to `version` without running SQL: every version up to and including it counts
as applied, and the rows of later versions are removed. Use it after a change was applied by hand, to
move the recorded version back when later files never ran
(https://atlasgo.io/faq/revisions-table-delete-latest), and in the out-of-order flow
(`references/rebase.md`). On a shared database it rewrites deployment history: only with the user's
explicit approval and a named target.
