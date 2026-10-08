# MySQL Security as Code

Users, roles, GRANT and REVOKE, and passwords. The guides:
https://atlasgo.io/guides/mysql/security-declarative and
https://atlasgo.io/guides/mysql/security-versioned.

## Turn It On

Roles and permissions are left out of inspection and planning by default, and managing them needs a
login. If `atlas whoami` shows no login, ask the user to run `atlas login`; a user without an account can
start a free trial with it. Enable them in the env's `schema` block
(https://atlasgo.io/guides/mysql/security-declarative):

```hcl
env "local" {
  url = getenv("DATABASE_URL")
  schema {
    src = "file://schema.my.hcl"
    mode {
      roles       = true   // Inspect and manage roles and users
      permissions = true   // Inspect and manage GRANT / REVOKE
      sensitive   = ALLOW  // Allow password handling in declarative mode
    }
  }
}
```

- Users and roles belong to the server, not to a database. The URL and the dev URL use server scope,
  without a database in the path, such as `docker://mysql/8` for the dev database.
- With `roles = true`, the desired state is authoritative over every user and role Atlas inspects on
  the server (https://atlasgo.io/atlas-schema/hcl#external-roles-and-users). MySQL has no `external`
  attribute, which PostgreSQL uses to protect accounts, so an account the schema does not declare is
  planned for a drop, including the user Atlas connects as. Declare that user in the schema, and review
  the first plan for other accounts. To keep drops out of every plan, set `drop_user = true` and
  `drop_role = true` in the env's `diff { skip { ... } }` policy
  (https://atlasgo.io/versioned/diff#diff-policy).
- Passwords come from input variables, never from literals in the file. The declarative workflow needs
  `sensitive = ALLOW`, and the plan masks them:
  `CREATE USER (sensitive)@(sensitive) IDENTIFIED BY (sensitive)`.
- The versioned workflow leaves passwords out of migration files. On MySQL, the user is created
  `IDENTIFIED WITH caching_sha2_password`. Set the passwords from a template directory, with the
  `-- atlas:sensitive` directive so they are never logged
  (https://atlasgo.io/guides/mysql/security-versioned):

  ```sql
  -- atlas:sensitive
  ALTER USER `api_user` IDENTIFIED BY '{{ .api_password }}';
  ```

## Users and Roles

A `user` is a login account; a `role` cannot authenticate. `member_of` builds a hierarchy, where each
role inherits the privileges of the roles it is a member of:

```hcl
variable "api_password" {
  type = string
}

role "app_readonly" {
  comment = "Read-only access for reporting"
}

role "app_writer" {
  comment   = "Read-write access for the application"
  member_of = [role.app_readonly]
}

user "api_user@%" {
  password             = var.api_password
  max_user_connections = 20
  comment              = "Application API service account"
  member_of            = [role.app_writer]
}
```

- The host is part of the label: `@%` for any host, `@localhost` for local connections, or
  `@10.0.0.%` for a subnet. A name with a host is referenced with brackets: `role["reader@localhost"]`,
  `user["api_user@%"]`.
- Granted roles are inactive at login by default. On MySQL, run `SET DEFAULT ROLE ALL TO 'user'@'host'`
  for each user, or enable `activate_all_roles_on_login` on the server. On MariaDB, a user has one
  default role, set with `SET DEFAULT ROLE <role> FOR 'user'@'host'`. The schema has no attribute for
  default roles.
- Account limits: `max_user_connections` (simultaneous), `max_connections`, `max_questions`, and
  `max_updates` (per hour), plus `password_lifetime`, `password_expired`, and `comment`
  (https://atlasgo.io/hcl/mysql).
- An RDS IAM user sets `auth_plugin` and no password. It is the HCL form of
  `CREATE USER 'app_iam' IDENTIFIED WITH AWSAuthenticationPlugin AS 'RDS';`
  (https://atlasgo.io/atlas-schema/hcl#external-roles-and-users):

  ```hcl
  user "atlas_migrations" {
    auth_plugin = "AWSAuthenticationPlugin"
  }
  ```

  The dev database needs a mock of the plugin, whether the users come from HCL or from migration files
  (`references/dev-database.md`, RDS IAM users on the dev database). With `roles = true`, declare every
  IAM user, including the one Atlas itself connects as: an undeclared user can be planned for a drop.

## Permissions

The `permission` block grants on a schema (a database), a table, a view, a function, a procedure, or a
single column. `to` takes a role name as a string, a role reference, or a user reference such as
`user["api_user@%"]`. Routines take `EXECUTE`. `for_each` expands one block into a grant per object
(https://atlasgo.io/guides/mysql/security-declarative):

```hcl
permission {
  for_each   = [table.orders, table.products, table.users]
  for        = each.value
  to         = role.app_readonly
  privileges = [SELECT]
}

permission {
  to         = user["api_user@%"]
  for        = table.users.column.email
  privileges = [UPDATE]
}
```

A column grant covers one column; grant each column in its own block.
