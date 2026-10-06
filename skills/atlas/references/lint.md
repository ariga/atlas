# Linting Reference (`atlas migrate lint`, `atlas schema lint`)

Atlas lints changes before they run. `migrate lint` analyzes migration files for destructive changes,
backward-incompatible changes, data-dependent changes, table locks, naming violations, and custom
rules, and it exits non-zero on errors so CI blocks the change. `schema lint` runs the same policy
rules against the whole desired schema.

| Command | Analyzes | Reports |
|---------|----------|---------|
| `atlas migrate lint --env <name>` | The new migration files: those not yet pushed to the registry, or with `--latest N` / `--git-base` the files those flags select | Only findings introduced by those files |
| `atlas schema lint --env <name>` | The entire desired schema | Every finding in the schema |

Use `migrate lint` in the versioned workflow and in CI. Use `schema lint` to audit an existing schema
against policy. Run `atlas login` before linting.

## Running `migrate lint`

```bash
atlas migrate lint --env <name>                     # registry project: files not yet pushed to the registry
atlas migrate lint --env <name> --latest 1          # the newest N migration files (local development)
atlas migrate lint --env <name> --git-base master   # no registry: files added since the git base branch
atlas migrate lint --env <name> -w                  # any of the above: open the report in the browser
atlas migrate lint --env <name> --format '{{ json . }}'   # any of the above: machine-readable report
```

Changeset detection picks which files to analyze. Read `atlas.hcl` first and use the case that
matches the project:

1. Registry comparison (most projects): the directory is pushed to the Atlas Registry
   (`migration.repo.name` is set, or `dir` is an `atlas://` URL). The local directory is compared with
   the latest pushed state. No flag and no git configuration needed; this is the CI default whenever
   a registry repo exists.
2. `--latest N`: the last N files by version. For local development, just pass `--latest 1`; no
   configuration needed.
3. `--git-base <branch>` (plus `--git-dir <path>` when the directory is not the repo root): files added
   on the current branch compared with the base branch. Only for projects that do not push to the
   registry.

The equivalent `atlas.hcl` settings:

```hcl
# Registry project: nothing to configure for lint beyond the repo.
env "ci" {
  migration {
    dir = "file://migrations"
    repo {
      name = "app"
    }
  }
}

# No registry: compare with the git base branch.
env "ci" {
  lint {
    git {
      base = "master"
      dir  = "<path>"   # optional working directory for git
    }
  }
}
```

`migrate lint` needs a dev database (`dev` in the env or `--dev-url`) to replay the files. The dev
URL scope must match the project (see `SKILL.md`, Dev Database).

## Built-in Analyzers

Each analyzer owns a family of check codes. The report prints the code and a link to its doc.

| Analyzer | Codes | Detects | Default |
|----------|-------|---------|---------|
| Destructive changes | `DS101`-`DS103` | Dropping schemas, tables, non-virtual columns | Error |
| Constraint drops | `CD101`-`CD103` | Dropping foreign-key, check, or primary-key constraints | Warning |
| Data-dependent changes | `MF101`-`MF104` | Changes that may fail on existing data: unique index on a populated column, NOT NULL column without a default | Warning |
| Backward-incompatible changes | `BC101`-`BC104` | Renaming or dropping tables and columns that running code still uses | Warning |
| Non-linear changes | (no code) | Files added out of order, or edited after they were pushed | Warning |
| Table locks and rebuilds (MySQL) | `MY101`-`MY148` | ALTERs that lock or copy the table | Warning |
| Concurrent index policy (PostgreSQL) | `PG101`-`PG110` | Index changes without `CONCURRENTLY`, missing `atlas:txmode none`, constraint creation that takes `ACCESS EXCLUSIVE` | Warning |
| Blocking changes (PostgreSQL) | `PG301`-`PG308` | Type changes that rewrite the table, volatile defaults, constraints that scan the whole table | Warning |
| Nested transactions | `TX101`, `TX201` | Transaction statements inside a migration that Atlas already wraps | Warning |
| Naming conventions | `NM101`-`NM106` | Names that violate the configured pattern | Warning |
| Ownership policy | `OW101`, `OW102` | Changes to objects the author's team does not own | Warning |
| SQL injection | `SA101` | Unsafe string concatenation in functions and procedures | Warning |
| Vulnerable extensions | per CVE | Installed extensions with a published CVE | Warning |
| Statement rules | per rule | Custom regex rules over raw statements | As configured |

The full catalog, with an example and a fix for every code, is at https://atlasgo.io/lint/analyzers.

## Configuring Lint Policy

The `lint` block in `atlas.hcl` sets policy globally or per environment. An env-level block inherits
and overrides the global one.

```hcl
lint {
  # Declarative workflow: when "schema apply" needs manual approval.
  # ERROR: only when the lint report has errors. WARNING: on warnings too. ALWAYS (default).
  review = ERROR

  destructive {
    error = false            # report drops as warnings instead of failing
    force = true             # developers cannot override with nolint
    allow_table  { match = "drop_.+" }   # deprecation workflow: tables renamed drop_* may be dropped
    allow_column { match = "drop_.+" }
  }

  data_depend  { error = true }
  incompatible {
    error = true
    drop_column { message = "deprecate ${self.table.name}.${self.name} in the application first" }
  }
  non_linear   {
    error   = true
    on_edit = WARN           # relax non-linear errors when editing the latest file
  }

  naming {
    match   = "^[a-z_]+$"
    message = "must be snake_case"
    index {
      match   = "^[a-z_]+_idx$"
      message = "indexes end with _idx"
    }
  }

  # Per-check overrides.
  check "PG301" { error = true }
  check "DS102" { skip = true }

  # Custom rules written in HCL (see below).
  rule "hcl" "policy" {
    src = ["schema.rule.hcl"]
  }

  # Custom report format.
  format = <<EOS
{{- range $f := .Files }}{{ json $f }}{{ end }}
EOS
}
```

`review = ERROR` also gates `atlas schema apply`: the apply auto-approves when the plan lints clean
and asks for approval when the linter reports an error.

## Silencing a Finding

Annotate the statement with `-- atlas:nolint` to exclude it from analysis. Prefer fixing the change.
Never add `nolint` without telling the user why the finding is a false positive.

```sql
-- atlas:nolint                    -- all analyzers, this statement
ALTER TABLE t1 DROP COLUMN c1;

-- atlas:nolint destructive        -- one analyzer by name
ALTER TABLE t2 DROP COLUMN c2;

-- atlas:nolint DS103              -- one check by code
ALTER TABLE t3 DROP COLUMN c3;
```

A directive on the first line of the file (`-- atlas:nolint` before any statement) applies to the
whole file. `destructive { force = true }` disables `nolint` for destructive checks.

## Custom Rules

Custom rules are HCL files with the `.rule.hcl` extension. Three blocks: `predicate` defines a reusable
condition, `rule "schema"` applies to the desired schema (reported by both `schema lint` and
`migrate lint`), and `rule "migrate"` applies to the change itself (`migrate lint` only). Reference:
https://atlasgo.io/hcl/rule.

```hcl
# schema.rule.hcl

# A predicate: true when a column is NOT NULL or has a default.
predicate "column" "not_null_or_have_default" {
  or {
    default { ne = null }
    null    { eq = false }
  }
}

# Schema rule: applies to every column in the schema.
rule "schema" "disallow-null-columns" {
  description = "require columns to be not null or have a default value"
  table {
    column {
      assert {
        predicate = predicate.column.not_null_or_have_default
        message   = "column ${self.name} must be not null or have a default value"
      }
    }
  }
}

# Migration rule: applies only to tables added by the analyzed migrations.
rule "migrate" "disallow-add-table-with-null-columns" {
  description = "disallow adding tables with null columns"
  add {
    table {
      column {
        assert {
          predicate = predicate.column.not_null_or_have_default
          message   = "column ${self.name} must be not null or have a default value"
        }
      }
    }
  }
}
```

- `predicate "<type>" "<name>"`: `type` is an object kind of the driver (`table`, `column`, `index`,
  `foreign_key`, `function`, `trigger`, `policy`, `role`, `user`, `permission`, ...). Reference it as
  `predicate.<type>.<name>`. Predicates can take arguments through `variable` blocks.
- `rule "schema" "<name>"` and `rule "migrate" "<name>"`: `description` is required and appears in the
  report. Nested blocks traverse the schema (`table { column { ... } }`) or the change
  (`add { table { ... } }`, `modify`, `drop`). Inside, `match` filters and `assert` checks.
- `self` is the current object; `${self.name}`, `${self.table.name}`, `${self.schema.name}` interpolate.

Wire the file in with `lint { rule "hcl" "<name>" { src = ["schema.rule.hcl"] } }`, globally or per
env, then run `migrate lint` or `schema lint`. Examples for columns, foreign keys, functions, audit
tables, RLS, security invoker views, roles, and permissions: https://atlasgo.io/lint/rules.

## Agent Workflow: Lint Failed

1. Read the code and message. `--format '{{ json . }}'` gives structured output (with `--latest 1`
   locally; no extra flag in CI when the directory is in the registry).
2. Decide whether the finding is real:
   - Destructive (`DS*`): confirm with the user. Prefer a deprecation step (rename to `drop_*`, drop
     later) or a backup.
   - Data-dependent (`MF*`): add a default, backfill first, or split into two migrations.
   - Backward-incompatible (`BC*`): coordinate the rename with application code, or add and migrate
     instead of renaming.
   - Concurrent index (`PG*`): use `CREATE INDEX CONCURRENTLY` and add `-- atlas:txmode none` at the
     top of the file.
   - Naming (`NM*`) and custom rules: rename to match the policy.
3. Fix the migration file (unapplied only), then `atlas migrate hash --env <name>` and re-lint.
4. If the change is applied already, write a new corrective migration instead.
5. Use `-- atlas:nolint <code>` only with the user's approval, and say so in the summary.

## Error Handling

| Error | Action |
|-------|--------|
| `checksum mismatch` | The directory was edited by hand. Run `atlas migrate hash --env <name>` |
| Lint exits 1 with no findings printed | Run with `-w` or `--format '{{ json . }}'` to see the full report |
| `--git-base` finds no files | The branch has no new files, or `dir` is wrong. Use `--latest 1` locally |
| Custom rule fails to parse | Check the `.rule.hcl` extension, the `predicate.<type>.<name>` reference, and that `description` is set |
| `command requires 'atlas login'` | Run `atlas login` |

## Documentation

- [Migration linting](https://atlasgo.io/versioned/lint)
- [Analyzers and check codes](https://atlasgo.io/lint/analyzers)
- [Custom linting rules](https://atlasgo.io/lint/rules)
- [Rule language reference](https://atlasgo.io/hcl/rule)
- [Lint block in atlas.hcl](https://atlasgo.io/atlas-schema/projects#configure-migration-linting)
- [Destructive change policy](https://atlasgo.io/guides/destructive-change-policy)
- [Review policy for schema apply](https://atlasgo.io/declarative/apply#review-policy)
