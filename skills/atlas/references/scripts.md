# Data Scripts Reference (`atlas script`)

Data Scripts are HCL files that describe data operations: transactional mutations, reads and reports,
and batched loops. Atlas runs them with `atlas script exec`, `atlas script query`, and `atlas script loop`,
with transactions, guards, assertions, and output masking built in. They are Day 2 work that comes after
the schema is in place, run on demand, and not recorded in the migration history.

Run `atlas login` before using them.

## When to Use a Script

| Task | Use |
|------|-----|
| Backfill a column, re-key rows, re-encrypt a field | `script "exec"` or `script "loop"` |
| Delete or anonymize a user's data in bounded batches | `script "loop"` |
| Print a report, health check, or masked export | `script "query"` |
| Verify an invariant on a schedule (probe / monitor) | `script "exec"` with only `assert` blocks, or `script "query"` |
| Change the schema, or data that must ship with a schema change | A migration (`atlas migrate diff`), not a script |
| Lookup and reference rows that must match a desired state everywhere | The declarative `data` block, not a script |

A migration is applied once per database, in order, and Atlas records that it ran. A script runs every
time it is invoked, so the script carries its own safety: `condition` guards, `expect_rows`, and batch
bounds.

## Commands

```bash
atlas script exec  --url "$URL" --file "file://scripts.hcl" --run '^archive_user$'
atlas script query --url "$URL" --file "file://scripts.hcl" --run '^vip_count$' --quiet
atlas script loop  --url "$URL" --file "file://scripts.hcl" --run '^gdpr_purge$'
atlas script test  --env <name>                     # run test "script" cases on the dev database
atlas script push  --file "file://scripts" <repo>   # publish to the Atlas Registry
```

Each verb runs only scripts of its own kind. Prefer `--env <name>`: the selected env supplies `url`
and the script source from its `script { src = "file://scripts" }` block, so `--file` becomes optional.

| Flag | Purpose |
|------|---------|
| `--url` | Runtime database the script runs against (omit when `--env` provides it) |
| `--file` | Script source: one file, a directory of `*.script.hcl` files, or a pushed `atlas://<repo>` archive |
| `--run <regexp>` | Select scripts by name. Unanchored: `--run purge` also matches `purge_v2`. Anchor it: `'^purge$'` |
| `--var name=value` | Bind a `variable` declared in the script |
| `-q`, `--quiet` | Print only the script's product (query sections and `output` lines), no streaming report |
| `--format '{{ json . }}'` | Render the run report through a Go template, for machine-readable results |

`--run` rules:
- Every match runs. One failure does not stop the others; errors are reported together at the end, and
  scripts that already committed stay committed.
- A pattern that matches nothing prints `No scripts found to run` and exits `0`. In CI, assert on the
  output, not the exit code.
- Omit `--run` and every script of that kind in the source runs.

## File Structure

A script file is a flat list of top-level blocks: runnable `script` wrappers plus the `variable`,
`locals`, and `mask` declarations they reference. Every script accepts a `description`.

```hcl
variable "days" {
  type    = number
  default = 30
}

script "exec" "cancel_pending" {
  description = "Cancel orders left pending for more than var.days days"
  exec {
    sql  = "UPDATE orders SET status = 'canceled' WHERE status = 'pending' AND created_at < now() - ($1 || ' days')::interval"
    args = [var.days]
  }
}
```

Variables bind from `--var`, or from extra attributes on the selected `env` block (the env wins), with
`default` used when neither is set.

### SQL placeholders

Atlas never parses or rewrites SQL. Placeholders are driver-native and `args` bind scalars only:

| Driver | Placeholder | Expand a JSON list with |
|--------|-------------|-------------------------|
| PostgreSQL | `$1`, `$2` | `json_array_elements_text($1)` |
| MySQL | `?` | `JSON_TABLE(?, ...)` |
| SQLite | `?` | `SELECT value FROM json_each(?)` |
| SQL Server | `@p1` | `OPENJSON(@p1)` |
| ClickHouse | `?` | `arrayJoin(JSONExtract(?, 'Array(Int64)'))` |
| Oracle | `:1` | `JSON_TABLE(:1, ...)` |

To pass a list, `jsonencode` it: `args = [jsonencode(query.ids.rows[*].id)]`.

## `script "exec"`: Transactional Mutations

Blocks run in written order inside one transaction. A run ends in one of three ways: the body completes
and commits; an unmet `condition` or a fired `break` stops the script and commits the work done so far;
a failing `assert` or `check` aborts the run and rolls it back. The body needs at least one `exec`,
`assert`, or `check`.

| Block | Purpose |
|-------|---------|
| `tx { mode, on_error }` | `mode = AUTO` (default, one transaction) or `NONE`; `on_error = ROLLBACK` (default) or `COMMIT` |
| `condition "<name>" { sql }` | Pre-flight guard, only at the start of the body. Falsy (`false`/`0`) stops gracefully; `NULL` is an error. Wrap nullable expressions in `COALESCE` with a value of the same type, such as `COALESCE(..., false)` |
| `break "<name>" { sql \| expr }` | Stops mid-body when true, commits work so far. `expr` evaluates in-process, e.g. `length(query.pending.rows) == 0` |
| `assert "<name>" { sql, error_message }` | Invariant check. Falsy or `NULL` fails the run and rolls back |
| `check "<name>" { sql, output \| match, format }` | Compares serialized query output (CSV default, or TABLE) to an exact `output` or a regexp `match` |
| `exec "<name>" { sql, args, expect_rows }` | The write. `expect_rows` asserts the affected row count; a mismatch aborts |
| `query "<name>" { sql, args, rows { col = type } }` | Bound read: captures rows for later blocks as `query.<name>.rows[*].<col>` |
| `output { message }` | A line printed as the script's product, interpolating `${...}` |
| `http { ... }` | Call an external endpoint from the script |

```hcl
script "exec" "archive_user" {
  condition "is_active" {
    sql = "SELECT EXISTS (SELECT 1 FROM users WHERE id = $1 AND status = 'active')"
    args = [var.user_id]
  }
  exec "archive" {
    sql         = "UPDATE users SET status = 'archived' WHERE id = $1"
    args        = [var.user_id]
    expect_rows = 1
  }
  assert "archived" {
    sql           = "SELECT status = 'archived' FROM users WHERE id = $1"
    args          = [var.user_id]
    error_message = "user was not archived"
  }
}
```

An exec script made of only `assert` blocks is a smoke test or post-deploy verifier: it writes nothing
and fails when an invariant no longer holds.

## `script "query"`: Reads and Reports

Read-only, no write transaction. Each inner `query` runs in written order and prints its result as a
section. Use it for recurring reports, health checks, masked exports, and reads that feed a later query.

| Block | Purpose |
|-------|---------|
| `query "<name>" { sql, args, format }` | Prints the result. `format = TABLE` (default, aligned grid) or `CSV` (no header; a single scalar prints bare) |
| `query "<name>" { sql, rows { col = type } }` | Bound query: prints nothing, exposes `query.<name>.rows`. Mutually exclusive with `format` and `mask` |
| `break "<name>" { sql \| expr, message }` | Ends the run early, e.g. when a report has nothing to say; sections already printed stay printed |
| `output { message }` | A custom line composed from the results |
| `mask { columns, method }` | Redacts result columns before they are printed (see Masking) |
| `http { ... }` | Calls an endpoint; a trailing `http` block posts the report (Slack, webhook, incident tool) |

```hcl
script "query" "health" {
  query "orphans" {
    sql    = "SELECT count(*) AS orphan_orders FROM orders WHERE user_id NOT IN (SELECT id FROM users)"
    format = CSV
  }
  query "top_ids" {
    sql = "SELECT user_id FROM orders GROUP BY user_id ORDER BY sum(total) DESC LIMIT 5"
    rows {
      user_id = int
    }
  }
  query "top_detail" {
    sql    = "SELECT email, plan FROM users WHERE id IN (SELECT value FROM json_each(?))"
    args   = [jsonencode(query.top_ids.rows[*].user_id)]
    format = TABLE
    mask {
      columns = ["email"]
      method  = PARTIAL
      keep_left  = 2
      keep_right = 4
    }
  }
  output {
    message = "top ${length(query.top_ids.rows)} spenders shown above"
  }
}
```

Run with `--quiet` to get only the sections and `output` lines, or `--format '{{ json . }}'` for the
full report as JSON.

## `script "loop"`: Batched Work

Runs a `do` body repeatedly, once per page of an optional source, each iteration in its own transaction
by default. Use it for data migrations too large for one statement: purges, backfills, re-encryption.

Body order is enforced by the parser:

1. Pre-loop: zero or more `condition` blocks, run once before the iterator opens.
2. The loop: exactly one `do { }` and at most one `iterator "<mode>" { }`.
3. Post-loop: zero or more `assert` / `check` blocks, run once after the loop ends, outside any transaction.

`iterator` modes:
- `iterator "keyset"`: you write the paginated SELECT with a `> cursor` seek and `LIMIT`; Atlas threads
  the cursor. Sub-blocks: `cursor { col = type }`, `batch { col = type }` (the page), `init { sql }`
  (iteration 1), `next { sql, args = [cursor.col] }` (iterations 2+). The page is exposed as
  `iterator.keyset.batch[*].<col>`.
- `iterator "range" { from, to, step }`: walks a numeric interval; the chunk is exposed as
  `iterator.range.from` / `iterator.range.to`.
- No iterator: the `do` body repeats until a `break` fires or `policy.schedule` bounds it. `self.index`
  is the 0-based iteration counter.

`do` commands, run in written order: `exec`, `query` with `rows`, `assert`, `check`, `break`, `continue`,
`log`, `output`, `sleep`, `http`, and an explicit `tx` command. `break` stops the whole loop, `continue`
skips the rest of the iteration; both commit the work done so far. `on_error = ABORT` (default) or
`CONTINUE` decides whether the loop proceeds after an iteration fails.

`policy` sub-blocks (all optional):
- `tx { mode = PER_ITERATION | MANUAL }`: `PER_ITERATION` (default) wraps each iteration in one transaction.
  `MANUAL` autocommits unless grouped in a `do`-body `tx` command. `http` and the `tx` command require `MANUAL`.
- `schedule { every, limit, timeout, pause_when }`: fixed delay between iterations, max iterations,
  wall-clock bound, and a boolean SQL probe (replica lag, connection count) that pauses the loop while true.
- `ramp { stage { size, iterations | hold } ... }`: staged batch-size ramp-up. The current size is
  available in iterator SQL as `${ramp.size}`.

```hcl
script "loop" "purge_inactive" {
  condition "have_inactive" {
    sql = "SELECT count(*) > 0 FROM users WHERE active = false"
  }
  iterator "keyset" {
    cursor { id = int }
    batch  { id = int }
    init {
      sql = "SELECT id FROM users WHERE active = false ORDER BY id LIMIT ${ramp.size}"
    }
    next {
      sql  = "SELECT id FROM users WHERE active = false AND id > $1 ORDER BY id LIMIT ${ramp.size}"
      args = [cursor.id]
    }
  }
  do {
    exec {
      sql  = "DELETE FROM posts WHERE user_id IN (SELECT value::int FROM json_array_elements_text($1))"
      args = [jsonencode(iterator.keyset.batch[*].id)]
    }
    exec {
      sql  = "DELETE FROM users WHERE id IN (SELECT value::int FROM json_array_elements_text($1))"
      args = [jsonencode(iterator.keyset.batch[*].id)]
    }
    log {
      message = "batch ${self.index}: purged ${length(iterator.keyset.batch)} users"
    }
  }
  assert "all_gone" {
    sql = "SELECT count(*) = 0 FROM users WHERE active = false"
  }
  policy {
    schedule {
      every   = "200ms"
      timeout = "30m"
    }
    ramp {
      stage {
        size       = 100
        iterations = 3
      }
      stage { size = 1000 }
    }
  }
}
```

## Masking

A `mask` block redacts result columns before they leave Atlas. Use it whenever a report, export, or
the agent's own context would otherwise receive PII (`email`, `ssn`, `phone`, `*_enc`).

| Method | Effect |
|--------|--------|
| `REDACT` | Replaces the whole value with a token |
| `PARTIAL` | Keeps the ends, stars the middle: `keep`, `keep_left`, `keep_right` |
| `HASH` | Deterministic digest: HMAC-SHA256 with `salt`, SHA256 without. Same input, same token |
| `REPLACE` | Regex substitution over the value |

`columns` takes exact names or globs. A mask on a `query` masks that query; a script-level `mask` is
a default for every query. Declare a top-level `mask "<name>"` once and apply it with `use = [mask.<name>]`.
Masking protects the serialized output, not data at rest, so it does not replace restricting the query.

## Testing Scripts

Scripts are tested with the Atlas testing framework. A `script` command inside a `test "schema"`,
`test "migrate"`, `test "plan"`, or `test "script"` block runs matching scripts on the dev database
and asserts on the result.

```hcl
# scripts.test.hcl
test "script" "archive_user" {
  schema {
    url = "file://schema.sql"     # first: the case starts on an empty database
  }
  exec {
    sql = "INSERT INTO users (id, status) VALUES (1, 'active')"
  }
  script "exec" {
    file = "scripts.hcl"
    run  = "^archive_user$"
    vars = { user_id = 1 }
  }
  script "query" {
    file   = "scripts.hcl"
    run    = "^user_status$"
    vars   = { user_id = 1 }
    output = "archived"
  }
}
```

```hcl
# atlas.hcl
env "dev" {
  dev = "docker://postgres/17/dev"
  test {
    script {
      src = ["scripts.test.hcl"]
    }
  }
}
```

```bash
atlas script test --env dev
atlas script test --env dev --run archive_user
```

- `output` (exact) and `match` (regexp) compare everything the script prints; `error` expects the run to
  fail and matches the failure. The three are mutually exclusive.
- `as { role = "<role>" }` or `as { user = "<user>", password = "..." }` runs the script under another
  principal to prove it works with exactly those privileges.
- `test "schema"` runs against the desired schema on a clean dev database; `test "migrate"` migrates to a
  chosen version first; `test "script"` starts from an empty database, so load the schema with a
  `schema` block before any `exec`.

## Registry

`atlas script push` uploads a script source to the Atlas Registry. Every command that takes `--file`
then accepts `atlas://<repo>` in place of the local path, so a CronJob or CI job pulls scripts by name:

```bash
atlas script push --file "file://scripts" db-monitors
atlas script query --url "$URL" --file "atlas://db-monitors" --run '^pending_orders$' --quiet
```

Pushing again publishes a new version. In `atlas.hcl`, `script { src = "file://scripts" repo { name = "db-monitors" } }`
binds an env to its registry repo.

## Agent Workflow

1. Read `atlas.hcl`: find the env, its `script { src }` block, and whether a `test { script { src } }`
   block exists.
2. Pick the kind: `exec` for a bounded mutation, `query` for a read, `loop` for anything that must run in
   batches.
3. Write the script with guards: a `condition` that makes the run a no-op when there is nothing to do,
   `expect_rows` on writes with a known row count, a post-loop `assert` on the invariant the loop should
   leave behind.
4. Write a `test "script"` case that seeds data, runs the script, and asserts on the outcome. Run
   `atlas script test --env <name>`.
5. Run the script with an anchored `--run '^name$'`. Review the streaming report, then the closing summary.
6. For reports the agent will read, add `mask` blocks on PII columns and use `--quiet` or `--format '{{ json . }}'`.

## Key Rules

1. Anchor `--run` (`'^name$'`) so a prefix match does not run a second script.
2. Never put credentials in a script or on the command line. Use `--env` with `getenv()` in `atlas.hcl`.
3. Every `loop` must be bounded: an exhausting iterator, a `break`, or `policy.schedule` `limit`/`timeout`.
4. `http` calls cannot be rolled back. They require `tx { mode = NONE }` in an `exec` script and
   `policy { tx { mode = MANUAL } }` in a `loop`; set `expect_status`, or a 4xx/5xx reply counts as
   success.
5. Mask PII before it reaches a report, an export, or the agent's context.
6. A `NULL` from `condition` or `assert` is an error, not a graceful stop. Use `COALESCE`.
7. Use a migration, not a script, when the change belongs to the schema.

## Error Handling

| Error | Action |
|-------|--------|
| `No scripts found to run` | The `--run` regexp matched nothing. Check the script name and kind (`exec` vs `query` vs `loop`) |
| `exec kind requires at least one 'assert', 'check', or 'exec' block` | A body of only a bound `query` and `output`. Use `script "query"` for a read |
| `command requires 'atlas login'` or `available only to Atlas Pro users` | Run `atlas login` |
| `tx` command or `http` rejected | Set `tx { mode = NONE }` in an `exec` script, or `policy { tx { mode = MANUAL } }` in a loop |
| `expect_rows` mismatch | The write affected a different row count. The run aborted and rolled back; inspect the predicate |

## Documentation

- [Data Scripts](https://atlasgo.io/scripts)
- [Transactional mutations](https://atlasgo.io/scripts/exec)
- [Queries and reports](https://atlasgo.io/scripts/query)
- [Batched loops](https://atlasgo.io/scripts/loop)
- [Masking](https://atlasgo.io/scripts/masking)
- [Testing scripts](https://atlasgo.io/scripts/testing)
- [Scheduled monitors on Kubernetes](https://atlasgo.io/scripts/kubernetes)
