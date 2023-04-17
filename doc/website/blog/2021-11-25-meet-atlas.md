---
title: "Meet Atlas CLI: Inspect and Apply changes to your database schema"
date: "2021-11-25"
author: Yoni Davidson
authorURL: "https://github.com/yonidavidson"
authorImageURL: "https://avatars0.githubusercontent.com/u/5472778"
authorTwitter: yonidavidson
url: /meet-atlas-cli/
image: https://blog.ariga.io/images/logo.png
---

At [Ariga](https://ariga.io), we are building a new kind of platform that we call an Operational Data Graph.
This platform enables software engineers to manage, maintain and access complex data architectures as if they were one
database. Today, we are open-sourcing a CLI for [Atlas](https://atlasgo.io), one of the fundamental building blocks of our platform.

During my career, the scope of what is expected of me as a software engineer has increased significantly.
Developers are no longer expected just to write code, we are expected to provision infrastructure,
manage databases, define deployments and monitor systems in production.

Nowadays, one of the responsibilities we have as software engineers is to manage the database schema of our applications.
Once seen as falling strictly under the domain of DBAs, today developers everywhere are responsible for defining
database schemas and changing them over time. Because an application's database carries its state,
all clients and servers are severely impacted if it stops functioning properly.  Therefore,
over the years many techniques and tools were developed to deal with this process,
which is called _migrating_ the database.

In the last few years we have seen a lot of progress in the field of tools for provisioning infrastructure.
From early projects such as [Chef](https://www.chef.io/) and [Puppet](https://puppet.com/), to more recent work such as
[Terraform](https://www.terraform.io/), a lot of thought and effort has been put across the industry to build tools
that simplify and standardize the process.
Instead of manually installing and configuring software and services, the common thread between all of
these projects is that they are based on machine-readable definition files, a concept also 
known as [infrastructure-as-code](https://en.wikipedia.org/wiki/Infrastructure_as_code) (IaC).


### Enter: Atlas
Atlas is at the core of Ariga's platform. In this post, I would like to share with you the work we've done so far to
provide a solid foundation for managing databases in a way that's akin to infrastructure-as-code practices.

* The [Atlas DDL](/guides/ddl) (Data-definition Language): we have created the Atlas DDL, a new configuration language designed
to capture an organization's data topology - including relational database schemas.
This language is currently described in an HCL syntax (similar to TerraForm),
but will support more syntaxes such as JSON and TypeScript in the future. The Atlas DDL currently supports
defining schemas for [SQL](https://atlasgo.io/atlas-schema/sql-resources) databases such as MySQL, Postgres, SQLite and MariaDB, but in the future, we plan to add support for other types of databases. For example: 
```hcl
table "users" {
  schema = "default"
  column "id" {
    type = "int"
  }
  column "name" {
    type = "string"
  }
  column "manager_id" {
    type = "int"
  }
  primary_key {
    columns = [
        table.users.column.id
    ]
  }
  index "idx_name" {
    columns = [
      table.users.column.name
    ]
    unique = true
  }
  foreign_key "manager_fk" {
    columns = [table.users.column.manager_id]
    ref_columns = [table.users.column.id]
    on_delete = "CASCADE"
    on_update = "NO ACTION"
  }
}
```

* __The Atlas CLI__ On top of the building blocks provided by the DDL, we started building our CLI tool to support the
two most basic functions:

  * __"Schema Inspect"__ -  Create a schema specification file from a database.
  * __"Schema Apply"__   -  Migrate a database to a new desired state.

Many infrastructure-as-code projects have taken the declarative approach, in which the developer articulates the desired 
state of the system and the tool is responsible for figuring out a plan to get there. As we discussed above,
changing database schemas safely is a delicate practice, so we had to build the Atlas CLI to be smart enough to
understand the nuance of changes for each type of database.

### Atlas in action

Let's see how Atlas CLI works with real databases.
Let's start a MySQL container:

```shell
docker run --name atlas-db  -p 3306:3306  -e MYSQL_ROOT_PASSWORD=pass -e MYSQL_DATABASE=example   mysql:8.0.27
```

Connect to our database using a native client to validate:
```shell
docker  exec -it  atlas-db  mysql --password='pass' example
```

```shell
mysql> show tables;
Empty set (0.00 sec)

mysql>
```

Let's see how Atlas inspects it:

```shell
atlas schema inspect -u "mysql://root:pass@localhost:3306/example" > atlas.hcl
```

As expected, an empty schema:
```hcl
# cat atlas.hcl
schema "example" {
}
```

Let's update our schema to:

```hcl
# cat atlas.hcl
table "users" {
  schema = "example"
  column "id" {
    null = false
    type = "int"
  }
  column "name" {
    null = false
    type = "string"
    size = 255
  }
  column "manager_id" {
    null = false
    type = "int"
  }
  primary_key {
    columns = [table.users.column.id, ]
  }
  foreign_key "manager_fk" {
    columns     = [table.users.column.manager_id, ]
    ref_columns = [table.users.column.id, ]
    on_update   = "NO ACTION"
    on_delete   = "CASCADE"
  }
  index "idx_name" {
    unique  = true
    columns = [table.users.column.name, ]
  }
  index "manager_fk" {
    unique  = false
    columns = [table.users.column.manager_id, ]
  }
}
schema "example" {
}

```
And apply our changes!
```shell
atlas schema apply -u "mysql://root:pass@localhost:3306/example" -f atlas.hcl



-- Planned Changes:
-- Add Table : users
CREATE TABLE `example`.`users` (`id` int NOT NULL, `name` varchar(255) NOT NULL, `manager_id` int NOT NULL, PRIMARY KEY (`id`), UNIQUE INDEX `idx_name` (`name`), CONSTRAINT `manager_fk` FOREIGN KEY (`manager_id`) REFERENCES `example`.`users` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE) ;
Use the arrow keys to navigate: ↓ ↑ → ←
? Are you sure?:
  ▸ Apply
    Abort

```

Of course we are sure !

Using CLI to examine our database:
```sql
mysql> describe users;
+------------+--------------+------+-----+---------+-------+
| Field      | Type         | Null | Key | Default | Extra |
+------------+--------------+------+-----+---------+-------+
| id         | int          | NO   | PRI | NULL    |       |
| name       | varchar(255) | NO   | UNI | NULL    |       |
| manager_id | int          | NO   | MUL | NULL    |       |
+------------+--------------+------+-----+---------+-------+
3 rows in set (0.00 sec)

mysql>
```

Let's make sure that it has the FK:
```sql
mysql> show create table users;
+-------+-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------+
| Table | Create Table                                                                                                                                                                                                                                                                                                                                                            |
+-------+-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------+
| users | CREATE TABLE `users` (
  `id` int NOT NULL,
  `name` varchar(255) NOT NULL,
  `manager_id` int NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_name` (`name`),
  KEY `manager_fk` (`manager_id`),
  CONSTRAINT `manager_fk` FOREIGN KEY (`manager_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci |
+-------+-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------+
1 row in set (0.00 sec)

mysql>

```
Now let's see that Atlas inspects this correctly:

```shell
atlas schema inspect -u "mysql://root:pass@localhost:3306/example" > atlas.hcl
```

```hcl
# cat atlas.hcl
table "users" {
  schema = "example"
column "id" {
    null = false
    type = "int"
  }
  column "name" {
    null = false
    type = "string"
    size = 255
  }
  column "manager_id" {
    null = false
    type = "int"
  }
  primary_key {
    columns = [table.users.column.id, ]
  }
  foreign_key "manager_fk" {
    columns     = [table.users.column.manager_id, ]
    ref_columns = [table.users.column.id, ]
    on_update   = "NO ACTION"
    on_delete   = "CASCADE"
  }
  index "idx_name" {
    unique  = true
    columns = [table.users.column.name, ]
  }
  index "manager_fk" {
    unique  = false
    columns = [table.users.column.manager_id, ]
  }
}
schema "example" {
}

```

Let's see what happens when we try to reapply the same change:

```shell
atlas schema apply -u "mysql://root:pass@localhost:3306/example" -f atlas.hcl
Schema is synced, no changes to be made
```

In this example we have shown how we can inspect a MySQL database schema and apply a change.

### What's Next?

The Atlas DDL opens up a world of tools and services, and with the help of our community,
we are planning to push the development ecosystem forward. A list of tools that are on our road map includes:
* Integrations with Terraform, GitHub actions and Kubernetes.
* Extended migration logics such as renaming columns, adding or dropping nullability and altering enums.
* Toolsets for examining the migration history and reproducing it. 

We hope that you find Atlas CLI as exciting as we do, and we invite you to [contribute](https://github.com/ariga/atlas)
your ideas and code. 
