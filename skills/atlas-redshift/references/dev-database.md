# Redshift Dev Database and Connections

The dev database is a temporary database Atlas uses to load the desired state, replay migrations, and
compute plans (https://atlasgo.io/concepts/dev-database). Redshift runs only on AWS and has no Docker
image, so on Redshift it is a real, empty database you provide, with the target's scope and URL form.
Every Redshift command needs `atlas login`. If `atlas whoami` shows no login, ask the user to run
`atlas login`; a user without an account can start a free trial with it.

## Scope

The scope decides what Atlas covers (https://atlasgo.io/concepts/url#scope):

| URL | Scope | Atlas covers | DDL it writes |
|-----|-------|--------------|---------------|
| `redshift://u:p@host:5439/analytics?search_path=public` | Schema | That schema only | Unqualified (`"users"`) |
| `redshift://u:p@host:5439/analytics` | Database | Every schema in `analytics` | Qualified (`"public"."users"`) |
| `redshift+http://cluster(c)/analytics` | Schema (the default `mode=schema`) | The user's current schema, `SELECT CURRENT_SCHEMA()` | Unqualified |
| `redshift+http://cluster(c)/analytics?mode=database` | Database | Every schema in `analytics` | Qualified |

- `redshift+http://` ignores `search_path`. To bind a Data API connection to another schema, set the
  user's default: `ALTER USER <user> SET search_path = <schema>;`.
- Users, groups, roles, and datashares are inspected and planned only at database scope. So are
  several schemas, and external schemas next to local ones.
- At database scope, the desired state covers every schema in the database. A schema it does not
  declare, external schemas included, is planned for `DROP SCHEMA "name" CASCADE`. Declare
  `schema "public" {}`, or limit the run with `include` and `exclude`.
- `exclude` and `include` follow the scope. At database scope a table pattern starts with the schema:
  `"public.*"`. Principals and datashares take a type: `"app_*[type=role]"`, `"*[type=datashare]"`.
- An HCL file with several `schema` blocks fails against a schema-scoped dev URL with
  `cannot use HCL with more than 1 schema when dev-url is limited to schema "..."`.

## Your Own Dev Database

Redshift has no Docker image, so `docker://redshift/...` fails with
`unsupported docker image "redshift"`, and `atlas tool docker` cannot start one. Give Atlas an empty
database you own instead (https://atlasgo.io/concepts/dev-database#providing-your-own-dev-database):

| Option | Works for | Notes |
|--------|-----------|-------|
| A second database on the target's cluster or workgroup | Tables, views, materialized views, functions, procedures, external schemas, and declarative plans of principals and grants | The cheapest option. Create it once, outside Atlas: `CREATE DATABASE atlas_dev;` |
| A separate Serverless workgroup or cluster | Everything, including migration directories that create users, groups, roles, or datashares | Required for the versioned workflow with principals or datashares; a small Serverless workgroup bills only while it runs |

Principals and datashare names belong to the cluster, not to a database. On a dev database that shares
the target's cluster:

- Replaying a migration directory with `CREATE USER` or `CREATE ROLE` fails on a principal the cluster
  already holds, and a replayed `DROP` would drop a live one.
- With `roles = true`, the dev database is restored after each run by diffing the cluster's principals
  against the snapshot taken at the start, so principals created by anyone else during the run can be
  dropped.
- A datashare replayed from a migration file collides with the one the target already holds.

When loading an HCL or SQL desired state, Atlas never creates principals on the dev database, and it
holds grants to roles and users out of the replay, so a schema that declares them still loads.

Atlas restores the dev database only for what the env inspects. In an env without
`mode "redshift" { datashares = true }`, a datashare that a replayed migration file creates stays on
the dev database after the run, and the next replay fails on its name. Turn the mode on in every env
whose migrations create datashares.

### URLs for the dev database

The dev URL uses the target's URL form and scope. At schema scope, its `search_path` names an empty
schema with the same name as the target's, created once in the dev database: `CREATE SCHEMA app;` for a
target bound to `app`. A dev schema of another name, such as `public`, which carries Redshift's
built-in comment, makes `atlas migrate diff` write `COMMENT ON SCHEMA "app" IS ''` into the first file,
and every later replay fails with `pq: schema "app" does not exist`:

```hcl
env "prod" {
  url = getenv("REDSHIFT_URL")      # redshift://...:5439/analytics?search_path=public
  dev = getenv("REDSHIFT_DEV_URL")  # redshift://...:5439/atlas_dev?search_path=public
  schema {
    src = "file://schema.rs.hcl"
  }
}
```

With the Data API, both URLs take the same `mode`:

```bash
atlas schema apply \
  --url "redshift+http://cluster(atlas-demo)/dev?mode=database" \
  --dev-url "redshift+http://root@cluster(atlas-demo)/devdb?mode=database" \
  --to file://schema.sql
```

### The dev database must be clean

Atlas checks the dev database before using it, and restores it to its original state on exit:

- Schema scope: the connected schema may hold no tables.
- Database scope: the database may hold only an empty `public` schema.

Otherwise Atlas fails with `connected database is not clean: found table "t" in connected schema`, or
`connected database is not clean: found schema "sales"`. Point `dev` at an empty database dedicated to
Atlas. Environments can share one only if they never run at the same time.

## The `dev` Block and `baseline`

When the schema depends on objects it does not manage, such as another team's schema, a view source, or
an external schema, a `dev` block applies a `baseline` script to the dev database before each run
(https://atlasgo.io/concepts/dev-database#baseline-schema):

```hcl
dev "redshift" "shared" {
  url      = getenv("REDSHIFT_DEV_URL")
  baseline = file("baseline.sql")
}

env "prod" {
  url = getenv("REDSHIFT_URL")
  dev = dev.redshift.shared.url
  schema {
    src = "file://schema.rs.hcl"
  }
}
```

- `url` is required; `baseline` is optional and takes SQL, or `file("baseline.sql")`.
- Atlas does not remove what the baseline creates, so it runs again on a database that already holds
  its objects. Write it with `IF NOT EXISTS`, or the second run fails with
  `execute baseline: pq: Schema "other" already exists (42P06)`.
- `template = true` is PostgreSQL only and fails with
  `template attribute is supported only for postgres dev-databases`.
- There is no `docker "redshift"` block.

## Connection URLs

| Scheme | Transport | Use it when |
|--------|-----------|-------------|
| `redshift://` | The PostgreSQL wire protocol to the cluster or workgroup endpoint, port `5439` by default | Atlas can reach the endpoint: a public endpoint, a VPN, or a runner in the VPC |
| `redshift+http://` | The Redshift Data API over HTTPS, authenticated with AWS credentials | Atlas has AWS credentials but no network path to the endpoint |

### `redshift://`

```
redshift://<user>:<pass>@<endpoint>:5439/<database>?search_path=<schema>
```

- The endpoint of a provisioned cluster and of a Serverless workgroup take the same URL shape:
  `atlas-dev.123456789012.us-east-1.redshift-serverless.amazonaws.com:5439` for a workgroup.
- Without a port, Atlas uses `5439`.
- Query parameters reach the PostgreSQL client: `sslmode=disable` turns TLS off
  (https://atlasgo.io/concepts/url).
- Percent-encode special characters in the password, or build the URL with `urlescape` in `atlas.hcl`.

### `redshift+http://` (Data API)

```
redshift+http://workgroup(<workgroup>)/<database>               # Serverless, IAM identity
redshift+http://<secret-arn>@workgroup(<workgroup>)/<database>  # Serverless, Secrets Manager
redshift+http://cluster(<cluster-id>)/<database>                # Provisioned, IAM identity
redshift+http://<db-user>@cluster(<cluster-id>)/<database>      # Provisioned, temporary credentials
redshift+http://<secret-arn>@cluster(<cluster-id>)/<database>   # Provisioned, Secrets Manager
```

- AWS credentials and the region come from the standard AWS configuration: `AWS_ACCESS_KEY_ID`,
  `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`, `AWS_REGION`, or a profile. The identity needs access
  to the Data API and to the credentials it logs in with: temporary credentials, or the secret.
- A user that starts with `arn` is a Secrets Manager ARN. A plain user name is a database user, and is
  used only with `cluster(...)`.
- `timeout` bounds each statement, or each transaction batch (default `5m`), and `polling` sets how
  often Atlas polls for its result (default `50ms`):
  `redshift+http://cluster(atlas-demo)/dev?timeout=10m`. Both take a number with a unit (`90s`,
  `10m`); anything else fails with `invalid timeout: ...` or `invalid polling: ...`.
- The database name is required. A host other than `workgroup(...)` or `cluster(...)` fails with
  `invalid host: ...` or `invalid authorization source: ...`; a missing database fails with
  `database name is required`.
- An AWS-side rejection, such as missing credentials or IAM permissions, is reported as
  `execute statement: ...`. A statement Redshift rejects is reported as
  `query failed: <Redshift error>`.
- Each statement outside a transaction runs in its own session, and a transaction is sent as one batch
  when it commits (`references/migrations.md`, The Data API).

## Errors

| Error | Cause | Fix |
|-------|-------|-----|
| `login is required to use the 'redshift' driver` | No Atlas login | `atlas login` |
| `unsupported docker image "redshift"` | A `docker://redshift/...` dev URL | Use a real, empty Redshift database |
| `"dev-url" not set` | `migrate diff` without a dev database | Set `dev` on the env, or pass `--dev-url` |
| `connected database is not clean: ...` | The dev database holds objects | Use an empty, dedicated database, or move the objects to a `baseline` |
| `cannot diff a schema "..." with a database connection`, or `cannot diff a database connection with a schema "..."` | The URL and the dev URL have different scopes | Give both the same `search_path`, or the same `mode` |
| `modify schema "..." is not allowed when migration plan is scoped to one schema` | A schema-level change through schema-scoped URLs | Use database scope for both URLs |
| `sql/redshift: connection mode must be schema or database, received "..."` | A bad `mode` on a `redshift+http://` URL | `mode=schema` or `mode=database` |
| `invalid host: ...`, or `invalid authorization source: ...` | The Data API host is not `workgroup(...)` or `cluster(...)` | Fix the host |
| `database name is required` | A Data API URL without a database in its path | Add `/<database>` |
| `execute statement: ...` | The Data API rejected the call: credentials, region, or IAM permissions | Fix the AWS identity Atlas runs with |
| `query failed: ...` | Redshift rejected a statement sent through the Data API | Read the Redshift error after the prefix |
| `invalid port ":..." after host` | A special character in the password breaks the URL | Percent-encode the password, or use `urlescape` |
