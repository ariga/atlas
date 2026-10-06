# Schema Sources Reference

## HCL Schema

```hcl
data "hcl_schema" "<name>" {
  path = "schema.hcl"
}

env "<name>" {
  schema {
    src = data.hcl_schema.<name>.url
  }
}
```

## External Schema (ORM Integration)

The `external_schema` data source imports SQL schema from an ORM or external program.

```hcl
# GORM (Go)
data "external_schema" "gorm" {
  program = ["go", "run", "-mod=mod", "ariga.io/atlas-provider-gorm", "load", "--path", "./models", "--dialect", "postgres"]
}

# Drizzle (TypeScript)
data "external_schema" "drizzle" {
  program = ["npx", "drizzle-kit", "export"]
}

# SQLAlchemy (Python)
data "external_schema" "sqlalchemy" {
  program = ["atlas-provider-sqlalchemy", "--path", "./models", "--dialect", "postgresql"]
}

# Django (Python)
data "external_schema" "django" {
  program = ["python", "manage.py", "atlas-provider-django", "--dialect", "postgresql"]
}

# Ent (Go)
env "<name>" {
  schema {
    src = "ent://ent/schema"
  }
}

# Sequelize (Node.js)
data "external_schema" "sequelize" {
  program = ["npx", "@ariga/atlas-provider-sequelize", "load", "--path", "./models", "--dialect", "postgres"]
}

# TypeORM (TypeScript)
data "external_schema" "typeorm" {
  program = ["npx", "@ariga/atlas-provider-typeorm", "load", "--path", "./entities", "--dialect", "postgres"]
}
```

Wire into an environment:
```hcl
env "<name>" {
  schema {
    src = data.external_schema.<orm>.url
  }
}
```

## Composite Schema

Combine multiple schema sources into one:

```hcl
data "composite_schema" "app" {
  schema "users" {
    url = data.external_schema.auth_service.url
  }
  schema "graph" {
    url = "ent://ent/schema"
  }
  schema "shared" {
    url = "file://schema/shared.hcl"
  }
}
```

## Dev-Database Dialects

The dev URL format depends on whether your project uses **schema-scoped** or **database-scoped** migrations. Getting this wrong causes errors like `modify schema "public" is not allowed when migration plan is scoped to one schema`, or silently drops database-level objects (extensions, event triggers) from migrations.

**Schema-scoped** (single schema — most common): include the database name and schema scope so Atlas creates objects in the correct schema. Use this when all tables live in one schema (e.g., `public`).

| Dialect    | Dev URL (schema-scoped)                              |
|------------|------------------------------------------------------|
| MySQL      | `docker://mysql/8/dev`                               |
| MariaDB    | `docker://maria/latest/dev`                          |
| PostgreSQL | `docker://postgres/17/dev?search_path=public`        |
| SQLite     | `sqlite://dev?mode=memory`                           |
| SQL Server | `docker://sqlserver/2022-latest/dev?mode=schema`     |
| ClickHouse | `docker://clickhouse/23.11/dev`                      |

**Database-scoped** (multiple schemas or database-level objects): omit the schema scope so Atlas can manage multiple schemas and detect database-level objects like extensions and event triggers.

| Dialect    | Dev URL (database-scoped)                            |
|------------|------------------------------------------------------|
| MySQL      | `docker://mysql/8`                                   |
| MariaDB    | `docker://maria/latest`                              |
| PostgreSQL | `docker://postgres/17/dev`                           |
| SQL Server | `docker://sqlserver/2022-latest/dev?mode=database`   |
| ClickHouse | `docker://clickhouse/23.11`                          |

**PostgreSQL with extensions** — use PostGIS or pgvector images when the schema uses those extensions:
```
docker://postgis/latest/dev?search_path=public
docker://pgvector/pg17/dev?search_path=public
```

**How to choose:** Check the project's `atlas.hcl` or target database URL. If it includes `search_path=public` (Postgres) or a specific database name (MySQL), use schema-scoped. If the project manages multiple schemas, extensions, or event triggers, use database-scoped.

See https://atlasgo.io/concepts/dev-database for additional drivers and options.
