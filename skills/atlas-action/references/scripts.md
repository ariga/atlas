# Data Scripts Reference: `script/exec`, `script/query`, `script/loop`

Run Data Scripts from CI: scheduled purges and reports, one-off backfills behind an approval, invariant
checks after a deploy. Writing the scripts (guards, asserts, loops, masking) is in
`atlas/references/scripts.md`; testing them in CI is `script/test` (`references/testing.md`); publishing
them is `script/push` (`references/push.md`).

| Step | Runs | Use for |
|------|------|---------|
| `script/exec` | `atlas script exec` | A bounded mutation in one transaction, or an invariant check with only `assert` blocks |
| `script/query` | `atlas script query` | A read or report |
| `script/loop` | `atlas script loop` | Batched work: purges, backfills, re-keying |

- Scripts come from `files` (`file://scripts`, or `atlas://<repo>` after `script/push`) or from the env's
  `script { src }`. The target is `url` or the env's `url`.
- `match` selects scripts by name with an unanchored regexp, like the CLI's `--run`: anchor it
  (`'^gdpr_purge$'`), or `purge` also runs `purge_v2`. Without it, every script of that kind in the
  source runs.
- Each step runs only scripts of its own kind. A pattern that selects nothing passes after running zero
  scripts: check the log or the `report` output when a run must happen.
- Data Scripts need the login, on a plan that includes them.

## Results

| Outcome | Step |
|---------|------|
| Every selected script completed | Passes: `` `atlas script <kind>` completed successfully, N script(s) were executed `` |
| A `condition` was not met | Passes; logs `<script>: skipped, condition "<name>" was not met` |
| An `assert` or `check` failed, a `condition` errored, or a script failed | Fails, listing each failure with its error or diff |

- Outputs: `output` (the values of `output` blocks and the results of `query` blocks, one per line) and
  `report` (JSON with each script's messages, outputs, queries, conditions, asserts, and checks).
- Each selected script runs even when another fails; what a script already committed stays committed.
- Outputs and logs are visible to anyone who can read the run. Mask PII with `mask` blocks in the script
  before a report reaches them.

## Workflows

Scheduled purge, bounded per run so it fits the job's time limit; the schedule continues the work:

```yaml
name: GDPR Purge
on:
  schedule:
    - cron: '15 3 * * *'
jobs:
  purge:
    runs-on: ubuntu-latest
    environment: purge                      # default branch only, no reviewers: the purge user's URL
    steps:
      - uses: ariga/setup-atlas@v0
        with:
          cloud-token: ${{ secrets.ATLAS_CLOUD_TOKEN }}
      - uses: ariga/atlas-action/script/loop@v1
        with:
          files: atlas://my-scripts           # pushed on merge with script/push
          url: ${{ secrets.DATABASE_URL }}
          match: '^gdpr_purge$'
```

One-off backfill behind an approval: a `workflow_dispatch` job in a GitHub environment with required
reviewers, `script/exec` with an anchored `match`, and a `script/test` case for the script merged first.

Report to the job summary:

```yaml
- uses: ariga/atlas-action/script/query@v1
  id: report
  with:
    env: prod
    match: '^pending_orders$'
- run: echo "$REPORT" >> "$GITHUB_STEP_SUMMARY"
  env:
    REPORT: ${{ steps.report.outputs.output }}  # through env, never inlined into the script
```
