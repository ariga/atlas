# Testing Changes and Rollouts

The operator applies what it is given, without a dry run. Every check runs before the change reaches the
cluster, from the CLI on a dev database, and every rollout is verified from the resource's status. Make
both a habit: a change is done when the tests pass and the resource reports `Ready=True`.

## Before the Push

The `atlas` skill covers these commands in depth; run them against a dev database on the target's engine
version, such as `docker://postgres/16/dev`.

### Declarative (AtlasSchema)

1. Validate the desired state: `atlas schema validate --env local`.
2. Test the logic it holds: `atlas schema test --env local` runs `test "schema"` cases for functions,
   views, triggers, and checks (https://atlasgo.io/testing/schema).
3. Review the plan from the deployed version, as the operator will compute it:

   ```shell
   atlas schema plan --env local \
     --from "atlas://k8s-quickstart?tag=latest" \
     --to env://schema.src \
     --name drop-email --push
   ```

   A plan that is approved and pushed here is applied by the operator without stopping
   (`references/declarative.md`).

### Versioned (AtlasMigration)

1. Validate the directory: `atlas migrate validate --env local`.
2. Lint the new files: `atlas migrate lint --env local --latest 1`, in CI on every pull request.
3. Test data migrations with seeded rows: `atlas migrate test --env local`
   (https://atlasgo.io/testing/migrate).
4. Preview what a database will run: `atlas migrate apply --dry-run` against a staging database, or a
   restored copy, never production from a laptop.

## Validating the Manifest

Check the resource against the installed CRDs without applying it:

```shell
kubectl apply --dry-run=server -f atlas-schema.yaml
```

The server rejects unknown fields, wrong types, and invalid values, such as an `AtlasDriftCheck` interval
under `1m`. Then check that every Secret the resource references exists, with the right keys:

```shell
kubectl describe secret postgres-credentials
```

`kubectl describe` lists the keys and their sizes, not the values. Never print a Secret's data.

## After the Rollout

Wait for the resource, then check what it applied:

```shell
kubectl wait --for=condition=Ready atlasmigration/atlasmigration-pg --timeout=2m
kubectl get atlasmigration atlasmigration-pg -o jsonpath='{.status.lastAppliedVersion}{"\n"}'
```

- `AtlasMigration`: `status.lastAppliedVersion` is the last file of the deployed tag.
- `AtlasSchema`: the `Ready` message carries the applied statements; a plan-based apply sets
  `status.planLink`. Atlas Cloud shows each run, with its statements and logs.
- With an `AtlasDriftCheck`, `kubectl get atlasdriftchecks` shows the database clean after the rollout.
- With a scan triggered by the resource, `Compliant` shows whether the new version stays within policy.
- A resource that does not become ready within the timeout is a failed rollout: read its reason
  (`references/troubleshooting.md`), and keep the application on its previous version.

## Trying the Operator Itself

Try a new operator version, chart values, or policies in a throwaway cluster first, as the quickstarts do
(https://atlasgo.io/integrations/kubernetes/quickstart): a local cluster, a database in a Deployment, and
the same resources pointed at it.
