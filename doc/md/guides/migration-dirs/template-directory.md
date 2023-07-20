---
id: template-directory
title: Working with template directories
slug: /guides/migration-dirs/template-directory
---

Atlas supports working with dynamic template-based directories, where their content is computed based on the data
variables injected at runtime. These directories adopt the [Go-templates] format, the very same format used by popular 
CLIs such as `kubectl`, `docker` or `helm`.

To create a template directory, you first need to create an Atlas configuration file (`atlas.hcl`) and define the
`template_dir` data source there:

```hcl title="atlas.hcl" {1-4}
data "template_dir" "migrations" {
  path = "migrations"
  vars = {}
}

env "dev" {
  migration {
    dir = data.template_dir.migrations.url
  }
}
```
The `path` defines a path to a local directory, and `vars` defines a map of variables that will be used to interpolate
the templates in the directory.

### Basic Example

We start our guide with a simple MySQL-based example where migration files are manually written and the auto-increment
initial value is configuration based. Let's run `atlas migrate new` with the `--edit` flag and paste the following statement:

```sql {7}
-- Create "users" table.
CREATE TABLE `users` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `role` enum('user', 'admin') NOT NULL,
  `data` json,
  PRIMARY KEY (`id`)
) AUTO_INCREMENT={{ .users_initial_id }};
```

After creating our first migration file, the `users_initial_id` variable should be defined in `atlas.hcl`. Otherwise,
Atlas will fail to interpolate the template.

```hcl title="atlas.hcl" {3-5}
data "template_dir" "migrations" {
  path = "migrations"
  vars = {
    users_initial_id = 1000
  }
}

env "dev" {
  dev = "docker://mysql/8/dev"
  migration {
    dir = data.template_dir.migrations.url
  }
}
```

In order to test our migration directory, we can run `atlas migrate apply` on a temporary MySQL container that Atlas
will spin up and tear down automatically for us:

```shell {3}
atlas migrate apply \
  --env dev \
  --url docker://mysql/8/dev
```

<details><summary>Example output</summary>

```text title="Output"
Migrating to version 20230719093802 (1 migrations in total):

  -- migrating version 20230719093802
    -> CREATE TABLE `users` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `role` enum('user', 'admin') NOT NULL,
  `data` json,
  PRIMARY KEY (`id`)
) AUTO_INCREMENT=1000;
  -- ok (30.953207ms)

  -------------------------
  -- 74.773738ms
  -- 1 migrations 
  -- 1 sql statements
```

</details>

### Inject Data Variables From Command Line

Variables are not always static, and there are times when we need to inject them from the command line. The Atlas
configuration file supports this injection using the `--var` flag. Let's modify our `atlas.hcl` file such that the
value of the `users_initial_id` variable isn't statically defined and must be provided by the user executing the CLI:

```hcl title="atlas.hcl" {1-3,8}
variable "users_initial_id" {
  type = number
}

data "template_dir" "migrations" {
  path = "migrations"
  vars = {
    users_initial_id = var.users_initial_id
  }
}

env "dev" {
  dev = "docker://mysql/8/dev"
  migration {
    dir = data.template_dir.migrations.url
  }
}
```

Trying to execute `atlas migrate apply` without providing the `users_initial_id` variable, will result in an error:
```text
Error: missing value for required variable "users_initial_id"
```

Let's run it the right way and provide the variable from the command line:

```shell
atlas migrate apply \
  --env dev \
  --url docker://mysql/8/dev \
  --var users_initial_id=1000
```

<details><summary>Example output</summary>

```text title="Output"
Migrating to version 20230719093802 (1 migrations in total):

  -- migrating version 20230719093802
    -> CREATE TABLE `users` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `role` enum('user', 'admin') NOT NULL,
  `data` json,
  PRIMARY KEY (`id`)
) AUTO_INCREMENT=1000;
  -- ok (30.953207ms)

  -------------------------
  -- 74.773738ms
  -- 1 migrations 
  -- 1 sql statements
```

</details>

### Read Data Variables From File

Let's add a bit more complexity to our example by inserting seed data to the `users` table. But, to keep our
configuration file tidy, we'll keep the seed data in a different file (`seed_data.json`) and read it from there.

First, we'll create a new migration file by running `atlas migrate new seed_users --edit` and paste the following
statement:

```sql {6-8}
{{ range $line := .seed_users }}
  INSERT INTO `users` (`role`, `data`) VALUES ('user', '{{ $line }}');
{{ end }}
```

The file above expects a data variable named `seed_users` of type `[]string`. It then loops over this variable and
`INSERT`s a record into the `users` table for each JSON line.

For the sake of this example, let's define an example `seed_users.json` file and update the `atlas.hcl` file to inject
the data variable from its content:

```json title="seed_users.json"
{"name": "Ariel"}
{"name": "Rotem"}
```

```hcl title="atlas.hcl" {7,13}
variable "users_initial_id" {
  type = number
}

locals {
  # The path is relative to the `atlas.hcl` file.
  seed_users = split("\n", file("seed_users.json"))
}

data "template_dir" "migrations" {
  path = "migrations"
  vars = {
    seed_users       = local.seed_users
    users_initial_id = var.users_initial_id
  }
}

env "dev" {
  dev = "docker://mysql/8/dev"
  migration {
    dir = data.template_dir.migrations.url
  }
}
```

To check that our data interpolation works as expected, let's run `atlas migrate apply` on a temporary MySQL container
that Atlas will spin up and tear down automatically for us:

```shell {3}
atlas migrate apply \                       
  --env dev \
  --url docker://mysql/8/dev \
  --var users_initial_id=1000
```

<details><summary>Example output</summary>

```text title="Output"
Migrating to version 20230719102332 (2 migrations in total):

  -- migrating version 20230719093802
    -> CREATE TABLE `users` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `role` enum('user', 'admin') NOT NULL,
  `data` json,
  PRIMARY KEY (`id`)
) AUTO_INCREMENT=1000;
  -- ok (38.380244ms)

  -- migrating version 20230719102332
    -> INSERT INTO `users` (`role`, `data`) VALUES ('user', '{"name": "Ariel"}');
    -> INSERT INTO `users` (`role`, `data`) VALUES ('user', '{"name": "Rotem"}');
  -- ok (13.313962ms)

  -------------------------
  -- 95.387439ms
  -- 2 migrations 
  -- 3 sql statements
```

</details>

### Running `migrate diff` on template directories

When running the `atlas migrate diff` command on a template directory, we want to ensure that the data variables defined
in our `atlas.hcl` are shared between the desired state (e.g., HCL or SQL schema) and the current state of the migration
directory, to get an accurate SQL script that moves our database from its previous state to the new one.

Let's demonstrate this using an HCL schema that describes our desired schema and expects one variable: `users_initial_id`.

```hcl title="schema.hcl" {1-3,20}
variable "users_initial_id" {
  type = number
}

table "users" {
  schema = schema.public
  column "id" {
    type = bigint
  }
  column "role" {
    type = enum("user", "admin")
  }
  column "data" {
    type = json
    null = true
  }
  primary_key {
    columns = [column.id]
  }
  auto_increment = var.users_initial_id
}
schema "public"{}
```

Then, we update our `atlas.hcl` configuration to inject the data variable to this schema file and then use it as our
desired state:

```hcl {17-22,25}
variable "users_initial_id" {
  type = number
}

locals {
  seed_users = split("\n", file("seed_users.json"))
}

data "template_dir" "migrations" {
  path = "migrations"
  vars = {
    seed_users       = local.seed_users
    users_initial_id = var.users_initial_id
  }
}

data "hcl_schema" "app" {
  path = "schema.hcl"
  vars = {
    users_initial_id = var.users_initial_id
  }
}

env "dev" {
  src = data.hcl_schema.app.url
  dev = "docker://mysql/8/dev"
  migration {
    dir = data.template_dir.migrations.url
  }
}
```

To test that our data interpolation works as expected, let's run `atlas migrate diff` and ensure the HCL schema and
the migration directory are in sync:

```shell
atlas migrate diff \
  --env dev \
  --var users_initial_id=1000
```

```text
The migration directory is synced with the desired state, no changes to be made
```

Then, let's change our `data` column to be `NOT NULL` by updating the `schema.hcl` file and run `atlas migrate diff`:

```diff title="schema.hcl"
  column "data" {
    type = json
-   null = true
+   null = false
  }
```

```shell
atlas migrate diff modify_user_data \
  --env dev \
  --var users_initial_id=1000
```


After checking our migration directory, we can see that Atlas has generated a new migration file that modifies the `data`
column to be `NOT NULL`, while leaving the template files untouched:

```sql title="migrations/20230720074923_modify_user_data.sql"
-- Modify "users" table
ALTER TABLE `users` MODIFY COLUMN `data` json NOT NULL;
```

### Conclusion

In this example, we've seen how to use the `template_dir` data source to create a migration directory whose content is
dynamically computed at runtime, based on the data variables defined in the `atlas.hcl` file. We've also seen how
the data variables can be injected from various sources, such as JSON files or CLI flags. Lastly, we've showed how
data variables can be shared between template directories and HCL schemas to ensure commands like `atlas migrate diff`
can be utilized to generate migration plan automatically for us.

Have questions? Feedback? Find our team [on our Discord server](https://discord.gg/zZ6sWVg6NT).

[Go-templates]: https://pkg.go.dev/text/template#hdr-Actions