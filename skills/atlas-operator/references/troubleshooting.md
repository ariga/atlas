# Troubleshooting

Errors live on the resource: its conditions and events say what failed and why. Read them first, then
the dev database, then the operator log.

## Where to Look

```shell
kubectl get atlasschema atlasschema-pg
kubectl describe atlasschema atlasschema-pg
```

- `kubectl get` shows `READY` and `REASON`. `kubectl describe` adds each condition's message and the
  resource's Events.
- The `Ready` message alone:

  ```bash
  kubectl get atlasschema atlasschema-mysql -o jsonpath='{.status.conditions[?(@.type=="Ready")].message}'
  ```

- Events of one kind:

  ```shell
  kubectl get events --field-selector involvedObject.kind=AtlasDriftCheck
  ```

- The dev database: `kubectl get deploy,pods -l app.kubernetes.io/instance=<resource-name>-atlas-dev-db`
  (`references/dev-database.md`).
- The operator log: `kubectl logs deployment/atlas-operator`, in the operator's namespace. Each line
  carries the operator and Atlas versions. Security scans write their errors only there. The Deployment
  is named after the Helm release when the release name contains `atlas-operator`, and
  `<release>-atlas-operator` otherwise.

## Conditions

Every kind the operator reconciles reports three conditions, which Argo CD, Flux, and Helm read:

| Condition | Meaning |
|-----------|---------|
| `Ready` | `True` once the resource is applied; otherwise its reason and message say why not |
| `Reconciling` | `True` while the operator works or waits: `Reconciling`, `GettingDevDB`, `ApprovalPending` |
| `Stalled` | `True` after a failure; `status.failed` counts consecutive failures |

`AtlasDriftCheck` adds `Drifted` (`references/drift.md`), and `AtlasSecurityScan` adds `Compliant`
(`references/security-scan.md`).

## Retrying

The operator reconciles a resource when:

- its spec changes;
- a Secret or ConfigMap it reads and watches changes: the `urlFrom`, `credentials.passwordFrom`, and
  `cloud.tokenFrom` Secrets, and the schema or migration ConfigMap;
- the operator restarts.

Other changes do not trigger it: annotations, labels, registry pushes, or changes in the database. The
exception is `AtlasSecurityScan`, where a new `db.atlasgo.io/scan-requested-at` annotation requests a
scan (`references/security-scan.md`).

- An `AtlasMigration` retries a failure with a growing delay, up to `backoffLimit` (default 20). After
  that, it emits a `BackoffLimitExceeded` event and waits.
- An `AtlasSchema` retries failures to read its inputs or log in, the same way. A failed plan, lint, or
  apply is not retried: the error is on the resource until something triggers a new reconcile.

To retry after fixing the cause, change the spec, or restart the operator, which reconciles every
resource. For a release named `atlas-operator`:

```shell
kubectl rollout restart deployment/atlas-operator
```

## Reading the Resource

| Error | Cause | Fix |
|-------|-------|-----|
| `no target database defined` | No `url`, `urlFrom`, or `credentials` | Set `urlFrom` (`references/credentials.md`) |
| `secrets "<name>" not found` | The Secret is missing, or in another namespace | Create it in the resource's namespace |
| `secret <namespace>/<name> does not contain key "<key>"` | Wrong key in `cloud.tokenFrom` or `devURLFrom` | Fix the key |
| `unknown driver for schema "<scheme>"`, often with an empty scheme | A wrong scheme, or a `urlFrom` key the Secret does not have, which reads as an empty URL | Check the Secret's key and the URL's scheme |
| `configmaps "<name>" not found` | The schema or migration ConfigMap is missing | Create it; the operator picks it up |
| `configmaps key "<key>" must be ending with .sql or .hcl` | The schema ConfigMap key has another extension | Rename the key, for example `schema.sql` |
| `unsupported URL schema "<scheme>"` | `schema.url` is not an `atlas://` URL | Use `atlas://<repo>?tag=<tag>`, or an inline or ConfigMap schema |
| `failed to parse project configuration` | `config` is not valid HCL | Fix the HCL |
| `install the operator with "--set allowCustomConfig=true" to use custom atlas.hcl config` | Custom config is off | `helm upgrade` with `--set allowCustomConfig=true` and the release's values |
| `env name must be set when using custom atlas.hcl config`, `env block "<name>" is not found` | `envName` is missing, or names no block | Match `envName` to the env block (`references/config.md`) |
| `multiple targets are not supported` | The custom env uses `for_each` | One resource per database |
| `no directory specified` | `AtlasMigration` has an empty `dir` | Set `dir.remote`, `dir.configMapRef`, or `dir.local` |
| `cannot use both configmaps and local directory` | Two directory sources | Keep one |
| `cannot use remote directory without Atlas Cloud token` | `dir.remote` without `cloud.tokenFrom` | Add `cloud.tokenFrom`; the operator-wide token does not count |
| `spec.policy.drift requires a migration directory on the Atlas Registry` | `policy.drift` on a ConfigMap or inline directory | Use `dir.remote`, or `AtlasDriftCheck` with a dev URL (`references/drift.md`) |

## The Dev Database

| Error | Fix |
|-------|-----|
| The resource stays at `GettingDevDB`: `waiting for dev database to be ready`, `no pods found`, `no running pods found` | Inspect the dev database pod: image pulls, admission policies, quotas, taints, the SQL Server license |
| `devdb: unsupported driver "<driver>". You need to provide the devURL on the resource` | Set `devURL` or `devURLFrom` |
| `devURL is required when devDB is provided` | Add `devURL` with host `localhost`; the operator swaps in the pod's address |
| `role "..." does not exist`, `function ... does not exist`, `schema "..." does not exist`, only on the operator | The desired state uses an object the empty dev database lacks (`references/dev-database.md`) |
| `cannot diff a database connection with a schema "<name>"` | The dev URL's scope differs from the target's: match `search_path` |
| `connected database is not clean`, on a `devURL` | The dev database is not empty: use a dedicated, empty one |
| `driver: bad connection` or `unexpected EOF` with a service mesh | A sidecar intercepts the dev database connection: opt the pod out with `devDB.metadata` |

## Login and Approvals

| Error | Cause | Fix |
|-------|-------|-----|
| `command requires 'atlas login'`, `login is required to use custom atlas.hcl config` | Custom config without a token | Add `cloud.tokenFrom` |
| `login is required to use ...` | A login-gated driver or feature without a token | Add `cloud.tokenFrom`, or the operator-wide token (`references/install.md`) |
| Reason `Login` | The token is invalid or revoked | Ask the user to replace the Secret's value with a new bot token |
| `creating resource home directory: ...` | The persistence volume is not writable | Check the volume's permissions |
| `Rejected by review policy: errors or warnings were found` | A token, a flagged change, and no plan repository | Set `cloud.repo`, or use an `atlas://` schema URL (`references/declarative.md`) |
| `auto-approve is not allowed when a lint policy is set to "ERROR"` | `policy.lint.review` without a `cloud` block | Add `cloud.tokenFrom` and a repository |
| `flag '--format' is not allowed in interactive mode` | `review: ALWAYS` without `cloud.repo` | Set `cloud.repo` |
| `ApprovalPending`: `Schema plan is waiting for approval` | A plan waits in Atlas Cloud | Approve it at `status.planLink` |
| `multiple schema plans found: <urls>` | Several plans match the same change | Remove the extras with `atlas schema plan rm --url <url>` |
| `ApprovalPending`: `plan approval pending, review here: <url>` | A revert waits for approval | Approve it at the link (`references/versioned.md`) |
| `PlanRejected`: `plan rejected, review here: <url>` | The revert plan was rejected | Point the manifest back at the version the database runs, or approve a new plan |

## Planning and Applying

| Error | Cause | Fix |
|-------|-------|-----|
| `FirstRunDestructive`: `... first run of a schema must not contain destructive changes.` | The first apply would drop objects the desired state omits | Declare, exclude, or scope them out, then change the spec |
| `destructive changes detected: - <statement>` | `policy.lint.destructive.error` blocked a drop | Confirm and adjust the policy, or skip the change with `policy.diff.skip` |
| Reason `VerifyingFirstRun` or `ApplyingSchema` with an Atlas error | Bad SQL or HCL, a connection error, or missing privileges | Fix it, then change the spec |
| Warning events `ApplyingSchema` or `Migrating` while `Ready=True` | Atlas applied the change and printed a warning, such as a plan limit | Read the warning |
| `ProtectedFlowError` | The manifest points at a version older than the database, or `protectedFlows` is set wrong | `references/versioned.md` |
| `unable to downgrade, no dir-state found` | The Secret with the applied files is gone | Restore the missing files, or revert from the CLI |
| `DriftDetected` | The database changed outside Atlas | `references/drift.md` |
| `checksum mismatch` | `atlas.sum` does not match the files | Rebuild the ConfigMap from the files; `atlas migrate hash` if they changed on purpose |
| `connected database is not clean: ... baseline version or allow-dirty is required` | An existing database without a baseline, or a URL wider than the directory | Set `baseline`; scope the URL (`references/versioned.md`) |
| `CREATE INDEX CONCURRENTLY cannot run inside a transaction block` | The file runs in a transaction | Start the file with `-- atlas:txmode none` and a blank line |
| `pq: unsupported startup parameter: search_path` | The URL points at a connection pooler | Connect to PostgreSQL directly |

## Stuck Resources

- `Reconciling` after a successful apply, under Argo CD or a similar tool that writes the resource:
  older operators lost the status update. Upgrade the operator; restarting it clears the state.
- A failed `AtlasSchema` after the database recovered: its failed apply is not retried. Change the spec,
  or restart the operator.
- `GettingDevDB` for minutes: the dev database pod does not become ready (`references/dev-database.md`).
- `ApprovalPending` after approving: the operator checks pending plans at intervals of up to five
  minutes.
- An `AtlasMigration` that stopped after `BackoffLimitExceeded`: fix the cause, then change the spec.

## Upgrading as a Fix

Each operator release ships its own Atlas build, and several reported problems were fixed by upgrading
the operator. Compare the Atlas version in the operator log with the fix, then upgrade with
`helm upgrade` and the release's values (`references/install.md`).
