---
id: inspect
slug: /declarative/inspect
title: Inspecting existing schemas with Atlas 
---

### Automatic Schema Inspection
Many projects begin with an existing database that users wish to start managing
with Atlas. In this case, instead of having developers learn the [Atlas Language](/atlas-schema/sql-resources)
and reverse engineer a schema definition file that precisely describes the existing database,
Atlas supports _automatic schema inspection_.

With automatic schema inspection, users simply provide Atlas with a connection string
to their target database and Atlas prints out a schema definition file in the Atlas
language that they can use as the starting point for working with this database.

### Flags

When using `schema inspect` to inspect an existing database, users may supply multiple
parameters:
* `--url` (required, `-u` accepted as well) - the [URL](/concepts/url) of database to be inspected.
* `--schema` (optional, may be supplied multiple times) - schemas to inspect within 
 the target database. 

### Examples

MySQL (entire database):
```
atlas schema inspect -u "mysql://user:pass@localhost:3306"
```

A single schema (`test`):
```
atlas schema inspect --url mysql://root:pass@localhost:3306/test
```

Multiple schemas:
```
atlas schema inspect --url mysql://root:pass@localhost:3306/ --schema test --schema test_2
```

PostgreSQL (without SSL):
```text
 atlas schema inspect --url "postgres://user:pass@host:port/dbname?sslmode=disable"
```

### Reference

[CLI Command Reference](/cli-reference#atlas-schema-apply)
