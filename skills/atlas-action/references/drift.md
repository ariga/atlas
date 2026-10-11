# Drift Reference: `migrate/drift` and `monitor/schema`

Two scheduled steps that watch databases after deployment. `migrate/drift` answers "does this database
still match its migration history?" and fails when it does not. `monitor/schema` sends the live schema to
Atlas Cloud Schema Monitoring, which keeps the history and alerts. The CLI side of drift, and when to use
which tool, is in `atlas/references/drift.md`.

## `migrate/drift`

Runs `atlas migrate drift`: reads the target's revisions table, computes the state the directory defines
at the applied version, inspects the database, and diffs the two. Files after the applied version are
pending, not drift. It never changes the database.

```yaml
name: Drift Detection
on:
  schedule:
    - cron: '0 * * * *'                     # hourly
  workflow_dispatch:
jobs:
  drift:
    runs-on: ubuntu-latest
    environment: monitoring                 # default branch only, no reviewers: a read-only user's URL
    steps:
      - uses: ariga/setup-atlas@v0
        with:
          cloud-token: ${{ secrets.ATLAS_CLOUD_TOKEN }}
      - uses: ariga/atlas-action/migrate/drift@v1
        id: drift
        with:
          dir: atlas://app
          url: ${{ secrets.DATABASE_URL }}
```

- Expected state: from the registry when `dir` is `atlas://` or the env names a registry repo
  (`migration { repo { name } }`); otherwise the step replays a `file://` directory on `dev-url`, which
  then is required (and needs a checkout).
- With an `env` that has several targets, it checks each one and writes one report per database.
- Fails when any database drifted, and when any database could not be checked, since its state is then
  unknown. A scheduled run that fails is the alert: GitHub notifies on failed scheduled runs, or add a
  step with `if: failure()` that posts to the team's channel.
- The log and the job summary list each drifted object as `extra` (only in the database), `missing`
  (only in the expected state), or `modified`, with the SQL that reproduces the drift: the change made
  outside the migrations, not its fix. Outputs: `drifted` (`true` or `false`) and `report`, a JSON
  array with one entry per database.
- Each `report` entry has a `Fingerprint` that stays the same while the drift is unchanged: key alerts
  on it to send one alert per drift, not one per run. It is not a step output.
- The step takes the same lock as `migrate/apply`, so a deploy in progress is not reported as drift. A
  database with no migration history fails the check, like an unreachable one.
- Needs the login. Use a read-only database user: the step only reads.
- `exclude` skips objects that something other than Atlas manages, the same patterns as
  `schema inspect --exclude`.

### Drift at Deploy Time

To stop a deploy on a drifted target, add `check "migrate_apply" { drift { on_error = FAIL } }` to the
deploy env (`references/apply.md`). In `migrate/apply`, that check fails the step without printing the
difference; run `migrate/drift` against the same target to see it. Fixing drift: bring the database
back with a corrective migration, or capture the change in the directory, then redeploy; never by
`migrate/set` alone.

## `monitor/schema`

Inspects a database and pushes a snapshot to Atlas Cloud Schema Monitoring
(https://atlasgo.io/monitoring), which records each change to the schema over time.

```yaml
- uses: ariga/setup-atlas@v0                        # installs the CLI; no token needed here
- uses: ariga/atlas-action/monitor/schema@v1
  with:
    cloud-token: ${{ secrets.ATLAS_CLOUD_TOKEN }}   # required: this step logs in by itself
    url: ${{ secrets.DATABASE_URL }}
    schemas: |
      app
      auth
```

- Give either `url` or `config`/`env`, not both:
  `only one of the inputs 'config' or 'url' must be given`.
- It pushes a new snapshot only when the schema changed since the last one.
- `slug` keeps the same identity in Atlas Cloud when the database URL changes (a renamed host, a moved
  instance).
- `collect-stats` (default `true` on GitHub and GitLab) also sends the database's table statistics
  (sizes, row counts, index usage) on each run; on Bitbucket, set it explicitly. The CircleCI orb and the
  Azure DevOps task send none.
- `include`, `exclude`, and `schemas` narrow what is inspected.
- Output: `url` of the schema in Atlas Cloud.
- Run it on a schedule, in a job that names an environment limited to the default branch, since it
  holds a database URL (`references/setup.md`). The GitLab component runs only on schedules or manually.
