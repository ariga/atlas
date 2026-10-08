# Custom Lint Rules on PostgreSQL

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

## What a Rule Can Check

The attributes PostgreSQL objects expose to predicates, and the child objects a predicate or rule can
walk into (https://atlasgo.io/hcl/rule#predicate):

| Object | Attributes | Child objects |
|--------|------------|---------------|
| `schema` | `name`, `comment` | `table`, `view`, `function`, `procedure`, `permission`, `policy` |
| `table` | `name`, `comment`, `schema`, `primary_key`, `row_security_enabled`, `row_security_enforced` | `column`, `index`, `foreign_key`, `check`, `trigger`, `permission`, `policy` |
| `column` | `name`, `type`, `null`, `default`, `comment`, `primary_key` | `permission` |
| `view` | `name`, `query`, `comment`, `security_barrier`, `security_invoker` | `column`, `index`, `trigger`, `permission` |
| `index` | `name`, `comment`, `unique`, `type`, `where`, `nulls_distinct` | `part` |
| `foreign_key` | `name`, `comment`, `table`, `ref_table`, `on_update`, `on_delete`, `deferrable`, `initially_deferred` | `column`, `ref_column` |
| `check` | `name`, `expr`, `comment` | |
| `trigger` | `name`, `for`, `event`, `body`, `table`, `view`, `comment`, `deferrable`, `initially_deferred` | |
| `function` | `name`, `schema`, `lang`, `return`, `body`, `comment`, `security`, `volatility`, `strict`, `leakproof`, `parallel` | `arg` |
| `procedure` | `name`, `schema`, `lang`, `body`, `comment`, `security` | `arg` |
| `arg` | `name`, `type`, `default`, `mode` | |
| `policy` | `name`, `comment`, `for`, `using`, `check`, `restrictive` | |
| `role`, `user` | `name`, `login`, `superuser`, `create_db`, `create_role`, `inherit`, `replication`, `bypass_rls`, `conn_limit` | |
| `permission` | `grantee`, `privilege`, `grantable`, `table`, `view`, `column`, `schema` | |

## Examples

From the docs (https://atlasgo.io/lint/rules#examples). Adapt the closest one before writing a rule
from scratch.

Row-level security enabled and enforced on every table:

```hcl
predicate "table" "has_row_security_enabled" {
  row_security_enabled {
    eq  = true
  }
  row_security_enforced {
    eq  = true
  }
}

rule "schema" "ensure-row-security-enabled" {
  description = "All tables must have row security enabled and enforced"
  table {
    assert {
      predicate = predicate.table.has_row_security_enabled
    }
  }
}
```

An index on the columns of every foreign key, which PostgreSQL does not create on its own. The table
predicate takes a variable, and `condition` compares the index parts with an HCL expression:

```hcl
predicate "foreign_key" "columns_are_indexed" {
  table {
    predicate = predicate.table.has_columns_index
    vars = {
      columns = [for c in self.columns: c]
    }
  }
}

predicate "table" "has_columns_index" {
  variable "columns" {
    type = list(string)
  }
  any {
    index {
      condition = alltrue([for p in self.parts : p.expr == null]) && [for p in self.parts : p.column] == var.columns
    }
  }
}

rule "schema" "require-index-fk-columns" {
  description = "Require index on columns used by foreign keys"
  table {
    foreign_key {
      assert {
        predicate = predicate.foreign_key.columns_are_indexed
        message   = "Missing index for columns used by foreign-key ${self.name}"
      }
    }
  }
}
```

Views with invoker security:

```hcl
predicate "view" "security_invoker_enabled" {
  security_invoker {
    eq = true
  }
}

rule "schema" "view-security-invoker" {
  description = "All views must have invoker security"
  view {
    assert {
      predicate = predicate.view.security_invoker_enabled
      message = "view \"${self.name}\" must have invoker security"
    }
  }
}
```

No role with `SUPERUSER`:

```hcl
predicate "role" "not_superuser" {
  superuser {
    eq = false
  }
}

rule "schema" "no-superuser" {
  description = "Roles must not have SUPERUSER privilege"
  role {
    assert {
      predicate = predicate.role.not_superuser
      message   = "role ${self.name} must not have SUPERUSER"
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

The docs have more: deferrable triggers and foreign keys, users with a connection limit, admin roles
without `SUPERUSER`, audit and history tables, and comments on PII columns.

## Wire and Run

Add the rule files to `atlas.hcl`, globally or inside an env (https://atlasgo.io/lint/rules):

```hcl
lint {
  rule "hcl" "custom-rules" {
    src = ["schema.rule.hcl"]
  }
}
```

- The `rule "hcl"` block also takes `vars`, which set the rule file's `variable` blocks.
- `atlas schema lint --env <name>` reports every violation in the schema. `atlas migrate lint` reports
  only the violations that the new migrations introduce.
- Check a new rule before relying on it: run `atlas schema lint --env <name>`, and confirm it reports
  the objects that break it and nothing else.
