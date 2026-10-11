# Spanner Dev Database and Connections

The dev database is a temporary database Atlas uses to load the desired state, replay migrations, and
compute plans (https://atlasgo.io/concepts/dev-database). For Spanner it is usually the Spanner emulator
in Docker, which differs from production in ways this file lists.

## Scope

A Spanner URL names a database, and Atlas always covers all of it: the default schema and every named
schema (`CREATE SCHEMA sales`). There is no schema-scoped URL.

- Objects in the default schema are written unqualified (`` `Users` ``), and objects in a named schema
  qualified (`` `sales`.`Orders` ``).
- A named schema that the desired state does not declare is planned for a drop, after its objects.
  Limit a run with `--schema default`, `schemas = ["default"]` in the env, or `exclude`.
- `migrate apply` keeps its revisions table in a named schema, `atlas_schema_revisions`, which a
  declarative plan on the same database drops unless excluded (`references/migrations.md`).
- The dev URL covers a whole database too, so dev and target scopes always match.

## Docker Dev URLs

```bash
# GoogleSQL databases.
--dev-url "docker://spanner/1.5.58"

# PostgreSQL-dialect databases.
--dev-url "docker://spannerpg/1.5.58"
```

- Both run `gcr.io/cloud-spanner-emulator/emulator:<tag>`, and `spannerpg` creates a
  PostgreSQL-dialect database in it. Match the target's dialect: Atlas reads the target's dialect on its
  own and plans in it.
- Pin a release. `latest` is whatever image the machine pulled last, and an old one rejects newer DDL:
  `SQL SECURITY DEFINER` views need 1.5.56 or later, and fail on older images with
  `Expecting 'INVOKER' but found 'DEFINER'`. Releases:
  https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/releases.
- The emulator starts in a few seconds and keeps everything in memory.
- `docker://` needs a Docker client and a reachable daemon. Without one, run the emulator yourself and
  point `dev` at an empty database in it (Connection URLs, below).
- There is no `docker "spanner"` block: a config with one fails with
  `missing docker block handler for "spanner"`. The dev database starts empty, without an `init` or a
  `baseline`.

### Named containers with `atlas tool docker`

`atlas tool docker` starts a named emulator and prints its URL. Use it for a local database to apply
migrations to (https://atlasgo.io/cli-reference#atlas-tool-docker):

```bash
export DATABASE_URL=$(atlas tool docker --url "docker://spanner/1.5.58" --name my-spanner)
# spanner://host.docker.internal:32929/projects/emulator-project/instances/test-instance/databases/test-database?useplaintext=true
atlas tool docker kill --name my-spanner
```

## What the Emulator Does Not Do

The emulator accepts most DDL, but it is not production
(https://docs.cloud.google.com/spanner/docs/emulator,
https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/master/README.md):

| Area | On the emulator | Effect on Atlas |
|------|-----------------|-----------------|
| Roles and grants | Parsed and ignored; no privilege views; column grants rejected | `mode { roles = true }` or `permissions = true` fails: plan them on a dev database you provide (`references/security.md`) |
| Placements | Accepted as no-ops | A SQL schema loses them; declare them in HCL (`references/hcl.md`) |
| Locality groups | Some options not reported | A SQL schema loses `storage`; plans against an emulator target never converge |
| Vector index options | Not reported | A SQL schema loses them; later plans fail (`references/objects.md`, Indexes) |
| Mutation limits and quotas | Not enforced | Large data migrations pass tests and fail in production |
| Some invalid SQL | Accepted | A plan can pass on the emulator and fail on the target |
| Long-running changes | Finish at once | Index builds and validations take minutes to hours in production (`references/migrations.md`) |
| Concurrency | One read-write transaction or schema change at a time | Runs on one emulator wait for each other |

## Your Own Dev Database

For roles and grants, or to plan against exactly what production accepts, point `dev` at an empty
database on a real Spanner instance, with the target's dialect and edition:

```bash
gcloud spanner databases create atlas-dev --instance=<instance>
gcloud spanner databases create atlas-dev-pg --instance=<instance> --database-dialect=POSTGRESQL
```

```hcl
env "prod" {
  url = "spanner://projects/<project>/instances/<instance>/databases/orders"
  dev = "spanner://projects/<project>/instances/<instance>/databases/atlas-dev"
}
```

- Atlas loads the desired state into it and drops everything afterwards, leaving it empty. On a real
  instance every one of those statements is a schema change that takes seconds, and Spanner throttles a
  database that receives many of them
  (https://docs.cloud.google.com/spanner/docs/schema-updates-best-practices). Use it only for the envs
  that need it, and keep `docker://` for the rest.
- Give each env and CI job its own dev database: two runs that share one at the same time see each
  other's objects.
- Never point `dev` at a real environment's database. Atlas refuses a database that is not empty:
  `connected database is not clean: found table "Users" in default schema "default"`.

## Connection URLs

- Production: `spanner://projects/<project>/instances/<instance>/databases/<database>`. The URL carries
  no credentials: Atlas uses Application Default Credentials, from
  `gcloud auth application-default login` on a workstation, or the service account of the CI runner
  (https://docs.cloud.google.com/spanner/docs/authentication). Without them, Atlas fails with
  `credentials: could not find default credentials`.
- The principal needs to read and update the schema and to write the revisions table:
  `roles/spanner.databaseUser` on the database covers it (https://docs.cloud.google.com/spanner/docs/iam).
- An emulator you run: `spanner://localhost:9010/projects/<project>/instances/<instance>/databases/<database>?useplaintext=true`,
  or the production URL with `SPANNER_EMULATOR_HOST=localhost:9010` set. The emulator starts without
  instances or databases; create them through its REST port:

  ```bash
  docker run -d --name spanner -p 9010:9010 -p 9020:9020 gcr.io/cloud-spanner-emulator/emulator:1.5.58
  curl -X POST localhost:9020/v1/projects/my-project/instances -H 'Content-Type: application/json' \
    -d '{"instanceId":"dev","instance":{"config":"emulator-config","displayName":"dev","nodeCount":1}}'
  curl -X POST localhost:9020/v1/projects/my-project/instances/dev/databases -H 'Content-Type: application/json' \
    -d '{"createStatement":"CREATE DATABASE `app`"}'
  # PostgreSQL dialect:
  curl -X POST localhost:9020/v1/projects/my-project/instances/dev/databases -H 'Content-Type: application/json' \
    -d '{"createStatement":"CREATE DATABASE \"app-pg\"","databaseDialect":"POSTGRESQL"}'
  ```

  Instance IDs start with a lowercase letter and hold 2 to 64 lowercase letters, digits, and hyphens.

## Login and the Extended Build

The Spanner driver ships in the extended build of Atlas, and needs a login:

- `Error: The spanner driver is available only as custom build`: install the extended build with
  `curl -sSf https://atlasgo.sh | ATLAS_FLAVOR="spanner" sh`, or use the `arigaio/atlas:latest-extended`
  image.
- `Abort: The spanner driver is available only to Atlas Pro users`: ask the user to run `atlas login`; a
  user without an account can start a free trial with it.

## Errors

| Error | Cause | Fix |
|-------|-------|-----|
| `The spanner driver is available only as custom build` | The installed Atlas has no Spanner driver | Install the extended build |
| `The spanner driver is available only to Atlas Pro users` | No login | Ask the user to run `atlas login` |
| `credentials: could not find default credentials` | No Application Default Credentials | `gcloud auth application-default login`, or a service account |
| `connected database is not clean` | `dev` points at a database that is not empty | Use a Docker dev URL or an empty database |
| `Instance not found` or `Database not found` | The instance or database does not exist; an emulator starts with neither | Create them, on an emulator through its REST port |
| `Table not found: INFORMATION_SCHEMA.TABLE_PRIVILEGES` | Roles or permissions planned on the emulator | Use a dev database you provide |
| `Expecting 'INVOKER' but found 'DEFINER'` | The emulator image is older than 1.5.56 | Pin a newer emulator release |
| `missing docker block handler for "spanner"` | A `docker "spanner"` block | Use a `docker://spanner/<version>` URL |
