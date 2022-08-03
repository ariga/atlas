---
id: apply
slug: /declarative/apply
title: Declarative schema migrations
---

With Atlas, users do not need to plan database schema changes themselves. Instead
of figuring out the correct SQL statements to get their database to the desired state,
Atlas supports a kind of workflow that we call _declarative schema migration_.
With declarative schema migrations the user provides a connection string to the target database
and the desired schema and Atlas does all of the planning.

[Read more about declarative workflows](/concepts/declarative-vs-versioned)

### Auto-approval
Before executing the migration against the target database, Atlas will print the SQL
statements that it is going to run and prompt the user for approval. Users that wish
to automatically approve may run the `schema apply` command with the `--auto-approve`
flag.

### Dry-runs
In order to skip the execution of the SQL queries against the target database,
users may provide the `--dry-run` flag. When invoked with this flag, Atlas will
connect to the target database, inspect its current state, calculate the diff
between the provided desired schema and print out a series of SQL statements to
reconcile any gaps between the inspected and desired schemas.

### Dev-database
When storing schema definitions, many database engines perform some form of
normalization. That is, despite us providing a specific definition of some
aspect of the schema, the database will store it in another, equivalent form.
This means in certain situations it may appear to Atlas as if some diff exists
between the desired and inspected schemas, whereas in reality there is none.

To overcome these situations, users may use the `--dev-url` flag to provide
Atlas with a connection string to a _Dev-Database_.
This database is used to normalize the schema prior to planning migrations and
for simulating changes to ensure their applicability before execution.

[Read more about Dev-Databases](/concepts/dev-database)

### Reference

[CLI Command Reference](/cli-reference#atlas-schema-apply)