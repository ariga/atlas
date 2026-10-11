---
name: atlas-redshift
description: "Redshift specifics for Atlas: the login-only driver, redshift:// and Data API (redshift+http://) URLs and their scope, the dev database without Docker, distribution, sort keys, and encodings, identity columns, column changes Redshift cannot ALTER and the allow_recreate policy, views and materialized views, functions and procedures, users, groups, roles and grants, datashares, external schemas, statements that cannot run in a transaction, no advisory lock, schema and migration tests, and custom lint rules. Use with the atlas skill whenever Atlas manages a Redshift database."
---

# Atlas for Redshift

What is specific to Amazon Redshift when Atlas manages the schema. The `atlas` skill covers the
workflows and commands; this skill covers the rules, edge cases, and answers that differ on Redshift.
Redshift speaks the PostgreSQL wire protocol, but it is a separate Atlas driver with its own HCL and
its own limits: do not apply PostgreSQL advice (indexes, sequences, extensions, triggers, `docker://`
dev URLs) to it.

## Files

This skill is `SKILL.md` plus seven reference files under `references/`, each served at
`https://atlasgo.io/skills/atlas-redshift/references/<name>`. Read a reference when the task needs it:

| File | Read it for |
|------|-------------|
| `references/dev-database.md` | The dev URL and its scope, the dev database without Docker, the `dev` block and `baseline`, `redshift://` and Data API URLs, Serverless and provisioned clusters, connection errors |
| `references/security.md` | Users, groups, roles, memberships, GRANT and REVOKE, `ALL` expansion, `PUBLIC` on functions, external and IAM principals, passwords |
| `references/objects.md` | What needs a login, distribution, sort keys and encodings, identity columns, column changes and rebuilds, views, routines, external schemas, datashares, what Atlas does not manage |
| `references/hcl.md` | Writing the schema in HCL: the `NOT NULL` default, column types, defaults, `identity`, `encode`, `distribution`, `sort`, views, routines, `external`, `datashare`, roles and grants |
| `references/migrations.md` | Lint checks on Redshift, statements that run without a transaction, the Data API, no advisory lock, the revisions table, plans Atlas refuses |
| `references/testing.md` | Testing every change: schema tests for functions, procedures, and views, migration tests with seeded rows, the cost of a real dev database |
| `references/custom-rules.md` | Writing a custom lint rule in the HCL rule language when no analyzer covers a check: predicates, operators, what each Redshift object exposes, examples |

## Requires

The `atlas` skill, installed next to this one (`../atlas/SKILL.md`). If it is missing, download
https://atlasgo.io/skills/atlas/SKILL.md to `../atlas/SKILL.md` and each reference file listed in its
Files section to `../atlas/references/<name>`.

## Rules

1. Check the login first with `atlas whoami`. If it shows no login, ask the user to run `atlas login`:
   logged out, every Redshift command fails with `login is required to use the 'redshift' driver`. A
   user without an account can start a free trial with `atlas login`.
2. Pick the URL scope from what the schema holds:
   - Schema scope, a `redshift://` URL with `search_path`: Atlas manages that one schema and writes
     unqualified statements (`"users"`). Database scope, no `search_path`: Atlas covers every schema
     in the database and qualifies each statement (`"public"."users"`).
   - Data API URLs (`redshift+http://`) ignore `search_path`: they are schema-scoped to the user's
     current schema by default, and `mode=database` gives database scope.
   - Users, groups, roles, datashares, and several schemas need database scope. At database scope, a
     schema the desired state does not declare is planned for `DROP SCHEMA ... CASCADE`.
   - The dev URL takes the target's scope and URL form. At schema scope, its `search_path` names an
     empty schema with the target schema's name. A dev schema of another name, such as `public`, puts
     `COMMENT ON SCHEMA "<name>" IS ''` into the first migration file, and every later replay fails
     with `schema "<name>" does not exist` (`references/dev-database.md`).
3. Redshift has no Docker image: `docker://redshift/...` fails. The dev database is a real, empty
   database you provide. A second database on the target's cluster works for schema objects. A
   migration directory that creates users, groups, roles, or datashares replays them on the dev
   database's cluster, where their names are cluster-wide: give such projects a dev database on its own
   cluster or Serverless workgroup (`references/dev-database.md`). Never point `dev` at a real
   environment's database.
4. Always plan with a dev URL. Redshift picks column encodings, distribution and sort styles, and type
   sizes (`varchar` becomes `character_varying(256)`); without the dev database the desired state does
   not match what inspection reads back, and every plan repeats changes.
5. Redshift fixes a column's nullability, default, and identity when the column is created, and alters
   a type only to change a `varchar` length. Atlas refuses the other changes with
   `... requires recreating the column`. The `modify_column { allow_recreate = true }` diff policy
   plans a drop and an add, which loses the column's data: for data you keep, write the migration by
   hand (`references/objects.md`, Column Changes).
6. A new `NOT NULL` column on an existing table needs a `default`, or `null = true`. An identity column
   exists only from `CREATE TABLE`: adding one to an existing table needs a hand-written rebuild
   (`references/objects.md`, Identity Columns).
7. Never write `BEGIN` or `COMMIT` in a migration file. Some statements cannot run in a transaction,
   such as `ALTER COLUMN ... TYPE`, `VACUUM`, `COPY`, external table DDL, and
   `GRANT USAGE ON DATASHARE`. Atlas adds `-- atlas:txmode none` to the files it generates with them;
   a hand-written file needs the directive itself, and stays half applied if it fails midway, so keep
   such files short (`references/migrations.md`).
8. Redshift has no advisory locks, so Atlas takes no lock while applying: two `migrate apply` or
   `schema apply` runs against one database are not stopped from running at the same time. Serialize
   deployments in CI.
9. With `roles = true`, Atlas inspects every principal on the cluster. With a dev database on its own
   cluster, an undeclared principal is planned for a drop: scope the env with `include` patterns such
   as `"app_*[type=role]"`, and mark the admin and other teams' principals `external = true`. With a
   dev database on the target's cluster, the desired state picks up every principal the cluster holds,
   so a principal removed from the schema is never dropped: drop it by hand. Inspection skips the user Atlas
   connects as: declaring it without `external = true` plans a `CREATE USER` that fails on apply
   (`references/security.md`).
10. Primary keys, foreign keys, unique and check constraints, indexes, sequences, and triggers are not
    part of the Redshift schema in Atlas. Inspection skips them, and the HCL `table` block has no
    blocks for them. Do not add them to HCL.
11. Test every change before reporting it done, as a habit. Run `atlas schema validate`, add a
    `test "schema"` case for each function, procedure, or view you add or change, and a
    `test "migrate"` case with seeded rows for each migration that rebuilds a column, changes a type,
    or moves data. Run them with `atlas schema test` and `atlas migrate test` (`references/testing.md`).
12. When no built-in analyzer covers a check, write a custom lint rule in the HCL rule language, and do
    the same for a convention the user states, such as an explicit encoding on every column or a
    distribution key on large tables. A rule can check any attribute of any object Atlas inspects, and
    lint enforces it on every change (`references/custom-rules.md`).

## Quick Answers

| Question | Answer |
|----------|--------|
| `login is required to use the 'redshift' driver` | Ask the user to run `atlas login` (Rule 1) |
| `unsupported docker image "redshift"` | Redshift has no Docker image: give Atlas an empty database on a cluster or workgroup (`references/dev-database.md`, Your Own Dev Database) |
| Which URL for Serverless, a provisioned cluster, or a cluster without network access | `redshift://` over the endpoint, or `redshift+http://workgroup(name)/db` and `redshift+http://cluster(name)/db` through the Data API (`references/dev-database.md`, Connection URLs) |
| `invalid host: ...` or `invalid authorization source: ...` on a Data API URL | The host must be `workgroup(<name>)` or `cluster(<id>)` (`references/dev-database.md`, Connection URLs) |
| `sql/redshift: connection mode must be schema or database, received "..."` | The `mode` parameter of a Data API URL takes `schema` or `database` (`references/dev-database.md`) |
| `connected database is not clean: ...` | The dev database holds objects in its scope: use an empty, dedicated one (`references/dev-database.md`) |
| `cannot diff a schema "..." with a database connection` | The URL and the dev URL have different scopes (`references/dev-database.md`) |
| The same encoding, `varchar(256)`, or `DISTSTYLE AUTO` change shows up in every plan | Plan with a dev URL, or declare what inspection reads back (`references/hcl.md`, Column Types) |
| `Column "c1" default change requires recreating the column.` | Redshift cannot alter a default or nullability: rebuild with `allow_recreate`, or migrate the data by hand (`references/objects.md`, Column Changes) |
| `... type change requires recreating the column` | The column has a default or a `BYTEDICT`, `RUNLENGTH`, `TEXT255`, or `TEXT32K` encoding (`references/objects.md`, Column Changes) |
| `..., but a non-nullable column requires a default value in Redshift`, or `..., but it is the distribution key of table "t"` | The column cannot be rebuilt: add a default, or change the table's `distribution` or `sort` in an earlier migration (`references/objects.md`, Column Changes) |
| `pq: cannot alter column "c" of relation "t" (0A000)` on apply, such as for `integer` to `bigint` | Redshift alters only `varchar` lengths, but Atlas plans any type change: add a new column, copy, drop, rename (`references/objects.md`, Column Changes) |
| `pq: ALTER TABLE ALTER COLUMN cannot run inside a transaction block (25001)` | A hand-written file without `-- atlas:txmode none`: add it, followed by an empty line, and run `atlas migrate hash` (`references/migrations.md`, Transactions) |
| `adding non-nullable column "c" requires a default value in Redshift` | Give the new column a `default`, or `null = true` (Rule 6) |
| `adding identity column "c" is not supported after the table is created` | Identity columns exist only in `CREATE TABLE` (`references/objects.md`, Identity Columns) |
| `modify column identity is not supported in Redshift` | An identity's seed, step, or generation cannot change, nor can a column gain or lose one (`references/objects.md`, Identity Columns) |
| `modify sort key to INTERLEAVED is not supported in Redshift` | Recreate the table with the interleaved key in a hand-written migration (`references/objects.md`) |
| `unknown block type primary_key` (or `index`, `foreign_key`) | Not modeled on Redshift (Rule 10) |
| `cannot create table "t1" in external schema "..."` | An external schema's tables come from its source (`references/objects.md`, External Schemas) |
| An external schema is dropped and recreated in every plan | Atlas reads an external schema owned by another user as a local one: run as its owner (`references/objects.md`, External Schemas) |
| A failed file left some statements applied | It ran under `-- atlas:txmode none`: fix the rest of the file, run `atlas migrate hash`, and re-apply (`references/migrations.md`, Transactions) |
| A `SET` or a temporary table is gone in the next statement, through the Data API | Each statement outside a transaction runs in its own session (`references/migrations.md`, The Data API) |
| A plan drops users, roles, or groups no one declared | Scope `roles` with `include`, and mark the rest `external = true` (`references/security.md`) |
| A user or role removed from the schema is never dropped | The dev database shares the target's cluster, so the desired state holds every principal: drop it by hand, or use a dev database on its own cluster (`references/security.md`) |
| `pq: schema "..." does not exist` when replaying migrations on the dev database | The dev URL's `search_path` names another schema than the target's: create an empty schema of the same name in the dev database (Rule 2) |
| `execute baseline: pq: ... already exists` on the second run | Atlas does not remove what a `baseline` created: write it with `IF NOT EXISTS` (`references/dev-database.md`, The `dev` Block and `baseline`) |
| Every plan creates the user Atlas connects as (`CREATE USER ... PASSWORD DISABLE`) | Inspection skips the connecting user: declare it `external = true`, or leave it out (`references/security.md`) |
| `permission.for reference to undeclared function ... must carry the exact signature Redshift grants by` | Write the argument types into the reference: `ref("function.s.f(integer)")` (`references/security.md`, Permissions) |
| A grant on one overload of a function shows up in every plan | Redshift reports routine grants by name: grant every overload the same (`references/security.md`, Permissions) |
| Every new function gets `REVOKE EXECUTE ... FROM PUBLIC` | With `permissions = true`, declare the `PUBLIC` grant to keep it (`references/security.md`) |
| `redshift: the grant option cannot be granted to role "..."` | `grantable = true` works only for a user grantee (`references/security.md`) |
| `redshift: only a user can be a member of group "..."` | Groups take users only; use a role for a role hierarchy (`references/security.md`) |
| A datashare is missing from inspection | Enable `mode "redshift" { datashares = true }` (`references/objects.md`, Datashares) |
| A Python UDF is missing from inspection | Atlas does not inspect `plpythonu` functions (`references/objects.md`, Not Managed by Atlas) |
| `ambiguous revision table` after changing the URL's scope | Set `--revisions-schema` to the schema that holds the existing table (`references/migrations.md`) |

## Documentation

- Redshift HCL reference: https://atlasgo.io/hcl/redshift
- Getting started: https://atlasgo.io/guides/redshift/automatic-migrations
- Dev database: https://atlasgo.io/concepts/dev-database
- Security: https://atlasgo.io/guides/redshift/security-declarative,
  https://atlasgo.io/guides/redshift/security-versioned
- Lint checks: https://atlasgo.io/lint/analyzers
- Guides for each database: https://atlasgo.io/databases
