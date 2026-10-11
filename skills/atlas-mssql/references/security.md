# SQL Server Security as Code

Database roles, database users, GRANT and REVOKE, and what Atlas leaves to hand-written migrations on
SQL Server. The guides: https://atlasgo.io/guides/mssql/security-declarative and
https://atlasgo.io/guides/mssql/security-versioned.

## Turn It On

Roles, users, and permissions are left out of inspection and planning by default. Enable them in the
env's `schema` block (https://atlasgo.io/atlas-schema/hcl#enabling-roles-and-permissions):

```hcl
env "dev" {
  url = getenv("DATABASE_URL")   # sqlserver://...?database=app&mode=database
  dev = "docker://sqlserver/2022-latest/dev?mode=database"
  schema {
    src = "file://schema.sql"
    mode {
      roles       = true      # Database roles and users. Defaults to false.
      permissions = true      # GRANTs. Defaults to false.
    }
  }
}
```

- Roles and users belong to the database, not to a schema, so an env that manages them uses
  `mode=database` on both the URL and the dev URL. With `mode=schema`, Atlas never inspects or plans
  roles and users: it sees the grants on that schema's objects, with grantees as plain names.
- With `roles = true`, the desired state is authoritative over every role and user Atlas inspects: an
  undeclared one is planned for `DROP USER` or `DROP ROLE`. Review the first plan. The `skip` diff
  policy takes `drop_role = true` and `drop_user = true` to keep those drops out of plans
  (https://atlasgo.io/versioned/diff#diff-policy).
- SQL Server roles and users have no `external` attribute (it fails with
  `unknown attribute external in block of type role`), and no `password`. The `sensitive` mode has no
  effect on SQL Server.

## What Atlas Manages

| Principal | Managed | Notes |
|-----------|---------|-------|
| Database role | Yes, `role` | `CREATE ROLE`, `ALTER ROLE ... ADD MEMBER` |
| Database user without a login | Yes, `user` | `CREATE USER [u] WITHOUT LOGIN WITH DEFAULT_SCHEMA = [s]` |
| Database user for a login | Yes, `user` with `login_name` | `CREATE USER [u] FOR LOGIN [l]`. The login must already exist, on the dev server too |
| Server login, server role | No | Server-level: create them outside Atlas |
| Contained user with a password, Windows or Entra ID user, application role | No | Not inspected; manage them in hand-written migrations |

Never inspected or planned: `dbo`, `guest`, `sys`, `INFORMATION_SCHEMA`, `public`, `##MS*` principals,
the user Atlas connects as, and every role that user belongs to (`db_owner` when connecting as `sa`).

### Users for logins

A `user` with `login_name` maps to a server login, which Atlas never creates. The dev database needs the
login too, or the plan fails with
`mssql: 'svc_login' is not a valid login or you do not have permission.` Create it in the dev
`baseline`, guarded, because a login belongs to the server and stays after the run on a dev database
you provide (`execute baseline: mssql: The server principal 'svc_login' already exists.`):

```hcl
dev "sqlserver" "dev" {
  url      = getenv("DEV_URL")      # ...?database=dev&mode=database
  baseline = <<-SQL
    IF SUSER_ID('svc_login') IS NULL CREATE LOGIN svc_login WITH PASSWORD = 'Dev_Only_Pass1';
  SQL
}
```

The password above exists only on the dev server. Create the real login on each target outside Atlas.

## Fixed Database Roles

`db_datareader`, `db_datawriter`, `db_ddladmin`, `db_owner`, and the other fixed roles always exist, and
Atlas never creates, alters, or drops them:

- Leave them out of an HCL desired state. Declaring `role "db_datareader" {}` fails on the dev database
  with `User, group, or role 'db_datareader' already exists in the current database.`
- At `mode=database`, `atlas schema inspect` prints them as `role "db_..." {}` blocks, so inspected HCL
  fails the same way when used as a desired state. Delete those blocks, or inspect with
  `--exclude 'db_*[type=role]'`.
- An HCL `member_of` takes only references to declared roles, so membership in a fixed role cannot be
  written in HCL. Write it in a SQL schema file, which Atlas plans as `ALTER ROLE ... ADD MEMBER`:

  ```sql
  CREATE USER app_reader WITHOUT LOGIN;
  ALTER ROLE db_datareader ADD MEMBER app_reader;
  ```

  `db_datareader` reads every table and view in the database, which avoids a grant per object.

## Roles and Users in HCL

```hcl
role "reporting" {}

user "app_user" {
  default_schema = schema.dbo
  member_of      = [role.reporting]
}

user "svc" {
  login_name = "svc_login"
}
```

- A `user` takes `default_schema`, `login_name`, `default_language`,
  `allow_encrypted_value_modifications`, `comment`, and `member_of`; a `role` takes `member_of`, which
  builds role hierarchies (https://atlasgo.io/atlas-schema/hcl#enabling-roles-and-permissions). Roles
  and users share one namespace: a role and a user cannot have the same name.
- Without `default_schema`, Atlas creates the user `WITH DEFAULT_SCHEMA = [dbo]`.

## Permissions

The `permission` block grants on a schema, table, view, function, procedure, `type_alias`, `type_table`,
or a column of a table or view, to a role or user reference or a name as a string
(https://atlasgo.io/hcl/mssql):

```hcl
permission {
  to         = role.reporting
  for        = schema.sales
  privileges = [SELECT, EXECUTE]
}

permission {
  for_each   = [table.orders, table.customers]
  for        = each.value
  to         = user.app_user
  privileges = [INSERT, UPDATE]
}

permission {
  to         = role.reporting
  for        = table.orders.column.total
  privileges = [SELECT]
}
```

- `privileges` takes SQL Server object privileges such as `SELECT`, `INSERT`, `UPDATE`, `DELETE`,
  `EXECUTE`, `REFERENCES`, and `CONTROL`. A privilege with a space is written with an underscore.
- `grantable = true` adds `WITH GRANT OPTION`.
- With `permissions = true` and `roles` off, Atlas plans grants without managing principals: name the
  grantee as a string, `to = "app_reader"`.
- Grants are planned as `GRANT ... ON SCHEMA :: [s]`, `ON [s].[t]`, `ON OBJECT :: [s].[f]`, and
  `ON TYPE :: [s].[t]`, and removing a grant plans the matching `REVOKE`. Dropping an object drops its
  grants with it.

### Not managed

Atlas ignores these. Write them in a hand-written migration file (`atlas migrate new`), and expect no
drift report for them:

- `DENY`. A `DENY` in a SQL desired state is dropped without an error, and existing denies are not
  inspected, so Atlas never revokes them.
- Database-level permissions, such as `GRANT CREATE TABLE TO ...`, and grants on sequences.
- Grants to `public` or `guest`. A `GRANT ... TO public` in the desired state is dropped without an
  error.

## Ownership and EXECUTE AS

Object and schema owners are not part of the desired state: Atlas writes `CREATE SCHEMA [s]` without
`AUTHORIZATION`, and objects belong to the schema's owner. The exception is a full-text catalog's
`owner`. Functions, procedures, and triggers take `execute_as`, which sets their `EXECUTE AS` clause.

## Not Supported

Row-level security (`CREATE SECURITY POLICY`), dynamic data masking (`MASKED WITH`), and Always Encrypted
are not modeled. Atlas does not inspect or plan them, and a column's mask or encryption is not part of
the desired state. Apply them in hand-written migration files, and test them (`references/testing.md`).

## Errors

| Error | Fix |
|-------|-----|
| `unknown attribute external in block of type role` | SQL Server roles have no `external`; leave fixed and other unmanaged roles out of the schema |
| `User, group, or role 'db_...' already exists in the current database` | Remove the fixed-role `role` blocks from the HCL, or inspect with `--exclude 'db_*[type=role]'` |
| `cannot use tuple([string]) as references value in assignment` on `member_of` | `member_of` takes role references; write fixed-role membership in SQL |
| `'x' is not a valid login or you do not have permission` | Create the login on the dev server, in the dev `baseline` |
| `The server principal 'x' already exists` from `execute baseline` | Guard the login: `IF SUSER_ID('x') IS NULL CREATE LOGIN ...` |
| `duplicate user/role "x" found in spec` | A role and a user share a name |
| Every plan drops a user or role nobody declared | With `roles = true`, the desired state owns every role and user: declare it, or set `drop_user`/`drop_role` in the `skip` diff policy |
