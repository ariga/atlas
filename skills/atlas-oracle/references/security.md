# Oracle Security as Code

Roles, role hierarchies, and object and column grants. The guides:
https://atlasgo.io/guides/oracle/security-declarative and
https://atlasgo.io/guides/oracle/security-versioned.

## Turn It On

Roles and permissions are left out of inspection and planning by default, and managing them needs a
login. If `atlas whoami` shows no login, ask the user to run `atlas login`; a user without an account can
start a free trial with it.

Roles belong to the whole database, so managing them needs database scope, where plans have the gaps in
`references/dev-database.md`, Scope: they drop indexes, and with `permissions = true` a `sequence` in
the desired state fails planning with `drop schema named "APP": ORA-01922`. Split the work in two envs
instead:

```hcl
# Roles only, at database scope. No dev database is needed.
env "roles" {
  url = "oracle://APP:${urlescape(getenv("APP_PASSWORD"))}@db:1521/FREEPDB1?mode=database"
  schema {
    src = "file://roles.oc.hcl"
    mode {
      schemas = false
      objects = false
      roles   = true
    }
  }
  diff {
    skip {
      drop_role = true
    }
  }
}

# Tables, code, and the grants on them, at schema scope.
env "app" {
  url = "oracle://APP:${urlescape(getenv("APP_PASSWORD"))}@db:1521/FREEPDB1"
  dev = docker.oracle.dev.url
  schema {
    src = "file://schema.oc.hcl"
    mode {
      permissions = true
    }
  }
}
```

- `roles` plans only `CREATE ROLE`, role grants, and their changes; it never touches tables.
- `app` plans grants next to the objects they name, with the grantee as a string:
  `to = "APP_READONLY"`. At schema scope, inspection writes grants that way and lists no roles.
- The dev database for `app` needs the roles that grants name. Create them in the `docker` block's
  `init` (`references/dev-database.md`, The `docker` Block):

  ```hcl
  docker "oracle" "dev" {
    image    = "container-registry.oracle.com/database/free:latest-lite"
    database = "FREEPDB1"
    init     = <<-SQL
      CREATE TABLESPACE USERS DATAFILE 'users01.dbf' SIZE 100M AUTOEXTEND ON;
      ALTER DATABASE DEFAULT TABLESPACE USERS;
      CREATE ROLE APP_READONLY;
      CREATE ROLE APP_WRITER;
    SQL
  }
  ```

- Apply `roles` before `app`, so the roles exist when the grants run.
- The user Atlas connects as needs `CREATE ROLE`, besides the privileges in
  `references/dev-database.md`.
- The guides manage roles, tables, and grants in one database-scope env with `schemas = ["APP"]`. That
  works for a schema without secondary indexes or sequences; for any other, use the two envs above.

## Roles

```hcl
role "APP_READONLY" {
}

role "APP_WRITER" {
  member_of = [role.APP_READONLY]
}

role "APP_DELEGATE" {
  membership {
    role  = role.APP_READONLY
    admin = true
  }
}
```

- Atlas creates roles `NOT IDENTIFIED`. `member_of` plans `GRANT "APP_READONLY" TO "APP_WRITER"`, and a
  `membership` block with `admin = true` adds `WITH ADMIN OPTION`.
- `password`, `auth`, and `auth_package`, one at a time, create roles that need a password, external or
  global authorization, or a package (a secure application role). Only `NOT IDENTIFIED` roles are
  enabled at login as default roles; the others need `SET ROLE` or the package
  (https://docs.oracle.com/en/database/oracle/oracle-database/26/sqlrf/ALTER-USER.html).
- The `role` block has no `external` attribute. Every role that Oracle does not maintain is in scope,
  including other teams' roles, and a role the schema does not declare is planned for `DROP ROLE`. Keep
  `drop_role = true` in the env's `skip` policy (https://atlasgo.io/versioned/diff#diff-policy).
- The user Atlas connects as can grant or drop only the roles it created, which it holds
  `WITH ADMIN OPTION`, or roles granted to it that way. A role that another account created fails with
  `ORA-01924: Role "APP_READONLY" not granted or does not exist`. Have an administrator run
  `GRANT APP_READONLY TO APP WITH ADMIN OPTION;` once.
- Users are not managed. Inspection lists roles and their grants, but not users or the roles granted to
  them. Create login users, and grant them roles, outside Atlas: `GRANT "APP_WRITER" TO "HR_APP";`. A
  granted role reaches a user's sessions at the next login.

## Permissions

The `permission` block grants object privileges on a table, a view, a function, a procedure, or a
column. `for_each` expands one block into a grant per object:

```hcl
permission {
  for_each   = [table.EMPLOYEES, table.DEPARTMENTS]
  for        = each.value
  to         = "APP_READONLY"
  privileges = [READ]
}

permission {
  to         = "APP_PAYROLL"
  for        = table.EMPLOYEES.column.SALARY
  privileges = [UPDATE]
}
```

- `privileges` take `ALTER`, `DEBUG`, `DELETE`, `EXECUTE`, `INDEX`, `INSERT`, `READ`, `REFERENCES`,
  `SELECT`, `UPDATE`, and `WRITE`. Functions and procedures take `EXECUTE`.
- Prefer `READ` to `SELECT` for read-only access: `SELECT` also allows `SELECT ... FOR UPDATE` and
  `LOCK TABLE`, so a reader can lock rows (12.1.0.2 and later,
  https://docs.oracle.com/en/database/oracle/oracle-database/26/sqlrf/GRANT.html). Changing one to the
  other plans a `GRANT` and a `REVOKE`.
- Column grants exist only for `INSERT`, `UPDATE`, and `REFERENCES`. A revoke always covers the whole
  table, so removing one column of a multi-column grant plans a `REVOKE` and a new `GRANT` for the rest.
- `grantable = true` adds `WITH GRANT OPTION`, which Oracle refuses for a role grantee (`ORA-01926`).
  Use it only with a user in `to`.
- `REFERENCES` cannot be granted to a role either. A foreign key to another schema's table needs a direct
  grant to the owner of the referencing table.
- `for` cannot be a schema, and fails with
  `unsupported permission.for type "schema" ($schema.APP), expected table, view, function or procedure`.
  Write schema privileges (23ai and later), such as `GRANT SELECT ANY TABLE ON SCHEMA APP TO APP_READONLY`,
  and system privileges in hand-written migration files
  (https://docs.oracle.com/en/database/oracle/oracle-database/26/dbseg/managing-system-privileges.html).

## Roles in Code and Views

Oracle disables roles inside definer's-rights PL/SQL, the default for functions, procedures, packages,
and triggers, and when it compiles a view
(https://docs.oracle.com/en/database/oracle/oracle-database/26/dbseg/user-roles.html):

- A procedure that reads another schema's table compiles only if its owner holds the grant directly.
  Through a role, it fails to compile with `PL/SQL: ORA-00942: table or view does not exist`. Grant the
  privilege to the owner, not to a role it has.
- A view over another schema's table needs the same direct grant, and granting the view onward needs
  that grant `WITH GRANT OPTION`, or it fails with `ORA-01720: grant option does not exist`.
- Invoker's-rights code (`AUTHID CURRENT_USER`) runs with the caller's roles, but compiles like
  definer's-rights code.

## Applying

Each statement commits on its own (`references/migrations.md`). A failure leaves the statements before
it applied, and the next plan lists the rest. A grant on a table needs the table, so apply the env that
creates tables first when both change at once.

## Errors

| Error | Fix |
|-------|-----|
| `ORA-01924: Role "X" not granted or does not exist` | Another account created the role: grant it to the Atlas user `WITH ADMIN OPTION` |
| A plan drops other teams' roles | `diff { skip { drop_role = true } }` |
| `drop schema named "APP": ORA-01922: CASCADE must be specified to drop 'APP'` | Grants and sequences planned together at database scope: plan grants at schema scope |
| `DROP INDEX` statements in a database-scope plan | Plan tables and indexes at schema scope |
| `grant on table "DEPARTMENTS" to "APP_READONLY": ORA-01917: user or role 'APP_READONLY' does not exist` | The dev database lacks a role that a grant names: create it in `init` |
| `ORA-41900: missing COMMENT privilege` | Database scope on a 23ai or later dev image: `GRANT COMMENT ANY TABLE TO PDBADMIN;` in `init` |
| `ORA-01926` on `WITH GRANT OPTION` | Grant to a user, not a role |
| `unsupported permission.for type "schema"` | Write schema privileges by hand |
| `PL/SQL: ORA-00942` when compiling code that reads another schema | Grant the privilege directly to the code's owner |
