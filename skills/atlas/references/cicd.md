# CI/CD Reference

Atlas ships first-party CI integrations for GitHub Actions, GitLab CI, CircleCI, Bitbucket Pipelines,
and Azure DevOps, plus deployers for Kubernetes (operator, Helm, ArgoCD, Flux), Terraform, and plain
pipelines. The shape is the same everywhere: on a pull request, lint (or plan) and comment the report;
on merge, push the artifact to the Atlas Registry; on deploy, apply from the registry.

Every CI step needs an Atlas Cloud bot token in a secret, created by an org admin in Atlas Cloud under
Settings > Bots, and an `atlas.hcl` committed to the repo with an env for CI (dev database, lint policy).
The docs name the secret `ATLAS_CLOUD_TOKEN` on GitHub Actions and GitLab CI, and `ATLAS_TOKEN` on
CircleCI, Bitbucket, and Azure DevOps; the CLI also reads `ATLAS_TOKEN` from the environment. If the
repo already has a token secret, use its name.

How `migrate lint` decides which files are "new" depends on whether the directory is pushed to the
Atlas Registry. Read `atlas.hcl` and pick the matching case:

```hcl
# Case 1 (most projects): the directory is pushed to the registry on merge.
# Lint compares the PR's directory with the latest pushed state. No git settings needed.
env "ci" {
  dev = "docker://postgres/17/dev?search_path=public"   # or a service container URL
  migration {
    dir = "file://migrations"
    repo {
      name = "app"                                      # the registry repo
    }
  }
}

# Case 2: no registry. Lint compares the PR branch with the base branch in git.
env "ci" {
  dev = "docker://postgres/17/dev?search_path=public"
  migration {
    dir = "file://migrations"
  }
  lint {
    git {
      base = "master"
    }
  }
}
```

With a registry repo configured, do not add `git { base }` or `--git-base`: the registry comparison is
what makes lint independent of branch history and what the drift checks read from.

The registry repo must exist before the first CI run. Create it once, after the user approves the name:
from CI with the `create-repo` action, or with the first `atlas migrate push` / `atlas schema push`.

```yaml
# .github/workflows/atlas-create-repo.yaml: run once, from the Actions tab
name: Create Atlas Registry Repo
on:
  workflow_dispatch:
jobs:
  create-repo:
    runs-on: ubuntu-latest
    steps:
      - uses: ariga/setup-atlas@v0
        with:
          cloud-token: ${{ secrets.ATLAS_CLOUD_TOKEN }}
      - uses: ariga/atlas-action/create-repo@v1
        with:
          name: app
          type: migration_directory      # or schema
          driver: postgres
```

`type` takes `migration_directory` (or `m`) or `schema` (or `s`); the CLI rejects `migration`. The
action fails when the repo already exists, so run it once, not on every push. Its `url` output links to
the repo in Atlas Cloud.

Keep database URLs out of the workflow files. Declare them as variables that read the CI environment,
and set the secrets on the job:

```hcl
variable "url" {
  type    = string
  default = getenv("DATABASE_URL")      # target database, from a CI secret
}

variable "dev_url" {
  type    = string
  default = getenv("DEV_URL")           # dev database: a docker:// URL or a service container
}

env "ci" {
  url = var.url
  dev = var.dev_url
  schema {
    src = "file://schema.sql"           # declarative source
    repo {
      name = "app"
    }
  }
  migration {
    dir = "file://migrations"           # versioned directory; one env can serve both workflows
  }
}
```

Engines without transactional DDL (ClickHouse, Databricks) need `tx-mode: none` on the apply steps,
or `migration { tx_mode = none }` in the env, so Atlas does not wrap files in a transaction the engine
cannot roll back.

The docs are the reference for every action and input (links at the end). Two runnable repositories
show both pipelines end to end as additional examples: ClickHouse, with the databases started by
`docker compose` in the job and their URLs loaded into `GITHUB_ENV`
(https://github.com/atlasdemos/clickhouse/tree/main/.github/workflows), and Databricks, with URLs
from secrets and `tx-mode: none` (https://github.com/atlasdemos/databricks/tree/main/.github/workflows).

## Versioned Pipeline

| Stage | Trigger | What runs | GitHub Action |
|-------|---------|-----------|---------------|
| Lint | pull request | Replays the files not yet in the registry (or, without a registry, the files added since the base branch) on a dev database, comments findings on the PR, fails on errors | `ariga/atlas-action/migrate/lint@v1` |
| Diff (optional) | pull request | Plans migrations from the schema source and commits them to the PR when the developer did not | `ariga/atlas-action/migrate/diff@v1` |
| Test (optional) | pull request | `migrate test` and `schema test` on the dev database, results on the PR | `ariga/atlas-action/migrate/test@v1`, `schema/test@v1` |
| Autorebase | push to a non-default branch | Moves the branch's migration files after the ones on the target branch and updates `atlas.sum` | `ariga/atlas-action/migrate/autorebase@v1` |
| Push | merge to main | Pushes the directory to the registry, tagged with the commit and `latest` | `ariga/atlas-action/migrate/push@v1` |
| Deploy | release / manual / GitOps | `migrate apply` from `atlas://app?tag=...` against the target | `ariga/atlas-action/migrate/apply@v1`, operator, Terraform |

```yaml
# .github/workflows/atlas-ci.yaml
name: Atlas CI
on:
  push:
    branches: [master]
  pull_request:
    paths: ['migrations/*', 'schema.sql', 'atlas.hcl']
permissions:
  contents: read
  pull-requests: write
jobs:
  lint:
    if: github.event_name == 'pull_request'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0                       # only needed without a registry (--git-base)
      - uses: ariga/setup-atlas@v0
        with:
          cloud-token: ${{ secrets.ATLAS_CLOUD_TOKEN }}
      - uses: ariga/atlas-action/migrate/lint@v1
        with:
          env: ci
        env:
          GITHUB_TOKEN: ${{ github.token }}    # lets the action comment on the PR
      - uses: ariga/atlas-action/migrate/test@v1
        with:
          env: ci
  push:
    if: github.event_name == 'push'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: ariga/setup-atlas@v0
        with:
          cloud-token: ${{ secrets.ATLAS_CLOUD_TOKEN }}
      - uses: ariga/atlas-action/migrate/push@v1
        with:
          env: ci
          dir-name: app
      # Small projects deploy in the same job, right after the push, as in the atlasdemos example:
      # - uses: ariga/atlas-action/migrate/apply@v1
      #   with:
      #     env: ci                        # url comes from DATABASE_URL through atlas.hcl
```

Without an `env`, pass the inputs directly: `dir: file://migrations`, `dir-name: app`,
`dev-url: ${{ env.DEV_URL }}`, and for apply `url: ${{ secrets.DATABASE_URL }}`.

```yaml
# .github/workflows/atlas-deploy.yaml
name: Deploy Migrations
on:
  workflow_dispatch:
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: ariga/setup-atlas@v0
        with:
          cloud-token: ${{ secrets.ATLAS_CLOUD_TOKEN }}
      - uses: ariga/atlas-action/migrate/apply@v1
        with:
          url: ${{ secrets.DATABASE_URL }}
          dir: atlas://app?tag=latest          # or ?tag=<commit> for a pinned release
```

Action inputs worth knowing: `env` and `config` select the `atlas.hcl` env; `vars` passes
`--var` values as a JSON string; `dir-name` is the registry repo slug; `tag` pins a registry version;
`tx-mode`, `allow-dirty`, and `dry-run` map to the CLI flags; `working-directory` for monorepos.

### Automatic migration generation (`migrate/diff`)

When the schema is code (HCL, SQL, or ORM) and a developer changes it without generating the
migration, `migrate/diff` plans it on the PR and commits the file to the migration directory. It also
fails when the directory and the schema are out of sync. `dir`, `to`, and `dev-url` are required but
come from `atlas.hcl` when `env` is set.

```yaml
# .github/workflows/atlas-diff.yaml
name: Generate Migrations
on:
  pull_request:
    paths: ['schema.sql', 'migrations/*', 'atlas.hcl']
jobs:
  migrate-diff:
    permissions:
      contents: write                    # push the generated file to the PR branch
      pull-requests: write               # comment on the PR
    env:
      GITHUB_TOKEN: ${{ github.token }}
    runs-on: ubuntu-latest
    steps:
      - uses: ariga/setup-atlas@v0
        with:
          cloud-token: ${{ secrets.ATLAS_CLOUD_TOKEN }}
      - uses: actions/checkout@v4
        with:
          token: ${{ secrets.PAT }}      # a PAT, so the pushed commit triggers the lint workflow
          fetch-depth: 0
      - name: config git to commit changes
        run: |
          git config --local user.email "github-actions[bot]@users.noreply.github.com"
          git config --local user.name "github-actions[bot]"
      - uses: ariga/atlas-action/migrate/diff@v1
        with:
          env: ci                        # or explicit dir, to, and dev-url inputs
```

A commit pushed with the default `github.token` does not trigger other workflows; use a personal
access token (`PAT` secret) so `migrate/lint` runs on the generated file.

### Migration conflicts (`migrate/autorebase`)

Migration history is linear. When two branches add files, the second to merge conflicts on
`atlas.sum`, because the file is a checksum of the directory and is only correct once Atlas recomputes
it for the final order. `migrate/autorebase` moves the branch's migration files after the ones on the
base branch and updates `atlas.sum`. It reorders files without reading the schema, so pair it with
`migrate/lint`, which replays the directory in its new order; a lint failure goes back to the
developer, who fixes it at the schema level and runs `atlas migrate hash`.

```yaml
# .github/workflows/atlas-rebase.yaml
name: Rebase Atlas Migrations
on:
  push:
    branches-ignore: [master]            # every branch except the default one
jobs:
  migrate-auto-rebase:
    permissions:
      contents: write                    # push the rebased files
    runs-on: ubuntu-latest
    steps:
      - uses: ariga/setup-atlas@v0
        with:
          cloud-token: ${{ secrets.ATLAS_CLOUD_TOKEN }}
      - uses: actions/checkout@v4
        with:
          token: ${{ secrets.PAT }}      # so the pushed commit triggers the lint workflow
          fetch-depth: 0                 # the rebase needs the branch history
      - name: config git to commit changes
        run: |
          git config --local user.email "github-actions[bot]@users.noreply.github.com"
          git config --local user.name "github-actions[bot]"
      - uses: ariga/atlas-action/migrate/autorebase@v1
        with:
          base-branch: master
          dir: file://migrations
```

Trigger it on `push`, not `pull_request`: a conflicted PR no longer fires `pull_request` workflows,
which is exactly the state this action fixes. Without the action, the developer resolves it locally:
take either side of `atlas.sum`, `atlas migrate rebase <version>`, `atlas migrate hash`, re-lint.

The same two actions exist for other platforms: `migrate-diff` on GitLab, Azure DevOps, and
Bitbucket; `migrate_autorebase` on CircleCI, and `migrate autorebase` on Azure DevOps and Bitbucket.

### Deploying from the registry

`atlas://app?tag=latest` deploys whatever CI pushed last; `?tag=<commit>` or `?version=<version>` pins a
release. The same URL works in the Kubernetes operator (`AtlasMigration` resource), the Terraform
provider (`atlas_migration` resource), Helm hooks, ArgoCD and Flux, and ECS or Fly.io tasks. Every
deploy is recorded in Atlas Cloud; `references/cloud.md` shows how to read it back.

Add drift protection to the deploy env: `check "migrate_apply" { drift { on_error = FAIL } }`
(`references/drift.md`), and pre-execution `deny` rules for batch size or peak hours
(`references/versioned.md`).

## Declarative Pipeline

| Stage | Trigger | What runs | GitHub Action |
|-------|---------|-----------|---------------|
| Plan | pull request | Plans the transition from the registry's last state (or the database URL) to `schema.src`, lints it, comments the SQL on the PR, stores it as `PENDING` | `ariga/atlas-action/schema/plan@v1` |
| Approve | merge to main | Approves the pending plan in the registry | `ariga/atlas-action/schema/plan/approve@v1` |
| Push | merge to main | Pushes the schema to the registry (`latest` tag) | `ariga/atlas-action/schema/push@v1` |
| Deploy | release / GitOps | `schema apply` finds the approved plan for the transition and runs it with no prompt | `ariga/atlas-action/schema/apply@v1`, operator, Terraform |

```yaml
# .github/workflows/atlas-plan.yaml
name: Plan Declarative Migrations
on:
  pull_request:
    paths: ['schema.sql', 'atlas.hcl']
  push:
    branches: [master]
    paths: ['schema.sql', 'atlas.hcl']
permissions:
  contents: read
  pull-requests: write
jobs:
  plan:
    if: github.event_name == 'pull_request'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: ariga/setup-atlas@v0
        with:
          cloud-token: ${{ secrets.ATLAS_CLOUD_TOKEN }}
      - uses: ariga/atlas-action/schema/plan@v1
        env:
          GITHUB_TOKEN: ${{ github.token }}
        with:
          env: ci
  approve-push:
    if: github.event_name == 'push'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: ariga/setup-atlas@v0
        with:
          cloud-token: ${{ secrets.ATLAS_CLOUD_TOKEN }}
      - uses: ariga/atlas-action/schema/plan/approve@v1
        with:
          env: ci
      - uses: ariga/atlas-action/schema/push@v1
        with:
          env: ci
      # Deploy on merge: apply runs the plan approved above with no prompt.
      # Move this step to a separate deploy workflow for staged rollouts.
      - uses: ariga/atlas-action/schema/apply@v1
        with:
          env: ci
```

The order on merge is approve, push, apply: approve turns the PR's `PENDING` plan into the approved
plan for this transition, push records the new desired state, and apply finds that plan and runs it.

Rules that keep the plan usable at deploy time:

- `schema/plan`, `schema/plan/approve`, and `schema/apply` must see the same database state. Either give
  all three the same `url` (env var or `from` input) or let all three use the registry's last known
  state. A mismatch fails with `The plan "From" hash does not match the current state hash`.
- Directives can be added from the PR description: `/atlas:nolint destructive` and
  `/atlas:txmode none` are parsed by `schema/plan` and written into the plan.
- Ad-hoc approval: when `schema apply` runs in CD and no approved plan matches (a manual change
  drifted the database, or the change skipped CI), the `lint { review }` policy decides whether it
  pauses. With `review = ALWAYS` on `prod` it pauses and prints a registry link where a human approves.

## Other CI Platforms

| Platform | Integration | Docs |
|----------|-------------|------|
| GitLab CI | `migrate-lint`, `migrate-push`, `migrate-diff`, `schema-plan`, `schema-push` components | https://atlasgo.io/integrations/gitlab-ci-components |
| CircleCI | `atlas-orb` (`migrate_lint`, `migrate_push`, `migrate_autorebase`, `schema_plan`) | https://atlasgo.io/integrations/circleci-orbs |
| Bitbucket | `migrate/lint`, `migrate/push`, `migrate/autorebase`, `schema/plan`, `schema/push` pipes | https://atlasgo.io/integrations/bitbucket-pipes |
| Azure DevOps | `migrate lint`, `migrate push`, `migrate diff`, `migrate autorebase`, `schema plan`, `schema push` tasks | https://atlasgo.io/integrations/azure-devops |
| Kubernetes | Atlas Operator: `AtlasMigration` (versioned) and `AtlasSchema` (declarative) resources | https://atlasgo.io/integrations/kubernetes |
| Terraform | `atlas_migration` and `atlas_schema` resources | https://atlasgo.io/integrations/terraform-provider |
| ArgoCD / Flux | GitOps deployment with the operator | https://atlasgo.io/guides/deploying/k8s-argo, https://atlasgo.io/guides/deploying/k8s-flux |
| Go SDK | `atlasexec` for programmatic apply | https://atlasgo.io/integrations/go-sdk |

The CLI equivalents run anywhere: `atlas login --token "$ATLAS_TOKEN"`, then the same `migrate lint`,
`migrate push`, `schema plan`, `schema push`, and `migrate apply` / `schema apply` commands with
`--env ci` and `--format '{{ json . }}'` for machine-readable reports.

## Agent Workflow: Set Up CI/CD

1. Read `atlas.hcl` and decide versioned or declarative (`SKILL.md`, Choosing a Workflow).
2. Confirm a registry repo exists (`atlas cloud repo list`, `references/cloud.md`). If not, ask the user
   to approve the name, then create it with the `create-repo` action (above) or the first
   `migrate push` / `schema push`. Confirm the bot token is available as a secret.
3. Add an `env "ci"` with a dev database and lint policy. Prefer `docker://` dev URLs on hosted runners
   that have Docker; otherwise a service container.
4. Write the PR workflow (lint or plan, tests) and the merge workflow (push, plus approve for
   declarative). Trigger paths on the migration directory, schema source, and `atlas.hcl`.
5. Write the deploy step from `atlas://<repo>` with drift and pre-execution checks on the deploy env.
6. Verify with a dry run: `atlas migrate lint --env ci` locally (`--git-base master` only without a registry), and a `--dry-run`
   apply against a staging database.
7. Check the first CI deployment in Atlas Cloud: `atlas cloud migration list --repo <slug>`.

## Documentation

- [Versioned CI/CD setup](https://atlasgo.io/versioned/setup-cicd)
- [Declarative CI/CD setup](https://atlasgo.io/declarative/setup-cicd)
- [GitHub Actions reference](https://atlasgo.io/integrations/github-actions)
- [atlas-action repository, every action with inputs and examples](https://github.com/ariga/atlas-action)
- [Additional example: both pipelines on ClickHouse](https://github.com/atlasdemos/clickhouse/tree/main/.github/workflows)
- [Additional example: both pipelines on Databricks](https://github.com/atlasdemos/databricks/tree/main/.github/workflows)
- [Pre-approval workflow on GitHub Actions](https://atlasgo.io/integrations/github-actions/pre-approval)
- [Ad-hoc approval on GitHub Actions](https://atlasgo.io/integrations/github-actions/ad-hoc-approval)
- [Kubernetes operator](https://atlasgo.io/integrations/kubernetes)
- [Deployment guides](https://atlasgo.io/guides/deploying/intro)
- [Bot tokens](https://atlasgo.io/cloud/bots)
