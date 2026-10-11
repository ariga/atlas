---
name: atlas-action
description: "Atlas in CI/CD pipelines with the Atlas actions: ariga/setup-atlas and ariga/atlas-action on GitHub Actions, the GitLab CI components, the CircleCI orb, the Bitbucket pipe, and the Azure DevOps AtlasAction task. The CI/CD steps (migrate lint, push, diff, hash, autorebase, test, apply, down, set, drift; schema plan, plan approve, push, apply, lint, test; script exec, query, loop, test, push; security scan; monitor schema; create-repo) with its inputs, outputs, tokens, permissions, and pull request comments, plus pinning versions, runners, and fixing a failed run. Use with the atlas skill whenever a pipeline runs Atlas or an Atlas check in CI fails."
allowed-tools: Bash(atlas version) Bash(atlas whoami) Bash(atlas config validate:*) Bash(atlas cloud repo list:*) Bash(atlas cloud repo describe:*) Bash(atlas cloud database list:*) Bash(atlas cloud migration list:*) Bash(atlas cloud migration describe:*) Bash(atlas schema plan list:*) Bash(atlas schema validate:*) Bash(atlas migrate validate:*) Bash(atlas migrate lint:*) Bash(atlas migrate ls:*) Bash(gh pr checks:*) Bash(gh run list:*) Bash(gh run view:*) Bash(gh workflow list:*) Bash(gh secret list:*) Bash(glab ci list:*) Bash(glab ci trace:*)
---

# Atlas Actions: CI/CD Pipelines

Atlas runs in pipelines through one binary, `atlas-action`, shipped as GitHub Actions
(`ariga/atlas-action/<step>`), GitLab CI components, a Bitbucket pipe, a CircleCI orb, and an Azure
DevOps task. Each step wraps one Atlas command. The steps a platform has, the input names and defaults,
and the outputs a later job can read differ by platform (`references/actions.md`,
`references/platforms.md`). The `atlas` skill covers the CLI and the workflows; this skill covers the
steps that run them in CI: which step, its inputs and outputs, the tokens and permissions it needs, and
what to do when one fails.

## Files

This skill is `SKILL.md` plus fourteen reference files under `references/`, each served at
`https://atlasgo.io/skills/atlas-action/references/<name>`. Read a reference when the task needs it:

| File | Read it for |
|------|-------------|
| `references/setup.md` | `setup-atlas`, which job gets which credential, `permissions`, comment tokens, common inputs, outputs, runners and dev databases, pinning versions, GitHub Enterprise, monorepos, `gh atlas init-action` |
| `references/actions.md` | Each step's inputs, defaults, and outputs, and their names on each platform |
| `references/lint.md` | `migrate/lint` (which files are new, comments, failures) and `schema/lint` |
| `references/diff.md` | Failing a pull request on a missing migration, and `migrate/diff` and `migrate/hash`, the steps that commit to the branch |
| `references/rebase.md` | `migrate/autorebase`, `atlas.sum` conflicts, out-of-order migrations, `exec-order` |
| `references/push.md` | `migrate/push`, `schema/push`, `script/push`, `create-repo`, registry URLs and tags |
| `references/apply.md` | `migrate/apply` (deploy env checks, baseline, several targets, promotion), `migrate/down`, `migrate/set` |
| `references/declarative.md` | `schema/plan`, `schema/plan/approve`, `schema/apply`, ad-hoc approval |
| `references/testing.md` | `migrate/test`, `schema/test`, `script/test` |
| `references/drift.md` | `migrate/drift` on a schedule, drift at deploy time, `monitor/schema` |
| `references/scripts.md` | `script/exec`, `script/query`, `script/loop` in pipelines |
| `references/security-scan.md` | `security/scan`: severities, failing, the report |
| `references/platforms.md` | GitLab CI, CircleCI, Bitbucket Pipelines, Azure DevOps, and CI systems without an integration, such as Jenkins or TeamCity |
| `references/troubleshooting.md` | Reading a failed run, each error message with its cause and fix, and common questions |

## Requires

The `atlas` skill, installed next to this one (`../atlas/SKILL.md`). If it is missing, download
https://atlasgo.io/skills/atlas/SKILL.md to `../atlas/SKILL.md` and each reference file listed in its
Files section to `../atlas/references/<name>`. Paths below that start with `atlas/` point into that
skill. For the dev database and engine rules, also use the database skill installed next to this one
for the project's engine, such as `atlas-postgres` or `atlas-mysql`, and for Kubernetes deploys,
`atlas-operator`.

## Setup Path

1. Read `atlas.hcl`: the workflow (versioned or declarative), the env names, the registry repo
   (`atlas/SKILL.md`, Choosing a Workflow). Find the CI system from the repository (`.github/workflows/`,
   `.gitlab-ci.yml`, `.circleci/`, `bitbucket-pipelines.yml`, `azure-pipelines.yml`).
2. Credentials (`references/setup.md`): one Atlas Cloud bot token, a repository secret that every job
   uses, and the database URLs as secrets of environments only the default branch can use. An
   organization admin creates the bot; the user stores the secrets.
3. An `env "ci"` in `atlas.hcl` with the dev database, the registry repo, and the lint policy; database
   URLs only through `getenv()`; the organization pinned with `atlas { cloud { org = "<org>" } }`
   (`references/setup.md`). Check the file with `atlas config validate` before committing it: it needs
   the login and no database.
4. The registry repo exists (`atlas cloud repo list`); if not, ask the user to approve the name, then the
   first push or `create-repo` creates it (`references/push.md`).
5. The pull request workflow: `migrate/lint` (plus the test steps) or `schema/plan`.
6. The merge workflow: `migrate/push`, or `schema/plan/approve` then `schema/push`.
7. The deploy workflow, from the registry, with the drift check in the deploy env
   (`references/apply.md`, `references/declarative.md`). Production only when the user asks, in its own
   pull request.
8. Optional jobs, when they fit: `migrate/diff`, `migrate/autorebase`, scheduled `migrate/drift`,
   `security/scan`, `monitor/schema`, Data Scripts.
9. Verify: the pull request's Atlas check ran and commented (`gh pr checks`), the merge pushed a version
   (`atlas cloud repo describe`), and the first deploy is recorded (`atlas cloud migration list`); a
   `file://` deploy is recorded only with `report = true` (`references/apply.md`).

## Steps

| Step | Runs | When |
|------|------|------|
| `migrate/lint` | `atlas migrate lint` against the registry repo | Pull request |
| `migrate/test`, `schema/test`, `script/test` | The Atlas test framework on the dev database | Pull request |
| `migrate/diff` | Generates a missing migration and commits it | Pull request |
| `migrate/hash` | Recomputes `atlas.sum` and commits it | Pull request |
| `migrate/autorebase` | Moves the branch's files after the base's, fixes `atlas.sum` | Push to a feature branch |
| `migrate/push` | Pushes the directory to the registry | Merge to the default branch |
| `migrate/apply` | Deploys pending migrations | Deploy |
| `migrate/down` | Reverts applied migrations, with approval | On demand |
| `migrate/set` | Marks versions applied without running them | On demand, with approval |
| `migrate/drift` | Checks a database against its migration history | Schedule |
| `schema/plan` | Plans and lints the declarative change, comments it | Pull request |
| `schema/plan/approve` | Approves the pull request's plan | Merge to the default branch |
| `schema/push` | Pushes the desired schema to the registry | Merge to the default branch |
| `schema/apply` | Applies the approved plan, or waits for an ad-hoc approval | Deploy |
| `schema/lint` | Lints the whole desired schema, comments it | Pull request |
| `script/exec`, `script/query`, `script/loop` | Data Scripts on a target | Schedule or on demand |
| `script/push` | Pushes Data Scripts to the registry | Merge to the default branch |
| `security/scan` | Reports vulnerable extensions in databases | Schedule |
| `monitor/schema` | Sends a schema snapshot to Atlas Cloud monitoring | Schedule |
| `create-repo` | Creates a registry repo | Once |

## Rules

1. Every job logs in with a bot token from Atlas Cloud before its first Atlas step. On GitHub Actions,
   `ariga/setup-atlas@v0` installs the CLI and logs in with `cloud-token`, and `monitor/schema` takes
   its own `cloud-token`. The other platforms log in as `references/platforms.md` shows. Logged out,
   most steps fail, but inspect and diff pass without views, functions, triggers, extensions, and
   other objects that need the login, with at most a notice in the log. Pin the organization in
   `atlas.hcl` and select an env on every step that takes one, so a job that is not logged in fails
   instead. A user without an account starts a free trial with `atlas login`.
2. Database URLs stay out of pull request jobs. A job that a branch triggers runs that branch's
   workflow file and `atlas.hcl`, whose `data "external"` blocks run programs, so it can read every
   secret it gets. Keep the URLs in environments that only the default branch can use, and name them
   only in deploy and scheduled jobs. Never run Atlas on `pull_request_target` with the pull request's
   code (`references/setup.md`).
3. Configuration lives in `atlas.hcl`, selected with `env`. Database URLs come from secrets through
   `getenv()` or the `url` input; never write one with a password into a workflow file.
4. `migrate/lint` always takes `dir-name`, the registry repo slug. It lints the files that the repo's
   `latest` tag does not have, so `migrate/push` runs on every merge to the default branch, and only
   there: a push from a pull request moves `latest` to unreviewed files.
5. Pull request comments need a pull request event (`pull_request` on GitHub, not `pull_request_target`;
   merge request pipelines on GitLab) and the platform's comment token. On GitHub: `GITHUB_TOKEN` in the
   step's `env` and `pull-requests: write`. A failed comment does not change the step's result.
6. Steps that commit (`migrate/diff`, `migrate/hash`, `migrate/autorebase`) push with a token that the
   branch's code can read. Prefer failing the pull request and letting the developer commit the fix.
   Where the team uses them, they need `contents: write`, a git identity, `fetch-depth: 0`, and a
   checkout with a fine-grained token limited to this repository, so their commit starts lint
   (`references/diff.md`).
7. Deploy from the registry with a pinned tag (`atlas://<repo>?tag=<commit>`). Safety checks (drift,
   `deny` rules, `baseline`) go in the deploy env, not in the workflow. Answer
   `connected database is not clean` with `baseline`, not `allow-dirty`, unless the user confirms the
   existing objects stay outside the directory.
8. Declarative: plan on the pull request, approve and push on merge, apply the approved plan. All three
   read the same current state (`references/declarative.md`).
9. Pin what must not move: the action at a release tag or the full SHA of a release commit (v1.15.4 or
   later), and the CLI with `setup-atlas`'s `version` where a broken pipeline costs more than a missed
   fix.
10. Runners: the GitHub steps download a `linux/amd64` binary. On ARM or macOS runners, add
    `ariga/atlas-action/setup` to build it on the runner; Windows runners are not supported.
    `docker://` dev URLs work only where Docker runs (GitHub-hosted Ubuntu); elsewhere use a service
    container, and give steps that create server-level objects (roles) their own dev database.
11. On Azure DevOps and Bitbucket, input defaults do not apply: set `latest: true` on push steps there,
    and on the CircleCI orb's `schema_push` (`references/platforms.md`).
12. Fix a failed step from its log (`gh run view <id> --log-failed`), with
    `references/troubleshooting.md`. Reproduce against dev databases only.
13. Commit migration files with LF line endings (`* text=auto eol=lf` in `.gitattributes`): a CRLF
    checkout breaks `atlas.sum` (https://atlasgo.io/faq/checksum-mismatch).

## Permissions

| Action | Rule |
|--------|------|
| Reading runs, logs, checks, and `atlas cloud` state; lint and validate on a dev database | Run |
| Writing workflow files and `atlas.hcl` on a branch, opening a pull request | Run; the pull request is the gate |
| Creating registry repos | Ask first |
| Bot tokens and CI secrets | The user creates them; values never enter the conversation (`gh secret list` prints names only) |
| Triggering, rerunning, or approving a workflow run, deployment, revert, or plan | Never; tell the user what is waiting and where |
| Production deploy workflows | Only in a separate pull request, when the user asks |

## Quick Answers

| Question | Answer |
|----------|--------|
| `atlasaction: missing required parameter dir-name` | Add `dir-name: <repo slug>` to `migrate/lint`, even when the env names the repo |
| `slug must be lowercase alphanumeric and may contain /.-_` | `dir-name` holds `atlas://` or a display name: pass the slug alone, the part of the repo URL after `atlas://` |
| Lint fails on old files in every pull request | The registry repo has no pushed version, so every file is new: push on merge (`references/lint.md`) |
| `--latest, --git-base, and --base are mutually exclusive` | Remove `latest` (and `git`) from the env's `lint` block, and the step's `git-base` input: the action always passes `--base` |
| Lint ran, but no comment on the pull request | Not a `pull_request` event, or `GITHUB_TOKEN`/`pull-requests: write` missing (`references/setup.md`) |
| `Resource not accessible by integration` | The job's token cannot write to the pull request: add `permissions: pull-requests: write` to the job |
| Where is the lint report link? | The `report-url` output; the metadata's `url` output is never set |
| Lint without Atlas Cloud | The step needs the login and a registry repo. Without a registry, run `atlas migrate lint --env ci --git-base origin/<branch>` in a plain step (`references/lint.md`) |
| A CLI step ignores the `lint` policy in `atlas.hcl` | Atlas reads `atlas.hcl` only with `--env`, or with `-c` for its top-level blocks: run `atlas migrate lint --env ci` |
| `Invalid version: <ref>` | Pin a release tag, or the full SHA of a release commit (v1.15.4 or later); for any other commit, build with `ariga/atlas-action/setup@<sha>` |
| A step fails at once on an ARM or macOS runner | The downloaded binary is `linux/amd64`: add `ariga/atlas-action/setup` before the steps |
| Pipeline broke after an Atlas release, nothing changed in the repo | Pin `version` in `setup-atlas` to the last good release |
| `The specified Atlas version does not exist or is no longer supported.` | The pinned CLI version was retired: move `version` to a current release |
| `command requires 'atlas login'` | `cloud-token` is missing or empty in this job: fork pull requests and protected variables get no secrets |
| Which secrets go in which job? | One bot token, a repository secret, in every job. Database URLs only in deploy and scheduled jobs, from environments only the default branch can use; a read-only user's for drift, scan, and monitoring (`references/setup.md`) |
| A merge job cannot push or approve after protected flows were turned on | Give the bot the Writer role on the repo (https://atlasgo.io/cloud/roles-and-permissions) |
| The commit from `migrate/diff`, `migrate/hash`, or `migrate/autorebase` did not trigger lint | Check out with a PAT |
| `atlas.sum` conflict between branches | `migrate/autorebase` on push to feature branches, then lint (`references/rebase.md`) |
| `checksum mismatch` in CI only | Line endings: `* text=auto eol=lf` in `.gitattributes`, renormalize, rehash |
| `migration file ... was added out of order` | Rebase the branch's files; `exec-order` only for environments that need it (`references/rebase.md`) |
| `missing scheme for dir url` or `"<path>" is not a dir` from `migrate/apply` | `dir` takes a migration directory URL with its scheme (`file://migrations`, `atlas://app`); a schema file deploys with `schema/apply` |
| `connected database is not clean` on the first deploy | `baseline = "<version>"` in the deploy env's migration block |
| A renamed table or column shows as a drop and a create in the plan or lint comment | CI cannot answer the rename prompt: declare the rename with `renamed_from` (HCL) or `-- atlas:renamed_from` (SQL), not with `nolint` (https://atlasgo.io/guides/destructive-change-policy) |
| `ariga/atlas-action@v0`, `ariga/atlas-sync-action`, or `ariga/atlas-deploy-action` in a workflow | Deprecated in October 2023 and no longer updated: replace them with `migrate/lint@v1`, `migrate/push@v1`, and `migrate/apply@v1` after `ariga/setup-atlas@v0` with a token (`references/troubleshooting.md`) |
| Deploy stopped on the drift check | Run `migrate/drift` on the target to see the difference |
| `the action should be run in a branch context` | `schema/plan/approve` runs on push to the default branch, not on pull requests |
| `found multiple schema plans, please approve or delete the existing plans` | Delete stale pending plans in Atlas Cloud, or pass `plan` |
| `The plan "From" hash does not match the current state hash` | Plan, approve, and apply must read the same state; re-plan |
| `plan approval pending, review here: <url>` | The user approves in Atlas Cloud and reruns, or the step sets `wait-timeout` |
| `exec: "docker": executable file not found in $PATH` | No Docker where the step runs: use a service container dev database |
| `role "..." already exists` on a later step | A shared service-container dev database keeps roles: separate the steps |
| `Failed to download atlas-action binary from all sources` | Allow `atlasbinaries.com` and `release.ariga.io`, or build with `ariga/atlas-action/setup` |
| `create-repo` rejects `type: migration` | The CLI takes `migration_directory` (or `m`) and `schema` (or `s`); it fails if the repo exists |
| `required input "cloud-token" is missing` | `monitor/schema` does not use `setup-atlas`'s login: pass it `cloud-token` |
| `security/scan` passes with CVEs reported | By design: set `fail-on` to the severity that must fail the job |
| Bitbucket pipe runs an old binary | Use `arigaio/atlas-action:v1` or a release tag, not `master` or `latest` |
| `latest` tag does not move on Azure DevOps, Bitbucket, or the orb's `schema_push` | Set `latest: true` (Bitbucket: `ATLAS_INPUT_LATEST: "true"`) |
| How to fail CI when a migration was not generated | `atlas migrate diff --env ci`, then fail on `git status --porcelain` (`references/diff.md`) |
| Lint, test, or plan reports on GitLab merge requests | `gitlab-token` input, and run in merge request pipelines (`references/platforms.md`) |
| Jenkins, TeamCity, Buildkite, or another CI | Run the CLI in a script step: `atlas login --token`, then each command with `--env` (`references/platforms.md`) |
| A step is missing on the CircleCI orb or the Azure DevOps task | They lag the release: run the step's CLI command in a script step (`references/platforms.md`) |

## Documentation

- GitHub Actions: https://atlasgo.io/integrations/github-actions
- GitLab CI components: https://atlasgo.io/integrations/gitlab-ci-components
- CircleCI orb: https://atlasgo.io/integrations/circleci-orbs
- Bitbucket pipes: https://atlasgo.io/integrations/bitbucket-pipes
- Azure DevOps: https://atlasgo.io/integrations/azure-devops
- Versioned CI/CD: https://atlasgo.io/versioned/setup-cicd
- Declarative CI/CD: https://atlasgo.io/declarative/setup-cicd
- Pre-approval and ad-hoc approval: https://atlasgo.io/integrations/github-actions/pre-approval,
  https://atlasgo.io/integrations/github-actions/ad-hoc-approval
- Modern database CI/CD: https://atlasgo.io/guides/modern-database-ci-cd
- Bot tokens: https://atlasgo.io/cloud/bots
