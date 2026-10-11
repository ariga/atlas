# Drift Detection

Drift is a database that no longer matches the state its migrations produced, usually after a manual
change. The operator does not re-check a database between deployments on its own. Two features detect
drift for an `AtlasMigration`, both with Atlas Pro
(https://atlasgo.io/integrations/kubernetes/versioned#drift-detection):

| | `policy.drift` | `AtlasDriftCheck` |
|---|---|---|
| What | A check inside each deployment | A separate resource that checks on a schedule |
| When | Only when the directory has files to apply | Every `interval`, by default `5m` |
| Directory | Registry (`dir.remote`) only | Registry, or a ConfigMap or inline directory with a dev URL |
| On drift | `onError: FAIL` (default) blocks the deployment; `CONTINUE` applies and records it | Reports `Drifted=True`; `onDrift: Fail` also sets `Ready=False` |
| Changes the database | No | No |
| Since | v0.7.52 | v0.8.0 |

`AtlasSchema` has no drift detection. Its next apply reverts manual changes
(`references/declarative.md`).

## Registry States

In registry mode, both features compare the database with the state the registry stored for the applied
version. The registry stores that state only for pushes without a tag: a push with a tag uploads the
files only. Push both ways from CI:

```shell
atlas migrate push k8s-versioned --env local
atlas migrate push k8s-versioned:v1 --env local
```

Without a stored state, the deployment check is skipped, and `AtlasDriftCheck` needs a dev database to
compute the state instead (local mode).

## Before Each Deployment: policy.drift

```yaml
spec:
  dir:
    remote:
      name: "atlas"
      tag: "commit-id"
  cloud:
    tokenFrom:
      secretKeyRef:
        key: token
        name: atlas-token
  policy:
    drift:
      onError: FAIL # FAIL (default) or CONTINUE
      exclude: # Optional. Replaces the env-level exclude list.
        - "public.audit_*"
        - "*[type=extension]"
```

- The check runs inside the deployment, so it runs only when the new tag has files to apply. It is a
  deployment gate, not a monitor.
- It needs a registry directory: a ConfigMap or inline directory fails with
  `spec.policy.drift requires a migration directory on the Atlas Registry`.
- It skips a fresh database, and a database whose applied version the registry does not hold, for
  example one migrated from a ConfigMap before moving to the registry.
- `exclude` patterns follow the URL scope: `audit_log` or `users.nickname` at schema scope;
  `public.audit_log`, `internal.*`, or `*[type=extension]` at database scope. The history table is
  always excluded.

### When drift blocks a deployment

```
$ kubectl get atlasmigrations
NAME   READY   REASON
app    False   DriftDetected
```

The resource reports `Ready=False` with reason `DriftDetected`, a `Warning` event of the same name, and
the message `database state does not match expected state at version <version>`. No file ran. The
operator retries with a growing delay up to `backoffLimit`, then waits for a change to the resource.

The message names the version, not the objects. See what drifted from the CLI, against the same
database and registry directory:

```shell
atlas migrate drift --env prod
```

Then pick one:

1. Revert the manual change, so the database matches the expected state again.
2. Add an `exclude` pattern for objects that other tools own.
3. Adopt the change: set `onError: CONTINUE`, deploy a migration that applies the same change and does
   nothing when it already exists, then set `onError: FAIL` again.

With `CONTINUE`, the deployment proceeds and the drift is recorded only in Atlas Cloud's deployment log:
the resource is `Ready`, and neither its condition nor its events mention the drift.

## On a Schedule: AtlasDriftCheck

```yaml
apiVersion: db.atlasgo.io/v1alpha1
kind: AtlasDriftCheck
metadata:
  name: myapp-drift
spec:
  targetRef:
    name: myapp # An AtlasMigration in the same namespace.
  interval: 5m
  onDrift: Report # Report (default) or Fail
  exclude:
    - "public.audit_*"
    - "*[type=extension]"
```

| Field | Default | What it does |
|-------|---------|--------------|
| `targetRef.name` | required | The `AtlasMigration` to check, in the same namespace |
| `interval` | `5m` | Time between checks, at least `1m` |
| `timeout` | `5m` | Limit for one check |
| `suspend` | `false` | Stops checking and keeps the last result |
| `onDrift` | `Report` | `Report` sets `Drifted=True` only. `Fail` also sets `Ready=False` and `Stalled=True`, for GitOps health |
| `exclude` | the target's `policy.drift.exclude` | Objects to ignore; when set, replaces the target's list |

- The check never changes the database. It uses the target's URL, token, and directory, and needs Atlas
  Pro on the target's token.
- It reads the target when it runs, so a change to the target, such as a new dev URL, applies at the
  next check.
- While the target is applying, or another client holds the migration lock, the check waits and retries
  in about 30 seconds.
- A target env that uses `for_each` is not supported.

### Local mode

A ConfigMap or inline directory, or a registry version without a stored state, is checked by replaying
the directory on a dev database. The operator does not create one for the check: set `devURL` or
`devURLFrom` on the target `AtlasMigration`, at the target's scope.

```yaml
apiVersion: db.atlasgo.io/v1alpha1
kind: AtlasMigration
metadata:
  name: myapp
spec:
  urlFrom:
    secretKeyRef:
      key: url
      name: db-credentials
  devURLFrom:
    secretKeyRef:
      key: devURL
      name: db-credentials
  dir:
    configMapRef:
      name: migrations
```

`status.mode` shows `registry` or `local`.

### Results

```text
Drifted=False  NoDrift: no drift detected at version 20250901000000
Drifted=True   DriftDetected: 2 drifted objects (extra 1, modified 1) at version 20250901000000: table 2
```

| Event | Type | Emitted when |
|-------|------|--------------|
| `DriftDetected` | `Warning` | A clean database is found drifted |
| `DriftChanged` | `Warning` | The database is still drifted, but a different set of objects drifted |
| `DriftResolved` | `Normal` | A drifted database matches its expected state again |
| `CheckFailed` | `Warning` | A check could not run. A repeat of the same failure does not emit a new event |

```shell
kubectl get events --field-selector involvedObject.kind=AtlasDriftCheck
```

The status and events name the kinds and counts of drifted objects, not their definitions: run
`atlas migrate drift` to see the difference. A check that cannot run sets `Drifted=Unknown`, keeps the
last result, and retries.

| Message | Fix |
|---------|-----|
| `AtlasMigration "<name>" not found in namespace "<namespace>"` | Fix `targetRef.name`; the target must be in the same namespace |
| `local migration directory requires a dev database to replay: set spec.devURL or spec.devURLFrom on AtlasMigration/<name>, or push the directory to the Atlas Registry` | Set a dev URL on the target, or move the directory to the registry |
| `no migration history found` | The database was never migrated, or uses another `revisionsSchema` |
| `no state found for version ...` | Push the directory without a tag, or give the target a dev URL |
| `command requires 'atlas login'` | Add `cloud.tokenFrom` to the target |
| `command 'atlas migrate drift' is available only to Atlas Pro users` | Use a token from an Atlas Pro organization |
| `the drift check timed out` | Raise `timeout` |
| `spec.interval: Invalid value: "30s": interval must be at least 1m` | Use an interval of `1m` or more |
