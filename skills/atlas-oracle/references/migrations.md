# Oracle Migrations: Transactions, Locks, and Migration Files

How Atlas runs migrations on a database that commits DDL as it goes, how to write PL/SQL in migration
files, and how to change a busy table without blocking it. Tests are in `references/testing.md`. The
Oracle driver, lint, custom rules, and hooks need `atlas login`. If `atlas whoami` shows no login, ask
the user to run `atlas login`; a user without an account can start a free trial with it.

## What Lint Catches on Oracle

Oracle has no analyzers of its own, like the `MY` and `PG` checks of MySQL and PostgreSQL. The generic
ones run (https://atlasgo.io/lint/analyzers):

| Code | Reports |
|------|---------|
| `DS102`, `DS103` | Dropping a table, or a column that holds data |
| `BC101` to `BC104` | Renaming or dropping a table or column that clients use |
| `CD101` to `CD103` | Dropping a foreign key, check, or primary key |
| `MF101` | A unique index on existing data, which fails on duplicates |
| `MF103` | A `NOT NULL` column without a default added to an existing table |
| `MF104` | A nullable column made `NOT NULL`, which fails on `NULL` rows |
| `TX101` | A file with more than one DDL statement, which cannot be rolled back as a unit |

Lint does not report what blocks a busy Oracle table, such as a `CREATE INDEX` without `ONLINE` or a
foreign key added with validation, nor PL/SQL that does not compile. Cover those with:

- A `statement` rule that flags indexes built without `ONLINE`
  (https://atlasgo.io/lint/analyzers#statement-rules). Rules run in order, and the first match wins:

  ```hcl
  lint {
    statement {
      allow "online-index" {
        match = "(?is)^\\s*CREATE\\s.*\\bINDEX\\b.*\\bONLINE\\b"
      }
      deny "blocking-index" {
        match   = "(?is)^\\s*CREATE\\s+(UNIQUE\\s+|BITMAP\\s+)?INDEX\\b"
        message = "Build indexes ONLINE: a plain CREATE INDEX blocks writes to the table"
      }
    }
  }
  ```

  It also reports indexes on tables the same file creates; skip those with
  `-- atlas:nolint statement`. Set `error = true` in the `statement` block to fail the run.
- Custom rules for conventions, such as an index on every foreign key (`references/custom-rules.md`).
- A migration test that checks for invalid objects (`references/testing.md`).

A statement opts out of named checks with a directive on the line before it
(https://atlasgo.io/versioned/lint#nolint-directive):

```sql
-- atlas:nolint DS103 BC104
ALTER TABLE "CUSTOMERS" RENAME COLUMN "NICK" TO "NICKNAME";
```

## Transactions and Partial Failures

Oracle commits the open transaction before and after every DDL statement
(https://docs.oracle.com/en/database/oracle/oracle-database/26/sqlrf/COMMIT.html). A migration file or a
declarative plan that fails halfway stays half applied, and `--tx-mode` cannot prevent it.

- Atlas records each statement it applies, so `atlas migrate status` shows a failed file as partially
  applied:

  ```text
  Migration Status: PENDING
    -- Current Version: 20261001000003 (1 statements applied)
    -- Next Version:    20261001000003 (2 statements left)
    -- Executed Files:  3 (last one partially)
    -- Pending Files:   1
  ```

- The next `atlas migrate apply` continues from the statement that failed. Fix that statement and the
  ones after it, leave the applied ones unchanged, and run `atlas migrate hash`.
- When the failed statement is a DML statement or a PL/SQL call, rather than DDL, the rollback also
  undoes Atlas's record of the statement before it, and the next run starts one statement early. Make
  sure that statement can run twice, such as a `CREATE OR REPLACE`, or check the schema and correct the
  file. Keep data changes in files of their own, apart from DDL.
- `atlas migrate down` reverts applied files on Oracle. It computes the revert on the dev database and
  checks that a column it drops holds no data (https://atlasgo.io/versioned/down).
- `atlas migrate set <version>` updates the revisions table only; it runs and reverts no SQL.
- Keep files small: one DDL statement per file leaves nothing half applied (`TX101`).

## PL/SQL in Migration Files

Migration files are SQL sent to the server, not SQL*Plus scripts.

- Atlas ends the PL/SQL objects it generates with `END;;`, the block's own `END;` and the statement
  delimiter:

  ```sql
  -- Create "ADD_TAX" function
  CREATE FUNCTION "ADD_TAX" (
    "P_AMOUNT" NUMBER
  )
  RETURN NUMBER DETERMINISTIC AS
  BEGIN
    RETURN ROUND(p_amount * 1.17, 2);
  END;;
  ```

- In a hand-written file, end each `CREATE [OR REPLACE]` of a function, procedure, trigger, type, package,
  or package body with `END;;`, or with a `/` on a line of its own, as in SQL*Plus
  (https://docs.oracle.com/en/database/oracle/oracle-database/26/sqpug/SQL-Plus-basics.html):

  ```sql
  CREATE OR REPLACE PACKAGE billing AS
    FUNCTION total RETURN NUMBER;
  END billing;
  /

  CREATE OR REPLACE PACKAGE BODY billing AS
    FUNCTION total RETURN NUMBER IS
      v NUMBER;
    BEGIN
      SELECT NVL(SUM(balance), 0) INTO v FROM accounts;
      RETURN v;
    END total;
  END billing;
  /
  ```

- Packages live only in migration files: the declarative workflow leaves them out of plans, and
  `migrate diff` neither creates nor drops them (`references/objects.md`).
- Anonymous blocks (`BEGIN ... END;` and `DECLARE ... END;`) fail: Atlas strips the block's final
  semicolon, with `/`, `END;;`, or `-- atlas:delimiter` alike, and Oracle answers
  `PLS-00103: Encountered the symbol "end-of-file" when expecting one of the following`. Write the logic
  as plain SQL statements, call procedures with `CALL`, or wrap the block in a procedure that the file
  creates, calls, and drops:

  ```sql
  CREATE OR REPLACE PROCEDURE "TMP_BACKFILL" AS
  BEGIN
    FOR r IN (SELECT id FROM orders WHERE status IS NULL) LOOP
      UPDATE orders SET status = 'NEW' WHERE id = r.id;
    END LOOP;
  END;
  /

  CALL "TMP_BACKFILL"();

  DROP PROCEDURE "TMP_BACKFILL";
  ```

- SQL*Plus commands fail: `SET DEFINE OFF` with `ORA-00922: missing or invalid option`, and `EXEC` with
  `ORA-00900: invalid SQL statement`. Use `CALL` in place of `EXEC`, and drop `SET`, `SHOW ERRORS`,
  `PROMPT`, `WHENEVER`, and `@file` lines.
- Oracle creates PL/SQL that does not compile, marks it `INVALID`, and Atlas reports the statement as
  applied. An invalid trigger fails every statement that fires it, with
  `ORA-04098: trigger '...' is invalid and failed re-validation`. Lint does not catch it; a migration test
  does (`references/testing.md`).
- After a deployment that changes tables, views, or PL/SQL that other code depends on, recompile the
  schema: `CALL DBMS_UTILITY.COMPILE_SCHEMA(schema => 'APP', compile_all => FALSE);`
  (https://docs.oracle.com/en/database/oracle/oracle-database/26/admin/managing-schema-objects.html).

## Locks and Online Changes

- `migrate apply` and `schema apply` hold a `DBMS_LOCK` lock for the run, which needs
  `EXECUTE ON SYS.DBMS_LOCK`. `--lock-timeout` waits 10 seconds by default; `--skip-lock` skips the lock
  (https://atlasgo.io/versioned/apply#controlling-advisory-locks). `migrate diff` and `migrate lint` take
  the lock on the dev database.
- `DDL_LOCK_TIMEOUT` is 0 by default, so DDL on a table with an open transaction fails at once
  (https://docs.oracle.com/en/database/oracle/oracle-database/26/refrn/DDL_LOCK_TIMEOUT.html):

  ```
  ORA-00054: Failed to acquire a lock (Type: "TM", Name: "DML", Description: "Synchronizes accesses to an object") because it is currently held by another session.
  ```

  Before 23ai the message is `ORA-00054: resource busy and acquire with NOWAIT specified or timeout
  expired`; match on the code. To wait instead, set the timeout at the top of the file:

  ```sql
  ALTER SESSION SET DDL_LOCK_TIMEOUT = 30;
  CREATE INDEX "ORDERS_TOTAL_IDX" ON "ORDERS" ("TOTAL");
  ```

  or for every file, with a hook in `atlas.hcl` (https://atlasgo.io/versioned/apply#migration-hooks):

  ```hcl
  hook "sql" "ddl_lock_timeout" {
    transaction {
      after_begin = [
        "ALTER SESSION SET DDL_LOCK_TIMEOUT = 30",
      ]
    }
  }
  ```

  While a DDL statement waits, new DML on the table queues behind it, so keep the timeout short.
- Atlas plans `CREATE INDEX` without `ONLINE`, which blocks writes for the whole build. On a populated
  table, write the index by hand with `ONLINE`, which lets DML continue
  (https://docs.oracle.com/en/database/oracle/oracle-database/26/sqlrf/CREATE-INDEX.html):

  ```sql
  CREATE INDEX "ORDERS_CUSTOMER_IDX" ON "ORDERS" ("CUSTOMER_ID") ONLINE;
  ```

  - `ONLINE` operations are included in Enterprise Edition and Free, not in Standard Edition 2
    (https://docs.oracle.com/en/database/oracle/oracle-database/26/dblic/Licensing-Information.html).
  - An online build ignores `DDL_LOCK_TIMEOUT` and waits for every open transaction on the table, so an
    idle session that never commits stalls it.
  - Atlas reads the index the same either way, and the next plan reports no changes.
- Add foreign keys and checks to populated tables without validating them, then validate in a later
  file. Validating an `ENABLE NOVALIDATE` constraint blocks neither reads nor writes
  (https://docs.oracle.com/en/database/oracle/oracle-database/26/sqlrf/constraint.html):

  ```sql
  ALTER TABLE "ORDERS" ADD CONSTRAINT "ORDERS_CUSTOMER_FK"
    FOREIGN KEY ("CUSTOMER_ID") REFERENCES "CUSTOMERS" ("ID") ENABLE NOVALIDATE;
  ```

  ```sql
  ALTER TABLE "ORDERS" MODIFY CONSTRAINT "ORDERS_CUSTOMER_FK" VALIDATE;
  ```

  Foreign keys and checks have no `validated` attribute, so the plan matches the schema in either
  state. A unique or primary key constraint needs a non-unique index first (`references/hcl.md`).
- With supplemental logging on, as GoldenGate and logical standby require, Oracle runs `ADD COLUMN`,
  `ENABLE NOVALIDATE`, and `VALIDATE` as blocking DDL
  (https://docs.oracle.com/en/database/oracle/oracle-database/26/sqlrf/Automatic-and-Manual-Locking-Mechanisms-During-SQL-Operations.html).
- Adding a column with a `DEFAULT` stores the default as metadata, without rewriting rows, for nullable
  and `NOT NULL` columns alike, except on index-organized, temporary, and some other tables
  (https://docs.oracle.com/en/database/oracle/oracle-database/26/sqlrf/ALTER-TABLE.html).
- A column's size or precision can always grow. Changing its type needs an empty column
  (`ORA-01439: column to be modified must be empty to change datatype`): add a new column, copy the
  data, and drop the old one in a later file.
- Changes that rebuild a large table online use `DBMS_REDEFINITION`
  (https://docs.oracle.com/en/database/oracle/oracle-database/26/admin/managing-tables.html), and
  upgrades of PL/SQL in use without downtime use edition-based redefinition
  (https://docs.oracle.com/en/database/oracle/oracle-database/26/adfns/editions.html). Run both outside
  Atlas, then record the file with `atlas migrate set <version>`.

## Renames

Atlas cannot plan a rename on Oracle: there is no `renamed_from`, and the SQL schema directive
`-- atlas:renamed_from` is ignored. A renamed column or table is planned as a drop and an add, which
loses its data.

1. Write the rename in a migration file. Lint reads it as a dropped column and an added one, so after
   checking it, silence `DS103` and `BC104`, plus `MF103` when the column is `NOT NULL`:

   ```sql
   -- atlas:nolint DS103 BC104 MF103
   ALTER TABLE "CUSTOMERS" RENAME COLUMN "NICK" TO "NICKNAME";
   ```

2. Rename the column in the schema. The next `migrate diff` reports no changes.

In the declarative workflow, edit the plan instead, replacing the drop and the add with the rename
(the `atlas` skill, `references/declarative.md`, Editing a plan).

## Revisions Table

- At schema scope, `"atlas_schema_revisions"`, a lowercase name, lives in the schema of the URL's user.
  Query it quoted: `SELECT * FROM "atlas_schema_revisions"`.
- At database scope, Atlas creates a schema-only account named `"atlas_schema_revisions"`, without a
  password, to hold the table, so the login needs `CREATE USER`. Point `--revisions-schema`, or
  `revisions_schema` in the env's `migration` block, at an existing schema instead.
- The user needs a quota on its default tablespace to write the table (`references/dev-database.md`).

## Existing Databases

A schema that has objects but no revisions table is refused until it gets a baseline, the first
migration recorded as applied without running (https://atlasgo.io/versioned/apply#existing-databases).
At database scope the error reads
`connected database is not clean: found schema "APP". baseline version or allow-dirty is required`.

```bash
atlas migrate diff baseline \
  --dir "file://migrations" \
  --dev-url "docker://oracle/free:latest-lite" \
  --to "oracle://APP:pass@localhost:1521/FREEPDB1"
```

Then pass its version once with `atlas migrate apply --baseline <version>`, or set `baseline` in the
env's `migration` block. The baseline holds only what Atlas manages: add the DDL of packages,
partitioned tables, and materialized views to it by hand, from `DBMS_METADATA.GET_DDL`, so new databases
get them too.

## Large Changes

Nothing in Atlas batches a single huge plan for you. To spread a large change over several deployments,
split it into several migration files and apply a few at a time with `atlas migrate apply <n>`.
