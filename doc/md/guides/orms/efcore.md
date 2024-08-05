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

## Automatic Migration Planning for EF Core

EF Core is one of the most popular ORMs widely used in the .NET community, supported by **Microsoft**. EF Core allows 
users to manage their database schemas using its [migrations](https://docs.microsoft.com/en-us/ef/core/managing-schemas/migrations/) 
feature, which is usually sufficient during development and in many simple cases.

However, there are two well-known issues with EF Core migrations ([efcore#31790](https://github.com/dotnet/efcore/issues/31790)):

1. The SQL is not visible in merge/pull requests for review purposes.
2. The designer files repeat the entire schema for each migration.

Atlas supports SQL-based [versioned migrations](/concepts/declarative-vs-versioned#versioned-migrations) methodology and can 
automatically plan database schema migrations for developers using EF Core.
Atlas plans migrations by calculating the diff between the _current_ state of the database,
and its _desired_ state.

In the context of versioned migrations, the current state can be thought of as the database schema that would have
been created by applying all previous migration scripts.

The desired schema of your application can be provided to Atlas via an [External Schema Datasource](/atlas-schema/projects#data-source-external_schema),
which is any program that can output a SQL schema definition to stdout.
To use Atlas with EF Core, users can utilize the [EF Core Atlas Provider](https://github.com/ariga/atlas-provider-ef),
a .NET tool that can be used to load the schema of an EF Core project into Atlas.

In this guide, we will show how Atlas can be used to automatically plan schema migrations for
EF Core users.

## Prerequisites

* A local project that uses EF Core for data access.

If you don't have one, you can use [TodoApi](https://github.com/davidfowl/TodoApi) from David Fowler (Microsoft Engineer for 
things like ASP.NET, Aspire, and NuGet, among others) as a starting point:

```bash
git clone git@github.com:davidfowl/TodoApi.git
```

## Using the Atlas EF Core Provider

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

Then add the directory where the Atlas binary is located to the system's PATH environment variable.

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

### Configuration

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
can see the current DbContext uses Sqlite. Therefore we need to use the correct dev database for Atlas to work.
```csharp title="TodoApi/Program.cs"
builder.Services.AddSqlite<TodoDbContext>(connectionString);
```
:::

### Usage

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

```
atlas-migrations
|-- 20240805093706.sql
|-- atlas.sum

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
  |-- 20240805093706.sql
+ |-- 20240805093730.sql
  |-- atlas.sum

1 directory, 3 files
```

```sql title="20240805093730.sql"
-- Add column "Description" to table: "Todos"
ALTER TABLE `Todos` ADD COLUMN `Description` text NOT NULL;
```

## Conclusion

In this guide, we demonstrated how projects using EF Core can use Atlas to automatically
plan schema migrations based only on their entity. To learn more about executing
migrations against your production database, read the documentation for the
[`migrate apply`](/versioned/apply) command.

Have questions? Feedback? Find our team [on our Discord server](https://discord.gg/zZ6sWVg6NT).

