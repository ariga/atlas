---
id: lint
slug: /versioned/lint
title: Verifying migration safety
---
With the `atlas migrate lint` command, users can analyze the migration directory
to detect potentially dangerous changes to the database schema. This command may be
incorporated in _continuous integration_ pipelines to enable teams to enforce desired
policies with regard to schema changes. 

[Learn more about Atlas's analyzers](/lint/analyzers)

### Flags
When using `migrate lint` to analyze migrations, users must supply multiple parameters:
* `--dev-url` a [URL](/concepts/url) to a [Dev-database](/concepts/dev-database) that will be used
 to simulate the changes and verify their correctness.
* `--dir` the URL of the migration directory, by default it is `file://migrations`, e.g a
 directory named `migrations` in the current working directory.

### Changeset detection

When we run the `lint` command, we need to instruct Atlas on how to decide what 
set of migration files to analyze. Currently, two modes are supported.

* `--git-base <branchName>`: which selects the diff between the provided branch and 
 the current one as the changeset.
* `--latest <n>` which selects the latest n migration files as the changeset.

### Output

Users may supply a [Go template](https://pkg.go.dev/text/template) string as the `--log` parameter to
format the output of the `lint` command.

### Examples

Analyze all changes relative to the `master` Git branch:
```shell
atlas migrate lint \
  --dir "file://my/project/migrations" \
  --dev-url "mysql://root:pass@localhost:3306/dev" \
  --git-base "master"
```
Analyze the latest 2 migration files:
```shell
atlas migrate lint \
  --dir "file://my/project/migrations" \
  --dev-url "mysql://root:pass@localhost:3306/dev" \
  --latest 2
```
Format output as JSON:
```shell
atlas migrate lint \
  --dir "file://my/project/migrations" \
  --dev-url "mysql://root:pass@localhost:3306/dev" \
  --git-base "master" \
  --log "{{ json .Files }}"
```

### Reference

[CLI Command Reference](/cli-reference#atlas-migrate-lint)