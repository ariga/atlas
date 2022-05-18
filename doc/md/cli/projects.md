---
title: Project Structure
id: projects
slug: /cli/projects
---
## Atlas Project Files

Project files provide a convenient way to describe and work with different
environments which are managed using Atlas.  A project file is a file named
`atlas.hcl` which is located at the root of the project and contains one or 
`env` blocks. For example:

```hcl
// Define an environment named "local"
env "local" {
  // Declare where the schema definition file resides.
  src = "./schema/project.hcl"
  
  // Define the URL of the database which is managed in
  // this environment.
  url = "mysql://localhost:3306"
  
  // Define the URL of the Dev Database for this environment
  // See: https://atlasgo.io/dev-database
  dev = "mysql://localhost:3307"
  
  // The schemas in the database that are managed by Atlas.
  schemas = ["users", "admin"]
}

env "dev" {
  // ... a different env
}
```

Once defined, a project's environment can be worked against using the `--env` flag. 
For example:

```shell
$ atlas schema apply --env local
```

Will run the `schema apply` command against the database that is defined for the `local`
environment. 

### Projects with Versioned Migrations

Environments may declare a `migration_dir` block to configure how versioned migrations
work in the specific environment:

```hcl
env "local" {
    // ..
    migration_dir {
        // URL where the migration directory resides. Only filesystem directories
        // are currently supported but more options will be added in the future.
        url = "file://migrations"
        // Format of the migration directory: atlas | flyway | liquibase | goose | golang-migrate
        format = atlas
    }
}
```

Once defined, `migrate` commands can use this configuration, for example:
```shell
$ atlas migrate validate --env local
```
Will run the `migrate validate` command against the Dev Database defined in the 
`local` environment. 

### Passing Input Values

Project files may pass [input values](/ddl/input-variables) to variables defined in 
the Atlas schema of the environment. To do this simply provide additional attributes
to the environment block:
```hcl
env "local" {
  url = "sqlite://test?mode=memory&_fk=1"
  src = "./schema.hcl"
  
  // Other attributes are passed as input values to "schema.hcl":
  tenant = "rotemtam"
}
```

These values can then be consumed by variables defined in the schema file:

```hcl
variable "tenant" {
  type = string
}
schema "main" {
  name = var.tenant
}
```

### Project Input Variables

Project files may also declare [input variables](../ddl/input.md) that can be supplied to the CLI 
at runtime. For example: 

```hcl
variable "tenant" {
  type = string
}

env "local" {
  url = "sqlite://test?mode=memory&_fk=1"
  src = "./schema.hcl"
  
  // The value for "tenant" is determined by the user at runtime.
  tenant = var.tenant
}
```
To set the value for this variable at runtime, use the `--var` flag:

```shell
$ atlas schema apply --env local --var tenant=rotemtam
```

It is worth mentioning that when running Atlas commands within a project using
the `--env` flag, all input values supplied at the command-line are passed only
to the project file, and not propagated automatically to children schema files. 
This is done with the purpose of creating an explicit contract between the environment
and the schema file.