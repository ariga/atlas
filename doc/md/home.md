---
id: home
title: Atlas
slug: /
sidebar_position: 1
---
### Welcome to the Future of Database Migrations
Atlas is a set of tools designed to help companies better manage changes to databases.
It is conceptually similar to infrastructure-as-code projects (such as Terraform, etc.),
but focused on the special challenges of managing stateful, always-online resources
containing many organizations' most prized resources asset - their data.

Today, most companies manage their database schemas using the migration features of
their ORM libraries, very close to the application code. Atlas strives to change this
approach and to help organizations push schema management (provisioning, change management,
visibility, etc.) down to the infrastructure layer.

### The Atlas Project includes:
* The Atlas DDL - an HCL-based, dialect-agnostic, data definition language designed to
  describe all entity types and their relations across an organization.
* A set of Go packages that can:
    * Inspect an existing database or codebase to produce Atlas DDL documents.
    * Provision resources within databases.
    * Safely manage the change (migration) of database schemas from one version to another.
* A set of tools (built on top of these packages) that integrate with your existing
  Infrastructure-as-Code processes to help you apply DevOps philosophy to data management.

### Stability
Atlas is under heavy development.  We follow SemVer and so the APIs are subject to change until we reach v1.0.0.  We commit to mention any breaking change on the relevant version’s release notes. The “master” branch is continuously tested and considered safe to use.
