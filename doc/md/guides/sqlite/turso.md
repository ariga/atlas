---
id: turso
title: Working with Turso
slug: /guides/sqlite/turso
---
[Turso](https://turso.tech) is an edge-hosted, distributed database based on
[libSQL](https://github.com/libsql/libsql), an open-source and open-contribution
fork of SQLite. It was designed to minimize query latency for applications where 
queries come from anywhere in the world. 

Engineers can use Atlas to manage their Turso databases using the SQLite driver by
using the `libsql+ws://` (for local environments) or `libsql+wss://` schemas in their connection URLs.

This guide will walk you through the process of setting up Turso and using Atlas to manage
your Turso databases.

## Set up Turso on your local machine

1. Start by installing Turso on your machine:
  ```
  curl -sSfL https://get.tur.so/install.sh | bash
  ```
2. Verify that Turso is installed by running the following command:
  ```
  turso --version
  ```
3. Sign up for a free Turso account using this command:
   ```
   turso auth signup
   ```
   The CLI launches your default browser and asks to log in with GitHub. 
   The first time you log in, you are asked to grant the GitHub Turso 
   app some permissions to your account. Accept this in order to continue.

## Create your first Turso edge-database

1. Create a new Turso database using the following command:
   ```
   turso db create atlas
   ```
   The Turso system will provision a new edge database for you and print something similar to this:
   ```
   Created database atlas in Frankfurt, Germany (fra) in 9 seconds.
   ```

2. Next, run the following command to find the URL of your newly created database:
   ```
   turso db show atlas --url
   ```
   Make note of this address as we will use it in the next section.

3. Next, create an access token to use with Atlas:
   ```
   turso db tokens create atlas
   ```
   The CLI will print the access token to the console. Make note of this token  
   as we will use it in the next section as well. 

## Create a configuration file for Atlas

Next, create an Atlas [configuration file](/atlas-schema/projects) named `atlas.hcl`
with the following contents:

```hcl
variable "token" {
  type    = string
  default = getenv("TURSO_TOKEN")
}

env "turso" {
  url     = "libsql+wss://<REPLACE WITH YOUR SUBDOMAIN>.turso.io?authToken=${var.token}"
  exclude = ["_litestream*"]
}
```
Let's break down this configuration file:

1. We define an input variable named `token` which will be used to store the Turso access token.
   The variable is of the `string` type and takes the value of the `TURSO_TOKEN` environment variable by default.
2. We define an environment named `turso` which configures our interactions with our new Turso database:
  * We use the `libsql+wss://` schema to connect to the Turso database.
  * The `url` parameter is set to the URL of the Turso database which we created earlier.
  * The `authToken` parameter is set to the value of the `token` variable which we defined earlier.
  * The `exclude` parameter is set to exclude the `_litestream*` tables which are created by the Litestream
    replication engine and are usually not relevant to the application.

## Use Atlas to manage your Turso database

With this configuration file in place, we can now use Atlas to manage our Turso database.

Let's show some common operations we can perform with Atlas:

### Declarative migrations (HCL)

We can use the `atlas schema apply` command to manage the schema of our Turso database.

Start by defining a schema file named `schema.hcl` with the following contents:

```hcl
schema "main" {
}

table "users" {
  schema = schema.main
  column "id" {
    type = int
  }
  column "name" {
    type = varchar(255)
  }
  column "manager_id" {
    type = int
  }
  primary_key {
    columns = [
      column.id
    ]
  }
  index "idx_name" {
    columns = [
      column.name
    ]
    unique = true
  }
  foreign_key "manager_fk" {
    columns = [column.manager_id]
    ref_columns = [column.id]
  }
}
```

Next, set the `TURSO_TOKEN` environment variable to the value of the Turso access token
which we created earlier:

```
export TURSO_TOKEN=<your jwt token>
```

Finally, run the following command to apply the schema to the Turso database:
```
atlas schema apply --env turso --to file://schema.hcl
```
Atlas will prompt you to approve the changes before applying them to the database:
```
-- Planned Changes:
-- Create "users" table
CREATE TABLE `users` (`id` int NOT NULL, `name` varchar NOT NULL, `manager_id` int NOT NULL, PRIMARY KEY (`id`), CONSTRAINT `manager_fk` FOREIGN KEY (`manager_id`) REFERENCES `users` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE);
-- Create index "idx_name" to table: "users"
CREATE UNIQUE INDEX `idx_name` ON `users` (`name`);
Use the arrow keys to navigate: ↓ ↑ → ←
? Are you sure?:
  ▸ Apply
    Abort
```
After approving the changes, Atlas will apply the schema to the Turso database.

### Inspection 

We can use the `atlas schema inspect` command to inspect the schema of our Turso database.
After applying the changes we mentioned above, we can run the following command to inspect
the schema of our Turso database:
```
atlas schema inspect --env turso
```
Atlas will print the schema of the Turso database:
<details>
<summary>HCL Output</summary>

```hcl
table "users" {
   schema = schema.main
   column "id" {
      null = false
      type = int
   }
   column "name" {
      null = false
      type = varchar
   }
   column "manager_id" {
      null = false
      type = int
   }
   primary_key {
      columns = [column.id]
   }
   foreign_key "manager_fk" {
      columns     = [column.manager_id]
      ref_columns = [table.users.column.id]
      on_update   = NO_ACTION
      on_delete   = CASCADE
   }
   index "idx_name" {
      unique  = true
      columns = [column.name]
   }
}
schema "main" {
}
```
</details>
You can also use the `--format` flag to change the output format. For example, you can use
the `--format` flag to print the schema in SQL format:

```
atlas schema inspect --env turso --format '{{ sql . "  " }}'
```

<details>
<summary>SQL Output</summary>

Output:
```
-- Create "users" table
CREATE TABLE `users` (
  `id` int NOT NULL,
  `name` varchar NOT NULL,
  `manager_id` int NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `manager_fk` FOREIGN KEY (`manager_id`) REFERENCES `users` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "idx_name" to table: "users"
CREATE UNIQUE INDEX `idx_name` ON `users` (`name`);
```

</details>
