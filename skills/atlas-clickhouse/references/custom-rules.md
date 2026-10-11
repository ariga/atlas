# Custom Lint Rules on ClickHouse

Lint runs Atlas's built-in analyzers (`references/migrations.md`). When no analyzer covers a check, or
the user states a convention, write a custom rule in Atlas's HCL rule language. A rule can check the
attributes Atlas exposes for each ClickHouse object, and `atlas schema lint` and `atlas migrate lint`
enforce it on every change (https://atlasgo.io/lint/rules). Lint needs `atlas login`, as the ClickHouse
driver does. If `atlas whoami` shows no login, ask the user to run `atlas login`; a user without an
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
- An attribute that is not set, such as the `ttl` of a table or the `codec` of a column without one, is
  null. `eq`, `match`, and the number comparisons are false on null, so `ttl { match = ".+" }` tests
  that a TTL is set. `contains`, `contains_fold`, `eq_fold`, and `in` expect a string: use `match` on
  attributes that can be unset.

## What a Rule Can Check

The attributes ClickHouse objects expose to predicates, and the child objects a predicate or rule can
walk into (https://atlasgo.io/hcl/rule):

| Object | Attributes | Child objects |
|--------|------------|---------------|
| `schema` | `name`, `comment`, `engine` | `table`, `view`, `dictionary`, `attr` |
| `table` | `name`, `comment`, `schema`, `primary_key`, `engine`, `ttl` | `column`, `index`, `check`, `sort_part`, `attr`, and `permission` in rules |
| `column` | `name`, `type`, `null`, `default`, `comment`, `primary_key`, `codec`, `default_kind` | |
| `view` | `name`, `query`, `comment` | `column`, `index`, `attr` |
| `index` | `name`, `comment`, `unique`, `type`, `granularity` | `part` |
| `index_part` (an index's `part`) | `column`, `expr`, `desc` | |
| `table_sort_part` (a table's `sort_part`) | `column`, `expr` | |
| `check` | `name`, `expr`, `comment` | |
| `dictionary` | `name`, `source`, `layout`, `lifetime_min`, `lifetime_max` | |
| `role` | `name` | `attr` |
| `user` | `name`, `auth_type`, `storage_type` | `attr` |
| `permission` | `grantee`, `privilege`, `grantable` | |

- `engine` is the engine as written, with its arguments, such as `ReplicatedMergeTree` or
  `Distributed('{cluster}', 'app', 'events_local')`: compare it with `match`.
- `codec` is the full clause, such as `CODEC(ZSTD(1))`: compare it with `match`.
- `ttl` and the `expr` of a sort part are read as the server normalizes them, such as
  `date + toIntervalMonth(1)`.

## Examples

From the rule tests and docs (https://atlasgo.io/lint/rules). Adapt the closest one before writing a
rule from scratch.

Tables on a replicated engine:

```hcl
predicate "table" "replicated" {
  engine {
    match = "^Replicated"
  }
}

rule "schema" "replicated-tables" {
  description = "Every table uses a Replicated engine"
  table {
    assert {
      predicate = predicate.table.replicated
      message   = "table ${self.name} must use a Replicated engine"
    }
  }
}
```

A TTL on every MergeTree table:

```hcl
predicate "table" "merge_tree" {
  engine {
    match = "MergeTree"
  }
}

predicate "table" "has_ttl" {
  ttl {
    match = ".+"
  }
}

rule "schema" "merge-tree-ttl" {
  description = "MergeTree tables must set a TTL"
  table {
    match {
      predicate = predicate.table.merge_tree
    }
    assert {
      predicate = predicate.table.has_ttl
      message   = "table ${self.name} must set a TTL"
    }
  }
}
```

A ZSTD codec on every `String` column:

```hcl
predicate "column" "string" {
  type {
    eq = "String"
  }
}

predicate "column" "zstd" {
  codec {
    match = "ZSTD"
  }
}

rule "schema" "string-columns-zstd" {
  description = "String columns must use the ZSTD codec"
  table {
    column {
      match {
        predicate = predicate.column.string
      }
      assert {
        predicate = predicate.column.zstd
        message   = "column ${self.name} of table ${self.table.name} must use the ZSTD codec"
      }
    }
  }
}
```

Tables sorted by `tenant_id`, through `sort_part`:

```hcl
predicate "table" "sorted_by_tenant" {
  exists {
    sort_part {
      condition = self.column != null ? self.column.name == "tenant_id" : false
    }
  }
}

rule "schema" "sort-by-tenant" {
  description = "Tables must sort by tenant_id"
  table {
    assert {
      predicate = predicate.table.sorted_by_tenant
      message   = "table ${self.name} must include tenant_id in its sort key"
    }
  }
}
```

Dictionaries without credentials in their source, and with a bounded lifetime:

```hcl
predicate "dictionary" "no_inline_password" {
  not {
    source {
      match = "(?i)PASSWORD"
    }
  }
}

predicate "dictionary" "short_lifetime" {
  lifetime_max {
    le = 3600
  }
}

rule "schema" "dictionary-hygiene" {
  description = "Dictionaries read credentials from a named collection and refresh within an hour"
  schema {
    dictionary {
      assert {
        predicate = predicate.dictionary.no_inline_password
        message   = "dictionary ${self.name} must not hold a password in its source"
      }
      assert {
        predicate = predicate.dictionary.short_lifetime
        message   = "dictionary ${self.name} must refresh at least hourly"
      }
    }
  }
}
```

Users with a password:

```hcl
predicate "user" "has_password" {
  auth_type {
    ne = "no_password"
  }
}

rule "schema" "users-authenticate" {
  description = "Users must authenticate"
  user {
    assert {
      predicate = predicate.user.has_password
      message   = "user ${self.name} must not use no_password"
    }
  }
}
```

A migration rule, reported only for tables the analyzed migrations add:

```hcl
predicate "table" "replicated" {
  engine {
    match = "^Replicated"
  }
}

rule "migrate" "added-tables-replicated" {
  description = "New tables must use a Replicated engine"
  add {
    table {
      assert {
        predicate = predicate.table.replicated
        message   = "Added table ${self.name} must use a Replicated engine"
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
