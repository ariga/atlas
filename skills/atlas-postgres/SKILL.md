---
name: atlas-postgres
description: "PostgreSQL specifics for Atlas: dev database and URL scope (search_path), extensions, roles and grants, ALTER DEFAULT PRIVILEGES, ownership, row-level security, sequences and serial or identity columns, enums and column type changes, partitions, triggers, concurrent indexes, lock-safe migrations, schema and migration tests, custom lint rules, managed services (RDS, Cloud SQL, Supabase, Neon, Azure), and connection poolers. Use with the atlas skill whenever Atlas manages a PostgreSQL database."
---

# Atlas for PostgreSQL

What is specific to PostgreSQL when Atlas manages the schema. The `atlas` skill covers the workflows and
commands; this skill covers the rules, edge cases, and answers that differ on PostgreSQL.
PostgreSQL-compatible engines, such as CockroachDB, YugabyteDB, and Aurora DSQL, behave differently in
places: follow their guides (https://atlasgo.io/databases).

## Files

This skill is `SKILL.md` plus seven reference files under `references/`, each served at
`https://atlasgo.io/skills/atlas-postgres/references/<name>`. Read a reference when the task needs it:

| File | Read it for |
|------|-------------|
| `references/dev-database.md` | The dev URL and its scope, `atlas tool docker`, the `docker` block and `baseline`, extensions, RDS, Cloud SQL, Supabase, Neon, Azure, connection URLs |
| `references/security.md` | Roles and users, GRANT and REVOKE, `ALTER DEFAULT PRIVILEGES`, ownership, row-level security, provider admin roles |
| `references/objects.md` | Sequences, serial and identity columns, enums, domains, column type changes, functions, triggers, views, partitions, indexes, what Atlas does not manage |
| `references/hcl.md` | Writing the schema in HCL: the `NOT NULL` default, column types and `sql()`, defaults, references, keys, indexes, table settings, function bodies, seed data, input variables |
| `references/migrations.md` | Locks and concurrent indexes, `PG*` lint findings, transactions, connection poolers, lock-safe `NOT NULL`, seed data |
| `references/custom-rules.md` | Writing a custom lint rule in the HCL rule language when no analyzer covers a check: predicates, operators, what each PostgreSQL object exposes, examples |
| `references/testing.md` | Testing every change: schema tests for functions, triggers, views, domains, and policies, migration tests with seeded rows, row-level security tests, template databases, pgTAP |

## Requires

The `atlas` skill, installed next to this one (`../atlas/SKILL.md`). If it is missing, download
https://atlasgo.io/skills/atlas/SKILL.md to `../atlas/SKILL.md` and each reference file listed in its
Files section to `../atlas/references/<name>`.

## Rules

1. Check the login first with `atlas whoami`. If it shows no login, ask the user to run `atlas login`:
   logged out, Atlas misses most of a PostgreSQL schema. Inspection silently skips views, functions,
   triggers, sequences, extensions, partitions, and policies, and a desired state that defines them
   fails with an error containing `are available to logged-in users only`. A user without an account
   can start a free trial with `atlas login`.
2. Pick the URL scope from the objects the schema holds:
   - Schema scope, a URL with `search_path`: Atlas manages that one schema and writes unqualified
     statements (`users`). Database scope, no `search_path`: Atlas covers every schema and qualifies
     each statement (`public.users`).
   - Database-level objects belong to no schema: extensions, event triggers, and foreign servers. A
     schema-scoped URL never sees them and leaves them out of plans without an error, so a schema that
     declares them must use database scope.
   - Schema-level objects work at either scope: tables with their indexes, constraints, triggers, and
     policies, plus views, functions, procedures, sequences, and types.
   - Roles and users are cluster-level, shared by every database on the instance. Managing them needs
     database scope (Rule 5).
   - The dev URL takes the target's scope. At schema scope, a Docker dev URL uses `search_path=public`,
     which exists in every new database, whatever schema the target names; a dev database you provide
     can use any empty schema.
3. The dev database runs the target's major version and extensions: `docker://postgres/<version>/dev`.
   Create what the schema uses but does not manage, such as extensions, provider roles, or other
   teams' schemas, in a `docker` block `baseline`. Never point `dev` at a real environment's database.
4. `ALTER DEFAULT PRIVILEGES` in the desired state never appears in a migration: Atlas runs it on the dev
   database and plans one GRANT per object it covers. Put it before those objects, without `FOR ROLE`.
5. Managing roles needs database scope and `mode { roles = true }`. Mark every role Atlas must not change
   with `external = true`, and create the external roles that grants name in the dev `baseline`. On a
   managed service, exclude the provider's admin role
   (`references/security.md`, Managed Providers).
6. Build and drop indexes on populated tables concurrently: the `concurrent_index` diff policy, or a
   hand-written file that starts with `-- atlas:txmode none`. Never write `BEGIN` or `COMMIT` in a
   migration file: Atlas runs each file in a transaction unless it sets `txmode none`.
7. Add `CHECK` constraints and foreign keys `NOT VALID`, then validate them in a separate transaction:
   a `txmode none` file, or a later file. Use the lock-safe `NOT NULL` policy on large tables.
8. Apply migrations over a direct connection, never through a connection pooler in transaction mode,
   such as Supabase's transaction pooler (port 6543), Neon's `-pooler` endpoint, or PgBouncer in
   transaction mode. The advisory lock Atlas holds for the run is per session, and these poolers drop
   it; Supabase and Neon both document the direct connection for migrations.
9. Some changes need a hand edit to the generated migration (`references/objects.md`):
   - A conversion PostgreSQL cannot make implicitly, such as `text` to `integer` or `uuid`, needs a
     `USING` clause.
   - A populated column converted to `serial` needs a `setval`, or inserts fail on duplicate keys.
   - `serial` to `bigserial` needs `ALTER SEQUENCE ... AS bigint`, or the sequence still stops at
     2147483647.
   - A new enum value goes in one file and its first use in a later one. Removing or reordering values
     needs a hand-written migration, and its plan can report no changes.
10. Test every change before reporting it done, as a habit. Run `atlas schema validate`, add a
    `test "schema"` case for each function, trigger, view, domain, check, or policy you add or change,
    and a `test "migrate"` case with seeded rows for each migration that moves or converts data. Run
    them with `atlas schema test` and `atlas migrate test` (`references/testing.md`).
11. When no built-in analyzer covers a check, write a custom lint rule in the HCL rule language, and do
    the same for a convention the user states, such as row-level security on every table or an index on
    every foreign key. A rule can check any attribute of any object Atlas inspects, and lint enforces it
    on every change (`references/custom-rules.md`).

## Quick Answers

| Question | Answer |
|----------|--------|
| A local PostgreSQL database to apply migrations to | `atlas tool docker --url "docker://postgres/17/myapp?search_path=public" --name my-db` prints its URL; `atlas tool docker kill --name my-db` stops it (`references/dev-database.md`) |
| `cannot drop type ... because other objects depend on it`, or views and functions missing from a plan | Run `atlas whoami`: logged out, Atlas skips views and functions, including the ones that depend on the type (Rule 1) |
| Which URL to give Atlas on Supabase, Neon, or behind PgBouncer | The direct connection, never the transaction-mode pooler (`references/migrations.md`, Connection Poolers) |
| `pq: unsupported startup parameter: search_path` | The URL goes through PgBouncer; use the direct connection (`references/migrations.md`, Connection Poolers) |
| `CREATE EXTENSION` is missing from the migration | The URLs are schema-scoped; extensions need database scope (`references/dev-database.md`) |
| A plan drops the `public` schema | At database scope, the desired state covers every schema: declare `schema "public" {}`, or use schema-scoped URLs (`references/dev-database.md`) |
| `cannot diff a schema with a database connection` | The URL and the dev URL have different scopes (`references/dev-database.md`) |
| `connected database is not clean` | `dev` points at a database that is not empty (`references/dev-database.md`) |
| The same change shows up in every plan | Set a dev URL with the target's scope: Atlas normalizes defaults and expressions on it (`references/dev-database.md`) |
| A column became `NOT NULL` without asking for it | HCL columns are `NOT NULL` unless they set `null = true` (`references/hcl.md`) |
| `ALTER COLUMN ... TYPE` fails for `text` to `integer` or `uuid` | Add a `USING` clause to the migration file (`references/objects.md`, Column Changes) |
| Inserts fail with a duplicate key after a column became `serial` | Move the sequence with `setval` (`references/objects.md`) |
| An enum value was removed, but the diff reports no changes | Removing enum values is not planned: write the change by hand (`references/objects.md`, Enums) |
| `unsafe use of new value` | Add the enum value and its first use in separate migration files (`references/objects.md`, Enums) |
| `pq: type "geometry" does not exist` | Create the extension in the schema or the dev `baseline`; the PostGIS image alone does not install it (`references/dev-database.md`) |
| Lint reports `PG101` or `PG103` on an index | Use the `concurrent_index` policy or `atlas:txmode none` (`references/migrations.md`) |
| A convention must hold on every table, such as row-level security or foreign-key indexes | Write a custom lint rule (`references/custom-rules.md`) |
| A row-level security test passes whatever the policy says | The dev database connects as a superuser, which bypasses policies; test as a non-privileged role (`references/testing.md`) |
| `pq: unexpected transaction status idle` | Remove `BEGIN`/`COMMIT` from the file: Atlas already runs it in a transaction (`references/migrations.md`) |
| `ambiguous revision table` after changing the URL's scope | Set `--revisions-schema` to the schema that holds the existing table (`references/migrations.md`) |
| `DROP and CREATE is required` on a partitioned table | Changing a partition key, or partitioning an existing table, needs a hand-written migration (`references/objects.md`) |
| `ERROR: out of shared memory` on the dev database | Raise `max_locks_per_transaction` above 128 in the `docker` block (`references/dev-database.md`) |
| An event trigger is missing from the migration | Use a database-scoped dev URL (`references/objects.md`) |
| Every plan revokes from `postgres` (Cloud SQL), the RDS master user, or another provider admin | Plan through a dev database where the admin is an ordinary role (`references/security.md`, Managed Providers) |
| A new role needs `SELECT` on thousands of tables | Make it a member of `pg_read_all_data` instead of granting per table (`references/security.md`) |
| Objects on the dev database belong to `postgres`, but production has other owners | Owners are not part of the desired state (`references/security.md`, Ownership) |
| The migration has no `ALTER DEFAULT PRIVILEGES` | Expected: Atlas plans the GRANTs it implies, per object (`references/security.md`) |
| A grant on a sequence turned a `bigserial` column into `bigint` | Expected: the sequence is now managed explicitly (`references/objects.md`) |

## Documentation

- PostgreSQL HCL reference: https://atlasgo.io/hcl/postgres
- Dev database: https://atlasgo.io/concepts/dev-database
- Security: https://atlasgo.io/guides/postgres/security-declarative,
  https://atlasgo.io/guides/postgres/security-default-privileges
- Lock-safe changes: https://atlasgo.io/guides/lock-safe-not-null, https://atlasgo.io/lint/analyzers
- Guides for each database: https://atlasgo.io/databases
