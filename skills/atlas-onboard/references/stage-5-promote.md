# Stage 5: Promote to Production

Goal: production applies only the version already deployed to staging. This is environment promotion,
and it builds on the earlier stages (https://atlasgo.io/guides/environment-promotion):

1. Push once: CI pushes every merge to the Atlas Registry (stage 3).
2. Validate in staging: staging deploys the pushed version, and Atlas Cloud records it (stage 4).
3. Promote: production reads, from Atlas Cloud, the version deployed to staging, and applies that
   version.

The agent writes the production configuration in its own PR, only when the user asks for it. The user
approves that PR and runs the first production deployment; the agent never applies to production.

## 5.1 The Production Env

The `cloud_databases` data source reads the version deployed to staging. Use it to pin production to
that version (the default), or to gate production with a pre-execution check that fails the deployment
when the planned version is not staging's.

Versioned units pin with `to_version`:

```hcl
data "cloud_databases" "staging" {
  repo = "<repo>"
  env  = "staging"
}

env "prod" {
  url     = urlqueryset(getenv("PROD_DATABASE_URL"), "search_path", "identity")
  exclude = local.exclude
  migration {
    dir        = "atlas://<repo>"
    baseline   = "<production baseline>"   # only if production existed before Atlas (stage 4.2)
    to_version = data.cloud_databases.staging.targets[0].current_version
  }
}
```

To gate instead, keep `dir` unpinned and add the guide's `check "migrate_apply"` with a `deny` rule on
`self.planned_migration.target_version > local.staging_version`
(https://atlasgo.io/guides/environment-promotion#example-pre-execution-check-for-promotion).

Declarative units pin the desired state to staging's version
(https://atlasgo.io/guides/environment-promotion-declarative):

```hcl
env "prod" {
  url = urlqueryset(getenv("PROD_DATABASE_URL"), "search_path", "identity")
  schema {
    src = "atlas://<repo>?version=${data.cloud_databases.staging.targets[0].current_version}"
  }
}
```

To gate instead, keep `src = "atlas://<repo>"` and add the guide's `check "schema_apply"` with a
`deny "version_mismatch"` rule. Checks also run on `--dry-run`, so the gate can be tested without
touching production.

`targets[0]` assumes one staging database for the unit.

## 5.2 The Production Deployment

Production deploys with the same target as staging (stage 4), with production values:

- CI: a separate job or workflow with `env: prod` that runs only after the staging deployment succeeded,
  in a protected environment that requires approval. It never runs on the merge alone.
- Atlas Operator: an `AtlasMigration` in the production overlay with `envName: prod` and the tag
  staging runs. With the Operator, promotion is the tag.
- Platform release step: the production environment's release command, with `PROD_DATABASE_URL`.

The same PR retires the old tool for production (stage 4.2). Before the first production deployment of
a versioned unit, the user checks production against its baseline, since it needs production access:

```bash
atlas schema apply --env prod --to "file://migrations?version=<production baseline>" --dry-run
```

## Check

After the user runs the first production deployment:

```bash
atlas cloud migration list --repo <repo> --env-name prod --status PASSED
atlas cloud database list --env-name prod
atlas cloud database list --env-name staging
```

Done when the production deployment passed, and production's `CURRENT VERSION` is staging's.

Tell the user how a schema change reaches production now: pull request, CI, merge, staging, promotion.
Link https://atlasgo.io/guides/environment-promotion (or the declarative guide).
