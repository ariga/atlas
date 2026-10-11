# Generation Reference: Missing Migrations, `migrate/diff`, and `migrate/hash`

When a pull request changes the schema without its migration, or edits a migration without rehashing
`atlas.sum`, CI can fail the pull request and leave the fix to the developer, or commit the fix to the
branch. Failing is the default. The steps that commit, `migrate/diff` and `migrate/hash`, push with a
token that can write to the repository, in a job that runs the branch's workflow file, `atlas.hcl`, and
schema programs, and `actions/checkout` keeps that token in the job's git config: anyone who can push a
branch can read it. Use them only where everyone who can push a branch may hold that token (Steps That
Commit, below).

## Failing the Pull Request

A step in the pull request job, after `setup-atlas`; it needs no token that can push to the repository:

```yaml
- run: |
    atlas migrate diff --env ci             # the env's src is the desired state
    if [ -n "$(git status --porcelain)" ]; then
      echo "missing migration: run 'atlas migrate diff --env local <name>' and commit it"
      exit 1
    fi
```

The check is the one in https://atlasgo.io/faq/desired-state-drift. `migrate/lint` already fails a
directory whose `atlas.sum` is out of date, so a stale `atlas.sum` needs no extra step: the developer
runs `atlas migrate hash` and commits the result.

## `migrate/diff`

Plans `atlas migrate diff` from the desired state (`to`, or the env's `src` or external schema such as an
ORM) against the migration directory, on the dev database.

| Result | Step |
|--------|------|
| The directory already matches the desired state | Logs `The migration directory is synced with the desired state, no changes to be made`, passes |
| A change is missing | Writes the new file into `dir`, runs `atlas migrate hash`, commits `<dir>: add new migration file`, pushes to the branch, passes |
| Planning failed (dev database, login, `checksum mismatch`) | Fails with the CLI error |

```yaml
name: Generate Migrations
on:
  pull_request:
    paths: ['schema/**', 'migrations/**', 'atlas.hcl']
jobs:
  migrate-diff:
    runs-on: ubuntu-latest
    permissions:
      contents: write
      pull-requests: write
    env:
      GITHUB_TOKEN: ${{ github.token }}
    steps:
      - uses: ariga/setup-atlas@v0
        with:
          cloud-token: ${{ secrets.ATLAS_CLOUD_TOKEN }}
      - uses: actions/checkout@v4
        with:
          token: ${{ secrets.PAT }}             # fine-grained, this repository only (Steps That Commit)
          fetch-depth: 0
      - name: Configure git
        run: |
          git config --local user.email "github-actions[bot]@users.noreply.github.com"
          git config --local user.name "github-actions[bot]"
      - uses: ariga/atlas-action/migrate/diff@v1
        with:
          env: ci                               # or dir, to, and dev-url inputs
```

- It never fails because a migration was missing: it adds it.
- The generated file takes the CLI's default name. Rename it in the pull request before merge if the
  team names migrations; run `atlas migrate hash` after renaming.
- It needs the login: logged out, the plan leaves out views, functions, triggers, and other objects.

## `migrate/hash`

Runs `atlas migrate hash` with the step's `env`, `config`, and `dir`. When `atlas.sum` changed, it
stages every change under the working directory (`git add .`), commits `` run `atlas migrate hash` ``,
and pushes to the branch; otherwise it passes without a commit.

```yaml
- uses: ariga/atlas-action/migrate/hash@v1
  with:
    working-directory: db                   # see the note on git add below
    dir: file://migrations
```

- `git add .` takes any file the job created under the working directory. On GitHub the action
  downloads its own binary, `atlas-action`, into the workspace root, so run the step with
  `working-directory` set to the unit's directory, or add `atlas-action` to `.gitignore`.
- Rehashing accepts the current content of every file. Never rehash to get past `checksum mismatch` on
  a file that was already applied somewhere: that hides an edit of an applied migration. Registry-based
  lint reports such an edit as a non-linear change.

## Steps That Commit

`migrate/diff`, `migrate/hash`, and `migrate/autorebase` push commits to the branch:

1. `permissions: contents: write`.
2. Checkout with `fetch-depth: 0` and `token: ${{ secrets.PAT }}`: a fine-grained token or a GitHub App
   token, limited to this repository's contents and pull requests, from an identity that cannot bypass
   branch protection. Commits pushed with the default `github.token` do not start other workflows, so
   lint would never run on them. The branch's code can read this token: never use a personal token with
   wider access.
3. A git identity before the step (`git config user.name` and `user.email`).
4. A branch the token can push to: pull requests from forks get no secrets, protected branches reject
   the push, and the step fails with `failed to push changes`.
