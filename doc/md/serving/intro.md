---
id: deployment
slug: /serving/intro
title: Deployment
---

## Introduction

When working as a team, it is ideal to deploy Atlas as a service in your cloud data center, allowing you to enjoy a  centralized 
instance.
In these cases, we recommend using persistent storage.
Atlas encrypts all sensitive information using [AEAD](https://developers.google.com/tink/aead?hl=en) encryption with
*AES128_GCM* Key type. Following recommendations from [tink](https://developers.google.com/tink) library that Atlas uses
(and many other companies as well).

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
atlas serve  --storage "mysql://root:pass@tcp(localhost:3306)/atlas"
```
On your first run, Atlas will generate an encryption key and store it under $HOME/.atlas/keyset.json.
This (private) file should not be deleted, otherwise Atlas will not be able to restore your configuration data. 


### Docker