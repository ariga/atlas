# AtlasMigration: the Versioned Flow

An `AtlasMigration` applies the pending files of a migration directory, in order
(https://atlasgo.io/integrations/kubernetes/versioned). A history table records what ran, so each
database gets only what is pending on it. The operator acts when the resource changes.

## Directory Sources

### The Atlas Registry

The recommended source: a directory pushed from CI and referenced by name and tag. It is not limited by
the Kubernetes object size, CI builds it, and its tags map to commits. It needs `cloud.tokenFrom`:

```yaml
apiVersion: db.atlasgo.io/v1alpha1
kind: AtlasMigration
metadata:
  name: atlasmigration-pg
spec:
  urlFrom:
    secretKeyRef:
      key: url
      name: postgres-credentials
  cloud:
    tokenFrom:
      secretKeyRef:
        key: token
        name: atlas-token
  dir:
    remote:
      name: k8s-versioned
      tag: v1
```

Without `tag`, the operator uses the latest version at the time it reconciles. It does not check the
registry again, so always pin a tag and change it to deploy.

### A ConfigMap

One key per file, plus `atlas.sum`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: migration-dir
data:
  20230316085611.sql: |
    create table users (
      id int not null auto_increment,
      name varchar(255) not null,
      email varchar(255) unique not null,
      short_bio varchar(255) not null,
      primary key (id)
    );
  atlas.sum: |
    h1:sPURLhQRvLU79Dnlaw3aiU4KVkyUVEmW+ekenqu/V2o=
    20230316085611.sql h1:FKUFrD9E4ceSeBZ5owv2c05Ag8rokXGKXp53ZctbocE=
```

```yaml
apiVersion: db.atlasgo.io/v1alpha1
kind: AtlasMigration
metadata:
  name: atlasmigration-sample
spec:
  urlFrom:
    secretKeyRef:
      key: url
      name: db-credentials
  dir:
    configMapRef:
      name: "migration-dir"
```

Build the ConfigMap from the directory, so the files keep their exact bytes:

```bash
kubectl create configmap migration-dir --from-file=migrations -o yaml --dry-run=client --save-config > migrations.yaml
```

- `atlas.sum` hashes the exact bytes of every file. Rendering the files through a template, such as a
  Helm chart, changes their whitespace and fails with `checksum mismatch`. Re-hash with
  `atlas migrate hash` only when the files changed on purpose.
- Kubernetes limits a ConfigMap to 1 MiB. A larger directory belongs in the registry.

### Inline

`dir.local` maps file names to their contents, `atlas.sum` included:

```yaml
apiVersion: db.atlasgo.io/v1alpha1
kind: AtlasMigration
metadata:
  name: atlasmigration-sample
spec:
  urlFrom:
    secretKeyRef:
      key: url
      name: db-credentials
  dir:
    local:
      20230316085611.sql: |
        create table users (
          id int not null auto_increment,
          name varchar(255) not null,
          email varchar(255) unique not null,
          short_bio varchar(255) not null,
          primary key (id)
        );
      atlas.sum: |
        h1:sPURLhQRvLU79Dnlaw3aiU4KVkyUVEmW+ekenqu/V2o=
        20230316085611.sql h1:FKUFrD9E4ceSeBZ5owv2c05Ag8rokXGKXp53ZctbocE=
```

Use one source per resource: `dir.local` with `dir.configMapRef` fails with
`cannot use both configmaps and local directory`, and `dir.remote` wins over both.

## Pushing to the Registry

The project file builds the directory from the desired state, with a dev database at the target's scope:

```hcl
env "local" {
  src = "file://schema.sql"
  dev = "docker://postgres/16/dev"
  migration {
    dir = "file://migrations"
  }
}
```

```shell
atlas login
atlas migrate diff init --env local
atlas migrate push k8s-versioned --env local
atlas migrate push k8s-versioned:v1 --env local
```

- The bare push updates the directory's `latest`, and the tagged push pins this exact state. In CI, the
  tag is usually the commit SHA or a release version.
- Only pushes without a tag store the schema states that drift detection compares against, so push both
  ways (`references/drift.md`).
- To deploy, change `dir.remote.tag` in the manifest. Only the new files run, and
  `status.lastAppliedVersion` moves to the last of them.

## The History Table

- With a database-scoped URL, the history table `atlas_schema_revisions` lives in its own schema of the
  same name. With a schema-scoped URL, it lives in the connected schema.
- `revisionsSchema` sets the schema of the history table for a database-scoped URL:

  ```yaml
  revisionsSchema: "atlas"
  ```

- One directory per history table. Two resources that apply different directories, such as a schema
  directory and a seed directory, to the same history table each see the other's versions as the
  database running ahead, and stop to revert. Merge the directories, or give each its own
  `revisionsSchema`.

## Existing Databases

A database that already has a schema needs a baseline, a migration file that describes the existing
schema and is marked applied instead of run (https://atlasgo.io/versioned/apply#existing-databases):

```yaml
baseline: "20220811074144"
```

`connected database is not clean: ... baseline version or allow-dirty is required` means the operator
found objects it does not expect:

- Set `baseline` to the version of that file, the first in the directory.
- A database-scoped URL covers every schema, including ones the directory does not manage, such as
  schemas a cloud provider adds. Use a schema-scoped URL per directory, or a directory that covers the
  whole database.
- If the first deployment was interrupted, apply the baseline once from the CLI:

  ```shell
  atlas migrate apply \
    --url "mysql://root:pass@localhost:3306/example" \
    --dir "file://migrations" \
    --baseline "20220811074144"
  ```

## Order and Transactions

- `execOrder` sets how files apply when the history and the directory disagree on order: `linear` (the
  default), `linear-skip`, or `non-linear` (https://atlasgo.io/faq/out-of-order-migrations).
- Each file runs in its own transaction. A statement that cannot run in a transaction, such as
  `CREATE INDEX CONCURRENTLY`, needs the file to start with the `txmode` directive and a blank line:

  ```sql
  -- atlas:txmode none

  CREATE INDEX CONCURRENTLY name_idx ON users (name);
  ```

## Down Migrations

The operator reverts the database when the directory has fewer files than the database applied, for
example when the manifest moves back to an older tag. Reverting is off by default:

```
NAME                READY   REASON
atlasmigration-pg   False   ProtectedFlowError
```

```text
migrate down is not allowed, set `migrateDown.allow` to true to allow downgrade
```

Allow it on the resource:

```yaml
spec:
  protectedFlows:
    migrateDown:
      allow: true
```

- Registry directory: the directory's approval policy in Atlas Cloud decides who signs off. Turn on
  Protected Flows in the directory's settings and require approval for `atlas migrate down`
  (https://atlasgo.io/versioned/down). The operator plans the revert and waits, with
  `ApprovalPending: plan approval pending, review here: <link>`; it applies after approval, and a
  rejected plan sets `PlanRejected`. `autoApprove` is not allowed on a registry directory, so the cluster
  cannot opt out of the policy.
- ConfigMap or inline directory: set `allow: true` and `autoApprove: true` together; `allow` alone fails
  with `allow cannot be true without autoApprove for local migration directory`. The operator reverts
  without approval, using the files it stored in the Secret `io.atlasgo.db.v1.<resource-name>.latest`
  when it applied them. Without that Secret the revert fails with
  `unable to downgrade, no dir-state found`.
- A revert reverts the schema. Data the forward migrations changed is not restored unless the directory
  configures it.
- Preview a revert with `atlas migrate down --dry-run` from the CLI.

## Lint, Preview, and Retries

- The operator applies files as they are: it does not lint them and has no dry run. Run
  `atlas migrate lint` in CI while the directory is a pull request, and preview with
  `atlas migrate apply --dry-run`.
- A failed run is retried with a growing delay, up to `backoffLimit` (default 20). After that, the
  operator waits for a change to the resource:

  ```yaml
  backoffLimit: 5
  ```

## Status

```shell
kubectl apply -f atlas-migration.yaml
kubectl wait --for=condition=Ready atlasmigration/atlasmigration-pg --timeout=2m
kubectl get atlasmigration atlasmigration-pg -o jsonpath='{.status.lastAppliedVersion}{"\n"}'
```

```
20260807125351
```

- `status.lastAppliedVersion` is the database's version after the last run.
- The `Applied` event is emitted on every successful reconcile, including ones with nothing pending, so
  check `lastAppliedVersion` to know what ran.
- Every reason and its fix is in `references/troubleshooting.md`.
