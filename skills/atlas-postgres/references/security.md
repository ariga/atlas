# PostgreSQL Security as Code

Roles, users, GRANT and REVOKE, default privileges, ownership, row-level security, and the admin roles
of managed providers. The guides: https://atlasgo.io/guides/postgres/security-declarative,
https://atlasgo.io/guides/postgres/security-versioned, and
https://atlasgo.io/guides/postgres/security-default-privileges.

## Turn It On

Roles and permissions are left out of inspection and planning by default, and managing them needs a
login. If `atlas whoami` shows no login, ask the user to run `atlas login`; a user without an account can
start a free trial with it. Enable them in the env's `schema` block
(https://atlasgo.io/atlas-schema/hcl#enabling-roles-and-permissions):

```hcl
env "dev" {
  ...
  schema {
    src = "file://schema.hcl"
    mode {
      roles       = true      # Enable role inspection. Defaults to false.
      permissions = true      # Enable permission inspection. Defaults to false.
      sensitive   = ALLOW     # or DENY (default) - controls password handling
    }
  }
}
```

- Roles belong to the instance, not to a schema. An env that manages roles uses database scope: no
  `search_path` in the URL or the dev URL. The dev database must be an instance of its own, such as a
  Docker container: on the target's instance, its roles collide with the live ones.
- With `roles = true`, the desired state is authoritative over every role Atlas inspects; predefined
  `pg_*` roles are excluded. Declare every other role Atlas must not touch with `external = true`: the
  role Atlas connects as, the admin, replication roles, provider roles such as `rds_superuser`, and other
  teams' roles. Atlas never plans a `CREATE`, `ALTER`, or `DROP` for an external role.
- A grant or membership that names an external role needs that role in the dev `baseline`, as in the
  RDS example under Managed Providers.
- The `skip` diff policy takes `drop_role = true` and `drop_user = true` to keep drops of roles and
  users out of plans (https://atlasgo.io/versioned/diff#diff-policy).
- With `permissions = true` and `roles` off, Atlas plans grants without managing roles. Name the grantee
  as a string: `to = "app_reader"`.
- Default privileges as `default_permission` blocks are enabled apart, in the `mode` block labeled
  `postgres`, and only in an env that manages access to schemas whose objects another tool creates
  (Default Privileges below):

  ```hcl
  mode "postgres" {
    roles               = true
    permissions         = true
    default_permissions = true
  }
  ```
- To manage only roles and users, disable `schemas` and `objects`. This env needs no dev database
  (https://atlasgo.io/atlas-schema/projects):

  ```hcl
  env "roles" {
    url = getenv("DATABASE_URL")
    schema {
      src = "file://roles.hcl"
      mode {
        schemas = false
        objects = false
        roles   = true
      }
    }
  }
  ```

- Passwords: inspection cannot read a password back, so Atlas never detects a password changed outside
  it. With the default `sensitive = DENY`, Atlas leaves declared passwords out of the plan without an
  error. The declarative workflow needs `sensitive = ALLOW`, and prints passwords masked. The versioned
  workflow leaves passwords out of migration files; set them from a template directory with the
  `-- atlas:sensitive` directive
  (https://atlasgo.io/guides/postgres/security-versioned):

  ```sql
  -- atlas:sensitive
  ALTER ROLE "api_user" WITH PASSWORD '{{ .api_password }}';
  ```

## Default Privileges

Who creates the objects decides (https://atlasgo.io/guides/postgres/security-default-privileges):

- Atlas creates them, in the declarative or the versioned workflow: declare their grants with
  `permission` blocks or `GRANT` statements, and keep `default_permissions` off. Atlas plans and checks
  the grants of every object, and accounts for default privileges only on the objects it creates.
- Another tool creates them, such as the application's migration tool: Atlas cannot declare their
  grants, but it can manage the default privileges that grant access to them as they are created (When
  another tool creates the objects, below).

Never enable `default_permissions` in an env that migrates the schema. Atlas is declarative, and default
privileges are not the state of any object: they only decide what objects created later get, so existing
objects keep their grants, and the grants of an object depend on when it was created.

### When Atlas creates the objects

In a SQL schema, Atlas runs `ALTER DEFAULT PRIVILEGES` on the dev database while loading the desired
state. Every object defined after it receives the privileges, and the plan grants them object by object:

```sql
ALTER DEFAULT PRIVILEGES GRANT SELECT ON TABLES TO PUBLIC;

CREATE TABLE t1 (id integer);
CREATE TABLE t2 (id integer);
```

```sql
-- Create "t1" table
CREATE TABLE "public"."t1" ("id" integer NULL);
-- Grant on table "t1" to "PUBLIC"
GRANT SELECT ON TABLE "public"."t1" TO PUBLIC;
-- Create "t2" table
CREATE TABLE "public"."t2" ("id" integer NULL);
-- Grant on table "t2" to "PUBLIC"
GRANT SELECT ON TABLE "public"."t2" TO PUBLIC;
```

- No `ALTER DEFAULT PRIVILEGES` statement appears in a generated migration: the grants do. Tables that
  already exist on the target get the GRANTs they are missing.
- The statement is order-dependent: put it before the objects it covers, as the first imported file or
  the opening statement of `schema.sql`.
- Removing it later generates the matching `REVOKE` statements.
- Without `FOR ROLE`, the defaults apply to objects created by the role that runs the statement. On a
  Docker dev database that role is `postgres`, which creates every object, so leave `FOR ROLE` out, or
  name the role the dev URL connects as.
- To match a target whose defaults differ from a fresh container, such as a provider image or hardened
  defaults, run the same statement in the dev database's `baseline`. Atlas reads the dev database's
  `pg_default_acl` to decide what a new object receives:

  ```hcl
  docker "postgres" "hardened" {
    image = "postgres:18"
    baseline = <<-SQL
      ALTER DEFAULT PRIVILEGES REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC;
    SQL
  }
  ```

- In an HCL schema, any implicit privilege the schema does not declare gets a `REVOKE` right after the
  `CREATE`; declare `permission { to = PUBLIC ... }` to keep one. In a SQL schema, Atlas does not revoke
  implicit `PUBLIC` grants on its own.
- When the target itself has default ACLs, a new table or function receives their grants when it is
  created, and can need a second apply to match what the schema declares.

### One role on every table

When Atlas creates the objects, a default privilege, or a `for_each` permission, still becomes one GRANT
per object, so giving a new role access to every table in a large database produces thousands of
statements. For read access to
all data, make the role a member of PostgreSQL's predefined `pg_read_all_data` role instead (PostgreSQL
14 and later, https://www.postgresql.org/docs/current/predefined-roles.html): it reads every table,
view, and sequence in every schema, and row-level security still applies to it. Predefined roles need
no declaration; declare one as external only to reference it
(https://atlasgo.io/atlas-schema/hcl#external-roles-and-users):

```hcl
# A PostgreSQL predefined role
role "pg_read_all_data" {
  external = true
}

# The identity Atlas connects with
user "admin" {
  external = true
}

role "app_readonly" {
  member_of = [role.pg_read_all_data]
}
```

In a SQL schema: `CREATE ROLE "app_readonly" IN ROLE "pg_read_all_data";`.

### When another tool creates the objects

Manage access from an env of its own, which excludes the other tool's objects and enables the mode:

```hcl
env "prod" {
  url     = getenv("DATABASE_URL")
  dev     = "docker://postgres/18/dev"
  exclude = [
    "public.*",                          # the objects of the migration tool
    "*[type=default_permission].PUBLIC", # PostgreSQL's built-in PUBLIC privileges
  ]
  schema {
    src = "file://schema.pg.hcl"
    mode "postgres" {
      roles               = true
      permissions         = true
      default_permissions = true
    }
  }
}
```

```hcl
default_permission {
  owner      = user.migrations   # FOR ROLE; omitted: the role Atlas connects as
  schema     = schema.public     # IN SCHEMA; omitted: every schema
  on         = TABLES            # TABLES, SEQUENCES, FUNCTIONS, TYPES or SCHEMAS
  to         = user.app
  privileges = [DELETE, INSERT, SELECT, UPDATE]
}
```

- Start from `atlas schema inspect --env prod`. The blocks are the whole state for these object types:
  Atlas revokes every default privilege that no block declares, for every role it inspects.
- Both URLs need database scope: no `search_path` in `url` or `dev`.
- `public.*` excludes the objects of a schema, not the schema: its grants, extensions, and default
  privileges stay managed. Excluding the schema itself would exclude the default privileges set in it.
- Without the `PUBLIC` exclusion, inspection prints the built-in `FUNCTIONS` and `TYPES` blocks of every
  role for `PUBLIC`, and a role added without them gets them revoked. `privileges` cannot be empty: to
  revoke a built-in, delete its block.
- Review the plan with `atlas schema apply --env prod --dry-run` before applying it. To check for
  drift, `atlas schema diff --env prod --from env://url --to env://schema.src` prints the
  `ALTER DEFAULT PRIVILEGES` that restore the file. With `--format '{{ sql . }}'` it prints nothing
  when the database matches, which a scheduled CI job on the default branch can test. Keep the
  database URL in a scope only the default branch can read, such as a protected environment.
- Default privileges never change existing objects. Grant on those once, outside Atlas, for example
  `GRANT SELECT ON ALL TABLES IN SCHEMA public TO reader;` in the other tool's migrations.
- Before a `DROP ROLE`, Atlas revokes the role's default privileges and what they gave it on all
  tables, sequences, and routines of their schemas, and on the types and schemas Atlas manages.
  Privileges on types Atlas does not manage, and on excluded objects in schemas that no default
  privileges cover, need a manual `REVOKE`. With `schemas = false`, Atlas revokes only in the schemas
  that default privileges are defined in.
- A non-superuser Atlas user needs the privileges of each role whose default privileges it changes
  (`GRANT migrations TO atlas WITH INHERIT TRUE;`) and, to set those of the roles it creates,
  `ALTER ROLE atlas SET createrole_self_grant = 'inherit';`. Both need PostgreSQL 16 or later. Without
  them, the apply fails with `permission denied to change default privileges`. On earlier versions,
  use `GRANT migrations TO atlas;`. Exclude the admin role the user then sees:
  `postgres[type=role|user|default_permission]`.
- On managed services, exclude the provider's roles with their default privileges (Managed Providers).

## Ownership

Object owners are not part of the desired state: the schema format has no owner attribute for tables,
views, or functions (`sequence.owner` is the `OWNED BY` column, see `references/objects.md`). On a Docker
dev database, every object belongs to `postgres`, the role the dev URL connects as. Ownership still
matters for row-level security: the table owner bypasses every policy unless the table enforces it.

## Memberships

`member_of` lists the roles a role belongs to. To set a membership's options, use a `membership` block
with `admin`, `inherit`, and `set`; `inherit` and `set` need PostgreSQL 16 or later
(https://atlasgo.io/hcl/postgres#role.membership):

```hcl
user "vault" {
  membership {
    role  = role.reader
    admin = true
  }
}
```

## Permissions

The `permission` block grants on a schema, table, view, materialized view, function, procedure,
sequence, partition, foreign table, or columns, to `PUBLIC`, a role name, or a role reference
(https://atlasgo.io/hcl/postgres). `for_each` expands one block into a grant per object at plan time:

```hcl
permission {
  for_each   = [table.orders, table.users, table.products]
  for        = each.value
  to         = role.reader
  privileges = [SELECT]
}
```

- `grantable = true` adds `WITH GRANT OPTION`.
- On a materialized view, only `SELECT` is meaningful, but Atlas plans any table privilege the schema
  declares.
- In a SQL schema, `GRANT ... ON ALL TABLES IN SCHEMA` covers only the tables created before it, as in
  PostgreSQL. Put it after the tables, or put `ALTER DEFAULT PRIVILEGES` before them.
- A grant on the implicit sequence of a `serial` column changes how the column is managed; see
  Sequences in `references/objects.md`.

## Managed Providers

On managed services, the admin user is not a real superuser: `postgres` on Cloud SQL, the master user
on RDS and Aurora (a member of `rds_superuser`), the admin on Azure Flexible Server (`azure_pg_admin`),
and the admin on AlloyDB (`alloydbsuperuser`). With `permissions = true`, its grants show up in
inspection, and a diff through a local dev database plans a `REVOKE` from the admin on every object
(https://atlasgo.io/faq/cloud-sql-postgres-grants).

- Exclude the admin role, so Atlas never plans to create or drop it:

  ```hcl
  env "staging" {
    url = getenv("DATABASE_URL")
    dev = "docker://postgres/16/dev"
    exclude = [
      "postgres[type=role|user]",
    ]
  }
  ```

  This removes only the role object; its grants still show up.
- With `default_permissions`, exclude the provider's roles together with their default privileges, as
  the Atlas user cannot change those: `rds*[type=role|user|default_permission]` on RDS and Aurora,
  `cloudsql*[type=role|user|default_permission]` on Cloud SQL, and
  `azure*[type=role|user|default_permission]` on Azure.
- Grants to the admin cannot be planned through a local dev database, where `postgres` is the
  superuser and its grants are implicit. To manage them, use a dev database where the admin is an
  ordinary role, such as a second Cloud SQL instance. The other providers behave the same with their
  admin role.
- A schema that references provider roles needs them in the dev database. For RDS:

  ```hcl
  docker "postgres" "rds" {
    image = "postgres:16"
    baseline = <<-SQL
     CREATE ROLE rds_superuser;
     CREATE ROLE rds_replication;
     CREATE ROLE rds_iam;
     CREATE ROLE rdsadmin;
     GRANT rds_superuser TO postgres;
    SQL
  }
  ```

- RDS IAM authentication: the user Atlas connects as needs `member_of = [role.rds_iam]`, the AWS
  `rds-db:connect` permission, and `sslmode=require`. RDS owns `rds_iam`, so declare it as
  `role "rds_iam" { external = true }` (https://atlasgo.io/guides/deploying/rds-iam-migration-role).
- Supabase roles, grants, and their dev database are covered in `references/dev-database.md`.

## Row-Level Security

- A table enables policies with `row_security { enabled = true }`; `enforced = true` adds
  `FORCE ROW LEVEL SECURITY` (https://atlasgo.io/hcl/postgres). Policies are `AS PERMISSIVE` unless
  declared restrictive. A policy without `to` applies to every role.
- Restrictive policies only narrow what permissive policies grant. A table with only restrictive
  policies returns no rows to the roles row-level security applies to.
- Row-level security does not apply to superusers, roles with `BYPASSRLS`, or the table owner unless
  the table enforces it (https://atlasgo.io/guides/rls-policy).
- Test policies as a non-privileged role: the Docker dev database connects as the `postgres` superuser,
  which bypasses every policy (Row-level security in `references/testing.md`).

## Errors

| Error | Fix |
|-------|-----|
| `modify schema "public" is not allowed when migration plan is scoped to one schema` | With `permissions = true`, declare `USAGE` on `public` for `PUBLIC`, or Atlas plans to revoke it on a schema-scoped dev database |
| A `REVOKE` from the provider admin (`postgres` on Cloud SQL, the RDS master user, or another admin) on every object | Plan through a dev database where the admin is an ordinary role, as in Managed Providers |
| `role "anon" already exists` | The Supabase image already has the role; drop it from the dev `baseline` |
| `role "X" already exists` on the dev database | Roles belong to the instance, so a reused or shared dev database keeps them from an earlier run; use a Docker dev database, which starts empty on every run |
