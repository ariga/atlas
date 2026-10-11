# Snowflake Security as Code

Account roles, database roles, role grants to roles and users, and object privileges. The guide:
https://atlasgo.io/guides/snowflake/security-declarative. Block reference:
https://atlasgo.io/hcl/snowflake#role, https://atlasgo.io/hcl/snowflake#database_role,
https://atlasgo.io/hcl/snowflake#permission.

## Turn It On

Roles and permissions are left out of inspection and planning by default. Enable them on an env whose
URL and dev URL are database-scoped: a schema-scoped connection never reads roles or grants
(`references/dev-database.md`). The whole driver needs `atlas login`.

```hcl
env "platform" {
  url = "${local.base}/ANALYTICS?${local.args}"
  dev = "${local.base}/ANALYTICS_DEV?${local.args}"
  schema {
    src = ["file://schema.layer.hcl", "file://schema.platform.hcl"]
    mode {
      roles       = true
      permissions = true
    }
  }
  # Roles live on the account, not in a schema: scope them by name.
  include = [
    "PUBLIC",
    "BRONZE",
    "an_*[type=role]",
    "kim[type=user]",
  ]
}
```

- With `roles = true`, the desired state is authoritative over every account role and database role
  the env inspects. A role the env sees but does not declare is planned for `DROP ROLE` or
  `DROP DATABASE ROLE`. Scope the env with `include`, which matches account roles and database roles
  alike with `[type=role]`, or keep drops out of plans with the `skip` diff policy:

  ```hcl
  diff {
    skip {
      drop_role = true
    }
  }
  ```

- Atlas never inspects or changes Snowflake's system roles (`ORGADMIN`, `GLOBALORGADMIN`,
  `ACCOUNTADMIN`, `SECURITYADMIN`, `USERADMIN`, `SYSADMIN`, `PUBLIC`), the role the session runs as, or
  grants to any of them.
- Database roles are read from the database in the URL; a URL without a database has none. A
  `database_role` block's optional `database` must name that database, or conversion fails with
  `database role "x": database "OTHER" does not match the connected database "ANALYTICS"`.
- The schema file that holds the grants needs the objects they name, so list the schema file in `src`
  too, as above.

## Keep Roles out of Migration Directories

Manage roles and grants with `atlas schema apply` in an env of their own. In a migration directory, the
first `migrate apply` works, and the next `migrate diff` fails: replaying the directory on the dev
database runs the `CREATE ROLE` against the shared account, where the role already exists
(https://atlasgo.io/guides/dbt/snowflake):

```
Error: sql/migrate: read migration directory state: sql/migrate: executing statement "CREATE ROLE \"BI_READER\";" from version "20261008043300": 002002 (42710): SQL compilation error:
Object 'BI_READER' already exists.
```

The same guide runs this env with `atlas schema apply --env platform --skip-lint`, because the plan check
would run the plan's role statements on the account through the dev connection. Read the printed plan
before approving it.

## Roles and Users

```hcl
database_role "an_dbreader" {}
database_role "an_dbwriter" {
  comment   = "db writes"
  member_of = [database_role.an_dbreader]
}

role "an_reader" {}
role "an_analyst" {
  comment   = "analytics"
  member_of = [role.an_reader, database_role.an_dbreader]
}

user "kim" {
  member_of = [role.an_analyst]
}
```

- `role` is an account role, `CREATE ROLE`. Its `member_of` takes account roles and database roles; a
  database role in it is the grant that makes the database role's privileges reachable from the account
  role.
- `database_role` is `CREATE DATABASE ROLE` in the connected database. Its `member_of` takes database
  roles only, or fails with `member_of must reference a database role, but got (...)`.
- Atlas never creates, alters, or drops users. A `user` block manages the user's role grants:
  `GRANT ROLE` and `GRANT DATABASE ROLE ... TO USER`. The user must exist, or the grant fails when
  applied. Users the desired state does not declare are left alone, and so are their grants.
- Names are exact: `role "an_reader"` creates `"an_reader"`, which unquoted SQL cannot address
  (`references/hcl.md`).
- New roles are created first, then the memberships between them, so `member_of` order does not
  matter.

## Permissions

A `permission` block grants privileges to a role or a database role on the connected database
(`for = DATABASE`), a schema, a table of any kind, or a view:

```hcl
permission {
  to         = role.an_reader
  for        = DATABASE
  privileges = [USAGE]
}

permission {
  to         = role.an_reader
  for        = schema.BRONZE
  privileges = [USAGE, CREATE_TABLE, "CREATE VIEW"]
}

permission {
  to         = role.an_analyst
  for        = view.events_view
  privileges = [SELECT]
  grantable  = true
}
```

- `privileges` takes the enums `USAGE`, `MONITOR`, `MODIFY`, `CREATE_SCHEMA`, `CREATE_TABLE`, `SELECT`,
  `INSERT`, `UPDATE`, `DELETE`, and `REFERENCES`, and any other privilege as a string, such as
  `"CREATE VIEW"`. `ALL` is refused: the enum fails with
  `Unknown permission.privileges; There is no privileges named "ALL".`, and the string `"ALL"` with
  `permission.privileges does not support ALL, list the privileges to grant instead`.
- `grantable = true` adds `WITH GRANT OPTION`. Snowflake refuses it on objects in a schema with
  `managed_access = true`, and the apply fails with
  `003510 (23001): SQL execution error: Specification of grant option is not supported in a managed access schema`.
- `for` takes `DATABASE`, `schema.X`, `table.X`, `view.X`, `dynamic_table.X`, `hybrid_table.X`,
  `external_table.X`, and `iceberg_table.X`. Grants on typed tables are planned `ON TABLE`.
- A grant on the database that the file does not declare is planned as a `REVOKE`. Database roles
  receive `USAGE` on their database when created, so Atlas neither inspects nor plans that one.
- Ownership is not modeled: `OWNERSHIP` grants are not inspected, and objects belong to the role that
  creates them, the role in the URL.
- Atlas grants on the objects that exist. Future grants (`GRANT ... ON FUTURE TABLES`) are not inspected
  and not planned. For objects another tool creates later, such as dbt models, grant on the schema
  (`CREATE TABLE`, `CREATE VIEW`) and let that tool grant on its own objects.
- Grants are not modeled on functions, procedures, sequences, stages, tasks, warehouses, or columns.
  Run those once outside Atlas, for example `GRANT USAGE ON WAREHOUSE ETL_WH TO ROLE DBT_RUNNER`.
- A view change is `CREATE OR REPLACE VIEW`, which drops the view's grants. With `permissions = true`,
  Atlas grants the declared ones again in the same plan. Without it, grants made outside Atlas are lost
  (`references/migrations.md`).

## Grants on Objects Declared Elsewhere

`ref()` addresses an object or a role without a block in the HCL file, and only inside a
`composite_schema`, where another part creates the object first. In a plain HCL file, the reference
reaches the dev database as an object to create and fails while loading the desired state:

```hcl
data "hcl_schema" "grants" {
  path = "grants.hcl"
}

data "composite_schema" "app" {
  schema {
    url = "file://tables.sql"
  }
  schema {
    url = data.hcl_schema.grants.url
  }
}
```

```hcl
# grants.hcl
role "reader" {}

permission {
  to         = role.reader
  for        = ref("table.COMP.events")
  privileges = [SELECT]
}

permission {
  to         = role.reader
  for        = ref("dynamic_table.ANALYTICS.hourly")
  privileges = [SELECT]
}
```

- `for` takes `ref("schema.<name>")`, `ref("table.<schema>.<name>")`, `ref("view.<schema>.<name>")`,
  and, for typed tables, `ref("dynamic_table.<schema>.<name>")`, `hybrid_table`, `external_table`, and
  `iceberg_table`. A typed table reference without the schema fails with
  `permission.for reference to undeclared dynamic_table "hourly" must be qualified as ref("dynamic_table.<schema>.<name>")`.
- A reference of the wrong kind fails with
  `permission.for references dynamic_table "PUBLIC.events", which is declared as a table`.
- `to = ref("role.app")` names a role managed elsewhere: Atlas never creates, alters, or drops it, and
  only its grants are diffed.

## Other Teams' Roles and Grants

`include` keeps the env to the roles it names. To keep a principal's grants out of both sides of every
diff, so that its privileges on managed objects are never planned, exclude them with the `.permission`
suffix:

```hcl
env "platform" {
  exclude = [
    "DBT_RUNNER[type=role].permission",
    "BI_*[type=role].permission",
  ]
}
```

The role itself stays; only its grants are dropped from the comparison.

## Errors

| Error | Fix |
|-------|-----|
| `Object '...' already exists` while reading the migration directory | Move roles out of the migration directory (Keep Roles out of Migration Directories) |
| `permission.privileges does not support ALL, list the privileges to grant instead`, or `There is no privileges named "ALL"` | List the privileges |
| `Specification of grant option is not supported in a managed access schema` | Remove `grantable`, or grant on an object outside the managed access schema |
| `member_of must reference a database role, but got (...)` | A `database_role` can be a member of database roles only |
| `user member_of must reference an account or database role, but got (...)` | Reference `role.X` or `database_role.X` |
| `database role "x": database "..." does not match the connected database "..."` | Remove `database`, or connect to that database |
| `permission.to must reference a role` | Use `role.X`, `database_role.X`, or `ref("role.X")` |
| Roles or grants never show up | The URL is schema-scoped, or the env's `mode` block lacks `roles = true` and `permissions = true` |
