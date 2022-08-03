---
id: diff
slug: /versioned/diff
title: Planning new migrations
---
With the `atlas migrate` commands users can implement a kind fo workflow that we
call _versioned migration authoring_. This kind of workflow is a synthesis between _declarative_ workflows,
where developers specify the desired state of their database, and _versioned migrations_
where each change is explicitly defined as a migration script with a specific version.

Practically speaking, this means that the developer maintains the [schema definition](/atlas-schema/sql-resources),
e.g the _desired state_, and Atlas maintains the `migrations/` directory, which contains the
explicit SQL scripts to move from one version to the next.

In addition, Atlas maintains a file name `atlas.sum` which is used to ensure the integrity of
the migration directory and force developers to deal with situations where migration order or
contents was modified after the fact.

[Learn more about versioned migration authoring](/concepts/declarative-vs-versioned#migration-authoring)

### Flags
When using `migrate diff` to plan a migration users must supply multiple parameters:
* `--dev-url` a [URL](/concepts/url) to a [Dev-database](/concepts/dev-database) that will be used
 to compute the diff.
* `--dir` the URL of the migrations directory, by default it is `file://migrations`, e.g a
 directory named `migrations` in the current working directory.
* `--to` the URL of the desired state, can be an HCL file or another database.

### Example
```
atlas migrate diff --dir file://my/project/migrations --to schema.hcl --dev-url mysql://root:pass@localhost:3306/dev
```

### Reference

[CLI Command Reference](/cli-reference#atlas-migrate-diff)