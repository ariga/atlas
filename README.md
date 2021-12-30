
# Atlas -  A Database Toolkit

Atlas is a CLI designed to help companies better work with their data. It includes several components that can
be used individually but are designed to work very well together.

### Supported databases: 
* MySQL
* MariaDB
* PostgresSQL
* SQLiteDB

### Runs on all platforms:
* Mac
* Linux
* Windows


## Quick Installation

Download latest release.
```shell
curl -LO https://release.ariga.io/atlas/atlas-darwin-amd64-v0.1.1
```

Make the atlas binary executable.
```shell
chmod +x ./atlas-darwin-amd64-v0.1.1
```

Move the atlas binary to a file location on your system PATH.
```shell
sudo mv ./atlas-darwin-amd64-v0.1.1 /usr/local/bin/atlas
```
```shell
sudo chown root: /usr/local/bin/atlas
```
Instructions for other platforms and databases: [Getting Started](https://atlasgo.io/).

## Getting Started

### Schema Inspection

Inspect and save output to a schema file.
```shell
atlas schema inspect -d "mysql://root:pass@tcp(localhost:3306)/example" >> atlas.hcl
```

## Apply change to Schema

```shell
atlas schema apply -d "mysql://root:pass@tcp(localhost:3306)/example" -f atlas.hcl
```

Full [CLI documentation](https://atlasgo.io/CLI/atlas). 

### About the Project
Read more about the motivation of the project [Here](https://blog.ariga.io/meet-atlas-cli/).
