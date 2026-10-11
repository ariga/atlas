# Oracle Dev Database and Connections

The dev database is a temporary database Atlas uses to load the desired state, replay migrations, and
compute plans (https://atlasgo.io/concepts/dev-database). For Oracle it must match the target's scope,
release, and storage settings.

## Scope

In Oracle, a schema is a user: the objects that one user owns. The URL's `mode` parameter decides what
Atlas covers (https://atlasgo.io/concepts/url#scope):

| URL | Scope | Atlas covers | DDL it writes |
|-----|-------|--------------|---------------|
| `oracle://APP:pass@localhost:1521/FREEPDB1` | Schema, the default (`mode=schema`) | The schema of the user in the URL | Unqualified (`"ORDERS"`) |
| `oracle://ADMIN:pass@localhost:1521/FREEPDB1?mode=database` | Database | Every schema in the pluggable database that Oracle does not maintain | Qualified (`"APP"."ORDERS"`) |

- The path is the service name of a pluggable database (PDB), such as `FREEPDB1`. Connect to the PDB's
  service, not to the root container
  (https://docs.oracle.com/en/database/oracle/oracle-database/26/multi/administering-pdbs-with-sql-plus.html).
- At schema scope, Atlas manages the schema of the user it connects as, so connect as the schema owner.
- At database scope, every account Oracle does not maintain counts as a schema, including `PDBADMIN` and
  users that own nothing. A schema the desired state does not declare is planned for a drop: each of its
  objects, then `DROP USER`. Limit the run with `schemas = ["APP"]` in the env, or `--schema APP`. The
  list must name every schema the HCL declares, or Atlas fails with
  `schema "X" from paths [...] is not requested`.
- Database-scope plans can drop indexes that the desired state declares, with an unqualified
  `DROP INDEX`, and leave new indexes out. They can also plan default changes that schema scope does not,
  such as `MODIFY "IS_VIP" DEFAULT false` on a `BOOLEAN` column. Keep tables, indexes, code, and grants in
  an env per schema at schema scope, and review every database-scope plan for `DROP INDEX`. To keep an
  env at database scope, exclude indexes: `exclude = ["*.*.*[type=index]"]`.
- Roles belong to the whole database: managing them needs database scope, in an env of their own
  (`references/security.md`).
- The dev URL takes the target's scope: `docker://oracle/free:latest-lite` for schema scope, and
  `docker://oracle/free:latest-lite?mode=database` for database scope.

## Docker Dev URLs

The `docker://` driver starts a fresh container for each run (https://atlasgo.io/concepts/dev-database):

```bash
# When working on a single database schema.
--dev-url "docker://oracle/free:latest-lite"

# When working on multiple database schemas.
--dev-url "docker://oracle/free:latest-lite?mode=database"
```

- The image comes from Oracle's container registry: `free` is Oracle AI Database 26ai Free (tags
  `latest` and `latest-lite`, for amd64 and arm64), `express` is 21c XE (amd64 only), and `enterprise`
  holds 19c and 21c. Pulling an enterprise image needs an Oracle account, accepting the license in the
  registry, and `docker login container-registry.oracle.com`
  (https://container-registry.oracle.com/ords/ocr/ba/database/free).
- Atlas connects to the `FREEPDB1` database as `PDBADMIN`.
- A Free container starts in 20 to 50 seconds and writes about 4 GB to disk while it runs. Pull the image
  before the first run (`docker pull container-registry.oracle.com/database/free:latest-lite`): it is
  1.5 GB or more. Each command starts its own container, and the previous one can still be on disk while
  it is removed.
- `latest-lite` is whatever image the machine pulled last. Pin a release tag, such as
  `free:23.26.3.0-lite`, so every machine plans with the same release.
- `docker://` needs a Docker client and a reachable daemon. Without one, such as inside the
  `arigaio/atlas` image or a CI runner without Docker, use a dev database you provide (below).

### The lite image

The lite image removes features to stay small, such as XML DB, so it has no `XMLTYPE`, and Oracle Text
lexers other than `BASIC_LEXER` (https://container-registry.oracle.com/ords/ocr/ba/database/free). It
also has no `USERS` tablespace: `PDBADMIN` creates its tables in `SYSTEM`, a tablespace with manual
segment space management. `JSON` and `VECTOR` columns cannot live there:

```
Error: sql/oracle: sql/oracle: add new table named "DOCS": ORA-43853: JSON type cannot be used in non-automatic segment space management tablespace "SYSTEM"
```

Create a `USERS` tablespace in the `docker` block's `init`, below, or use the full image
(`free:latest`). A schema with `XMLTYPE` columns needs the full image.

### Named containers with `atlas tool docker`

`atlas tool docker` creates a named container from a `docker://` URL and prints its connection URL. It
cleans the database if it is not empty, so never run it against a container that holds data to keep
(https://atlasgo.io/cli-reference#atlas-tool-docker). Use it for a local database to apply migrations to:

```bash
export DATABASE_URL=$(atlas tool docker --url "docker://oracle/free:latest-lite" --name my-ora)
atlas tool docker kill --name my-ora
```

The printed URL connects as `PDBADMIN`, written as `%22PDBADMIN%22`. `--env <env>` reads the dev URL of
an env instead, such as a `docker` block.

## The `docker` Block

A `docker "oracle"` block configures the dev container (https://atlasgo.io/hcl/config#docker.oracle).
It needs `database`, the PDB name; without it, Atlas fails with
`the pluggable database (PDB) name is required for Oracle containers`. The `docker` and `dev` blocks need
`atlas login`. If `atlas whoami` shows no login, ask the user to run `atlas login`; a user without an
account can start a free trial with it.

```hcl
docker "oracle" "dev" {
  image    = "container-registry.oracle.com/database/free:latest-lite"
  database = "FREEPDB1"
  init     = <<-SQL
    CREATE TABLESPACE USERS DATAFILE 'users01.dbf' SIZE 100M AUTOEXTEND ON;
    ALTER DATABASE DEFAULT TABLESPACE USERS;
  SQL
}

env "local" {
  url = getenv("DATABASE_URL")
  dev = docker.oracle.dev.url
  src = "file://schema.oc.hcl"
}
```

- `init` runs with administrator privileges once the database is up: create tablespaces, set database
  defaults, and grant `PDBADMIN` what it lacks. `ALTER DATABASE DEFAULT TABLESPACE` moves `PDBADMIN`, and
  every user Atlas creates at database scope, to the new tablespace.
- `baseline` runs as `PDBADMIN`. Statements that need more, such as `CREATE TABLESPACE`, fail with
  `Error: execute baseline: ORA-01031: insufficient privileges`. Use `baseline` for objects the schema
  uses but does not manage, created in `PDBADMIN`'s schema.
- `mode = "database"` gives the container database scope. The value is a quoted string.
- At database scope on 23ai and later, `PDBADMIN` cannot comment on other schemas' tables, so a plan that
  sets a comment fails with `ORA-41900: missing COMMENT privilege on "APP"."ORDERS"`. Add
  `GRANT COMMENT ANY TABLE TO PDBADMIN;` to `init`.
- `timeout` raises the 1 minute wait for the database to start. Enterprise images start in minutes:
  `timeout = "10m"`.

## Your Own Dev Database

Oracle has no `dev` block. Instead, point `dev` at an empty user in a database dedicated to Atlas, such
as an Oracle container that runs as a CI service. At schema scope this takes seconds instead of a
container start per command, and Atlas leaves the user empty when it finishes:

```hcl
env "ci" {
  url = getenv("DATABASE_URL")
  dev = "oracle://DEV:${urlescape(getenv("DEV_PASSWORD"))}@oracle-ci:1521/FREEPDB1"
  src = "file://schema.oc.hcl"
}
```

- The dev user needs the same privileges as the Atlas user (Privileges below), on the target's release
  and `MAX_STRING_SIZE`.
- Give each env its own dev user: two runs that share one at the same time see each other's objects.
- At database scope, the dev database is a whole PDB: give it one that nothing else uses.
- Never point `dev` at a real environment's database: Atlas fails with
  `connected database is not clean: found table "ORDERS" in connected schema`, or, if the schema is empty,
  uses it as a sandbox.

## Versions and Settings

The dev database accepts what its release accepts, and a newer dev database plans statements the target
rejects. Read the target's values, and run the dev database with the same ones:

```sql
SELECT banner_full FROM v$version;
SELECT name, value FROM v$parameter WHERE name IN ('compatible', 'max_string_size', 'nls_length_semantics');
SELECT property_value FROM database_properties WHERE property_name = 'DEFAULT_PERMANENT_TABLESPACE';
```

- Release. The Free image is Oracle AI Database 26ai, release 23.26, which replaced 23ai through a release
  update (https://docs.oracle.com/en/database/oracle/oracle-database/26/nfcoa/). Features by release:

  | Feature | Release |
  |---------|---------|
  | Identity columns, `DEFAULT ON NULL`, metadata-only `ADD COLUMN ... DEFAULT` on nullable columns | 12.1 |
  | Names up to 128 bytes, when `COMPATIBLE` is 12.2 or higher (30 bytes before) | 12.2 |
  | `JSON` data type | 21c |
  | `BOOLEAN`, `VECTOR` (23.4), `IF [NOT] EXISTS`, `DEFAULT ON NULL FOR INSERT AND UPDATE`, schema privileges, up to 4096 columns | 23ai |

  A 26ai dev database accepts `BOOLEAN` and `JSON` columns, and the migration then fails on a 19c target
  with `ORA-00902: invalid datatype`. For a 19c target, run the dev database on a 19c enterprise image
  (`enterprise:19.3.0.0` on amd64, `enterprise:19.19.0.0` on arm64), or keep the Free image and forbid
  the newer types with a custom lint rule (`references/custom-rules.md`). No free 19c image exists.
- `MAX_STRING_SIZE`. `STANDARD`, the default and the Free image's setting, limits `VARCHAR2` to 4000
  bytes; `EXTENDED` allows 32767
  (https://docs.oracle.com/en/database/oracle/oracle-database/26/refrn/MAX_STRING_SIZE.html). A
  `VARCHAR2(8000)` column that an `EXTENDED` target accepts fails on a `STANDARD` dev database with
  `ORA-00910: specified length too long for its datatype`. Autonomous Database uses `EXTENDED`. The change
  cannot be undone, and the lite image cannot make it: for an `EXTENDED` target, provide a dev database
  set up the same way.
- `NLS_LENGTH_SEMANTICS` defaults to `BYTE`: `VARCHAR2(100)` holds 100 bytes, which is fewer characters
  for multibyte text. Write `VARCHAR2(100 CHAR)` for a character length (`references/hcl.md`).
- Free limits each instance to 2 CPUs, 2 GB of memory, and 12 GB of user data
  (https://docs.oracle.com/en/database/oracle/oracle-database/26/xeinl/licensing-restrictions.html),
  enough for a dev database.

## Privileges of the Atlas User

Connect Atlas as a dedicated user that owns the application schema, not as an administrator
(https://atlasgo.io/guides/oracle/security-declarative). It needs these, granted once by an
administrator:

```sql
CREATE USER APP IDENTIFIED BY "<password>" DEFAULT TABLESPACE USERS QUOTA UNLIMITED ON USERS;
GRANT CREATE SESSION, CREATE TABLE, CREATE VIEW, CREATE SEQUENCE, CREATE PROCEDURE,
      CREATE TRIGGER, CREATE TYPE, CREATE SYNONYM TO APP;
GRANT SELECT_CATALOG_ROLE TO APP;
GRANT EXECUTE ON SYS.DBMS_LOCK TO APP;
```

- Each `CREATE` privilege covers one object type in the user's own schema; `CREATE PROCEDURE` covers
  functions and packages too. Add `CREATE MATERIALIZED VIEW`, `CREATE DATABASE LINK`, or `CREATE ROLE` if
  the schema has them (https://docs.oracle.com/en/database/oracle/oracle-database/26/sqlrf/GRANT.html).
- `RESOURCE` does not include `UNLIMITED TABLESPACE` since 12c, so give the user a quota. On 19c,
  `RESOURCE` also lacks `CREATE VIEW` and `CREATE SYNONYM`
  (https://docs.oracle.com/en/database/oracle/oracle-database/19/dbseg/predefined-roles-oracle-database-installation.html).
- A missing grant fails with one of these:

  | Error | Missing |
  |-------|---------|
  | `query server property: ORA-00942: table or view "SYS"."V_$INSTANCE" does not exist` | `SELECT_CATALOG_ROLE`: Atlas reads the server version and the data dictionary |
  | `acquiring database lock: sql/oracle: allocate lock: ORA-06550 ... PLS-00201: identifier 'DBMS_LOCK' must be declared` | `EXECUTE ON SYS.DBMS_LOCK`: `migrate apply` and `schema apply` hold a lock for the run |
  | `write revision: ORA-01950: The user APP has insufficient quota on tablespace USERS` (19c: `no privileges on tablespace`) | A quota on the user's default tablespace |
  | `ORA-01031: insufficient privileges` | The `CREATE` privilege for an object type |

- A table created without a quota succeeds while it is empty, since Oracle defers its storage, and the
  first row then fails with `ORA-01950`.
- Database scope needs privileges on other users' objects, listed in the Oracle tab of
  https://atlasgo.io/concepts/url.

## Managed Providers

- Amazon RDS for Oracle: the master user is not `SYSDBA`, and grants on `SYS` objects go through an RDS
  procedure, which grants only privileges the master user holds
  (https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/Appendix.Oracle.CommonDBATasks.TransferPrivileges.html).
  Run it once from a SQL client, not from a migration file (`references/migrations.md`, PL/SQL in
  Migration Files):

  ```sql
  BEGIN
    rdsadmin.rdsadmin_util.grant_sys_object(
      p_obj_name  => 'DBMS_LOCK',
      p_grantee   => 'APP',
      p_privilege => 'EXECUTE');
  END;
  /
  ```

  RDS sets `MAX_STRING_SIZE` in the parameter group.
- Oracle Autonomous Database: connect as `ADMIN`, which is not `SYSDBA`, or as a user it creates, through
  a predefined service such as `<db>_low` or `<db>_tp`
  (https://docs.oracle.com/en/cloud/paas/autonomous-database/serverless/adbsb/predefined-database-services-names.html).
  Mutual TLS, the default on port 1522, needs the wallet: `?ssl=true&wallet=/path/to/wallet`. TLS without
  a wallet needs an access control list or a private endpoint
  (https://docs.oracle.com/en/cloud/paas/autonomous-database/serverless/adbsb/connect-introduction.html).
- Autonomous Database ignores tablespace and storage clauses in `CREATE TABLE`, and refuses tablespace
  changes (https://docs.oracle.com/en/cloud/paas/autonomous-database/serverless/adbsb/autonomous-sql-commands.html).
  It uses `MAX_STRING_SIZE = EXTENDED`, so plan on a dev database with the same setting.

## Connection URLs

- Write the URL as `oracle://user:pass@host:1521/service_name`. The service names a PDB.
- Percent-encode special characters in passwords, or build the URL with `urlescape`
  (https://atlasgo.io/faq/url-special-chars).
- TLS options go in the query: `?ssl=true` turns TLS on, and `wallet` points at a wallet directory for
  mutual TLS. Against a server without TLS, `ssl=true` fails with `query server property: EOF`.
- `mode` is read by Atlas and never sent to the server.

## Errors

| Error | Cause | Fix |
|-------|-------|-----|
| `The oracle driver is available only as custom build`, or `Error: timeout: ...` with that text on a `docker://` dev URL | The installed Atlas has no Oracle driver | Install the Oracle build: `curl -sSf https://atlasgo.sh \| ATLAS_FLAVOR="oracle" sh` |
| `The oracle driver is available only to Atlas Pro users` | No login | Ask the user to run `atlas login` |
| `ORA-43853: JSON type cannot be used in non-automatic segment space management tablespace "SYSTEM"` | The lite dev image has no `USERS` tablespace | Create one in `init`, or use the full image |
| `the pluggable database (PDB) name is required for Oracle containers` | The `docker` block has no `database` | Set `database = "FREEPDB1"` |
| `execute baseline: ORA-01031: insufficient privileges` | `baseline` runs as `PDBADMIN` | Move the statement to `init` |
| `ORA-41900: missing COMMENT privilege` | Database scope on a 23ai or later dev image | `GRANT COMMENT ANY TABLE TO PDBADMIN;` in `init` |
| `ORA-01940: cannot drop a user who is currently connected` | A database-scope plan drops `PDBADMIN`, the dev login | Declare the schema, or limit the run with `schemas` |
| `schema "X" from paths [...] is not requested` | `--schema` or `schemas` leaves out a schema the HCL declares | List every declared schema |
| `ORA-00902: invalid datatype` on the target | The dev database is newer than the target | Match the release (Versions and Settings) |
| `ORA-00910: specified length too long for its datatype` | `MAX_STRING_SIZE` differs | Match the target's `MAX_STRING_SIZE` |
| `connected database is not clean` | `dev` points at a schema that is not empty | Use a Docker dev URL or an empty user |
| `exec: "docker": executable file not found in $PATH` | No Docker client where Atlas runs | Use a dev database you provide |
