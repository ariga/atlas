---
title: "Terraform for databases: Announcing the official Terraform provider for Atlas"
authors: Hedwigz
tags: [terraform, announcement]
image: https://blog.ariga.io/uploads/images/posts/terraform-provider/terraform-provider.png
---

Today we are glad to announce the release of the official [Atlas Terraform Provider](https://registry.terraform.io/providers/ariga/atlas).

### What is Terraform
[Terraform](https://terraform.io/) is a popular open-source tool created by [HashiCorp](https://www.hashicorp.com/), used to greatly simplify the task of provisioning and managing resources in the cloud. With Terraform, organizations can describe the desired state of their infrastructure in a simple configuration language and let Terraform plan and apply these changes in an automated way. This way, Terraform allows teams to truly deliver infrastructure-as-code (IaC), which completely change how teams and organizations manage their cloud infrastructure.

### Infrastructure-as-Code and database management
Most cloud-native applications are backed by a database. The database is often the most critical part of many software systems, so making changes to its schema (structure and layout of the data inside) is a very risky business. However, schemas must evolve: as functionality changes over time, the backing tables are added, columns are dropped, indexes are created for performance reasons, and more.

Therefore it is surprising that there is no established way of integrating the management of schema changes (commonly called schema "migrations") into popular Infrastructure-as-Code workflows. For this reason, many organizations are running migrations from within the application code or using solutions outside the ecosystem of Terraform, meaning that management of the production environment is fragmented and hard to synchronize. Atlas aims to change that.

The Atlas Terraform provider allows you to synchronize your database with your desired schema, in a safe and stateful manner. By using Atlas’s core migration engine and embedding it in a Terraform provider, we are enabling teams to manage their database schemas as part of their full IaC workflow. This way, teams can use existing providers (such as AWS or GCP) to provision the database instance and use the Atlas provider to keep the schema in sync. Integrating Atlas with Terraform is especially useful because it couples the state of the infrastructure with the state of the database. It is also extremely neat when using a [dev database](https://atlasgo.io/concepts/dev-database), which is a feature that combines infrastructure and DB management to provide safety and correctness.

### Demo
#### Prerequisites
Make sure you have installed:
* [Docker](https://docs.docker.com/get-docker/)
* [Terraform](https://www.terraform.io/downloads)

Let’s see an example of the provider in action. First, spin a database using docker:
```bash
docker run -p 3306:3306 --name iloveatlas -e MYSQL_ROOT_PASSWORD=pass -e MYSQL_DATABASE=market -d mysql:8
```
Great! Now we have an instance of MySQL database running.

As an extra measure of safety, we will run another identical database which will serve as a [Dev Database](https://atlasgo.io/concepts/dev-database). In short, the dev-db helps to catch errors that can only be detected when applying the schema. It is also useful to format the schema in a correct and predictable way. Read more about it [here](https://atlasgo.io/concepts/dev-database).
Run a second instance of MySQL on another port, to serve as a dev-db:

```shell
docker run -p 3307:3306 --name devdb-greatness -e MYSQL_ROOT_PASSWORD=pass -e MYSQL_DATABASE=market -d mysql:8
```

Next, we need an HCL file describing the desired state of our database. You can use [atlas cli](https://atlasgo.io/cli/getting-started/setting-up) to [inspect](https://atlasgo.io/cli/getting-started/inspection) the state of another database or you can use the following basic schema:
```hcl title="schema.hcl"
table "orders" {
  schema = schema.market
  column "id" {
    null           = false
    type           = int
    auto_increment = true
  }
  column "name" {
    null = false
    type = varchar(20)
  }
  primary_key {
    columns = [column.id]
  }
}

schema "market" {
  charset = "utf8mb4"
  collate = "utf8mb4_0900_ai_ci"
}
```
Save the schema file locally in a file named `schema.hcl`.

Now that we have our database schema we can use terraform to apply that schema to our database.
Create a file named `main.tf` and copy the following snippet:
```hcl title="main.tf"
terraform {
  required_providers {
    atlas = {
      version = "~> 0.4.0"
      source  = "ariga/atlas"
    }
  }
}

provider "atlas" {}

data "atlas_schema" "market" {
  dev_db_url = "mysql://root:pass@localhost:3307/market"
  src = file("${path.module}/schema.hcl")
}

resource "atlas_schema" "market" {
  hcl = data.atlas_schema.market.hcl
  url = "mysql://root:pass@localhost:3306/market"
  dev_db_url = "mysql://root:pass@localhost:3307/market"
}
```
Finally, init terraform:
```shell
terraform init
```
And apply the schema to the database by executing:
```shell
terraform apply --auto-approve
```

Awesome! Now your database should have a table named `orders`. To verify that we can connect to the database:
```shell
$ docker exec -it iloveatlas mysql -ppass --database=market

mysql> show tables;
+------------------+
| Tables_in_market |
+------------------+
| orders           |
+------------------+
1 row in set (0.00 sec)

mysql> show create table orders;
+--------+------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------+
| Table  | Create Table                                                                                                                                                                       |
+--------+------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------+
| orders | CREATE TABLE `orders` (
  `id` int NOT NULL AUTO_INCREMENT,
  `name` varchar(20) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci |
+--------+------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------+
1 row in set (0.00 sec)
```

For more examples and documentation visit the official [GitHub repository](https://github.com/ariga/terraform-provider-atlas) or the [provider page](https://registry.terraform.io/providers/ariga/atlas/latest) on Terraform registry.

### What's next

In this post, we presented the [Atlas Terraform Provider](https://registry.terraform.io/providers/ariga/atlas/latest). The provider currently supports the basic, declarative migration workflow that is available in the Atlas engine. In upcoming versions, we will add support for an additional kind of workflow that is supported by the engine and is called [versioned migration authoring](https://entgo.io/blog/2022/03/14/announcing-versioned-migrations/). In addition, more advanced safety checks (such as simulation on database snapshots) and migration strategies are also being worked on.

While the Terraform provider has just been released, the [core engine](https://github.com/ariga/atlas) that it is driving, is well tested and widely used (especially as the migration engine backing the popular [Ent](https://github.com/ent/ent) framework.) If you, like me, have always wanted to manage your database schema as part of your team's infrastructure-as-code workflow, give the Atlas Terraform provider a try!

Have questions? Feedback? Find our team [on our Discord server](https://discord.gg/zZ6sWVg6NT).
