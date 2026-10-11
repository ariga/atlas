# Action Reference

The CI/CD steps of atlas-action v1.17.7 that this skill covers, with their inputs and outputs as the
binary reads them. Behavior, examples, and failure modes are in the step references; this file answers
"what is the input called".

## Names on Each Platform

The spelling of a step and its inputs changes per platform:

| Platform | Step | Input `allow-dirty` | Reading output `applied_count` |
|----------|------|---------------------|--------------------------------|
| GitHub Actions | `uses: ariga/atlas-action/migrate/apply@v1` | `allow-dirty` | `${{ steps.<id>.outputs.applied_count }}` |
| GitLab CI component | `component: $CI_SERVER_FQDN/arigaio/atlas/migrate-apply@~latest` | `allow-dirty` | `$ATLAS_OUTPUT_MIGRATE_APPLY_APPLIED_COUNT` in jobs that `need` it (dotenv artifact) |
| CircleCI orb | `atlas-orb/migrate_apply` | `allow_dirty` | `$ATLAS_OUTPUT_MIGRATE_APPLY_APPLIED_COUNT` in later steps of the job |
| Bitbucket pipe | `pipe: docker://arigaio/atlas-action:v1` with `ATLAS_ACTION: migrate/apply` | `ATLAS_INPUT_ALLOW_DIRTY` | `$ATLAS_OUTPUT_MIGRATE_APPLY_APPLIED_COUNT` after `source .atlas-action/outputs.sh` |
| Azure DevOps | `task: AtlasAction@1` with `action: 'migrate apply'` | `allow_dirty` | `$(<step name>.applied_count)` |

GitHub, GitLab, and Bitbucket run the current release, with every step below. The CircleCI orb and the
Azure DevOps task are published on their own schedule and lag it: each step below names its orb command
and Azure action only where they exist. Where a platform has no name for a step, run the step's CLI
command in a script step there (`references/platforms.md`).

Multi-line inputs (`config`, `schema`, `to`, `from`, `url` where noted, `paths`, `files`, `include`,
`exclude`, `urls`, `ignore`) take one value per line. Booleans take `true` or `false`; any other value
stops the step. `vars` takes a JSON object (`'{"var1": "value1", "var2": "value2"}'`) and stops the step
if it is not valid JSON. Durations (`wait-timeout`, `wait-interval`) take a unit: `30s`, `5m`, `30m`.

Defaults in the tables apply on GitHub and GitLab, whose metadata carries them. Azure DevOps and
Bitbucket pass only the inputs you set, so `latest` on the push steps is `false` there unless you set it,
and so is `collect-stats` on Bitbucket. The CircleCI orb has its own parameters: set `latest: true` on
`schema_push`.

## Common Inputs

| Input | Description |
|-------|-------------|
| `working-directory` | Directory to run in, relative to the repository root. Use it for each unit of a monorepo. |
| `config` | URL of the config file, default `file://atlas.hcl` in the working directory. Several URLs, one per line, are joined. |
| `env` | The env block of `atlas.hcl` to use. Prefer it over passing URLs as inputs. |
| `vars` | JSON object of input variables for `atlas.hcl` (the CLI's `--var`). |
| `dev-url` | Dev database URL: a `docker://` URL (needs Docker on the runner) or a service container's URL. |
| `revisions-schema` | Schema that holds the revisions table, when it is not the default. |

## `create-repo`

Create a new repository in Atlas Cloud.

GitHub `ariga/atlas-action/create-repo@v1` · GitLab `create-repo` · Bitbucket
`ATLAS_ACTION: create-repo`

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `description` |  |  | Accepted, but not passed to the CLI in this version. |
| `driver` | yes |  | The database driver to use for the repository. |
| `name` | yes |  | The name of the repository to create in Atlas Cloud. |
| `type` | yes |  | `migration_directory` (or `m`) or `schema` (or `s`). The CLI rejects `migration`, the value the action metadata lists, and the GitLab component accepts only that list. |

| Output | Description |
|--------|-------------|
| `url` | Link to the new repository in Atlas Cloud (set, though not declared in the metadata). |

## `migrate/apply`

Applies a migration directory on a target database.

GitHub `ariga/atlas-action/migrate/apply@v1` · GitLab `migrate-apply` · orb `atlas-orb/migrate_apply` ·
Bitbucket `ATLAS_ACTION: migrate/apply` · Azure `action: 'migrate apply'`

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `allow-dirty` |  | `false` | Allow working on a non-clean database. |
| `amount` |  |  | The maximum number of migration files to apply. Default is all. |
| `dir` |  |  | The URL of the migration directory to apply. For example: `atlas://dir-name` for cloud based directories or `file://migrations` for local ones. |
| `dry-run` |  | `false` | Print SQL without executing it. Either "true" or "false". |
| `exec-order` |  |  | How Atlas computes and executes pending migration files to the database. Values: `linear`, `linear-skip`, `non-linear`. |
| `to-version` |  |  | The target version to apply migrations to. Mutually exclusive with `amount`. |
| `tx-mode` |  |  | Transaction mode to use. Values: `file`, `all`, `none`. |
| `url` |  |  | The URL of the target database. For example: `mysql://root:pass@localhost:3306/dev`. |

Common inputs: `working-directory`, `config`, `env`, `vars`, `revisions-schema`.

| Output | Description |
|--------|-------------|
| `applied_count` | The number of migrations that were applied. |
| `current` | The current version of the database. (before applying migrations) |
| `error` | The error message when the step fails. |
| `pending_count` | The number of migrations that will be applied. |
| `runs` | A JSON array with the current version, target version, applied count, and pending count of each target. Set only when every target succeeded. |
| `target` | The target version of the database. |

## `migrate/autorebase`

Resolves `atlas.sum` conflicts and rebases the migration directory onto the target branch. Needs a git
checkout with history; commits and pushes to the current branch.

GitHub `ariga/atlas-action/migrate/autorebase@v1` · GitLab `migrate-autorebase` · orb
`atlas-orb/migrate_auto_rebase` · Bitbucket `ATLAS_ACTION: migrate/autorebase` · Azure
`action: 'migrate autorebase'`

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `base-branch` |  |  | The base branch to rebase the migration directory onto. Default to the default branch of the repository. |
| `base-sha` |  |  | Optional. When set, use this git commit SHA as the base instead of base-branch. Requires the git server to support fetch-by-SHA (e.g. GitHub, GitLab). |
| `dir` |  |  | The URL of the migration directory to rebase on. By default: `file://migrations`. |
| `force-rebase` |  | `false` | When true, skip the merge step and rebase whenever there are dev-only migrations (e.g. out-of-order workflow after merging base and running apply with exec-order non-linear). |
| `remote` |  |  | The remote to fetch from. Defaults to `origin`. |

Common inputs: `working-directory`.

| Output | Description |
|--------|-------------|
| `latest_version` | The latest migration version in the directory after rebase. |
| `rebased` | Whether migration files were rebased. Either "true" or "false". |

## `migrate/diff`

Generates versioned migrations whenever the schema changes, and commits them to the migration directory.
Needs a git checkout; commits and pushes to the current branch.

GitHub `ariga/atlas-action/migrate/diff@v1` · GitLab `migrate-diff` · Bitbucket
`ATLAS_ACTION: migrate/diff` · Azure `action: 'migrate diff'`

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `dir` |  |  | The URL of the migration directory. For example: `file://migrations`. |
| `remote` |  |  | The remote to push changes to. Defaults to `origin`. |
| `to` |  |  | The URL of the desired state. |

Common inputs: `working-directory`, `config`, `env`, `vars`, `dev-url`.

No outputs (`url` is declared in the metadata but never set).

## `migrate/down`

Reverts deployed migration files on a target database.

GitHub `ariga/atlas-action/migrate/down@v1` · GitLab `migrate-down` · orb `atlas-orb/migrate_down` ·
Bitbucket `ATLAS_ACTION: migrate/down` · Azure `action: 'migrate down'`

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `amount` |  |  | The amount of applied migrations to revert. Mutually exclusive with `to-tag` and `to-version`. |
| `dir` |  |  | The URL of the migration directory to apply. For example: `atlas://dir-name` for cloud based directories or `file://migrations` for local ones. |
| `to-tag` |  |  | The tag to revert to. Mutually exclusive with `amount` and `to-version`. |
| `to-version` |  |  | The version to revert to. Mutually exclusive with `amount` and `to-tag`. |
| `url` |  |  | The URL of the target database. For example: `mysql://root:pass@localhost:3306/dev`. |
| `wait-interval` |  | `1s` | Duration between approval checks, e.g. `30s` or `5m`. Each check re-inspects the target database, so prefer `30s` or more when `wait-timeout` is long. |
| `wait-timeout` |  |  | How long to wait for approval, e.g. `30m`. If unset, the action does not wait: it creates the plan, prints its link and fails. Re-run the action after approving. |

Common inputs: `working-directory`, `config`, `env`, `vars`, `dev-url`, `revisions-schema`.

| Output | Description |
|--------|-------------|
| `current` | The current version of the database. (before applying migrations) |
| `error` | The error message when the step fails. |
| `planned_count` | The number of migrations planned for reverting. |
| `reverted_count` | The number of migrations that were reverted. |
| `target` | The target version of the database. |
| `url` | If given, the URL for reviewing the revert plan. |

## `migrate/drift`

Detects schema drift between a target database and its migration history.

GitHub `ariga/atlas-action/migrate/drift@v1` · GitLab `migrate-drift` · Bitbucket
`ATLAS_ACTION: migrate/drift`

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `dir` |  |  | The URL of the migration directory that defines the expected state. For example: `atlas://dir-name` for cloud based directories or `file://migrations` for local ones. |
| `exclude` |  |  | List of glob patterns used to select which resources to filter in inspection. |
| `url` |  |  | The URL of the target database to check for drift. For example: `mysql://root:pass@localhost:3306/prod`. |

Common inputs: `working-directory`, `config`, `env`, `vars`, `dev-url`, `revisions-schema`.

| Output | Description |
|--------|-------------|
| `drifted` | Whether drift was detected in any of the checked databases. Either "true" or "false". |
| `report` | A JSON array of the drift reports, one for each checked database. |

## `migrate/hash`

Recomputes `atlas.sum` and commits it to the current branch when it changed. Needs a git checkout;
commits and pushes to the current branch.

GitHub `ariga/atlas-action/migrate/hash@v1` · GitLab `migrate-hash` · Bitbucket
`ATLAS_ACTION: migrate/hash` · Azure `action: 'migrate hash'`

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `base-branch` |  |  | The branch to fetch before hashing. Default to the default branch of the repository. |
| `dir` |  |  | The URL of the migration directory to hash. By default: `file://migrations`. |
| `remote` |  |  | The remote to fetch from. Defaults to `origin`. |

Common inputs: `working-directory`, `config`, `env`.

No outputs.

## `migrate/lint`

Lints the migration files that are new compared to the registry repo, comments the report on the pull
request, and fails on lint errors.

GitHub `ariga/atlas-action/migrate/lint@v1` · GitLab `migrate-lint` · orb `atlas-orb/migrate_lint` ·
Bitbucket `ATLAS_ACTION: migrate/lint` · Azure `action: 'migrate lint'`

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `dir` |  |  | The URL of the migration directory to lint. For example: `file://migrations`. |
| `dir-name` | yes |  | The registry repo slug. The action fails without it, even when the env names the repo. |
| `git-base` |  |  | Passed as `--git-base`. Leave unset: the step always passes `--base` from `dir-name`, and the CLI takes only one of the two (`references/lint.md`). |
| `git-dir` |  |  | Passed as `--git-dir`, the git working tree for `--git-base`. Leave unset, like `git-base`. |
| `tag` |  | `latest` | The registry tag to compare with. Defaults to `latest`. |

Common inputs: `working-directory`, `config`, `env`, `vars`, `dev-url`, `revisions-schema`.

| Output | Description |
|--------|-------------|
| `report-url` | Link to the lint report in Atlas Cloud. The action metadata lists it as `url`, which is never set. |

## `migrate/push`

Pushes the migration directory to the Atlas Registry, tagged with the commit (or `tag`) and, by default,
`latest`.

GitHub `ariga/atlas-action/migrate/push@v1` · GitLab `migrate-push` · orb `atlas-orb/migrate_push` ·
Bitbucket `ATLAS_ACTION: migrate/push` · Azure `action: 'migrate push'`

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `dir` |  |  | The URL of the migration directory to push. For example: `file://migrations`. |
| `dir-name` |  |  | The name (slug) of the project in Atlas Cloud. |
| `latest` |  | `true` | If true, push also to the `latest` tag. |
| `tag` |  |  | The tag to apply to the pushed migration directory. By default the current git commit hash is used. |

Common inputs: `working-directory`, `config`, `env`, `vars`, `dev-url`, `revisions-schema`.

| Output | Description |
|--------|-------------|
| `url` | Link to the pushed version in Atlas Cloud (set, though not declared in the metadata). |

## `migrate/set`

Edits the revision table to consider all migrations up to and including the given version to be applied.

GitHub `ariga/atlas-action/migrate/set@v1` · GitLab `migrate-set` · Bitbucket `ATLAS_ACTION: migrate/set`

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `dir` |  |  | The URL of the migration directory. For example: `file://migrations`. |
| `url` |  |  | The URL of the target database. For example: `mysql://root:pass@localhost:3306/dev`. |
| `version` | yes |  | The version to set the revision table to. All migrations up to and including this version will be marked as applied. |

Common inputs: `working-directory`, `config`, `env`, `vars`, `revisions-schema`.

| Output | Description |
|--------|-------------|
| `error` | The error message when the step fails. |

## `migrate/test`

Runs `atlas migrate test` on the dev database.

GitHub `ariga/atlas-action/migrate/test@v1` · GitLab `migrate-test` · orb `atlas-orb/migrate_test` ·
Bitbucket `ATLAS_ACTION: migrate/test` · Azure `action: 'migrate test'`

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `dir` |  |  | The URL of the migration directory to apply. For example: `atlas://dir-name` for cloud based directories or `file://migrations` for local ones. |
| `paths` |  |  | List of directories containing test files. |
| `run` |  |  | Filter tests to run by regexp. For example, `^test_.*` will only run tests that start with `test_`. Default is to run all tests. |

Common inputs: `working-directory`, `config`, `env`, `vars`, `dev-url`, `revisions-schema`.

No outputs.

## `monitor/schema`

Syncs the database schema to Atlas Cloud.

GitHub `ariga/atlas-action/monitor/schema@v1` · GitLab `monitor-schema` · orb
`atlas-orb/monitor_schema` · Bitbucket `ATLAS_ACTION: monitor/schema` · Azure `action: 'monitor schema'`

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `cloud-token` | yes |  | The token that is used to connect to Atlas Cloud (should be passed as a secret). |
| `collect-stats` |  | `true` | Whether to also send the database's table statistics (sizes, row counts, index usage) to Atlas Cloud. Not on the orb or the Azure task. |
| `exclude` |  |  | List of glob patterns used to select which resources to filter in inspection. |
| `include` |  |  | List of glob patterns used to select which resources to keep in inspection. |
| `schemas` |  |  | List of database schemas to include (by default includes all schemas). |
| `slug` |  |  | Optional unique identifier for the database server. |
| `url` |  |  | URL of the database to sync (mutually exclusive with `config` and `env`). |

Common inputs: `config`, `env`.

| Output | Description |
|--------|-------------|
| `url` | URL of the schema of the database inside Atlas Cloud. |

## `schema/apply`

Applies schema changes to a target database.

GitHub `ariga/atlas-action/schema/apply@v1` · GitLab `schema-apply` · orb `atlas-orb/schema_apply` ·
Bitbucket `ATLAS_ACTION: schema/apply` · Azure `action: 'schema apply'`

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `auto-approve` |  | `false` | Automatically approve and apply changes. Either "true" or "false". |
| `dry-run` |  | `false` | Print SQL without executing it. Either "true" or "false". |
| `exclude` |  |  | List of glob patterns used to select which resources to filter in inspection. |
| `include` |  |  | List of glob patterns used to select which resources to keep in inspection. |
| `lint-review` |  |  | Automatically generate an approval plan before applying changes. Options are "ALWAYS", "ERROR" or "WARNING". Use "ALWAYS" to generate a plan for every apply, or "WARNING" and "ERROR" to generate a plan only based on review policy. With "WARNING" and "ERROR", set the same policy in the `lint { review }` block of the project config. Otherwise, changes that do not require approval are applied with the config policy (default "ALWAYS"), and the step fails. If unset, and the project config requires approval for the changes, the step fails. In this case, set this input (plus `wait-timeout`/`wait-interval`) to create a plan and wait for a user to approve it during deployment, or re-run the PR's `schema/plan` step. Values: `ALWAYS`, `ERROR`, `WARNING`. |
| `plan` |  |  | The plan to apply. For example, `atlas://<schema>/plans/<id>`. |
| `schema` |  |  | List of database schema(s). For example: `public`. |
| `to` |  |  | URL(s) of the desired schema state. |
| `tx-mode` |  |  | Transaction mode to use. Values: `file`, `all`, `none`. |
| `url` |  |  | The URL of the target database to apply changes to. For example: `mysql://root:pass@localhost:3306/prod`. |
| `wait-interval` |  | `1s` | Duration between approval checks, e.g. `30s` or `5m`. Each check re-inspects the target database, so prefer `30s` or more when `wait-timeout` is long. |
| `wait-timeout` |  |  | How long to wait for approval, e.g. `30m`. If unset, the action does not wait: it creates the plan, prints its link and fails. Re-run the action after approving. |

Common inputs: `working-directory`, `config`, `env`, `vars`, `dev-url`.

| Output | Description |
|--------|-------------|
| `error` | The error message if the action fails. |

## `schema/lint`

Lints the database schema with Atlas.

GitHub `ariga/atlas-action/schema/lint@v1` · GitLab `schema-lint` · Bitbucket
`ATLAS_ACTION: schema/lint` · Azure `action: 'schema lint'`

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `schema` |  |  | The database schema(s) to include. For example: `public`. |
| `url` |  |  | Schema URL(s) to lint. For example: `file://schema.hcl`. |

Common inputs: `working-directory`, `config`, `env`, `vars`, `dev-url`.

| Output | Description |
|--------|-------------|
| `error` | The error message when the step fails. |

## `schema/plan`

Plans a declarative migration to move from the current state to the desired state.

GitHub `ariga/atlas-action/schema/plan@v1` · GitLab `schema-plan` · orb `atlas-orb/schema_plan` ·
Bitbucket `ATLAS_ACTION: schema/plan` · Azure `action: 'schema plan'`

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `exclude` |  |  | List of glob patterns used to select which resources to filter in inspection. |
| `from` |  |  | URL(s) of the current schema state. If not provided, Atlas uses the [`url`](https://atlasgo.io/hcl/config#env.url) from the config file. If that is also not set, Atlas defaults to the last known state in the Atlas Registry (use [`schema.repo`](https://atlasgo.io/hcl/config#env.schema.repo) from the config file). |
| `include` |  |  | List of glob patterns used to select which resources to keep in inspection. |
| `name` |  |  | The explicit name of the plan. |
| `schema` |  |  | List of database schema(s). For example: `public`. |
| `schema-name` |  |  | The name (slug) of the schema repository in Atlas Registry. |
| `to` |  |  | URL(s) of the desired schema state. |

Common inputs: `working-directory`, `config`, `env`, `vars`, `dev-url`.

| Output | Description |
|--------|-------------|
| `link` | Link to the schema plan on Atlas. |
| `plan` | The plan to be applied or generated. (e.g. `atlas://<schema>/plans/<id>`) |
| `status` | The status of the plan. For example, `PENDING` or `APPROVED`. |

## `schema/plan/approve`

Approves a migration plan by its URL.

GitHub `ariga/atlas-action/schema/plan/approve@v1` · GitLab `schema-plan-approve` · orb
`atlas-orb/schema_plan_approve` · Bitbucket `ATLAS_ACTION: schema/plan/approve` · Azure
`action: 'schema plan approve'`

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `exclude` |  |  | List of glob patterns used to select which resources to filter in inspection. |
| `from` |  |  | URL(s) of the current schema state. If not provided, Atlas uses the [`url`](https://atlasgo.io/hcl/config#env.url) from the config file. If that is also not set, Atlas defaults to the last known state in the Atlas Registry (use [`schema.repo`](https://atlasgo.io/hcl/config#env.schema.repo) from the config file). |
| `include` |  |  | List of glob patterns used to select which resources to keep in inspection. |
| `plan` |  |  | The URL of the plan to be approved. For example, `atlas://<schema>/plans/<id>`. If not provided, Atlas will search the registry for a plan corresponding to the given schema transition and approve it (typically, this plan is created during the PR stage). If multiple plans are found, an error will be thrown. |
| `schema` |  |  | List of database schema(s). For example: `public`. |
| `schema-name` |  |  | The name (slug) of the schema repository in Atlas Registry. |
| `to` |  |  | URL(s) of the desired schema state. |

Common inputs: `working-directory`, `config`, `env`, `vars`, `dev-url`.

| Output | Description |
|--------|-------------|
| `link` | Link to the schema plan on Atlas. |
| `plan` | The plan to be applied or generated. (e.g. `atlas://<schema>/plans/<id>`) |
| `status` | The status of the plan. (e.g., `PENDING`, `APPROVED`) |

## `schema/push`

Pushes a schema version with an optional tag to Atlas.

GitHub `ariga/atlas-action/schema/push@v1` · GitLab `schema-push` · orb `atlas-orb/schema_push` ·
Bitbucket `ATLAS_ACTION: schema/push` · Azure `action: 'schema push'`

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `description` |  |  | The description of the schema. |
| `latest` |  | `true` | If true, push also to the `latest` tag. |
| `schema` |  |  | List of database schema(s). For example: `public`. |
| `schema-name` |  |  | The name (slug) of the schema repository in Atlas Registry. |
| `tag` |  |  | The tag to apply to the pushed schema. By default, the current git commit hash is used. |
| `url` |  |  | Desired schema URL(s) to push. For example: `file://schema.lt.hcl`. |
| `version` |  |  | The version of the schema. |

Common inputs: `working-directory`, `config`, `env`, `vars`, `dev-url`.

| Output | Description |
|--------|-------------|
| `link` | Link to the schema version on Atlas. |
| `slug` | The slug of the pushed schema version. |
| `url` | The URL of the pushed schema version. |

## `schema/test`

Runs schema tests against the desired schema.

GitHub `ariga/atlas-action/schema/test@v1` · GitLab `schema-test` · orb `atlas-orb/schema_test` ·
Bitbucket `ATLAS_ACTION: schema/test` · Azure `action: 'schema test'`

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `paths` |  |  | List of directories containing test files. |
| `run` |  |  | Filter tests to run by regexp. For example, `^test_.*` will only run tests that start with `test_`. Default is to run all tests. |
| `url` |  |  | The desired schema URL(s) to test. |

Common inputs: `working-directory`, `config`, `env`, `vars`, `dev-url`.

No outputs.

## `script/exec`

Executes Atlas scripts on a target database.

GitHub `ariga/atlas-action/script/exec@v1` · GitLab `script-exec` · Bitbucket `ATLAS_ACTION: script/exec`

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `files` |  |  | URL(s) of the script files or directories to run. For example: `file://scripts`. If not set, Atlas uses the scripts defined in the Atlas configuration file. |
| `match` |  |  | Run only the scripts matching the given regexp. For example, `^purge_.*` will only run scripts that start with `purge_`. Default is to run all scripts. |
| `url` |  |  | The URL of the target database to run the scripts against. For example: `mysql://root:pass@localhost:3306/prod`. |

Common inputs: `working-directory`, `config`, `env`, `vars`.

| Output | Description |
|--------|-------------|
| `output` | The output of the executed scripts, joined by newlines. It contains the values emitted by their `output` blocks and the results of their `query` blocks. |
| `report` | A JSON report of the run, containing the result of each executed script. |

## `script/loop`

Runs Atlas loop scripts, processing rows in batches, on a target database.

GitHub `ariga/atlas-action/script/loop@v1` · GitLab `script-loop` · Bitbucket `ATLAS_ACTION: script/loop`

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `files` |  |  | URL(s) of the script files or directories to run. For example: `file://scripts`. If not set, Atlas uses the scripts defined in the Atlas configuration file. |
| `match` |  |  | Run only the scripts matching the given regexp. For example, `^purge_.*` will only run scripts that start with `purge_`. Default is to run all scripts. |
| `url` |  |  | The URL of the target database to run the scripts against. For example: `mysql://root:pass@localhost:3306/prod`. |

Common inputs: `working-directory`, `config`, `env`, `vars`.

| Output | Description |
|--------|-------------|
| `output` | The output of the executed scripts, joined by newlines. It contains the values emitted by their `output` blocks and the results of their `query` blocks. |
| `report` | A JSON report of the run, containing the result of each executed script. |

## `script/push`

Pushes Atlas scripts to the Atlas Registry.

GitHub `ariga/atlas-action/script/push@v1` · GitLab `script-push` · Bitbucket `ATLAS_ACTION: script/push`

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `files` |  |  | URL(s) of the script files or directories to push. For example: `file://scripts`. If not set, Atlas uses the scripts defined in the Atlas configuration file. |
| `script-name` |  |  | The name (slug) of the script repository in Atlas Registry. |

Common inputs: `working-directory`, `config`, `env`, `vars`.

| Output | Description |
|--------|-------------|
| `files` | The number of script files that were pushed. |
| `link` | Link to the scripts in Atlas Registry. |
| `name` | The name of the script repository that was pushed to. |

## `script/query`

Runs Atlas query scripts on a target database.

GitHub `ariga/atlas-action/script/query@v1` · GitLab `script-query` · Bitbucket
`ATLAS_ACTION: script/query`

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `files` |  |  | URL(s) of the script files or directories to run. For example: `file://scripts`. If not set, Atlas uses the scripts defined in the Atlas configuration file. |
| `match` |  |  | Run only the scripts matching the given regexp. For example, `^purge_.*` will only run scripts that start with `purge_`. Default is to run all scripts. |
| `url` |  |  | The URL of the target database to run the scripts against. For example: `mysql://root:pass@localhost:3306/prod`. |

Common inputs: `working-directory`, `config`, `env`, `vars`.

| Output | Description |
|--------|-------------|
| `output` | The output of the executed scripts, joined by newlines. It contains the values emitted by their `output` blocks and the results of their `query` blocks. |
| `report` | A JSON report of the run, containing the result of each executed script. |

## `script/test`

Runs tests for Atlas scripts.

GitHub `ariga/atlas-action/script/test@v1` · GitLab `script-test` · Bitbucket `ATLAS_ACTION: script/test`

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `paths` |  |  | List of directories containing test files. |
| `run` |  |  | Filter tests to run by regexp. For example, `^test_.*` will only run tests that start with `test_`. Default is to run all tests. |

Common inputs: `working-directory`, `config`, `env`, `vars`, `dev-url`.

No outputs.

## `security/scan`

Scans databases for security issues with Atlas.

GitHub `ariga/atlas-action/security/scan@v1` · GitLab `security-scan` · Bitbucket
`ATLAS_ACTION: security/scan`

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `fail-on` |  |  | The lowest severity that fails the action. By default, reported issues do not fail it, and only a database that could not be scanned does. Values: `NORMAL`, `ELEVATED`, `HIGH`, `CRITICAL`. |
| `ignore` |  |  | CVE identifier(s) not to report. For example: `CVE-2017-18359`. |
| `min-severity` |  |  | The lowest severity to report. For example, `HIGH` reports only the `HIGH` and `CRITICAL` issues. Values: `NORMAL`, `ELEVATED`, `HIGH`, `CRITICAL`. |
| `urls` |  |  | URL(s) of the target databases, one per line. For example: `postgres://localhost:5432/app?sslmode=disable`. If not set, Atlas uses the [`url`](https://atlasgo.io/hcl/config#env.url) from the config file. |

Common inputs: `working-directory`, `config`, `env`, `vars`.

| Output | Description |
|--------|-------------|
| `count` | The number of security issues that were reported. |
| `failures` | The number of databases that could not be scanned. |
| `report` | A JSON report of the scan, containing the issues found in each scanned database. |

## `ariga/setup-atlas` and `ariga/atlas-action/setup`

Not part of the manifest. `ariga/setup-atlas@v0` installs the Atlas CLI and logs in:

| Input | Default | Description |
|-------|---------|-------------|
| `cloud-token` | | Atlas Cloud bot token. Logs in with `atlas login --token` for the later steps of the job. |
| `version` | `latest` | The Atlas CLI release to install, as a release tag. |
| `flavor` | | The driver that needs the extended build, such as `snowflake` or `spanner`. |

`ariga/atlas-action/setup@<sha>` builds the atlas-action binary from source in the job, so the other
steps can be pinned to any commit and run on any runner architecture. It does not install the Atlas CLI:
keep `setup-atlas` too. Its one input, `token`, is optional: on GitHub Enterprise Server, a github.com
token for downloading Go without rate limits.
