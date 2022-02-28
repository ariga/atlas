
# Atlas -  A Database Toolkit

<img width="50%" align="right" style="display: block; margin:40px auto;"
     src="https://atlasgo.io/uploads/images/gopher.png"/>

Atlas is a CLI designed to help companies better work with their data. It includes several components that can
be used individually but are designed to work very well together.

### Supported databases: 
* MySQL
* MariaDB
* PostgresSQL
* SQLite

### Runs on all platforms:
* Mac
* Linux
* Windows


## Quick Installation

On macOS:

```shell
brew install ariga/tap/atlas
```

Instructions for other platforms and databases: [Getting Started](https://atlasgo.io/).

## Getting Started

### Schema Inspection

Inspect and save output to a schema file.
```shell
atlas schema inspect -u "mysql://root:pass@tcp(localhost:3306)/example" > atlas.hcl
```

## Apply change to Schema

```shell
atlas schema apply -u "mysql://root:pass@tcp(localhost:3306)/example" -f atlas.hcl
```

Full [CLI documentation](https://atlasgo.io/cli/reference). 

### About the Project
Read more about the motivation of the project [Here](https://blog.ariga.io/meet-atlas-cli/).
