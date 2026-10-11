---
name: atlas-operator
description: "Atlas Kubernetes Operator setup and operations: Helm install and values, AtlasSchema (declarative) and AtlasMigration (versioned) resources, Atlas Cloud bot tokens and the registry, database credentials and URL scope, the dev database in the cluster, lint and review policies, pre-approved and ad-hoc plans, down migrations, drift detection (policy.drift and AtlasDriftCheck), security scans (AtlasSecurityScan), custom atlas.hcl config and IAM auth, GitOps with Argo CD, Argo Rollouts, Flux, and Octopus Deploy, and troubleshooting from status conditions and events. Use with the atlas skill whenever Atlas runs in Kubernetes, or the user asks to deploy schema changes with the operator."
---

# Atlas Kubernetes Operator

The Atlas Kubernetes Operator applies schema changes from Kubernetes resources
(https://atlasgo.io/integrations/kubernetes). This skill covers installing it, writing its resources,
connecting it to Atlas Cloud, deploying through GitOps, and reading its status when something fails. The
`atlas` skill covers the Atlas CLI, which still generates, lints, tests, and pushes the changes the
operator deploys.

## Files

This skill is `SKILL.md` plus eleven reference files under `references/`, each served at
`https://atlasgo.io/skills/atlas-operator/references/<name>`. Read a reference when the task needs it:

| File | Read it for |
|------|-------------|
| `references/install.md` | Helm install and upgrade, chart values, the operator-wide token, persistence, SQL Server, TLS certificates, what `helm uninstall` removes |
| `references/credentials.md` | Database URLs in Secrets, the `credentials` block, URL scope, in-cluster DNS, Atlas Cloud bot tokens |
| `references/dev-database.md` | The dev database the operator creates, `devURL`, `devURLFrom`, `devDB.spec`, `devDB.metadata`, prewarming |
| `references/declarative.md` | `AtlasSchema`: schema sources, the registry, lint, review, and diff policies, approvals, `exclude`, `schemas` |
| `references/versioned.md` | `AtlasMigration`: directory sources, registry tags, baseline, history table, execution order, down migrations |
| `references/drift.md` | `policy.drift` before each deployment, the scheduled `AtlasDriftCheck`, recovering from drift |
| `references/gitops.md` | Deploying by changing the manifest, CI that updates the tag, Argo CD sync waves and health, Argo Rollouts, Octopus Deploy, Flux, Crossplane |
| `references/config.md` | Custom `atlas.hcl` (`config`, `configFrom`, `vars`, `envName`), RDS and Cloud SQL IAM authentication |
| `references/security-scan.md` | `AtlasSecurityScan` and `AtlasSecurityReport`: schedules, triggers, policy, waivers, access |
| `references/troubleshooting.md` | Reading conditions, events, and logs, forcing a retry, each error with its cause and fix |
| `references/testing.md` | Testing changes before they reach the cluster, and verifying each rollout |

## Requires

The `atlas` skill, installed next to this one (`../atlas/SKILL.md`). If it is missing, download
https://atlasgo.io/skills/atlas/SKILL.md to `../atlas/SKILL.md` and each reference file listed in its
Files section to `../atlas/references/<name>`. On PostgreSQL, MySQL, or MariaDB, also read the database
skill (`../atlas-postgres/SKILL.md` or `../atlas-mysql/SKILL.md`): its rules on scope and objects
apply to what the operator deploys.

## Setup Path

From an empty cluster to production, in order:

1. Install the operator with Helm, with `persistence` on so the token's cached grant survives restarts
   (`references/install.md`).
2. Store the database URL and an Atlas Cloud bot token in Secrets (`references/credentials.md`).
3. Pick the workflow the project uses with the CLI: `AtlasMigration` for a migration directory
   (`references/versioned.md`), `AtlasSchema` for a desired state (`references/declarative.md`).
4. Push the directory or schema to the Atlas Registry from CI, and reference it by tag
   (`references/gitops.md`).
5. Set the safety policies: lint and review on `AtlasSchema`; `migrate lint` in CI and protected down
   migrations for `AtlasMigration`.
6. Order the rollout: the database, then the Atlas resource, then the application
   (`references/gitops.md`).
7. Add drift detection and security scans (`references/drift.md`, `references/security-scan.md`).
8. Verify every rollout from the resource's status (`references/testing.md`).

## Rules

1. Give each resource a bot token. On the operator, Atlas Pro features read the token from
   `cloud.tokenFrom`, a Secret holding a bot token from Atlas Cloud: the registry, review policies and
   approvals, drift checks, security scans, custom config, and objects beyond tables and indexes, such as
   views, triggers, functions, and extensions. The operator has no `atlas login` session, and without a
   token those features fail or are skipped. A user without an account starts a free trial with
   `atlas login`, then creates the bot in Atlas Cloud (`references/credentials.md`). Enable the chart's
   `persistence`: the operator caches the token's grant on it, which keeps Pro features working when
   Atlas Cloud is unreachable, across pod restarts (`references/install.md`).
2. Keep credentials in Secrets: `urlFrom` for the target, `devURLFrom` for a dev database, and
   `cloud.tokenFrom` for the token, each in the resource's namespace. Never write a password or token
   into a manifest or the conversation; ask the user to create the Secret. The operator connects from its
   own namespace, so an in-cluster host needs its namespace-qualified name, such as `postgres.default`.
3. Match the URL scope to the schema. A URL that names a schema (`search_path` in PostgreSQL, the
   database in the MySQL path) limits Atlas to it and writes unqualified DDL. Qualified objects, several
   schemas, extensions, and roles need a database-scoped URL. A `devURL` you provide has the target's
   scope (`references/credentials.md`).
4. Atlas computes each change on a dev database: an empty database the operator creates for each
   resource, at the target's scope. A role, extension, or schema that the desired state uses but does not
   create fails there with `does not exist`: create it in the desired state, or point `devURL` at a dev
   database that has it. The dev database runs the engine's latest image; pin the target's version with
   `devDB.spec` when they differ (`references/dev-database.md`).
5. Deploy by changing the manifest. The operator acts only when the resource changes: a new push to the
   registry, even under the same tag, is not deployed until the manifest names it. Push from CI with a
   tag such as the commit SHA, then update `dir.remote.tag` or the tag in `schema.url`
   (`references/gitops.md`).
6. Set the policy of every `AtlasSchema` explicitly. Without a token, the operator refuses drops only on
   the first run, then applies whatever the desired state implies, drops included. With a token, it
   holds destructive changes for review and needs a plan repository (`cloud.repo`, or an `atlas://`
   schema URL) to publish them for approval. Use `policy.lint.destructive.error: true` to block drops,
   `policy.lint.review` with a repository to approve changes, and `policy.diff.skip` for changes that
   must never be planned (`references/declarative.md`).
7. Lint and test migrations in CI, before the push: the operator applies the files as they are and has
   no dry run. Preview with `atlas migrate apply --dry-run` from the CLI. Down migrations are blocked
   until `protectedFlows.migrateDown.allow: true`; with a registry directory, the sign-off comes from the
   directory's protected flow in Atlas Cloud (`references/versioned.md`).
8. The operator never re-checks a database on its own. For `AtlasMigration`, gate deployments with
   `policy.drift` and check on a schedule with `AtlasDriftCheck`. Both compare against states that the
   registry stores only for pushes without a tag, so CI pushes untagged as well. `AtlasSchema` has no
   drift detection: its next apply reverts manual changes, or holds the revert for review
   (`references/drift.md`).
9. Roll out in order: the database, then the Atlas resource, then the application, which starts only
   after the resource reports `Ready=True`. Use Argo CD sync waves, Flux `dependsOn` with health checks,
   or `kubectl wait --for=condition=Ready`. With canary or blue-green rollouts, such as Argo Rollouts,
   old and new versions share the new schema: keep migrations backward compatible. Helm hooks and init
   containers that run migrations are deprecated (`references/gitops.md`).
10. Custom `atlas.hcl` (`config`, `configFrom`) needs the operator installed with
    `allowCustomConfig=true`, a token, and `envName` naming the env block. When that env sets `url`, the
    resource's `url`, `urlFrom`, and `credentials` are ignored (`references/config.md`).
11. Read the resource before anything else: `kubectl describe` shows the `Ready` reason and message and
    the Events. A failed `AtlasSchema` apply is not retried: fix the cause, then change the spec or
    restart the operator. An `AtlasMigration` retries up to `backoffLimit` (20 by default), then waits
    for a change. Annotations do not trigger a retry (`references/troubleshooting.md`).
12. Each operator release ships its own Atlas build, so Atlas fixes reach the cluster through
    `helm upgrade`. Never `helm uninstall` to reset the operator: it deletes the CRDs and every Atlas
    resource with them. Deleting a resource leaves its database as it is (`references/install.md`).
13. Test every change before it reaches the cluster, as a habit: validate and lint it, and run
    `atlas schema test` and `atlas migrate test` from the CLI, as the `atlas` skill describes. After the
    rollout, wait for `Ready=True` and check the applied version or plan (`references/testing.md`).

## Quick Answers

| Question | Answer |
|----------|--------|
| `helm repo add` fails for the chart | The chart is published only to an OCI registry: `helm install atlas-operator oci://ghcr.io/ariga/charts/atlas-operator` (`references/install.md`) |
| A new tag was pushed, and nothing happened | The operator acts on resource changes only: update the tag in the manifest (`references/gitops.md`) |
| `ProtectedFlowError`: ``migrate down is not allowed, set `migrateDown.allow` to true to allow downgrade`` | The manifest points at a version older than the database: restore the tag, or allow the revert with `protectedFlows` (`references/versioned.md`) |
| `ApprovalPending`, with a link in `status.planLink` | Approve the plan in Atlas Cloud; the operator applies it on its next check (`references/declarative.md`) |
| `Rejected by review policy: errors or warnings were found` | The resource has a token but no plan repository: set `cloud.repo`, or use an `atlas://` schema URL (`references/declarative.md`) |
| `auto-approve is not allowed when a lint policy is set to "ERROR"` | `policy.lint.review` without a `cloud` block: add `cloud.tokenFrom` and a repository (`references/declarative.md`) |
| `flag '--format' is not allowed in interactive mode` with `review: ALWAYS` | The plan has nowhere to go: set `cloud.repo` (`references/declarative.md`) |
| `FirstRunDestructive`: `first run of a schema must not contain destructive changes` | The desired state omits objects that exist in the database: declare them, `exclude` them, or narrow the URL scope, then change the spec (`references/declarative.md`) |
| `destructive changes detected: - Dropping ...` | `policy.lint.destructive.error` blocked a drop: confirm it and adjust the policy, or skip the change with `policy.diff.skip` (`references/declarative.md`) |
| `role "..." does not exist`, `function ... does not exist`, or `schema "..." does not exist`, only on the operator | The dev database lacks it: create it in the desired state, add a token for Pro objects such as extensions, or set `devURL` (`references/dev-database.md`) |
| `cannot diff a database connection with a schema "<name>"` | The dev URL's scope differs from the target's: give both the same `search_path` (`references/dev-database.md`) |
| `devdb: unsupported driver "<driver>". You need to provide the devURL on the resource` | No automatic dev database for that engine: set `devURL` or `devURLFrom` (`references/dev-database.md`) |
| The resource stays at `GettingDevDB` | The dev database pod does not start: inspect the `<name>-atlas-dev-db` Deployment and its pods (`references/dev-database.md`) |
| `cannot use remote directory without Atlas Cloud token` | Add `cloud.tokenFrom` to the `AtlasMigration`; an operator-wide `ATLAS_TOKEN` does not count (`references/credentials.md`) |
| `login is required to use ...` for SQL Server, ClickHouse, Redshift, or a Pro feature | The operator has no token: add `cloud.tokenFrom` to the resource, or `ATLAS_TOKEN` to the chart's `extraEnvs`, a Helm value and not a resource field (`references/install.md`) |
| `install the operator with "--set allowCustomConfig=true" to use custom atlas.hcl config` | Upgrade the release with `--set allowCustomConfig=true` (`references/config.md`) |
| `env block "<name>" is not found`, or `env name must be set when using custom atlas.hcl config` | Set `envName` to the env block's name, or name the block `name = atlas.env` (`references/config.md`) |
| `connected database is not clean: ... baseline version or allow-dirty is required` | Set `baseline` to a version in the directory, and keep the URL to the schemas the directory manages (`references/versioned.md`) |
| `checksum mismatch` | `atlas.sum` does not match the files: re-hash with `atlas migrate hash`, and build the ConfigMap with `kubectl create configmap --from-file` instead of a template (`references/versioned.md`) |
| `CREATE INDEX CONCURRENTLY cannot run inside a transaction block` | Start the file with `-- atlas:txmode none` and a blank line (`references/versioned.md`) |
| `DriftDetected` on an `AtlasMigration` | The database changed outside Atlas: see what drifted with `atlas migrate drift`, then revert it, exclude it, or adopt it (`references/drift.md`) |
| `spec.policy.drift requires a migration directory on the Atlas Registry` | The pre-apply drift check needs `dir.remote` (`references/drift.md`) |
| A failed `AtlasSchema` stays failed after the database recovered | Its failed applies are not retried: change the spec, or restart the operator (`references/troubleshooting.md`) |
| The resource stays `Reconciling` after a successful apply under Argo CD | Upgrade the operator (`references/troubleshooting.md`) |
| Pro features fail after the operator pod restarts while Atlas Cloud is unreachable | The grant cache was lost with the pod: enable the chart's `persistence` (`references/install.md`) |
| The SQL Server dev database never starts | Accept the image's EULA: `MSSQL_ACCEPT_EULA` and `MSSQL_PID` in `extraEnvs` (`references/install.md`) |
| The database is unreachable at `localhost` or a short service name | Use the namespace-qualified service name, such as `postgres.default` (`references/credentials.md`) |

## Documentation

- Operator overview: https://atlasgo.io/integrations/kubernetes
- Quickstarts: https://atlasgo.io/integrations/kubernetes/quickstart,
  https://atlasgo.io/integrations/kubernetes/versioned-quickstart
- Installation: https://atlasgo.io/integrations/kubernetes/install
- Resource references: https://atlasgo.io/integrations/kubernetes/declarative,
  https://atlasgo.io/integrations/kubernetes/versioned
- Approvals: https://atlasgo.io/integrations/kubernetes/pre-approval,
  https://atlasgo.io/integrations/kubernetes/ad-hoc-approval
- Custom configuration: https://atlasgo.io/integrations/kubernetes/project-configuration
- GitOps: https://atlasgo.io/guides/deploying/k8s-argo, https://atlasgo.io/guides/deploying/k8s-flux
- Security scans: https://atlasgo.io/guides/security-scan/kubernetes
