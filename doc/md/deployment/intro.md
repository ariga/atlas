---
id: deployment
slug: /deployment/intro
title: Deployment
---

## Introduction

Teams using Atlas to manage their databases can benefit from deploying the Atlas Management UI to their
cloud environment. The Management UI can be used to gain visibility of your team's database schemas as well
as provide an intuitive interface for planning and running database migrations using Atlas's core engine.
In addition, the Management UI automatically tracks and audits all changes using the "Activity & History" feature.

The Atlas Management UI is designed to be a long-running process.
It is therefore recommended to run it against a persistent database that will make sure your team's schemas,
database credentials, migration history and more are not lost in case you need to restart the service.
Atlas uses [tink](https://developers.google.com/tink), a battle-tested encryption library created at Google, to encrypt all sensitive information.
Following recommendations from the developers of tink, Atlas uses [AEAD](https://developers.google.com/tink/aead?hl=en)
encryption with an AES128_GCM type key.

In this section we will review two deployment and usage options:
1. Deploying on a VM/EC2 instance with persistent storage.
2. Deploying using an official Atlas Docker container.

### Persistent Storage

When serving Atlas on a VM/EC2, you will need to decide on which persistent storage you would like Atlas to store the
configuration data.
For now, you can choose any of the supported databases:
1. SQLite (file) (in-memory is also supported but does not require encryption)
2. MySQL
3. MariaDB
4. PostgresDB

An example for serving Atlas with MySQL persistent storage:
```
atlas serve --storage "mysql://root:pass@tcp(localhost:3306)/atlas"
```
On your first run, Atlas will generate an encryption key and store it under ```$HOME/.atlas/keyset.json```.
This (private) file should not be deleted, otherwise Atlas will not be able to restore your configuration data. 


### Docker

Atlas has an official docker that is updated with the latest and tagged versions, 
which can be found on [Docker Hub](https://hub.docker.com/r/arigaio/atlas).
You can use this image to run the CLI and to serve the Atlas UI.

To run Atlas in served and persisted mode inside a container:
```
docker run -v $HOME/.atlas/keyset.json:/root/.atlas/keyset.json -p 8080:8080 arigaio/atlas:latest serve --addr 0.0.0.0:8080 --storage "mysql://root:pass@tcp(host.docker.internal:3306)/atlas"
```

Let's review this command step by step:
1. ```docker run``` - running a command.
2. ```-v $HOME/.atlas/keyset.json:/root/.atlas/keyset.json``` binds a secret key from a persisted host into the docker container. If you don't have a key, use an empty file and Atlas will generate one for you.
3. ``` -p 8080:8080``` binds an exposed serving port for connecting with a web client.
4. ```8080:8080 arigaio/atlas:latest``` use the latest atlas image, for a versioned release use a specific tag such as ```arigaio/atlas:0.3.6```.
5. ```serve --addr 0.0.0.0:8080 --storage "mysql://root:pass@tcp(host.docker.internal:3306)/atlas"``` serve Atlas on port 8080 with a persistent MySQL Database.  


