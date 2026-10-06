# Stage 4: Deploy

Goal: staging deployments of the unit apply versions from the Atlas Registry, every run is recorded in
Atlas Cloud, and the old tool no longer deploys the unit to staging. This stage writes staging
configuration only: no production job, manifest, overlay, or module. Production is stage 5, in a
separate PR when the user asks for it.

## 4.1 Pick the Target

Detect it from the scan; ask only when several rows apply or none does. Record the answer as `deploy`
in `.atlas-onboarding.json`.

| Found | Target |
|-------|--------|
| CI deploys the application (a deploy job, a release workflow) | CI deploy job (4.4) |
| A release or pre-deploy command, or an entrypoint that runs migrations (Render, Heroku, Fly.io, ECS) | Platform release step (4.5) |
| Argo CD `Application` | Atlas Operator with Argo CD (4.6) |
| Flux `Kustomization` | Atlas Operator with Flux (4.7) |
| Terraform with the `ariga/atlas` provider, or the user wants schema changes in Terraform plans | Terraform (4.8) |
| Terraform that only provisions the database instance | Not a migration target: use the row for how the application deploys |
| A Helm `pre-upgrade` migration hook | The Atlas Operator (4.6 or 4.7); Helm hooks for migrations are deprecated |

## 4.2 Plan the Cutover

The old tool stops deploying the unit to an environment in the same change that starts Atlas for it.

1. Bring the code up to date. If the old tool applied changes since stage 1, add them to the desired
   state (versioned units: generate them with `atlas migrate diff --env local <name>`), and rerun the
   stage 1 and stage 2 checks.
2. Find each existing database's baseline: the version of the baseline file. For an imported history,
   it is the Atlas version of the newest file the old tool's history table records as successful in
   that environment. Ask the user for the tool's status output, such as `flyway info` or
   `goose status`, or for a query on its history table. Environments can differ: never copy staging's
   baseline to production. Set it as stage 2.2 describes: `baseline` in the env (4.3), or a one-time
   `--baseline` run by the user.
3. Check each existing database against its baseline before its first deploy, with the `staging` env
   from 4.3 and `STAGING_DATABASE_URL` set to a read-only role:

   ```bash
   atlas schema apply --env staging --to "file://migrations?version=<baseline>" --dry-run
   ```

   `Schema is synced, no changes to be made` means the database is at the baseline. Planned statements
   mean the environment drifted from it: show them and let the user decide.
4. Retire the old tool for staging:
   - If it deploys only this unit, remove its staging step in the deploy PR. When environments share
     one artifact (one image, one entrypoint), switch by environment instead, with a variable only
     staging sets. Never remove production's step before production deploys with Atlas.
   - If it also deploys other units, keep it, and stop new files for this unit in the old directory
     (a CODEOWNERS entry, or a CI check). Two tools may deploy to one database only when they change
     different schemas.
   - When the Atlas deploy and the removal are in different repositories, the user merges the Atlas
     deploy first, then the removal, with no schema change in between.
5. ORM migration tools (Django, Alembic, Prisma Migrate) stop deploying too. Ask the user what
   developers keep running them for, such as building test databases.

## 4.3 The Staging Env

Add a `staging` env to the unit's `atlas.hcl`, or extend the one stage 1 added for a remote database.
For a cloud sign-in (AWS IAM, Microsoft Entra ID, GCP IAM) or a password from a secret store, build
`url` as in stage 1, Remote Databases:

```hcl
env "staging" {
  url     = urlqueryset(getenv("STAGING_DATABASE_URL"), "search_path", "identity")
  dev     = local.dev
  exclude = local.exclude
  schema {
    src = local.src
  }
  migration {
    dir      = "atlas://identity"
    baseline = "<baseline version>"   # only when the staging database existed before Atlas
  }
  check "migrate_apply" {
    drift {
      on_error = CONTINUE             # report drift; switch to FAIL once staging is clean
    }
  }
}
```

- Declarative units drop the `migration` block and deploy the desired state from the registry: in
  this env, `src = "atlas://<repo>"` replaces `src = local.src`.
- The env name is the environment name in Atlas Cloud. Deployments from `atlas://` directories are
  reported automatically.
- Two roles, one env: locally, `STAGING_DATABASE_URL` holds a read-only role, enough for the checks and
  dry runs in this stage. Deployments use a deploy role: the one the old tool deployed with, or one that
  owns the unit's objects. Only the CI secret or the cluster Secret holds it, never the agent's shell.
- Database-scoped units that share a database with other units set `revisions_schema` and `lock_name`
  in the `migration` block to `atlas_<unit>`, so each unit keeps its own history and lock. Schema-scoped
  units keep the history table in their own schema.
- The drift check compares the database with the registry's state for its last applied version
  before anything runs; it starts after the first deploy (`atlas/references/drift.md`). Only untagged
  pushes store that state, such as the `latest` push that `migrate/push` makes by default. For a
  version pushed only with a tag, the check prints `no state found for version ...` and is skipped,
  even with `FAIL`. Keep `latest` on in the push job; a CLI pipeline pushes `<repo>` as well as
  `<repo>:<sha>`.
- Versioned deploys have no approval step of their own. Review happens on the PR, and `check
  "migrate_apply"` rules, the drift check, and promotion (stage 5) guard the apply.

## 4.4 CI Deploy Job

GitHub Actions, added to the stage 3 workflow after the push job:

```yaml
  deploy-staging:
    needs: push
    runs-on: ubuntu-latest                   # copy runs-on and network setup from the old migration job
    environment: staging
    env:
      STAGING_DATABASE_URL: ${{ secrets.STAGING_DATABASE_URL }}
    steps:
      - uses: actions/checkout@v4
      - uses: ariga/setup-atlas@v0
        with:
          cloud-token: ${{ secrets.ATLAS_CLOUD_TOKEN }}
      - uses: ariga/atlas-action/migrate/apply@v1
        with:
          working-directory: <unit dir>
          env: staging
          dir: atlas://<repo>?tag=${{ github.sha }}   # the version this merge pushed
```

- The runner must reach the staging database. Hosted runners cannot reach a private database: copy
  the runner and network setup of the job that runs the old tool today.
- The job must finish before the application deploys. Add it to the application deploy job's `needs`.
- The user stores `STAGING_DATABASE_URL` as a secret, with the deploy user.
- GitLab CI: the `migrate-apply` component, as its own job with `rules` that limit it to the default
  branch, `env: staging`, and the runner tags of the application's staging deploy.
- Other CI systems: install the CLI, export the bot token as `ATLAS_TOKEN`, and run
  `atlas migrate apply --env staging`.
- Declarative units: `ariga/atlas-action/schema/apply@v1` after the approve and push steps of
  `atlas-plan.yaml`. Plan, approve, and apply must see the same starting state
  (`atlas/references/cicd.md`, Declarative Pipeline).

## 4.5 Platform Release Step

On platforms that run a command before each release (Render's pre-deploy command, Heroku's release
phase, Fly.io's `release_command`, an ECS task before the service update), run
`atlas migrate apply --env staging` there instead of the old tool, not in the container entrypoint
(https://atlasgo.io/guides/deploying/fly-io). The image needs the `atlas` binary and the unit's
`atlas.hcl`. The user sets `ATLAS_TOKEN` and `STAGING_DATABASE_URL` in the platform's staging
environment. Release only after the stage 3 push job passed, so the registry has the version.

## 4.6 Atlas Operator with Argo CD

The Operator must be installed in the cluster (https://atlasgo.io/integrations/kubernetes/install).
That is usually the platform team's change; ask before adding it. Ask for a local checkout of the GitOps
repository, and write to its staging overlay only, never to a shared base. Reference Secrets that the
cluster's secret manager creates, by name, and never write secret values to Git:

```yaml
apiVersion: db.atlasgo.io/v1alpha1
kind: AtlasMigration
metadata:
  name: <unit>
  annotations:
    argocd.argoproj.io/sync-wave: "1"       # before the application's resources (wave "2")
spec:
  envName: staging                          # environment name in Atlas Cloud
  urlFrom:
    secretKeyRef:
      name: <unit>-db-credentials           # deploy user; the URL carries the unit's scope (search_path)
      key: url
  cloud:
    tokenFrom:
      secretKeyRef:
        name: atlas-token
        key: token
  dir:
    remote:
      name: <repo>
      tag: "<commit sha>"
  baseline: "<baseline version>"            # only for databases that existed before Atlas
```

- Health: Argo CD 2.10 and later ship health checks for `AtlasMigration` and `AtlasSchema`. Add the
  custom check from https://atlasgo.io/guides/deploying/k8s-argo only on older versions. A failed
  migration, or a plan waiting for approval, reports `Degraded` and holds back later waves.
- Updating `tag` is the deployment; a new registry version does not trigger the Operator. Argo CD Image
  Updater changes only container images, so add a CI job that commits the new tag to the GitOps
  repository after the push job, with a token the user stores, limited to that repository.
- Sync waves order resources within one sync of one Application. If the application's image is bumped
  in a separate commit or sync, the app can run before its migration: bump both in the same commit, or
  require backward-compatible migrations (expand, deploy, contract) and add that rule to `AGENTS.md`.
- Database-scoped units that share a database set `revisionsSchema: atlas_<unit>`.
- Declarative units use `AtlasSchema` with `schema.url: atlas://<repo>?tag=<tag>` and a review policy
  (`policy.lint.review: ERROR`, `WARNING`, or `ALWAYS`). A plan that needs review waits for a human to
  approve it in Atlas Cloud (https://atlasgo.io/integrations/kubernetes/declarative).
- Optional drift monitoring: an `AtlasDriftCheck` next to the `AtlasMigration`, with `onDrift: Report`
  first. Registry versions pushed with a tag need a dev database on the `AtlasMigration` (`devURLFrom`)
  for the check to work (https://atlasgo.io/integrations/kubernetes/versioned).

## 4.7 Atlas Operator with Flux

The same Atlas resources as 4.6, ordered through Kustomizations instead of sync waves: the
Kustomization that holds the Atlas resource lists it under `healthChecks`, and the application's
Kustomization lists that one under `dependsOn` (https://atlasgo.io/guides/deploying/k8s-flux).

## 4.8 Terraform (decision point)

Offer this only when Terraform already deploys the application, or the user asks for schema changes in
Terraform plans. Use a provider `cloud` block plus `data "atlas_migration"` and
`resource "atlas_migration"` reading `atlas://<repo>`
(https://atlasgo.io/guides/terraform/opentaco), in the staging root module or workspace only. The
Terraform runner then needs network access to the database and the bot token. Pin the `ariga/atlas`
provider to the latest release on the Terraform Registry; version pins in older examples lag behind.

## 4.9 Run and Check (staging only)

1. Readiness: `atlas cloud migration list --repo <repo> --env-name staging --status FAILED` prints
   `No migrations found.`
2. Approval with a named target. Ask the user for the staging database's name instead of reading the
   URL: "Deploy `<repo>` version `<version>` to staging (`<database>`)?"
3. Dry run, with `STAGING_DATABASE_URL` in the agent's shell: `atlas migrate apply --env staging
   --dry-run` (declarative: `atlas schema apply --env staging --dry-run`). Atlas Cloud records it as
   `DRY_RUN`.
4. The user merges the deploy PR, or merges the GitOps PR, or runs the release.
5. Check:

   ```bash
   atlas migrate ls --dir atlas://<repo> --latest --short
   atlas cloud migration list --repo <repo> --env-name staging
   atlas cloud database list --env-name staging
   ```

   Done when the newest staging event is `PASSED` for that version and the database list shows the
   unit's repo `SYNCED` at it. A first deploy that only records the baseline runs no file and is still
   recorded as `PASSED`. On `FAILED`, follow `atlas/references/cloud.md`, Investigate a failed
   deployment.

Tell the user what runs on each merge now, and where to see it (https://atlasgo.io/cloud/deployment).
Production is stage 5 (`references/stage-5-promote.md`): environment promotion from staging, in its own
PR when the user asks for it.
