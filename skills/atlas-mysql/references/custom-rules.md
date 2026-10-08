# Custom Lint Rules on MySQL

Lint runs Atlas's built-in analyzers (`references/migrations.md`). When no analyzer covers a check, or
the user states a convention, write a custom rule in Atlas's HCL rule language. A rule can check any
attribute of any object Atlas inspects, and `atlas schema lint` and `atlas migrate lint` enforce it on
every change (https://atlasgo.io/lint/rules). Lint needs `atlas login`. If `atlas whoami` shows no
login, ask the user to run `atlas login`; a user without an account can start a free trial with it.

## The Rule Language

Rules live in `.rule.hcl` files. The language reference is https://atlasgo.io/hcl/rule.

- `predicate "<kind>" "<name>"` is a reusable condition on one kind of object, referenced as
  `predicate.<kind>.<name>`. Each block inside it names an attribute and compares it with `eq`, `ne`,
  `lt`, `le`, `gt`, `ge`, `eq_fold`, `contains`, `contains_fold`, `in`, or `match` (a regular
  expression).
- `not`, `or`, and `and` combine conditions. `all`, `any`, `exists`, and `count` quantify over child
  objects, such as the columns of a table.
- `condition` takes an HCL expression over `self`, the current object, for checks the operators cannot
  express: `primary_key { condition = self != null }`.
- `variable` blocks parameterize a predicate, and the caller passes values with `vars = { ... }`.
- `rule "schema" "<name>"` checks the whole desired schema. `rule "migrate" "<name>"` checks only what
  the analyzed migrations change, wrapped in `add`, `drop`, `modify`, or `rename` blocks. Both need a
  `description`, which appears in the report.
- Inside a rule, nested blocks walk the objects, such as `table { column { ... } }`. `match` filters
  them, and `assert` applies a predicate and reports its `message` when the predicate is false.
  `${self.name}` and `${self.table.name}` interpolate into messages.
- An attribute without a block of its own, such as a schema's `charset` or `collate`, is read through
  `attr` children, each with a `name` and a `value` (see the charset example below).

## What a Rule Can Check

The attributes MySQL objects expose to predicates, and the child objects a predicate or rule can walk
into (https://atlasgo.io/hcl/rule#predicate):

| Object | Attributes | Child objects |
|--------|------------|---------------|
| `schema` | `name`, `comment` | `table`, `view`, `function`, `procedure`, `permission`, `attr` |
| `table` | `name`, `comment`, `schema`, `primary_key` | `column`, `index`, `foreign_key`, `check`, `trigger`, `permission`, `attr` |
| `column` | `name`, `type`, `null`, `default`, `comment`, `primary_key` | `permission` |
| `view` | `name`, `query`, `comment`, `security` | `column`, `index`, `trigger`, `permission` |
| `index` | `name`, `comment`, `unique`, `type` | `part` |
| `index_part` (an index's `part`) | `column`, `expr`, `desc`, `prefix` | |
| `foreign_key` | `name`, `comment`, `table`, `ref_table`, `on_update`, `on_delete` | `column`, `ref_column` |
| `check` | `name`, `expr`, `comment` | |
| `trigger` | `name`, `for`, `event`, `body`, `table`, `view`, `comment` | |
| `function` | `name`, `schema`, `lang`, `return`, `body`, `comment`, `deterministic`, `security`, `data_access` | `arg` |
| `procedure` | `name`, `schema`, `lang`, `body`, `comment`, `deterministic`, `security`, `data_access` | `arg` |
| `arg` | `name`, `type`, `default`, `mode` | |
| `role`, `user` | `name`, `account_locked`, `max_connections`, `max_questions`, `max_updates`, `max_user_connections`, `password_expired`, `password_lifetime` | |
| `permission` | `grantee`, `privilege`, `grantable`, `table`, `view`, `column`, `schema` | |
| `attr` | `name`, `value` | |

## Examples

From the docs (https://atlasgo.io/lint/rules#examples). Adapt the closest one before writing a rule
from scratch. InnoDB creates an index for the columns of a foreign key on its own, so the docs' rule
that requires one is not needed on MySQL.

Schemas with an allowed charset and collation, read through `attr` children:

```hcl
variable "allowed_charsets" {
  type    = list(string)
  default = ["utf8mb4"]
}

variable "allowed_collations" {
  type    = list(string)
  default = ["utf8mb4_0900_ai_ci", "utf8mb4_general_ci", "utf8mb4_bin"]
}

predicate "attr" "charset_allowed" {
  name {
    eq = "charset"
  }
  value {
    in = var.allowed_charsets
  }
}

predicate "attr" "collation_allowed" {
  name {
    eq = "collate"
  }
  value {
    in = var.allowed_collations
  }
}

predicate "schema" "has_allowed_charset" {
  exists {
    attr {
      predicate = predicate.attr.charset_allowed
    }
  }
}

predicate "schema" "has_allowed_collation" {
  exists {
    attr {
      predicate = predicate.attr.collation_allowed
    }
  }
}

rule "schema" "schema-charset-collation-allowed" {
  description = "All schemas must use allowed charset and collation"
  schema {
    assert {
      predicate = predicate.schema.has_allowed_charset
      message = "Schema ${self.name} must have an allowed charset"
    }
    assert {
      predicate = predicate.schema.has_allowed_collation
      message = "Schema ${self.name} must have an allowed collation"
    }
  }
}
```

Deterministic functions:

```hcl
predicate "function" "deterministic" {
  deterministic {
    eq = true
  }
}

rule "schema" "all-function-deterministic" {
  description = "All functions must be deterministic"
  schema {
    function {
      assert {
        predicate = predicate.function.deterministic
        message   = "Func ${self.name} must be deterministic"
      }
    }
  }
}
```

Procedures without `OUT` arguments, with `condition` and an HCL function:

```hcl
predicate "procedure" "no_out_arg" {
  not {
    exists {
      arg {
        condition = self.mode != "" && upper(self.mode) == "OUT"
      }
    }
  }
}

rule "schema" "procedures-no-out-arg" {
  description = "Ensure all procedures have no OUT arguments"
  schema {
    procedure {
      assert {
        predicate = predicate.procedure.no_out_arg
        message = "procedure ${self.name} must not have OUT arguments"
      }
    }
  }
}
```

Roles with a connection limit:

```hcl
predicate "role" "has_max_connections" {
  max_connections {
    ne = 0
  }
}

rule "schema" "role-max-connections" {
  description = "Roles must have max_connections configured"
  role {
    assert {
      predicate = predicate.role.has_max_connections
      message   = "role ${self.name} must have max_connections configured"
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

A migration rule, reported only for tables the analyzed migrations add:

```hcl
predicate "table" "has_primary_key" {
  primary_key {
    condition = self != null
  }
}

predicate "column" "not_null" {
  null {
    eq = false
  }
}

rule "migrate" "added-table-has-primary-key-and-not-null-columns" {
  description = "Added tables must have a primary key and all columns must be not null"
  add {
    table {
      assert {
        predicate = predicate.table.has_primary_key
        message   = "Added table ${self.name} must have a primary key"
      }
      column {
        assert {
          predicate = predicate.column.not_null
          message   = "Column ${self.name} in added table ${self.table.name} must not be null"
        }
      }
    }
  }
}
```

The docs have more: comments on PII columns, foreign keys that reference primary keys with
`ON DELETE CASCADE`, function naming, and audit and history tables.

## Wire and Run

Add the rule files to `atlas.hcl`, globally or inside an env (https://atlasgo.io/lint/rules):

```hcl
lint {
  rule "hcl" "custom-rules" {
    src = ["schema.rule.hcl"]
  }
}
```

- The `rule "hcl"` block also takes `vars`, which set the rule file's `variable` blocks, such as
  `allowed_charsets` above.
- `atlas schema lint --env <name>` reports every violation in the schema. `atlas migrate lint` reports
  only the violations that the new migrations introduce.
- Check a new rule before relying on it: run `atlas schema lint --env <name>`, and confirm it reports
  the objects that break it and nothing else.
