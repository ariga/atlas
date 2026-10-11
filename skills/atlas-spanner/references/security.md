# Spanner Security as Code

Database roles and grants, Spanner's fine-grained access control. Spanner has no database users:
Google Cloud IAM authenticates principals, and a principal acts as a database role to get the role's
privileges (https://docs.cloud.google.com/spanner/docs/fgac-about). Atlas manages the roles and their
grants; the IAM bindings that let principals use the roles stay outside Atlas.

## Turn It On

Roles and permissions are left out of inspection and planning by default, and managing them needs a
login. If `atlas whoami` shows no login, ask the user to run `atlas login`; a user without an account can
start a free trial with it.

```hcl
env "prod" {
  url = getenv("SPANNER_URL")
  dev = getenv("SPANNER_DEV_URL")
  schema {
    src = "file://schema.sp.hcl"
    mode {
      roles       = true
      permissions = true
    }
  }
}
```

The dev database must be an empty database on a real Spanner instance, not the emulator
(`references/dev-database.md`, Your Own Dev Database). The emulator parses role and grant statements
and does nothing with them
(https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/releases/tag/v1.5.41), so planning
against it fails:

| Mode | Error on the emulator |
|------|-----------------------|
| `permissions = true` | `iterating table acls: ... Table not found: INFORMATION_SCHEMA.TABLE_PRIVILEGES` |
| `roles = true` alone | `Abort: the planned state does not match the desired state after applying the file.` |
| A column grant in a SQL schema | `Emulator does not yet support column level access controls` |

## Roles and Grants

```hcl
role "hr_reader" {
}

role "hr_writer" {
  member_of = [role.hr_reader]
}

role "payroll" {
}

permission {
  to         = role.hr_reader
  for        = table.Employees
  privileges = [SELECT]
}

permission {
  to         = role.hr_writer
  for        = table.Employees
  privileges = [INSERT, UPDATE, DELETE]
}

permission {
  to         = role.payroll
  for        = table.Employees.column.Salary
  privileges = [SELECT, UPDATE]
}
```

Atlas plans:

```sql
CREATE ROLE `hr_reader`;
CREATE ROLE `hr_writer`;
GRANT ROLE `hr_reader` TO ROLE `hr_writer`;
CREATE ROLE `payroll`;
GRANT SELECT ON TABLE `Employees` TO ROLE `hr_reader`;
GRANT DELETE, INSERT, UPDATE ON TABLE `Employees` TO ROLE `hr_writer`;
GRANT SELECT (`Salary`), UPDATE (`Salary`) ON TABLE `Employees` TO ROLE `payroll`;
```

- `for` takes a table, a view, or a column of a table or view, and `privileges` take `SELECT`,
  `INSERT`, `UPDATE`, and `DELETE`. `DELETE` cannot be granted on a column
  (https://docs.cloud.google.com/spanner/docs/fgac-privileges). A `permission` on a schema fails with
  `unsupported change` (a schema modification).
- Model every grantee as a `role`. The `user` block plans no statement on Spanner, and a `permission`
  whose `to` is a `user` plans `GRANT ... TO ROLE <user>` for a role that does not exist.
- A role that holds privileges cannot be dropped, so Atlas plans each `REVOKE` before the `DROP ROLE`
  (https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language).
- A database holds up to 100 roles (https://docs.cloud.google.com/spanner/docs/fgac-about). The names
  `public` and those starting with `spanner_` are reserved in GoogleSQL, and names starting with `pg_` in
  the PostgreSQL dialect.
- The system roles `public`, `spanner_info_reader`, and `spanner_sys_reader` exist in every database
  and cannot be dropped. Every role inherits the privileges of `public`
  (https://docs.cloud.google.com/spanner/docs/fgac-system-roles).
- Grants that `permission` cannot express, such as `SELECT` on a change stream, `EXECUTE` on its read
  function, `SELECT` and `UPDATE` on a sequence, or `USAGE` on a schema, go in hand-written migration
  files. `SELECT` on a table does not grant `SELECT` on the change stream that watches it.

## Views with Definer's Rights

A view with `security = DEFINER` lets a role read it with a grant on the view alone, without grants on
the tables it reads (https://docs.cloud.google.com/spanner/docs/views):

```hcl
view "EmployeeNames" {
  schema   = schema.default
  as       = "SELECT e.Id, e.Name FROM Employees AS e"
  security = DEFINER
}

permission {
  to         = role.directory
  for        = view.EmployeeNames
  privileges = [SELECT]
}
```

An emulator older than 1.5.56 rejects `SQL SECURITY DEFINER` with `Expecting 'INVOKER' but found
'DEFINER'` (`references/dev-database.md`, Docker Dev URLs).

## IAM Outside Atlas

Who may act as a role is an IAM binding, not DDL
(https://docs.cloud.google.com/spanner/docs/configure-fgac):

- Grant the principal `roles/spanner.fineGrainedAccessUser`, and `roles/spanner.databaseRoleUser`
  with a condition that names the roles it may use:
  `resource.type == "spanner.googleapis.com/DatabaseRole" && resource.name.endsWith("/hr_reader")`.
  Without a condition, the principal can act as every role.
- A principal that also holds database-level IAM permissions, such as `roles/spanner.databaseUser`, uses
  them whenever it connects without naming a role, which bypasses fine-grained access control
  (https://docs.cloud.google.com/spanner/docs/access-with-fgac). Give application principals the
  fine-grained roles only.
- Atlas itself connects with database-level IAM permissions, so it sees the whole schema: it needs to
  read and update the schema (`spanner.databases.getDdl`, `spanner.databases.updateDdl`), and to read and
  write the revisions table. `roles/spanner.databaseUser` covers both
  (https://docs.cloud.google.com/spanner/docs/iam).
- Database exports do not include roles and privileges
  (https://docs.cloud.google.com/spanner/docs/fgac-about). A database restored from an export needs
  `atlas schema apply` with roles enabled to get them back.

## Errors

| Error | Fix |
|-------|-----|
| `Table not found: INFORMATION_SCHEMA.TABLE_PRIVILEGES` | The target or the dev database is the emulator: use a real Spanner database |
| `Emulator does not yet support column level access controls` | Plan column grants on a real Spanner dev database |
| `the planned state does not match the desired state` with roles enabled | The dev database is the emulator: use a real Spanner database |
| A grant to a `user` fails because the role does not exist | Declare the grantee as a `role` |
| `unsupported change` (a schema modification) | A `permission` on a schema: write schema grants by hand |
