# Setup Reference: Install, Credentials, Permissions, Versions

What every Atlas job shares on GitHub Actions: installing the CLI and logging in, the tokens, the
`permissions` block, the inputs every step takes, dev databases on runners, and pinning versions. Other
CI platforms are in `references/platforms.md`. The input and output names of every step are in
`references/actions.md`.

## The First Steps of Every Job

```yaml
jobs:
  lint:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      pull-requests: write                  # only for steps that comment on the PR
    steps:
      - uses: actions/checkout@v4
      - uses: ariga/setup-atlas@v0
        with:
          cloud-token: ${{ secrets.ATLAS_CLOUD_TOKEN }}
      - uses: ariga/atlas-action/migrate/lint@v1
        with:
          env: ci
          dir-name: app
        env:
          GITHUB_TOKEN: ${{ github.token }}   # lets the step comment on the PR
```

- `ariga/setup-atlas` installs the Atlas CLI (`version`, default `latest`) and, given `cloud-token`, runs
  `atlas login --token`; the later steps of the job use that login. Every `ariga/atlas-action/*` step
  runs the `atlas` binary from `PATH`, so `setup-atlas` comes first in every job that uses them.
- The first `ariga/atlas-action/*` step of a job downloads the `linux/amd64` `atlas-action` binary of
  its ref (`@v1`) to `atlas-action` in the workspace root, and the later steps of the job reuse that
  file, so pin every step of a job to the same ref. Add `atlas-action` to `.gitignore` when a step
  commits (`migrate/hash`), or it can end up in the commit.
- `monitor/schema` is the exception to the login rule: it takes its own required `cloud-token` input.
- `setup-atlas` inputs: `version` (default `latest`), `flavor` (the driver that needs the extended
  build, such as `snowflake` or `spanner`), `cloud-token`. It runs on Linux, macOS, and Windows runners;
  the steps that follow it do not (Runners below).

## Credentials and Trust

CI authenticates with an Atlas Cloud bot token, not a person's login. An organization admin creates a
bot in Atlas Cloud under Settings > Bots (https://atlasgo.io/cloud/bots); its token is shown once. A
user without an account starts a free trial with `atlas login`, then creates the bot. The user stores
the token as a repository secret, `ATLAS_CLOUD_TOKEN`, which every job uses; it never passes through
the conversation.

A job that a branch of the repository triggers (`push`, `pull_request`) runs that branch's workflow
file and `atlas.hcl`, and `atlas.hcl` can run programs (`data "external"`, external schema loaders).
Anyone who can push a branch can therefore read every secret such a job gets; pull requests from forks
get none. Keep the database URLs out of those jobs:

| Job | Runs on | Database URLs |
|-----|---------|---------------|
| Lint, test, plan, push, approve | Pull requests, and merges to the default branch | None: only the dev database |
| Deploy, revert, Data Scripts | The default branch: on merge, on demand, or on a schedule | The deploy user's, or the script's own user; an environment secret |
| Drift, security scan, monitoring | Schedule | A read-only user's, an environment secret |

- On GitHub, store the database URLs as environment secrets (Settings > Environments), with
  deployment branches limited to the default branch. A job gets them only by naming the environment
  (`environment: production`), and only when it runs on that branch. Required reviewers pause every job
  that names the environment, so put them on the environments that deploy, and give scheduled jobs an
  environment without reviewers.
- Never run Atlas on `pull_request_target` or `workflow_run` with the pull request's code checked out:
  those events hand the repository's secrets to code from forks.
- With protected flows on a registry repo, only members and bots with the Writer role can push to it
  or approve its plans, so the bot needs that role (https://atlasgo.io/cloud/roles-and-permissions).
  Protected flows are an Atlas Pro feature: a user without it starts a free trial with `atlas login`.
  For stricter separation, a second bot with the Writer role can serve only the merge jobs, from an
  environment limited to the default branch, so pull request jobs cannot push or approve.
- A deploy job can hold no database password at all: the `aws_rds_token`, `gcp_cloudsql_token`,
  `aws_dsql_token`, and `azure_db_token` (Microsoft Entra ID) data sources in `atlas.hcl` mint a
  short-lived token from the runner's cloud identity, and `runtimevar` reads the password from a secret
  manager when the step runs (https://atlasgo.io/guides/deploying/secrets). Azure SQL takes the
  identity through `fedauth` in its `azuresql://` URL
  (https://atlasgo.io/concepts/url#supported-schemes).
- The other platforms keep database URLs on the default branch the same way: protected variables on
  GitLab, which only protected branches read; deployment variables on Bitbucket; restricted contexts
  on CircleCI; and environments with a branch control check on Azure DevOps
  (`references/platforms.md`).
- Run `setup-atlas` with a token in every job. Lint, test, push, plan, approve, drift, Data Scripts,
  security scan, `create-repo`, and deploying from `atlas://` fail without the login. Inspect and diff
  do not fail: logged out, they skip views, functions, triggers, extensions, and other objects that
  need the login, so `migrate/diff`, `schema/plan`, and `schema/apply` lose them. The log shows at
  most a notice that these objects were skipped, and later errors name what is missing, such as
  `pq: type "<type>" does not exist` for an extension's type (`references/troubleshooting.md`). A SQL
  schema file that defines them fails with an error containing `are available to logged-in users only`.
- Pin the organization in `atlas.hcl`: a step that reads the file then fails unless the job is logged
  in to that organization, so a missing or wrong token stops the job instead of losing objects
  (https://atlasgo.io/atlas-schema/projects#atlas-block). The pin applies to steps that read the file,
  so select an env in every step.

```hcl
atlas {
  cloud {
    org = "acme"
  }
}
```

## Permissions and Comment Tokens

| Steps | `permissions` | Also needs |
|-------|---------------|------------|
| `migrate/lint`, `schema/lint`, `schema/plan` | `contents: read`, `pull-requests: write` | `GITHUB_TOKEN: ${{ github.token }}` in the step's `env` |
| `migrate/diff`, `migrate/hash`, `migrate/autorebase` | `contents: write` | Checkout with `fetch-depth: 0` and a fine-grained token; a git identity (`references/diff.md`) |
| Everything else | `contents: read` | Nothing |

- The steps read the comment token from the `GITHUB_TOKEN` environment variable, never from an input.
  Without it the step logs `GITHUB_TOKEN is not set, the action may not have all the permissions`.
- A failed comment is logged (`failed to comment on the pull request: ...`) and does not change the
  step's result: lint still fails on lint errors and passes without them.
- Comments, file annotations, and suggestions appear only on `pull_request` events. On `push`,
  `workflow_dispatch`, and `pull_request_target`, the steps run without a pull request: no comment, and
  `schema/plan` names its plan after the commit instead of the PR.
- Commits pushed with the default `github.token` do not start other workflows. The steps that commit
  (`migrate/diff`, `migrate/hash`, `migrate/autorebase`) need the checkout to use a fine-grained token
  or a GitHub App token, or `migrate/lint` never runs on their commit. That token sits in the job's git
  config while the branch's code runs, so limit it to this repository's contents and pull requests
  (`references/diff.md`):

```yaml
- uses: actions/checkout@v4
  with:
    token: ${{ secrets.PAT }}
    fetch-depth: 0
- name: config git to commit changes
  run: |
    git config --local user.email "github-actions[bot]@users.noreply.github.com"
    git config --local user.name "github-actions[bot]"
```

## Inputs Every Step Takes

| Input | Use |
|-------|-----|
| `env` | The `atlas.hcl` env to use. Keep URLs, the dev database, the registry repo, and lint policy there. |
| `config` | Another config file, `file://path/atlas.hcl`; several files, one per line. |
| `vars` | Values for `atlas.hcl` input variables, as a JSON object. |
| `working-directory` | The unit's directory in a monorepo. Paths in annotations are relative to it. |
| `dev-url` | The dev database, when the env does not set one. |

An input set next to `env` becomes a CLI flag and overrides that attribute of the env. A list input
such as `exclude` replaces the env's list instead of adding to it.

```hcl
env "ci" {                                # pull request jobs: no database URL
  dev = "docker://postgres/17/dev?search_path=public"
  migration {
    dir = "file://migrations"
    repo {
      name = "app"                        # the registry repo; migrate/lint still needs dir-name: app
    }
  }
  lint {
    destructive {
      error = true
    }
  }
}

env "prod" {
  url = getenv("DATABASE_URL")            # names the schema too: ...?search_path=public
  migration {
    dir = "atlas://app"
  }
}
```

```yaml
- uses: ariga/atlas-action/migrate/apply@v1   # in a deploy job that names its environment
  with:
    env: prod
  env:
    DATABASE_URL: ${{ secrets.DATABASE_URL }}  # an environment secret
```

The dev URL above is schema-scoped (`search_path=public`), so the migration files hold unqualified
names, and the target URL must name the same schema. A directory that spans several schemas uses
database-scoped URLs for both (https://atlasgo.io/concepts/url#scope).

Never write a database URL with a password into a workflow file. Pass it from a secret, either as an
environment variable that `atlas.hcl` reads with `getenv()`, or as the `url` input.

A password with `@`, `#`, `/`, or another reserved character breaks the URL. Keep the password in its
own secret and escape it in `atlas.hcl` with `urlescape` (https://atlasgo.io/faq/url-special-chars):

```hcl
locals {
  db_pass = urlescape(getenv("DB_PASSWORD"))
}

env "local" {
    url = "postgres://user:${local.db_pass}@localhost:5432/database"
}
```

## Outputs

Give the step an `id` and read `${{ steps.<id>.outputs.<name> }}`. Outputs are also set when the step
fails (`error`, the report of a failed lint), so a later step with `if: failure()` or `if: always()` can
use them. The output names of every step are in `references/actions.md`; the lint report link is
`report-url`, not the `url` the action metadata lists.

## Runners

| Runner | Steps | Dev database |
|--------|-------|--------------|
| GitHub-hosted Ubuntu, x64 | Run as is | `docker://` URLs work: Docker is installed |
| Self-hosted Linux x64 | Run as is; the runner must reach the download hosts (below) | `docker://` if Docker runs there, otherwise a service container |
| Job with `container:` | Run as is | A service container; `docker://` fails with `exec: "docker": executable file not found in $PATH` |
| ARM Linux, macOS | Add `ariga/atlas-action/setup` first: the downloaded binary is `linux/amd64` | A service container on Linux; an external empty database on macOS |
| Windows | Not supported (`setup-atlas` installs the CLI, but the steps do not run) | |

## Dev Databases on Runners

Where `docker://` cannot run, a service container gives the steps their dev database:

```yaml
jobs:
  lint:
    runs-on: ubuntu-latest
    services:
      # Spin up a postgres:15 container to be used as the dev-database for analysis.
      postgres:
        image: postgres:15
        env:
          POSTGRES_DB: dev
          POSTGRES_PASSWORD: pass
        ports:
          - 5432:5432
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-start-period 10s
          --health-timeout 5s
          --health-retries 5
    steps:
      # ...checkout, setup-atlas...
      - uses: ariga/atlas-action/migrate/lint@v1
        with:
          dir: 'file://migrations'
          dir-name: 'app'
          dev-url: 'postgres://postgres:pass@localhost:5432/dev?search_path=public&sslmode=disable'
```

- In a `container:` job, reach the service by its name (`postgres:5432`), not `localhost`.
- The dev database runs the target's engine and major version, with the same extensions, and its URL has
  the target's scope (`atlas/references/schema-sources.md`, and `atlas-postgres` or `atlas-mysql` for the
  engine).
- A `docker://` URL starts a fresh container for each command. A service container is shared by every
  step of the job: objects that live outside a database, such as PostgreSQL roles, survive from one step
  to the next, so a second step that creates the same role fails with `role "..." already exists`. Give
  such steps their own job, or use `docker://`.

## Versions and Pinning

| Reference | What runs |
|-----------|-----------|
| `ariga/atlas-action/<step>@v1` | The latest v1 release, which moves with each release |
| `@v1.17.7` | That release |
| `@<40-character SHA of a release commit>` | That release, for v1.15.4 and later; older release commits have no published binary |
| `@<40-character SHA of any other commit>` | Fails with `Failed to download atlas-action binary from all sources`, unless the job first builds the binary with `ariga/atlas-action/setup@<same SHA>` |
| `@master` | A binary that stopped updating in March 2025: the steps run old code |
| `@<short SHA>`, another branch | Fails with `Invalid version: <ref>` |

```yaml
- uses: ariga/setup-atlas@v0
  with:
    cloud-token: ${{ secrets.ATLAS_CLOUD_TOKEN }}
    version: v1.x.y                         # a release tag; the default `latest` follows every release
- uses: ariga/atlas-action/setup@<sha>      # builds atlas-action from source; keeps setup-atlas
- uses: ariga/atlas-action/migrate/apply@<sha>
```

- The Atlas CLI version and the action version are separate. A CLI release that changes behavior reaches
  every job that installs `latest`. Pin `version` in `setup-atlas` where a broken pipeline costs more
  than a missed fix, and upgrade on purpose.
- The steps run on Node 24. A Node 20 deprecation warning means the workflow pins an old release.
- Short SHAs never work: use the full 40 characters.
- A pinned CLI version can be retired, and `setup-atlas` then fails with
  `The specified Atlas version does not exist or is no longer supported.` Move `version` forward.
  Dependabot updates the action refs, not the `version` input.

## Reaching the Database, GitHub Enterprise Server, and Firewalled Runners

- Deploy, drift, scan, monitoring, and script steps connect from the runner to the database. A
  GitHub-hosted runner reaches only a database open to its IP ranges; otherwise use a self-hosted runner
  in the database's network (https://atlasgo.io/guides/deploying/connect-to-db-from-github-actions).
  GitHub-hosted runners have no IPv6 route, so a database reachable only over IPv6, such as a direct
  Supabase connection without the IPv4 add-on, needs the provider's pooler from CI.

- Comments go to the API of the server the run comes from (`GITHUB_API_URL`), so GitHub Enterprise
  Server needs no setting for them. The actions themselves must be available on the instance (GitHub
  Connect, or a mirror of `ariga/setup-atlas` and `ariga/atlas-action`).
- The runner downloads from: `atlasgo.sh` and the binaries it fetches (`setup-atlas`),
  `atlasbinaries.com` with `release.ariga.io` as fallback (the `atlas-action` binary), and Atlas Cloud.
  A runner that cannot
  reach the binary hosts fails with `Failed to download atlas-action binary from all sources`; allow
  them, or build the binary with `ariga/atlas-action/setup`, which downloads Go and the Go modules
  instead. On GitHub Enterprise Server, `setup`'s optional `token` input takes a github.com token, used
  to download Go without rate limits.

## Monorepos

Each unit is its own Atlas project. Give each unit its own job, or a matrix entry, with its own
`working-directory`, `dir-name`, and path filter:

```yaml
strategy:
  matrix:
    unit:
      - { dir: services/billing/db, repo: billing }
      - { dir: services/identity/db, repo: identity }
steps:
  # ...checkout, setup-atlas...
  - uses: ariga/atlas-action/migrate/lint@v1
    with:
      working-directory: ${{ matrix.unit.dir }}
      env: ci
      dir-name: ${{ matrix.unit.repo }}
    env:
      GITHUB_TOKEN: ${{ github.token }}
```

The lint comment on a pull request is keyed by `dir-name`: two units linted with the same `dir-name`
overwrite each other's comment.

## `gh atlas init-action`

The GitHub CLI extension writes a starter workflow that runs `migrate lint` on pull requests and
`migrate push` on the main branch, and opens it as a pull request:

```bash
gh extension install ariga/gh-atlas
gh auth refresh -s write:packages,workflow
gh atlas init-action
```

It asks for a bot token and stores it as a repository secret named `ATLAS_CLOUD_TOKEN_<suffix>`; the
generated workflow lints and pushes with that secret. `gh atlas init-action --help` lists its flags.
