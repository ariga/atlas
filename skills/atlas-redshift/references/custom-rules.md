# Custom Lint Rules on Redshift

Lint runs Atlas's built-in analyzers (`references/migrations.md`). When no analyzer covers a check, or
the user states a convention, write a custom rule in Atlas's HCL rule language. A rule can check any
attribute of any object Atlas inspects, and `atlas schema lint` and `atlas migrate lint` enforce it on
every change (https://atlasgo.io/lint/rules). Redshift has no analyzers of its own, so distribution
keys, sort keys, encodings, and identity settings are checked only by custom rules. Lint needs
`atlas login`. If `atlas whoami` shows no login, ask the user to run `atlas login`; a user without an
account can start a free trial with it.

## The Rule Language

Rules live in `.rule.hcl` files. The language reference is https://atlasgo.io/hcl/rule.

- `predicate "<kind>" "<name>"` is a reusable condition on one kind of object, referenced as
  `predicate.<kind>.<name>`. Each block inside it names an attribute and compares it with `eq`, `ne`,
  `lt`, `le`, `gt`, `ge`, `eq_fold`, `contains`, `contains_fold`, `in`, or `match` (a regular
  expression).
- `not`, `or`, and `and` combine conditions. `all`, `any`, `exists`, and `count` quantify over child
  objects, such as the columns of a table.
- `condition` takes an HCL expression over `self`, the current object, for checks the operators cannot
  express.
- `variable` blocks parameterize a predicate, and the caller passes values with `vars = { ... }`.
- `rule "schema" "<name>"` checks the whole desired schema. `rule "migrate" "<name>"` checks only what
  the analyzed migrations change, wrapped in `add`, `drop`, `modify`, or `rename` blocks. Both need a
  `description`, which appears in the report.
- Inside a rule, nested blocks walk the objects, such as `table { column { ... } }`. `match` filters
  them, and `assert` applies a predicate and reports its `message` when the predicate is false.
  `${self.name}` and `${self.table.name}` interpolate into messages.

## What a Rule Can Check

The attributes Redshift objects expose to predicates, and the child objects a predicate or rule can
walk into (https://atlasgo.io/hcl/rule#predicate):

| Object | Attributes | Child objects |
|--------|------------|---------------|
| `schema` | `name`, `comment` | `table`, `view`, `function`, `procedure`, `permission` |
| `table` | `name`, `comment`, `schema`, `dist_style`, `dist_column`, `sort_style` | `column`, `sort_column`, `permission` |
| `column` | `name`, `type`, `null`, `default`, `comment`, `encode`, `identity_generated`, `identity_seed`, `identity_step` | `permission` |
| `view` | `name`, `query`, `comment` | `column`, `permission` |
| `function` | `name`, `schema`, `lang`, `return`, `body`, `comment` | `arg` |
| `procedure` | `name`, `schema`, `lang`, `body`, `comment` | `arg` |
| `arg` | `name`, `type`, `default`, `mode` | |
| `role`, `user` | `name` | |
| `permission` | `grantee`, `privilege`, `grantable` | |

- `dist_style` is `AUTO`, `EVEN`, `ALL`, or `KEY`; `sort_style` is `AUTO`, `COMPOUND`, or
  `INTERLEAVED`. Both are compared as strings. `dist_column` is the distribution key column, checked
  with a `column` predicate or a `condition`.
- `encode` is the column's encoding, as inspection reports it (`RAW`, `AZ64`, `LZO`, `ZSTD`, ...).
- `identity_generated` is `ALWAYS` or `BY_DEFAULT`, and null for a column without an identity, where
  every string operator is false; `identity_seed` and `identity_step` are numbers.
- `sort_column` walks the columns of the table's sort key.
- Materialized views are walked as `view`.
- Redshift tables have no indexes, keys, checks, or triggers in Atlas, so predicates on them never
  match.

## Examples

Adapt the closest one before writing a rule from scratch.

No interleaved sort keys:

```hcl
predicate "table" "not_interleaved" {
  sort_style {
    ne = "INTERLEAVED"
  }
}

rule "schema" "no-interleaved-sort-keys" {
  description = "Tables must not use INTERLEAVED sort keys"
  table {
    assert {
      predicate = predicate.table.not_interleaved
      message   = "table ${self.name} uses an INTERLEAVED sort key"
    }
  }
}
```

Fact tables distributed by key, matched by name:

```hcl
predicate "table" "is_fact" {
  name {
    match = "^fact_"
  }
}

predicate "table" "key_distributed" {
  dist_style {
    eq = "KEY"
  }
}

rule "schema" "fact-tables-distkey" {
  description = "Fact tables must be distributed by a key column"
  table {
    match {
      predicate = predicate.table.is_fact
    }
    assert {
      predicate = predicate.table.key_distributed
      message   = "fact table ${self.name} must set distribution { style = KEY, column = ... }"
    }
  }
}
```

No `RAW` encoding outside the sort key, for new tables only:

```hcl
predicate "column" "compressed" {
  encode {
    ne = "RAW"
  }
}

predicate "column" "raw" {
  encode {
    eq = "RAW"
  }
}

rule "migrate" "added-columns-compressed" {
  description = "Columns of added tables must be compressed unless they are in the sort key"
  add {
    table {
      column {
        match {
          predicate = predicate.column.raw
        }
        assert {
          predicate = predicate.column.compressed
          message   = "column ${self.name} in ${self.table.name} is RAW; set encode, or add it to the sort key"
        }
      }
    }
  }
}
```

Redshift picks `RAW` for sort key columns and some types (`references/objects.md`), so review the
first report and narrow the `match` before turning a rule like this into an error.

Identity columns generated `ALWAYS`:

```hcl
predicate "column" "identity_always" {
  identity_generated {
    eq = "ALWAYS"
  }
}

predicate "column" "has_identity" {
  identity_generated {
    in = ["ALWAYS", "BY_DEFAULT"]
  }
}

rule "schema" "identity-always" {
  description = "Identity columns must be GENERATED ALWAYS"
  table {
    column {
      match {
        predicate = predicate.column.has_identity
      }
      assert {
        predicate = predicate.column.identity_always
        message   = "identity column ${self.table.name}.${self.name} must be GENERATED ALWAYS"
      }
    }
  }
}
```

No grantable table permissions:

```hcl
predicate "permission" "not_grantable" {
  grantable {
    eq = false
  }
}

rule "schema" "no-grantable-perms" {
  description = "Table permissions must not be grantable"
  table {
    permission {
      assert {
        predicate = predicate.permission.not_grantable
        message   = "permission ${self.privilege} on ${self.table.name} must not be grantable"
      }
    }
  }
}
```

## Wire and Run

Add the rule files to `atlas.hcl`, globally or inside an env (https://atlasgo.io/lint/rules):

```hcl
lint {
  rule "hcl" "redshift-conventions" {
    src = ["schema.rule.hcl"]
  }
}
```

- The `rule "hcl"` block also takes `vars`, which set the rule file's `variable` blocks.
- `atlas schema lint --env <name>` reports every violation in the schema. `atlas migrate lint` reports
  only the violations that the new migrations introduce.
- Check a new rule before relying on it: run `atlas schema lint --env <name>`, and confirm it reports
  the objects that break it and nothing else.
