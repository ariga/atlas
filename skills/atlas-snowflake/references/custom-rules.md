# Custom Lint Rules on Snowflake

Lint runs Atlas's built-in analyzers (`references/migrations.md`). When no analyzer covers a check, or
the user states a convention, write a custom rule in Atlas's HCL rule language, and `atlas schema lint`
and `atlas migrate lint` enforce it on every change (https://atlasgo.io/lint/rules). Lint needs
`atlas login`. If `atlas whoami` shows no login, ask the user to run `atlas login`; a user without an
account can start a free trial with it.

Snowflake uses the base rule language, with no Snowflake extension: rules see the attributes every
database shares, not Snowflake-only ones such as `transient`, `retention_time`, `cluster_by`, tags,
policies, or a table's kind. For those, use `statement` rules on the SQL of each migration (Statement
Rules below).

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

| Object | Attributes | Child objects |
|--------|------------|---------------|
| `schema` | `name`, `comment` | `table`, `function`, `procedure`, `permission`, `attr` |
| `table` | `name`, `comment`, `schema`, `primary_key` | `column`, `index`, `foreign_key`, `permission`, `attr` |
| `column` | `name`, `type`, `null`, `default`, `comment`, `primary_key` | `permission`, `attr` |
| `view` | `name`, `query`, `comment` | `column`, `permission` |
| `index` | `name`, `comment`, `unique` | `part` |
| `index_part` (an index's `part`) | `column`, `expr`, `desc` | |
| `foreign_key` | `name`, `comment`, `table`, `ref_table` | `column`, `ref_column` |
| `function` | `name`, `schema`, `lang`, `return`, `body`, `comment` | `arg` |
| `procedure` | `name`, `schema`, `lang`, `body`, `comment` | `arg` |
| `arg` | `name`, `type`, `default`, `mode` | |
| `role`, `user` | `name` | |
| `permission` | `grantee`, `privilege`, `grantable`, and the `table`, `view`, or `schema` it is on | |
| `attr` | `name`, `value` | |

What these mean on Snowflake:

- `table` covers every table kind: regular, transient, dynamic, hybrid, external, and Iceberg.
- Unique constraints are `index` objects with `unique = true`; Snowflake has no other indexes.
- `type` is the type as Atlas formats it, such as `NUMBER(38)` or `VARCHAR(255)`.
- `lang` is `SQL`, `JAVASCRIPT`, `PYTHON`, `JAVA`, or `SCALA`.
- `attr` children are named `comment`, and `collate` for a column with a collation.
- Names are case-sensitive, as stored: compare with `eq_fold` or a `(?i)` pattern when case does not
  matter.
- Roles and grants are present only when the env enables `roles` and `permissions` at database scope
  (`references/security.md`).

## Examples

Adapt the closest one before writing a rule from scratch (https://atlasgo.io/lint/rules#examples).

Uppercase table and column names, so unquoted SQL reaches them:

```hcl
predicate "table" "upper_name" {
  name {
    match = "^[A-Z][A-Z0-9_]*$"
  }
}

predicate "column" "upper_name" {
  name {
    match = "^[A-Z][A-Z0-9_]*$"
  }
}

rule "schema" "uppercase-identifiers" {
  description = "Tables and columns use unquoted-safe uppercase names"
  table {
    assert {
      predicate = predicate.table.upper_name
      message   = "table ${self.name} must be uppercase"
    }
    column {
      assert {
        predicate = predicate.column.upper_name
        message   = "column ${self.name} in ${self.table.name} must be uppercase"
      }
    }
  }
}
```

A comment on every column whose name marks personal data:

```hcl
predicate "column" "is_pii" {
  name {
    match = "(?i)^(email|phone|ssn|address|dob)$"
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
  description = "Columns with personal data carry a PII comment"
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

Functions in approved languages only:

```hcl
predicate "function" "approved_lang" {
  lang {
    in = ["SQL", "PYTHON"]
  }
}

rule "schema" "function-languages" {
  description = "Functions are written in SQL or Python"
  schema {
    function {
      assert {
        predicate = predicate.function.approved_lang
        message   = "function ${self.name} uses an unapproved language"
      }
    }
  }
}
```

A migration rule, reported only for tables the analyzed migrations add:

```hcl
predicate "table" "has_comment" {
  comment {
    # A table without a comment never matches.
    match = ".+"
  }
}

rule "migrate" "added-table-has-comment" {
  description = "Added tables must have a comment"
  add {
    table {
      assert {
        predicate = predicate.table.has_comment
        message   = "Added table ${self.name} must have a comment"
      }
    }
  }
}
```

## Statement Rules

For what rules cannot see, the `statement` analyzer matches each statement of a migration file against
ordered `allow` and `deny` regular expressions (https://atlasgo.io/lint/analyzers#statement-rules). It
warns by default; `error = true` fails lint, and `force = true` stops `-- atlas:nolint` from skipping it:

```hcl
lint {
  statement {
    error = true
    deny "no-transient" {
      match   = "(?i)CREATE\\s+TRANSIENT\\s+(TABLE|SCHEMA)"
      message = "transient objects have no fail-safe period"
    }
    deny "no-replace-table" {
      match   = "(?i)CREATE\\s+OR\\s+REPLACE\\s+(TRANSIENT\\s+)?TABLE"
      message = "CREATE OR REPLACE TABLE drops the table's data"
    }
    deny "no-zero-retention" {
      match   = "(?i)DATA_RETENTION_TIME_IN_DAYS\\s*=\\s*0\\b"
      message = "tables keep at least one day of time travel"
    }
  }
}
```

A violation reads
`statement was denied by rule "no-transient": transient objects have no fail-safe period`.

## Wire and Run

Add the rule files to `atlas.hcl`, globally or inside an env (https://atlasgo.io/lint/rules):

```hcl
lint {
  rule "hcl" "custom-rules" {
    src = ["schema.rule.hcl"]
  }
}
```

- `atlas schema lint --env <name>` reports every violation in the schema. `atlas migrate lint` reports
  only the violations that the new migrations introduce.
- Check a new rule before relying on it: run `atlas schema lint --env <name>`, and confirm it reports the
  objects that break it and nothing else.
