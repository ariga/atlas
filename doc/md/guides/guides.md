---
title: Guides
id: guides
slug: /guides
---

# Guides

Welcome to the Atlas Guides.

## Deployments

[Intro to deploying schema migrations](deploying/intro.md)

[Creating container images for migrations](deploying/image.md)

[Deploying to AWS ECS (Fargate)](deploying/ecs-fargate.md)

[Deploying schema migrations to Kubernetes with Helm](deploying/helm.md)

[Deploying schema migrations to Kubernetes with Init Containers](deploying/init.md)

[Using Atlas with Fly.io](/guides/deploying/fly-io)

## Databases

### MySQL

[Descending Indexes](mysql/descending-indexes.md)

[Functional Indexes](mysql/functional-indexes.md)

[Prefix Indexes](mysql/prefix-indexes.md)

[CHECK Constraint](mysql/check-constraint.md)

[Generated Columns](mysql/generated-columns.md)

[Managing MySQL schemas on RDS with Terraform](mysql/terraform.md)

[Inline REFERENCES clauses (MY102)](mysql/my-102.md)

[Dropping non-virtual columns (DS103)](mysql/ds-103.md)

### PostgreSQL

[Serial Type Columns](postgres/serial-columns.md)

[Partial Indexes](postgres/partial-indexes.md)

[Included Columns](postgres/included-columns.md)

[Index Operator Classes](postgres/index-operator-classes.md)

[Optimal data alignment (AR101)](postgres/ar-101.md)

[Descending Indexes](postgres/descending-indexes.md)

[Functional Indexes](postgres/functional-indexes.md)

### SQLite

[Partial Indexes](sqlite/partial-indexes.md)

[Descending Indexes](sqlite/descending-indexes.md)

[Functional Indexes](sqlite/functional-indexes.md)

## Migration tools

Automatic migration planning for [`golang-migrate`](migration-tools/golang-migrate.md)

Importing from [`pressly/goose`](migration-tools/goose-import.md)

## ORMs

[GORM](orms/gorm.md)

## Terraform 

[Provisioning schemas (named databases)](terraform/named-databases.mdx)

## Cloud

[How to inspect a local database in the Cloud](cloud/explore-inspection.md)

## Tests

[Integration tests with docker-compose](testing/docker-compose.md)

[Integration tests with GitHub Actions](testing/github-actions.mdx)

