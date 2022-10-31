---
id: intro
title: Deploying schema migrations
slug: /guides/deploying/intro
---

## Schema changes as Deployments

Changes to database schemas rarely happen in isolation. Most commonly, changes to the database
schema are related to some change in the application code. Because incompatibility between
the database schema and the application can cause serious problems, it is advised to give careful thought
to how these changes are rolled out.

Based on our experience, we have come to the conclusion that changes
to the database schema should be thought of as part of the  _deployment_ sequence, alongside
changing the application version, provisioning infrastructure or applying
configuration changes.

This guide describes some strategies teams can employ to incorporate schema
changes into their deployment sequence.

### Running migrations on server initialization

In many cases, we have seen teams that run schema migration logic as part
of the application code: when servers start, before listening for traffic,
code that ensures that the database schema is up-to-date is invoked.
This is especially common for teams using ORMs that support an "auto-migration"
flow.

In our experience, this strategy may work for simple use-cases, but may
cause issues in larger, more established projects. Some downsides of running
migrations on boot are:
* If multiple replicas of the server code are deployed concurrently
  to avoid dangerous race conditions, some form of synchronization must be
  employed to make sure only one instance tries to run the migration.
* If migrations fail, the server crashes, often entering a crash-loop,
  which may reduce the overall capacity of the system to handle traffic.
* If migrations are driven by a dedicated tool (such as Atlas, Liquibase, Flyway, etc.)
  the tool needs to be packaged into the same deployment artifact. This is both
  cumbersome to invoke and goes against security best practices to reduce attack surface
  by including only the bare minimum into runtime containers.

### Running migrations as part of deployment pipelines

Instead of running migrations on server init, we suggest using a deployment
strategy that follows these principles:

1. Schema migrations are deployed as a discrete step in the deployment pipeline,
   preceding application version changes.
2. If a migration fails, the whole deployment pipeline should halt.
3. Measures should be taken to ensure that only one instance of the migration
   script runs concurrently. 