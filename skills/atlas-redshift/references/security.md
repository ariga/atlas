# Redshift Security as Code

Users, groups, roles, memberships, and GRANT and REVOKE on Redshift. The guides:
https://atlasgo.io/guides/redshift/security-declarative and
https://atlasgo.io/guides/redshift/security-versioned. Every Redshift command needs `atlas login`. If
`atlas whoami` shows no login, ask the user to run `atlas login`; a user without an account can start a
free trial with it.

## Turn It On

Principals and permissions are left out of inspection and planning by default. Enable them in the env's
`schema` block, at database scope (no `search_path`, or `mode=database` on a Data API URL), with a dev
database on its own cluster or workgroup (`references/dev-database.md`):

```hcl
env "redshift" {
  url = getenv("REDSHIFT_URL")
  dev = getenv("REDSHIFT_DEV_URL")
  schema {
    src = "file://schema.rs.hcl"
    mode {
      roles       = true   // users, groups, and roles
      permissions = true   // GRANT and REVOKE
    }
  }
  include = [
    "analytics",
    "rpt_*[type=role]",
    "rpt_*[type=user]",
  ]
}
```

- Principals belong to the cluster, so inspection reads every user, group, and role on it, including IAM
  identities and other teams' roles. With `roles = true` and a dev database on its own cluster, the
  desired state is authoritative over all of them: a principal it does not declare is planned for a
  `DROP`. Scope the env with `include` patterns, or `exclude` the ones to leave alone, and declare the
  rest with `external = true`.
- With a dev database on the target's cluster, loading the desired state there brings back every
  principal the cluster holds, so an undeclared principal is part of the desired state and is never
  dropped. Removing a `user` or `role` block then plans nothing: drop the principal by hand.
- Inspection skips the user Atlas connects as, the AWS-internal `rdsdb` user, and the `sys:` and `ds:`
  roles. The connecting user therefore never shows up in plans as a drop, but declaring it as a plain
  `user` block plans a `CREATE USER` on every run, which fails on apply: leave it out, or declare it
  `external = true` when grants name it.
- `atlas migrate diff` reads `include` and `exclude` from the env's `migration` block, not from the env:

  ```hcl
  env "diff" {
    migration {
      dir     = "file://migrations"
      include = ["public", "rpt_*[type=role]", "rpt_*[type=user]"]
    }
  }
  ```

- To manage only principals and grants, turn schemas off: `mode { schemas = false, roles = true }`. The
  state then holds no `schema` blocks.
- With `permissions = true` and `roles` off, Atlas plans grants without managing principals; grants on
  objects the state does not define name them with `ref()` (Permissions below).
- The `skip` diff policy takes `drop_role = true` and `drop_user = true` to keep drops of principals out
  of plans (https://atlasgo.io/versioned/diff#diff-policy).

## Users, Groups, and Roles

Redshift keeps three kinds of principal, and Atlas writes the statement each one needs:

| Kind | HCL | Created with |
|------|-----|--------------|
| RBAC role | `role "reader" {}` | `CREATE ROLE "reader"` |
| Group | `role "analysts" { group = true }` | `CREATE GROUP "analysts"` |
| User | `user "etl" {}` | `CREATE USER "etl" PASSWORD DISABLE` |

```hcl
role "rpt_readonly" {}
role "rpt_writer" {
  member_of = [role.rpt_readonly]
}
role "rpt_analysts" {
  group = true
}
user "rpt_etl" {
  conn_limit = 10
  member_of  = [role.rpt_writer]
  params = {
    statement_timeout = 60000
  }
}
user "rpt_bi" {
  valid_until = "2030-01-01 00:00:00"
  member_of   = [role.rpt_analysts]
  membership {
    role  = role.rpt_readonly
    admin = true
  }
}
```

- Roles and users have separate namespaces: `role "x"` and `user "x"` are two principals.
- `member_of` puts a role in a role (`GRANT ROLE "a" TO ROLE "b"`), a user in a role
  (`GRANT ROLE "a" TO "u"`), or a user in a group (`ALTER GROUP "g" ADD USER "u"`). Only users can join
  a group: anything else fails with
  `redshift: only a user can be a member of group "g", but "x" is a role`.
- A `membership` block with `admin = true` adds `WITH ADMIN OPTION`. Redshift takes it only for a role
  granted to a user: a group or a role member fails with
  `redshift: the membership of "x" in group "g" cannot be granted with the admin option`.
- A user takes `password`, `valid_until`, `superuser`, `create_db`, `conn_limit`, and `params`; roles
  and groups take none of them. Unsetting one plans its negative form, such as
  `CONNECTION LIMIT UNLIMITED`.
- `params` pins session parameters with `ALTER USER ... SET`. Without a `params` block, Atlas leaves the
  pinned values alone; an empty `params = {}` resets every pin to `DEFAULT`.
- Inspection reports groups as `role` blocks with `group = true`, and merges group members and role
  grants into one membership list.

## External, Predefined, and IAM Principals

- `external = true` on a `role` or `user` marks a principal Atlas never creates, alters, or drops. Use
  it for the admin user, the user Atlas connects as when grants name it, and other teams' principals.
  Atlas creates no principal on the dev database, external or not.
- Principals whose names start with `sys:` are provisioned by Redshift, and the `ds:` roles belong to
  datashares. Atlas never plans them, and never reads the privileges of a `ds:` role as grants.
- Federated identities are users named with an `IAM:` or `IAMR:` prefix. Atlas manages them like any
  user; reference them with index syntax, as the name holds a colon:

  ```hcl
  user "IAM:alice" {
    member_of = [role.rpt_readonly]
  }

  permission {
    to         = user["IAM:alice"]
    for        = table.orders
    privileges = [SELECT]
  }
  ```

## Passwords

- Inspection never reads a password back. Atlas compares passwords only with `sensitive = ALLOW` in the
  env's `schema.mode` block; otherwise a password change is never planned.
- `atlas schema apply` writes a declared password into `CREATE USER` and `ALTER USER`.
  `atlas migrate diff` never writes one: a new user gets `PASSWORD DISABLE`, so the file holds no
  secret, and a password change alone produces no statement. Set the credential out of band, manage
  it declaratively with `sensitive = ALLOW`, or let IAM authenticate a user declared
  `external = true` (https://atlasgo.io/guides/redshift/security-versioned).
- Pass passwords in through input variables, never as literals (`references/hcl.md`).
- Redshift rejects a superuser without a password. A user with `superuser = true` and no `password`
  fails with `redshift: a superuser requires a password, but "u" defines none`, and a migration file
  cannot create a superuser, since it never carries the password. Dropping a superuser has no reverse
  statement for `atlas migrate down`.

## Permissions

```hcl
permission {
  to         = role.rpt_readonly
  for        = schema.reporting
  privileges = [USAGE]
}

permission {
  for_each   = [table.customers, table.orders]
  for        = each.value
  to         = role.rpt_readonly
  privileges = [SELECT]
}

permission {
  to         = role.rpt_analysts
  for        = table.customers.column.region
  privileges = [SELECT]
}
```

- `to` takes a `role` reference (role or group), a `user` reference, `PUBLIC`, or a string, which names
  a user. Atlas writes `TO ROLE "x"`, `TO GROUP "x"`, `TO PUBLIC`, or a bare user name.
- `for` takes a schema, table, view, materialized view, function, or procedure, or one column of a
  table, view, or materialized view.
- `privileges` takes `SELECT`, `INSERT`, `UPDATE`, `DELETE`, `DROP`, `REFERENCES`, `TRUNCATE`, `ALTER`,
  `TRIGGER`, `RULE`, `CREATE`, `USAGE`, `EXECUTE`, and `ALL`. Atlas expands `ALL` to what Redshift
  reports for it: `ALTER, CREATE, DROP, USAGE` on a schema; `ALTER, DELETE, DROP, INSERT, REFERENCES,
  RULE, SELECT, TRIGGER, TRUNCATE, UPDATE` on a table or view; `SELECT, UPDATE` on a column; `EXECUTE`
  on a routine.
- `grantable = true` adds `WITH GRANT OPTION`. Redshift accepts it only for a user grantee; a role or
  group fails with `redshift: the grant option cannot be granted to role "x"`.
- A grantee that holds column privileges on a table cannot be given table privileges on it first, so
  Atlas plans every `REVOKE` on a table before every `GRANT`.
- Function and procedure grants name the exact signature:
  `GRANT EXECUTE ON FUNCTION "s"."f" ("a" integer) TO "u"`.
- Redshift reports routine grants by name, not by signature, so inspection attaches a grant to every
  overload of the name. A `permission` on `function.f` covers every overload, and Atlas grants each
  one. A grant written by hand on one overload alone shows up in every plan: grant all overloads the
  same.
- Grants on the database itself, and grants scoped to every schema of the database
  (`GRANT ... FOR SCHEMAS IN DATABASE`), are not inspected or planned.
- A grant on an object the state does not define, in a permissions-only env, uses `ref()` with the
  object's kind, and a routine's argument types:

  ```hcl
  permission {
    to         = "xpr_user"
    for        = ref("table.xpr_s.t")
    privileges = [INSERT, SELECT]
  }
  permission {
    to         = "xpr_user"
    for        = ref("function.xpr_s.f(integer)")
    privileges = [EXECUTE]
  }
  ```

  Write the kind inspection reports: naming a view as `ref("view.s.t")` and a table as
  `ref("table.s.t")` are two different grants to Atlas.

### `PUBLIC` on functions

Redshift grants `EXECUTE` to `PUBLIC` on every new function, not on procedures. With
`permissions = true`, a function whose state does not declare that grant is created and then revoked
in the same plan:

```sql
CREATE FUNCTION "sales"."calc" (integer) RETURNS integer IMMUTABLE AS $$SELECT $1$$ LANGUAGE sql;
REVOKE EXECUTE ON FUNCTION "sales"."calc" (integer) FROM PUBLIC;
```

To keep it, declare it:

```hcl
permission {
  to         = PUBLIC
  for        = function.calc
  privileges = [EXECUTE]
}
```

## Errors

| Error | Fix |
|-------|-----|
| A plan drops principals no one declared | Scope the env with `include`, and mark the others `external = true` |
| Every plan creates the user Atlas connects as | Inspection skips it: remove its `user` block, or set `external = true` |
| `permission.for reference to undeclared function "..." must carry the exact signature Redshift grants by` | Write the argument types: `ref("function.s.f(integer)")` |
| `permission.for reference to undeclared function "..." must be qualified as ref("function.<schema>.<name>(<args>)")` | Put the schema and the argument types in the reference |
| Replaying migrations fails on the dev database for a user or role that already exists | The dev database shares the target's cluster; give it its own cluster or workgroup |
| A user or role removed from the schema stays, and the plan reports no changes | The dev database shares the target's cluster: drop it by hand, or give the dev database its own cluster |
| `redshift: only a user can be a member of group "..."` | Put roles in roles, and users in groups |
| `redshift: the membership of "..." in ... cannot be granted with the admin option` | `admin = true` only for a role granted to a user |
| `redshift: the grant option cannot be granted to role "..."` | Use `grantable` with a user grantee |
| A grant on one overload of a function shows up in every plan | Redshift reports routine grants by name: grant every overload the same |
| `redshift: a superuser requires a password, but "..." defines none` | Declare `password` through a variable, or drop `superuser` |
| `redshift: ... "..." has no alterable attributes` | A role or group has no user attributes; remove them from the block |
