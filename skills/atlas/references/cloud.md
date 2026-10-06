# Atlas Cloud Reference (`atlas cloud`)

`atlas cloud` commands read and manage resources in Atlas Cloud: the Atlas Registry (repos), the
databases those repos are deployed to (targets), and the deployment events recorded each time
`atlas migrate apply` or `atlas schema apply` reports to the cloud. They do not connect to databases.

Use them for operational visibility and pre-flight checks: infrastructure summaries, database status,
failed deployments, and what is waiting to be deployed. Use local commands (`atlas migrate status`,
`atlas schema diff`, `atlas migrate apply`) to inspect or change an actual database.

## Prerequisites

Every `atlas cloud` command requires an Atlas Cloud login. Check first, and stop if it fails:

```bash
atlas whoami                    # current user and org
atlas login                     # interactive login
atlas login --token "$TOKEN"    # CI / non-interactive; ATLAS_TOKEN env var also works
```

The error `command requires 'atlas login'` means no session exists. `login with token failed:
Unauthorized` means a stale token: run `atlas logout`, then `atlas login`.

## Command Tree

```
atlas cloud
├── repo
│   ├── list          Registry repos with synced / failed / pending database counts
│   ├── describe      One repo: slug, type, driver, URL, database counts
│   ├── create        Create a repo before the first push
│   ├── lingraph      Lineage graph of a repo (object dependencies)
│   └── secgraph      Security graph of a repo (roles, grants, access paths)
├── database
│   ├── list          Every tracked database: env, status, current version
│   └── describe      One database, including last deployment time
└── migration
    ├── list          Deployment events, filterable by status, repo, env, database name
    └── describe      One deployment event, including per-target results
```

Every command accepts `--format <go-template>`. Use `--format '{{ json . }}'` for machine-readable
output. List commands accept `--page <n>`.

### `atlas cloud repo`

| Command | Flags |
|---------|-------|
| `repo list` | `--page` |
| `repo describe` | exactly one of `--id`, `--slug`, `--name` |
| `repo create` | `--type schema` (or `s`) / `--type migration_directory` (or `m`), `--name`, `--driver` (`postgres`, `mysql`, `mariadb`, `sqlite`, `mssql`, `clickhouse`, `cockroach`, ...), optional `--description`, `--skip-if-exists` |
| `repo lingraph` | `--slug`, optional `--open-lineage` for OpenLineage format |
| `repo secgraph` | `--slug` |

A repo is either a schema repo (declarative, pushed with `atlas schema push`) or a migration
directory (versioned, pushed with `atlas migrate push`). `repo list` and `repo describe` show how many
of the repo's databases are synced, pending, or failed.

### `atlas cloud database`

| Command | Flags |
|---------|-------|
| `database list` | `--env-name <env>` |
| `database describe` | exactly one of `--id`, `--ext-id` |

`database list` columns: `ID`, `REPO`, `NAME`, `ENV`, `STATUS`, `CURRENT VERSION`. `database describe` adds
`Last Deployment Time`.

| Status | Meaning |
|--------|---------|
| `SYNCED` | The database is on the latest version of its repo |
| `PENDING` | The database is behind the latest version pushed to the registry. A deployment is waiting to run |
| `FAILED` | The last deployment to this database failed |

Environment names match the `env` block names in `atlas.hcl` (`dev`, `staging`, `prod`; names vary
per project).

### `atlas cloud migration`

A migration event is one deployment: a `migrate apply` or `schema apply` reported to the cloud.

| Command | Flags |
|---------|-------|
| `migration list` | `--status` (repeatable: `PASSED`, `FAILED`, `NO_ACTION`, `DRY_RUN`), `--repo <slug>`, `--env-name <env>`, `--name <substring>` (database name contains), `--page` |
| `migration describe` | `--id <event-id>` |

`migration list` columns: `ID`, `REPO`, `TYPE`, `ENV`, `DATABASE`, `TARGETS`, `VERSION`, `STATUS`.
Multi-target deployments (multi-tenant) show `(multiple)` under `DATABASE` and a `succeeded/total` ratio
under `TARGETS`. `migration describe` adds `Completed At` and `Targets 8/8 succeeded`.

| Status | Meaning |
|--------|---------|
| `PASSED` | Deployment succeeded |
| `FAILED` | Deployment failed on at least one target |
| `NO_ACTION` | The target was already up to date, nothing ran |
| `DRY_RUN` | Preview only, nothing applied |

### Pagination

List commands return 20 records per page and end with a footer:

```
--------------------------------
Page: 1  Page Size: 20  Total: 45
```

If `Total > Page Size`, fetch `--page 2`, `--page 3`, ... until `Page == ceil(Total / Page Size)`.
Do not summarize a list as complete until every page is read. With `--format '{{ json . }}'` the same
metadata is in `PageInfo` (`Page`, `PageSize`, `Total`).

## Answering Operational Questions

| Question | Commands |
|----------|----------|
| Summarize our database infrastructure | `atlas whoami`, `atlas cloud repo list`, `atlas cloud database list` (all pages) |
| What is the status of our databases? | `atlas cloud database list`, optionally `--env-name <env>` |
| Which databases are waiting for a deployment? | `atlas cloud database list`, keep rows with `STATUS=PENDING` |
| Which deployments failed? | `atlas cloud migration list --status FAILED` (add `--env-name`, `--repo`, or `--name`) |
| Why did a deployment fail? | `atlas cloud migration describe --id <id>`, then `atlas cloud database describe --id <db>` |
| What version is production on? | `atlas cloud database list --env-name prod` |
| Is staging ahead of production? | `atlas cloud database list --env-name staging` vs `--env-name prod`, compare `CURRENT VERSION` |
| What ran in the last release? | `atlas cloud migration list --repo <slug> --status PASSED` |
| Which plans are waiting for approval? | `atlas schema plan list --env <name> --pending` (declarative repos) |
| What depends on this table? | `atlas cloud repo lingraph --slug <slug>` |
| Who can access what in this schema? | `atlas cloud repo secgraph --slug <slug>` |
| Is a repo healthy? | `atlas cloud repo describe --slug <slug>`, check the failed and pending counts |

### Infrastructure summary

When asked to summarize the infrastructure, status, or health of the account:

1. `atlas whoami` to name the org.
2. `atlas cloud repo list` for every repo and its synced / pending / failed counts.
3. `atlas cloud database list`, all pages, grouped by `ENV`.
4. `atlas cloud migration list --status FAILED`, all pages, for open failures.
5. For each repo that is a schema repo with a `plan` workflow, `atlas schema plan list --env <name> --pending`.

Report in this shape:

```
Org: acme (atlas whoami)

Repos (3): payments (migration dir, postgres), identity (schema, postgres), analytics (migration dir, clickhouse)

Databases (18):
  ENV         SYNCED  PENDING  FAILED
  prod        6       1        1
  staging     5       0        0
  dev         5       0        0

Waiting for deployment (PENDING):
  prod-eu   payments  prod  on 20260512141500

Failed deployments:
  #131  payments  prod  prod-us  20260514083000  FAILED  2026-05-15T09:30:00Z

Pending approvals: identity has 1 plan awaiting approval (atlas://identity/plans/20260515_add_index)

Next step: investigate event #131 (atlas cloud migration describe --id 131) before the next prod deploy.
```

State the filters used and which pages were fetched. Highlight `FAILED` first, `PENDING` second.

### "Who is waiting for deployment?"

Two different things can be waiting:

- A database with `STATUS=PENDING`: the registry has a newer version than the database. Deploying
  (`atlas migrate apply` with `dir = "atlas://<repo>"`, or the CI/CD pipeline, or the Kubernetes
  operator) will bring it forward. List them with `atlas cloud database list` and filter on `PENDING`.
- A declarative plan awaiting approval: `atlas schema plan list --env <name> --pending`. Approve with
  `atlas schema plan approve --url "atlas://<repo>/plans/<name>"` only when the user asks; approval
  authorizes the exact SQL in the plan to run on the next `atlas schema apply`.

Report both, and say which kind each item is.

## Agent Workflows

### Pre-deployment readiness

Before recommending `atlas migrate apply`, `atlas schema apply`, or a CI deploy to a shared environment:

```
- [ ] atlas whoami
- [ ] atlas cloud database list --env-name <env>
- [ ] atlas cloud migration list --status FAILED --env-name <env>
- [ ] atlas cloud repo describe --slug <slug>
- [ ] atlas migrate status --env <env>              # live database, local command
```

Decision rules:
- Any target `FAILED`: investigate before applying. `migration list --status FAILED` filtered to the
  env or repo, then `migration describe --id <id>`.
- Any target `PENDING`: a deployment may be in flight or a version is waiting. Confirm with the user
  before starting another.
- All targets `SYNCED` and no recent `FAILED` events: cloud state is healthy. Still run
  `atlas migrate status` and `atlas migrate lint` locally before applying.
- `repo describe` shows failed databases > 0: drill into `database list` and `migration list`.

### Investigate a failed deployment

1. `atlas cloud migration list --status FAILED --repo <slug>` (narrow with `--env-name` or `--name`).
2. `atlas cloud migration describe --id <id>` for the version, completion time, and per-target results.
3. `atlas cloud database describe --id <id>` on each affected target: current version vs expected.
4. `atlas migrate status --env <env>` to compare the live revision table with the cloud record.
5. Report: what failed, on which targets, at which version, and whether the database is now behind
   (`PENDING`) or stuck (`FAILED`).

### Audit environments

Read `atlas.hcl` for the `env` block names. For each:

```bash
atlas cloud database list --env-name <env>
atlas cloud migration list --env-name <env>
```

Compare `CURRENT VERSION` across environments to spot version skew. Production behind staging is
normal; production ahead of staging violates promotion policy and should be flagged.

### Set up a new registry repo

Creating a repo creates a resource in the user's organization: ask first, and check that the name is
free (`atlas cloud repo describe --slug <name>`). A push to an existing repo of the same name adds a
version to it.

1. The first push creates the repo: `atlas migrate push --env <name> <slug>` or
   `atlas schema push --env <name> <slug>`. To create an empty repo instead:
   `atlas cloud repo create --type migration_directory --name <slug> --driver postgres` (or
   `--type schema` for the declarative workflow).
2. Envs that lint or push keep the local directory and name the repo:
   `migration { dir = "file://migrations" repo { name = "<slug>" } }`, or `schema { repo { name = "<slug>" } }`.
3. Envs that deploy read from the registry: `migration { dir = "atlas://<slug>" }`.

### Environment promotion in atlas.hcl

The same database statuses are available inside `atlas.hcl` through the `cloud_databases` data source,
which lets a `prod` env pin `to_version` to the version already running in a lower environment:

```hcl
data "cloud_databases" "staging" {
  repo = "payments"
  env  = "staging"
}

env "prod" {
  url = getenv("DATABASE_URL")
  migration {
    dir        = "atlas://payments"
    to_version = data.cloud_databases.staging.targets[0].current_version
  }
}
```

## Cloud vs Local Commands

| Concern | Atlas Cloud (`atlas cloud ...`) | Local (`atlas migrate/schema ...`) |
|---------|-------------------------------|-----------------------------------|
| Registry repos and artifacts | `repo list`, `repo describe` | `migrate push`, `schema push` |
| Tracked database status and version | `database list`, `database describe` | |
| Deployment history | `migration list`, `migration describe` | |
| Plans awaiting approval | | `schema plan list --pending` |
| Live database revision table | | `migrate status` |
| Schema drift vs desired state | | `schema diff`, `migrate lint` |
| Apply changes to a database | | `migrate apply`, `schema apply` |

Cloud commands answer "what does Atlas Cloud know?" Local commands answer "what is actually in the
database?" Use both when validating readiness, and keep the two apart in the report.

## Reporting Results

1. State the org (`atlas whoami`), the filters used, and how many pages were read.
2. Lead with `FAILED`, then `PENDING`, then the healthy summary.
3. For deployment events include `ID`, `REPO`, `ENV`, `VERSION`, `STATUS`, and `TARGETS` when present.
4. Distinguish cloud-reported state from live database state. Recommend `atlas migrate status` when the
   user plans to apply changes.
5. Do not approve plans, create repos, or apply migrations from a status request. Suggest the command
   and let the user decide.

## Error Handling

| Error | Action |
|-------|--------|
| `command requires 'atlas login'` | Run `atlas login`, or set `ATLAS_TOKEN` and run `atlas login --token` |
| `login with token failed: Unauthorized` | `atlas logout`, then `atlas login` |
| `repository not found` / `database not found` / `migration not found` | Verify the identifier. Run the matching `list` command first to get the ID or slug |
| `invalid --status value` | Use `PASSED`, `FAILED`, `NO_ACTION`, or `DRY_RUN` |
| `invalid --type` | Use `schema`/`s` or `migration_directory`/`m` |
| Mutually exclusive flags (`--id` vs `--slug`) | Provide exactly one identifier |

## Documentation

- [Inspecting deployments from the CLI](https://atlasgo.io/cloud/deployment#inspecting-deployments-from-the-cli)
- [CLI reference: atlas cloud](https://atlasgo.io/cli-reference#atlas-cloud)
- [Declarative plan and approval](https://atlasgo.io/declarative/plan)
- [Environment promotion](https://atlasgo.io/guides/environment-promotion)
- [Schema registry](https://atlasgo.io/cloud/features/registry)
