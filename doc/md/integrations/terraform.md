---
title: Terraform Provider
id: terraform-provider
slug: /integrations/terraform-provider
---

import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

## Introduction

The official [Atlas Terraform provider](https://registry.terraform.io/providers/ariga/atlas/latest)
allows you to use Atlas with Terraform to manage your database schemas as part of you Infrastructure-as-Code (Iac)
workflow . Read about the release announcement [here](https://atlasgo.io/blog/2022/05/04/announcing-terraform-provider).
* [Documentation](https://registry.terraform.io/providers/ariga/atlas/latest/docs)
* [GitHub Repository](https://github.com/ariga/terraform-provider-atlas)

## Installation
Add Atlas to your [required providers](https://www.terraform.io/language/providers/requirements#requiring-providers):
```hcl
terraform {
  required_providers {
    atlas = {
      source = "ariga/atlas"
      version = "~> 0.4.0"
    }
  }
}
```

## Declarative Migrations

In the declarative workflow, the Atlas Terraform provider uses an [HCL file](/atlas-schema/sql.mdx) to describe the
desired state of the database, and performs migrations according to the state difference
between the HCL file and the target database.

To use the Terraform provider, you will need such a file. If you are working against a fresh,
empty database, start by creating a file named `schema.hcl` that only contains a single [`schema`](/atlas-schema/sql.mdx#schema)
resource. If your database contains a schema (named database) is named `example`, use something like:

```hcl
schema "example" {
  // Basic charset and collation for MySQL.
  charset = "utf8mb4"
  collate = "utf8mb4_0900_ai_ci"
}
```

For instructions on using a database with an existing schema, [see below](#working-with-an-existing-database)

### Configure Terraform

Use the following configuration to apply the HCL file `schema.hcl` onto a target MySQL
database (but you can specify any of the [supported databases](https://github.com/ariga/atlas#supported-databases)):

```hcl title="main.tf"
provider "atlas" {}

// Load (and normalize) the desired schema from an HCL file.
data "atlas_schema" "market" {
  dev_db_url = "mysql://root:pass@localhost:3307/market"
  src = file("${path.module}/schema.hcl")
}

// Sync the state of the target database with the hcl file.
resource "atlas_schema" "market" {
  hcl = data.atlas_schema.market.hcl
  url = "mysql://root:pass@localhost:3306/market"
  dev_db_url = "mysql://root:pass@localhost:3307/market"
}
```

For the full documentation and examples of the provider visit the [registry page](https://registry.terraform.io/providers/ariga/atlas/latest/docs).

#### Working with an existing database

When you first run the Atlas Terraform Provider on a database, the database's state isn't yet present
in Terraform's representation of the world (described in the [Terraform State](https://www.terraform.io/language/state)).
To prevent a situation where Terraform accidentally creates a plan that includes the deletion of resources (such as tables or
schemas) that exist in your database on this initial run, make sure that the HCL file that you pass to the `atlas_schema`
resources is up-to-date.

Luckily, you do not need to write this file by hand. The Atlas CLI's [`schema inspect` command](https://atlasgo.io/cli-reference#atlas-schema-inspect)
can do this for you. To inspect an existing database and write its HCL representation to a file simply run:
```
atlas schema inspect -u <database url> > <target file>
```
Replacing `<database url>` with the [URL](/concepts/url) for your database, and `<target file>`
with the name of the file you want to write the output to. For example:
```
atlas schema inspect -u mysql://user:pass@localhost:3306 > schema.hcl
```

## Versioned Migrations

In the versioned workflow, the Atlas Terraform provider uses the migrations directory to manage changes to the
database across versions. To use this workflow, we first need to [create a migrations directory](/versioned/new.mdx) using the Atlas CLI.

Here is the example of a migration directory that uses the versioned workflow:

<Tabs
defaultValue="migration_file2"
values={[
{label: '20220811074144_create_users.sql', value: 'migration_file1'},
{label: '20220811074314_add_users_name.sql', value: 'migration_file2'},
{label: 'atlas.sum', value: 'sum_file'},
]}>
<TabItem value="migration_file1">

```sql
-- create "users" table
CREATE TABLE `users` (`id` int NOT NULL) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
```

</TabItem>
<TabItem value="migration_file2">

```sql
-- modify "users" table
ALTER TABLE `users` ADD COLUMN `name` varchar(255) NOT NULL;
```

</TabItem>
<TabItem value="sum_file">

```text
h1:w2ODzVxhTKdBVBdzqntHw7rHV8lKQF98TmNevOEZfIo=
20220811074144_create_users_table.sql h1:KnMSZM/E4TBGidYCZ+UHxkHEWaRWeyuPIUjSHRybQqA=
20220811074314_add_users_name.sql h1:jUpaANgD0SjI5DjaHuJxtHZ6Wq98act0MmE5oZ+NRU0=
```

</TabItem>
</Tabs>

### Configure Terraform

Use the following configuration to apply the migration directory onto a target MySQL
database (but you can specify any of the [supported databases](https://github.com/ariga/atlas#supported-databases)):

```hcl title="main.tf"
provider "atlas" {}

// Inspect the target database and load its state.
// This is used to determine which migrations to run.
data "atlas_migration" "shop" {
  dir = "migrations?format=atlas"
  url = "mysql://root:pass@localhost:3306/shop"
}

// Sync the state of the target database with the migrations directory.
resource "atlas_migration" "shop" {
  dir     = "migrations?format=atlas"
  version = data.atlas_migration.shop.latest # Use latest to run all migrations
  url     = data.atlas_migration.shop.url
  dev_url = "mysql://root:pass@localhost:3307/shop"
}
```

For the full documentation and examples for using the provider, visit the [registry page](https://registry.terraform.io/providers/ariga/atlas/latest/docs).

:::info

Note, the dev_url is used to run the Lint on the migrations directory. See [Migration Lint](/versioned/lint.mdx) for more information.

:::