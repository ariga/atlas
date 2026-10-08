---
name: atlas-mysql
description: "MySQL and MariaDB specifics for Atlas: dev database and URL scope (schema vs server), character sets and collations, users, roles and grants, enum and set columns, generated columns, foreign-key indexes, functional and prefix indexes, triggers and stored programs, table copies and MY lint checks, partial migration failures (no transactional DDL), GET_LOCK, managed services (RDS, Aurora, Cloud SQL, Azure), schema and migration tests, and custom lint rules. Use with the atlas skill whenever Atlas manages a MySQL or MariaDB database."
---

# Atlas for MySQL

What is specific to MySQL and MariaDB when Atlas manages the schema. The `atlas` skill covers the
workflows and commands; this skill covers the rules, edge cases, and answers that differ on MySQL.
MariaDB is covered where it differs. MySQL-compatible engines, such as TiDB, PlanetScale, and
SingleStore, are not covered.

## Files

This skill is `SKILL.md` plus seven reference files under `references/`, each served at
`https://atlasgo.io/skills/atlas-mysql/references/<name>`. Read a reference when the task needs it:

| File | Read it for |
|------|-------------|
| `references/dev-database.md` | The dev URL and its scope, `atlas tool docker`, the `docker` block, character sets and collations, RDS, Cloud SQL, Azure, MariaDB, connection URLs |
| `references/security.md` | Users and roles, hosts, GRANT and REVOKE, passwords, role activation |
| `references/objects.md` | What needs a login, enums and sets, foreign-key indexes, checks, generated columns, functions and triggers, normalized definitions, MariaDB |
| `references/hcl.md` | Writing the schema in HCL: the `NOT NULL` default, defaults and `ON UPDATE`, `AUTO_INCREMENT`, types, indexes, engines, partitions, views, routines, seed data |
| `references/migrations.md` | `MY*` lint findings, table copies, partial failures and recovery, delimiters, locks, the revisions table, baselines |
| `references/testing.md` | Testing every change: schema tests, migration tests with seeded rows, CI |
| `references/custom-rules.md` | Writing a custom lint rule in the HCL rule language when no analyzer covers a check: predicates, operators, what each MySQL object exposes, examples |

## Requires

The `atlas` skill, installed next to this one (`../atlas/SKILL.md`). If it is missing, download
https://atlasgo.io/skills/atlas/SKILL.md to `../atlas/SKILL.md` and each reference file listed in its
Files section to `../atlas/references/<name>`.

## Rules

1. Check the login first with `atlas whoami`. If it shows no login, ask the user to run `atlas login`:
   logged out, Atlas misses views, triggers, stored procedures, functions, partitions, and users and
   grants. A user without an account can start a free trial with `atlas login`.
2. Pick the URL scope from what the schema holds:
   - In MySQL, a schema is a database. Schema scope, a URL with a database in its path: Atlas manages
     that database and writes unqualified statements (`users`). Server scope, no database in the path:
     Atlas covers every database and qualifies each statement (`app.users`).
   - Users and roles belong to the server, so managing them needs server scope. So do several
     databases, and foreign keys across databases.
   - At server scope, a database the desired state does not declare is planned for `DROP DATABASE`.
   - The dev URL takes the target's scope: `docker://mysql/8/dev`, or `docker://mysql/8`.
3. The dev database runs the target's version and settings: character set and collation,
   `lower_case_table_names`, `sql_mode`, and `explicit_defaults_for_timestamp`. A `docker` block pins
   the image (`image = "mysql:8.4"`) and starts the server with the target's settings (`command`). Use
   a MariaDB dev database for MariaDB. On RDS, migrations that create IAM users
   (`IDENTIFIED WITH AWSAuthenticationPlugin`) need a dev image with a mock of the plugin
   (`references/dev-database.md`). Never point `dev` at a real environment's database.
4. Declare `charset` and `collate` on every `schema` block. Otherwise the desired state takes the dev
   server's defaults, `utf8mb4_0900_ai_ci` on MySQL 8, and a schema-scoped plan against a target with
   other defaults fails with a `modify schema` error (`references/dev-database.md`).
5. Managing users and roles needs server scope and `mode { roles = true }`. MySQL has no `external`
   attribute, so an undeclared account is planned for a drop, including the user Atlas connects as:
   declare it, review the first plan for other accounts, and set `drop_user` and `drop_role` in the
   `skip` diff policy to keep drops out of plans. Granted roles take effect only after
   `SET DEFAULT ROLE` (`references/security.md`).
6. MySQL and MariaDB have no transactional DDL: every DDL statement commits on its own, so a migration
   file or a plan that fails halfway stays half applied, and `--tx-mode` cannot prevent it. Keep files
   small and test them before applying (Rule 9). To recover, fix the failed statement and the ones
   after it and run `atlas migrate hash`, or undo the applied statements with `atlas migrate down`
   (`references/migrations.md`).
7. Review every `MY1*` lint finding before applying to a large table: those changes copy or rebuild the
   table, or block writes. Add enum and set values at the end, prefer `VIRTUAL` generated columns, and
   check `engine` edits (`references/migrations.md`).
8. Always plan with a dev URL. Without one, Atlas compares definitions MySQL rewrote, such as `_utf8mb4`
   introducers, and can plan to drop the index MySQL created for a foreign key
   (`references/objects.md`).
9. Test every change before reporting it done, as a habit. Run `atlas schema validate`, add a
   `test "schema"` case for each function, procedure, trigger, view, generated column, or check you add
   or change, and a `test "migrate"` case with seeded rows for each migration that moves or converts
   data. Run them with `atlas schema test` and `atlas migrate test` (`references/testing.md`).
10. When no built-in analyzer covers a check, write a custom lint rule in the HCL rule language, and do
    the same for a convention the user states, such as an allowed character set or deterministic
    functions. A rule can check any attribute of any object Atlas inspects, and lint enforces it on
    every change (`references/custom-rules.md`).

## Quick Answers

| Question | Answer |
|----------|--------|
| A local MySQL database to apply migrations to | `atlas tool docker --url "docker://mysql/8/myapp" --name my-db` prints its URL; `atlas tool docker kill --name my-db` stops it (`references/dev-database.md`) |
| `modify schema "..." is not allowed when migration plan is scoped to one schema` | The dev database's charset, collation, or comment differ from the target's (`references/dev-database.md`) |
| Every table shows a charset or collation change | The dev server's defaults leaked into the desired state: declare them on the schema, or match the dev server to the target (`references/dev-database.md`) |
| Drift that differs only by `_latin1` or `_utf8mb4` | Start the dev server with the target's character set through `command` (`references/dev-database.md`) |
| Table names keep showing as changed | The dev server's `lower_case_table_names` differs from the target's (`references/dev-database.md`, Server Versions and Settings) |
| A plan drops other databases (`DROP DATABASE`) | Server scope covers every database: limit the run with `--schema`, or use schema scope (`references/dev-database.md`) |
| `Error 1046: No database selected` | A schema-scoped plan applied through a server-scoped URL (`references/dev-database.md`) |
| `cannot use HCL with more than 1 schema when dev-url is limited to schema` | Use a server-scoped dev URL, `docker://mysql/8` (`references/dev-database.md`) |
| A migration failed halfway, and `migrate status` shows `(last one partially)` | MySQL kept the statements before the failure: fix the rest and run `atlas migrate hash`, or undo them with `atlas migrate down` (`references/migrations.md`) |
| A rename shows up as a drop and an add | Declare it with `renamed_from`, or the data is lost (`references/hcl.md`) |
| Lint reports `MY1*` checks | The change copies or rebuilds the table, or blocks writes. To avoid the copy, force `ALGORITHM=INSTANT` in a hand-written file, or use an online schema change tool (`references/migrations.md`, Table Copies) |
| `Error 1553 (HY000): Cannot drop index ...: needed in a foreign key constraint` | Plan with a dev URL, or declare the foreign key's index (`references/objects.md`) |
| Foreign keys never show up in inspection or plans | The table uses MyISAM, which ignores foreign keys: use InnoDB (`references/objects.md`) |
| `Error 1418 (HY000): This function has none of DETERMINISTIC, NO SQL, or READS SQL DATA ...` | Set `deterministic` or `data_access`, or enable `log_bin_trust_function_creators` (`references/objects.md`) |
| Removing an enum value fails, or turns rows into `''` | Strict `sql_mode` fails on rows that hold the value, and non-strict mode blanks them: update those rows first (`references/objects.md`) |
| `Error 1826` or `Error 3822`, a duplicate constraint name | Foreign key and check names are unique per database, not per table (`references/objects.md`) |
| `Error 1059 (42000): Identifier name ... is too long` | Names are limited to 64 characters: name the index or constraint explicitly (`references/objects.md`) |
| `Error 1227 (42000): Access denied` on a `DEFINER` clause, on RDS | Remove `DEFINER` clauses from the migration files (`references/objects.md`) |
| `Error 1146 (42S02): Table ... doesn't exist` when loading a view | The view reads another database: create it in the dev `baseline`, or use server scope (`references/objects.md`) |
| `IDENTIFIED WITH mysql_native_password` fails on the dev database | MySQL 8.4 disables the plugin and 9.0 removes it: run the dev database on the target's version (`references/dev-database.md`) |
| `Error 1067: Invalid default value` on a timestamp | Write the default as `sql("current_timestamp(6)")`, with the column's precision (`references/hcl.md`) |
| A column became `NOT NULL` without asking for it | HCL columns are `NOT NULL` unless they set `null = true` (`references/hcl.md`) |
| `ERROR 1170 (42000): BLOB/TEXT column ... used in key specification without a key length` | Give the index part a `prefix` (`references/hcl.md`) |
| Users or roles the schema does not declare are planned for a drop | Declare them, or skip drops with `drop_user` and `drop_role` (`references/security.md`) |
| Granted roles have no effect | On MySQL, run `SET DEFAULT ROLE ALL TO 'user'@'host'`, or enable `activate_all_roles_on_login`. On MariaDB, a user has one default role: `SET DEFAULT ROLE <role> FOR 'user'@'host'` (`references/security.md`) |
| `Error 1524 (HY000): Plugin 'AWSAuthenticationPlugin' is not loaded`, from migrations that create RDS IAM users | Build the dev image with a mock of the plugin, and start local databases from it with `atlas tool docker --env <env>` (`references/dev-database.md`, RDS IAM users on the dev database) |
| `invalid port ":..." after host` | Percent-encode the password, or build the URL with `urlescape` (`references/dev-database.md`) |
| `server does not allow insecure connections, client must use SSL/TLS` | Add `?tls=true` to the URL (`references/dev-database.md`) |
| Databases on one server never apply in parallel | They share the default lock name: set `lock_name` per target (`references/migrations.md`) |

## Documentation

- MySQL HCL reference: https://atlasgo.io/hcl/mysql, and for MariaDB: https://atlasgo.io/hcl/mariadb
- Dev database: https://atlasgo.io/concepts/dev-database
- Security: https://atlasgo.io/guides/mysql/security-declarative,
  https://atlasgo.io/guides/mysql/security-versioned
- Lint checks: https://atlasgo.io/lint/analyzers
- Guides for each database: https://atlasgo.io/databases
