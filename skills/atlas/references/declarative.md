# Declarative Workflow Reference (`atlas schema apply`, `atlas schema plan`)

In the declarative workflow the desired schema (HCL, SQL, ORM, or another database) is the source of
truth. `atlas schema apply` inspects the target, plans the SQL that moves it to the desired state,
asks for approval, and runs it. `atlas schema plan` moves the planning and approval earlier: the plan
is reviewed and approved in a pull request, stored in the Atlas Registry, and `schema apply` later runs
exactly those statements without re-planning.

## `atlas schema apply`

```bash
atlas schema apply --env <name> --dry-run                 # print the plan, run nothing
atlas schema apply --env <name>                           # plan, lint, ask, apply
atlas schema apply --env <name> --auto-approve            # no prompt (development only)
atlas schema apply --env <name> --edit                    # edit the plan in $EDITOR before it runs
atlas schema apply --env <name> --plan "atlas://app/plans/<name>"   # run a specific approved plan
atlas schema apply --url "$URL" --to file://schema.sql --dev-url docker://postgres/17/dev
```

| Flag | Purpose |
|------|---------|
| `--url`, `-u` | Target database |
| `--to` | Desired state: HCL, SQL, ORM loader, migration directory (`file://migrations`), or another database. Repeatable |
| `--dev-url` | Dev database used to normalize the desired state and to lint the plan |
| `--schema`, `--exclude`, `--include` | Limit the scope |
| `--dry-run` | Print SQL and the lint report, apply nothing |
| `--auto-approve` | Skip the prompt. Never on shared environments unless the user asks |
| `--plan` | Use a named pre-approved plan from the registry |
| `--tx-mode` | `file` (default: one transaction) or `none`. A plan directive `atlas:txmode none` has the same effect |
| `--format` | `{{ json . }}` for machine-readable output |

Approval happens in one of three ways:

1. Manual: Atlas prints the SQL and prompts. The default.
2. Review policy: `lint { review = ERROR }` (or `WARNING`) auto-approves plans that lint clean and
   prompts only when the report has errors (or warnings). `ALWAYS` is the default and always prompts.
   Requires `atlas login`: once the policy is in `atlas.hcl`, every `schema apply` for that env,
   including `--dry-run`, aborts without one.
3. A pre-approved plan for this exact transition exists in the registry: Atlas applies it with no
   prompt and no re-planning.

The `diff` block (`skip`, `concurrent_index`, `materialized`, `add_table`, `drop_table`, ...) shapes the
plan the same way it shapes `migrate diff`; see `references/versioned.md`, Diff policy. A
`check "schema_apply"` block with `allow`/`deny` rules and `lint` policy apply before the plan runs,
and a drift comparison against the desired state is inherent: every apply re-plans from the live state.

## `atlas schema plan`

A plan is a file with three attributes: `from` and `to`, fingerprints of the schema states, and
`migration`, the SQL. `schema apply` looks up an approved plan whose `from` matches the current state
and whose `to` matches the desired state, and runs its `migration` verbatim. If the database moved,
the `from` no longer matches and the plan is not used.

```hcl
plan "20260923085308" {
  from      = "vJYpErjN4kWJpw4nRaJcEX3xx/jExj4a05Ll3Y7gXr4="
  to        = "B5OVckDEeHcaSdYCUMEfYe8CZN85ahLkef44hfwCe2g="
  migration = <<-SQL
  -- Add column "email" to table: "users"
  ALTER TABLE "users" ADD COLUMN "email" text NOT NULL DEFAULT 'unknown';
  UPDATE "users" SET "email" = "name" || '@example.com' WHERE "email" = 'unknown';
  SQL
}
```

Prerequisites: `atlas login` (every `schema plan` subcommand needs it, including `--dry-run` and
`--save`), and the env's `schema` block names a registry repo:

```hcl
env "dev" {
  dev = "docker://postgres/17/dev?search_path=public"
  schema {
    src = "file://schema.sql"
    repo {
      name = "app"
    }
  }
}
```

Create the repo with the first `atlas schema push --env dev` (or `atlas cloud repo create --type schema`).

### Lifecycle

```bash
atlas schema plan --env dev                          # plan from the registry's latest state (or url) to src,
                                                     # lint it, prompt "Approve and push", store as APPROVED
atlas schema plan --env dev --pending                # store as PENDING for someone else to approve
atlas schema plan --env dev --dry-run                # print the plan only
atlas schema plan --env dev --save -o add_email.plan.hcl   # write the file locally instead of pushing
atlas schema plan --env dev --edit                   # open in $EDITOR, then approve and push
atlas schema plan --env dev --name add_email         # explicit plan name in the registry
atlas schema plan --env dev --from "$URL"            # plan from a live database instead of the registry
atlas schema plan --env dev -d "atlas:txmode none"   # add a directive to the plan

atlas schema plan lint     --env dev --file file://add_email.plan.hcl
atlas schema plan validate --env dev --file file://add_email.plan.hcl   # from/to still match?
atlas schema plan push     --env dev --file file://add_email.plan.hcl   # push an edited local plan
atlas schema plan push     --env dev --file file://add_email.plan.hcl --pending
atlas schema plan pull     --url "atlas://app/plans/add_email" > add_email.plan.hcl
atlas schema plan list     --env dev                 # plans for the current transition
atlas schema plan list     --env dev --pending       # only the ones waiting for approval
atlas schema plan approve  --url "atlas://app/plans/add_email"
atlas schema plan rm       --url "atlas://app/plans/add_email"
atlas schema plan test     --env dev                 # run test "plan" cases (references/testing.md)
atlas schema plan new      --env dev --name add_email   # like plan, always creates a new plan file
```

| Flag (plan, new, push, validate) | Purpose |
|------|---------|
| `--from`, `--to` | Override the transition states. `--from` defaults to the env `url`, then the registry's last known state; `--to` to `schema.src` |
| `--repo` | Registry repo URL when the env has no `schema.repo` |
| `--name`, `--name-format` | Plan name; format is a Go template, e.g. `plan_{{ slice .ToHash 0 8 }}` |
| `--pending` | Push in `PENDING` state |
| `--auto-approve` | Approve without the prompt |
| `--skip-lint` | Skip the lint step |
| `--directive`, `-d` | Add `atlas:nolint ...` or `atlas:txmode none` directives to the plan |
| `--push`, `--save`, `-o` | Push to the registry, or save to a file |

### Editing a plan

Three ways, all ending with the plan in the registry:

1. `atlas schema plan --env dev --edit`: edit in place, then approve and push.
2. `atlas schema plan --env dev --save`, edit the file, `atlas schema plan push --env dev --file file://<path>`.
3. `atlas schema plan pull --url atlas://app/plans/<name> > x.plan.hcl`, edit the `migration`
   attribute only, `atlas schema plan push --env dev --file file://x.plan.hcl`.

Edit only `migration`. Atlas re-checks that the edited SQL still brings `from` to `to`; a plan that
lands somewhere else is rejected as drift. This is how a data backfill (`UPDATE ...`) is added to a
column-add plan so both run together at deploy time.

### Approval and multiple plans

- `atlas schema plan` pushes as `APPROVED` unless `--pending`. A `PENDING` plan is approved with
  `atlas schema plan approve --url ...`, in the registry UI, or by the `schema/plan/approve` CI step.
- Protected flows in the registry restrict who may push schemas, push approved plans, or approve.
- If two approved plans exist for the same transition (one per environment, say), `schema apply`
  aborts with "multiple pre-planned migrations were found". Pass `--plan <name>` or `rm` the extra one.
- Approve only when the user asks. Approving authorizes the SQL to run on the next apply.

## `atlas schema push`

```bash
atlas schema push --env dev app                       # push src to the registry, tag = git commit
atlas schema push --env dev app --tag v1.2.0
atlas schema push --env dev app --version 20260301120000 --desc "add email"
```

The pushed schema is what `schema plan` uses as the last known state, what `schema apply` can deploy
from (`--to "atlas://app?tag=latest"`), and what the Kubernetes operator and Terraform provider read.

## Agent Workflow

Local development (no registry):

1. Edit the schema source, `atlas schema validate --env <name>`.
2. `atlas schema apply --env <name> --dry-run`. Read the SQL to the user.
3. `atlas schema apply --env <name>` (or `--auto-approve` on a throwaway local database).

Reviewed change (registry, plan workflow):

1. Edit the schema source, validate, then `atlas schema plan --env <name> --dry-run` and show the SQL.
2. If the plan needs data statements, `--save`, add them to `migration`, `plan lint`, then `plan push`.
3. Push with `--pending` unless the user is the approver. Report the plan URL.
4. The user (or CI on merge) approves. Deploy with `atlas schema apply --env <prod>`; Atlas picks up
   the approved plan for that transition.
5. To see what is waiting: `atlas schema plan list --env <name> --pending` (`references/cloud.md`).

## Error Handling

| Error | Action |
|-------|--------|
| `The plan "From" hash does not match the current state hash` | The database (or `--from`) is not at the plan's source state. Re-plan from the real database URL, or pass the same URL the plan was made with |
| `multiple pre-planned migrations were found in the registry` | `atlas schema apply --plan <name>` or `atlas schema plan rm --url ...` |
| Plan rejected after editing (drift) | The edited SQL does not reach `to`. Keep the DDL Atlas planned; add only data statements |
| `schema.repo` not set / repository not found | Add `schema { repo { name = "..." } }`, run `atlas schema push` once, or `--repo` |
| Prompt appears in CI | Set `lint { review = ERROR }` (needs `atlas login` in CI), pre-approve a plan, or pass `--auto-approve` only where the pipeline is the approver |
| `available only to Atlas Pro users` | `schema plan`, the review policy, and `schema test` need a login. Run `atlas login` |

## Documentation

- [Declarative apply](https://atlasgo.io/declarative/apply)
- [Review policy](https://atlasgo.io/declarative/apply#review-policy)
- [Pre-planning and approving migrations](https://atlasgo.io/declarative/plan)
- [Plan file reference](https://atlasgo.io/hcl/plan)
- [Plan testing](https://atlasgo.io/testing/plan)
- [Schema diff](https://atlasgo.io/declarative/diff)
- [Declarative CI/CD setup](https://atlasgo.io/declarative/setup-cicd)
