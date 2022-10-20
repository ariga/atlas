---
title: Project Configuration
id: projects
slug: /atlas-schema/projects
---

### Project Files

Project files provide a convenient way to describe and interact with multiple
environments when working with Atlas. A project file is a file named
`atlas.hcl` and contains one or more `env` blocks. For example:

```hcl
// Define an environment named "local"
env "local" {
  // Declare where the schema definition resides.
  // Also supported:
  //   src = "./dir/with/schema"
  //   src = ["multi.hcl", "file.hcl"]
  src = "./project/schema.hcl"

  // Define the URL of the database which is managed in
  // this environment.
  url = "mysql://localhost:3306"

  // Define the URL of the Dev Database for this environment
  // See: https://atlasgo.io/concepts/dev-database
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
atlas schema apply --env local
```

Will run the `schema apply` command against the database that is defined for the `local`
environment.

### Projects with Versioned Migrations

Environments may declare a `migration` block to configure how versioned migrations
work in the specific environment:

```hcl
env "local" {
    // ..
    migration {
        // URL where the migration directory resides. Only filesystem directories
        // are currently supported but more options will be added in the future.
        dir = "file://migrations"
        // Format of the migration directory: atlas | flyway | liquibase | goose | golang-migrate
        format = atlas
    }
}
```

Once defined, `migrate` commands can use this configuration, for example:
```shell
atlas migrate validate --env local
```
Will run the `migrate validate` command against the Dev Database defined in the
`local` environment.

### Passing Input Values

Project files may pass [input values](input-variables) to variables defined in
the Atlas schema of the environment. To do this simply provide additional attributes
to the environment block:
```hcl
env "local" {
  url = "sqlite://test?mode=memory&_fk=1"
  src = "schema.hcl"

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

Project files may also declare [input variables](../atlas-schema/input.md) that can be supplied to the CLI
at runtime. For example:

```hcl
variable "tenant" {
  type = string
}

env "local" {
  url = "sqlite://test?mode=memory&_fk=1"
  src = "schema.hcl"

  // The value for "tenant" is determined by the user at runtime.
  tenant = var.tenant
}
```
To set the value for this variable at runtime, use the `--var` flag:

```shell
atlas schema apply --env local --var tenant=rotemtam
```

It is worth mentioning that when running Atlas commands within a project using
the `--env` flag, all input values supplied at the command-line are passed only
to the project file, and not propagated automatically to children schema files.
This is done with the purpose of creating an explicit contract between the environment
and the schema file.

## Schema Arguments and Attributes

Project configuration files support different types of blocks.

### Input Variables

Project files support defining input variables that can be injected through the CLI, [read more here](input.md).

- `type` - The type constraint of a variable.
- `default` - Define if the variable is optional by setting its default value.

```hcl
variable "tenants" {
  type = list(string)
}

variable "url" {
  type    = string
  default = "mysql://root:pass@localhost:3306/"
}

env "local" {
  // Reference an input variable.
  url = var.url
}
```

### Local Values

The `locals` block allows defining a list of local variables that can be reused multiple times in the project.

```hcl
locals {
  tenants  = ["tenant_1", "tenant_2"]
  base_url = "mysql://${var.user}:${var.pass}@${var.addr}"
  
  // Reference local values. 
  db1_url  = "${local.base_url}/db1"
  db2_url  = "${local.base_url}/db2"
}
```

### Data Sources

Data sources enable users to retrieve information stored in an external service or database. The currently supported
data sources are: [`sql`](#data-source-sql).

#### Data source: `sql`

##### Arguments {#data-source-sql-arguments}

- `url` - The [URL](../concepts/url.mdx) of the target database.
- `query` - Query to execute.
- `args` - Optional arguments for any placeholder parameters in the query.

##### Attributes {#data-source-sql-attributes}

- `count` - The number of returned rows.
- `values` - The returned values. e.g. `list(string)`.
- `value` - The first value in the list, or `nil`.

```hcl
data "sql" "tenants" {
  url = var.url
  query = <<EOS
SELECT `schema_name`
  FROM `information_schema`.`schemata`
  WHERE `schema_name` LIKE ?
EOS
  args = [var.pattern]
}

env "prod" {
  // Reference a data source.
  for_each = toset(data.sql.tenants.values)
  url      = urlsetpath(var.url, each.value)
}
```

### Environments

The `env` block defines an environment block that can be selected by using the `--env` flag.

##### Arguments {#environment-arguments}

- `for_each` - A meta-argument that accepts a map or a set of strings and is used to compute an `env` instance for each
set or map item. See the example [below](#multi-environment-example).

- `url` - The [URL](../concepts/url.mdx) of the target database.

- `dev` - The [URL](../concepts/url.mdx) of the [Dev Database](../concepts/dev.md).

- `schemas` - A list of strings defines the schemas that Atlas manages.

- `exclude` - A list of strings defines glob patterns used to filter resources on inspection.

- `migration` - A block defines the migration configuration of the env.
  - `dir` - The [URL](../concepts/url.mdx) to the migration directory.
  - `revisions_schema` - An optional name to control the schema that the revisions table resides in.

- `log` - A block defines the logging configuration of the env per command.
  - `migrate`
    - `apply` - Set the `--log` flag by setting custom logging for `migrate apply`.
    - `lint` - Set the `--log` flag by setting custom logging for `migrate lint`.
    - `status` - Set the `--log` flag by setting custom logging for `migrate status`.

- `lint` - A block defines the migration linting configuration of the env.
  - `log` - Override the `--log` flag by setting a custom logging for `migrate lint`.
  - `lastest` - A number configures the `--latest` option.
  - `git.base` - A run analysis against the base Git branch.
  - `git.dir` - A path to the repository working directory.

##### Multi Environment Example

Atlas adopts the `for_each` meta-argument that [Terraform uses](https://www.terraform.io/language/meta-arguments/for_each)
for `env` blocks. Setting the `for_each` argument will compute an `env` block for each item in the provided value. Note
that `for_each` accepts a map or a set of strings.

```hcl
env "prod" {
  for_each = toset(data.sql.tenants.values)
  url      = urlsetpath(var.url, each.value)
  migration {
    dir = "file://migrations"
  }
  log {
    migrate {
      apply = format(
        "{{ json . | json_merge %q }}",
        jsonencode({
          Tenant : each.value
        })
      )
    }
  }
}
```

## Configure Migration Linting

Project files may declare `lint` blocks to configure how migration linting runs in a specific environment or globally.

```hcl
lint {
  destructive {
    // By default, destructive changes cause migration linting to error
    // on exit (code 1). Setting `error` to false disables this behavior.
    error = false
  }
  // Custom logging can be enabled using the `log` attribute.
  log = <<EOS
{{- range $f := .Files }}
	{{- json $f }}
{{- end }}
EOS
}

env "local" {
  // Define a specific migration linting config for this environment.
  // This block inherits and overrides all attributes of the global config.
  lint {
    latest = 1
  }
}

env "ci" {
  lint {
    git {
      base = "master"
      // An optional attribute for setting the working
      // directory of the git command (-C flag).
      dir = "<path>"
    }
  }
}
```



