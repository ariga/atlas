# Testing Reference: `migrate/test`, `schema/test`, `script/test`

Run the Atlas test framework on every pull request, next to lint. Each step runs one CLI command on the
dev database and fails when a case fails. Writing the cases (`test "schema"`, `test "migrate"`,
`test "script"`, their commands, and the env's `test` block) is in `atlas/references/testing.md`.

| Step | Runs | Starts from | Inputs beyond the common ones |
|------|------|-------------|-------------------------------|
| `migrate/test` | `atlas migrate test` | An empty dev database migrated to each case's version | `dir` (`file://` or `atlas://`), `paths`, `run` |
| `schema/test` | `atlas schema test` | The desired schema created on the dev database | `url` (desired schema), `paths`, `run` |
| `script/test` | `atlas script test` | An empty dev database, or the schema the script's `schema` block loads | `paths`, `run` |

```yaml
- uses: ariga/atlas-action/migrate/test@v1
  with:
    env: ci                                 # the env's test { migrate { src } } lists the files
- uses: ariga/atlas-action/schema/test@v1
  with:
    env: ci
    run: '^test_.*'                         # optional: only cases whose name matches
```

- `paths` names the test files or directories to run, one per line; without it, the env's `test`
  block decides.
- The result is in the job log only: no pull request comment, no outputs. A failed case fails the step
  with `` `atlas <command> test` completed with errors `` and the case's output.
- Testing needs the login, and a dev database like lint's (`references/setup.md`).
- On PostgreSQL, `template = true` on the dev database's `docker` block runs each test group on a
  fresh copy of the desired schema, which keeps a large suite fast
  (https://atlasgo.io/guides/postgres/schema-test-template-databases).
- Plans have their own tests (`atlas schema plan test`), which no action wraps: run the CLI in a step
  after `setup-atlas` if the project uses them.
