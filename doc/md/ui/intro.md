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

To launch the Atlas UI, run either `schema inspect` or `schema apply` commands with the `-w` command-line
flag:

#### Inspect
```
atlas schema inspect -d "mysql://root:pass@tcp(localhost:3306)/example" -w
```

#### Apply
```
atlas schema apply -d "mysql://root:pass@tcp(localhost:3306)/example" -f atlas.hcl -w
```

## Features

### 