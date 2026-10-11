# Registry Reference: `migrate/push`, `schema/push`, `script/push`, `create-repo`

The Atlas Registry holds the reviewed artifacts: migration directories, desired schemas, and Data
Scripts. Lint compares pull requests with it, deployments read from it, and drift checks and approvals
use it. CI pushes on every merge to the default branch; deployments never read from a branch.

## The Repo Must Exist First

The first `migrate/push` or `schema/push` creates the repo. To create it on its own, run `create-repo`
once, after the user approves the name:

```yaml
name: Create Atlas Registry Repo
on: workflow_dispatch
jobs:
  create-repo:
    runs-on: ubuntu-latest
    steps:
      - uses: ariga/setup-atlas@v0
        with:
          cloud-token: ${{ secrets.ATLAS_CLOUD_TOKEN }}
      - uses: ariga/atlas-action/create-repo@v1
        with:
          name: app
          type: migration_directory          # or schema
          driver: postgres
```

- `type` takes `migration_directory` (or `m`) or `schema` (or `s`). The action metadata lists
  `migration`, which the CLI rejects. The GitLab component accepts only the listed values, so there,
  create a migration repo with the first push instead. The orb and the Azure task have no `create-repo`
  step.
- The step fails when the repo exists, so it runs once, not on every push.
- `description` is accepted but not sent in this version. The `url` output links to the repo.

## `migrate/push`

Pushes the directory twice by default: once as the `latest` tag, then tagged with the commit SHA (or
`tag`). `latest: false` pushes only the second.

```yaml
name: Atlas Push
on:
  push:
    branches: [main]
    paths: ['migrations/**', 'atlas.hcl']
jobs:
  push:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: ariga/setup-atlas@v0
        with:
          cloud-token: ${{ secrets.ATLAS_CLOUD_TOKEN }}
      - uses: ariga/atlas-action/migrate/push@v1
        id: push
        with:
          env: ci
          dir-name: app                     # or the env's migration { repo { name } }
```

- Push from the default branch only. A push from a pull request moves `latest` to unreviewed files:
  every other pull request then lints against them, and deployments of `atlas://app` pick them up.
- Push replays the directory on the dev database to compute each version's state, so the dev database
  must be able to run every file. When the dev image or the `docker` block's `baseline` already holds
  an object the directory creates, such as the PostGIS extension, the push fails with
  `extension "postgis" already exists`. Keep only what the directory does not manage in the baseline,
  and use a database-scoped dev URL when the directory manages extensions
  (`atlas-postgres/references/dev-database.md`).
- On Azure DevOps and Bitbucket, input defaults do not apply: set `latest: true` (Bitbucket:
  `ATLAS_INPUT_LATEST: "true"`), or the `latest` tag stays where it was. The same holds for the CircleCI
  orb's `schema_push`.
- Keep the `latest` push. The registry stores the expected state that drift checks read only for
  untagged pushes, so a directory pushed only with a tag makes the deploy-time drift check print
  `no state found for version ...` and skip.
- The `url` output links to the pushed version. It is set, though the action metadata does not declare
  it, so on GitLab, which exports only declared outputs, it is not available to later jobs.
- A `backup` list in the env's `repo` block replicates each push to storage such as S3: the push
  succeeds when Atlas Cloud or one backup accepts it, and `atlas://` reads fall back to the backups in
  order (https://atlasgo.io/guides/backup-continuity-migration-versioned).

Registry URLs for the steps that read it:

| URL | Version |
|-----|---------|
| `atlas://app` | The `latest` tag: whatever the last push from the default branch was |
| `atlas://app?tag=<commit SHA>` | The version pushed from that commit: pin deployments to it |
| `atlas://app?version=<migration version>` | The directory as of that migration version |

## `schema/push`

Pushes the desired schema, tagged the same way (`latest` and the commit SHA, or `tag`). `schema-name`
names the repo, or the env's `schema { repo { name } }` does. `version` and `description` label the
version. Outputs: `link`, `slug`, `url`. In the declarative pipeline it runs on merge after
`schema/plan/approve` (`references/declarative.md`).

## `script/push`

Pushes Data Scripts (`files`, or the env's `script { src }`) to a script repo named by `script-name`.
Outputs: `name`, `files` (how many), `link`. Production runs then read the scripts from the registry
(`files: atlas://my-scripts`) instead of the checkout (`references/scripts.md`). A warning
`failed to back up the scripts to ...` is about an optional replica and does not fail the push.
