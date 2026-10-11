# Platforms Reference: GitLab CI, CircleCI, Bitbucket Pipelines, Azure DevOps, Other CI

The pipelines and rules of the other references carry over to every platform. This file covers what
differs: how a step is invoked, how the login and the comment token are passed, how outputs are read, and
which steps a platform lacks. Input names per platform: `references/actions.md`.

| Platform | Steps | Comments on | Comment credential | Input defaults apply |
|----------|-------|-------------|--------------------|----------------------|
| GitLab CI | CI/CD catalog components, every step | Merge requests | `gitlab-token` input | Yes |
| CircleCI | `ariga/atlas-orb` commands, 12 steps | GitHub pull requests | `GITHUB_TOKEN`, `GITHUB_REPOSITORY` | The orb's own: `schema_push` defaults `latest` to `false` |
| Bitbucket Pipelines | The `arigaio/atlas-action` pipe, every step | Pull requests, plus Code Insights reports | `BITBUCKET_ACCESS_TOKEN` | No |
| Azure DevOps | The `AtlasAction` task, all steps but 9 | GitHub or Azure Repos pull requests | `githubConnection`, or `SYSTEM_ACCESSTOKEN` | No |

"Input defaults apply: No" means only the inputs you set reach the step: set `latest: true` on push steps
there, or the `latest` tag does not move. The orb and the Azure task are published on their own schedule
and lag the release; a step they lack runs as its CLI command in a script step.

## GitLab CI

Components come from the CI/CD catalog project `arigaio/atlas`, one per step, named with dashes:

```yaml
image: ubuntu:latest

services:
  - postgres:latest

variables:
  POSTGRES_DB: dev
  POSTGRES_USER: user
  POSTGRES_PASSWORD: pass

stages:
  - lint
  - push

include:
  - component: $CI_SERVER_FQDN/arigaio/atlas/migrate-lint@~latest
    inputs:
      stage: lint
      dir: "file://migrations"
      dir-name: "app"
      dev-url: "postgres://user:pass@postgres/dev?sslmode=disable"
      atlas-cloud-token: $ATLAS_CLOUD_TOKEN
      gitlab-token: $GITLAB_TOKEN

  - component: $CI_SERVER_FQDN/arigaio/atlas/migrate-push@~latest
    inputs:
      stage: push
      dir: "file://migrations"
      dir-name: "app"
      dev-url: "postgres://user:pass@postgres/dev?sslmode=disable"
      atlas-cloud-token: $ATLAS_CLOUD_TOKEN
    rules:
      - if: $CI_COMMIT_BRANCH == "main"
```

- Every component takes `stage`, `job_name` (default `atlas-<step>`), `atlas-version` (the CLI, default
  `latest`), `atlas-cloud-token` (`monitor-schema` takes `cloud-token` instead), `gitlab-token`, and the
  step's inputs in dash-case.
- `migrate-lint` and `migrate-diff` run only in merge request pipelines; `monitor-schema` runs on
  schedules or manually. The other components have no rules and run in every pipeline: give their
  `include` entry `rules`, as above, so push, approve, and apply run on the default branch only.
- The job installs `curl` with `apk` or `apt`, so its image must be Alpine- or Debian-based. It installs
  Atlas, logs in, and caches the grant in `.atlas/`.
- Credentials (`references/setup.md`): `ATLAS_CLOUD_TOKEN` is a masked variable, not protected, so
  merge request pipelines can read it. A merge request pipeline runs its branch's `.gitlab-ci.yml` and
  `atlas.hcl`, so it reads every variable it gets: database URLs go in protected variables, which only
  pipelines on protected branches read, and only components that run on the default branch take them.
- `GITLAB_TOKEN`: a project access token with the "Reporter" role and the API scope, enough to comment.
  `migrate-diff` pushes with it, so there it needs a role that can push, and the branch's code can read
  it (`references/diff.md`).
- Outputs go to a dotenv artifact as `ATLAS_OUTPUT_<STEP>_<OUTPUT>` (the lint report link is
  `ATLAS_OUTPUT_MIGRATE_LINT_REPORT_URL`), read by later jobs that list the job in `needs` with
  `artifacts: true`. Only components that declare outputs publish the artifact, so the `url` that
  `migrate-push` sets does not reach later jobs.
- Dev database: a `services:` database, with the service name as the host in its URL, as above.
- Self-managed GitLab: mirror the `arigaio/atlas` component project to the instance first.

## CircleCI

```yaml
version: 2.1

orbs:
  atlas-orb: ariga/atlas-orb@0.3.1

jobs:
  lint-migrations:
    docker:
      - image: cimg/base:current
      - image: cimg/postgres:15.0
        environment:
          POSTGRES_DB: postgres
          POSTGRES_USER: postgres
          POSTGRES_PASSWORD: postgres
    steps:
      - checkout
      - run:
          name: Wait for Postgres
          command: dockerize -wait tcp://127.0.0.1:5432 -timeout 60s
      - atlas-orb/setup:
          version: "latest"
      - atlas-orb/migrate_lint:
          env: dev
          dir_name: "app"
          dev_url: "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable&search_path=public"
workflows:
  version: 2
  atlas-workflow:
    jobs:
      - lint-migrations:
          context: dev                      # holds ATLAS_TOKEN, GITHUB_TOKEN, GITHUB_REPOSITORY
          filters:
            branches:
              ignore: main
```

- `atlas-orb/setup` installs the CLI (`version`) and logs in from the variable `cloud_token_env` names
  (default `ATLAS_TOKEN`). Commands are named with underscores, as are their parameters (`dir_name`).
- The published orb has commands for `migrate_lint`, `migrate_push`, `migrate_apply`, `migrate_down`,
  `migrate_test`, `migrate_auto_rebase`, `schema_plan`, `schema_plan_approve`, `schema_push`,
  `schema_apply`, `schema_test`, and `monitor_schema`. Run any other step as the CLI in a `run` step
  after `atlas-orb/setup`.
- `schema_push` defaults `latest` to `false`: set `latest: true`, or the `latest` tag does not move.
  `monitor_schema` sends no table statistics.
- CircleCI runs on pushes, not pull request events. With `GITHUB_TOKEN` set, the step looks up the open
  pull request for the branch and comments there; set `GITHUB_REPOSITORY` to `owner/repo`, or the
  lookup fails the step. Without a token: no comments.
- The images cannot run Docker-in-Docker, so `docker://` dev URLs do not work: pass `dev_url` to a
  database image in the job, as above.
- Outputs are exported to `$BASH_ENV` as `ATLAS_OUTPUT_<STEP>_<OUTPUT>` for the later steps of the job.
- Credentials: the context the jobs share holds `ATLAS_TOKEN`. Put the database URLs in a separate
  context restricted to the default branch, and attach it only to the jobs that run there
  (`references/setup.md`).

## Bitbucket Pipelines

```yaml
image: atlassian/default-image:3

definitions:
  commonItems:
    atlas-action-variables: &atlasVars
      ATLAS_TOKEN: $ATLAS_TOKEN # Required
      # The access token is required to comment on the PR
      # It must have Pull requests/Write permission
      BITBUCKET_ACCESS_TOKEN: $BITBUCKET_ACCESS_TOKEN
pipelines:
  pull-requests:
    '**':
      - step:
          name: 'Lint Migrations'
          script:
            - name: Lint migrations
              pipe: docker://arigaio/atlas-action:v1
              variables:
                <<: *atlasVars
                ATLAS_ACTION: migrate/lint # Required
                ATLAS_INPUT_ENV: dev
                ATLAS_INPUT_DIR_NAME: "app"
  branches:
    master:
      - step:
          name: 'Push Migrations'
          script:
            - name: Push migrations
              pipe: docker://arigaio/atlas-action:v1
              variables:
                <<: *atlasVars
                ATLAS_ACTION: migrate/push # Required
                ATLAS_INPUT_ENV: dev
                ATLAS_INPUT_DIR_NAME: "app"
                ATLAS_INPUT_LATEST: "true"
            - source .atlas-action/outputs.sh
```

- Use the `v1` tag or a release tag. The image's `master` and `latest` tags stopped updating in March
  2025.
- The image carries the Atlas CLI that was current when its release was built; there is no CLI version
  input. The CLI reads `ATLAS_TOKEN` from the pipe's variables.
- Inputs are `ATLAS_INPUT_<NAME>` variables, in upper case with underscores. Input defaults do not
  apply: set `ATLAS_INPUT_LATEST: "true"` on push steps.
- Credentials: the repository variable `ATLAS_TOKEN` reaches every pipeline, pull requests included.
  Keep the database URLs out of repository variables: put them in deployment variables, and give only
  the steps that deploy that `deployment` (`references/setup.md`). Limiting a deployment to the
  default branch needs Bitbucket Premium's deployment restrictions; without them, a branch's pipeline
  can name the deployment and read its variables.
- Comments go to the pull request only in `pull-requests` pipelines. Lint results are also posted as a
  Code Insights report on the commit, which needs no token.
- Outputs are written to `.atlas-action/outputs.sh` (or `ATLAS_OUTPUT_DIR`); `source` it in a later
  script line to get `ATLAS_OUTPUT_<STEP>_<OUTPUT>`.
- Dev database: a `docker://` dev URL in the env, or a database service in the step reached through
  `host.docker.internal`.

## Azure DevOps

Install the Atlas extension (`Ariga.atlas-action`) from the Visual Studio Marketplace; it adds the
`AtlasAction` task. The task does not install Atlas: install and log in with script steps first.

```yaml
pool:
  vmImage: ubuntu-latest

variables:
  - group: atlas-vars                       # ATLAS_TOKEN only: pull request builds load it too

steps:
  - checkout: self
    persistCredentials: true
    fetchDepth: 0

  - script: curl -sSf https://atlasgo.sh | sh
    displayName: Install Atlas

  - script: atlas login --token $(ATLAS_TOKEN)
    displayName: Atlas Login

  - task: AtlasAction@1
    condition: eq(variables['Build.Reason'], 'PullRequest')
    inputs:
      action: 'migrate lint'
      dir: 'file://migrations'
      dir_name: 'app'
      env: 'ci'
      githubConnection: 'GitHub Connection'
    displayName: Lint Migrations

  - task: AtlasAction@1
    condition: and(succeeded(), eq(variables['Build.SourceBranch'], 'refs/heads/main'))
    inputs:
      action: 'migrate push'
      dir: 'file://migrations'
      dir_name: 'app'
      latest: true
      env: 'ci'
    displayName: Push Migrations
```

- `action` names the step with spaces (`schema plan approve`); inputs are snake_case (`dir_name`,
  `dry_run`), except `githubConnection`.
- The task has no `create-repo`, `migrate set`, `migrate drift`, `script` (exec, query, loop, test,
  push), or `security scan` step, and no `collect_stats` input: run those as the CLI in a script step.
- Comments: on GitHub-hosted code, set `githubConnection` to a GitHub service connection. On Azure Repos,
  map `env: SYSTEM_ACCESSTOKEN: $(System.AccessToken)` on the task, and give the build service
  permission to contribute to pull requests. Bitbucket, TFVC, and other Git or Subversion repositories
  get no comments.
- Outputs are task output variables: name the step (`name: PushSchema`) and read
  `$(<name>.<output>)` in the same job.
- Credentials: a group under the pipeline's top-level `variables` reaches every build, pull requests
  included, so keep only the bot token there. Put the database URLs in a variable group or environment
  protected by a branch control check that allows only the default branch, and link it only in the
  stage that deploys (`references/setup.md`).
- Agents without Docker cannot start `docker://` dev databases: point `dev_url` at an empty database
  the pipeline can reach.

## Other CI Systems

On CI systems without an Atlas integration, such as Jenkins, TeamCity, Buildkite, Cloud Build, or Argo
Workflows, run the CLI: install it (`curl -sSf https://atlasgo.sh | sh`, or the `arigaio/atlas` image),
log in with `atlas login --token "$ATLAS_TOKEN"`, and run each step's command with `--env ci`
(`atlas/references/cicd.md`). There is no pull request comment; the commands print their reports, and
`--format '{{ json . }}'` makes them machine-readable. Run on Jenkins or Buildkite, the `atlas-action`
binary stops with `atlasaction: no action found for the current environment`. The `arigaio/atlas` image
has no shell, Docker client, or git: use its Alpine-based images for scripts, a separate dev database
container instead of `docker://`, and an image with git for `--git-base`.
