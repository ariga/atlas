---
id: terraform-mysql-rds
title: Manage MySQL on RDS with Terraform and Atlas
slug: /guides/mysql/terraform
image: https://atlasgo.io/uploads/images/tf-mysql.png
tags: [terraform, rds, mysql, aws, iac]
---

### TL;DR

* Many teams use [Terraform](https://terraform.io/), a popular, open-source, infrastructure-as-code (IaC)
  tool created by [HashiCorp](https://www.hashicorp.com/) to provision their cloud infrastructure.
* [RDS](https://aws.amazon.com/rds/) is a fully managed relational database service by AWS.
* The popular [AWS Terraform Provider](https://registry.terraform.io/providers/hashicorp/aws/latest/docs)
  can be used to easily provision database instances and manage them.
* This guide shows how to use the [Atlas Terraform Provider](https://atlasgo.io/integrations/terraform-provider)
  to manage the schema of an RDS-managed MySQL database using Terraform as part of their IaC pipelines.

### Schema management in IaC

[Terraform](https://terraform.io/) is widely used in the industry to provision and manage resources in AWS. 
One of the popular use-cases teams use Terraform for is to provision the database
backing their application on RDS (a fully managed relational database service) using the
AWS Terraform Provider. The provider supports a resource named
[`aws_db_instance`](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/db_instance)
to provision databases using RDS. 

This resource can be used to configure many attributes of the database like the amount of storage, the engine
and version of the database, backup settings, etc. However, when it comes to provisioning
the database's schema itself, the AWS provider does not offer any support. As a result, 
many teams do not manage their database schemas as part of their infrastructure-as-code
pipeline in Terraform.

In this guide, we will demonstrate how to use Terraform to both provision a MySQL database
on AWS with RDS _and_ manage its schema in a single pipeline. 

### Prerequisites

* Install Terraform ([guide](https://learn.hashicorp.com/tutorials/terraform/install-cli))
* An [AWS account](https://aws.amazon.com/free/) (free-tier available) with credentials 
 [configured for Terraform](https://registry.terraform.io/providers/hashicorp/aws/latest/docs#authentication-and-configuration)

### Source code

The full source code for this guide can be found on the [Atlas Examples](https://github.com/ariga/atlas-examples/tree/master/terraform/rds-mysql-example)
repository on GitHub.

### Dev database
To plan a migration from the current to the desired state, Atlas uses a [Dev Database](/concepts/dev-database),
which is usually provided by a locally running container with an empty database of the type
you work with (such as MySQL or PostgreSQL).

To spin up a local MySQL database that will be used as a dev-database in our example, run:

```text
docker run --rm --name atlas-db-dev -d -p 3306:3306 -e MYSQL_DATABASE=example -e MYSQL_ROOT_PASSWORD=pass mysql:8
```

As reference for the next steps, the URL for the Dev Database will be:
```text
mysql://root:pass@localhost:3306/example
```

### Getting started

Create a new directory for our new Terraform project and in it a file named `main.tf`.
In this file we will define all the required resources for our project. Start by configuring
the providers which we will use in our project:

```hcl
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
    atlas = {
      source  = "ariga/atlas"
      version = "0.3.0"
    }
  }
}

# Configure the AWS Provider
provider "aws" {
  region = "us-east-1"
}
```

### Networking
Next, our database must reside in a network topology. We will use the AWS-maintained 
Terraform module `terraform-aws-modules/vpc/aws` for creating a new VPC for our demo.

```hcl
// Fetch the list of availability zones from the current region.
data "aws_availability_zones" "available" {
  state = "available"
}

// Provision a VPC and subnets in these AZs.
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "3.16.1"

  name                 = "atlas-rds-demo"
  cidr                 = "10.0.0.0/16"
  azs                  = data.aws_availability_zones.available.names
  public_subnets       = ["10.0.4.0/24", "10.0.5.0/24", "10.0.6.0/24"]
  enable_dns_hostnames = true
  enable_dns_support   = true
}

// Create a DB subnet to provision the database. 
resource "aws_db_subnet_group" "atlas" {
  name       = "atlas-rds-demo"
  subnet_ids = module.vpc.public_subnets

  tags = {
    Name = "Demo"
  }
}
```
A few things worth noting:
1. In a more realistic scenario we will not provision a dedicated VPC for our database, rather our
   database will reside in a more general-purpose VPC which will include compute and other resources.
2. For the sake of simplicity we are placing our instance in *public* subnets. In a production use case,
   it is usually recommended to provision specific subnets for databases which should not be reachable
   from the internet.

### Security groups

Next, we provision the security group which governs access to our database instance. 
*Notice*: in this demo we are creating a database which is _accessible from the public
internet_. The reason we need this is that we are running Terraform locally and need it 
to be able to connect to the database directly in order to manage our schema. In a realistic
scenario, we will run Terraform from a server which is located in our VPC and can
access our database.

```hcl
// Security group which allows *public access* to our database.
// DO NOT use this in production.
resource "aws_security_group" "rds" {
  name   = "atlas-demo"
  vpc_id = module.vpc.vpc_id

  ingress {
    from_port   = 3306
    to_port     = 3306
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 3306
    to_port     = 3306
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "atlas"
  }
}
```

### Provision the database

Next, we use the `aws_db_instance` to provision a new database instance:

```hcl
// Generate a random password for our db user.
resource "random_password" "password" {
  length  = 16
  special = true
}

// Our RDS-based MySQL 8 instance.
resource "aws_db_instance" "atlas-demo" {
  identifier             = "atlas-demo"
  instance_class         = "db.t3.micro"
  allocated_storage      = 5
  engine                 = "mysql"
  engine_version         = "8.0.28"
  username               = "atlas"
  password               = random_password.password.result
  db_subnet_group_name   = aws_db_subnet_group.atlas.name
  vpc_security_group_ids = [aws_security_group.rds.id]
  parameter_group_name   = "default.mysql8.0"
  publicly_accessible    = true
  skip_final_snapshot    = true
}
```

### Define the desired schema

In a separate file named `schema.hcl` define the desired database schema. To learn more
about defining SQL resources with the Atlas language, [see the docs](https://atlasgo.io/atlas-schema/hcl).

```hcl
// Create a new database named "hello"
schema "hello" {}

// Create a table named "users".
table "users" {
  schema = schema.hello
  column "id" {
    type = int
  }
  column "name" {
    type = varchar(255)
  }
}
```

### Connect everything together

Finally, configure the Atlas Terraform Provider to apply the schema in our `schema.hcl` file
on the RDS-managed database instance. 

```hcl
locals {
  dev_db_url = "mysql://root:pass@localhost:3306/example"
}

// Load the schema from file and normalize it using the dev database.
data "atlas_schema" "hello" {
  dev_db_url = local.dev_db_url
  src        = file("schema.hcl")
}

// Apply the normalized schema to the RDS-managed database.
resource "atlas_schema" "hello" {
  hcl        = data.atlas_schema.hello.hcl
  dev_db_url = local.dev_db_url
  
  // The connection string will be: mysql://user:pass@endpoint/
  url = "mysql://${aws_db_instance.atlas-demo.username}:${urlencode(random_password.password.result)}@${aws_db_instance.atlas-demo.endpoint}/"
}

```

### Time to provision

We're now ready to provision the resources in our AWS account. Start by initializing
the project:

```shell
terraform init
```
Observe that Terraform downloads the necessary modules and plugins:

```text
Initializing modules...
Downloading registry.terraform.io/terraform-aws-modules/vpc/aws 3.16.1 for vpc...
- vpc in .terraform/modules/vpc

Initializing the backend...

Initializing provider plugins...
- Finding ariga/atlas versions matching "0.3.0-pre.1"...
- Finding hashicorp/aws versions matching ">= 3.73.0, ~> 4.0"...
- Finding latest version of hashicorp/random...
- Installing ariga/atlas v0.3.0-pre.1...
- Installed ariga/atlas v0.3.0-pre.1 (self-signed, key ID 45441FCEAAC3770C)
- Installing hashicorp/aws v4.35.0...
- Installed hashicorp/aws v4.35.0 (signed by HashiCorp)
- Installing hashicorp/random v3.4.3...
- Installed hashicorp/random v3.4.3 (signed by HashiCorp)

Partner and community providers are signed by their developers.
If you'd like to know more about provider signing, you can read about it here:
https://www.terraform.io/docs/cli/plugins/signing.html

Terraform has created a lock file .terraform.lock.hcl to record the provider
selections it made above. Include this file in your version control repository
so that Terraform can guarantee to make the same selections by default when
you run "terraform init" in the future.

Terraform has been successfully initialized!
```

Finally, let's provision our infrastructure. Run:

```text
terraform apply
```

Terraform offers a plan similar to:

```text
Terraform used the selected providers to generate the following execution plan. Resource actions are indicated with the following symbols:
  + create

Terraform will perform the following actions:

  // highlight-start
  # atlas_schema.hello will be created
  + resource "atlas_schema" "hello" {
      + dev_db_url = (sensitive value)
      + hcl        = <<-EOT
            table "users" {
              schema = schema.hello
              column "id" {
                null = false
                type = int
              }
              column "name" {
                null = false
                type = varchar(255)
              }
            }
            schema "hello" {
              charset = "latin1"
              collate = "latin1_swedish_ci"
            }
        EOT
      + id         = (known after apply)
      + url        = (sensitive value)
    }
    // highlight-end
    
    // ... A long list of more changes

Plan: 15 to add, 0 to change, 0 to destroy.

Do you want to perform these actions?
  Terraform will perform the actions described above.
  Only 'yes' will be accepted to approve.

  Enter a value:
```
If the plan looks right to you, write "yes" and hit Enter.

After a few minutes Terraform should finish provisioning everything and print:

```text
Apply complete! Resources: 15 added, 0 changed, 0 destroyed.
```

### Evolving the schema

The Atlas provider is useful also in cases where your schema evolves, and
you wish to change it. Let's see how we can add a new, unique, column to our `users`
table.

Edit `schema.hcl`:

```hcl
schema "hello" {}

table "users" {
  schema = schema.hello
  column "id" {
    type = int
  }
  column "name" {
    type = varchar(255)
  }
  // highlight-start
  column "email" {
    type = varchar(255)
  }
  index "idx_email" {
    columns = [
      column.email
    ]
    unique = true
  }
  // highlight-end
}
```
Now, re-apply with Terraform:

```text
terraform apply
```
Terraform understands the diff in the desired schema for our database and produces
a plan accordingly:

```text
Terraform will perform the following actions:

  # atlas_schema.hello will be updated in-place
  ~ resource "atlas_schema" "hello" {
      ~ hcl        = <<-EOT
            table "users" {
              schema = schema.hello
              column "id" {
                null = false
                type = int
              }
              column "name" {
                null = false
                type = varchar(255)
              }
          // highlight-start              
          +   column "email" {
          +     null = false
          +     type = varchar(255)
          +   }
          +   index "idx_email" {
          +     unique  = true
          +     columns = [column.email]
          +   }
          // highlight-end
            }
            schema "hello" {
              charset = "latin1"
              collate = "latin1_swedish_ci"
            }
        EOT
        id         = "mysql://atlas:7%28%3Ej%24rQ%3Ez%3C%3DPrqU%5D@atlas-demo.cripwukl9y7k.us-east-1.rds.amazonaws.com:3306/"
        # (2 unchanged attributes hidden)
    }

Plan: 0 to add, 1 to change, 0 to destroy.

Do you want to perform these actions?
  Terraform will perform the actions described above.
  Only 'yes' will be accepted to approve.

  Enter a value:
```
After reviewing the plan, write "yes" at the prompt and hit Enter:
```text
Apply complete! Resources: 0 added, 1 changed, 0 destroyed.
```

### Wrapping up

In this guide, we demonstrated how to provision and manage a MySQL instance using Terraform
and the AWS and Atlas providers. Following the example here, teams can finally integrate schema
management with their full Infrastructure-as-Code workflows.

## Need More Help?

[Join the Ariga Discord Server](https://discord.gg/zZ6sWVg6NT) for early access to features and the ability to provide
exclusive feedback that improves your Database Management Tooling.
