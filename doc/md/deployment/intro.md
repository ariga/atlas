---
id: deployment
slug: /deployment/intro
title: Deployment
---

## Introduction

:::info
As our team is working on completely rewriting The Management UI, support for it has been temporarily removed in the most recent version of Atlas. To explore the UI in previous versions, download [v0.3.7](https://github.com/ariga/atlas/releases/tag/v0.3.7) or earlier.
:::

Teams using Atlas to manage their databases can benefit from deploying the Atlas Management UI to their
cloud environment. The Management UI can be used to gain visibility of your team's database schemas as well
as provide an intuitive interface for planning and running database migrations using Atlas's core engine.
In addition, the Management UI automatically tracks and audits all changes using the "Activity & History" feature.

The Atlas Management UI is designed to be a long-running process.
It is therefore recommended running it against a persistent database that will make sure your team's schemas,
database credentials, migration history and more are not lost in case you need to restart the service.
Atlas uses [tink](https://developers.google.com/tink), a battle-tested encryption library created at Google, to encrypt all sensitive information.
Following recommendations from the developers of tink, Atlas uses [AEAD](https://developers.google.com/tink/aead?hl=en)
encryption with an AES256_GCM type key. 

In this section we will review two deployment and usage options:
1. Deploying on a VM/EC2 instance with persistent storage.
2. Deploying using an official Atlas Docker container.

### Persistent Storage

When serving Atlas on a VM/EC2, you will need to decide on which persistent storage you would like Atlas to store the
configuration data.
For now, you can choose any of the supported databases:
1. SQLite (file) (in-memory is also supported but does not require encryption)
2. MySQL
3. TiDB
4. MariaDB
5. PostgresDB

An example for serving Atlas with MySQL persistent storage:
```
atlas serve --storage "mysql://root:pass@localhost:3306/atlas"
```
On your first run, Atlas will generate an encryption key and store it under ```$HOME/.atlas/keyset.json```.
This (private) file should not be deleted, otherwise Atlas will not be able to restore your configuration data. 

### Docker

Atlas has an official docker that is updated with the latest and tagged versions, 
which can be found on [Docker Hub](https://hub.docker.com/r/arigaio/atlas).
You can use this image to run the CLI and to serve the Atlas UI.

To run Atlas in served and persisted mode inside a container:
```
docker run -v $HOME/.atlas/keyset.json:/root/.atlas/keyset.json -p 5800:5800 arigaio/atlas:latest serve --addr :5800 --storage "mysql://root:pass@tcp(host.docker.internal:3306)/atlas"
```

Let's review this command step by step:
1. ```docker run``` - running a command.
2. ```-v $HOME/.atlas/keyset.json:/root/.atlas/keyset.json``` binds a secret key from a persisted host into the docker container. If you don't have a key, use an empty file and Atlas will generate one for you.
3. ``` -p 5800:5800``` binds an exposed serving port for connecting with a web client.
4. ```5800:5800 arigaio/atlas:latest``` use the latest atlas image, for a versioned release use a specific tag such as ```arigaio/atlas:0.3.6```.
5. ```serve --addr :5800 --storage "mysql://root:pass@tcp(host.docker.internal:3306)/atlas"``` serve Atlas on port 5800 with a persistent MySQL Database.  

### Encryption

On its first run, Atlas generates a keyset.json file under `$HOME/.atlas/` containing 
an encryption keyset for you. Do not lose this file! Without this file you cannot later use
any database credentials that save to Atlas. 

If you want to generate this key yourself, you can
[install Tinkey](https://developers.google.com/tink/install-tinkey), Tink's official CLI,
and use it to generate a keyset:

```shell
brew tap google/tink https://github.com/google/tink
brew install tinkey
tinkey create-keyset --key-template AES256_GCM --out ~/.atlas/keyset.json
```

Alternatively, you can use [rotemtam/tinkey](https://hub.docker.com/r/rotemtam/tinkey), an unofficial Docker image that
wraps the official binary distribution:

```shell
docker run --rm rotemtam/tinkey create-keyset --key-template AES256_GCM > ~/.atlas/keyset.json
```
