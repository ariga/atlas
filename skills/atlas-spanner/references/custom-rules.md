# Custom Lint Rules on Spanner

Lint runs Atlas's built-in analyzers, and Spanner has none of its own (`references/migrations.md`). When
no analyzer covers a check, or the user states a convention, write a custom rule in Atlas's HCL rule
language. `atlas schema lint` and `atlas migrate lint` enforce it on every change
(https://atlasgo.io/lint/rules). Lint needs `atlas login`. If `atlas whoami` shows no login, ask the
user to run `atlas login`; a user without an account can start a free trial with it.

## The Rule Language

Rules live in `.rule.hcl` files. The language reference is https://atlasgo.io/hcl/rule.

- `predicate "<kind>" "<name>"` is a reusable condition on one kind of object, referenced as
  `predicate.<kind>.<name>`. Each block inside it names an attribute and compares it with `eq`, `ne`,
  `lt`, `le`, `gt`, `ge`, `eq_fold`, `contains`, `contains_fold`, `in`, or `match` (a regular
  expression).
- `not`, `or`, and `and` combine conditions. `all`, `any`, `exists`, and `count` quantify over child
  objects, such as the columns of a table.
- `condition` takes an HCL expression over `self`, the current object:
  `primary_key { condition = self != null }`.
- `variable` blocks parameterize a predicate. The caller passes values with `vars = { ... }`, on an
  `assert` or on a nested block, where `self` is the caller's object.
- `rule "schema" "<name>"` checks the whole desired schema. `rule "migrate" "<name>"` checks only what
  the analyzed migrations change, wrapped in `add`, `drop`, `modify`, or `rename` blocks. Both need a
  `description`, which appears in the report.
- Inside a rule, nested blocks walk the objects, such as `table { column { ... } }`. `match` filters
  them, and `assert` applies a predicate and reports its `message` when the predicate is false.
  `${self.name}` and `${self.table.name}` interpolate into messages.

## What a Rule Can Read on Spanner

These were checked against GoogleSQL and PostgreSQL-dialect schemas:

| Object | Readable | Not readable |
|--------|----------|--------------|
| `table` | `name`, `schema`, `primary_key` (with `parts`), child `column`, `index`, and `foreign_key` blocks | `columns` and `indexes` as expressions; interleaving; row deletion policy |
| `column` | `name`, `type`, `null`, `default`, `self.table.primary_key` | The `primary_key` attribute: `primary_key { eq = true }` never matches; options such as `allow_commit_timestamp` |
| `index` | `name`, `unique`, `parts` with `column`, `desc`, and `expr` | The types of its columns; `null_filtered`, `STORING` columns, interleaving |

- A column's `type` is the type as Spanner reports it: `INT64`, `STRING(36)`, `TIMESTAMP`, or
  `ARRAY<STRING(64)>` in GoogleSQL, and `bigint`, `character varying(36)`, or
  `timestamp with time zone` in the PostgreSQL dialect.
- A column's `default` is the expression text, such as
  `GET_NEXT_SEQUENCE_VALUE(SEQUENCE OrderSeq)`, or `nextval('order_seq'::text)` in the PostgreSQL
  dialect.
- A rule that reads an attribute the object lacks reports
  `Unsupported attribute; This object does not have an attribute named "..."`. When one rule in a file
  fails this way, other rules in the same file can stop reporting, so check each new rule alone first.

## Examples

Each rule below was checked against a Spanner schema; adapt the closest one before writing a rule from
scratch. Both enforce Google's guidance to keep new rows from landing on one split: no keys whose first
part grows with every insert, such as a timestamp or an id the application increments
(https://docs.cloud.google.com/spanner/docs/schema-design).

A single-column `INT64` primary key that takes its values from a bit-reversed sequence. Interleaved
tables have two or more key columns, so the rule skips them:

```hcl
predicate "table" "single_column_key" {
  primary_key {
    condition = self != null && length(self.parts) == 1
  }
}

# The column is not the named key column, is not INT64, or gets its value from a sequence.
predicate "column" "not_a_plain_int64_key" {
  variable "key" {
    type = string
  }
  or {
    name {
      ne = var.key
    }
    type {
      ne = "INT64"
    }
    default {
      match = "GET_NEXT_SEQUENCE_VALUE"
    }
  }
}

predicate "table" "key_from_sequence" {
  variable "key" {
    type = string
  }
  all {
    column {
      predicate = predicate.column.not_a_plain_int64_key
      vars = {
        key = var.key
      }
    }
  }
}

rule "schema" "int64-key-from-sequence" {
  description = "A single-column INT64 primary key must take its values from a bit-reversed sequence"
  table {
    match {
      predicate = predicate.table.single_column_key
    }
    assert {
      predicate = predicate.table.key_from_sequence
      vars = {
        key = self.primary_key.parts[0].column
      }
      message = "table ${self.name}: the INT64 primary key has no GET_NEXT_SEQUENCE_VALUE default, so increasing ids written by the application land on one split"
    }
  }
}
```

A primary key that starts with a `TIMESTAMP` column:

```hcl
predicate "column" "is_timestamp" {
  type {
    eq = "TIMESTAMP"
  }
}

predicate "column" "not_named" {
  variable "name" {
    type = string
  }
  name {
    ne = var.name
  }
}

rule "schema" "no-leading-timestamp-key" {
  description = "A primary key must not start with a TIMESTAMP column"
  table {
    column {
      match {
        predicate = predicate.column.is_timestamp
      }
      assert {
        predicate = predicate.column.not_named
        vars = {
          name = self.table.primary_key.parts[0].column
        }
        message = "the primary key of ${self.table.name} starts with the TIMESTAMP column ${self.name}, so new rows land on one split"
      }
    }
  }
}
```

For the PostgreSQL dialect, change `"INT64"` to `"bigint"`, `"TIMESTAMP"` to
`"timestamp with time zone"`, and `"GET_NEXT_SEQUENCE_VALUE"` to `"nextval"`.

An index that starts with a timestamp hotspots the same way, but an index cannot read the types of its
columns. Match the first column's name against the team's naming convention for timestamps instead:

```hcl
predicate "table" "no_index_led_by_a_time_column" {
  all {
    index {
      condition = !can(regex("(At|_at|Time)$", self.parts[0].column))
    }
  }
}

rule "schema" "no-leading-timestamp-index" {
  description = "An index must not start with a timestamp column, by the naming convention"
  table {
    assert {
      predicate = predicate.table.no_index_led_by_a_time_column
      message   = "table ${self.name} has an index that starts with a timestamp column"
    }
  }
}
```

The docs have more, such as naming conventions and foreign keys that reference primary keys
(https://atlasgo.io/lint/rules#examples).

## Wire and Run

Add the rule files to `atlas.hcl`, globally or inside an env (https://atlasgo.io/lint/rules):

```hcl
env "dev" {
  src = "file://schema.sql"
  dev = "docker://spanner/1.5.58"
  lint {
    rule "hcl" "spanner" {
      src = ["spanner.rule.hcl"]
    }
  }
}
```

- `atlas schema lint --env <name> --url env://src` reports every violation in the schema.
  `atlas migrate lint` reports only the violations that the new migrations introduce.
- Check a new rule before relying on it: run `atlas schema lint` on a schema that breaks it, and confirm
  it reports those objects and nothing else.
- Regular expressions in rules and in `statement` blocks use RE2 syntax, which has no lookahead. Express
  "but not" with an `allow` rule before a `deny` rule, or with `or` and `not` in a predicate.
- Statement rules, which match the SQL text of migration files, catch what object rules cannot, such
  as `DROP CHANGE STREAM` (`references/migrations.md`).
