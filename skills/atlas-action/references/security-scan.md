# Security Scan Reference: `security/scan`

Runs `atlas security scan` against one or more databases: it lists the extensions installed in each and
reports the ones with known vulnerabilities, as recorded in the Atlas Security Graph. It needs the login,
on a plan that includes the Security Graph. The step only reads, so give it a read-only database user.

```yaml
name: Database Security Scan
on:
  schedule:
    - cron: '0 6 * * *'
  workflow_dispatch:
jobs:
  scan:
    runs-on: ubuntu-latest
    environment: monitoring                 # default branch only, no reviewers: read-only users' URLs
    steps:
      - uses: ariga/setup-atlas@v0
        with:
          cloud-token: ${{ secrets.ATLAS_CLOUD_TOKEN }}
      - uses: ariga/atlas-action/security/scan@v1
        id: scan
        with:
          urls: |
            ${{ secrets.STAGING_DATABASE_URL }}
            ${{ secrets.PROD_DATABASE_URL }}
          fail-on: HIGH
```

| Input | Effect |
|-------|--------|
| `urls` | Targets, one per line. Without it, the env's `url`, including every target of an env with several |
| `min-severity` | Lowest severity reported: `NORMAL`, `ELEVATED`, `HIGH`, `CRITICAL` |
| `fail-on` | Lowest severity that fails the step. Unset: findings never fail it |
| `ignore` | CVE IDs not to report, one per line. Record why each is accepted, next to the line |

## Results

| Outcome | Step |
|---------|------|
| No findings | Passes |
| Findings below `fail-on`, or no `fail-on` | Passes; each finding is a warning annotation, and the job summary lists them |
| A finding at or above `fail-on` | Fails: `` `atlas security scan` completed with N issue(s): ... `` |
| A database could not be scanned | Fails: `` `atlas security scan` could not scan N database(s): <urls> ``, because its state is unknown |

- Outputs: `count` (findings reported), `failures` (databases not scanned), `report` (JSON per database:
  its extensions and vulnerabilities, each with its CVE ID and severity level).
- With `env`, the step uses the env's `security` block (`min_severity`, `fail_on`, `cve { ignore }`,
  and `notify` endpoints), so the thresholds live in `atlas.hcl`
  (https://atlasgo.io/guides/security-scan#configuration).
- Each finding carries its fix in `Suggestion`. For an extension built into the engine, such as
  `hstore` or `pgcrypto`, the fix ships in an engine release, so the finding names an engine upgrade.
- For a custom gate, read the `report` output: `.Targets[].Vulnerabilities[]` has `Name` (extension),
  `Version`, `ID` (the CVE), `Level` (the Security Graph's severity, always set), `Severity` (the
  record's CVSS rating, may be empty), `Title`, `Description`, and `Suggestion` (often the version to
  upgrade to); `.Targets[].Error` is set for a database that was not scanned.
- A failure after the report, such as a notification the command could not send, fails the step even
  when no finding reached `fail-on`; the findings are still reported.
- A registry repo's Security Graph prints as JSON with `atlas cloud repo secgraph --slug <slug>`, each
  finding with its `check` (the CVE) and `level`. To gate on it, run it in a script step piped to
  `jq '.nodes[] | select(.reports[]?.level == "critical")'`, and fail the step when that prints
  anything: `jq` exits 0 either way (https://atlasgo.io/cloud/features/security-graph).
- Guide: https://atlasgo.io/guides/security-scan.
