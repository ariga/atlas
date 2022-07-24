
# Atlas -  A Database Toolkit
[![Discord](https://img.shields.io/discord/930720389120794674?label=discord&logo=discord&style=flat-square&logoColor=white)](https://discord.gg/zZ6sWVg6NT)

<img width="50%" align="right" style="display: block; margin:40px auto;"
     src="https://atlasgo.io/uploads/images/gopher.png"/>

Atlas CLI is an open source tool that helps developers manage their database schemas by applying modern DevOps principles. Contrary to existing tools, Atlas intelligently plans schema migrations for you. Atlas users can use the [Atlas DDL](https://atlasgo.io/concepts/ddl#hcl) (data definition language) to describe their desired database schema and use the command-line tool to plan and apply the migrations to their systems.

### Supported databases:
* MySQL
* MariaDB
* PostgresSQL
* SQLite
* TiDB

## Quick Installation

On macOS:

```shell
brew install ariga/tap/atlas
```

Click [here](https://atlasgo.io/cli/getting-started/setting-up) to read instructions for other platforms.

## Getting Started
Get started with Atlas by following the [Getting Started](https://atlasgo.io/cli/getting-started/setting-up) docs.
This tutorial teaches you how to inspect a database, generate a migration plan and apply the migration to your database.

## Features
- **Inspecting a database**: easily inspect your database schema by providing a database URL.
```shell
atlas schema inspect -u "mysql://root:pass@localhost:3306/example" > atlas.hcl
```
- **Applying a migration**: generate a migration plan to apply on the database by providing an HCL file with the desired Atlas schema.
```shell
atlas schema apply -u "mysql://root:pass@localhost:3306/example" -f atlas.hcl
```
- **Declarative Migrations vs. Versioned Migrations**: Atlas offers two workflows. Declarative migrations allow the user to provide a desired state and Atlas gets the schema there instantly (simply using inspect and apply commands). Alternatively, versioned migrations are explicitly defined and assigned a version. Atlas can then bring a schema to the desired version by following the migrations between the current version and the specified one.

### About the Project
Read more about the motivation of the project [here](https://atlasgo.io/blog/2021/11/25/meet-atlas).
