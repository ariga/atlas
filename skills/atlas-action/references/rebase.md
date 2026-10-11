# Rebase Reference: `migrate/autorebase` and Out-of-Order Migrations

Migration history is linear, and `atlas.sum` is a checksum of the whole directory in order. When two
branches each add a migration, the second one to merge conflicts on `atlas.sum`, and its file may sort
before the file that merged first. `migrate/autorebase` fixes that on the branch: it moves the branch's
new files after the base branch's files and rewrites `atlas.sum`.

## What the Step Does

1. Fetches the base: `base-branch` (default: the repository's default branch) from `remote` (default
   `origin`), or the commit `base-sha`.
2. Checks out the current branch, since CI runs on a detached commit.
3. Reads `atlas.sum` from `origin/<base>` and from `origin/<branch>`. The branch's new files are the ones
   only the branch has. None: `rebased=false`, done.
4. Merges the base into the branch. A clean merge means there is no conflict to fix: `rebased=false`,
   nothing pushed.
5. If the merge conflicts, every conflict must be in `atlas.sum`. Any other conflicted file fails the
   step with `conflict found in files other than <dir>/atlas.sum`, for a person to resolve.
6. Runs `atlas migrate hash` and `atlas migrate rebase` on the new files, so they sort after the base's
   files, commits `<dir>: rebase migration files` (which completes the merge), and pushes to the branch.
   Outputs: `rebased=true` and `latest_version`.

```yaml
name: Rebase Atlas Migrations
on:
  push:
    branches-ignore: [main]                 # every branch except the default one
jobs:
  migrate-auto-rebase:
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - uses: ariga/setup-atlas@v0
        with:
          cloud-token: ${{ secrets.ATLAS_CLOUD_TOKEN }}
      - uses: actions/checkout@v4
        with:
          token: ${{ secrets.PAT }}         # so the pushed commit triggers lint
          fetch-depth: 0                    # the step reads both branches' history
      - name: Configure git
        run: |
          git config --local user.email "github-actions[bot]@users.noreply.github.com"
          git config --local user.name "github-actions[bot]"
      - uses: ariga/atlas-action/migrate/autorebase@v1
        with:
          base-branch: main
          dir: file://migrations
```

- Trigger on `push`, not `pull_request`: GitHub does not run `pull_request` workflows for a pull request
  with a merge conflict, which is the state this step fixes.
- The job holds a token that can push, on a push to any branch, so anyone who can push a branch can
  read it: use a fine-grained token limited to this repository (`references/diff.md`, Steps That
  Commit).
- The rebase reorders files; it does not re-plan them. A migration written against the old order can
  conflict logically with the merged one, so `migrate/lint` must run on the rebased commit (hence the
  PAT). A lint failure goes back to the developer, who fixes the schema change and runs
  `atlas migrate hash`.
- The step passes only `dir` to `migrate hash` and `migrate rebase`, not the env. A directory in another
  format needs the format in the URL: `file://migrations?format=golang-migrate`.
- There is no rebase-only mode: the step merges the base into the branch, so the history gets a merge
  commit. `force-rebase: true` skips the merge, for the out-of-order flow below, where the base was
  already merged in.
- Rebase renames files without reading any revisions table. A branch file that some environment already
  applied (a shared dev database deployed from the branch) would run again under its new version: use the
  out-of-order flow for those environments.

Without the step, the developer resolves it locally: merge the base, take either side of `atlas.sum`,
`atlas migrate rebase <version>`, `atlas migrate hash`, and lint again (`atlas/references/versioned.md`).

## Out-of-Order Migrations

Teams that deploy branches to shared environments before merge (git-flow hotfixes, a `develop` branch
ahead of `main`) apply files out of order there. `migrate apply` refuses that by default
(`migration file ... was added out of order`, https://atlasgo.io/versioned/apply#non-linear-error). The
execution order decides:

| `exec-order` | A pending file older than the database's version |
|--------------|-----------------------------------------------|
| `linear` (default) | Fails the apply |
| `linear-skip` | Is skipped and never applied |
| `non-linear` | Is applied |

Keep `linear` for production, where history must match the directory. For an environment that receives
out-of-order files, apply with `non-linear`, rebase the files with `force-rebase`, then mark the rebased
versions applied with `migrate/set`, so later deploys do not run them again:

```yaml
- uses: ariga/atlas-action/migrate/apply@v1
  with:
    dir: file://migrations
    url: ${{ secrets.DEV_DATABASE_URL }}
    exec-order: non-linear
- uses: ariga/atlas-action/migrate/autorebase@v1
  id: rebase
  with:
    dir: file://migrations
    base-branch: main
    force-rebase: true
- uses: ariga/atlas-action/migrate/set@v1
  with:
    dir: file://migrations
    url: ${{ secrets.DEV_DATABASE_URL }}
    version: ${{ steps.rebase.outputs.latest_version }}
```

Run this only when the base was merged into the branch and the directory changed; the README of
ariga/atlas-action has a detection step for that. The job holds that database's URL, which the
branch's code can read: use the flow only for a shared development database. Guide:
https://atlasgo.io/faq/out-of-order-migrations.
