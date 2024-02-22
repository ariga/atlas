---
id: postgres-automatic-migrations
title: Automatic Migrations for PostgreSQL with Atlas
slug: /guides/postgres/automatic-migrations
tags: [postgresql]
---

import InstallationInstructions from '../../components/_installation_instructions.mdx';
import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

[PostgreSQL](https://www.postgresql.org/) is an open-source relational database management system known for its reliability and robust feature set.
It offers powerful capabilities for handling complex queries, ensuring data integrity, and scaling to meet the needs of
growing applications.

However, managing a large database schema in Postgres can be challenging due to the complexity of related data
structures and the need for coordinated schema changes across multiple teams and applications.

#### Enter: Atlas

Atlas helps developers manage their database schema as code - abstracting away the intricacies of database schema
management. With Atlas, users provide the desired state of the database schema and Atlas automatically plans the
required migrations.

In this guide, we will dive into setting up Atlas for PostgreSQL, and introduce the different workflows available.

## Prerequisites

1. Docker
2. Atlas installed on your machine:
<InstallationInstructions />


## Inspecting our Database

Let's start off by spinning up a database using Docker:
```shell
docker run --rm -d --name atlas-demo -e POSTGRES_PASSWORD=pass -e POSTGRES_DB=demo -p 5432:5432 postgres
```

For this example we will begin with a minimal database with a `users` table and an `id` as the primary key.

```sql
CREATE TABLE "users" (
    "id" bigint,
    "name" varchar NOT NULL,
    PRIMARY KEY ("id")
);
```

To create the table above on our local database, we can run the following command:

```sql
docker exec -i atlas-demo psql -U postgres -c "CREATE TABLE \"users\" (\"id\" bigint, \"name\" varchar NOT NULL, PRIMARY KEY (\"id\"));" demo
```

The `atlas schema inspect` command supports reading the database description provided by a [URL](/concepts/url) and outputting it in
different formats, including [Atlas DDL](/atlas-schema/hcl.mdx) (default), SQL, and JSON. In this guide, we will
demonstrate the flow using both the Atlas DDL and SQL formats, as the JSON format is often used for processing the
output using `jq`.

<Tabs>
<TabItem value="hcl" label="Atlas DDL (HCL)" default>

To inspect our locally-running Postgres instance, use the `-u` flag and write the output to a file named `schema.hcl`:
```shell
atlas schema inspect -u "postgres://postgres:pass@:5432/demo?search_path=public&sslmode=disable"  > schema.hcl
```

Open the `schema.hcl` file to view the Atlas schema that describes our database.

```hcl title="schema.hcl"
table "users" {
  schema = schema.public
  column "id" {
    null = false
    type = bigint
  }
  column "name" {
    null = false
    type = character_varying
  }
  primary_key {
    columns = [column.id]
  }
}

schema "public" {
  comment = "standard public schema"
}
```
This first block represents a [table](/atlas-schema/hcl.mdx#table) resource with `id` and `name`
columns. The `schema` field references the `public` schema that is defined in the block below. In addition, the `primary_key`
sub-block defines the `id` column as the primary key for the table. Atlas strives to mimic the syntax of the database
that the user is working against. In this case, the type for the `id` column is `bigint`, and `character_varying` for the `name` column.


</TabItem>
<TabItem value="sql" label="SQL">

To inspect our locally-running Postgres instance, use the `-u` flag and write the output to a file named `schema.sql`:

```shell
atlas schema inspect -u "postgres://postgres:pass@:5432/demo?search_path=public&sslmode=disable" --format '{{ sql . }}' > schema.sql
```

Open the `schema.sql` file to view the inspected SQL schema that describes our database.

```sql title="schema.sql"
-- Create "users" table
CREATE TABLE "users" (
    "id" bigint NOT NULL,
    "name" character varying NOT NULL,
    PRIMARY KEY ("id")
);
```

</TabItem>
</Tabs>

:::info
For in-depth details on the `atlas schema inspect` command, covering aspects like inspecting specific schemas,
handling multiple schemas concurrently, excluding tables, and more, refer to our documentation
[here](/declarative/inspect).
:::

To generate an Entity Relationship Diagram (ERD), or a visual representation of our schema, we can add the `-w` flag
to the inspect command:

```shell
atlas schema inspect -u "postgres://postgres:pass@:5432/demo?search_path=public&sslmode=disable" -w
```

![pg-inspect](https://atlasgo.io/uploads/postgres/images/postgres-inspect.png)

## Declarative Migrations

The declarative approach lets users manage schemas by defining the desired state of the database as code.
Atlas then inspects the target database and calculates an execution plan to reconcile the difference between the desired and actual states.
Let's see this in action.

We will start off by making a change to our schema file, such as adding a `repos` table:

<Tabs>
<TabItem value="hcl" label="Atlas DDL (HCL)" default>

```hcl title=schema.hcl
table "users" {
  schema = schema.public
  column "id" {
    null = false
    type = bigint
  }
  column "name" {
    null = false
    type = character_varying
  }
  primary_key {
    columns = [column.id]
  }
}
// highlight-start
table "repos" {
  schema = schema.public
  column "id" {
    type = bigint
    null = false
  }
  column "name" {
    type = character_varying
    null = false
  }
  column "owner_id" {
    type = bigint
    null = false
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "fk_repo_owner" {
    columns     = [column.owner_id]
    ref_columns = [table.users.column.id]
  }
// highlight-end
schema "public" {
  comment = "standard public schema"
}
```

</TabItem>
<TabItem value="sql" label="SQL">

```sql title="schema.sql"
-- Create "users" table
CREATE TABLE "users" (
  "id" bigint NOT NULL,
  "name" character varying NOT NULL,
  PRIMARY KEY ("id")
);

// highlight-start
-- Create "repos" table
CREATE TABLE "repos" (
    "id" bigint NOT NULL,
    "name" character varying NOT NULL,
    "owner_id" bigint NOT NULL,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("id") REFERENCES "users" ("id")
);
//highlight-end
```

</TabItem>
</Tabs>

Now that our _desired state_ has changed, to apply these changes to our database, Atlas will plan a migration for us
by running the `atlas schema apply` command:


<Tabs>
<TabItem value="hcl" label="Atlas DDL (HCL)" default>

```shell
atlas schema apply \
-u "postgres://postgres:pass@:5432/demo?search_path=public&sslmode=disable" \
--to file://schema.hcl \
--dev-url "docker://postgres/15/dev?search_path=public"
```

</TabItem>
<TabItem value="sql" label="SQL">

```shell
atlas schema apply \
-u "postgres://postgres:pass@:5432/demo?search_path=public&sslmode=disable" \
--to file://schema.sql \
--dev-url "docker://postgres/15/dev?search_path=public"
```

</TabItem>
</Tabs>

Approve the proposed changes, and that's it! You have successfully run a declarative migration.
:::info
For a more detailed description of the `atlas schema apply` command refer to our documentation
[here](/declarative/apply).
:::

To ensure that the changes have been made to the schema, let's run the `inspect` command with the `-w` flag once more
and view the ERD:

![pg-repos-inspect](https://atlasgo.io/uploads/postgres/images/postgres-repos-inspect.png)

## Versioned Migrations

Alternatively, the versioned migration workflow, sometimes called "change-based migrations", allows each change to the
database schema to be checked-in to source control and reviewed during code-review. Users can still benefit from Atlas
intelligently planning migrations for them, however they are not automatically applied.

### Creating the first migration

In the versioned migration workflow, our database state is managed by a _migration directory_. The migration directory
holds all of the migration files created by Atlas, and the sum of all files in lexicographical order represents the current
state of the database.

To create our first migration file, we will run the `atlas migrate diff` command, and we will provide the necessary parameters:

* `--dir` the URL to the migration directory, by default it is file://migrations.
* `--to` the URL of the desired state. A state can be specified using a database URL, HCL or SQL schema, or another migration directory.
* `--dev-url` a URL to a [Dev Database](/concepts/dev-database) that will be used to compute the diff.


<Tabs>
<TabItem value="hcl" label="Atlas DDL (HCL)" default>

```shell
atlas migrate diff initial \
--to file://schema.hcl \
--dev-url "docker://postgres/15/dev?search_path=public"
```

</TabItem>
<TabItem value="sql" label="SQL">

```shell
atlas migrate diff initial \
--to file://schema.sql \
--dev-url "docker://postgres/15/dev?search_path=public"
```

</TabItem>
</Tabs>

Run `ls migrations`, and you'll notice that Atlas has automatically created a migration directory for us, as well as
two files:

<Tabs>
<TabItem value="file" label="20240221153232_initial.sql" default>

```sql
-- Create "users" table
CREATE TABLE "public"."users" (
    "id" bigint NOT NULL,
    "name" character varying NOT NULL,
    PRIMARY KEY ("id")
);
-- Create "repos" table
CREATE TABLE "public"."repos" (
    "id" bigint NOT NULL,
    "name" character varying NOT NULL,
    "owner_id" bigint NOT NULL,
    PRIMARY KEY ("id"),
    CONSTRAINT "fk_repo_owner" FOREIGN KEY ("owner_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
```

</TabItem>
<TabItem value="atlas.sum" label="atlas.sum">

```shell
h1:19FfvbJvenroC2lBiH2G46Oeao6YDJSqz7co+bNKCFY=
20240221153232_initial.sql h1:ACoDqBEQ80lC6pTSyjEL1wsjZhc5RLzOrVisBb+SEDQ=
```

</TabItem>
</Tabs>

The migration file represents the current state of our database, and the sum file is used by Atlas to maintain the integrity
of the migration directory. To learn more about the sum file, read the [documentation](/concepts/migration-directory-integrity).

### Pushing migration directories to Atlas

Now that we have our first migration, we can apply it to a database. There are multiple ways to accomplish this, with
most methods covered in the [guides](/guides) section. In this example, we'll demonstrate how to push migrations to
[Atlas Cloud](https://atlasgo.cloud), much like how Docker images are pushed to Docker Hub.

<div style={{textAlign: 'center'}}>
<img src="https://atlasgo.io/uploads/postgres/images/first-push.png" alt="postgres migrate push" width="100%"/>
<p style={{fontSize: 12}}>Migration Directory created with <code>atlas migrate push</code></p>
</div>

First, let's [log in to Atlas](https://auth.atlasgo.cloud/signup). If it's your first time,
you will be prompted to create both an account and a workspace (organization):

<Tabs>
<TabItem value="web" label="Via Web">

```shell
atlas login
```

</TabItem>
<TabItem value="token" label="Via Token">

```shell
atlas login --token "ATLAS_TOKEN"
```

</TabItem>
<TabItem value="env-var" label="Via Environment Variable">

```shell
ATLAS_TOKEN="ATLAS_TOKEN" atlas login
```

</TabItem>
</Tabs>

Let's name our new migration project `app` and run `atlas migrate push`:

```shell
atlas migrate push app \
--dev-url "docker://postgres/15/dev?search_path=public"
```

Once the migration directory is pushed, Atlas prints a URL to the created directory, similar to the once shown in the
image above.


### Applying migrations

Once our `app` migration directory has been pushed, we can apply it to a database from any CD platform without
necessarily having our directory there.

Let's create another database using Docker to resemble a local environment, this time on port `5431`:
```shell
docker run --rm -d --name atlas-local-demo -e POSTGRES_PASSWORD=pass -e POSTGRES_DB=local -p 5431:5432 postgres
```

Next, we'll create a simple Atlas configuration file (`atlas.hcl`) to store the settings for our local environment:

```hcl title="atlas.hcl" {1}
# The "dev" environment represents our local testings.
env "local" {
  url = "postgres://postgres:pass@:5431/local?search_path=public&sslmode=disable"
  migration {
    dir = "atlas://app"
  }
}
```

The final step is to apply the migrations to the database. Let's run `atlas migrate apply` with the `--env` flag
to instruct Atlas to select the environment configuration from the `atlas.hcl` file:

```shell
atlas migrate apply --env local
```

Boom! After applying the migration, you should receive a link to the deployment and the database where the migration
was applied. Here's an example of what it should look like:

<div style={{textAlign: 'center'}}>
<img src="https://atlasgo.io/uploads/postgres/images/first-deployment.png" alt="first deployment" width="100%"/>
<p style={{fontSize: 12}}>Migration deployment report created with <code>atlas migrate apply</code></p>
</div>

### Generating another migration

After applying the first migration, it's time to update our schema defined in the schema file and tell Atlas to generate
another migration. This will bring the migration directory (and the database) in line with the new state defined by the
desired schema (schema file).

Let's make two changes to our schema:

* Add a new `description` column to our repos table
* Add a new `commits` table

<Tabs>
<TabItem value="hcl" label="Atlas DDL (HCL)" default>

```hcl title=schema.hcl
table "users" {
  schema = schema.public
  column "id" {
    null = false
    type = bigint
  }
  column "name" {
    null = false
    type = character_varying
  }
  primary_key {
    columns = [column.id]
  }
}
table "repos" {
  schema = schema.public
  column "id" {
    type = bigint
    null = false
  }
  column "name" {
    type = character_varying
    null = false
  }
// highlight-start
  column "description" {
    type = character_varying
    null = true
  }
// highlight-end
  column "owner_id" {
    type = bigint
    null = false
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "fk_repo_owner" {
    columns     = [column.owner_id]
    ref_columns = [table.users.column.id]
  }
// highlight-start
table "commits" {
  schema = schema.public
  column "id" {
    type = bigint
    null = false
  }
  column "message" {
    type = character_varying
    null = false
  }
  column "repo_id" {
    type = bigint
    null = false
  }
  column "author_id" {
    type = bigint
    null = false
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "fk_commit_repo" {
    columns     = [column.repo_id]
    ref_columns = [table.repos.column.id]
  }
  foreign_key "fk_commit_author" {
    columns     = [column.author_id]
    ref_columns = [table.users.column.id]
  }
}
// highlight-end
schema "public" {
  comment = "standard public schema"
}
```
</TabItem>
<TabItem value="sql" label="SQL">

```sql title = "schema.sql"
-- Create "users" table
CREATE TABLE "users" (
  "id" bigint NOT NULL,
  "name" character varying NOT NULL,
  PRIMARY KEY ("id")
);

-- Create "repos" table
CREATE TABLE "repos" (
  "id" bigint NOT NULL,
  "name" character varying NOT NULL,
  // highlight-next-line
  "description" character varying NULL,
  "owner_id" bigint NOT NULL,
  PRIMARY KEY ("id"),
  FOREIGN KEY ("id") REFERENCES "users" ("id")
);

// highlight-start
-- Create "commits" table
CREATE TABLE "commits" (
  "id" bigint,
  "message" character varying NOT NULL,
  "repo_id" bigint NOT NULL,
  "author_id" bigint NOT NULL,
  PRIMARY KEY ("id"),
  FOREIGN KEY ("repo_id") REFERENCES "repos" ("id"),
  FOREIGN KEY ("author_id") REFERENCES "users" ("id")
);
// highlight-end
```
</TabItem>
</Tabs>

Next, let's run the `atlas migrate diff` command once more:

<Tabs>
<TabItem value="hcl" label="Atlas DDL (HCL)" default>

```shell
atlas migrate diff add_commits \
--to file://schema.hcl \
--dev-url "docker://postgres/15/dev?search_path=public"
```

</TabItem>
<TabItem value="sql" label="SQL">

```shell
atlas migrate diff add_commits \
--to file://schema.sql \
--dev-url "docker://postgres/15/dev?search_path=public"
```

</TabItem>
</Tabs>

Run `ls migrations`, and you'll notice that a new migration file has been generated.

```sql title="20240222075145_add_commits.sql"
-- Modify "repos" table
ALTER TABLE "public"."repos" ADD COLUMN "description" character varying NULL;
-- Create "commits" table
CREATE TABLE "public"."commits" (
  "id" bigint NOT NULL,
  "message" character varying NOT NULL,
  "repo_id" bigint NOT NULL,
  "author_id" bigint NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_commit_author" FOREIGN KEY ("author_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "fk_commit_repo" FOREIGN KEY ("repo_id") REFERENCES "public"."repos" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
```

Let's run `atlas migrate push` again and observe the new file on the migration directory page.

```shell
atlas migrate push app \
--dev-url "docker://postgres/15/dev?search_path=public"
```

<div style={{textAlign: 'center'}}>
<img src="https://atlasgo.io/uploads/postgres/images/second-push.png" alt="postgres migrate push" width="100%"/>
<p style={{fontSize: 12}}>Migration Directory created with <code>atlas migrate push</code></p>
</div>

## Next Steps

In this guide we learned about the declarative and versioned workflows, and how to use Atlas to generate migrations,
push them to an Atlas workspace and apply them to databases.

Next steps:
* Read the [full docs](/atlas-schema/hcl) to learn HCL schema syntax or about specific Postgres [column types](/atlas-schema/hcl-types#postgresql)
* Learn how to [set up CI](/cloud/setup-ci) for your migration directory
* Deploy schema changes with [Terraform](/integrations/terraform-provider) or [Kubernetes](/integrations/kubernetes/operator)
* Learn about [modern CI/CD principles](/guides/modern-database-ci-cd) for databases

For more in-depth guides, check out the other pages in [this section](/guides) or visit our [Docs](/getting-started) section.

Have questions? Feedback? Find our team on our [Discord server](https://discord.com/invite/zZ6sWVg6NT).
