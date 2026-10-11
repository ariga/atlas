# Snowflake Dev Database and Connections

The dev database is a temporary database Atlas uses to load the desired state, replay migrations, and
compute plans (https://atlasgo.io/concepts/dev-database). Snowflake does not run in Docker, so on
Snowflake the dev database is a second database that you create and dedicate to Atlas
(https://atlasgo.io/concepts/dev-database#providing-your-own-dev-database).

## Install and Log In

The Snowflake driver ships in the extended build of the CLI, and every command that opens a
`snowflake://` URL needs a login:

```bash
curl -sSf https://atlasgo.sh | ATLAS_FLAVOR="snowflake" sh
atlas login
```

| Error | Cause | Fix |
|-------|-------|-----|
| `sql/sqlclient: unknown driver "snowflake". See: https://atlasgo.io/url` | The community edition is first on `PATH`; it has no Snowflake driver | Install the extended build, and check `which atlas` |
| `The snowflake driver is available only as custom build.` | A build without the Snowflake driver compiled in | Install with `ATLAS_FLAVOR="snowflake"`, or use the `arigaio/atlas:latest-extended` image |
| `login is required to use the 'snowflake' driver` | No login | Ask the user to run `atlas login`; in CI, `atlas login --token "$ATLAS_TOKEN"` |

If `atlas whoami` shows no login, ask the user to run `atlas login`; a user without an account can start
a free trial with it.

## URL Format and Scope

The account identifier is the host, the database is the first path segment, and the schema the second
(https://atlasgo.io/concepts/url). Query parameters, such as `warehouse` and `role`, go to the Snowflake
driver unchanged:

| URL | Scope | Atlas covers | DDL it writes |
|-----|-------|--------------|---------------|
| `snowflake://user:pass@account/APP/PUBLIC?warehouse=WH` | Schema | The `PUBLIC` schema of `APP` | Unqualified (`"users"`) |
| `snowflake://user:pass@account/APP?warehouse=WH` | Database | Every schema in `APP`, plus account-level objects the env enables | Qualified (`"PUBLIC"."users"`) |

- The query form works too: `snowflake://user:pass@account?database=APP&schema=PUBLIC&warehouse=WH`.
- Atlas folds an unquoted schema name in the URL to uppercase, as Snowflake does. A lowercase or
  mixed-case schema needs double quotes, percent-encoded in the URL: `.../APP/%22app_v2%22`.
- What only database scope reads (Rule 2 in `SKILL.md`): account roles, database roles, users, grants,
  the database's own grants, warehouses, resource monitors, external volumes, and external access
  integrations. A schema-scoped URL never inspects them and never plans them, without an error.
- What works at either scope: tables of every kind, views, functions, procedures, stages, file formats,
  sequences, tasks, pipes, alerts, tags, policies, secrets, and network rules.
- At database scope, the desired state covers every schema in the database, so an undeclared schema,
  `PUBLIC` included, is planned for `DROP SCHEMA`. Declare `schema "PUBLIC" {}`, or name the managed
  schemas on the env: `schemas = ["PUBLIC", "BRONZE"]`.
- At schema scope, a migration plan cannot change the schema's own attributes: a different `comment`,
  `retention_time`, `managed_access`, or tag fails `migrate diff` with
  `modify schema "PUBLIC" is not allowed when migration plan is scoped to one schema`. Use database scope
  for both URLs to manage schema attributes.
- `exclude` and `include` patterns follow the scope and are case-sensitive. At schema scope a pattern
  names the object (`"EVENTS"`); at database scope it starts with the schema (`"BRONZE.*"`).
  Account-level objects take a type selector: `"an_*[type=role]"`, `"kim[type=user]"`,
  `"ETL_WH[type=warehouse]"`.

## The Dev Database

Create an empty database next to the target, on the same account, and give it the target's scope:

```hcl
locals {
  base = "snowflake://${getenv("SNOWFLAKE_USER")}:${urlescape(getenv("SNOWFLAKE_TOKEN"))}@${getenv("SNOWFLAKE_ACCOUNT")}"
  args = "warehouse=${getenv("SNOWFLAKE_WAREHOUSE")}&role=${getenv("SNOWFLAKE_ROLE")}"
}

env "prod" {
  url = "${local.base}/ANALYTICS?${local.args}"
  dev = "${local.base}/ANALYTICS_DEV?${local.args}"
  schemas = ["PUBLIC", "BRONZE"]
}
```

For a schema-scoped target, add a schema to both: `url = ".../ANALYTICS/BRONZE?..."` with
`dev = ".../ANALYTICS_DEV/PUBLIC?..."`. The dev schema can have any name, since schema-scoped statements
carry no qualifier; `PUBLIC` exists in every new database. The schema must exist, or Atlas fails with
`snowflake: schema "PUBLIC" was not found`.

What Atlas requires of it:

- Empty. Before each use Atlas checks the scope it connects to. At database scope, the dev database may
  hold only `PUBLIC`, and `PUBLIC` must be empty; any other schema fails with
  `connected database is not clean: found schema "BRONZE"`. At schema scope, any object in the schema
  fails, such as `connected database is not clean: found table "t" in schema "PUBLIC"`. An interrupted
  run can leave objects behind: drop them before the next run.
- Dedicated. Atlas creates and drops objects in it and restores it on exit. The role in the dev URL needs
  to create and drop every kind of object the schema declares. Environments can share one dev database
  only if they never run at the same time.
- Never a real environment's database. If it is empty, Atlas uses it as a sandbox; otherwise it fails
  with `connected database is not clean`.
- Planning is slow on Snowflake: every plan and lint run round-trips through the dev database. A lint
  of a few statements can take about a minute (https://atlasgo.io/guides/dbt/snowflake), and a
  database-scoped plan of about 30 objects, with roles, grants, and tags on, takes about 10 minutes.
  Budget for it in CI, and set command timeouts above it.

A `dev "snowflake"` block connects to the same database and adds a `baseline`, for objects the schema
references but does not manage, such as a table in another team's schema. The `dev` block needs
`atlas login` (https://atlasgo.io/concepts/dev-database#baseline-schema):

```hcl
dev "snowflake" "dev" {
  url      = "${local.base}/ANALYTICS_DEV?${local.args}"
  baseline = file("baseline.sql")
}

env "prod" {
  url = "${local.base}/ANALYTICS?${local.args}"
  dev = dev.snowflake.dev.url
}
```

## Account-Level Objects During Planning

A database has its own copy in the dev database; an account-level object does not. One account holds one
`ETL_WH` warehouse and one `ANALYST` role, and the dev database sits next to them.

- Account roles, users, and grants to account roles are never replayed on the dev database while Atlas
  loads the desired state: the declared ones are kept as declared. A user the account does not hold is
  still planned, and its `GRANT ROLE ... TO USER` fails when applied.
- Database roles belong to a database, so they are created in the dev database like tables.
- Warehouses, resource monitors, external volumes, and external access integrations have no database:
  loading a desired state that declares them creates them on the account, and Atlas drops them again when
  it restores the dev database. The dev role needs the privileges to create them (resource monitors need
  `ACCOUNTADMIN`), and a name the account already holds collides with the live object. Keep each one in
  an env scoped to it with `include`, and plan with names that do not exist yet.
- That collision holds after the first apply too: once a declared resource monitor exists, every later
  plan fails while loading the desired state, with
  `create resource monitor "<name>": 002002 (42710): SQL compilation error: ... already exists`. Create
  such objects once, then remove their blocks from the desired state, or keep them out of Atlas.
- Replaying a migration directory runs its SQL on the dev database as written. A `CREATE ROLE` or
  `CREATE WAREHOUSE` in a migration file runs against the account on every replay, and fails once the
  object exists. Keep account-level objects in a declarative env (`references/security.md`).

## Connection URLs

The URL forms the docs list (https://atlasgo.io/concepts/url):

```
# User and password, or a programmatic access token in the password position.
snowflake://<username>:<password>@<account_identifier>/<database>?warehouse=<warehouse>

# Key pair. <private_key> is the URL-safe base64 of the PKCS#8 DER key.
snowflake://<username>@<account_identifier>/<database>?warehouse=<warehouse>&authenticator=SNOWFLAKE_JWT&privateKey=<private_key>
```

- Percent-encode special characters in a password or token, or build the URL with `urlescape`, as in the
  `locals` block above.
- `role=` picks the role the session uses. A programmatic access token restricted to one role can only
  assume that role (https://atlasgo.io/guides/dbt/snowflake).
- Set `warehouse=` on the target and the dev URL. Atlas restores the session's warehouse after a plan
  creates or drops one, and an unnamed dynamic table refreshes on it (`references/objects.md`).
- The user needs to read every object Atlas manages: inspection sees only what its role can see. Grants
  to the role Atlas connects as are not inspected.

## Errors

| Error | Cause | Fix |
|-------|-------|-----|
| `connected database is not clean: found schema "..."` | The database-scoped dev database holds a schema other than `PUBLIC` | Drop it, or use a fresh database |
| `connected database is not clean: found table "..." in schema "..."` | The dev schema is not empty | Empty it, or use a fresh schema |
| `snowflake: schema "..." was not found` | The URL's schema does not exist, or was folded to uppercase | Create it, or quote a lowercase name as `%22name%22` |
| `modify schema "..." is not allowed when migration plan is scoped to one schema` | A schema attribute changed under a schema-scoped migration plan | Use database scope for both URLs |
| `drop schema "..." is not allowed when migration plan is scoped to one schema` | A schema-scoped plan that adds or drops schemas | Use database scope for both URLs |
| `Object '...' already exists` while reading the migration directory | An account-level `CREATE` replayed on the dev account | Move it to a declarative env |
| `invalid port ":..." after host` | A special character in the password breaks the URL | Percent-encode it, or use `urlescape` |
