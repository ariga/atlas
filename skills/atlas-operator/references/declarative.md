# AtlasSchema: the Declarative Flow

An `AtlasSchema` holds the desired state of a database. The operator compares it with the database, plans
the `CREATE`, `ALTER`, and `DROP` statements between them, and applies them
(https://atlasgo.io/integrations/kubernetes/declarative). It acts when the resource changes, not when the
database does.

## Schema Sources

Inline SQL:

```yaml
apiVersion: db.atlasgo.io/v1alpha1
kind: AtlasSchema
metadata:
  name: atlasschema-pg
spec:
  urlFrom:
    secretKeyRef:
      key: url
      name: postgres-credentials
  schema:
    sql: |
      create table t1 (
        id int
      );
```

Inline HCL replaces `sql` with `hcl`:

```yaml
  schema:
    hcl: |
      schema "public" {
      }

      table "t1" {
        schema = schema.public
        column "id" {
          type = "int"
        }
      }
```

A ConfigMap key, which must end in `.sql` or `.hcl`:

```yaml
kind: ConfigMap
apiVersion: v1
metadata:
  name: mysql-schema
data:
  schema.sql: |
    create table users (
       id int not null auto_increment,
       name varchar(255) not null,
       email varchar(255) unique not null,
       short_bio varchar(255) not null,
       primary key (id)
    );
```

```yaml
apiVersion: db.atlasgo.io/v1alpha1
kind: AtlasSchema
metadata:
  name: configmap-schema
spec:
  schema:
    configMapKeyRef:
      key: schema.sql
      name: mysql-schema
```

The Atlas Registry, the recommended source: a schema version pushed from CI and referenced by tag, like a
container image. It needs the token:

```yaml
apiVersion: db.atlasgo.io/v1alpha1
kind: AtlasSchema
metadata:
  name: atlasschema-pg
spec:
  urlFrom:
    secretKeyRef:
      key: url
      name: postgres-credentials
  cloud:
    tokenFrom:
      secretKeyRef:
        name: atlas-token
        key: token
  policy:
    lint:
      review: WARNING
  schema:
    url: atlas://k8s-quickstart?tag=v1
```

`schema.url` accepts only `atlas://` URLs.

## Pushing the Schema to the Registry

The project file names the registry repository, and a dev database for planning, but not the target:

```hcl
env "local" {
  schema {
    src = "file://schema.sql"
    repo {
      name = "k8s-quickstart"
    }
  }
  dev = "docker://postgres/16/dev?search_path=public"
}
```

The first push creates the repository and cannot carry a tag. Later pushes move `latest`, and a tagged
push gives the operator an immutable version:

```shell
atlas schema push --env local
atlas schema push --env local k8s-quickstart:v1
```

`k8s-quickstart:v1` is shorthand for `--tag v1`. In CI, the tag is usually the commit SHA. Deploy a
version by changing the tag in `schema.url`: the operator does not watch the registry, so a resource that
keeps `tag=latest` is not updated by a new push (`references/gitops.md`).

## What Happens to a Destructive Change

The outcome depends on the token, the plan repository, and the policy:

| Resource | A change that drops a table, column, or other object |
|----------|---------------------------------------------------------|
| No token, first apply | Refused with `FirstRunDestructive`, nothing applied |
| No token, later applies, no lint policy | Applied |
| No token, `policy.lint.destructive.error: true` | Refused: `destructive changes detected: ...` |
| Token, no plan repository | Rejected: `Rejected by review policy: errors or warnings were found` |
| Token, plan repository (`cloud.repo` or an `atlas://` schema URL) | Planned, published to Atlas Cloud, and held as `ApprovalPending` |
| Any, with an approved plan for this exact change | Applied with the approved plan |

Set the policy explicitly on every resource rather than rely on these defaults. With a token, the
operator reviews changes at `ERROR` level and treats destructive changes as errors, unless the resource
sets `policy.lint` itself.

## Policies

Block destructive changes:

```yaml
policy:
  lint:
    destructive:
      error: true
```

Review changes before they apply, with Atlas Pro and a plan repository:

```yaml
policy:
  lint:
    review: WARNING # or ERROR
```

| `review` | The operator waits for approval when |
|----------|--------------------------------------|
| `ERROR` | the plan has errors: destructive changes by default, plus any check you classified at error severity |
| `WARNING` | the plan has any diagnostic, errors or warnings |
| `ALWAYS` | on every change, with or without diagnostics |

Never plan a kind of change:

```yaml
policy:
  diff:
   skip:
    drop_column: true
```

The kinds are `add_column`, `add_foreign_key`, `add_index`, `add_schema`, `add_table`, `drop_column`,
`drop_foreign_key`, `drop_index`, `drop_schema`, `drop_table`, `modify_column`, `modify_foreign_key`,
`modify_index`, `modify_schema`, and `modify_table`. A skipped change stays out of every plan: the column
remains in the database after you remove it from the desired state.

All of it on one resource:

```yaml
apiVersion: db.atlasgo.io/v1alpha1
kind: AtlasSchema
metadata:
  name: myapp
spec:
  # Load the Atlas API token from a Kubernetes secret.
  cloud:
    tokenFrom:
      secretKeyRef:
        name: atlas-token
        key: token
  # Load the URL of the target database from a Kubernetes secret.
  urlFrom:
    secretKeyRef:
      key: url
      name: mysql-credentials
  # Define the desired schema of the target database. This can be defined in either
  # plain SQL, in Atlas HCL or by referencing a schema version in the Schema Registry.
  schema:
    url: atlas://declarative-flow?tag=7e0d7c798a15c1d3788dc515b87925f7bb00d217
  # Define policies that control how the operator will apply changes to the target database.
  policy:
    # Define a policy that will stop the reconciliation loop if the operator detects
    # a destructive change such as dropping a column or table.
    lint:
      destructive:
        error: true
    # Define a policy that omits DROP COLUMN changes from any produced plan.
    diff:
      skip:
        drop_column: true
  # Exclude a table from being managed by the operator. This is useful for resources
  # that belong to other applications and are managed in other ways.
  exclude:
    - external_table_managed_elsewhere
```

## Approving Changes

Plans need Atlas Pro, the token, and a plan repository: the repository of an `atlas://` schema URL, or
`cloud.repo` for an inline or ConfigMap schema. A plan matches a transition from one state to another,
not a database, so one approved plan serves every database in the same state
(https://atlasgo.io/faq/multi-db-plan-approve).

### Ahead of time

Plan the change before the deployment, from the version running in the cluster. Plan before pushing the
new schema, so `latest` still points at what is deployed:

```shell
atlas schema plan --env local \
  --from "atlas://k8s-quickstart?tag=latest" \
  --to env://schema.src \
  --name drop-email --push
```

Atlas prints the plan with its diagnostics, and `Approve and push` stores it as approved. Then push the
new version, tag it, and update the tag in `schema.url`. The operator finds the approved plan and applies
it without stopping.

In CI, the pull request pushes the plan as pending, and the merge approves it: GitHub Actions runs
`ariga/atlas-action/schema/plan@v1` on the pull request, and `schema/plan/approve@v1` and
`schema/push@v1` on merge. The push tags the schema with the merge commit's SHA, the tag to put in
`schema.url` (https://atlasgo.io/integrations/kubernetes/pre-approval).

### Ad hoc

Without an approved plan, a change that the review policy flags is planned by the operator, published to
Atlas Cloud, and held: `Ready=False` with reason `ApprovalPending`, message
`Schema plan is waiting for approval`, and the plan's address in `status.planLink`:

```shell
kubectl get atlasschema atlasschema-pg \
  -o=jsonpath='{.status.conditions[?(@.type=="Ready")].reason}{"\n"}{.status.planLink}{"\n"}'
```

- The database is untouched while the plan waits. Approve it in Atlas Cloud, or from the CLI with
  `atlas schema plan approve --url <plan URL>`.
- The operator checks a pending plan every few seconds at first, and at most every five minutes later,
  so the apply follows the approval within a few minutes.
- More than one matching plan fails with `multiple schema plans found: <urls>`: remove the extras with
  `atlas schema plan rm --url <url>`.

For an inline schema, create a repository in Atlas Cloud (a declarative workflow repository) and name it
in `cloud.repo` (https://atlasgo.io/integrations/kubernetes/ad-hoc-approval):

```yaml
apiVersion: db.atlasgo.io/v1alpha1
kind: AtlasSchema
metadata:
  name: atlasschema-pg
spec:
  urlFrom:
    secretKeyRef:
      key: url
      name: postgres-credentials
  cloud:
    repo: ad-hoc-approval
    tokenFrom:
      secretKeyRef:
        name: atlas-token
        key: token
  policy:
    lint:
      review: ALWAYS
  schema:
    sql: |
      create table t1 (
        id int
      );
```

`review: ALWAYS` without `cloud.repo` fails, with an error about `--format` in interactive mode.

### Renames

A rename in the desired state is a drop and an add, and loses the column's data. The
`-- atlas:renamed_from` directive plans a rename instead:

```sql
CREATE TABLE t1 (
  id int,
  -- atlas:renamed_from name
  full_name text
);
```

The rename is still reported as a backward-incompatible change (`BC102`), so a review policy holds it for
approval.

## Excluding and Scoping

`exclude` leaves objects to other tools:

```yaml
exclude:
- external_table_managed_elsewhere
```

`schemas` manages several schemas in one database, with a database-scoped URL only:

```yaml
schemas:
- schema1
- schema2
```

- Several `AtlasSchema` resources on one database must not overlap: each one plans to drop what the
  others manage. Prefer one resource per database.
- One resource targets one database. A custom env with `for_each` fails with
  `multiple targets are not supported`; create a resource per database.
- On the first apply without a token, objects in the database that the desired state does not declare
  are drops, and fail the run with `FirstRunDestructive`. Declare them, exclude them, or narrow the URL
  to the schema the resource manages.

## Retries and Manual Changes

`backoffLimit` (default 20) sets how many times the operator retries a resource whose preparation
failed, such as a missing Secret, a failed login, or a desired state that does not load:

```yaml
backoffLimit: 5
```

A failed plan or apply is not retried. Fix the cause, then change the spec, or restart the operator, to
run it again.

The operator does not watch the database. Its next apply, after a change to the resource, reverts manual
changes: directly without a token, or as a plan held for approval when the revert is destructive and a
review policy applies. To keep a manual change, add it to the desired state first.

## Status

```shell
kubectl apply -f atlas-schema.yaml
kubectl wait --for=condition=Ready atlasschema/atlasschema-pg --timeout=2m
```

```
NAME             READY   REASON
atlasschema-pg   True    Applied
```

- The `Ready` message of an applied resource carries the apply report, with the statements it ran.
- A lint failure is in the message too:

  ```bash
  kubectl get atlasschema atlasschema-mysql -o jsonpath='{.status.conditions[?(@.type=="Ready")].message}'
  ```

  ```text
  destructive changes detected:
  - Dropping non-virtual column "short_bio"
  ```

- `status.observed_hash` is the hash of the desired state, not of the database. `status.last_applied`
  is the time of the last apply, and `status.planLink` the plan the resource waits for or applied.
- Every reason and its fix is in `references/troubleshooting.md`.
