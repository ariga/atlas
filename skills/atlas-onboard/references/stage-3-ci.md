# Stage 3: CI

Goal: every pull request that touches the unit gets an Atlas lint report (or a plan, for declarative
units), and every merge to the default branch pushes a new version to the Atlas Registry. Start after
the project PR from stage 2 is merged. Nothing deploys in this stage.

This stage decides which workflows to add, handles the manual steps, and checks the result. Each CI
system has a step-by-step guide for each workflow; follow the one that matches the scan and the
workflow from stage 0. `atlas/references/cicd.md` summarizes the GitHub Actions pipelines and the
other platforms' integrations.

| CI system | Versioned | Declarative |
|-----------|-----------|-------------|
| GitHub Actions | https://atlasgo.io/guides/ci-platforms/github-versioned | https://atlasgo.io/guides/ci-platforms/github-declarative |
| GitLab CI | https://atlasgo.io/guides/ci-platforms/gitlab-versioned | https://atlasgo.io/guides/ci-platforms/gitlab-declarative |
| Bitbucket Pipelines | https://atlasgo.io/guides/ci-platforms/bitbucket-versioned | https://atlasgo.io/guides/ci-platforms/bitbucket-declarative |
| CircleCI | https://atlasgo.io/guides/ci-platforms/circleci-versioned | https://atlasgo.io/guides/ci-platforms/circleci-declarative |
| Azure DevOps, code on GitHub | https://atlasgo.io/guides/ci-platforms/azure-devops-github#versioned-migrations-workflow | https://atlasgo.io/guides/ci-platforms/azure-devops-github#declarative-workflow |
| Azure DevOps, code in Azure Repos | https://atlasgo.io/guides/ci-platforms/azure-devops-repos#versioned-migrations-workflow | https://atlasgo.io/guides/ci-platforms/azure-devops-repos#declarative-workflow |

All the CI guides are listed at https://atlasgo.io/guides.

## 3.1 Decide

- CI system: from the scan, with its guide from the table above. Anything else, such as Jenkins or
  TeamCity, installs the CLI (`curl -sSf https://atlasgo.sh | sh`), runs
  `atlas login --token "$ATLAS_TOKEN"`, then the same commands with `--env ci`.
- Optional jobs, off by default for the pilot: automatic migration generation and automatic rebase of
  `atlas.sum` conflicts. Which platforms have them, and the extra token they need so their commits
  trigger lint, is in `atlas/references/cicd.md`. Offer them once several developers change the schema.
  For ORM units, the migration generation job also fails a PR whose models and directory disagree.

## 3.2 Credentials (manual step)

CI authenticates with a bot token. Only an organization admin can create one, and only in the Atlas
Cloud UI: ☰ > Settings > Bots > Create Bot. Create it in the organization the `atlas` block of
`atlas.hcl` pins; a token from another organization fails every command that reads the file. The token
is shown once. Ask the user to create it and store it as a CI secret themselves; it never passes
through the conversation.

| CI | Where | Name |
|----|-------|------|
| GitHub Actions | Settings > Secrets and variables > Actions, or `gh secret set ATLAS_CLOUD_TOKEN` in their own terminal | `ATLAS_CLOUD_TOKEN` |
| GitLab CI | Settings > CI/CD > Variables, masked, with Protect variable unchecked so merge request pipelines can read it | `ATLAS_CLOUD_TOKEN` |
| CircleCI | Organization Settings > Contexts | `ATLAS_TOKEN` |
| Bitbucket | Repository settings > Repository variables, secured | `ATLAS_TOKEN` |
| Azure DevOps | Pipelines > Library, variable group, marked secret | `ATLAS_TOKEN` |

If the repository already has an Atlas token secret under another name, reuse it and write that name
into the workflows. When the registry repo has protected flows enabled, the bot needs the Writer role
on it (https://atlasgo.io/cloud/roles-and-permissions).

Atlas posts its lint report, or the plan, as a comment on the pull request. Each CI system needs its
own credential for that, as its guide describes:

| CI | Comment credential |
|----|--------------------|
| GitHub Actions | None to create: the workflow sets `GITHUB_TOKEN: ${{ github.token }}` and `permissions: pull-requests: write` (both in the `atlas/references/cicd.md` templates) |
| GitLab CI | `GITLAB_TOKEN`: a project access token with the Reporter role and the `api` scope, stored as a CI/CD variable |
| Bitbucket | `BITBUCKET_ACCESS_TOKEN`: an app password with `pullrequest:write`, stored as a secured variable |
| CircleCI | `GITHUB_TOKEN`: a GitHub personal access token with the `repo` scope, and `GITHUB_REPOSITORY` (`owner/repo`), in the context |
| Azure DevOps, code on GitHub | A GitHub service connection, with OAuth or a personal access token |
| Azure DevOps, code in Azure Repos | The Build Service account allowed to Contribute and to Contribute to pull requests on the repository |

## 3.3 First Push (ask first)

The first push creates the registry repo, so lint on the CI PR has a version to compare with. Push from
an up-to-date default branch, and make sure the name is free:

```bash
git switch <default> && git pull --ff-only
atlas cloud repo describe --slug <repo>
```

If the repo exists and is not this unit's, agree on another slug with the user and record it as the
unit's `repo` in `.atlas-onboarding.json`. Then ask "Create registry repo `<repo>` in `<org>` with this
push?" and push:

```bash
atlas migrate push --env local <repo>        # versioned
atlas schema push --env local <repo>         # declarative
```

For ORM units, `--env local` runs the loader. If the loader's dependencies are not installed in this
shell, pass `--dir file://migrations --dev-url "<dev URL>"` instead of `--env local`.

Teams that keep every Atlas Cloud change in CI can create the repo with the `create-repo` action instead,
in a workflow run once by hand (`atlas/references/cicd.md`). The repo then starts empty, and its first
version arrives with the first merge.

## 3.4 Workflows (one PR)

Create the branch `atlas-onboard/<unit>-3` (Rule 4), then:

1. Add `env "ci"` to the unit's `atlas.hcl`. It needs no database URL: lint and plans run on the dev
   database and compare with the registry (`atlas/references/cicd.md`, case 1).

   ```hcl
   env "ci" {                              # versioned
     dev     = local.dev
     exclude = local.exclude
     migration {
       dir = "file://migrations"
       repo {
         name = "<repo>"
       }
     }
   }

   env "ci" {                              # declarative
     dev     = local.dev
     exclude = local.exclude
     schema {
       src = local.src
       repo {
         name = "<repo>"
       }
     }
   }
   ```

   Units that manage roles and permissions copy the `mode` block and the dev database from stage 2.
2. Add the PR and merge pipeline from the platform's guide in the table above, with the secret names
   from 3.2. On GitHub Actions, `atlas/references/cicd.md` has the same pipelines: Versioned Pipeline
   (`atlas-ci.yaml`) or Declarative Pipeline (`atlas-plan.yaml`). Remove every apply step, such as the
   declarative `schema/apply` and the commented `migrate/apply`: stage 4 adds deployment.
3. For ORM units, every job whose env loads the models (the declarative plan, and the optional migration
   generation job) installs the ORM's runtime and its Atlas provider before Atlas runs, as the ORM's
   guide shows. For Django: `actions/setup-python`, then `pip install django atlas-provider-django`
   (https://atlasgo.io/guides/orms/django/linting).
4. Keep test steps only for tests that exist: `schema/test` when the unit has schema tests,
   `migrate/test` when it has migration tests.
5. Use the real default branch (`git symbolic-ref --short refs/remotes/origin/HEAD`), not `master`.
   Filter paths with `**` globs under the unit's directory, and set `working-directory` to that
   directory. A repository with several units runs one job per unit, each with its own directory and
   paths.
6. Lint runs only on pull requests, and push runs only on the default branch. Check every generated job
   for this, including jobs included from templates or components: on GitLab, give the included push job
   `rules` that limit it to the default branch.
7. Keep the push job's `latest` push on (the default of `migrate/push`). The stage 4 drift check reads
   the state that untagged pushes store; a pipeline that pushes only commit tags silently skips it.

The dev database in CI: hosted GitHub runners have Docker, so the `docker://` URL works. On runners
without a Docker daemon (GitLab's Docker executor, many self-hosted runners), point `dev` at a service
container attached to the Atlas jobs only, with the target's version and scope, or at an empty database
the runner can reach (`dev = getenv("DEV_URL")` from a CI secret,
https://atlasgo.io/concepts/dev-database#providing-your-own-dev-database). Never add services, images, or
variables at the top level of a pipeline that other jobs share.

Open the PR, titled `Add Atlas CI for <unit>`.

## Check

1. The PR's Atlas job passes. The PR adds no migration files, so lint has nothing to analyze and may post
   no report; the first report appears on the next schema PR. Read the result with `gh pr checks
   <number>`, `glab ci status`, or ask the user.
2. After the user merges, the push run on the default branch passed (`gh run list --workflow <file>
   --branch <default> --limit 1`, the GitLab pipeline page, or ask), and the registry has the merge
   commit's version:

   ```bash
   atlas migrate ls --dir "atlas://<repo>?tag=<merge commit sha>" --latest
   ```

   For declarative units, `atlas cloud repo describe --slug <repo>` shows the repo.
3. The Atlas check is required for merging, so a failing lint blocks the merge. The user sets this in
   the repository host's branch protection; on Azure Repos, the guide's branch policy with build
   validation set to Required does it.
4. Optionally, the user opens a draft PR with a sample change, such as the destructive change from stage
   2.3, sees the report as a PR comment, and closes it without merging.

Tell the user what CI now does on every PR and every merge. Link https://atlasgo.io/versioned/setup-cicd
(or https://atlasgo.io/declarative/setup-cicd).
