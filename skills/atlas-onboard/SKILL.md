---
name: atlas-onboard
description: "Guided onboarding of a repository onto Atlas, one verified stage at a time: plan the units and the workflow, define the schema as code, set up the migration workflow, add CI with the Atlas Registry, deploy to staging, and promote to production. Each stage ends on a check the agent runs itself, such as a dry run with no changes or a deployment recorded in Atlas Cloud. Use when the user asks to onboard, adopt, or set up Atlas for a project or a team, or to resume an onboarding."
disable-model-invocation: true
allowed-tools: Bash(atlas version) Bash(atlas whoami) Bash(atlas cloud repo list:*) Bash(atlas cloud repo describe:*) Bash(atlas cloud database list:*) Bash(atlas cloud database describe:*) Bash(atlas cloud migration list:*) Bash(atlas cloud migration describe:*) Bash(atlas schema inspect:*) Bash(atlas schema validate:*) Bash(atlas schema test:*) Bash(atlas migrate ls:*) Bash(atlas migrate validate:*) Bash(atlas migrate lint:*) Bash(atlas migrate diff:*) Bash(atlas migrate hash:*) Bash(atlas migrate import:*)
---

# Atlas Onboarding

Takes a repository from no Atlas setup to production in six stages, the same path as the Evaluating
Atlas guide (https://atlasgo.io/guides/evaluation/intro). Each stage ends on a check the agent runs and
shows the user. This skill holds the order, the decisions, and the checks. The `atlas` skill holds the
command reference.

## Files

This skill is `SKILL.md` plus seven files, each served at `https://atlasgo.io/skills/atlas-onboard/<path>`:
`agents/openai.yaml`, `references/stage-0-plan.md`, `references/stage-1-schema.md`,
`references/stage-2-workflow.md`, `references/stage-3-ci.md`, `references/stage-4-deploy.md`, and
`references/stage-5-promote.md`. Read a stage reference when its stage starts.

## Requires

The `atlas` skill, installed next to this one (`../atlas/SKILL.md`). If it is missing, download
https://atlasgo.io/skills/atlas/SKILL.md to `../atlas/SKILL.md` and each reference file listed in its
Files section to `../atlas/references/<name>` before starting. Paths below that start with `atlas/`
point into that skill.

## Stages

| Stage | What it sets up | Read | Done when |
|-------|-----------------|------|-----------|
| 0 Plan | Login, an inventory of the database, the units, the pilot, and the workflow | `references/stage-0-plan.md` | The user approved the units, the pilot, and the workflow |
| 1 Schema as code | `atlas.hcl` and the desired state: the database exported to SQL or HCL, or the ORM models | `references/stage-1-schema.md` | `atlas schema apply --env local --dry-run` reports no changes |
| 2 Migration workflow | Versioned: the baseline migration and the `migrate diff` loop. Declarative: the `schema apply` loop | `references/stage-2-workflow.md` | Versioned: `atlas migrate diff` reports the directory is synced; the project PR is open |
| 3 CI | Lint (versioned) or plan (declarative) on pull requests, push to the Atlas Registry on merge | `references/stage-3-ci.md` | CI passed on the merge, and the registry has its version |
| 4 Deploy | Staging deploys from the registry, and the old tool stops deploying to staging | `references/stage-4-deploy.md` | Atlas Cloud records the staging deployment |
| 5 Promote | Production applies only the version staging runs (environment promotion) | `references/stage-5-promote.md` | The user ran the first production deployment, on staging's version |

Run the stages for the pilot unit first. Every other unit repeats stages 1 to 5 with the decisions
already recorded. After the pilot, Hand Off.

## Rules

1. Detect, do not assume. At the start of every session, run Detect the Stage. Progress lives in the
   default branch, open pull requests, and Atlas Cloud, not in the conversation.
2. Log in before reading a database. Logged out, `atlas schema inspect` and `atlas schema diff` skip
   views, functions, procedures, triggers, and other objects and still exit 0. A schema exported that
   way loses them. The skip notice prints once per machine, so its absence proves nothing. Run
   `atlas whoami` first.
3. Ask only at the decision points each stage lists. Everywhere else, take the stated default and say
   which one you took.
4. Before the first file write of a stage, create a branch `atlas-onboard/<unit>-<stage>` from an
   up-to-date default branch with a clean working tree. Stage files by path, never `git add -A`. Every
   change lands as a pull request, and the PR is the review gate. A separate GitOps repository gets its
   own PR.
5. Credentials of shared environments never enter the conversation. The user exports those URL
   variables in the shell that starts the agent; a `!` command would put the value in the conversation.
   Check a variable with `test -n "$STAGING_DATABASE_URL" && echo set`. Never print, parse, or echo such
   a URL, and never open `.env`, `*.tfvars`, or other secret files. Local development credentials that
   the repository defines, such as a compose service's password, are not secrets.
6. Close each stage with its check: show the command and its output. A check that cannot run (variable
   unset, Docker not running, no network) is not a failed stage: fix the precondition and rerun it.
7. Explain as you go. Define each term from Terms in one sentence the first time you use it. After each
   stage, say in two or three sentences what now exists and why, and link the page from the Docs Map.
8. Read schemas from a local or development database, which the agent may find and connect to on its
   own. Read a remote environment only when the user asks for it and provides the connection; never look
   for remote URLs or credentials. Connecting the agent to a production database is not recommended. For
   a remote environment, recommend a read-only role, and a cloud sign-in where the database supports it
   (stage 1, Remote Databases).

## Permissions

| Action | Rule |
|--------|------|
| Repo scan, `schema inspect`, `migrate lint`, `atlas cloud` reads, `--dry-run` (plans only, changes nothing) | Run |
| Writing files in the repository, branches, pull requests | Run; the PR is the gate |
| Creating Atlas Cloud repos (the first `migrate push` or `schema push`, or the `create-repo` action), bot tokens, CI secrets | Ask first |
| `migrate apply` or `schema apply` against anything but a local or `docker://` dev database | Explicit approval, a named target, and a dry run first |
| Production deploy configuration | Only in a separate PR, when the user asks for it |
| Approving any plan, review, deployment, workflow run, or pull request | Never; tell the user what is waiting and where |

## Project Layout

Each unit is one Atlas project in its own directory, next to the code that owns it (for example
`services/billing/db/`). A repository with one unit can use the root. A unit directory holds
`atlas.hcl`, the desired state (`schema/`, unless the source is an ORM), and, for versioned units, the
migration directory (`migrations/`). Every `atlas.hcl` uses the same env names, so Atlas Cloud shows
real environment names:

| Env | Used for | Written in |
|-----|----------|------------|
| `local` | The local or development database the schema is read from, where developers apply changes; generating and linting migrations | Stage 1 |
| `staging` | Deploying to staging; a read-only check before the first deploy | Stage 4 (stage 1 only when the schema is read from staging) |
| `ci` | Lint and registry push in CI | Stage 3 |
| `prod` | Production deployments, promoted from staging | Stage 5 |

Run every Atlas command from the unit's directory.

## Detect the Stage

1. `atlas whoami`. If it fails, ask the user to log in, then continue.
2. `git fetch origin`. Read the files below from the default branch
   (`git show origin/<default>:<path>`), not from the working tree.
3. Look for work an earlier session left open: `git branch -r --list 'origin/atlas-onboard/*'` and open
   PRs (`gh pr list` or `glab mr list` when installed; otherwise ask). Continue on an open branch instead
   of starting another one.
4. Rerun the scan rows of stage 0 (0.1) for the CI system and the deploy tooling. They need no database.
5. For each unit in `.atlas-onboarding.json`, pilot first, find the last finished stage. Check from
   stage 5 down; the first check that passes is the last finished stage. A stage listed under
   `deferred` for the unit counts as finished.

| Stage | Finished when |
|-------|---------------|
| 5 | The `prod` env promotes from staging (`to_version`, a `version` in `src`, or a `deny` check), and `atlas cloud migration list --repo <repo> --env-name prod --status PASSED` lists a deployment |
| 4 | `atlas cloud migration list --repo <repo> --env-name staging` shows `PASSED` or `NO_ACTION` for the version `atlas migrate ls --dir atlas://<repo> --latest --short` prints |
| 3 | A workflow on the default branch runs Atlas lint (versioned) or plan (declarative) for the unit, and its last push run on the default branch passed |
| 2 | The project PR is merged: `atlas.hcl` and the source are on the default branch, and versioned units also have `migrations/atlas.sum` there |
| 1 | The unit's `atlas.hcl` and source exist on its onboarding branch, and `atlas schema apply --env local --dry-run` reports no changes |
| 0 | The unit is in `.atlas-onboarding.json` |

## Checklist

Copy this into the conversation and tick items as they close:

```
Stage 0  [ ] scan  [ ] atlas login  [ ] inventory  [ ] units + pilot (user)  [ ] workflow (user)
Stage 1  [ ] atlas.hcl  [ ] desired state (export or ORM)  [ ] dry run: no changes
Stage 2  [ ] baseline (versioned)  [ ] new local database from the directory (versioned)  [ ] change loop shown  [ ] lint shown (versioned)  [ ] project PR
Stage 3  [ ] bot token + secret (user)  [ ] first push (ask)  [ ] CI PR  [ ] green run  [ ] CI pushed the merge
Stage 4  [ ] cutover plan  [ ] deploy config PR  [ ] staging dry run  [ ] staging deployment recorded
Stage 5  [ ] prod env with promotion (when asked)  [ ] prod deploy behind approval  [ ] first prod deploy (user)
Hand off [ ] AGENTS.md  [ ] skill in repo  [ ] next steps  [ ] summary + next unit
```

## Decisions File

`.atlas-onboarding.json` at the repository root holds only what the repository cannot show. Update it
in place and commit each change with the PR of the stage that made it.

```json
{
  "workflow": "versioned",
  "units": [
    { "name": "identity", "dir": "services/identity/db", "repo": "identity", "schemas": ["identity"], "pilot": true },
    { "name": "billing", "dir": "services/billing/db", "repo": "billing", "schemas": ["billing"] }
  ],
  "deploy": { "target": "argocd", "gitops_repo": "github.com/acme/gitops" },
  "deferred": [{ "unit": "billing", "stage": 4, "reason": "deployments are owned by the platform team" }]
}
```

`workflow` is `versioned` or `declarative`. `repo` is the registry repo slug. Never store database URLs,
tokens, or status in this file: status comes from the checks above.

## Terms

- Desired state: the schema as it should be, written as SQL, HCL, or ORM models. Both workflows start
  from it.
- Versioned workflow: `atlas migrate diff` writes each change to the desired state as a migration file,
  reviewed and applied in order. Declarative workflow: no migration files; `atlas schema plan` and
  `atlas schema apply` plan each change against the target database, and the plan is reviewed.
- Dev database: a temporary, isolated database Atlas uses as a sandbox to plan and check changes, usually
  an ephemeral Docker container (https://atlasgo.io/concepts/dev-database#introduction). It
  never holds data and is not the application's development database.
- Baseline: the first migration file of a versioned unit, a snapshot of the existing schema. Existing
  databases record it as applied without running it.
- Unit: one Atlas project, a schema or a set of tables one team owns. Pilot: the first unit, taken
  through every stage before the others.
- Atlas Registry: the copy of the migration directory or schema in Atlas Cloud that CI pushes and
  deployments read.
- Bot token: an Atlas Cloud credential for CI, not tied to a person.
- Environment promotion: production applies only the version already deployed to staging.

## Hand Off

When the pilot finishes stage 5, or the last stage the user wants:

1. Add an Atlas section to `AGENTS.md`, creating the file if needed. If the repository has a
   `CLAUDE.md`, add the line `@AGENTS.md` to it.

   ```markdown
   ## Database schema (Atlas)

   - Unit `<unit>` in `<dir>`: `atlas.hcl`, desired state in `schema/` (or the ORM models), migrations in
     `migrations/`. Run Atlas from `<dir>`.
   - Change the desired state, then run `atlas migrate diff --env local <name>`. A new file in `schema/`
     needs an `-- atlas:import` line at the top of `schema/main.sql`.
   - Never edit an applied migration. After editing an unapplied one, run `atlas migrate hash --env local`.
   - Before opening a PR: `atlas migrate lint --env local --latest 1`.
   - Deployments run from <CI workflow or GitOps target>, staging first, then production promoted from
     staging. Never run `migrate apply` or `schema apply` against a shared database from a laptop.
   - Use the `atlas` skill for schema work.
   ```

   For declarative units, the change lines are: change the desired state, then `atlas schema apply
   --env local` against the local database; CI plans the change for review on the PR.
2. Offer to commit the skills, so every teammate's agent gets them:
   - Claude Code: merge these keys into `.claude/settings.json`. Claude Code installs the Atlas plugin
     for each teammate who trusts the repository folder.

     ```json
     {
       "extraKnownMarketplaces": {
         "ariga": { "source": { "source": "github", "repo": "ariga/atlas" } }
       },
       "enabledPlugins": { "atlas@ariga": true }
     }
     ```

   - Codex, Cursor, GitHub Copilot: run `npx skills add ariga/atlas -a <agent> -y` from the repository
     root, with `codex`, `cursor`, or `github-copilot`, and commit `.agents/skills/` and
     `skills-lock.json`. If the install already put them there, commit them unchanged.
3. Next steps: offer the practices from the guides that fit what the scan found, one at a time. The
   full list is https://atlasgo.io/guides/evaluation/advanced-topics.

   | Practice | Guide |
   |----------|-------|
   | Production applies only reviewed and approved migrations | https://atlasgo.io/guides/reviewed-approved-migrations |
   | Block accidental drops in CI, with a deprecation workflow | https://atlasgo.io/guides/destructive-change-policy |
   | Detect drift at deploy time and between deployments | https://atlasgo.io/guides/drift-detection |
   | Test functions, views, triggers, and data migrations | https://atlasgo.io/testing/schema |
   | Lock-safe `NOT NULL` changes on PostgreSQL | https://atlasgo.io/guides/lock-safe-not-null |
   | Teams change only the objects they own | https://atlasgo.io/guides/schema-ownership |
   | Roles, permissions, and row-level security as code | https://atlasgo.io/guides/security-as-code |
   | Staged rollouts for database-per-tenant | https://atlasgo.io/guides/database-per-tenant/rollout |

4. Summarize per stage: what exists now, the doc page for it, what was deferred and why, and which units
   remain. For the next unit, the user runs this skill again; Detect the Stage starts it at stage 1.

## Docs Map

| Stage | Read |
|-------|------|
| 0 | [Connect](https://atlasgo.io/guides/evaluation/connect), [Choose a workflow](https://atlasgo.io/guides/evaluation/project-structure#choose-a-workflow), [Declarative vs versioned](https://atlasgo.io/concepts/declarative-vs-versioned) |
| 1 | [Verify Atlas and export your schema](https://atlasgo.io/guides/evaluation/verify-atlas), [Schema as code](https://atlasgo.io/guides/evaluation/schema-as-code), [Dev database](https://atlasgo.io/concepts/dev-database), [ORM guides](https://atlasgo.io/orms) |
| 2 | [Set up your migration workflow](https://atlasgo.io/guides/evaluation/setup-migrations), [Schema change workflow](https://atlasgo.io/guides/evaluation/developer-workflow), [Migration linting](https://atlasgo.io/versioned/lint) |
| 3 | [Database CI/CD](https://atlasgo.io/guides/evaluation/ci-cd), [Versioned CI/CD](https://atlasgo.io/versioned/setup-cicd), [Declarative CI/CD](https://atlasgo.io/declarative/setup-cicd) |
| 4 | [Deployments in Atlas Cloud](https://atlasgo.io/cloud/deployment), [Deployment guides](https://atlasgo.io/guides/deploying/intro), [Existing databases](https://atlasgo.io/versioned/apply#existing-databases) |
| 5 | [Environment promotion](https://atlasgo.io/guides/environment-promotion), [Declarative environment promotion](https://atlasgo.io/guides/environment-promotion-declarative) |

## Gotchas

- The org pin in `atlas.hcl` protects only commands that read it (`--env`). Commands that take `--url`
  directly run logged out, so run `atlas whoami` before them.
- A command-line `--exclude` replaces the env's `exclude` list instead of adding to it. Keep exclusions
  in the env and pass no `--exclude` with `--env`.
- Exclude patterns follow the URL scope. With a schema-scoped URL, a pattern names objects in that
  schema: `audit_*` for tables, `*[type=function]` for every function. With a database-scoped URL, the
  first segment names the schema: `public.audit_*`, `public.*[type=function]`, or `internal` to skip a
  whole schema.
- The dev database must match the target's engine version and scope (https://atlasgo.io/concepts/dev-database).
  The wrong scope fails with `modify schema "<name>" is not allowed when migration plan is scoped to one schema`, or silently drops extensions.
- The same difference on every table (collation, charset, owner) means the dev database's defaults
  differ from the target's. Fix the dev database; never write a migration for it.
- A split SQL source (`schema/main.sql` plus one file per object) reads only the files `main.sql`
  imports. A new file needs an `-- atlas:import <path>` line in the directive block at the top of
  `main.sql`, with no blank line before it; otherwise it is ignored and `migrate diff` reports no changes.
- `Error: missing scheme` on an env that reads `getenv()` means the variable is not set in the agent's
  shell, not that a stage failed.
- Without the baseline version, the first `migrate apply` on an existing database aborts with
  `connected database is not clean`. Set `baseline`; never get past it with `--allow-dirty`.
- Codex runs commands in its `workspace-write` sandbox by default, with outbound network access off
  until the user approves it. Atlas needs the network for Atlas Cloud and the database, so ask the user
  to approve Atlas commands, or to enable network access, before Detect the Stage.
- Pull requests opened by a cloud agent may not run CI until a human approves the run. Tell the user when
  a check is waiting on that.
