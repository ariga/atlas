# Installing and Upgrading the Operator

The operator installs with Helm from an OCI registry
(https://atlasgo.io/integrations/kubernetes/install). It adds the Atlas resource kinds to the cluster,
and a controller that reconciles them.

## Install

```shell
helm install atlas-operator oci://ghcr.io/ariga/charts/atlas-operator
```

- The chart is published only to the OCI registry. `helm repo add` with that address fails; install from
  the `oci://` URL directly.
- `helm install` returns before the operator is ready. Add `--wait`, or wait on its pod:

  ```bash
  kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=atlas-operator -n default
  ```

- Pass values with `-f values.yaml`, or one at a time with `--set`:

  ```shell
  helm install atlas-operator oci://ghcr.io/ariga/charts/atlas-operator -f values.yaml
  ```

The chart installs these resource kinds, all in `db.atlasgo.io/v1alpha1`:

| Kind | Purpose | Since |
|------|---------|-------|
| `AtlasSchema` | Declarative: apply a desired state (`references/declarative.md`) | v0.1 |
| `AtlasMigration` | Versioned: apply a migration directory (`references/versioned.md`) | v0.2 |
| `AtlasDriftCheck` | Check an `AtlasMigration`'s database for drift on a schedule (`references/drift.md`) | v0.8.0 |
| `AtlasSecurityScan` | Scan a database for vulnerable extensions (`references/security-scan.md`) | v0.8.0 |
| `AtlasSecurityReport` | The findings of a scan, written by the operator | v0.8.0 |

The kinds have no short names: use `kubectl get atlasschemas`, `atlasmigrations`, `atlasdriftchecks`,
`atlassecurityscans`, and `atlassecurityreports`.

## Chart Values

| Value | Default | What it does |
|-------|---------|--------------|
| `prewarmDevDB` | `true` | Keeps the dev database Deployment between reconciliations, so changes apply faster. `false` scales it to zero after each run (`references/dev-database.md`) |
| `allowCustomConfig` | `false` | Allows `config` and `configFrom` on resources (`references/config.md`) |
| `extraEnvs` | `[]` | Environment variables for the operator: `ATLAS_TOKEN`, `MSSQL_ACCEPT_EULA`, `MSSQL_PID` |
| `extraVolumes`, `extraVolumeMounts` | `[]` | Volumes on the operator pod, from a Secret or ConfigMap, such as TLS certificates |
| `persistence.enabled` | `false` | A volume for Atlas state, including the cached grant of the token |
| `rbac.securityReports.*` | see `references/security-scan.md` | Who can read security reports |

`extraEnvs` configures the operator Deployment. It is a chart value, not a field of `AtlasSchema` or
`AtlasMigration`, and it has no effect in a resource's spec.

Keep the operator at one replica: the chart does not enable leader election, so extra replicas reconcile
the same resources at once.

## The Operator-Wide Token

Resources read their bot token from `cloud.tokenFrom` (`references/credentials.md`). The chart can also
give one token to every resource, through `ATLAS_TOKEN`. Create the Secret in the operator's namespace:

```shell
kubectl create secret generic atlas-token-secret \
  --from-literal=ATLAS_TOKEN='aci_xxxxxxx'
```

```yaml
extraEnvs:
- name: ATLAS_TOKEN
  valueFrom:
    secretKeyRef:
      key: ATLAS_TOKEN
      name: atlas-token-secret
```

- It enables login-gated drivers (SQL Server, ClickHouse, Redshift) and objects such as triggers,
  functions, procedures, and sequences, for every resource.
- An `AtlasMigration` with a registry directory (`dir.remote`) still needs `cloud.tokenFrom`: the
  operator-wide token does not count there.
- Prefer `cloud.tokenFrom` on each resource; use the operator-wide token when a single token serves the
  whole cluster.

## Persisting the Grant Cache

With a token, the operator logs in with `atlas login --token --grant-only` and caches the grant under
`.atlas` on disk. The cached grant keeps Atlas Pro features working for some time when Atlas Cloud is
unreachable, so deployments in the cluster do not depend on Atlas Cloud being up
(https://atlasgo.io/cloud/bots#offline-access). Without a volume, the cache is lost when the operator pod
is replaced, and a pod that starts while Atlas Cloud is unreachable has no grant. Enable `persistence` in
production (https://atlasgo.io/integrations/kubernetes/install#persisting-the-atlas-grant-cache):

```yaml
persistence:
  enabled: true
  size: 1Gi
  accessModes:
    - ReadWriteOnce
  mountPath: /data
  # storageClassName: standard
  # annotations: {}
```

- The chart creates a PersistentVolumeClaim, mounts it at `persistence.mountPath` (default `/data`), and
  points the operator's data directory at it.
- Each `AtlasSchema` and `AtlasMigration` keeps its own Atlas data, grant included, in a subdirectory
  named after its namespace and name.
- The operator must be able to write to the volume. A volume it cannot write fails resources with
  `creating resource home directory: ...`.

## SQL Server

The SQL Server dev database runs Microsoft's image, which requires accepting its license. Set both
variables on the operator, or the dev database never starts:

```yaml
extraEnvs:
- name: MSSQL_ACCEPT_EULA
  value: "Y"
- name: MSSQL_PID
  value: "Developer"
```

## TLS Certificates

To connect with client certificates (https://atlasgo.io/guides/deploying/k8s-operator-certs), store
them in a Secret:

```shell
kubectl create secret generic my-secret \
  --from-file=ca.crt=./path/to/ca.crt \
  --from-file=tls.key=./path/to/tls.key \
  --from-file=tls.crt=./path/to/tls.crt
```

Mount it into the operator with these values:

```yaml
extraVolumes:
   - name: certs
     secret:
       secretName: my-secret
       defaultMode: 0640
extraVolumeMounts:
   - name: certs
     mountPath: /certs
     readOnly: true
```

Then reference the files in the database URL:

```shell
postgresql://username@hostname:port/database?sslmode=verify-full&sslcert=/certs/tls.crt&sslkey=/certs/tls.key&sslrootcert=/certs/ca.crt
```

## Upgrading

```shell
helm upgrade atlas-operator oci://ghcr.io/ariga/charts/atlas-operator --version 0.8.0
```

- Without `--version`, Helm installs the latest chart.
- Pass the same values on every upgrade (`-f values.yaml`). An upgrade with only `--set` resets the
  other values to their defaults, and drops settings such as `extraEnvs`.
- Each operator release ships its own Atlas build. A fix in Atlas reaches the cluster with the next
  operator release, so upgrading is the first step when a known Atlas issue appears. The operator log
  prints the operator and Atlas versions.
- Features by release: `policy.drift` and `devDB.metadata` need v0.7.52; `AtlasDriftCheck`,
  `AtlasSecurityScan`, and the per-resource `prewarmDevDB` need v0.8.0.

## Uninstalling

`helm uninstall` removes the CRDs, and with them every `AtlasSchema`, `AtlasMigration`, drift check,
and scan in the cluster. Kubernetes then deletes what those resources own: the dev database Deployments
and the Secrets that keep the migration files needed to migrate down. The databases are not changed.

- Never uninstall to reset or fix the operator. Upgrade it, or restart it with
  `kubectl rollout restart deployment/atlas-operator`, for a release named `atlas-operator`.
- Deleting a single resource leaves its database as it is.
