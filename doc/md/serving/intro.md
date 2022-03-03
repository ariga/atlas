---
id: atlas-serving
slug: /serving/intro
title: Serving
---

## Introduction

Usually as a team using Atlas, you would like to deploy it as a service in your cloud data center and enjoy a centralized 
instance.
When using Atlas in such cases we recommend using persistent storage.
Atlas encrypts all sensitive information using [AEAD](https://developers.google.com/tink/aead?hl=en) encryption with
*AES128_GCM* Key type. Following recommendations from [tink](https://developers.google.com/tink) library that Atlas uses
(and many other companies as well).

In this section we will review two deployment and usage options:
1. Deploying on a VM/EC2 instance with persistent storage.
2. Deploying using an official Atlas Docker container.

### Persistent Storage

When serving Atlas on a VM/EC2 you'll need to decide which persistent storage you would like Atlas to store all the
configuration data at.
For now, you can choose any or the supported Databases such as:
1. SQLite (file) (in-memory is also supported but does not require encryption)
2. MySQL
3. MariaDB
4. PostgresDB

An example for serving Atlas with MySQL persistent storage:
```
atlas serve  --storage "mysql://root:pass@tcp(localhost:3306)/atlas"
```
On your first run, Atlas will generate an encryption key and store it under $HOME/.atlas/keyset.json.
This (private) file should not be deleted, otherwise Atlas will not be able to restore your configuration data. 


### Docker