---
title: Announcing Atlas CLI v0.4.2 with preview support for CockroachDB 🪳
authors: Hedwigz
tags: [cockroachdb, integration, announcement]
image: https://blog.ariga.io/uploads/images/posts/cockroachdb/cockroachdb.png
---

Today, I'm happy to announce the release of [v0.4.2](https://github.com/ariga/atlas/releases/tag/v0.4.2) of the Atlas CLI. 
This version includes many improvements and fixes, but I wanted to share with you  exciting news about something I
personally worked on. As of v0.4.2, Atlas includes preview support for CockroachDB 🎉 

## Atlas
[Atlas](https://atlasgo.io) is an open-source project that helps developers better manage their database
schemas. It has a [CLI tool](https://atlasgo.io/cli/reference) and a
[Terraform integration](https://atlasgo.io/blog/2022/05/04/announcing-terraform-provider). By using Atlas's
Data Definition Language (with a syntax similar to Terraform), users can plan, verify and apply changes
to their databases in a simple, declarative workflow.
Earlier this year, Atlas became the [migration engine for Ent](https://entgo.io/blog/2022/01/20/announcing-new-migration-engine),
a widely popular, Linux Foundation backed entity framework for Go.

## CockroachDB
[CockroachDB](https://www.cockroachlabs.com/) is an [open-source](https://github.com/cockroachdb/cockroach) NewSQL 
database. From their README:
> CockroachDB is a distributed SQL database built on a transactional and strongly-consistent 
> key-value store. It scales horizontally; survives disk, machine, rack, and even datacenter 
> failures with minimal latency disruption and no manual intervention; supports strongly-consistent 
> ACID transactions; and provides a familiar SQL API for structuring, manipulating, and querying data.  
  
CockroachDB has been gaining popularity and many of you [have](https://github.com/ent/ent/issues/2545)
[been](https://github.com/ariga/atlas/issues/785#issue-1231951038) [asking](https://github.com/ariga/atlas/issues/785#issuecomment-1125853135) 
for Atlas to support it.

While CockroachDB aims to be PostgreSQL compatible, it still has some incompatibilities 
(e.g. [1](https://github.com/cockroachdb/cockroach/issues/20296#issuecomment-1066140651), 
[2](https://github.com/cockroachdb/cockroach/issues/82064),[3](https://github.com/cockroachdb/cockroach/issues/81659))
which prevented Atlas users from using the existing Postgres dialect from working with it.  
  
With the latest release of Atlas, the Postgres driver automatically detects when it is connected to a CockroachDB
database and uses a custom driver which provides compatability with CockroachDB.

### Getting started with Atlas and CockroachDB

Let's see how we can use Atlas CLI to manage the schema of a CockroachDB database. 
Start by downloading the latest version of Atlas, on macOS:
```
brew install ariga/tap/atlas
```
For installation instructions on other platforms, see [the docs](https://atlasgo.io/cli/getting-started/setting-up#install-the-cli).

For the purpose of this example, let's spin up a local, [single-node CockroachDB cluster](https://github.com/cockroachlabs-field/cockroachdb-single-node)
in a container by running:
```
docker run -it -p 8080:8080 -p 26257:26257 -e "DATABASE_NAME=test" -e "MEMORY_SIZE=.5" timveil/cockroachdb-single-node:latest
```

Next, use Atlas's `schema inspect` command to read the schema of our local database and save the result to a file:
```
atlas schema inspect -u 'postgres://root:pass@localhost:26257/?sslmode=disable' --schema public > schema.hcl
```
Observe the current HCL representation of the `public` schema, empty with no tables:
```hcl
schema "public" {
}
```

Next, edit `schema.hcl` to add a table: 

```hcl title="schema.hcl"
schema "public" {
}

table "test" {
    column "id" {
        type = int
    }
    primary_key {
        columns = [column.id]
    }
}
```
Now apply the schema using the `schema apply` command:
```
atlas schema apply -u 'postgres://root:pass@localhost:26257/?sslmode=disable' --schema public -f schema.hcl
```
Atlas prints out the planned changes and asks for your confirmation: 
```
-- Planned Changes:
-- Create "test" table
CREATE TABLE "public"."test" ("id" integer NOT NULL, PRIMARY KEY ("id"))
Use the arrow keys to navigate: ↓ ↑ → ←
? Are you sure?:
  ▸ Apply
    Abort
```
After hitting "Apply", Atlas connects applies our desired schema to the database:
```
✔ Apply
```

We have successfully applied our schema to our database. 

### Learn more about Atlas

In this short example, we demonstrated two of Atlas's basic features: database inspection
and declarative schema migration (applying a desired schema on a database). Here are some topics
you might want to explore when getting started with Atlas:
* [Learn the DDL](/ddl/sql) - learn how to define any SQL resource in Atlas's data definition
  language.
* [Try the Terraform Provider](https://atlasgo.io/blog/2022/05/04/announcing-terraform-provider) - see how you can use 
  the Atlas Terraform Provider to integrate schema management in your general Infrastructure-as-Code workflows.
* [Use the `migrate` command to author migrations](/cli/reference#atlas-migrate) - In addition to the Terraform-like
 declarative workflow, Atlas can manage a migration script directory for you based on your desired schema.

# Preview support
The integration of Atlas with CockroachDB is well tested with version `v21.2.11` (at the time of writing, 
`latest`) and will be extended in the future. If you're using other versions of CockroachDB or looking 
for help, don't hesitate to [file an issue](https://github.com/ariga/atlas/issues) or join our
[Discord channel](https://discord.gg/zZ6sWVg6NT).

Have questions? Feedback? Find our team [on our Discord server](https://discord.gg/zZ6sWVg6NT).
