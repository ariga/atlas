# ClickHouse Security and Secrets as Code

Users, roles, grants, row policies, settings profiles, passwords, and the secret settings that engines,
dictionaries, and named collections carry. The guides:
https://atlasgo.io/guides/clickhouse/security-declarative and
https://atlasgo.io/guides/clickhouse/security-versioned. Everything here needs `atlas login`, as the
ClickHouse driver does. If `atlas whoami` shows no login, ask the user to run `atlas login`; a user
without an account can start a free trial with it.

## Turn It On

Users, roles, and grants are left out of inspection and planning by default. Row policies, settings
profiles, and named collections have flags of their own in the `mode "clickhouse"` block:

```hcl
env "local" {
  url = getenv("DATABASE_URL")          // Server scope: no database in the path
  dev = "docker://clickhouse/24.10"
  schema {
    src = "file://schema.ch.hcl"
    mode "clickhouse" {
      roles             = true   // Users and roles
      permissions       = true   // GRANT and REVOKE
      policies          = true   // Row policies
      settings_profiles = true   // Settings profiles
      named_collections = true   // Named collections
    }
  }
}
```

- These are server-level objects: the URL and the dev URL use server scope (`references/dev-database.md`).
- Grants are read only on the databases Atlas inspects. A grant on an object outside them is not loaded.
- To manage grants alone, without the databases, tables, or roles they name, set `schemas = false` and
  `permissions = true`, and address each target with `ref()`. Atlas never creates or drops such a
  target; it diffs only its grants:

  ```hcl
  permission {
    to         = "analyst"
    for        = ref("table.analytics.events")
    privileges = [SELECT]
  }
  ```

## Which Users and Roles Atlas Manages

With `roles = true`, the desired state is authoritative over every user and role inspection returns. An
account the schema does not declare is planned for a drop. Inspection skips:

- The user Atlas connects as, and the roles granted to it.
- Users and roles in the read-only `users_xml` and `ldap` storages, defined in the server
  configuration.
- On self-hosted servers, users whose name starts with `root` or `default`. On ClickHouse Cloud, users
  whose name contains `default`, `internal`, `sql-console`, `clickhouse`, or `prometheus`, and roles
  whose name contains `sql_console`, `default_role`, or `clickpipes`.

A user that inspection skips is never managed: do not declare it, or every plan tries to create it.
Declare every other account, and review the first plan for drops. To keep drops out of every plan, set
`drop_user = true` and `drop_role = true` in the env's `diff { skip { ... } }` policy
(https://atlasgo.io/versioned/diff).

## Users and Roles

```hcl
variable "api_password" {
  type = string
}

role "app_readonly" {}

role "app_writer" {
  member_of = [role.app_readonly]
}

user "api" {
  password  = var.api_password
  auth_type = "sha256_password"
  member_of = [role.app_writer]
  settings_profile {
    setting "max_memory_usage" {
      value = 10000000000
    }
  }
}
```

- `member_of` grants roles. A `membership` block, with `role = role.app_writer` and `admin = true`,
  grants one `WITH ADMIN OPTION` (https://atlasgo.io/hcl/clickhouse#user).
- `storage` picks the access storage: `local_directory`, `replicated`, `memory`, `users_xml`, or `ldap`.
  The last two are read-only and fail with `storage "users_xml" is read-only and cannot be managed`.
  Without `storage`, the entity lives in the server's default storage, and declaring that default
  explicitly plans no change.
- Changing `storage` moves the entity with `MOVE USER` or `MOVE ROLE`. Moving into `memory` fails with
  `moving user "<name>" to storage "memory" loses it on the next restart of the server`. In cluster mode,
  a move into or out of `replicated` fails with
  `moving user "<name>" from storage "local_directory" to "replicated" should be handled manually`.
- A user or role in the `replicated` storage is created without `ON CLUSTER`, since the storage is
  shared by every node.
- An inline `settings_profile` block sets the user's or role's own settings; a `setting` takes `value`,
  `min`, `max`, and `writability` (`WRITABLE`, `CONST`, or `CHANGEABLE_IN_READONLY`), and the block takes
  `inherit` to name profiles.
- Role and user blocks take no `comment`.

## Passwords

A password is write-only: the server never reads it back.

- `schema apply` writes a password only with `sensitive = ALLOW` in the env's `schema.mode`. Without
  the mode, a new user is created without its declared password, as `CREATE USER <name>`, or
  `NOT IDENTIFIED` when it sets `auth_type`, and anyone can log in as it with no password. A password
  change is invisible to the plan. Turn on the mode for every env that creates users, or set their
  passwords in a hand-written migration (below).
- With the mode, the plan masks the password, as in
  `ALTER USER (sensitive) IDENTIFIED WITH sha256_password BY (sensitive)`. The server cannot return
  it, so every later plan sets it again: a plan for an env with users is never empty.
- A migration file never holds a password, whatever the mode. `migrate diff` creates the user without
  one, or `NOT IDENTIFIED` when the user sets `auth_type`. With `sensitive = ALLOW`, it fails on a
  password change with `changing password for user "<name>" should be handled manually`; without the
  mode, the change is invisible. Set passwords in a hand-written file
  that reads them from a template directory (https://atlasgo.io/atlas-schema/projects) and starts with
  `-- atlas:sensitive`, so they are never logged.
- Removing a password fails with `removing the password of user "<name>" should be handled manually`.

## Permissions

The `permission` block grants on a database (`schema`), a table, a view, a materialized view, a
dictionary, a function, a named collection, or a single column. `to` takes a role or user reference or a
name. `privileges` takes ClickHouse privileges, such as `SELECT`, `INSERT`, `ALTER`, `CREATE`, `DROP`,
`dictGet`, or `ALL` (https://atlasgo.io/hcl/clickhouse#permission):

```hcl
permission {
  for_each   = [table.orders, table.products, table.users]
  for        = each.value
  to         = role.app_readonly
  privileges = [SELECT]
}

permission {
  to         = role.app_writer
  for        = schema.demo
  privileges = [INSERT, SELECT]
  grantable  = true
}
```

In cluster mode, `GRANT` and `REVOKE` carry `ON CLUSTER`, like every other statement.

## Row Policies

Row policies need `policies = true` and server scope:

```hcl
policy "tenant_filter" {
  on     = table.users
  using  = "tenant_id = 1"
  to     = [role.analyst, user.api]
}

policy "db_wide" {
  on     = "app.*"
  using  = "1"
  to_all = true
}
```

- `on` takes a table reference, or a `"database.table"` string that can hold `*`.
- Set one of `to`, `to_all`, or `to_all_except`. `restrictive = true` makes it `AS RESTRICTIVE`.
- Atlas writes every policy `FOR SELECT`, and changes one in place with `ALTER ROW POLICY`.

## Settings Profiles

Settings profiles need `settings_profiles = true`:

```hcl
settings_profile "analyst_limits" {
  setting "max_memory_usage" {
    value       = 10000000000
    writability = CONST
  }
  inherit = ["default"]
  to      = [role.analyst]
}
```

The profile is created before the users and roles that inherit it, and changed in place. Do not name a
profile `default` or `readonly`: the official image defines both in `users.xml`, and creating one fails
with ``Cannot insert settings profile `readonly` to users_xml because this storage is readonly``.

## Secrets

ClickHouse never returns a password, and returns other secrets as `[HIDDEN]`. Atlas treats as secrets:
user passwords, named collection values, the arguments of engines that take credentials (`MySQL`,
`PostgreSQL`, `MongoDB`, `Redis`, `S3`, `S3Queue`, `URL`, `AzureBlobStorage`, and the
`MaterializedMySQL` and `MaterializedPostgreSQL` database engines), a dictionary `source` whose text
mentions `password`, `token`, `secret`, `credential`, or `auth`, and secret settings such as `kafka_sasl_password`,
`rabbitmq_password`, `nats_password`, `aws_secret_access_key`, `catalog_credential`, `auth_header`, and
a `nats_url` or `rabbitmq_address` that holds a password. Pass every secret through an input variable,
never a literal. Secret table settings depend on the server version: ClickHouse 24.10 rejects
`kafka_sasl_password` with `Unknown setting 'kafka_sasl_password': for storage Kafka`, and 26.9 accepts
it.

Without `sensitive = ALLOW` (the default):

- Secret settings are never written. A table created from a schema with
  `kafka_sasl_password = var.kafka_password` is created without it. The dev database receives a
  placeholder in their place, never the value.
- Masked values are compared as equal to the desired ones, so a plan stays quiet about them.
- A change that must drop and recreate an existing object that holds a secret setting fails instead of
  dropping the secret:
  `table "events" must be recreated for this change, which drops its secrets. Enable the sensitive mode to plan it:`.
  Kafka, RabbitMQ, NATS, S3Queue, and FileLog tables are recreated for any change, a comment included.
- Named collections are created and changed with their keys, and empty values. Value changes are not
  planned (https://atlasgo.io/guides/clickhouse/named-collections).

In migration files, whatever the mode:

- Secret settings are left out. A change that must recreate a table holding one fails with
  `..., as a deferred plan does not hold them. Recreate it manually to apply this change`.
- Passwords are left out (Passwords, above).
- Credentials inside engine arguments and dictionary sources are written into the file, and the
  statement is marked `-- atlas:sensitive`. Keep such files out of shared logs, or move the credential
  into a named collection.

With `sensitive = ALLOW`:

```hcl
env "local" {
  schema {
    src = "file://schema.ch.hcl"
    mode {
      sensitive = ALLOW
    }
  }
}
```

- `schema apply` writes secret settings, passwords, and named collection values. Statements that carry
  them print masked, such as `kafka_sasl_password = (sensitive)` or
  `ENGINE = MySQL((sensitive), (sensitive), (sensitive), (sensitive), (sensitive))`.
- Masked values are compared as they are read. A value the server returns as `[HIDDEN]` never matches
  the desired one, so the dictionary, table, or named collection that holds it is planned as changed on
  every run, and a table whose engine cannot be altered is recreated. For named collections, enable
  `show_named_collections_secrets` for the user Atlas connects as, so Atlas reads the real values.
- A named collection key whose desired value the server masked, such as one from a SQL schema replayed
  on the dev database, cannot be created: `named collection "<name>": value of item "<key>" cannot be read from the source`.
