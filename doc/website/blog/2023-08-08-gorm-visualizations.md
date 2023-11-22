---
title: "How to Visualize GORM Models with Atlas"
authors: rotemtam
tags: [gorm, visualization, erd]
---

We recently shipped a [new feature](/blog/2023/08/06/atlas-v-0-13#built-in-schema-visualization) 
in Atlas that enables developers to quickly create ERD (Entity Relationship Diagram) visualizations
of their database schemas.  Today, I want to show how this feature can be used in tandem with one
of Atlas's core capabilities - "Schema Loaders" - to produce visualizations of GORM models.

But first, let's introduce the different characters in this story:
* [Atlas](https://atlasgo.io) - an open-source tool for managing database schemas.
* [GORM](https://gorm.io) - one of the most popular ORMs for Go.
* [ERD](https://en.wikipedia.org/wiki/Entity%E2%80%93relationship_model) - a diagram that shows the relationships between entities in a database.
  (we'll see an example in a minute)
* [Schema Loaders](https://atlasgo.io/atlas-schema/external) - a feature in Atlas that allows users to load their
  database schemas from different sources.

In this blog post, we will create a toy GORM application, and show how you can use Atlas to visualize
the database schema that GORM generates for you.

### Step 1: Bootstrap the GORM application

Start by creating a new directory for our project. Then, initialize a new Go module:

```shell
mkdir gormviz
cd gormviz
go mod init example.io/gormviz
```

Next, we will install the GORM package and the SQLite driver:

```shell
go get -u gorm.io/gorm
go get -u gorm.io/driver/sqlite
```

Now, let's create a file that will contain our database models. We will call it `models/models.go`:

```go title="models/models.go"
package models

import "gorm.io/gorm"

type User struct {
  gorm.Model
  Name string
  Pets []Pet
}

type Pet struct {
  gorm.Model
  Name   string
  User   User
  UserID uint
}
```

### Step 2: Setup Atlas

Now that we have a GORM application, let's set up Atlas. First, we will install the Atlas CLI:

```shell
curl -sSf https://atlasgo.sh | sh
```
For more installation options, see [these instructions](/getting-started#installation).

Next, let's install the GORM Atlas provider which will allow us to load our GORM models into Atlas:

```shell
go get ariga.io/atlas-provider-gorm@v0.1.0
```

Next, to prevent the Go Modules system from dropping this dependency from our `go.mod` file, let's
follow the Go Module's [official recommendation](https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module)
for tracking dependencies of tools and add a file named `tools.go` with the following contents:

```go title="tools.go"
//go:build tools
package main

import _ "ariga.io/atlas-provider-gorm/gormschema"
```
Alternatively, you can simply add a blank import to the `models.go` file we created
above.

Finally, to tidy things up, run: 

```text
go mod tidy
```

After the provider is installed, let's create an Atlas project file to glue everything together.

Create a new file named `atlas.hcl` with the following contents:

```hcl title="atlas.hcl"
data "external_schema" "gorm" {
  program = [
    "go",
    "run",
    "-mod=mod",
    "ariga.io/atlas-provider-gorm",
    "load",
    "--path", "./models",
    "--dialect", "sqlite",
  ]
}

env "gorm" {
  src = data.external_schema.gorm.url
  dev = "sqlite://file?mode=memory&_fk=1"
}
```
This file defines two things:
* An [External Schema Loader](https://atlasgo.io/atlas-schema/projects#data-source-external_schema) named `gorm` that will load our GORM models
  into Atlas by executing the `ariga.io/atlas-provider-gorm` program on our `models` package. 
* An [Environment](/atlas-schema/projects#environments) named `gorm` that will use the `gorm` schema loader
  to load the models into a SQLite database in memory.

### Step 3: Visualize the GORM models

Now that we have everything set up, let's run Atlas and see what we get:

```shell
atlas schema inspect -w --env gorm --url 'env://src'
```

Let's break down this command: 
* The `schema inspect` command is used to inspect a database schema and produce a representation of it. 
* The `-w` flag tells Atlas to open a web browser and display the visualization.
* The `--env gorm` flag tells Atlas to use the `gorm` environment we defined in the `atlas.hcl` file.
* The `--url 'env://src'` flag tells Atlas to use schema defined in the `src` attribute of the `gorm` env.

When we run this command,

Atlas will prompt us to ask if we want this visualization to be shared on the
[Public Atlas Playground](https://gh.atlasgo.cloud/explore) or if we want to keep it private:

```text
? Where would you like to share your schema visualization?:
  ▸ Publicly (gh.atlasgo.cloud)
    Your personal workspace (requires 'atlas login')
```

As this is just a toy example, let's publish it to the public playground by pressing `Enter`.

Once we do this, Atlas will extract the database schema from our GORM models, and publish it to
the playground, printing the URL of the visualization:
```text
https://gh.atlasgo.cloud/explore/60d94de7
```

[![](https://atlasgo.io/uploads/gormviz/gorm-erd.png)](https://gh.atlasgo.cloud/explore/60d94de7)

### Privately sharing visualizations

By publishing visualizations to the public playground, you can easily share them with others
by simply sending them the URL. However, if you want to keep your visualizations private, such 
that they are only accessible to you and your team, you can do so by first logging in to Atlas
Cloud:

```shell
atlas login
```
Your browser should open and prompt you to either login or create a free account if you don't
already have one.  Upon completion, you should see a message in your terminal that looks like this:

```text
You are now connected to rotemtam85 on Atlas Cloud.
``` 

Once you are logged in, you can re-run the `atlas schema inspect --env gorm --web` command
and Atlas will prompt you to ask if you want to share the visualization in your personal workspace:

```text
? Where would you like to share your schema visualization?:
    Publicly (gh.atlasgo.cloud)
  ▸ Privately (rotemtam85.atlasgo.cloud)
```
Select the `Privately` option and press `Enter`. Atlas will then publish the visualization to your
personal workspace and print the URL of the visualization:

```text
https://rotemtam85.atlasgo.cloud/explore/2da80ffx
```

### Wrapping up

In this post, we have shown how to use Atlas to visualize the database schema that GORM generates
for your application. ERD visualizations are just one of the many features that Atlas provides for working with your database
schema. To learn more about what you can do with the Atlas GORM provider, check out the [relevant guide](/guides/orms/gorm).

#### How can we make Atlas better?

We would love to hear from you [on our Discord server](https://discord.gg/zZ6sWVg6NT) :heart:.
