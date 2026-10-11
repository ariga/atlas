# Deploying with GitOps

The operator applies what the manifest names. A deployment is a change to the manifest, which a GitOps
tool syncs to the cluster: Argo CD (https://atlasgo.io/guides/deploying/k8s-argo), Flux
(https://atlasgo.io/guides/deploying/k8s-flux), or a plain `kubectl apply`. This reference also covers
Argo Rollouts and Octopus Deploy on top of Argo CD.

## A Deployment Is a Manifest Change

The operator reconciles when the resource changes. A new version in the registry, even one pushed under
the tag the manifest already names, is not deployed until the manifest changes. So:

1. CI lints, tests, and pushes the directory or schema to the registry, tagged with the commit SHA
   (`references/testing.md`).
2. A CI step writes that tag into the manifest and commits it.
3. The GitOps tool syncs the manifest, and the operator applies the new version.

For an `AtlasMigration` on GitHub Actions, `ariga/atlas-action/migrate/push@v1` tags the push with the
commit SHA by default, and also moves `latest`. It needs a dev database: a service container in the job,
as in https://atlasgo.io/integrations/github-actions#arigaatlas-actionmigratepush. Then two steps write
the tag into the manifest and commit it:

```yaml
- uses: ariga/atlas-action/migrate/push@v1
  if: github.ref == 'refs/heads/main'
  with:
    dir: 'file://migrations'
    dir-name: 'my-app'
    dev-url: 'mysql://root:pass@localhost:3306/dev'

- name: Update tag in migration.yaml
  run: |
    SHA="${GITHUB_SHA}"
    sed -i "s|tag: \".*\"|tag: \"${SHA}\"|" manifests/migration.yaml

- name: Commit and push changes
  run: |
    git config user.name "github-actions[bot]"
    git config user.email "github-actions[bot]@users.noreply.github.com"
    git add manifests/migration.yaml
    git commit -m "chore: update Atlas migration tag to ${GITHUB_SHA}" || exit 0
    git push
```

For an `AtlasSchema`, push with `ariga/atlas-action/schema/push@v1` and replace the `tag=` value of
`schema.url` the same way. In a manifest, `schema:` and `url:` sit on separate lines, so the pattern
must match the `url: atlas://...?tag=` line; check that the commit step finds a change, since
`|| exit 0` hides a pattern that matched nothing.

## Ordering the Rollout

The database comes first, then the Atlas resource, then the application. The application rolls out only
after the Atlas resource reports `Ready=True`, so new code never runs against the old schema.

### Argo CD

Order resources with sync waves:

```yaml
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "<order-number>"
```

The database in wave `"0"`, the Atlas resource in `"1"`, the application in `"2"`:

```yaml
apiVersion: db.atlasgo.io/v1alpha1
kind: AtlasMigration
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "1"
  name: myapp
spec:
  urlFrom:
    secretKeyRef:
      key: url
      name: db-credentials
  cloud:
    tokenFrom:
      secretKeyRef:
        key: token
        name: atlas-token
  dir:
    remote:
      name: "my-app" # The name of the migration directory in Atlas Cloud
      tag: "commit-id"
```

Argo CD 2.10 and later include health checks for `AtlasMigration` and `AtlasSchema`, so a wave waits for
the resource to be ready. Earlier versions need a custom health check in the `argocd-cm` ConfigMap:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: argocd-cm
  namespace: argocd
  labels:
    app.kubernetes.io/name: argocd-cm
    app.kubernetes.io/part-of: argocd
data:
  resource.customizations: |
    db.atlasgo.io/AtlasMigration:
      health.lua: |
        hs = {}
        if obj.status ~= nil then
          if obj.status.conditions ~= nil then
            for i, condition in ipairs(obj.status.conditions) do
              if condition.type == "Ready" and condition.status == "False" then
                hs.status = "Degraded"
                hs.message = condition.message
                return hs
              end
              if condition.type == "Ready" and condition.status == "True" then
                hs.status = "Healthy"
                hs.message = condition.message
                return hs
              end
            end
          end
        end

        hs.status = "Progressing"
        hs.message = "Waiting for reconciliation"
        return hs
```

For `AtlasSchema`, the same check goes under `db.atlasgo.io/AtlasSchema`. The application's health then
shows the Atlas resource's `Ready` message:

```bash
kubectl get -n argocd applications.argoproj.io atlas-argocd-demo -o=jsonpath='{range .status.resources[*]}{"\n"}{.kind}: {"\t"} {.name} {"\t"} ({.status}) {"\t"} ({.health}){end}'
```

### Flux

Flux orders Kustomizations with `dependsOn`, and waits on `healthChecks`. The Kustomization that holds
the Atlas resource depends on the database's, and checks the Atlas resource's health:

```yaml
spec:
  dependsOn:
    - name: mysql
  healthChecks:
    - apiVersion: db.atlasgo.io/v1alpha1
      kind: AtlasSchema
      name: app-schema
      namespace: default
```

The application's Kustomization then depends on this one. Give each Kustomization its own `path`: when
they share a path, the first one already applies every manifest, and the dependencies order nothing.

Tools that read Kubernetes status conventions, such as Flux and Helm 4's `--wait`, also read the
`Ready`, `Reconciling`, and `Stalled` conditions directly.

### Without a GitOps tool

```shell
kubectl apply -f atlas-migration.yaml
kubectl wait --for=condition=Ready atlasmigration/atlasmigration-pg --timeout=2m
```

Run the application's rollout after the wait succeeds. A team that cannot read Events can still wait on
the condition.

### Argo Rollouts

Argo CD orders a `Rollout` like a `Deployment`, so the Atlas resource goes in an earlier sync wave, and
the canary or blue-green rollout starts only after the schema is applied. The operator has no Argo
Rollouts integration beyond that ordering, and progressive delivery adds two rules:

- During the rollout, the old and new versions run against the new schema at once, and a blue-green
  preview runs the new version before promotion. Keep each migration backward compatible: add first,
  and drop or rename only after no running version uses the object (expand and contract).
  `atlas migrate lint` reports backward-incompatible changes in CI
  (https://atlasgo.io/lint/analyzers#backward-incompatible-changes); move such a change to a later
  release.
- Aborting a rollout, or rolling it back, moves traffic to the old version and leaves the schema as it
  is. The schema reverts only when the manifest's tag moves back, and the resource allows it (see
  Rolling Back below). With backward-compatible migrations, the old version keeps working, so fix
  forward.

### Octopus Deploy

Octopus deploys to Argo CD by committing to the Git repository that an Argo CD application syncs
(https://octopus.com/docs/argo-cd/steps):

- Its Update Argo CD Application Image Tags step changes image tags in workload kinds only, not in custom
  resources. It does not move the tag of an `AtlasMigration` or `AtlasSchema`.
- Render the Atlas resource with the Update Argo CD Application Manifests step instead, from a template
  that sets the tag from an Octopus variable. Keep the database URL and the bot token in Kubernetes
  Secrets: Octopus fails a template that references a sensitive variable, so credentials cannot reach
  Git through it.
- Verify the deployment by waiting for Argo CD health. Octopus then waits until the applications are
  synced and every resource is healthy, the Atlas resource included through Argo CD's health check for
  it, so a migration that fails keeps the Octopus deployment from succeeding.

### Crossplane

Crossplane installs the chart with provider-helm, and wraps the Atlas resources in provider-kubernetes
`Object`s, whose service account needs RBAC on the Atlas kinds
(https://atlasgo.io/guides/deploying/crossplane):

```yaml
apiVersion: helm.crossplane.io/v1beta1
kind: Release
metadata:
  name: crossplane-atlas-operator
spec:
  forProvider:
    chart:
      name: atlas-operator
      repository: oci://ghcr.io/ariga/charts
    namespace: atlas-operator-system
  providerConfigRef:
    name: default
```

## Rolling Back

Revert the manifest to the previous tag:

- `AtlasSchema` applies the previous desired state. Removing what the newer version added is
  destructive, so the lint and review policies apply to the rollback too (`references/declarative.md`).
- `AtlasMigration` reverts only with `protectedFlows.migrateDown.allow: true`, and with a registry
  directory, after the revert plan is approved (`references/versioned.md`).

## Helm Hooks and Init Containers

Running `atlas migrate apply` in a Helm hook Job or an init container is deprecated
(https://atlasgo.io/guides/deploying/helm). Neither can tell an upgrade from a rollback, and an older
image does not carry the newer files that a revert needs. Move such setups to the operator. Do not run
the operator itself as an init container: it is a controller that runs alongside the cluster's workloads.
