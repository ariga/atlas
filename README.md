
# A database toolkit.

[Meet Atlas CLI](https://blog.ariga.io/meet-atlas-cli/).

Supported databases: 
* MySQL
* PostgresSQL

Runs on all platforms:
* Mac
* Linux
* Windows

Instructions for other platforms and databases: [Getting Started](https://atlasgo.io/).

## Installation

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

## Schema Inspection

Inspect and save output to a schema file.
```shell
atlas schema inspect -d "mysql://root:pass@tcp(localhost:3306)/example" >> atlas.hcl
```

## Apply change to Schema

```shell
atlas schema apply -d "mysql://root:pass@tcp(localhost:3306)/example" -f atlas.hcl
```