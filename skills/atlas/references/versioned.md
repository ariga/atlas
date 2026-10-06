# Versioned Migrations Reference (`atlas migrate`)

The versioned workflow keeps a migration directory of SQL files plus an `atlas.sum` integrity file.
`atlas migrate diff` plans new files from the desired schema, `atlas migrate lint` checks them,
`atlas migrate apply` runs pending files against a database and records each one in the
`atlas_schema_revisions` table. This reference covers diff generation in depth, directory
maintenance, and apply-time controls. Linting is in `references/lint.md`, tests in
`references/testing.md`, drift in `references/drift.md`, CI/CD in `references/cicd.md`.

## `atlas migrate diff`

Computes the diff between the migration directory (replayed on the dev database) and the desired
state, and writes a new migration file. Same schema change, same output, whichever model asked for it.

```bash
atlas migrate diff --env <name> "add_users_email"        # everything from atlas.hcl
atlas migrate diff add_users_email \
  --dir "file://migrations" \
  --to "file://schema.hcl" \                                # HCL, SQL file or directory, ORM, or a database URL
  --dev-url "docker://postgres/17/dev?search_path=public"
atlas migrate diff --env <name> "name" --edit             # open the generated file in $EDITOR before saving
atlas migrate diff --env <name> "name" --format '{{ sql . "  " }}'   # indented SQL
```

| Flag | Purpose |
|------|---------|
| `--to` | Desired state: `file://schema.hcl`, `file://schema.sql`, `file://schema/` (directory), an ORM loader (see `references/schema-sources.md`), or a live database URL (see Baseline) |
| `--dir` | Migration directory URL, default `file://migrations`. Append `?format=<fmt>` for other tools' layouts |
| `--dev-url` | Dev database. Scope must match the target (see `SKILL.md`, Dev Database) |
| `--schema`, `-s` | Limit to named schemas |
| `--qualifier` | Qualify table names with a custom schema name when working on a single schema |
| `--edit` | Edit the file before it is written; `atlas.sum` is updated after the editor closes |
| `--format` | Go template for the output |
| `--lock-timeout` | Wait for the directory lock (multi-writer CI) |

A no-op diff (directory already matches `--to`) prints "The migration directory is synced with the
desired state, no changes to be made" and creates nothing.

### Migration directory formats

Use the default `atlas` format. Atlas can also write `golang-migrate`, `goose`, `flyway`, `liquibase`,
and `dbmate` layouts (`migration { format = golang-migrate }` in `atlas.hcl`, or
`--dir "file://migrations?format=golang-migrate"`) so an existing deployer keeps working, but those
formats are limited and many Atlas features are not available on them. Read `atlas.hcl` before
generating a file: a file in the wrong format breaks the deployer.

### Diff policy

The `diff` block in `atlas.hcl` shapes what `migrate diff` (and `schema apply`) generates:

```hcl
variable "destructive" {
  type    = bool
  default = false
}

env "local" {
  diff {
    skip {
      drop_schema = !var.destructive   # do not plan DROP SCHEMA unless --var destructive=true
      drop_table  = !var.destructive
    }
    concurrent_index {                 # PostgreSQL: CREATE/DROP INDEX CONCURRENTLY
      add  = true                      # the file gets "-- atlas:txmode none"
      drop = true
    }
    materialized {
      with_no_data = true              # CREATE MATERIALIZED VIEW ... WITH NO DATA
    }
    add_table  { if_not_exists = true }
    drop_table { cascade = true, if_exists = true }
    add_column { if_not_exists = true }
    add_index  { if_not_exists = true }
  }
}
```

`atlas migrate diff --env local --var destructive=true` lets a single run plan the drops.

### Excluding objects from the diff

`migration { exclude = ["*.constraint_name", "audit_*"] }` tells `migrate diff` to ignore objects that
live in the directory but not in the desired schema (hand-written constraints, tables owned by another
tool). Use it only for objects that the schema source cannot express.

## Baseline an Existing Database

To adopt versioned migrations on a database that already has a schema. Start with an empty migration
directory: against an existing directory, step 1 writes an ordinary diff, not a baseline.

```bash
# 1. First migration = the current state. env://url reads the env's url so no database URL is typed.
atlas migrate diff baseline --env <name> --to env://url
#    or: atlas schema inspect --env <name> --format '{{ sql . | split | write "src" }}' then --to file://src

# 2. On every existing database, mark the baseline as applied without running it.
atlas migrate apply --env <name> --baseline "<version-from-filename>"

# 3. New databases run it in full.
atlas migrate apply --env <name>
```

`--allow-dirty` lets `migrate apply` start on a non-empty database that has no revisions table when a
baseline is not wanted (for example, a database that only contains objects excluded from the schema).

## Directory Maintenance

| Command | When |
|---------|------|
| `atlas migrate hash --env <name>` | After any manual edit of a migration file. Recomputes `atlas.sum`; `apply`, `lint`, `diff`, and `new` refuse a stale sum (`checksum mismatch`). `migrate new` and `migrate diff` rewrite `atlas.sum` themselves |
| `atlas migrate new --env <name> "name"` | Create an empty file for hand-written SQL (data fixes, statements Atlas cannot plan). Add `--edit` to open it |
| `atlas migrate edit --env <name> <version>` | Open an existing file in `$EDITOR` and re-hash when it closes |
| `atlas migrate rebase --env <name> <version>` | Move a migration to the end of the directory after a merge brought newer files in. Only for a file no environment has applied: rebase renames the file without reading the revisions table, so an applied file would run twice. Check `atlas migrate status` first. Re-lint afterwards: rebase reorders, it does not re-plan |
| `atlas migrate rm --env <name> <version>` | Remove an unapplied local file and update `atlas.sum`. Not for remote directories |
| `atlas migrate checkpoint --env <name> [tag]` | Write a checkpoint file that captures the whole directory state so new databases skip earlier files. Requires `--dev-url` or `dev` |
| `atlas migrate validate --env <name>` | Check `atlas.sum` and that every file parses |
| `atlas migrate ls --env <name>`, `atlas migrate show <version>` | List files, print one |
| `atlas migrate set --env <name> <version>` | Overwrite the revisions table to say the database is at `version`. Recovery only, with the user's explicit approval |
| `atlas migrate import --from "file://migrations?format=flyway" --to "file://atlas-migrations"` | Convert another tool's directory to Atlas format |

Rules: never edit a migration that any environment has applied; add a new file. After editing an
unapplied file, run `hash`, then `lint`. When `atlas.sum` conflicts in git, take either side, then run
`atlas migrate hash` and `atlas migrate rebase <your-version>` so your file sorts after the merged ones.

## `atlas migrate apply`

Applies pending files in order and records each in the revisions table. Always `--dry-run` first on
shared environments and check Atlas Cloud state (`references/cloud.md`).

```bash
atlas migrate apply --env <name> --dry-run          # prints the SQL with per-statement timings; nothing runs
atlas migrate apply --env <name>                    # all pending
atlas migrate apply --env <name> 1                  # at most one file
atlas migrate apply --env <name> --to-version 20260301120000
atlas migrate apply --env <name> --format '{{ json . }}'
atlas migrate apply --url "$DATABASE_URL" --dir "atlas://app?tag=latest"   # deploy from the registry
```

| Flag | Purpose |
|------|---------|
| `[amount]` | Apply at most N files |
| `--to-version` | Stop at this version |
| `--baseline` | First run on an existing database: mark this version applied without executing it |
| `--allow-dirty` | Start on a non-clean database that has no revisions table |
| `--tx-mode file\|all\|none` | One transaction per file (default), one for the whole run, or none. Per-file override: `-- atlas:txmode none` at the top of the file (needed for `CREATE INDEX CONCURRENTLY`) |
| `--exec-order linear\|linear-skip\|non-linear` | `linear` (default) fails when a file older than the current version is pending (out-of-order merge); `linear-skip` ignores it; `non-linear` applies it |
| `--revisions-schema` | Schema that holds `atlas_schema_revisions` |
| `--lock-timeout`, `--lock-name`, `--skip-lock` | Advisory lock so two deployers do not race |
| `--dry-run` | Print the SQL and the pre-checks without executing |
| `--format` | Go template; `{{ json . }}` for machine-readable output |

MySQL and other engines without transactional DDL cannot roll back a failed file completely; after a
failure there, compare `atlas migrate status` with the revisions table before retrying.

### Pre-execution checks

A `check "migrate_apply"` block in the env runs before any file executes. `allow` rules that evaluate
true pass; `deny` rules that evaluate true block the run with `message`. Rules see
`self.planned_migration.files` and `.statements`:

```hcl
env "prod" {
  check "migrate_apply" {
    deny "too_many_files" {
      condition = length(self.planned_migration.files) > 3
      message   = "Apply at most 3 files per run. Split the deployment."
    }
    deny "no_index_in_peak_hours" {
      condition = anytrue([for s in self.planned_migration.statements : regexmatch("(?i)create +index", s)])
                  && tonumber(formatdate("HH", timestamp())) >= 10
                  && tonumber(formatdate("HH", timestamp())) <= 14
      message   = "CREATE INDEX is blocked between 10:00 and 14:00 UTC"
    }
    drift {                          # see references/drift.md
      on_error = FAIL
    }
  }
}
```

Migration files can also carry their own checks (`-- atlas:txtar` pre-migration checks that abort
the file when a condition holds) and hooks; see the docs links below.

### Down migrations

```bash
atlas migrate down --env <name> --dry-run           # revert the last applied file
atlas migrate down --env <name> 2                   # the last two
atlas migrate down --env <name> --to-version 20260301120000
atlas migrate down --env <name> --to-tag <registry-tag>
```

Atlas plans the reverse of each applied file on the dev database, and validates hand-written down
files when the directory has them. Reverting data-destroying files cannot restore data; say so
before running `down` on anything but a development database.

### Multi-tenant apply

One `env` block with `for_each` expands to one target per tenant:

```hcl
data "sql" "tenants" {
  url   = var.url
  query = "SELECT schema_name FROM information_schema.schemata WHERE schema_name LIKE 'tenant_%'"
}

env "prod" {
  for_each = toset(data.sql.tenants.values)
  url      = urlqueryset(var.url, "search_path", each.value)
  migration {
    dir = "atlas://app"
  }
}
```

`atlas migrate apply --env prod` then applies to every tenant and reports per target; Atlas Cloud
records the run as one multi-target deployment (`references/cloud.md`).

## Registry

```bash
atlas migrate push --env <name> app               # push the directory; tag defaults to the git commit
atlas migrate push --env <name> app:v1.2.0        # explicit tag
atlas migrate apply --url "$URL" --dir "atlas://app?tag=v1.2.0"
atlas migrate apply --url "$URL" --dir "atlas://app?version=20260301120000"
```

Pushing makes the directory an immutable, versioned artifact that CD can deploy without the source
repo, and it is what `migrate lint` compares against by default and what the drift checks read.

## Agent Workflow: Generate a Migration

1. Read `atlas.hcl`: env name, `migration.dir` (and its `format`), `diff` policy, `dev`.
2. Edit the schema source, then `atlas schema validate --env <name>`.
3. `atlas migrate diff --env <name> "<snake_case_name>"`. Read the generated file back to the user.
4. `atlas migrate lint --env <name> --latest 1`. Fix findings in the schema source and regenerate
   (delete the file first with `atlas migrate rm`, or edit and `hash`).
5. If the change moves data, add a `test "migrate"` case (`references/testing.md`) and run it.
6. `atlas migrate apply --env <name> --dry-run`, then apply on the development database.
7. Commit the migration file and `atlas.sum` together.

## Error Handling

| Error | Action |
|-------|--------|
| `checksum mismatch` / `atlas.sum` out of sync | `atlas migrate hash --env <name>` |
| `migration file ... was not applied in order` (out-of-order) | Newer files landed first. `atlas migrate rebase <version>` locally, or `--exec-order linear-skip` only with approval |
| `connected database is not clean` | The database has objects but no revisions table. Baseline it (`--baseline`) or, if intended, `--allow-dirty` |
| `The migration directory is synced with the desired state` | Not an error: nothing to generate. Check the schema edit was saved and `--to` points at it |
| `ModifySchema is not allowed` | Dev URL scope does not match the target scope |
| Statement failed mid-file on MySQL | The file is partially applied. Compare `atlas migrate status` with the database, fix forward with a new file |

## Documentation

- [Migration authoring: migrate diff](https://atlasgo.io/versioned/diff)
- [Migration apply](https://atlasgo.io/versioned/apply)
- [Pre-execution checks](https://atlasgo.io/versioned/apply#pre-execution-checks)
- [Pre-migration checks in files](https://atlasgo.io/versioned/checks)
- [Migration hooks](https://atlasgo.io/versioned/pre-post-hooks)
- [Down migrations](https://atlasgo.io/versioned/down)
- [Checkpoints](https://atlasgo.io/versioned/checkpoint)
- [Directory integrity (atlas.sum)](https://atlasgo.io/concepts/migration-directory-integrity)
- [Import from other tools](https://atlasgo.io/versioned/import)
- [Troubleshooting](https://atlasgo.io/versioned/troubleshoot)
