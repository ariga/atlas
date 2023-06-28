---
id: hcl-variables
slug: /atlas-schema/input-variables
title: HCL Input Variables
---
In some cases, it is desirable to reuse an Atlas HCL document in different contexts.
For example, many organizations manage a multi-tenant architecture where the same database
schema is replicated per tenant. For this reason, the Atlas DDL supports input variables.

Input variables are defined using the `variable` block:

```hcl
variable "comment" {
  type    = string // | int | bool | list(string) | etc.
  default = "default value"
}
```

Once defined, their value can be referenced using `var.<name>`:

```hcl
schema "main" {
  comment = var.comment
}
```

Finally, input variables are passed to Atlas in the `schema apply` command using the
`--var` flag:

```shell
atlas schema apply -u ... -f atlas.hcl --var comment="hello"
```

If a variable is not set from the command line, Atlas tries to use its default value.
If no default value is set, an error is returned:

```text
schemahcl: failed decoding: input value "tenant" expected but missing
```

### Variable schema names

Returning to the use case we described above, let's see how we can use input variables
to manage a multi-tenant architecture.

First, we define our schema in a file named `multi.hcl`:

```hcl title="multi.hcl"
// Define the input variable that contains the tenant name.
variable "tenant" {
  type        = string
  description = "The name of the tenant (schema) to create"
}

// Define the schema, "tenant" here is a placeholder for the final
// schema name that will be defined at runtime.
schema "tenant" {
  // Reference to the input variable.
  name = var.tenant
}
table "users" {
  // Refer to the "tenant" schema. It's actual name will be
  // defined at runtime.
  schema = schema.tenant
  column "id" {
    type = int
  }
}
```

Now suppose we have two tenants, `jerry` and `george`. We can apply the same schema twice:

Once for Jerry:
```text
atlas schema apply -u mysql://user:pass@localhost:3306/ --schema jerry --var tenant=jerry
```
Observe the generated queries apply to the `jerry` schema:
```text
-- Planned Changes:
-- Add new schema named "jerry"
CREATE DATABASE `jerry`
-- Create "users" table
CREATE TABLE `jerry`.`users` (`id` int NOT NULL)
✔ Apply
```
And again for George:
```text
atlas schema apply -u mysql://user:pass@localhost:3306/ --schema george --var tenant=george
```
The generated queries create the `george` schema:
```text
-- Planned Changes:
-- Add new schema named "george"
CREATE DATABASE `george`
-- Create "users" table
CREATE TABLE `george`.`users` (`id` int NOT NULL)
✔ Apply
```
