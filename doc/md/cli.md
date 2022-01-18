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
[Getting Started](/cli/getting-started/setting-up) page.


## atlas env

Print atlas env params


Env prints atlas environment information.
Every set environment param will print in the form of NAME=VALUE.

List of supported environment parameters:
"ATLAS_NO_UPDATE_NOTIFIER": On any command, the CLI will check for updates with the GitHub public API once every 24 hours.
To cancel this behavior, set the environment parameter "ATLAS_NO_UPDATE_NOTIFIER".



#### Usage
```
atlas env
```







---


## atlas schema

Work with atlas schemas


Interact with the schema and data source



#### Usage
```
atlas schema
```







---


### atlas schema apply

Apply an atlas schema to a data source




#### Usage
```
atlas schema apply [flags]
```



#### Example
```

atlas schema apply -d "mysql://user:pass@tcp(localhost:3306)/dbname" -f atlas.hcl
atlas schema apply -d "mariadb://user:pass@tcp(localhost:3306)/dbname" -f atlas.hcl
atlas schema apply --dsn "postgres://user:pass@host:port/dbname" -f atlas.hcl
atlas schema apply -d "sqlite://file:ex1.db?_fk=1" -f atlas.hcl
```




#### Flags
```
      --addr string   used with -w, local address to bind the server to (default "127.0.0.1:5800")
  -d, --dsn string    [driver://username:password@protocol(address)/dbname?param=value] Select data source using the dsn format
  -f, --file string   [/path/to/file] file containing schema
  -w, --web           Open in a local Atlas UI

```



### atlas schema inspect

Inspect an atlas schema




#### Usage
```
atlas schema inspect [flags]
```



#### Example
```

atlas schema inspect -d "mysql://user:pass@tcp(localhost:3306)/dbname"
atlas schema inspect -d "mariadb://user:pass@tcp(localhost:3306)/dbname"
atlas schema inspect --dsn "postgres://user:pass@host:port/dbname"
atlas schema inspect -d "sqlite://file:ex1.db?_fk=1"
```




#### Flags
```
      --addr string   used with -w, local address to bind the server to (default "127.0.0.1:5800")
  -d, --dsn string    [driver://username:password@protocol(address)/dbname?param=value] Select data source using the dsn format
  -w, --web           Open in a local Atlas UI

```



## atlas version

Show atlas CLI version




#### Usage
```
atlas version
```







---


