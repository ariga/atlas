---
id: home
title: Atlas
slug: /
sidebar_position: 1
---
### Welcome to Atlas

Atlas is a set of tools designed to help companies better work with their data. It includes several components that can
be used individually but are designed to work very well together.

Atlas is currently in preview mode, we are still getting set up.

### Data Definition Language (DDL)
The Atlas DDL is an HCL-based, dialect-agnostic, data definition language designed to describe diverse data topologies
in organizations. For example,
```hcl
table "users" {
  schema = "app"

  column "id" {
    type = "uint"
    null = false
    default = 123
  }
  
  column "email" {
    type = "string"
    null = false
    unique = true
  }
  
  primary_key {
    columns = [
      table.users.column.id,
    ]
  }
}
```

In addition, Atlas provides a set of packages and tools built on-top of
them that allow users to inspect an existing database or codebase to
produce Atlas DDL documents.

### Atlas Data Management

It is conceptually similar to  infrastructure-as-code projects (such as
Terraform, etc.), but focused on the special challenges of managing
stateful, always-online resources containing many organizations' most
prized resources asset - their data.

Today, most companies manage their database schemas using the migration
features of their ORM libraries, very close to the application code.
Atlas strives to change this approach and to help organizations push
schema management(provisioning, change management, visibility, etc.)
down to the infrastructure layer.

Atlas Data Management provides a set of Go packages that can
provision resources within databases and safely manage the change
(migration) of database schemas from one version to another.

### Stability

Atlas is under heavy development. We follow SemVer and so the APIs are subject to change until we
reach v1.0.0. We commit to mention any breaking change on the relevant version’s release notes. The “master” branch is
continuously tested and considered safe to use.