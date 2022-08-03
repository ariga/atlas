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

### Reference

[CLI Command Reference](/cli-reference#atlas-schema-apply)
