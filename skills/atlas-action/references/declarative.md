# Declarative Reference: `schema/plan`, `schema/plan/approve`, `schema/apply`

In the declarative workflow there are no migration files: each change is planned against the current
state, reviewed as a plan, and applied as reviewed. The pipeline:

| When | Step | Result |
|------|------|--------|
| Pull request | `schema/plan` | A `PENDING` plan in the registry, linted, with its SQL commented on the PR |
| Merge to the default branch | `schema/plan/approve`, then `schema/push` | The plan is approved; the new desired state is the repo's `latest` |
| Deploy | `schema/apply` | Finds the approved plan for the transition and runs it as reviewed |

```yaml
name: Atlas Declarative
on:
  pull_request:
    paths: ['schema/**', 'atlas.hcl']
  push:
    branches: [main]
    paths: ['schema/**', 'atlas.hcl']
permissions:
  contents: read
jobs:
  plan:
    if: github.event_name == 'pull_request'
    runs-on: ubuntu-latest
    permissions:
      contents: read
      pull-requests: write                  # only the plan job comments
    steps:
      - uses: actions/checkout@v4
      - uses: ariga/setup-atlas@v0
        with:
          cloud-token: ${{ secrets.ATLAS_CLOUD_TOKEN }}
      - uses: ariga/atlas-action/schema/plan@v1
        with:
          env: ci
        env:
          GITHUB_TOKEN: ${{ github.token }}
  approve-push:
    if: github.event_name == 'push'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: ariga/setup-atlas@v0
        with:
          cloud-token: ${{ secrets.ATLAS_CLOUD_TOKEN }}
      - uses: ariga/atlas-action/schema/plan/approve@v1
        with:
          env: ci
      - uses: ariga/atlas-action/schema/push@v1
        with:
          env: ci
```

```hcl
env "ci" {
  # No url: plans start from the registry's last state, so pull request jobs need no database URL.
  dev = "docker://postgres/17/dev?search_path=public"
  schema {
    src = "file://schema"
    repo {
      name = "app"
    }
  }
}
```

## One Rule for All Three Steps

`schema/plan`, `schema/plan/approve`, and `schema/apply` must see the same current state. Either all
three read the same database (the env's `url`, or the `from` input), or none of them does and all start
from the registry's last known state. A mismatch fails at deploy time with
`The plan "From" hash does not match the current state hash`. A manual change on the target between
plan and apply does the same: re-plan. Starting from the registry's state keeps database URLs out of
pull request jobs, where the branch's code could read them. When the plan must read the real database,
pass `from` a read-only user's URL, never the deploy user's (`references/setup.md`).

## `schema/plan`

| Situation | Step |
|-----------|------|
| No plan for this transition | Creates `pr-<number>-<hash>` (outside a PR: `ref-<commit>-<hash>`, or `name`), lints it, comments it |
| One pending plan for this transition | Lints it again and updates the comment; does not create another |
| Several pending plans for this transition | Fails: `found multiple schema plans, please approve or delete the existing plans` |
| Nothing to change | Logs that the current state is synced with the desired state; passes, no comment |
| The plan has lint errors | Comments, then fails: `` `atlas schema plan` completed with lint errors `` |

- Outputs: `plan` (`atlas://<repo>/plans/<name>`), `link`, `status`.
- The comment shows the plan's SQL and lint results; on GitHub it also lists each changed object with
  its diff. Large plans are cut to fit the comment; the link has the whole plan.
- Directives in the pull request description, one per line, are written into the plan when it is
  created: `/atlas:nolint destructive`, `/atlas:txmode none`. The plan keeps the directives it was
  created with, so after editing the description, delete the pending plan in Atlas Cloud and rerun.
- Stale pending plans (an abandoned branch with the same transition) cause the "multiple plans" failure:
  delete them in Atlas Cloud.

## `schema/plan/approve`

Approves the plan that the pull request created. Runs on push to the default branch; on a pull request it
fails with `the action should be run in a branch context`.

- Without `plan`, it finds the one pending plan for the transition between the current state and the
  merged desired state. None found: it passes and logs `No pending schema plan found`, and a later
  `schema/apply` with changes fails, so rerun `schema/plan`. Several found: it fails.
- With `plan: atlas://<repo>/plans/<name>` it approves that plan.
- The bot token needs permission to approve on the repo when protected flows are on.
- Approving in CI is the team's review policy written down: the pull request review is the approval.
  The agent never approves a plan on the user's behalf: not in Atlas Cloud, and not by adding an
  approve step the user did not ask for.

## `schema/apply`

| Inputs | Behavior |
|--------|----------|
| none of the below | Runs the approved plan for the transition if one exists; otherwise plans and applies per the env's review policy, and fails when that policy needs an approval it does not have |
| `plan: atlas://app/plans/<name>` | Runs that plan |
| `auto-approve: true` | Applies without review: only where the pipeline is the approver, such as preview or disposable databases |
| `dry-run: true` | Prints the SQL, changes nothing |
| `lint-review` (+ `wait-timeout`, `wait-interval`) | Ad-hoc approval, below |

Ad-hoc approval, for changes that reach a target without a reviewed plan (a manual change drifted the
database, or the change skipped CI): with `lint-review: ALWAYS`, the step creates a plan for the
transition, prints its link, waits up to `wait-timeout` for someone to approve it in Atlas Cloud, then
applies it. With `ERROR` or `WARNING`, it creates the plan only when lint finds errors (or any finding),
and applies directly otherwise; set the same policy in the env's `lint { review = ... }`. Without
`wait-timeout`, the step fails right after printing the link: rerun it after approval. A pending plan
that already exists for the transition is waited on instead of creating another.

- Give `schema/apply` the dev database too (the env's `dev`, or `dev-url`). Without one, the plan can
  hold incorrect or unnecessary changes, and every run proposes them again
  (https://atlasgo.io/concepts/dev-database#diffing).
- Each poll re-inspects the target: keep `wait-interval` at `30s` or more for long waits.
- An env with several targets applies to each and reports each; the first failure fails the step and sets
  `error`.
- Engines without transactional DDL: `tx-mode: none`.
- Guides: https://atlasgo.io/declarative/plan,
  https://atlasgo.io/integrations/github-actions/pre-approval,
  https://atlasgo.io/integrations/github-actions/ad-hoc-approval.

## Promotion

Push the desired schema once, on merge, then apply that version to staging and to production. In the
production env, a `check "schema_apply"` block with a `deny` rule refuses a version staging has not
run, with the staging version read by a `cloud_databases` data source
(https://atlasgo.io/guides/environment-promotion-declarative).
