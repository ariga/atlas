# The Dev Database

Atlas computes every plan on a dev database, an empty database it can change freely
(https://atlasgo.io/concepts/dev-database). The operator creates one for each resource, unless the
resource names its own.

## The Operator's Dev Database

- It is a Deployment named `<resource-name>-atlas-dev-db`, in the resource's namespace. The operator
  manages its lifecycle, and deleting the resource deletes it.
- It follows the target's scope: a schema-scoped target gets a schema-scoped dev database, a
  database-scoped target a database-scoped one.
- The operator creates one for PostgreSQL, MySQL, MariaDB, SQL Server, ClickHouse, CockroachDB, and
  YugabyteDB. Other engines, such as Redshift, Snowflake, Databricks, Spanner, Aurora DSQL, and Oracle,
  fail with `devdb: unsupported driver "<driver>". You need to provide the devURL on the resource`: set
  `devURL` or `devURLFrom`.
- It runs the engine's latest official image, with no resource requests or limits. When the target runs
  an older major version, or needs extensions the official image lacks, pin the image with `devDB.spec`,
  or provide a `devURL`.
- SQL Server's image needs its license accepted on the operator (`MSSQL_ACCEPT_EULA` and `MSSQL_PID`,
  `references/install.md`), or the pod never starts.
- An `AtlasMigration` starts its dev database only when it has files to apply or a revert to run.
  `AtlasDriftCheck` and `AtlasSecurityScan` never use it.
- The Deployment is created once. A later change to `devDB.spec`, `devDB.metadata`, or the `devURL`
  template takes effect after you delete `<resource-name>-atlas-dev-db` and the operator recreates it.

### Prewarming

The chart value `prewarmDevDB` (default `true`) keeps the dev database Deployment between
reconciliations: after each successful run, the operator replaces its pod with a fresh, empty one, so
the next change starts sooner. With `false`, the operator scales the Deployment to zero after each run,
and the next run waits for a new pod.

Since v0.8.0, `spec.prewarmDevDB` on an `AtlasSchema` or `AtlasMigration` overrides the chart value for
that resource. With `false`, its dev database scales to zero after the resource is reconciled; with
`true`, it stays running; unset, the chart value applies:

```yaml
apiVersion: db.atlasgo.io/v1alpha1
kind: AtlasMigration
metadata:
  name: myapp
spec:
  prewarmDevDB: false
  urlFrom:
    secretKeyRef:
      key: url
      name: db-credentials
  dir:
    configMapRef:
      name: migrations
```

### Waiting for the dev database

While the pod starts, the resource reports `GettingDevDB` with `waiting for dev database to be ready`,
`no pods found`, or `no running pods found`. The operator keeps waiting, with no time limit, so a pod
that never becomes ready leaves the resource at `GettingDevDB`. Inspect the pod:

```shell
kubectl get deploy,pods -l app.kubernetes.io/instance=<resource-name>-atlas-dev-db
kubectl describe pods -l app.kubernetes.io/instance=<resource-name>-atlas-dev-db
```

Common causes: the image cannot be pulled, an admission policy or Pod Security level rejects the pod, a
ResourceQuota requires limits, a node taint has no matching toleration, or the SQL Server license is not
accepted. Fix them with `devDB.spec` or `devDB.metadata`, or provide a `devURL`.

## Objects the Desired State Needs

The dev database starts empty. A desired state that uses a role, an extension, or a schema without
creating it fails there, even when the object exists on the target:

- `pq: role "provisioner" does not exist`: the role exists only on the target. Point `devURL` at a dev
  database that has the role.
- `function public.uuid_generate_v4() does not exist`, with `CREATE EXTENSION` in the schema: extensions
  are an Atlas Pro feature. Give the resource a token (`cloud.tokenFrom`), so Atlas creates the
  extension on the dev database.
- `schema "media" does not exist`: add `CREATE SCHEMA` to the desired state, and do not exclude a schema
  the desired state writes to.

## Providing a Dev Database

`devURL` names an existing dev database, for special requirements such as preloaded extensions. The
operator then creates no pod:

```yaml
devURL: mysql://root:password@dev-db-svc.default:3306/dev
```

`devURLFrom` reads it from a Secret:

```yaml
devURLFrom:
  secretKeyRef:
    key: dev-url
    name: myapp
```

- The dev database must be empty and used by nothing else: Atlas changes and cleans it on every run. A
  database with tables fails with `connected database is not clean`
  (https://atlasgo.io/faq/migrate-diff-clean).
- Its scope matches the target's: a schema-scoped target needs a schema-scoped dev URL, and a
  database-scoped target a database-scoped one. A mismatch fails with
  `cannot diff a database connection with a schema "<name>"`, or plans the wrong changes.
- It runs the target's engine version, with the extensions and settings the target has. For extension
  version errors, see https://atlasgo.io/faq/extension-version-mismatch.

## A Custom Pod: devDB.spec

`devDB.spec` is the pod spec the operator uses for the dev database Deployment. It requires `devURL`,
whose host can be any name, such as `localhost`: the operator replaces it with the pod's IP address once
the pod runs.

```yaml
  devURL: mysql://root:pass@localhost:3306/dev
  devDB:
    spec:
      containers:
        - name: mysql-dev
          image: mysql:latest
          env:
            - name: MYSQL_ROOT_PASSWORD
              value: pass
            - name: MYSQL_DATABASE
              value: dev
          ports:
            - containerPort: 3306
              name: mysql
          startupProbe:
            exec:
              command: [ "mysql", "-ppass", "-h", "127.0.0.1", "-e", "SELECT 1" ]
            failureThreshold: 30
            periodSeconds: 10
```

Use it to pin the image version to the target's, to add tolerations, a node selector, resources, a
service account, or image pull secrets, or to use an image from a private registry.

## Pod Labels and Annotations: devDB.metadata

Since v0.7.52, `devDB.metadata` adds labels and annotations to the dev database pod, without a custom
pod spec or a `devURL`
(https://atlasgo.io/changelog/operator-dev-db-pod-metadata):

```yaml
apiVersion: db.atlasgo.io/v1alpha1
kind: AtlasSchema
metadata:
  name: myapp
spec:
  urlFrom:
    secretKeyRef:
      key: url
      name: myapp-db
  devDB:
    metadata:
      labels:
        team: platform
      annotations:
        example.com/purpose: schema-analysis
  schema:
    sql: |
      create table users (
        id int not null,
        primary key (id)
      );
```

- They apply to the pod template only. The operator's own labels win on a conflict.
- They apply when the Deployment is created; an existing Deployment keeps its pods until you delete it.
- Use them for a service mesh: the operator connects to the dev database pod directly, and a sidecar
  that intercepts the connection fails it with errors such as `driver: bad connection`. Opt the pod out
  of injection with the label or annotation the mesh documents.
