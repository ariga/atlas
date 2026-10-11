# Custom Lint Rules on SQL Server

Lint runs Atlas's built-in analyzers (`references/migrations.md`), and SQL Server has no analyzers of its
own. When no analyzer covers a check, or the user states a convention, write a custom rule in Atlas's
HCL rule language. A rule can check any attribute of any object Atlas inspects, and `atlas schema lint`
and `atlas migrate lint` enforce it on every change (https://atlasgo.io/lint/rules). Lint needs
`atlas login`, which SQL Server needs anyway.

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

The attributes SQL Server objects expose to predicates, and the child objects a predicate or rule can
walk into:

| Object | Attributes | Child objects |
|--------|------------|---------------|
| `schema` | `name`, `comment` | `table`, `view`, `function`, `procedure`, `sequence`, `type_alias`, `type_table`, `permission` |
| `table` | `name`, `comment`, `schema`, `primary_key`, `memory_optimized`, `fulltext`, `period_name`, `period_type`, `period_start`, `period_end`, `sv_table`, `sv_retention`, `sv_retention_unit` | `column`, `index`, `foreign_key`, `check`, `trigger`, `permission` |
| `column` | `name`, `type`, `null`, `default`, `comment`, `primary_key` | |
| `view` | `name`, `query`, `comment`, `check_option`, `schema`, `schema_bound`, `fulltext` | `column`, `index`, `permission` |
| `index` | `name`, `unique`, `comment` | `part` (`column`, `expr`, `desc`) |
| `foreign_key` | `name`, `comment`, `table`, `ref_table`, `on_update`, `on_delete` | `column`, `ref_column` |
| `check` | `name`, `expr`, `comment` | |
| `trigger` | `name`, `for`, `event`, `body`, `table`, `view`, `comment`, `execute_as` | |
| `function` | `name`, `schema`, `lang`, `return`, `body`, `comment` | `arg` |
| `procedure` | `name`, `schema`, `lang`, `body`, `comment`, `execute_as` | `arg` |
| `arg` | `name`, `type`, `default`, `mode` | |
| `sequence` | `name`, `schema`, `type`, `start`, `increment`, `min`, `max`, `cache`, `cycle` | |
| `type_alias` | `name`, `schema`, `type`, `null` | |
| `type_table` | `name`, `schema`, `primary_key` | `column`, `index`, `check` |
| `role` | `name`, `default_schema` | |
| `user` | `name`, `default_schema`, `login_name`, `default_language`, `allow_encrypted_value_modifications` | |
| `permission` | `grantee`, `privilege`, `grantable`, `table`, `view`, `column`, `schema` | |

- An `index` exposes no `nonclustered` or `type` attribute, so a rule cannot tell clustered from
  nonclustered indexes.
- Test a system-versioned table with `period_name { eq = "SYSTEM_TIME" }`. A comparison on an
  attribute the object does not have, such as `sv_table` on a plain table, matches nothing and reports
  nothing.
- `rule "migrate"` walks schemas, tables, views, functions, procedures, and triggers; sequences and
  types are checked with `rule "schema"`.

## Examples

Adapt the closest one before writing a rule from scratch. Each was checked on SQL Server.

A primary key on every table:

```hcl
predicate "table" "has_primary_key" {
  primary_key {
    condition = self != null
  }
}

rule "schema" "table-has-primary-key" {
  description = "Every table must have a primary key"
  table {
    assert {
      predicate = predicate.table.has_primary_key
      message   = "table ${self.name} has no primary key"
    }
  }
}
```

An index on the columns of every foreign key, which SQL Server does not create on its own. The table
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

Views with `SCHEMABINDING`:

```hcl
predicate "view" "schema_bound" {
  schema_bound {
    eq = true
  }
}

rule "schema" "view-schema-bound" {
  description = "Views must use SCHEMABINDING"
  view {
    assert {
      predicate = predicate.view.schema_bound
      message   = "view ${self.name} must be schema bound"
    }
  }
}
```

No procedure that runs `EXECUTE AS OWNER`:

```hcl
predicate "procedure" "no_execute_as_owner" {
  not {
    execute_as {
      eq = "OWNER"
    }
  }
}

rule "schema" "no-execute-as-owner" {
  description = "Procedures must not run as OWNER"
  schema {
    procedure {
      assert {
        predicate = predicate.procedure.no_execute_as_owner
        message   = "procedure ${self.name} runs as OWNER"
      }
    }
  }
}
```

`bigint` sequences:

```hcl
predicate "sequence" "is_bigint" {
  type {
    eq = "bigint"
  }
}

rule "schema" "sequence-bigint" {
  description = "Sequences must be bigint"
  schema {
    sequence {
      assert {
        predicate = predicate.sequence.is_bigint
        message   = "sequence ${self.name} must be bigint"
      }
    }
  }
}
```

System versioning on every table whose name starts with `price`, using `match` to pick the tables:

```hcl
predicate "table" "name_prices" {
  name {
    match = "^price"
  }
}

predicate "table" "system_versioned" {
  period_name {
    eq = "SYSTEM_TIME"
  }
}

rule "schema" "price-tables-versioned" {
  description = "Price tables must be system-versioned"
  table {
    match {
      predicate = predicate.table.name_prices
    }
    assert {
      predicate = predicate.table.system_versioned
      message   = "table ${self.name} must be system-versioned"
    }
  }
}
```

A migration rule, reported only for tables the analyzed migrations add:

```hcl
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
