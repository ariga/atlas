---
title: CLI Reference
id: cli-reference
slug: /cli/reference
---
## Introduction

This document serves as reference documentation for all available commands in the Atlas CLI.
Similar information can be obtained by running any atlas command with the `-h` or `--help`
flags.

For a more detailed introduction to the CLI capabilities, head over to the
[Getting Started](getting-started/01-introduction.mdx) page.

## Distributed Binaries

Starting [v0.3.0](https://github.com/ariga/atlas/releases/tag/v0.3.0),
ֿthe distributed binaries include code for a [Management UI](ui/intro.md) wrapping the
core atlas engine that is not currently released publicly. The binaries
themselves are still released under the same [Apache License 2.0](https://github.com/ariga/atlas/blob/master/LICENSE).

### Buliding from Source

If you would like to build Atlas from source without the UI code run:
```shell
go get ariga.io/atlas/cmd/atlas
```

## atlas env

Print atlas environment variables.

#### Usage
```
atlas env
```

#### Details
`atlas env`prints atlas environment information.

Every set environment param will be printed in the form of NAME=VALUE.

List of supported environment parameters:
* *ATLAS_NO_UPDATE_NOTIFIER*: On any command, the CLI will check for new releases using the GitHub API.
  This check will happen at most once every 24 hours. To cancel this behavior, set the environment 
  variable "ATLAS_NO_UPDATE_NOTIFIER".


## atlas schema

Work with atlas schemas.

#### Usage
```
atlas schema
```

#### Details
The `atlas schema` subcommand groups commands for working with Atlas schemas.


### atlas schema apply

Apply an atlas schema to a target database.

#### Usage
```
atlas schema apply [flags]
```

#### Details
`atlas schema apply` plans and executes a database migration to be bring a given database
to the state described in the Atlas schema file. Before running the migration, Atlas will print the migration
plan and prompt the user for approval.

If run with the "--dry-run" flag, atlas will exit after printing out the planned migration.

#### Example

```
atlas schema apply -d "mysql://user:pass@tcp(localhost:3306)/dbname" -f atlas.hcl
atlas schema apply -d "mysql://user:pass@tcp(localhost:3306)/dbname" -f atlas.hcl --dry-run 
atlas schema apply -d "mariadb://user:pass@tcp(localhost:3306)/dbname" -f atlas.hcl
atlas schema apply --dsn "postgres://user:pass@host:port/dbname" -f atlas.hcl
atlas schema apply -d "sqlite://file:ex1.db?_fk=1" -f atlas.hcl
```
#### Flags
```
      --addr string   used with -w, local address to bind the server to (default "127.0.0.1:5800")
      --dry-run       Dry-run. Print SQL plan without prompting for execution
  -d, --dsn string    [driver://username:password@protocol(address)/dbname?param=value] Select data source using the dsn format
  -f, --file string   [/path/to/file] file containing schema
  -w, --web           Open in a local Atlas UI

```


### atlas schema diff

Calculate and print the diff between two schemas.

#### Usage
```
atlas schema diff [flags]
```

#### Details
`atlas schema diff` connects to two given databases, inspects
them, calculates the difference in their schemas, and prints a plan of
SQL queries to bring the "from" database to the schema of the "to" database.

#### Flags
```
      --from string   [driver://username:password@protocol(address)/dbname?param=value] Select data source using the dsn format
      --to string     [driver://username:password@protocol(address)/dbname?param=value] Select data source using the dsn format

```


### atlas schema fmt

Formats Atlas HCL files

#### Usage
```
atlas schema fmt [path]
```

#### Details
`atlas schema fmt` formats all ".hcl" files under the given path using
cannonical HCL layout style as defined by the github.com/hashicorp/hcl/v2/hclwrite package. 
Unless stated otherwise, the fmt command will use the current directory.

After running, the command will print the names of the files it has formatted. If all
files in the directory are formatted, no input will be printed out.



### atlas schema inspect

Inspect an a database's and print its schema in Atlas DDL syntax.

#### Usage
```
atlas schema inspect [flags]
```

#### Details
`atlas schema inspect` connects to the given database and inspects its schema.
It then prints to the screen the schema of that database in Atlas DDL syntax. This output can be 
saved to a file, commonly by redirecting the output to a file named with a ".hcl" suffix:

	atlas schema inspect -d "mysql://user:pass@tcp(localhost:3306)/dbname" > atlas.hcl

This file can then be edited and used with the `atlas schema apply` command to plan
and execute schema migrations against the given database. 
	

#### Example

```

atlas schema inspect -d "mysql://user:pass@tcp(localhost:3306)/dbname"
atlas schema inspect -d "mariadb://user:pass@tcp(localhost:3306)/" --schema=schemaA,schemaB -s schemaC
atlas schema inspect --dsn "postgres://user:pass@host:port/dbname"
atlas schema inspect -d "sqlite://file:ex1.db?_fk=1"
```
#### Flags
```
      --addr string      Used with -w, local address to bind the server to (default "127.0.0.1:5800")
  -d, --dsn string       [driver://username:password@protocol(address)/dbname?param=value] Select data source using the dsn format
  -s, --schema strings   Set schema name
  -w, --web              Open in a local Atlas UI

```


## atlas version

Prints this Atlas CLI version information.

#### Usage
```
atlas version
```


