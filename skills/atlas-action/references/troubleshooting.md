# Troubleshooting Reference: Fixing a Failed Atlas Step

Start from the log, not from the workflow file: the step prints the exact CLI error. Find the message in
the tables below, fix the cause, and push; never rerun a deploy, approve a plan, or retry with
`allow-dirty` to make a red step green. Common questions that are not errors are at the end.

## Read the Run

| Platform | Commands |
|----------|----------|
| GitHub Actions | `gh pr checks <pr>`, `gh run list --workflow <file> --limit 5`, `gh run view <run-id> --log-failed` |
| GitLab CI | `glab ci list`, `glab ci trace <job-id>` |
| CircleCI, Bitbucket, Azure DevOps | The job log in the UI; ask the user to paste the failing step's output |

- On GitHub, also read the job summary, the file annotations, and the pull request comment: lint and plan
  details are there, not in the log.
- The step's last lines hold the result: the action's own message
  (`` `atlas migrate lint` completed with errors ``) or the CLI error, usually after `Error:`.
- Reproduce the same command locally with the same env when it touches only a dev database: lint, test,
  validate, diff. Never reproduce a deploy against a shared database.

## Install, Versions, and Runtime

| Message | Cause | Fix |
|---------|-------|-----|
| `Invalid version: <ref>` | The step is pinned to a short SHA or a branch other than `master` | Pin `@v1`, `@v1.x.y`, or the full 40-character SHA of a release commit (v1.15.4 or later); for any other commit, add `ariga/atlas-action/setup@<sha>` first |
| `Failed to download atlas-action binary from all sources` | The runner cannot reach `atlasbinaries.com` and `release.ariga.io`, or the step is pinned to the SHA of a commit with no published binary (anything but a release from v1.15.4 on) | Allow the hosts, or build with `ariga/atlas-action/setup@<sha>` |
| Steps pinned to `@master` behave like an old release | `@master` downloads a binary that stopped updating in March 2025 | Pin `@v1` or a release tag |
| A step fails right after the download on an ARM or macOS runner | The downloaded binary is `linux/amd64` | Add `ariga/atlas-action/setup` before the steps; Windows runners are not supported |
| `looking up atlas-cli: exec: "atlas": executable file not found in $PATH` | No `setup-atlas` before the step | Add `ariga/setup-atlas@v0` to the job |
| `no action found for the current environment` | The binary ran outside a supported CI (no `GITHUB_ACTIONS`, `GITLAB_CI`, `CIRCLECI`, `BITBUCKET_PIPELINE_UUID`, `TF_BUILD`, or `TEAMCITY_VERSION`) | Run it as a CI step; on other CI systems run the CLI (`references/platforms.md`) |
| `The specified Atlas version does not exist or is no longer supported.` | `setup-atlas` is pinned to a CLI version that was retired (https://atlasgo.io/cli-reference#distributed-binaries) | Move `version` to a current release, and keep moving it on purpose: Dependabot updates action refs, not this input |
| `version 'GLIBC_2.34' not found` | An old atlas-action release | Upgrade: current binaries are static |
| `Node.js 20 actions are deprecated` | An old release pinned | Upgrade to a current v1 release (Node 24) |
| `ariga/atlas-action@v0`, `ariga/atlas-sync-action`, or `ariga/atlas-deploy-action` in the workflow | Actions deprecated in October 2023, without updates (https://atlasgo.io/blog/2023/10/12/atlas-actions-v1) | `migrate/lint@v1` (with `dir-name`), `migrate/push@v1`, and `migrate/apply@v1` after `setup-atlas` with a token. There is no `dir-format` input: set `format` in the env's `migration` block, or `dir: file://migrations?format=golang-migrate` |
| `<feature> is no longer supported by this release`, such as `data.external_schema is no longer supported by this release` | The job installed the community build | Install the standard build with `ariga/setup-atlas` or `curl -sSf https://atlasgo.sh \| sh` (https://atlasgo.io/faq/not-supported-features) |
| A pipeline broke with no change in the repository | `setup-atlas` installs `latest`, and a new CLI release changed behavior | Pin `version: v<last good>` in `setup-atlas`, report the regression, upgrade when fixed |
| `migrate diff` fails or plans wrong statements after an ORM or Go upgrade, with no Atlas change | The ORM's schema loader does not support the new version | Upgrade the ORM's Atlas provider (such as `atlas-provider-gorm`), or Ent itself for an Ent schema |
| A job that runs scripts in the `arigaio/atlas` image cannot start a shell | The default image is distroless: it holds only Atlas | Use `arigaio/atlas:latest-alpine`, which has `/bin/sh` (https://atlasgo.io/guides/atlas-in-docker) |
| `the input "<name>" got invalid value for boolean` | A boolean input that is not `true` or `false` | Quote it: `"true"` |
| `the input "vars" is not a valid JSON string` | `vars` is not a JSON object | `vars: '{"var1": "value1", "var2": "value2"}'` |
| `the input "<name>" got invalid value for duration` | `wait-timeout` or `wait-interval` without a unit | `30s`, `5m`, `30m` |

## Login and Tokens

| Message | Cause | Fix |
|---------|-------|-----|
| `command requires 'atlas login'`, `available only to Atlas Pro users`, `are available to logged-in users only`, `login is required to use the '<driver>' driver` | No login in this job: `cloud-token` missing or empty (fork pull request, protected GitLab variable), or the plan lacks the feature | Pass the bot token to `setup-atlas`; check the secret is readable by this event. A user without an account starts a free trial with `atlas login` |
| `'atlas login' is required for organization <org> as specified in atlas.hcl` | The token belongs to another organization | A bot token from the organization `atlas.hcl` pins |
| `required input "cloud-token" is missing` | `monitor/schema` without its own token | Pass `cloud-token` to that step |
| A merge job cannot push or approve after protected flows were turned on | The bot has no role on the repo | Give the bot the Writer role on the repo (https://atlasgo.io/cloud/roles-and-permissions) |
| `Branch "<name>" is not allowed to deploy to <environment> due to environment protection rules.` | A job that names a protected environment ran on another branch | Run it only on the default branch: a `push` trigger on it, or `if: github.ref == 'refs/heads/main'` |
| Views, functions, triggers, or extensions missing from a plan or a generated migration | The job is logged out: inspect and diff leave out the objects that need the login, with at most a notice in the log | Log in with `setup-atlas`; pin the organization in `atlas.hcl` so a logged-out job fails (`references/setup.md`) |
| `drop enum type "<name>": pq: cannot drop type <name> because other objects depend on it` from `migrate/diff` | The job is logged out, so Atlas does not manage the function that uses the type | Log in with `setup-atlas` (functions need the login) |

## Pull Request Comments

| Symptom | Cause | Fix |
|---------|-------|-----|
| `GITHUB_TOKEN is not set, the action may not have all the permissions` | No token in the step's `env` | `env: GITHUB_TOKEN: ${{ github.token }}` |
| `failed to comment on the pull request: ... 403`, or `Resource not accessible by integration` | `pull-requests: write` missing, or a fork's read-only token | Add the permission; fork pull requests cannot be commented on with `github.token` |
| No comment and no warning | Not a pull request event: `push`, `pull_request_target`, a GitLab branch pipeline, a Bitbucket branch pipeline | Run lint and plan on `pull_request` (merge request and `pull-requests` pipelines) |
| `GITLAB_TOKEN is not set` | No `gitlab-token` input | A project access token with the `api` scope |
| `BITBUCKET_ACCESS_TOKEN is not set` | No token variable on the pipe | Add `BITBUCKET_ACCESS_TOKEN` |
| `the githubConnection input is not set` or `System.AccessToken is not available` | Azure DevOps without a comment credential | `githubConnection` for GitHub code; `env: SYSTEM_ACCESSTOKEN: $(System.AccessToken)` for Azure Repos |
| An old lint comment stays after a fix | Two steps share a `dir-name`, or an old release | One `dir-name` per unit; current releases update the comment in place, and `schema/lint` deletes it when clean |

## Lint

| Message | Cause | Fix |
|---------|-------|-----|
| `atlasaction: missing required parameter dir-name` | `migrate/lint` without `dir-name` | Add `dir-name: <repo slug>`, even when the env names the repo |
| `slug must be lowercase alphanumeric and may contain /.-_` | `dir-name` holds the repo's display name | The slug: the part of the repo URL after `atlas://` |
| `` `atlas migrate lint` completed with errors, see report: <url> `` | Findings at error level | Read the comment or report; fix the file and rehash, or a corrective migration; `-- atlas:nolint` only with approval |
| Every file is linted, and an old file fails new pull requests | The registry repo has no pushed version, or `tag` points at an old one | Push the directory (`migrate/push` on merge); see `references/lint.md` |
| `--latest, --git-base, and --base are mutually exclusive` | The env's `lint` block sets `latest` or `git { base }`, or the step sets `git-base` | Remove them: the action always passes `--base` |
| `Error: git diff: exit status 128` | A CLI `--git-base` names a ref the checkout does not have | `fetch-depth: 0` and `--git-base origin/<branch>` |
| `lookup git: exec: "git": executable file not found` | `--git-base` in an image without git | Install git in the image, or lint against the registry instead |
| `--git-base` reports no changes, though `--latest 1` lints the new file | Git-based detection sees committed files only | Commit the file, or lint against the registry |
| `` `atlas migrate lint` report was not uploaded to Atlas Cloud `` | Login or network problem for the report only | The lint result still counts; fix the login to get the report |
| `checksum mismatch` | `atlas.sum` does not match the files: a file edited without rehashing, CRLF line endings from a Windows checkout, or a directory format mismatch | Unapplied file: `atlas migrate hash` and commit. Line endings: `* text=auto eol=lf` in `.gitattributes`, renormalize, rehash (https://atlasgo.io/faq/checksum-mismatch). Format: set `format` once in the env's `migration` block, such as `format = golang-migrate` |

## Registry and Git Steps

| Message | Cause | Fix |
|---------|-------|-----|
| `failed to push directory: ... extension "postgis" already exists` | The dev image or `baseline` already has an object the directory creates | Remove it from the baseline; database-scoped dev URL when the directory manages extensions |
| `role "<name>" already exists` on the second step of a job | A service-container dev database shared by steps keeps cluster-level objects | One job per step, or `docker://` dev URLs |
| `failed to push directory`, `failed to push dir tag`, `failed to push schema tag` | The registry or the dev database rejected the push | Read the CLI error that follows; on repos with protected flows, the bot needs the Writer role |
| `failed to fetch the base ref <x>` | Missing base branch, shallow clone, or a host without fetch-by-SHA | `fetch-depth: 0`; check `base-branch` |
| `failed to get the atlas.sum file from the current branch` | The branch is not on the remote, or `dir` points elsewhere | Run on `push` of the branch; check `dir` and `working-directory` |
| `conflict found in files other than <dir>/atlas.sum` | The merge conflicts in code, not only in `atlas.sum` | A person resolves the merge, then the step can rehash |
| `failed to commit changes` | No git identity in the job | Configure `user.name` and `user.email` before the step |
| `failed to push changes` | No `contents: write`, a protected branch, or a fork | Grant it, or run the step only on branches the token can push |
| The generated or rebased commit did not trigger lint | Pushed with `github.token` | Check out with a PAT (`references/diff.md`) |

## Deploy

| Message | Cause | Fix |
|---------|-------|-----|
| `connected database is not clean: ... baseline version or allow-dirty is required` | First deploy to a database with existing objects | `baseline = "<version>"` in the env's migration block |
| `invalid port ":<text>" after host` or `net/url: invalid userinfo` | The password in the URL secret has characters URLs reserve, such as `#`, `?`, `@`, or `/` | Store the password percent-encoded, or in its own secret that `atlas.hcl` escapes with `urlescape` (`references/setup.md`; https://atlasgo.io/faq/url-special-chars) |
| `missing scheme for dir url. Did you mean "file://<path>"?` | `dir` has no scheme | `dir: file://migrations` or `atlas://<repo>` |
| `"<path>" is not a dir` | `dir` points at a schema file, not a migration directory | A migration directory; desired-state files deploy with `schema/apply` |
| `migration file ... was added out of order` | A pending file is older than the database's version | Rebase the branch's files (`references/rebase.md`); `exec-order` only with approval (https://atlasgo.io/versioned/apply#non-linear-error) |
| `plan approval pending, review here: <url>` (`migrate/down`) | The revert needs approval | The user approves in Atlas Cloud and reruns, or set `wait-timeout` |
| `plan rejected, review here: <url>` | The revert was rejected in Atlas Cloud | Nothing to run; talk to the reviewer |
| The deploy stopped on the drift check | The target differs from its migration history | Run `migrate/drift` to see the difference (`references/drift.md`) |
| `The INSERT permission was denied on the object 'atlas_schema_revisions'` on a dry run | Old CLI versions wrote to the revisions table on a dry run with `--env` and a token | Upgrade the CLI, or grant the role write access to the revisions table only |
| `unexpected transaction status idle` | `BEGIN`/`COMMIT` in a file Atlas already wraps | Remove them, or put `-- atlas:txmode none` at the top of the file; lint reports them as nested transactions (https://atlasgo.io/lint/analyzers#nested-transactions) |
| `migrate/apply` runs `.down.sql` files, or fails on a statement from one | A golang-migrate directory read in the Atlas format | `format = golang-migrate` in the env's `migration` block, or `dir: file://<path>?format=golang-migrate` (https://atlasgo.io/faq/migrations-with-golang) |

## Declarative

| Message | Cause | Fix |
|---------|-------|-----|
| `the action should be run in a branch context` | `schema/plan/approve` ran on a pull request | Run it on `push` to the default branch |
| `found multiple schema plans, please approve or delete the existing plans` | Stale pending plans for the same transition | Delete the stale plans in Atlas Cloud, or pass `plan` |
| `The plan "From" hash does not match the current state hash` | Plan, approve, and apply saw different current states, or the target changed since the plan | Same `url`/`from` for all three, or all from the registry; re-plan |
| `the schema plan "<name>" was not approved within the specified waiting period` | Ad-hoc approval timed out | The user approves the linked plan; rerun |
| `schema plan not found` while waiting | The pending plan was deleted in Atlas Cloud during the wait | Rerun the step |
| `No pending schema plan found` (approve) | Nothing was planned for this transition | Rerun `schema/plan` before deploying |
| `` `atlas schema plan` completed with lint errors `` | The plan breaks the policy | Change the schema, or a PR-description directive with approval |

## Drift, Security Scan, Data Scripts

| Message | Cause | Fix |
|---------|-------|-----|
| `` `atlas migrate drift` detected drift in N database(s) `` | Objects changed outside the migrations | The log lists them; fix forward with a migration or revert the manual change |
| `<url> could not be checked` or `could not be scanned` | Connection or permission problem on that database | Check the URL secret, network access from the runner, the user's privileges |
| `only one of the inputs 'config' or 'url' must be given` | `monitor/schema` got both `url` and `config`/`env` | Keep one |
| `` `atlas security scan` completed with N issue(s) `` | Findings at or above `fail-on` | Upgrade the extension, or `ignore` the CVE with a recorded reason |
| `<script>: assert "<name>" failed` or `check "<name>" failed` | The data did not meet the script's expectation | The run rolled back what it could; read the diff in the log |
| A script step passed but did nothing | `match` selected no script of that kind | Anchor and check the name: `'^name$'` |

## Dev Database

| Message | Cause | Fix |
|---------|-------|-----|
| `exec: "docker": executable file not found in $PATH` | `docker://` dev URL where Docker is missing (container job, Atlas image, container agent) | A service container and its URL (`references/setup.md`) |
| `connect: connection refused` | The service is not healthy yet, or the host is wrong | Add health checks; `localhost` from the runner, the service name from a container job |
| `timeout: postgres: querying system variables: dial tcp <address>:<port>`, or `read tcp [::1]:<port>`, with a `docker://` dev URL | Atlas cannot reach the port Docker published for the dev container: some self-hosted runners, or Atlas in a container that shares the host's Docker socket | Provide the dev database: a service container, or an empty database the runner reaches, and pass its URL (https://atlasgo.io/concepts/dev-database#providing-your-own-dev-database). Atlas in a container can also use host networking |
| `connected database is not clean` (dev), such as `found schema "atlas_schema_revisions"` | The dev database is not empty, or the dev URL points at a real database | An empty database or schema used only by Atlas (https://atlasgo.io/faq/migrate-diff-clean) |
| `modify schema "public" is not allowed when migration plan is scoped to one schema` | The dev database's schema differs from the target's in charset, collation, or comment, or the dev URL's scope differs from the target's | Give the dev database the target's schema settings, or match the scope (`atlas/SKILL.md`, Dev Database; https://atlasgo.io/faq/modify-schema-error) |
| `pq: type "<type>" does not exist` or `pq: function <name>() does not exist`, for an extension's type or function (`geometry`, `geography`, `citext`, `uuid_generate_v4`) | The job is logged out, and extensions are managed only with the login; or nothing creates the extension | Log in with `setup-atlas`; otherwise create the extension in the schema or the dev `baseline` |
| `extension "<name>" has no installation script nor update path for version "<v>"` | The dev image lacks the target's extension version | A dev image with the target's extension versions, or exclude extension versions (https://atlasgo.io/faq/extension-version-mismatch) |
| `pq: out of shared memory` | The dev database's lock table is too small for the schema | Raise `max_locks_per_transaction` with the `docker` block's `command` (https://atlasgo.io/faq/out-of-shared-memory) |
| A MySQL or MariaDB plan or drift report keeps showing a generated column changed from `_latin1'...'` to `_utf8mb4'...'` | The dev database's charset or collation differs from the target's | Start the dev database with the target's charset and collation (https://atlasgo.io/faq/mysql-charset-collation-drift) |

## Questions

| Question | Answer |
|----------|--------|
| Install only the CLI, for the job's own scripts | `ariga/setup-atlas` puts `atlas` on the `PATH` for the later steps; without `cloud-token` it does not log in |
| Install with `go install` | Not supported: use `setup-atlas` or the install script (https://atlasgo.io/faq/go-install-deprecation) |
| Secrets from a secret manager | The `runtimevar` data source reads them when the step runs (https://atlasgo.io/guides/deploying/secrets), or a `data "external"` block runs the manager's CLI; an environment variable read with `getenv()` also works |
| Lint every file, not only the new ones | With the CLI: `atlas migrate lint --env ci --latest <N>`, with `N` the number of files in the directory |
| Run `SET ROLE` or another statement before each migration | Migration hooks in the deploy env, which need the login (https://atlasgo.io/versioned/apply#migration-hooks) |
| The database is reachable only through an SSH tunnel | Atlas has no built-in tunnel: open it in an earlier step and point the URL at its local end |
| Deploy without shipping migration files in the application image | Push the directory on merge and deploy from `atlas://<repo>` (`references/apply.md`) |
| Is there a `schema inspect` step? | No: run `atlas schema inspect` in a script step after `setup-atlas` |
