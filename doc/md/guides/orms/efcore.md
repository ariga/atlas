---
id: efcore
title: Automatic Migration Planning for Entity Framework Core
slug: /guides/orms/efcore
---
import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

## TL;DR
* [Entity Framework (EF) Core](https://docs.microsoft.com/en-us/ef/core/) is an object-relational mapping (ORM) framework for .NET.
* [Atlas](https://atlasgo.io) is an open-source tool for inspecting, planning, linting, and executing schema changes to your database.
* Developers using EF Core can use Atlas to automatically plan schema migrations
 based on the desired state of their schema instead of crafting them by hand.

## Issues with EF Core Migrations

EF Core is the most popular ORM used in the .NET community, supported by **Microsoft**. EF Core allows 
users to manage their database schemas using its [migrations](https://docs.microsoft.com/en-us/ef/core/managing-schemas/migrations/).
EF Core's migrations have long been a popular and reliable choice for managing database schema changes in the C# ecosystem.

However, EF Core migrations have lacks some capabilities can make them difficult to work with:

1. **Support for advanced database features.** Like many ORMs, EF Core is designed to be database-agnostic, which means
   it does not support all the features of every database it can connect to. This can make it difficult to use and manage
   database features such as triggers, stored procedures, Row-level security and custom data types.
2. **Testing migrations.** Migrations are typically considered the most risky part of a deployment. Therefore, automatically
   verifying they are [safe](/versioned/lint) and [correct](/testing/migrate) is paramount.  Like most ORMs, EF Core does not
   provide a way to automatically test migrations.
3. **Production Grade Declarative Flow.** EF Core supports a very basic declarative flow name
   [`EnsureCreated`](https://learn.microsoft.com/en-us/ef/core/managing-schemas/ensure-created#ensurecreated)
   that can be used to create the database without specifying migrations. However, as the documentation 
   [warns](https://learn.microsoft.com/en-us/ef/core/managing-schemas/ensure-created#ensurecreated), this method should
   not be used in production. For teams that want to adapt a "Terraform-for-databases" approach, this can be a blocker.
4. **Integration with modern CI/CD pipelines.** EF Core migrations are typically run using the `dotnet ef` command line tool.
   Migrations should be integrated into the software delivery pipeline to ensure that the database schema is always in sync
   with the application code. This can be difficult to achieve with EF Core migrations.

## Atlas and EF Core

[Atlas](https://atlasgo.io) is a database schema as code tool that allows developers to inspect, plan, test, and execute
schema changes to their database. Atlas can be used to replace EF Core migrations with a more modern DevOps approach. 

Comparing Atlas to EF Core migrations:
* **Loading Core Models.** Similarly to EF Core migrations, Atlas can load the schema of an EF Core project. EF Core users
  can keep using the EF Core models as the source of truth for their database schema.
* **Composing schemas.** Atlas can compose schemas from multiple sources, including EF Core models, SQL files, and
  external schema datasources. This enables users to natively declare schemas that layer advanced database features 
  (such as views, triggers) as part of the schema source of truth which is not possible with EF Core.
* **Automatic planning.** Similarly to EF Core migrations, with its "versioned migrations" workflow, Atlas can 
  automatically plan schema migrations by diffing the data model with the migration directory. 
* **Declarative flow.** Atlas supports a declarative flow that can be used to create the database schema from scratch
  without using migrations. This is useful for teams that want to adapt a "Terraform-for-databases" approach.
* **Testing migrations.** Atlas can automatically lint and test migrations to ensure they are safe and correct. Using
  this capability teams can reduce the risk of deploying migrations to production.
* **Integration with CI/CD pipelines.** Atlas can be integrated into modern CI/CD pipelines using native integrations 
  with popular CI/CD tools like GitHub Actions, CircleCI, GitLab CI, Terraform, Kubernetes, ArgoCD, and more.

## Getting Started

In this guide, we will show how Atlas can be used to automatically plan schema migrations for
EF Core users.

### Prerequisites

* A local project that uses EF Core for data access.

If you don't have one, you can use [TodoApi](https://github.com/davidfowl/TodoApi) written by David Fowler (Microsoft Engineer for 
things like ASP.NET, Aspire, and NuGet, among others) as a starting point:

```bash
git clone git@github.com:davidfowl/TodoApi.git
```

### Using the Atlas EF Core Provider

In this guide, we will use the [Atlas Provider for EF Core](https://github.com/ariga/atlas-provider-ef)
to automatically plan schema migrations for an EF Core project.

A connection to a live database is optional; it depends on the [Database Provider](https://learn.microsoft.com/en-us/ef/core/providers/?tabs=dotnet-core-cli) you are using.

### Installation

<Tabs groupId="operating-systems">
<TabItem value="win" label="Windows">

Use PowerShell to download the Atlas binary:

```powershell
Invoke-WebRequest https://release.ariga.io/atlas/atlas-windows-amd64-latest.exe -OutFile atlas.exe
```

Then move the atlas binary to a directory that is included in your system PATH. 
If you prefer a different directory, you can add it to your system PATH by editing the environment variables.

</TabItem>
<TabItem value="mac_linux" label="macOS + Linux">

Install Atlas from macOS or Linux by running:

```bash
curl -sSf https://atlasgo.sh | sh
```

</TabItem>
</Tabs>

See [atlasgo.io](https://atlasgo.io/getting-started#installation) for more installation options.

#### Atlas Provider for EF Core

This package is available on [NuGet](https://www.nuget.org/packages/atlas-ef/).

Navigate to the `TodoApi` folder and create a tool manifest file:

```bash
dotnet new tool-manifest
```

Install **atlas-ef** as a local tool using the `dotnet tool` command:

```bash
dotnet tool install --local atlas-ef
```

Verify that the tool is installed by running:

```bash
dotnet atlas-ef --version
```

### Loading the EF Core Schema

Next, let's see how to load the EF Core schema into Atlas so it can be used to plan migrations.

Navigate to the `TodoApi` folder and create a new file named `atlas.hcl` with the following contents:

```hcl
data "external_schema" "ef" {
  program = [
    "dotnet",
    "atlas-ef",
  ]
}

env {
  name = atlas.env
  src = data.external_schema.ef.url
  dev = "sqlite://dev?mode=memory" # list of dev dbs can be found here: https://atlasgo.io/concepts/dev-database
  migration {
    dir = "file://atlas-migrations"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}
```

:::note
By looking at [DbContext Creation](https://learn.microsoft.com/en-us/ef/core/cli/dbcontext-creation?tabs=dotnet-core-cli), you 
can see the current DbContext uses SQLite. Therefore, we need to use the correct [dev database](/concepts/dev-database) for Atlas to work.
```csharp title="TodoApi/Program.cs"
builder.Services.AddSqlite<TodoDbContext>(connectionString);
```
:::

### Verifying the Configuration

Next, let's verify Atlas is able to read our desired schema, by running the
[`schema inspect`](/declarative/inspect) command, to inspect our desired schema:

```shell
atlas schema inspect --env ef --url "env://src"
```

Notice that this command uses `env://src` as the target URL for inspection, meaning "the schema represented by the
`src` attribute of the `local` environment block."

Given we have a simple entity `Todo` :

```csharp title="Todo.cs"
public class Todo
{
    [Key]
    public int Id { get; set; }
    [Required]
    public string Title { get; set; } = default!;
    public bool IsComplete { get; set; }
}
```

We should get the following output after running the `inspect` command above:

```hcl
table "Todos" {
  schema = schema.main
  column "Id" {
    null           = false
    type           = integer
    auto_increment = true
  }
  column "Title" {
    null = false
    type = text
  }
  column "IsComplete" {
    null = false
    type = integer
  }
  primary_key {
    columns = [column.Id]
  }
}
schema "main" {
}
```

### Planning Migrations

Atlas supports a [versioned migrations](/concepts/declarative-vs-versioned#versioned-migrations)
workflow, where each change to the database is versioned and recorded in a migration file. You can use the
`atlas migrate diff` command to automatically generate a migration file that will migrate the database
from its latest revision to the current EF Core schema.

Suppose we have the following EF Core `TodoDbContext` class, which has a `Todo` entity:

```csharp title="TodoApi/Todos/Todo.cs"
public class Todo
{
    public int Id { get; set; }
    [Required]
    public string Title { get; set; } = default!;
    public bool IsComplete { get; set; }

    [Required]
    public string OwnerId { get; set; } = default!;
}
```

We can generate a migration file by running this command:

```bash
atlas migrate diff --env ef
```

Running this command will generate files similar to this in the `atlas-migrations` directory:

```bash
atlas-migrations
├── 20240805093706.sql
└── atlas.sum

1 directory, 2 files
```

Examining the contents of `20240805093706.sql`:

```sql
-- truncated Identity tables

CREATE TABLE "Todos" (
    "Id" INTEGER NOT NULL CONSTRAINT "PK_Todos" PRIMARY KEY AUTOINCREMENT,
    "Title" TEXT NOT NULL,
    "IsComplete" INTEGER NOT NULL,
    "OwnerId" TEXT NOT NULL,
    CONSTRAINT "FK_Todos_AspNetUsers_OwnerId" FOREIGN KEY ("OwnerId") REFERENCES "AspNetUsers" ("UserName") ON DELETE CASCADE
);
```

Amazing! Atlas automatically generated a migration file that will create the `Todos` table in our database.
Next, alter the `Todo` class to add a new `Description` field:

```diff
public class Todo
{
    public int Id { get; set; }
    [Required]
    public string Title { get; set; } = default!;
    public bool IsComplete { get; set; }

    [Required]
    public string OwnerId { get; set; } = default!;
+   public string Description { get; set; } = default!;
}
```

Re-run this command:

```bash
atlas migrate diff --env ef
```

Observe a new migration file is generated:

```diff
atlas-migrations
  ├── 20240805093706.sql
+ ├── 20240805093730.sql
  └── atlas.sum

1 directory, 3 files
```

```sql title="20240805093730.sql"
-- Add column "Description" to table: "Todos"
ALTER TABLE `Todos` ADD COLUMN `Description` text NOT NULL;
```

### Next Steps

In this guide, we demonstrated how projects using EF Core can use Atlas to automatically
plan schema migrations. To learn more about using Atlas, here are some
resources to study next:

* Executing migrations with [`migrate apply`](/versioned/apply)
* Verifying migrations are safe using [`migrate lint`](/versioned/lint)
* Testing migrations using [`migrate test`](/testing/migrate)
* Guide for [testing data migrations](/guides/testing/data-migrations)
* Using [Composite Schemas](/blog/2024/01/09/atlas-v0-18#composite-schemas) which can be used to layer advanced database features
  on top of EF Core models.

## Conclusion

In this guide, we demonstrated how projects using EF Core can use Atlas to automatically
plan schema migrations based only on their entity. To learn more about executing
migrations against your production database, read the documentation for the
[`migrate apply`](/versioned/apply) command.

Have questions? Feedback? Find our team [on our Discord server](https://discord.gg/zZ6sWVg6NT).

