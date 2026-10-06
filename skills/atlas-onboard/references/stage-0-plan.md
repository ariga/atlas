# Stage 0: Plan

Goal: know what the repository and its databases contain, log in, and agree with the user on the units,
the pilot, and the workflow. This stage writes only `.atlas-onboarding.json`, and nothing touches a
database except reads.

## 0.1 Scan the Repository

No login or database access needed. Read configuration for names and versions only: grep for image tags,
engine names, and variable names. Never open `.env`, `*.tfvars`, or secret files, and never repeat a
password or connection string. Report the findings as a short table before doing anything else.

| Look for | Where | Feeds |
|----------|-------|-------|
| Engines and major versions | `docker-compose*.yml` image tags, ORM config, Helm values, Terraform database resources | The dev database (stage 1). If the version is not in the repository, ask: this is a decision point |
| Schema source | ORM models (GORM, Drizzle, SQLAlchemy, Django, Ent, Sequelize, TypeORM, Prisma), `*.sql` or `*.hcl` schema files | Stage 1 |
| Current migration tool | Flyway `V1__*.sql` and `flyway.conf`, Liquibase changelogs, golang-migrate `*.up.sql`, goose `-- +goose Up`, dbmate `-- migrate:up`, ORM migration folders (Django, Alembic, Prisma Migrate) | The tool stage 4 retires, and its history table to exclude |
| Objects outside the ORM | `RunSQL` or raw `CREATE VIEW`, `CREATE TRIGGER`, `CREATE FUNCTION`, `CREATE EXTENSION` in migrations | 0.2, stage 1 |
| Atlas already present | `atlas.hcl`, `atlas.sum`, `.atlas-onboarding.json` | Resume with Detect the Stage |
| CI system | `.github/workflows/`, `.gitlab-ci.yml`, `.circleci/`, `bitbucket-pipelines.yml`, `azure-pipelines.yml`, `Jenkinsfile` | Stage 3 |
| Deploy tooling | Argo CD `Application`, Flux `Kustomization`, `*.tf`, Helm migration hooks, a release or pre-deploy command, an entrypoint that runs migrations | Stage 4. If none is found, ask where the services deploy from |
| The agent running this skill | Claude Code, Codex, Cursor, GitHub Copilot | Where to commit the skill at hand off |

## 0.2 Log In

```bash
atlas version
```

If the command is not found, ask before installing the CLI: `curl -sSf https://atlasgo.sh | sh` on macOS
and Linux, or `brew install ariga/tap/atlas`. Other methods are at https://atlasgo.io/getting-started.
Then check the login:

```bash
atlas whoami
```

If it fails, ask the user to run `atlas login` in their own terminal. It opens a browser; on the first
login the user creates an account and an organization. Explain why it comes first, naming the objects
the scan found: logged out, inspect and diff cover schemas, tables, columns, indexes, and constraints
only, and skip views, materialized views, functions, procedures, triggers, sequences, domains, and
extensions without an error. Some drivers, such as SQL Server and ClickHouse, need a login for every
command. The later stages need it too: lint, the Atlas Registry, CI, and deployment reporting.

If the user declines, stop before 0.3 for any existing database. Offer the scan report only.

## 0.3 Inventory

Read the schema from a local or development database. The recommended source is the database the
application uses in development, such as a docker compose service or a dev container, after its current
migrations ran: it has the same schema as production, and nothing the agent runs can affect a shared
environment. The agent may find it and connect on its own, with the local credentials the compose file
or the dev setup defines, and set `LOCAL_DATABASE_URL` to it. Connecting the agent to a production
database is not recommended.

Read a remote environment, such as staging, only when the user asks for it and provides the connection:
the environment variable that holds the URL, or the cloud sign-in to use. Never look for remote URLs,
credentials, or environment variables on your own. Recommend a read-only role that can read every table
in the schemas the application uses; tables the role cannot read can drop out of the inspection without
an error. Managed databases that sign in with a cloud identity instead of a password (AWS IAM,
Microsoft Entra ID, GCP IAM) get their URL from `atlas.hcl`: write that env first (stage 1, Remote
Databases) and run the commands below with `--env <name>` instead of `--url`.

The URL must use the Atlas format (`postgres://`, `mysql://`), not JDBC.

The URL's scope decides what Atlas sees (https://atlasgo.io/concepts/url#scope):

- Schema scope: the URL names one schema, with `search_path=<schema>` in PostgreSQL, the database name
  in the MySQL path, or `mode=schema` in SQL Server. Atlas inspects, plans, and applies changes inside
  that schema only, and writes DDL without schema qualifiers (`users`, not `public.users`).
- Database scope: the URL names no schema, such as `postgres://host:5432/app` or `mysql://host:3306/`.
  Atlas covers every schema and qualifies the DDL it writes (`public.users`).

The inventory uses database scope, so it sees every schema before the units are decided; each unit picks
its own scope in stage 1. In MySQL, where a schema is a database, use the server URL only when the
application uses more than one database.

```bash
test -n "$LOCAL_DATABASE_URL" && echo set || echo missing

# Tables per schema
atlas schema inspect --url "$LOCAL_DATABASE_URL" --format '{{ json . }}' \
  | jq -c '.schemas[] | {name, tables: ((.tables // []) | length)}'
```

The table count is often enough. When the database also has views, functions, triggers, or other
objects, count them by type: export the database to one file per object in a temporary directory, the
same export stage 1 uses (https://atlasgo.io/inspect/database-to-code), and count the files.

```bash
inv=$(mktemp -d)
atlas schema inspect --url "$LOCAL_DATABASE_URL" --format "{{ sql . | split | write \"$inv\" }}"
find "$inv" -name '*.sql' ! -name main.sql | sed -E "s|^$inv/||; s|/[^/]+\$||" | sort | uniq -c
rm -rf "$inv"
```

Then map tables to the code that writes them: search the repository for each table name in models and
SQL strings, and group tables by service or package. Roles and permissions are excluded from inspection
by default; ask whether the team wants Atlas to manage them, and see stage 2, Roles and Permissions.

Report the inventory in this shape:

```
Database: app (PostgreSQL 15, read through LOCAL_DATABASE_URL)
  schema    tables  views  mat. views  functions  triggers  written by
  billing   55      4      0           9          3         services/billing
  identity  20      0      0           1          0         services/identity
  catalog   31      0      2           0          0         services/catalog
  public    1       0      0           0          0         schema_migrations (current migration tool)
  extensions: pgcrypto, pg_trgm (schema public)
```

## 0.4 Units and a Pilot (decision point)

A unit is one Atlas project: its own directory, `atlas.hcl`, and registry repo (see Project Layout in
`SKILL.md`). Propose units along ownership lines:

| Situation | Unit | Mechanism |
|-----------|------|-----------|
| Each service owns its own schema | One per schema | Schema-scoped URLs: `search_path=<schema>` in PostgreSQL, the database name in MySQL, `mode=schema` in SQL Server |
| A service owns several schemas | One, covering those schemas | Database-scoped URLs and `schemas = ["a", "b"]` on the envs |
| Services share one schema | One per service, split by table | `exclude` on the envs: `["audit_*"]` with a schema-scoped URL, `["billing.audit_*"]` with a database-scoped one |
| Many databases share one schema (tenants) | One for all of them | `for_each` on the env (`atlas/references/versioned.md`, Multi-tenant apply) |
| Teams share a unit but must not change each other's tables | One, plus a lint rule | `lint { ownership "github" { ... } }` (https://atlasgo.io/guides/schema-ownership) |

Schema-scoped units can use objects in a schema no unit owns, such as extensions in `public`. Keep those
objects out of every unit, and create them in each unit's dev database with a `docker` block `baseline`
(stage 1, and https://atlasgo.io/concepts/dev-database#baseline-schema). New databases need them before
the first unit deploys. Never put the old tool's history table in a unit.

Pick a pilot: the smallest unit with recent schema changes and the fewest objects that need special
handling (extensions, partitions, triggers). Then ask one question that states the counts:

```
I found 3 schemas, 106 tables, 4 views, 2 materialized views, 10 functions, and 3 triggers, plus
pgcrypto and pg_trgm in public. I propose three units: billing, identity, and catalog, with the
extensions and the migration tool's history table left out. Pilot: identity (20 tables, one
function, changed 5 times this quarter). Does this match how your teams own the database?
```

Record the answer in `.atlas-onboarding.json`: `units` with `name`, `dir`, `repo` (default: the unit
name), `schemas`, and `"pilot": true` on one.

## 0.5 Workflow (decision point)

In both workflows, developers define the desired state of the schema as code. The difference is how
changes reach a database (https://atlasgo.io/guides/evaluation/project-structure#choose-a-workflow):

- Versioned: `atlas migrate diff` writes each change as a migration file, checked into source control,
  reviewed in the PR, and applied in order. Every database replays the same files, and the history is
  what gets promoted and audited.
- Declarative: `atlas schema plan` computes the change from the live schema to the desired state, the
  plan is reviewed and approved, and `atlas schema apply` runs it. There are no migration files.

Default to versioned: it is the common choice for shared environments such as staging and production.
Propose declarative when the project already applies a desired state without migration files, or when
the user asks for it. Ask in one question that says what changes for developers, and link
https://atlasgo.io/concepts/declarative-vs-versioned. Record `workflow` as `versioned` or `declarative`.

When the ORM's own migration tool is in use (Django `makemigrations`, Alembic, Prisma Migrate), say what
happens to it: stage 4 retires it as the deployer, and the user decides whether developers keep running
it for other purposes, such as building test databases.

## Check

`.atlas-onboarding.json` has `workflow` and `units`, with one pilot, and the user approved both.
Nothing is committed yet: the project PR in stage 2 carries this file.
