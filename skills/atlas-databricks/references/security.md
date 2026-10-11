# Databricks Security as Code

Grants on schemas, tables, views, materialized views, functions, and procedures, the Unity Catalog
principals they name, and the column masks and row filters that limit what a principal reads. The HCL
blocks: https://atlasgo.io/hcl/databricks#permission.

## Turn It On

Atlas inspects and plans grants only when the env enables permissions:

```hcl
env "prod" {
  url = "${local.dbx}?catalog=prod"
  dev = "${local.dbx}?catalog=atlas_dev"
  schema {
    src = "file://schema.dbx.hcl"
    mode {
      permissions = true
    }
  }
}
```

- Without the mode, `schema inspect` prints no `permission` blocks, and Atlas does not read the
  target's grants. Turn it on whenever the schema declares `permission` blocks.
- With the mode, the desired state is the full list of direct grants on every object Atlas manages. A
  grant the target holds and the schema does not declare is planned as a `REVOKE`. Before the first
  apply, run `atlas schema inspect --env <name>` with the mode on and copy the existing grants into the
  schema, or review the first plan for revokes.
- Atlas reads only direct grants. A privilege a principal holds through its catalog or schema is not
  listed on each table, and Atlas leaves it alone. Schema grants to the `_workspace...` principals that
  Databricks manages are skipped too.

## Principals

Atlas has no blocks for users, groups, or service principals, and creates none. A grant names a
principal that already exists in the account: Databricks resolves it when the `GRANT` runs.

| Principal | `to` value |
|-----------|-----------|
| User | The user's email, such as `"ana@example.com"` |
| Group | The group name, such as `"data-engineers"` |
| Service principal | Its application ID, such as `"e2d9532b-b61a-46ca-8bf3-aa9ed2060f46"` |

A grant to a principal that does not exist fails on Databricks, and on the dev catalog first when Atlas
loads the desired state there.

## Permissions

```hcl
permission {
  to         = "e2d9532b-b61a-46ca-8bf3-aa9ed2060f46"
  for        = table.users
  privileges = [SELECT, MODIFY]
}

permission {
  to         = "data-engineers"
  for        = schema.default
  privileges = [USE_SCHEMA, CREATE_TABLE]
}

permission {
  to         = "data-engineers"
  for        = function.bump
  privileges = [EXECUTE]
}
```

- `for` takes a `schema`, `table`, `view`, `materialized`, `function`, or `procedure` reference.
  Catalogs are not objects in Atlas: grant `USE CATALOG` and other catalog privileges outside Atlas.
- Privileges are written with underscores in HCL and with spaces in SQL: `USE_SCHEMA` plans
  `GRANT USE SCHEMA ON SCHEMA ...`. The accepted names are `ALL_PRIVILEGES`, `APPLY_TAG`, `BROWSE`,
  `CREATE_FEATURE`, `CREATE_FUNCTION`, `CREATE_MATERIALIZED_VIEW`, `CREATE_MODEL`, `CREATE_SCHEMA`,
  `CREATE_SECRET`, `CREATE_SERVICE`, `CREATE_TABLE`, `CREATE_VOLUME`, `DELETE`, `EXECUTE`,
  `EXTERNAL_USE_SCHEMA`, `INSERT`, `MANAGE`, `MODIFY`, `READ_FEATURE`, `READ_METADATA`, `READ_SECRET`,
  `READ_VOLUME`, `REFERENCE_SECRET`, `REFRESH`, `SELECT`, `UPDATE`, `USE_CATALOG`, `USE_SCHEMA`,
  `WRITE_SECRET`, and `WRITE_VOLUME`. Databricks decides which of them an object accepts.
- Each `GRANT` names one principal; Atlas groups that principal's privileges on one object into one
  statement: ``GRANT MODIFY, SELECT ON TABLE `default`.`users` TO ...``.
- Databricks has no `WITH GRANT OPTION`, so the `permission` block has no `grantable` attribute.
- A `permission` with an empty `privileges` list fails with
  `no privileges specified for ACL on table "..."`.

### Objects the schema does not declare

To grant on an object that another schema file or another team manages, name it with `ref()`. Atlas
plans the grants and revokes and never creates, alters, or drops the object itself:

```hcl
permission {
  to         = "e2d9532b-b61a-46ca-8bf3-aa9ed2060f46"
  for        = ref("table.default.users")
  privileges = [MODIFY]
}

permission {
  to         = "e2d9532b-b61a-46ca-8bf3-aa9ed2060f46"
  for        = ref("view.default.active_users")
  privileges = [SELECT]
}

permission {
  to         = "e2d9532b-b61a-46ca-8bf3-aa9ed2060f46"
  for        = ref("schema.default")
  privileges = [SELECT]
}
```

`ref()` takes a schema, a table, a view, or a materialized view. In a `composite_schema`, one part can
declare the objects and another the grants on them; both load on the dev catalog, so the grants plan
as ordinary ones (https://atlasgo.io/atlas-schema/projects#data-source-composite_schema).

### Rewrites keep grants

`CREATE OR REPLACE` drops the grants on a view, a materialized view, a function, or a procedure. When a
change rewrites one of them (`references/objects.md`), Atlas plans the grants again right after the
rewrite, so the plan shows `GRANT` statements for privileges that did not change. That is expected. A
plain view whose query changes is altered with `ALTER VIEW ... AS`, which keeps its grants.

## Ownership

Atlas does not inspect or change owners. Objects it creates belong to the principal in the URL. Set
owners outside Atlas, or in a hand-written migration with `ALTER TABLE ... SET OWNER TO ...`, and keep
the principal that runs Atlas able to alter every object it manages.

## Column Masks and Row Filters

A mask replaces a column's value through a function, and a row filter hides rows through a boolean
function. Both reference a `function` the schema declares, and Atlas creates the function first:

```hcl
function "mask_pii" {
  schema = schema.default
  lang   = SQL
  arg "value" {
    type = STRING
  }
  arg "region" {
    type = STRING
  }
  return = STRING
  as     = "IF(is_account_group_member(region || '_HumanResourceDept'), value, 'REDACTED')"
}

function "region_filter" {
  schema = schema.default
  lang   = SQL
  arg "region" {
    type = STRING
  }
  return = BOOLEAN
  as     = "region = current_user()"
}

table "persons" {
  schema = schema.default
  column "address" {
    null = true
    type = STRING
    mask {
      function      = function.mask_pii
      using_columns = [column.region]
    }
  }
  column "region" {
    null = true
    type = STRING
  }
  row_filter {
    function = function.region_filter
    columns  = [column.region]
  }
}
```

- Atlas always qualifies the function with its schema in the DDL, as Databricks resolves an unqualified
  one against the session, not the table's schema:
  ``MASK `default`.`mask_pii` USING COLUMNS (`region`)``.
- Adding, changing, and removing a mask plans `ALTER COLUMN ... SET MASK` or `DROP MASK`. A row filter
  plans `SET ROW FILTER` or `DROP ROW FILTER`; changing its function or columns drops it and sets it
  again.
- The function's result depends on who runs the query: `is_account_group_member` and `current_user`
  read the session's principal. A migration test sees what the dev URL's principal sees; check the
  other principals by querying as them (`references/testing.md`).

## Errors

| Error | Cause | Fix |
|-------|-------|-----|
| `unknown attribute grantable in block of type permission` | The block asks for `WITH GRANT OPTION`, which Databricks does not have | Remove `grantable` |
| `no privileges specified for ACL on ...` | An empty `privileges` list | List at least one privilege |
| `unsupported external object kind "..."` | `ref()` names an object kind other than a schema, table, view, or materialized view | Grant on that object outside Atlas |
| The first plan revokes grants nobody asked to change | The mode is on, and the schema does not declare the target's existing grants | Inspect with the mode on, and copy the grants into the schema |
| Grants are missing from inspection | `mode { permissions = true }` is off | Turn it on in the env's `schema` block |
