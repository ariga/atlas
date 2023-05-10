---
title: CLI Reference
id: cli-reference
slug: cli-reference
---
## Introduction

This document serves as reference documentation for all available commands in the Atlas CLI.
Similar information can be obtained by running any atlas command with the `-h` or `--help`
flags.

For a more detailed introduction to the CLI capabilities, head over to the
[Getting Started](/getting-started/) page.

## Distributed Binaries

The binaries distributed in official releases are released under the  [Apache 2 License](https://github.com/ariga/atlas/blob/master/LICENSE).
If you would like to build Atlas from source follow the instructions [here](https://atlasgo.io/cli-reference#building-from-source).

### Building from Source

If you would like to build Atlas from source without the UI code run:
```shell
go get ariga.io/atlas/cmd/atlas
```

## atlas license

Display license information

#### Usage
```
atlas license
```


## atlas migrate

Manage versioned migration files

#### Usage
```
atlas migrate
```

#### Details
'atlas migrate' wraps several sub-commands for migration management.

#### Flags
```
  -c, --config string        select config (project) file using URL format (default "file://atlas.hcl")
      --env string           set which env from the config file to use
      --var <name>=<value>   input variables (default [])

```


### atlas migrate apply

Applies pending migration files on the connected database.

#### Usage
```
atlas migrate apply [flags] [amount]
```

#### Details
'atlas migrate apply' reads the migration state of the connected database and computes what migrations are pending.
It then attempts to apply the pending migration files in the correct order onto the database. 
The first argument denotes the maximum number of migration files to apply.
As a safety measure 'atlas migrate apply' will abort with an error, if:
  - the migration directory is not in sync with the 'atlas.sum' file
  - the migration and database history do not match each other

If run with the "--dry-run" flag, atlas will not execute any SQL.

#### Example

```
  atlas migrate apply -u mysql://user:pass@localhost:3306/dbname
  atlas migrate apply --dir file:///path/to/migration/directory --url mysql://user:pass@localhost:3306/dbname 1
  atlas migrate apply --env dev 1
  atlas migrate apply --dry-run --env dev 1
```
#### Flags
```
  -u, --url string                [driver://username:password@address/dbname?param=value] select a resource using the URL format
      --dir string                select migration directory using URL format (default "file://migrations")
      --format string             Go template to use to format the output
      --revisions-schema string   name of the schema the revisions table resides in
      --dry-run                   print SQL without executing it
      --lock-timeout duration     set how long to wait for the database lock (default 10s)
      --from string               calculate pending files from the given version (including it)
      --baseline string           start the first migration after the given baseline version
      --tx-mode string            set transaction mode [none, file, all] (default "file")
      --allow-dirty               allow start working on a non-clean database

```


### atlas migrate diff

Compute the diff between the migration directory and a desired state and create a new migration file.

#### Usage
```
atlas migrate diff [flags] [name]
```

#### Details
'atlas migrate diff' uses the dev-database to re-run all migration files in the migration directory, compares
it to a given desired state and create a new migration file containing SQL statements to migrate the migration
directory state to the desired schema. The desired state can be another connected database or an HCL file.

#### Example

```
  atlas migrate diff --dev-url mysql://user:pass@localhost:3306/dev --to file://schema.hcl
  atlas migrate diff --dev-url mysql://user:pass@localhost:3306/dev --to file://atlas.hcl add_users_table
  atlas migrate diff --dev-url mysql://user:pass@localhost:3306/dev --to mysql://user:pass@localhost:3306/dbname
  atlas migrate diff --env dev --format '{{ sql . "  " }}'
```
#### Flags
```
      --to strings                [driver://username:password@address/dbname?param=value] select a desired state using the URL format
      --dev-url string            [driver://username:password@address/dbname?param=value] select a dev database using the URL format
      --dir string                select migration directory using URL format (default "file://migrations")
      --dir-format string         select migration file format (default "atlas")
      --revisions-schema string   name of the schema the revisions table resides in
  -s, --schema strings            set schema names
      --lock-timeout duration     set how long to wait for the database lock (default 10s)
      --format string             Go template to use to format the output
      --qualifier string          qualify tables with custom qualifier when working on a single schema
      --edit                      edit the generated migration file(s)

```


### atlas migrate hash

Hash (re-)creates an integrity hash file for the migration directory.

#### Usage
```
atlas migrate hash [flags]
```

#### Details
'atlas migrate hash' computes the integrity hash sum of the migration directory and stores it in the atlas.sum file.
This command should be used whenever a manual change in the migration directory was made.

#### Example

```
  atlas migrate hash
```
#### Flags
```
      --dir string          select migration directory using URL format (default "file://migrations")
      --dir-format string   select migration file format (default "atlas")

```


### atlas migrate import

Import a migration directory from another migration management tool to the Atlas format.

#### Usage
```
atlas migrate import [flags]
```

#### Example

```
  atlas migrate import --from file:///path/to/source/directory?format=liquibase --to file:///path/to/migration/directory
```
#### Flags
```
      --from string         select migration directory using URL format (default "file://migrations")
      --to string           select migration directory using URL format (default "file://migrations")
      --dir-format string   select migration file format (default "atlas")

```


### atlas migrate lint

Run analysis on the migration directory

#### Usage
```
atlas migrate lint [flags]
```

#### Example

```
  atlas migrate lint --env dev
  atlas migrate lint --dir file:///path/to/migration/directory --dev-url mysql://root:pass@localhost:3306 --latest 1
  atlas migrate lint --dir file:///path/to/migration/directory --dev-url mysql://root:pass@localhost:3306 --git-base master
  atlas migrate lint --dir file:///path/to/migration/directory --dev-url mysql://root:pass@localhost:3306 --format '{{ json .Files }}'
```
#### Flags
```
      --dev-url string      [driver://username:password@address/dbname?param=value] select a dev database using the URL format
      --dir string          select migration directory using URL format (default "file://migrations")
      --dir-format string   select migration file format (default "atlas")
      --format string       Go template to use to format the output
      --latest uint         run analysis on the latest N migration files
      --git-base string     run analysis against the base Git branch
      --git-dir string      path to the repository working directory (default ".")

```


### atlas migrate new

Creates a new empty migration file in the migration directory.

#### Usage
```
atlas migrate new [flags] [name]
```

#### Details
'atlas migrate new' creates a new migration according to the configured formatter without any statements in it.

#### Example

```
  atlas migrate new my-new-migration
```
#### Flags
```
      --dir string          select migration directory using URL format (default "file://migrations")
      --dir-format string   select migration file format (default "atlas")
      --edit                edit the created migration file(s)

```


### atlas migrate set

Set the current version of the migration history table.

#### Usage
```
atlas migrate set [flags] [version]
```

#### Details
'atlas migrate set' edits the revision table to consider all migrations up to and including the given version
to be applied. This command is usually used after manually making changes to the managed database.

#### Example

```
  atlas migrate set 3 --url mysql://user:pass@localhost:3306/
  atlas migrate set --env local
  atlas migrate set 1.2.4 --url mysql://user:pass@localhost:3306/my_db --revision-schema my_revisions
```
#### Flags
```
  -u, --url string                [driver://username:password@address/dbname?param=value] select a resource using the URL format
      --dir string                select migration directory using URL format (default "file://migrations")
      --dir-format string         select migration file format (default "atlas")
      --revisions-schema string   name of the schema the revisions table resides in

```


### atlas migrate status

Get information about the current migration status.

#### Usage
```
atlas migrate status [flags]
```

#### Details
'atlas migrate status' reports information about the current status of a connected database compared to the migration directory.

#### Example

```
  atlas migrate status --url mysql://user:pass@localhost:3306/
  atlas migrate status --url mysql://user:pass@localhost:3306/ --dir file:///path/to/migration/directory
```
#### Flags
```
  -u, --url string                [driver://username:password@address/dbname?param=value] select a resource using the URL format
      --format string             Go template to use to format the output
      --dir string                select migration directory using URL format (default "file://migrations")
      --dir-format string         select migration file format (default "atlas")
      --revisions-schema string   name of the schema the revisions table resides in

```


### atlas migrate validate

Validates the migration directories checksum and SQL statements.

#### Usage
```
atlas migrate validate [flags]
```

#### Details
'atlas migrate validate' computes the integrity hash sum of the migration directory and compares it to the
atlas.sum file. If there is a mismatch it will be reported. If the --dev-url flag is given, the migration
files are executed on the connected database in order to validate SQL semantics.

#### Example

```
  atlas migrate validate
  atlas migrate validate --dir file:///path/to/migration/directory
  atlas migrate validate --dir file:///path/to/migration/directory --dev-url mysql://user:pass@localhost:3306/dev
  atlas migrate validate --env dev
```
#### Flags
```
      --dev-url string      [driver://username:password@address/dbname?param=value] select a dev database using the URL format
      --dir string          select migration directory using URL format (default "file://migrations")
      --dir-format string   select migration file format (default "atlas")

```


## atlas schema

Work with atlas schemas.

#### Usage
```
atlas schema
```

#### Details
The `atlas schema` command groups subcommands working with declarative Atlas schemas.

#### Flags
```
  -c, --config string        select config (project) file using URL format (default "file://atlas.hcl")
      --env string           set which env from the config file to use
      --var <name>=<value>   input variables (default [])

```


### atlas schema apply

Apply an atlas schema to a target database.

#### Usage
```
atlas schema apply [flags]
```

#### Details
'atlas schema apply' plans and executes a database migration to bring a given
database to the state described in the provided Atlas schema. Before running the
migration, Atlas will print the migration plan and prompt the user for approval.

The schema is provided by one or more URLs (to a HCL file or 
directory, database or migration directory) using the "--to, -t" flag:
  atlas schema apply -u URL --to file://file1.hcl --to file://file2.hcl
  atlas schema apply -u URL --to file://schema/ --to file://override.hcl

As a convenience, schema URLs may also be provided via an environment definition in
the project file (see: https://atlasgo.io/cli/projects).

If run with the "--dry-run" flag, atlas will exit after printing out the planned
migration.

#### Example

```
  atlas schema apply -u "mysql://user:pass@localhost/dbname" --to file://atlas.hcl
  atlas schema apply -u "mysql://localhost" --to file://schema.hcl --schema prod --schema staging
  atlas schema apply -u "mysql://user:pass@localhost:3306/dbname" --to file://schema.hcl --dry-run
  atlas schema apply -u "mariadb://user:pass@localhost:3306/dbname" --to file://schema.hcl
  atlas schema apply --url "postgres://user:pass@host:port/dbname?sslmode=disable" --to file://schema.hcl
  atlas schema apply -u "sqlite://file:ex1.db?_fk=1" --to file://schema.hcl
```
#### Flags
```
  -u, --url string        [driver://username:password@address/dbname?param=value] select a resource using the URL format
      --to strings        [driver://username:password@address/dbname?param=value] select a desired state using the URL format
      --exclude strings   list of glob patterns used to filter resources from applying
  -s, --schema strings    set schema names
      --dev-url string    [driver://username:password@address/dbname?param=value] select a dev database using the URL format
      --dry-run           print SQL without executing it
      --auto-approve      apply changes without prompting for approval
      --format string     Go template to use to format the output

```


### atlas schema clean

Removes all objects from the connected database.

#### Usage
```
atlas schema clean [flags]
```

#### Details
'atlas schema clean' drops all objects in the connected database and leaves it in an empty state.
As a safety feature, 'atlas schema clean' will ask for confirmation before attempting to execute any SQL.

#### Example

```
  atlas schema clean -u mysql://user:pass@localhost:3306/dbname
  atlas schema clean -u mysql://user:pass@localhost:3306/
```
#### Flags
```
  -u, --url string     [driver://username:password@address/dbname?param=value] select a resource using the URL format
      --auto-approve   apply changes without prompting for approval

```


### atlas schema diff

Calculate and print the diff between two schemas.

#### Usage
```
atlas schema diff [flags]
```

#### Details
'atlas schema diff' reads the state of two given schema definitions, 
calculates the difference in their schemas, and prints a plan of
SQL statements to migrate the "from" database to the schema of the "to" database.
The database states can be read from a connected database, an HCL project or a migration directory.

#### Example

```
  atlas schema diff --from mysql://user:pass@localhost:3306/test --to file://schema.hcl
  atlas schema diff --from mysql://user:pass@localhost:3306 --to file://schema_1.hcl --to file://schema_2.hcl
  atlas schema diff --from mysql://user:pass@localhost:3306 --to file://migrations --format '{{ sql . "  " }}'
```
#### Flags
```
  -f, --from strings      [driver://username:password@address/dbname?param=value] select a resource using the URL format
      --to strings        [driver://username:password@address/dbname?param=value] select a desired state using the URL format
      --dev-url string    [driver://username:password@address/dbname?param=value] select a dev database using the URL format
  -s, --schema strings    set schema names
      --exclude strings   list of glob patterns used to filter resources from applying
      --format string     Go template to use to format the output

```


### atlas schema fmt

Formats Atlas HCL files

#### Usage
```
atlas schema fmt [path ...]
```

#### Details
'atlas schema fmt' formats all ".hcl" files under the given paths using
canonical HCL layout style as defined by the github.com/hashicorp/hcl/v2/hclwrite package.
Unless stated otherwise, the fmt command will use the current directory.

After running, the command will print the names of the files it has formatted. If all
files in the directory are formatted, no input will be printed out.



### atlas schema inspect

Inspect a database and print its schema in Atlas DDL syntax.

#### Usage
```
atlas schema inspect [flags]
```

#### Details
'atlas schema inspect' connects to the given database and inspects its schema.
It then prints to the screen the schema of that database in Atlas DDL syntax. This output can be
saved to a file, commonly by redirecting the output to a file named with a ".hcl" suffix:

  atlas schema inspect -u "mysql://user:pass@localhost:3306/dbname" > schema.hcl

This file can then be edited and used with the `atlas schema apply` command to plan
and execute schema migrations against the given database. In cases where users wish to inspect
all multiple schemas in a given database (for instance a MySQL server may contain multiple named
databases), omit the relevant part from the url, e.g. "mysql://user:pass@localhost:3306/".
To select specific schemas from the databases, users may use the "--schema" (or "-s" shorthand)
flag.
	

#### Example

```
  atlas schema inspect -u "mysql://user:pass@localhost:3306/dbname"
  atlas schema inspect -u "mariadb://user:pass@localhost:3306/" --schema=schemaA,schemaB -s schemaC
  atlas schema inspect --url "postgres://user:pass@host:port/dbname?sslmode=disable"
  atlas schema inspect -u "sqlite://file:ex1.db?_fk=1"
```
#### Flags
```
  -u, --url string        [driver://username:password@address/dbname?param=value] select a resource using the URL format
      --dev-url string    [driver://username:password@address/dbname?param=value] select a dev database using the URL format
  -s, --schema strings    set schema names
      --exclude strings   list of glob patterns used to filter resources from applying
      --format string     Go template to use to format the output

```


## atlas version

Prints this Atlas CLI version information.

#### Usage
```
atlas version
```



