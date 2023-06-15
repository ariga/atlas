---
id: getting-started
title: Getting Started with Atlas Cloud
sidebar_label: Getting Started
---
import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';
import Discord from '../../website/src/assets/icons/discord-white.svg'

[Atlas Cloud](https://atlasgo.cloud) is an online platform that allows developers to keep track of database
schema changes throughout the entire developer lifecycle.

With Atlas Cloud, you can:
- View entire database schemas visually and textually (in SQL and HCL)
- Get full CI reports about schema changes and their severity
- Track deployments running in every environment

...and much more!

### Signing Up
1. To get started with [Atlas Cloud](https://atlasgo.cloud/), sign up [here](https://auth.atlasgo.cloud/signup/).
2. Once verifying your account, you will be prompted to create an **organization**. Choose a meaningful name for
the organization, as it will also be your subdomain.
For example, "Acme Corp" will be available at "acmecorp.atlasgo.cloud".

:::info Creating Organizations
In our community plan, users are limited to creating **two** organizations. If you are interested in administering more
organizations, check out our [plans](https://atlasgo.cloud/pricing).
:::


### Migration Directories

Atlas Cloud allows developers to easily work with the [versioned migrations](/versioned/diff) workflow. Each change, or migration,
that is made to the database schema is checked into source control and reviewed during code review. Every change is saved
in a **migration directory**, which is a folder that contains all of the migration files, as well as a
[sum file](/versioned/new#recalculating-the-directory-hash) to ensure the validity of these changes.

To start, you must provide Atlas Cloud with a GitHub repository that has a migration directory. This directory can be in
one of four formats of the schema migration tools that are supported:
 - Atlas
 - [Liquibase](https://www.liquibase.org/)
 - [Flyway](https://flywaydb.org/)
 - [golang-migrate](https://github.com/golang-migrate/migrate)

If you do not have a project with an existing migration directory, you can get started by using an example
[Atlas template repository](https://github.com/ariga/atlas-template/generate).

:::info SQL Directories
Atlas Cloud only supports migration files that are written in SQL. If you are using Liquibase or Flyway,
your changesets/migration files must be written in SQL.
:::
