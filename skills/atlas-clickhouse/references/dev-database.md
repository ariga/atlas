# ClickHouse Dev Database and Connections

The dev database is a temporary database Atlas uses to load the desired state, replay migrations, and
compute plans (https://atlasgo.io/concepts/dev-database). For ClickHouse it must match the target's
scope, major version, cluster topology, and the settings the schema depends on. The ClickHouse driver
needs `atlas login` for every connection, the dev database included. If `atlas whoami` shows no login,
ask the user to run `atlas login`; a user without an account can start a free trial with it.

## Scope

The URL's path names a database, and decides what Atlas covers (https://atlasgo.io/concepts/url):

| URL | Scope | Atlas covers | DDL it writes |
|-----|-------|--------------|---------------|
| `clickhouse://user:pass@localhost:9000/app` | Database | The tables, views, materialized views, and dictionaries of `app`, and their grants | Unqualified (`users`) |
| `clickhouse://user:pass@localhost:9000` | Server | Every database except `system`, `information_schema`, `INFORMATION_SCHEMA`, and `_local`, plus server-level objects | Qualified (`app.users`) |

- Users, roles, row policies, settings profiles, named collections, UDFs, and WASM modules are
  server-level objects. A database-scoped URL never inspects them, so a schema that declares them uses
  server scope for the URL and the dev URL.
- At server scope, the desired state covers every database: one it does not declare, `default`
  included, is planned for `DROP DATABASE` (lint reports it as `DS101`). Declare `schema "default" {}`,
  limit the run with `--schema`, or set `drop_schema = true` in the env's `diff { skip { ... } }`
  policy.
- At server scope, the session's database is `default`. Atlas qualifies the names it writes, but not
  the queries you write: name each table with its database in view, materialized view, and dictionary
  queries (`SELECT ... FROM app.events`), or the dev database fails to create them.
- The dev URL takes the target's scope: `docker://clickhouse/24.10/dev` for a database-scoped target,
  and `docker://clickhouse/24.10` for a server-scoped one. Mixing them leads to wrong plans.
- A database-scoped URL whose database does not exist fails on connect with
  `code: 81, message: Database <name> does not exist`.
- At database scope, a migration file cannot change the database's own `settings`: `migrate diff`
  fails with `modify schema "<name>" is not allowed when migration plan is scoped to one schema`. Use
  server scope for both URLs, or change it with `schema apply`.
- Atlas does not plan changes to a database's `engine` or `comment`: create the database again by hand
  to change them.

## Docker Dev URLs

The `docker://` driver starts a fresh `clickhouse/clickhouse-server:<tag>` container for each run. The
URL path holds the image tag, then an optional database that Atlas creates in the container
(https://atlasgo.io/concepts/dev-database):

```bash
# When working on a single database.
--dev-url "docker://clickhouse/24.10/dev"

# When working on several databases or server-level objects.
--dev-url "docker://clickhouse/24.10"
```

Custom images:

```bash
# When working on a single database.
docker+clickhouse://org/image/dev
docker+clickhouse://user/image:tag/dev
# For local/official images, leave host empty or use "_".
docker+clickhouse:///local/dev
docker+clickhouse://_/clickhouse:latest/dev

# When working on several databases.
docker+clickhouse://local
docker+clickhouse://org/image
docker+clickhouse://user/image:tag
```

- The container connects as `default`, with access management enabled, so the dev database can create
  users, roles, and grants.
- `docker://` needs a Docker client and a reachable daemon. Without one, run a ClickHouse service
  container and pass its URL as the dev URL.

### Named containers with `atlas tool docker`

`atlas tool docker` creates a named container from a `docker://` URL and prints its connection URL. It
also cleans the database if it is not empty, so never run it against a container that holds data to
keep (https://atlasgo.io/cli-reference). The container runs until `atlas tool docker kill --name my-db`:

```bash
# Spin up a local ClickHouse database
export DATABASE_URL=$(atlas tool docker --url "docker://clickhouse/24.10/myapp" --name my-db)
```

## Versions

Commands that connect to the target and the dev database compare their major versions, and fail when
they differ:

```
failed to check compatibility with dev database: target connection is running version 24.x but dev database is running version 23.x
```

Read the target's version with `SELECT version()`, and pin the tag: `docker://clickhouse/24.10/dev`, or
`image` in a `docker` block. Some features depend on the server version:

| Feature | Version |
|---------|---------|
| `EXCHANGE TABLES` in table rebuilds | 21.4 |
| Named collections | 23.10 |
| Dictionary dependencies on tables read from the server | 24.1 |
| Projections inspected | 24.10 |
| `overridable` on named collection items | 25.5 |
| `TimeSeries` columns kept in the engine's inner tables | 26.0 |
| WASM modules and WASM UDFs | 26.3 |

Below 24.1, Atlas guesses a dictionary's dependencies from a `CLICKHOUSE` source's text only, so it can
order the dictionary before a table it reads.

## The `docker` Block

A `docker "clickhouse"` block configures the dev container: `image` (required), `schema`, `baseline`,
`init`, `volumes`, `env`, `command`, `args`, `platform`, `build`, and more
(https://atlasgo.io/hcl/config#docker.clickhouse). The `docker` and `dev` blocks need `atlas login`.

- `schema` makes the dev URL database-scoped, on a database of that name. Leave it out for server scope.
- `volumes` mounts server configuration into the container, for settings that SQL cannot change, such
  as WASM support, `named_collection_control`, or `show_named_collections_secrets`:

  ```hcl
  docker "clickhouse" "dev" {
    image   = "clickhouse/clickhouse-server:24.10"
    volumes = ["./users.xml:/etc/clickhouse-server/users.d/users.xml:ro"]
  }

  env "local" {
    url = getenv("DATABASE_URL")
    dev = docker.clickhouse.dev.url
    schema {
      src = "file://schema.ch.hcl"
    }
  }
  ```

- `baseline` runs SQL on the dev database before Atlas uses it, such as a table in another database that
  a view or dictionary reads.

## Your Own Dev Database

A `dev "clickhouse"` block, or a plain `clickhouse://` dev URL, connects to an existing database instead
of a container (https://atlasgo.io/hcl/config). The database must be empty and dedicated to Atlas.
Atlas refuses one that holds objects, with an error that starts with `connected database is not clean:`
and names the first object it found:

```
found table "users" in database "dev"
found view "v" in database "dev"
found function "plus_one" in database "default"
found database "app"
found named collection "s3"
found row policy "p"
found settings profile "analyst_profile"
```

An empty `default` database is allowed at server scope, and `atlas_schema_revisions` does not count.

```hcl
dev "clickhouse" "dev" {
  url = "clickhouse://user:pass@dev-host:9000/app_dev?mode=cluster"
}

env "prod" {
  url = getenv("DATABASE_URL")
  dev = dev.clickhouse.dev.url
}
```

## Clusters

When the target defines a `cluster` macro, the dev database must be on a cluster too, or commands fail
with `dev database must also be in a cluster configuration`. `docker://` starts a single node, so it
cannot serve a clustered target (https://atlasgo.io/guides/clickhouse/cluster-mode):

- At database scope, use an empty database dedicated to Atlas on a cluster with the target's
  topology. It can sit on the target's own cluster, next to the target database, such as `app_dev` next
  to `app`. Set `mode=cluster` on both URLs.
- At server scope, the dev server must hold nothing, so it needs a separate cluster. ClickHouse
  publishes Docker Compose recipes for local clusters.

```hcl
env "prod" {
  url = "clickhouse://user:pass@ch-host:9000/app?mode=cluster"
  dev = "clickhouse://user:pass@ch-host:9000/app_dev?mode=cluster"
}
```

## ClickHouse Cloud

- Connect on port `9440` with `secure=true`.
- Cloud replaces MergeTree and `Replicated*` engines with their `Shared` equivalents (`MergeTree`
  becomes `SharedMergeTree`), and Atlas treats each pair as equal, so a schema written with OSS engines
  plans no engine change against Cloud. Write OSS engines (`MergeTree`, `ReplacingMergeTree`,
  `Atomic`): a Docker dev database rejects `SharedMergeTree` and `Shared`. A schema inspected from Cloud
  carries the `Shared` names; convert them before using it with a Docker dev database.
- On Cloud, `CREATE DATABASE` is written without `ENGINE` and `SETTINGS`.
- When the target is Cloud, Atlas skips every dev database check: the major version, the cluster, and
  `has_user_defined_zookeeper_path`. Pin the dev image to the Cloud service's version yourself.
- The dev database can be a separate Cloud service, which matches the target fully, or a Docker
  container on OSS engines (https://atlasgo.io/guides/clickhouse/clickhouse-cloud-atlas).
- Inspection skips Cloud's own users and roles: users whose name contains `default`, `internal`,
  `sql-console`, `clickhouse`, or `prometheus`, and roles whose name contains `sql_console`,
  `default_role`, or `clickpipes`.
- Named collections are never inspected on Cloud, even with `named_collections = true`: a declared
  collection is planned for creation on every run (`references/objects.md`, Named Collections).
- WASM modules and WASM UDFs are not supported on Cloud.

## Connection URLs

```
clickhouse://localhost:9000
clickhouse://user:pass@localhost:9000/app
clickhouse://user:pass@localhost:9000/app?secure=true
clickhouse://user:pass@CLICKHOUSE-CLOUD-HOST:9440/app?secure=true
```

Atlas connects over the native protocol: port `9000`, or `9440` with TLS (`secure=true`), the port
ClickHouse Cloud uses (https://atlasgo.io/guides/clickhouse/clickhouse-cloud-atlas). Two query
parameters belong to Atlas and are removed before connecting; every other parameter is passed to the
server, and ClickHouse settings passed this way apply to every query of the session
(https://atlasgo.io/guides/clickhouse/session-settings):

| Parameter | Effect |
|-----------|--------|
| `mode=cluster` | Cluster mode: ``ON CLUSTER `{cluster}``` on every DDL statement (`references/migrations.md`, Cluster Mode). Any other value fails with `clickhouse: unknown mode "<value>"` |
| `has_user_defined_zookeeper_path=true` | The server sets `user_defined_zookeeper_path`, which replicates UDFs on its own and rejects `ON CLUSTER` on their DDL, so Atlas leaves it out. Set it on both URLs, or commands fail with `target and dev databases disagree on has_user_defined_zookeeper_path, set it on both URLs` |
| Any ClickHouse setting | Applied to the session, such as `allow_suspicious_low_cardinality_types=1` or `max_execution_time=60` |

- Set a setting the schema depends on, such as `allow_suspicious_low_cardinality_types`, on the dev URL
  too: the dev database replays the desired state, and fails on the same statement otherwise.
- On connect, Atlas reads `system.settings` and `system.macros`. A user who cannot read them fails with
  `failed to query server properties (version, cloud status, cluster): ...`.
- Percent-encode special characters in the password, or build the URL with `urlescape` in `atlas.hcl`.

## Errors

| Error | Cause | Fix |
|-------|-------|-----|
| `login is required to use the 'clickhouse' driver` | Not logged in | Run `atlas login` |
| `target connection is running version 24.x but dev database is running version 23.x` | Major versions differ | Pin the dev image to the target's version |
| `dev database must also be in a cluster configuration` | Clustered target, single-node dev database | Use a dev database on a cluster |
| `target and dev databases disagree on has_user_defined_zookeeper_path, set it on both URLs` | The parameter is on one URL only | Set it on both |
| `clickhouse: unknown mode "<value>"` | `mode` set to something other than `cluster` | Use `mode=cluster`, or remove it |
| `clickhouse: invalid has_user_defined_zookeeper_path value "<value>"` | The parameter is not a boolean | Use `has_user_defined_zookeeper_path=true` |
| `connected database is not clean: ...` | `dev` holds objects | Use a Docker dev URL, or an empty database |
| `dev database is not clean (found ...). Add a schema to the URL to limit the scope of the connection` | A server-scoped dev URL with a database-scoped desired state | Give the dev URL the target's scope, `docker://clickhouse/24.10/dev` |
| `code: 81, message: Database <name> does not exist` | The URL names a database that does not exist | Create it, or use server scope |
| `failed to query server properties (version, cloud status, cluster): ...` | The user cannot read `system.settings` or `system.macros` | Grant the user `SELECT` on them |
| `modify schema "<name>" is not allowed when migration plan is scoped to one schema` | A database settings change planned into a file at database scope | Use server scope for both URLs |
| `exec: "docker": executable file not found in $PATH` | No Docker client where Atlas runs | Use a separate dev database |
