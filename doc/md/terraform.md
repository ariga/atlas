---
title: Terraform Provider
id: terraform-provider
slug: /terraform-provider
---
## Introduction

The official [Atlas Terraform provider](https://registry.terraform.io/providers/ariga/atlas/latest) allows you to use atlas with Terraform to provide full state migrations for your databases. Read about the release announcement [here](https://atlasgo.io/blog/2022-05-03-announcing-terraform-provider).
* [Documentation](https://registry.terraform.io/providers/ariga/atlas/latest/docs)
* [GitHub Repository](https://github.com/ariga/terraform-provider-atlas)

## Install
Add Atlas to your [required providers](https://www.terraform.io/language/providers/requirements#requiring-providers):
```hcl
terraform {
  required_providers {
    atlas = {
      source = "ariga/atlas"
      version = "~> 0.1.0"
    }
  }
}
```

## Basic example
Currently, Atlas provider uses an [HCL file](http://localhost:3000/ddl/intro#hcl) describing the state of the database and performs migrations according to the state difference between the HCL file and the target database.
### Generate schema
You can either:
1. [Inspect](cli/getting-started/inspection#inspecting-our-database) an existing database
2. Start with a basic schema
```hcl
schema "example" {
  charset = "utf8mb4"
  collate = "utf8mb4_0900_ai_ci"
}
```
### Configure Terraform
Use the following configuration to apply the HCL file `schema.hcl` onto a target MySQL database (but you can specify any of the [supported databases](https://github.com/ariga/atlas#supported-databases)):
```hcl title="main.tf"
provider "atlas" {}

# normalize the schema.
data "atlas_schema" "market" {
  dev_db_url = "mysql://root:pass@localhost:3307/market"
  src = file("${path.module}/schema.hcl")
}

# sync the state of the target database with the hcl file.
resource "atlas_schema" "market" {
  hcl = data.atlas_schema.market.hcl
  url = "mysql://root:pass@localhost:3306/market"  
  dev_db_url = "mysql://root:pass@localhost:3307/market"
}
```

For the full documentation and examples of the provider visit the [registry page](https://registry.terraform.io/providers/ariga/atlas/latest/docs).