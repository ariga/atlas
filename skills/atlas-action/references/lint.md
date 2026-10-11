# Lint Reference: `migrate/lint` and `schema/lint`

The pull request gate. `migrate/lint` checks the migration files a branch adds, and `schema/lint` checks
the desired schema as a whole. Both comment on the pull request and fail on error-level findings. The
analyzers, policy, and custom rules are the CLI's: `atlas/references/lint.md`.

## `migrate/lint`

Runs `atlas migrate lint` with `--base atlas://<dir-name>?tag=<tag>` and `-w`, so the report is uploaded
to Atlas Cloud, then posts the report on the pull request. It replays the whole directory on the dev
database to compute each file's starting state, and lints only the files that are new.

```yaml
name: Atlas Lint
on:
  pull_request:
    paths: ['migrations/**', 'schema/**', 'atlas.hcl']
permissions:
  contents: read
  pull-requests: write
jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: ariga/setup-atlas@v0
        with:
          cloud-token: ${{ secrets.ATLAS_CLOUD_TOKEN }}
      - uses: ariga/atlas-action/migrate/lint@v1
        id: lint
        with:
          env: ci
          dir-name: app                         # required: the registry repo slug
        env:
          GITHUB_TOKEN: ${{ github.token }}
```

### Which Files Are New

| Setup | New files | Needs |
|-------|-----------|-------|
| Default | Files that the registry repo's `latest` tag does not have | `migrate/push` on every merge to the default branch |
| `tag: <tag>` | Files that tag does not have, such as the tag production runs | That tag pushed |

- A repo that was never pushed has no files, so every file is new and an old file that breaks a policy
  fails every pull request. Push the directory once (the first `migrate/push`, or `create-repo` then a
  push) and keep pushing on merge.
- Comparing with the registry also catches non-linear changes: a branch that edits, deletes, or reorders
  a file the registry already has. They are warnings by default; the policy can make them errors
  (https://atlasgo.io/lint/analyzers#non-linear-changes). `atlas.sum` alone does not catch them.
- The action always passes `--base atlas://<dir-name>?tag=<tag>`, and the CLI takes only one of
  `--base`, `--latest`, and `--git-base` (`--latest, --git-base, and --base are mutually exclusive`).
  Keep `latest` and `git { base }` out of the env's `lint` block, and leave the step's `git-base` and
  `git-dir` inputs unset.
- `dir-name` is required even when the env's `migration { repo { name } }` names the repo; without it
  the step fails with `atlasaction: missing required parameter dir-name`. It is the repo's slug, the
  part of its URL after `atlas://`, in lowercase; a display name fails with
  `slug must be lowercase alphanumeric and may contain /.-_`.

### Without the Registry

The step needs the login and a registry repo. A project that does not push to the registry lints in a
plain step, against the git base branch, and gets no pull request comment:

```yaml
- uses: actions/checkout@v4
  with:
    fetch-depth: 0                          # the base branch must exist in the checkout
- uses: ariga/setup-atlas@v0
  with:
    cloud-token: ${{ secrets.ATLAS_CLOUD_TOKEN }}
- run: atlas migrate lint --env ci --git-base origin/main
```

Name a ref the checkout has: a bare `main` that was never fetched fails with
`Error: git diff: exit status 128`. Git-based detection sees committed files only. Images without
`git`, such as `arigaio/atlas`, fail with `lookup git: exec: "git": executable file not found`.

### What It Reports

- One comment per `dir-name` on the pull request, updated on every run, with each file's findings and, on
  GitHub, the objects each file changes with their diffs.
- File annotations on the changed lines, and suggestions (review comments with a fix) on files the pull
  request touches.
- The job summary, and the report in Atlas Cloud. Its link is the `report-url` output and appears in the
  failure message. If the upload fails, the step warns
  `` `atlas migrate lint` report was not uploaded to Atlas Cloud `` and still decides on the findings.

### When It Fails

| Result | Step |
|--------|------|
| Error-level findings, per the env's `lint` policy | Fails: `` `atlas migrate lint` completed with errors, see report: <url> `` |
| Warnings only | Passes, with the findings in the comment |
| Lint could not run (dev database, login, config, `checksum mismatch`) | Fails with the CLI error |

Which findings are errors is the env's `lint` policy (`atlas/references/lint.md`). Destructive changes
are errors by default; to fail on backward-incompatible or data-dependent changes too, set
`incompatible { error = true }` or `data_depend { error = true }` in the env's `lint` block, and select
that env with the step's `env` input.

An `ownership "github"` policy in the env's `lint` block reports changes to objects the author may not
change (`OW101`, `OW102`); a change no rule matches is denied unless the block sets `default = ALLOW`.
The step then needs `GITHUB_TOKEN` in its `env`, which resolves team membership (the built-in
`github.token` works for public and internal repositories), `fetch-depth: 0` on the checkout, and
`env` naming the env with the policy (https://atlasgo.io/guides/schema-ownership).

Fixing findings: edit an unapplied file, then `atlas migrate hash`; for an applied file, add a
corrective migration. An intended drop passes with `-- atlas:nolint DS102` (or the finding's code) on
the line above the statement, only with the user's approval; `destructive { force = true }` in the
policy disables that escape. Lint time grows with the directory; a checkpoint
(`atlas migrate checkpoint`) shortens it.

## `schema/lint`

Runs `atlas schema lint` on the desired schema: the policy and custom rules applied to every object, not
only to what changed. Use it for conventions that must always hold, such as row-level security on every
table or an index on every foreign key (`atlas-postgres` custom rules).

```yaml
- uses: ariga/atlas-action/schema/lint@v1
  with:
    env: ci                                 # url: file://schema.hcl (or env://src); schema: public
  env:
    GITHUB_TOKEN: ${{ github.token }}
```

- Comments on the pull request when it finds issues, and deletes its comment once a later run is clean.
- Annotates the findings and writes the job summary.
- Fails when any finding is an error; warnings pass. On an existing schema it fails on every old
  violation: run it locally first to see the backlog, and gate pull requests with `migrate/lint`,
  which reports only what the change introduces.
- Some findings appear only when a change introduces them, such as `PG110` (data alignment):
  `schema/lint` does not report them; `migrate/lint` and `schema/plan` do.
