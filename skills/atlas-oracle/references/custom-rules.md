# Custom Lint Rules on Oracle

Lint runs Atlas's built-in analyzers, and Oracle has none of its own (`references/migrations.md`). When
no analyzer covers a check, or the user states a convention, write a custom rule in Atlas's HCL rule
language. A rule can check any attribute of any object Atlas inspects, and `atlas schema lint` and
`atlas migrate lint` enforce it on every change (https://atlasgo.io/lint/rules). Lint needs
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
  express: `primary_key { condition = self != null }`.
- `variable` blocks parameterize a predicate, and the caller passes values with `vars = { ... }`.
- `rule "schema" "<name>"` checks the whole desired schema. `rule "migrate" "<name>"` checks only what
  the analyzed migrations change, wrapped in `add`, `drop`, `modify`, or `rename` blocks. Both need a
  `description`, which appears in the report.
- Inside a rule, nested blocks walk the objects, such as `table { column { ... } }`. `match` filters
  them, and `assert` applies a predicate and reports its `message` when the predicate is false.
  `${self.name}` and `${self.table.name}` interpolate into messages.

## What a Rule Can Check

Oracle objects expose the attributes common to every driver
(https://atlasgo.io/hcl/rule#predicate). Driver-specific ones from other databases are rejected, such as
`deterministic` on a function: `unknown block type deterministic in block of type predicate`.

| Object | Attributes | Child objects |
|--------|------------|---------------|
| `schema` | `name`, `comment` | `table`, `view`, `function`, `procedure`, `permission`, `attr` |
| `table` | `name`, `comment`, `schema`, `primary_key` | `column`, `index`, `foreign_key`, `check`, `trigger`, `permission`, `attr` |
| `column` | `name`, `type`, `null`, `default`, `comment`, `primary_key` | `permission` |
| `index` | `name`, `comment`, `unique` | `part` |
| `index_part` (an index's `part`) | `column`, `expr`, `desc` | |
| `foreign_key` | `name`, `comment`, `table`, `ref_table`, `on_delete` | `column`, `ref_column` |
| `check` | `name`, `expr`, `comment` | |
| `trigger` | `name`, `for`, `event`, `body`, `table`, `view`, `comment` | |
| `function` | `name`, `return`, `body`, `comment`, `schema` | `arg` |
| `procedure` | `name`, `body`, `comment`, `schema` | `arg` |
| `permission` | `grantee`, `privilege`, `grantable`, `table`, `view`, `column`, `schema` | |

- A column's `type` is the type as Oracle reports it, such as `NUMBER(10)`, `VARCHAR2(100 CHAR)`,
  `BOOLEAN`, or `JSON`, which `match` can test.
- A table's `primary_key` is an index object with `parts`. `unique` constraints are exposed as indexes
  of the table.

## Examples

Each rule below was checked against an Oracle schema; adapt the closest one before writing a rule from
scratch.

Foreign-key columns that lead an index. Without one, updating a parent key or deleting a parent row
locks the whole child table, and Oracle's guidance is to index foreign keys
(https://docs.oracle.com/en/database/oracle/oracle-database/26/cncpt/data-concurrency-and-consistency.html).
An index, a unique constraint, or the primary key covers a foreign key when its leading columns are the
foreign key's columns, which is less strict than the docs' rule that needs an exact match:

```hcl
predicate "table" "fk_columns_lead_an_index" {
  variable "columns" {
    type = list(string)
  }
  or {
    primary_key {
      condition = self != null && slice([for p in self.parts : p.column], 0, min(length(self.parts), length(var.columns))) == var.columns
    }
    any {
      index {
        condition = slice([for p in self.parts : p.column], 0, min(length(self.parts), length(var.columns))) == var.columns
      }
    }
  }
}

predicate "foreign_key" "is_indexed" {
  table {
    predicate = predicate.table.fk_columns_lead_an_index
    vars = {
      columns = [for c in self.columns : c]
    }
  }
}

rule "schema" "fk-indexed" {
  description = "Foreign-key columns must lead an index, or Oracle locks the child table when a parent key changes"
  table {
    foreign_key {
      assert {
        predicate = predicate.foreign_key.is_indexed
        message   = "foreign key ${self.name} on ${self.table.name} has no index that starts with its columns"
      }
    }
  }
}
```

Uppercase names. Atlas quotes every name it writes, so a lowercase name must be quoted in every query
(`references/objects.md`, Names):

```hcl
predicate "table" "uppercase_name" {
  name {
    match = "^[A-Z][A-Z0-9_$#]*$"
  }
}

predicate "column" "uppercase_name" {
  name {
    match = "^[A-Z][A-Z0-9_$#]*$"
  }
}

rule "schema" "uppercase-identifiers" {
  description = "Names must be uppercase: Atlas quotes every identifier, so a lowercase name must be quoted in every query"
  table {
    assert {
      predicate = predicate.table.uppercase_name
      message   = "table ${self.name} must be uppercase"
    }
    column {
      assert {
        predicate = predicate.column.uppercase_name
        message   = "column ${self.table.name}.${self.name} must be uppercase"
      }
    }
  }
}
```

Character length semantics, so `VARCHAR2(n)` holds n characters of multibyte text:

```hcl
predicate "column" "char_length_semantics" {
  or {
    not {
      type {
        match = "^N?VARCHAR2"
      }
    }
    type {
      match = "CHAR\\)$"
    }
  }
}

rule "schema" "varchar2-char-semantics" {
  description = "VARCHAR2 lengths must count characters, not bytes"
  table {
    column {
      assert {
        predicate = predicate.column.char_length_semantics
        message   = "column ${self.table.name}.${self.name} must be VARCHAR2(n CHAR)"
      }
    }
  }
}
```

Types the target's release lacks, when the dev database is newer, such as a 19c target planned on the
26ai Free image (`references/dev-database.md`, Versions and Settings):

```hcl
predicate "column" "supported_on_19c" {
  not {
    type {
      match = "^(BOOLEAN|JSON|VECTOR)"
    }
  }
}

rule "schema" "oracle-19c-types" {
  description = "The target runs Oracle 19c, which has no BOOLEAN, JSON, or VECTOR columns"
  table {
    column {
      assert {
        predicate = predicate.column.supported_on_19c
        message   = "column ${self.table.name}.${self.name} uses a type that Oracle 19c does not support"
      }
    }
  }
}
```

A primary key on every table:

```hcl
predicate "table" "has_primary_key" {
  primary_key {
    condition = self != null
  }
}

rule "schema" "table-has-primary-key" {
  description = "Every table needs a primary key"
  table {
    assert {
      predicate = predicate.table.has_primary_key
      message   = "table ${self.name} has no primary key"
    }
  }
}
```

The docs have more, such as comments on PII columns, foreign keys that reference primary keys with
`ON DELETE CASCADE`, naming conventions, and audit tables (https://atlasgo.io/lint/rules#examples).

## Wire and Run

Add the rule files to `atlas.hcl`, globally or inside an env (https://atlasgo.io/lint/rules):

```hcl
lint {
  rule "hcl" "oracle" {
    src = ["oracle.rule.hcl"]
  }
}
```

- `atlas schema lint --env <name> --url env://src` reports every violation in the schema.
  `atlas migrate lint` reports only the violations that the new migrations introduce.
- Check a new rule before relying on it: run `atlas schema lint`, and confirm it reports the objects
  that break it and nothing else.
- Regular expressions in rules and in `statement` blocks use RE2 syntax, which has no lookahead. Express
  "but not" with an `allow` rule before a `deny` rule, or with `or` and `not` in a predicate, as above.
