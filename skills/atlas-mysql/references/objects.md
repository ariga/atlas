# MySQL Objects and Their Nuances

How Atlas models MySQL and MariaDB objects, and the cases where its plan differs from what a
hand-written migration would do. The full attribute reference is https://atlasgo.io/hcl/mysql, with
examples in https://atlasgo.io/atlas-schema/hcl. HCL syntax is in `references/hcl.md`, and the lint
checks for each change are in `references/migrations.md`.

## What Needs a Login

Tables and columns, indexes, foreign keys, check and unique constraints, and comments work without a
login. Partitions, views, full-text indexes, triggers, stored procedures, and functions need
`atlas login`, and so do users, roles, and grants (https://atlasgo.io/features). On MariaDB, temporal
(system-versioned) tables need it too.

If `atlas whoami` shows no login, ask the user to run `atlas login` before going further: logged out,
Atlas misses these objects. A user without an account can start a free trial with `atlas login`.

## Enums and Sets

- `enum` and `set` are column types with their values inline, such as `type = enum("a", "b")`; there is
  no `enum` block on MySQL.
- Add new values at the end of the list. Removing, reordering, or inserting values in the middle makes
  MySQL copy the table and block writes (`MY110` to `MY112`, `MY120` to `MY122`).
- Removing a value that rows still hold fails under strict `sql_mode`, the MySQL 8 default. Without
  strict mode, those rows silently become `''`. Update the rows first, in an earlier migration, and
  run the dev database with the target's `sql_mode` (`references/dev-database.md`).
- An enum reaching 256 values, or a set passing 8, 16, 24, or 32 values, changes the storage size,
  which copies the table (`MY113`, `MY123`). A set holds at most 64 values.

## Keys and Indexes

- MySQL needs an index on a foreign key's columns, and creates one when none exists. Inspection shows it
  as an `index` block named after the foreign key. A plan made without a dev database can try to drop
  it, and MySQL refuses with
  `Error 1553 (HY000): Cannot drop index '<name>': needed in a foreign key constraint`.
  Plan with a dev URL, or declare the index in the schema.
- A foreign key that references a table in another database needs both databases inspected: use server
  scope (`references/dev-database.md`).
- MyISAM tables have no foreign keys: MySQL parses a `FOREIGN KEY` clause on a MyISAM table and ignores
  it, so neither inspection nor a plan ever shows it. Tables with foreign keys use InnoDB
  (`engine = InnoDB`).
- A constraint the schema leaves unnamed, as many ORMs do, gets its name from the server, such as
  `orders_ibfk_1`. The dev database can pick a different name than the target, and the plan then
  renames the constraint. Skip those renames with the `diff` policy
  (https://atlasgo.io/faq/skip-constraint-rename):

  ```hcl
  env "local" {
    diff {
      skip {
        rename_constraint = true
      }
    }
  }
  ```
- A column added with an inline `REFERENCES` clause creates no foreign key on MySQL (`MY102`); declare a
  `foreign_key` block.
- Foreign key and check constraint names are unique per database, not per table. A name reused on
  another table, as ORMs often generate, fails with `Error 1826` for a foreign key or `Error 3822` for a
  check. Give each constraint a name unique in its database.
- Identifiers are limited to 64 characters. A generated index or constraint name over the limit fails
  with `Error 1059 (42000): Identifier name '...' is too long`; name it explicitly.
- Functional index expressions, prefix lengths on `BLOB` and `TEXT` columns, and descending parts are
  in `references/hcl.md`, Keys, Constraints, and Indexes.

## Checks and Generated Columns

- MySQL enforces checks from 8.0.16; earlier versions parse and ignore them. `enforced = false` keeps a
  check `NOT ENFORCED` on MySQL; MariaDB has no such state.
- Generated columns are `VIRTUAL` by default on MySQL. A generated column in a primary key or foreign
  key must be `STORED`. Adding a `STORED` column, or changing any generated column, copies the table
  (`MY140`, `MY143`).

## Functions, Procedures, and Triggers

- With binary logging on, MySQL refuses a function that declares none of `DETERMINISTIC`, `NO SQL`, or
  `READS SQL DATA`:

  ```
  Error 1418 (HY000): This function has none of DETERMINISTIC, NO SQL, or READS SQL DATA in its declaration and binary logging is enabled
  ```

  Set `deterministic = true` or `data_access` in the function, or enable
  `log_bin_trust_function_creators` on the server. The dev database's `init` example in
  `references/dev-database.md` enables it.
- A migration file that creates a stored program redefines the statement delimiter
  (`references/migrations.md`, Transactions).
- A view, trigger, or routine created with `DEFINER = 'user'@'host'` runs with that account's
  privileges. MySQL accepts an account that does not exist, but the object fails when it runs:
  `Error 1449 (HY000): The user specified as a definer ('user'@'host') does not exist`. Create the
  account in the dev database's `baseline` when tests use the object.
- On RDS and other managed services, the admin account lacks `SUPER`, so a `DEFINER` naming another
  account fails with `Error 1227 (42000): Access denied`, which asks for `SUPER` or `SET_USER_ID`.
  Migration files written from a dump carry such clauses: remove them, so the objects belong to the
  account that runs the migration.
- Triggers on the same table and event run in the order `follows` and `precedes` set.

## Normalized Definitions

The server rewrites expressions when it stores them: generated columns, expression defaults, functional
indexes, checks, and views. A functional index on `upper(concat('c', c))` inspects as
``upper(concat(_utf8mb4'c',`c`))``, so a plan made without a dev database drops and re-adds the index
on every run (https://atlasgo.io/concepts/dev-database). Plan with a dev URL, on a server with the
target's character set (`references/dev-database.md`, Character Sets and Collations).

## Objects That Reference Other Databases

A schema-scoped dev database holds one database. A view that selects from another database fails to
load there with `Error 1146 (42S02): Table '<database>.<table>' doesn't exist`. Create the referenced
tables in the dev database's `baseline`, or use server scope with both databases
(`references/dev-database.md`). MySQL checks a trigger's or a routine's references only when it runs,
so those load, and fail later in tests.

## Not Modeled

- Events (`CREATE EVENT`) have no block in the schema language.
- Atlas does not model MariaDB sequences.
- Default roles (`SET DEFAULT ROLE`) have no attribute; see `references/security.md`.

## MariaDB

MariaDB has its own driver, with `maria://` URLs, `docker://maria/...` dev URLs, `docker "mariadb"`
blocks, and `.ma.hcl` files (https://atlasgo.io/hcl/mariadb). Run a MariaDB dev database for a MariaDB
target: a MySQL dev database renders collations and defaults differently. Where it differs from MySQL:

- `system_versioned = true` makes a table system-versioned. Dropping it deletes the history (`MY146`).
- Checks have no `NOT ENFORCED` state, so `MY144` has no lock-free fix on MariaDB.
- `PERSISTENT` generated columns are added in place but still rebuild the table.
- An instant column add works only at the end of the table on MariaDB 10.3, and anywhere from 10.4
  (`MY142`).
- A column change that touches only the collation is instant from MariaDB 10.4.3, and `MY148` does not
  report it.
- `JSON` is stored as `LONGTEXT` with a `json_valid` check, which shows up as drift unless the plan runs
  on a MariaDB dev database.
- The native `uuid` and `inet6` types have no HCL type; write them with `sql("...")`.
