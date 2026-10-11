# Security Scans

Since v0.8.0, an `AtlasSecurityScan` runs `atlas security scan` against a database on a schedule, and
after an `AtlasSchema` or `AtlasMigration` listed in its `triggers` applies a new version
(https://atlasgo.io/guides/security-scan/kubernetes). It grades the findings, known vulnerabilities in
the database's extensions, against a policy, and writes them to an `AtlasSecurityReport` of the same
name. It needs Atlas Pro with the Security Graph, and a bot token.

## Creating a Scan

```shell
kubectl create secret generic app-db \
  --from-literal=url='postgres://postgres:pass@postgres.app:5432/app?sslmode=disable'
```

```yaml
apiVersion: db.atlasgo.io/v1alpha1
kind: AtlasSecurityScan
metadata:
  name: app
spec:
  urlFrom:
    secretKeyRef:
      name: app-db
      key: url
  cloud:
    tokenFrom:
      secretKeyRef:
        name: atlas-token
        key: token
  schedule: "0 6 * * *"
  timeZone: UTC
  policy:
    minSeverity: ELEVATED
    failOn: HIGH
```

The first scan runs as soon as the resource is created:

```shell
kubectl apply -f security-scan.yaml
kubectl wait atlassecurityscan/app --for=condition=Ready --timeout=5m
kubectl get atlassecurityscans
```

```
NAME   READY   REASON    COMPLIANT   FINDINGS   HIGHEST   LAST SCAN   NEXT SCAN              AGE
app    True    Scanned   False       3          HIGH      0s          2026-10-01T06:00:00Z   6s
```

## Fields

| Field | Default | What it does |
|-------|---------|--------------|
| `url`, `urlFrom`, `credentials` | | The database to scan, as on other resources |
| `cloud.tokenFrom` | | The bot token. `cloud.repo` is rejected on scans |
| `schedule` | | A five-field cron expression, or `@hourly`, `@daily`, `@weekly`, `@monthly`, `@yearly`. `@every` and `TZ=` prefixes are rejected |
| `timeZone` | `UTC` | An IANA zone name for `schedule` |
| `triggers` | | Up to 32 `AtlasSchema` or `AtlasMigration` resources in the same namespace whose applies start a scan |
| `policy.minSeverity` | `NORMAL` | The lowest level reported: `NORMAL`, `ELEVATED`, `HIGH`, `CRITICAL` |
| `policy.failOn` | unset | The lowest level, not waived, that makes the scan non-compliant. Unset reports only |
| `policy.ignore` | | Waivers: `id`, `reason`, and an optional `expirationTime` |
| `suspend` | `false` | Pauses scanning; on resume, one scan covers what became due |
| `backoffLimit` | `20` | Failed attempts before the scan stalls |

A scan needs at least one of `schedule` and `triggers`.

## After an Apply

```yaml
spec:
  schedule: "0 6 * * *"
  timeZone: UTC
  triggers:
    - kind: AtlasSchema
      name: app
    - kind: AtlasMigration
      name: app-migrations
```

A trigger that does not exist emits a `TriggerNotFound` event; the scan keeps its schedule.

## On Demand

The annotation `db.atlasgo.io/scan-requested-at` requests a scan. The value comes back in
`status.lastHandledScanRequest` when the scan completes:

```shell
T=$(date -u +%FT%TZ)
kubectl annotate atlassecurityscan/app db.atlasgo.io/scan-requested-at="$T" --overwrite
kubectl wait atlassecurityscan/app --for=jsonpath="{.status.lastHandledScanRequest}=$T" --timeout=10m
```

## Reading the Results

Two conditions answer two questions:

- `Ready`: did the last scan run? A finding never sets `Ready=False`; only a scan that could not run
  does.
- `Compliant`: is the database within the policy? `False` when a finding at or above `failOn` is not
  waived, `Unknown` before the first scan or without `failOn`.

Wait on `Ready` for a result, and on `Compliant` as a deployment gate:

```shell
kubectl get atlassecurityscan app -o yaml
kubectl get atlassecurityreport app -o yaml
```

The report lists each finding with its extension, version, level, and suggested fix. It is replaced on
every successful scan.

## Waivers

```yaml
spec:
  policy:
    minSeverity: ELEVATED
    failOn: HIGH
    ignore:
      - id: CVE-2026-14678
        reason: "pg_trgm is not reachable from the application role; SEC-1234"
        expirationTime: "2026-12-31T00:00:00Z"
```

A waived finding stays in the report, marked waived, and stops counting toward `failOn`. When the waiver
expires, a new scan runs.

## Access to Reports

Reports name the database's extensions and their vulnerabilities, so the built-in `view` and `edit` roles
cannot read them. The chart ships a ClusterRole for them, aggregated into `admin` by default. It is named
`<release>-atlas-operator-securityreport-viewer`, or `<release>-securityreport-viewer` when the release
name already contains `atlas-operator`.

```yaml
rbac:
  securityReports:
    aggregateToAdmin: true
    aggregateToEdit: false
    aggregateToView: false
```

To let a team read the reports of one namespace, bind the ClusterRole there:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: security-report-readers
  namespace: app
subjects:
  - kind: Group
    name: security-team
    apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: atlas-operator-securityreport-viewer
  apiGroup: rbac.authorization.k8s.io
```

## Notifications

A custom configuration can add a `security` block that notifies on each scan. The operator must run with
`allowCustomConfig=true`, and the resource needs `envName` (`references/config.md`):

```yaml
spec:
  envName: kubernetes
  config: |
    variable "slack_webhook" {
      type = string
    }
    env "kubernetes" {
      security {
        notify {
          http "slack" {
            url  = var.slack_webhook
            body = jsonencode({ text = "${scan.count} findings (${scan.high} high)" })
          }
        }
      }
    }
  vars:
    - key: slack_webhook
      valueFrom:
        secretKeyRef:
          name: slack
          key: webhook
```

The policy belongs in `spec.policy`: a configuration that sets the scan's severity, failure threshold,
waivers, or an env `exclude` is rejected with `InvalidTarget`.

## Failures

Status and events never carry the database URL or the error text. The `Ready` message names the class
of failure, and the operator log has the cause:

| Reason | Meaning |
|--------|---------|
| `ScanFailed` | The database could not be scanned: connection or credentials |
| `LoginFailed` | Atlas Cloud login failed: token, or a plan without the Security Graph |
| `CLIError` | Atlas failed before producing a report |
| `ReadingInputs` | A Secret or ConfigMap the scan reads is missing |
| `StoringReport` | The report could not be written |
| `BackoffLimitExceeded` | Retries ran out; the next schedule slot, apply, request, or spec change tries again |
| `InvalidSchedule`, `InvalidTimeZone`, `InvalidTarget` | The spec cannot work: fix it as the message says |
