# Custom Lint Rules on Databricks

Lint runs Atlas's built-in analyzers (`references/migrations.md`), and Databricks has none of its own.
When no analyzer covers a check, or the user states a convention, write a custom rule in Atlas's HCL
rule language. `atlas schema lint` and `atlas migrate lint` enforce it on every change
(https://atlasgo.io/lint/rules). Lint needs `atlas login`. If `atlas whoami` shows no login, ask the user
to run `atlas login`; a user without an account can start a free trial with it.

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

Databricks uses the base rule language, with no Databricks-specific attributes. The attributes each
object exposes, and the child objects a rule can walk into (https://atlasgo.io/hcl/rule#predicate):

| Object | Attributes | Child objects |
|--------|------------|---------------|
| `schema` | `name`, `comment` | `table`, `function`, `procedure`, `permission`, `attr` |
| `table` | `name`, `comment`, `schema`, `primary_key` | `column`, `foreign_key`, `check`, `permission`, `attr` |
| `column` | `name`, `type`, `null`, `default`, `comment`, `primary_key` | `attr` |
| `view` (plain and materialized) | `name`, `query`, `comment` | `column`, `permission`, `attr` |
| `foreign_key` | `name`, `comment`, `table`, `ref_table` | `column`, `ref_column` |
| `check` | `name`, `expr`, `comment` | |
| `function` | `name`, `schema`, `lang`, `return`, `body`, `comment` | `arg`, `attr` |
| `procedure` | `name`, `schema`, `lang`, `body`, `comment` | `arg`, `attr` |
| `arg` | `name`, `type`, `default`, `mode` | |
| `permission` | `grantee`, `privilege`, `grantable` | |
| `attr` | `name`, `value` | |

- `type` compares the type as Databricks writes it: `"BIGINT"`, `"STRING"`, `"DECIMAL(10,2)"`,
  `"ARRAY<STRING>"`.
- `lang` is `"SQL"` or `"PYTHON"`. A privilege reads with spaces, as in SQL: `"ALL PRIVILEGES"`,
  `"USE SCHEMA"`.
- `attr` children expose a comment (`name = "comment"`) and a collation (`name = "collate"`), such as
  `UTF8_LCASE`. A column or table reports a collation only when it differs from the one it inherits.
- Rules cannot read table properties, `location`, `provider`, partitioning, clustering, masks, row
  filters, identity clauses, or routine determinism. Check those in code review, or with statement
  rules on the migration SQL (`references/migrations.md`).
- Foreign keys carry no referential actions on Databricks, so `on_delete` and `on_update` checks match
  nothing.
- Atlas detects no renames on Databricks, so `rename` blocks in a `rule "migrate"` match nothing: a
  rename reaches the rule as a `drop` and an `add` (`references/objects.md`, Renames).

## Examples

Adapt the closest one before writing a rule from scratch (https://atlasgo.io/lint/rules#examples).

A comment on every table:

```hcl
predicate "table" "has_comment" {
  comment {
    ne = null
  }
}

rule "schema" "table-comment" {
  description = "Every table must have a comment"
  table {
    assert {
      predicate = predicate.table.has_comment
      message   = "table ${self.name} must have a comment"
    }
  }
}
```

Monetary columns stored as `DECIMAL`, never `FLOAT` or `DOUBLE`:

```hcl
predicate "column" "is_money" {
  name {
    match = ".*(amount|price|total).*"
  }
}

predicate "column" "is_decimal" {
  type {
    match = "^DECIMAL"
  }
}

rule "schema" "money-decimal" {
  description = "Monetary columns must be DECIMAL"
  table {
    column {
      match {
        predicate = predicate.column.is_money
      }
      assert {
        predicate = predicate.column.is_decimal
        message   = "column ${self.table.name}.${self.name} must be DECIMAL"
      }
    }
  }
}
```

SQL functions only, no Python:

```hcl
predicate "function" "is_sql" {
  lang {
    eq = "SQL"
  }
}

rule "schema" "sql-functions" {
  description = "Functions must be written in SQL"
  schema {
    function {
      assert {
        predicate = predicate.function.is_sql
        message   = "function ${self.name} must be written in SQL"
      }
    }
  }
}
```

No `ALL PRIVILEGES` grants on tables:

```hcl
predicate "permission" "not_all_privileges" {
  privilege {
    ne = "ALL PRIVILEGES"
  }
}

rule "schema" "no-all-privileges" {
  description = "Tables must not grant ALL PRIVILEGES"
  table {
    permission {
      assert {
        predicate = predicate.permission.not_all_privileges
        message   = "a table grants ALL PRIVILEGES to ${self.grantee}"
      }
    }
  }
}
```

PII columns with a comment that marks them:

```hcl
predicate "column" "is_pii" {
  name {
    in = ["email", "phone", "address", "ssn", "dob", "ip_address"]
  }
}

predicate "column" "comment_match" {
  variable "pattern" {
    type = string
  }
  comment {
    match = var.pattern
  }
}

rule "schema" "column-pii" {
  description = "PII columns must have a comment that marks them"
  table {
    column {
      match {
        predicate = predicate.column.is_pii
      }
      assert {
        predicate = predicate.column.comment_match
        vars = {
          pattern = ".*PII.*"
        }
        message = "column ${self.name} must have a comment containing PII"
      }
    }
  }
}
```

A migration rule, reported only for tables the analyzed migrations add:

```hcl
predicate "table" "has_comment" {
  comment {
    ne = null
  }
}

rule "migrate" "added-table-comment" {
  description = "Added tables must have a comment"
  add {
    table {
      assert {
        predicate = predicate.table.has_comment
        message   = "added table ${self.name} must have a comment"
      }
    }
  }
}
```

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
