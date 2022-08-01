---
title: "Announcing Atlas Project Files"
authors: rotemtam
tags: [cli, announcement]
image: https://blog.ariga.io/uploads/images/posts/v0.4.1/project-file.png
---

A few days ago we released [v0.4.1](https://github.com/ariga/atlas/releases/tag/v0.4.1) of
Atlas. Along with [a multitude](https://github.com/ariga/atlas/compare/v0.4.0...v0.4.1) of
improvements and fixes, I'm happy to announce the release of a feature that we've been planning
for a while: [Project Files](https://atlasgo.io/atlas-schema/projects).

Project files provide a way to describe and interact with multiple environments while working
with Atlas. A project file is a file named `atlas.hcl` that contains one or more `env` blocks,
each describing an environment. Each environment has a reference to where the schema definition
file resides, a database URL and an array of the schemas in the database that are managed by Atlas:

```hcl
// Define an environment named "local".
env "local" {
  // Declare where the schema definition file resides.
  src = "./schema/project.hcl"

  // Define the URL of the database which is managed in
  // this environment.
  url = "mysql://localhost:3306"

  // Define the URL of the Dev Database for this environment.
  // See: https://atlasgo.io/dev-database
  dev = "mysql://localhost:3307"

  // The schemas in the database that are managed by Atlas.
  schemas = ["users", "admin"]
}

env "dev" {
  // ... a different env
}
```

Project files arose from the need to provide a better experience for developers using the CLI.
For example, consider you are using Atlas to plan migrations for your database schema. In this case,
you will be running a command similar to this to plan a migration:

```
atlas migrate diff --dev-url mysql://root:password@localhost:3306 --to file://schema.hcl --dir file://migrations --format atlas
```

With project files, you can define an environment named `local`:

```hcl
env "local" {
    url = "mysql://root:password@localhost:3306"
    dev = "mysql://root:password@localhost:3307"
    src = "./schema.hcl"
    migration {
        dir = "file://migrations"
        format = atlas
    }
}
```

Then run the `migrate diff` command against this environment using the `--env` flag:

```
atlas migrate diff --env local
```

Alternatively, suppose you want to use Atlas to apply the schema on your staging environment.
Without project files, you would use:

```
atlas schema apply -u mysql://root:password@db.ariga.dev:3306 --dev-url mysql://root:password@localhost:3307 -f schema.hcl
```
To do the same using a project file, define another env named `staging`:

```hcl
env "staging" {
  url = "mysql://root:password@db.ariga.dev:3306"
  dev = "mysql://root:password@localhost:3307"
  src = "./schema.hcl"
}
```
Then run:
```
atlas schema apply --env staging
```

### Passing credentials as input values

Similar to [schema definition files](/atlas-schema/sql-resources), project files also support [Input Variables](/ddl/input-variables). This means
that we can define `variable` blocks on the project file to declare which values should be provided when the file is
evaluated. This mechanism can (and should) be used to avoid committing to source control database credentials.
To do this, first define a variable named `db_password`:

```hcl
variable "db_password" {
  type = string
}
```

Next, replace the database password in all connection strings with a reference to this variable, for example:

```hcl
env "staging" {
  url = "mysql://root:${var.db_password}@db.ariga.dev:3306"
  dev = "mysql://root:${var.db_password}@localhost:3307"
  src = "./schema.hcl"
}
```

If we run `schema apply` without providing the password input variable, we will receive an
error message:

```
Error: missing value for required variable "db_password"
```

To provide the input variable run:

```
atlas schema apply --env staging --var db_password=pass
```

Input variables can be used for many other use cases by passing them as [input values to schema files](https://atlasgo.io/atlas-schema/projects#project-input-variables).

### What's next

In this post, I presented [Project Files](https://atlasgo.io/atlas-schema/projects), a new feature recently added to Atlas
to help developers create more fluent workflows for managing changes to their database schemas. In the coming weeks
we will be adding a few more improvements to the dev flow, such as support for marking a specific environment as
the default one (alleviating the need to specify `--env` in many cases) and [multi-file schema definitions](https://github.com/ariga/atlas/issues/510).

Have questions? Feedback? Find our team [on our Discord server](https://discord.gg/zZ6sWVg6NT).
