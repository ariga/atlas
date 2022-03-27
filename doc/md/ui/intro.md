---
id: atlas-ui-intro
slug: /ui/intro
title: Management UI
---

![Management UI](https://blog.ariga.io/uploads/images/posts/atlas-v0.3.0/atlas-post-ui.png)

## Introduction

As many other CLI tools, you can use Atlas to manage your schemas via your terminal.
While this is the common interface for many infrastructure management workflows
we believe that a visual, integrated environment can be beneficial in many use-cases.

### Usage

Get started with the UI by running the command `atlas serve`.
To ensure that your team's schemas, database credentials, migration history and more are not lost, save the data
in the UI by running the command with [persistent storage](https://atlasgo.io/deployment/intro#persistent-storage).

Alternatively, you can run either `schema inspect` or `schema apply` commands with the `-w` command-line
flag:

#### Inspect
```
atlas schema inspect -d "mysql://root:pass@tcp(localhost:3306)/example" -w
```

#### Apply
```
atlas schema apply -d "mysql://root:pass@tcp(localhost:3306)/example" -f atlas.hcl -w
```
