---
id: explore-inspection
title: How to inspect a local database in the Cloud
slug: /guides/cloud/explore-inspection
---

import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';
import CodeBlock from '@theme/CodeBlock';


Atlas has a [Cloud platform](https://atlasgo.cloud/signup) that offers an [Explore feature](https://gh.atlasgo.cloud/explore)
without having to sign up. The Explore feature allows you to generate an Entity Relationship Diagram (ERD) from your
database by either manually typing in your schema, or importing from an existing database.

Due to the fact that this is a web-based platform, connecting to your local database won't work. There are two quick and
easy ways to get around this and still get the benefits of this feature:

1. Inspect your database with Atlas and copy it to your clipboard in a single command.
2. Use `ngrok` to create a public URL for your database.

### Inspecting with Atlas and copying to clipboard
One of Atlas's main features is inspection. Once [downloading Atlas](/getting-started), you can easily use the inspect command to
automatically generate an HCL schema from your database.

In the Cloud platform, in order to generate an ERD from a local database, you can provide the HCL schema itself. To
quickly do so, run the inspect command and pipe the result to copy the schema to your clipboard.

Depending on your operating system, the command will be:


<Tabs
defaultValue="mac"
values={[
{label: 'macOS', value: 'mac'},
{label: 'Windows', value: 'windows'},
]}>
<TabItem value="mac">

     atlas schema inspect -u "mysql://root:pass@localhost:3306/db_name" | pbcopy

</TabItem>
<TabItem value="windows">

    atlas schema inspect -u "mysql://root:pass@localhost:3306/db_name" | clip

</TabItem>
</Tabs>

:::info
**Linux users**, first follow [this tutorial](https://ostechnix.com/how-to-use-pbcopy-and-pbpaste-commands-on-linux/)
on how to install `pbcopy`, and then you can use the same command as macOS. Note that this will not work on all Linux
environments, so be sure to find a tutorial that fits yours.
:::

Paste the schema in the Cloud and click visualize.

![explore_schema](https://atlasgo.io/uploads/images/explore_schema.png)


### Connecting to a local database with ngrok

[ngrok](https://ngrok.com) is a popular tool that allows developers to securely expose a local development server to the
internet with a public URL. This can be useful for testing webhooks, or for allowing other people to access your
local development environment for collaboration or testing purposes.

We will learn how to connect to a local database using ngrok.

### Install ngrok

Begin by [installing ngrok](https://ngrok.com/download) on your local machine.

### Start a TCP tunnel
Next, we will start a _TCP tunnel_, which is where we will channel our traffic to. In our case, we will channel the
port on which our database is running.

To do so, use the command `ngrok tcp` with the relevant port that our service is running on.
For example, if we have a MySQL instance running on port 3306 we will run the following:
```
ngrok tcp 3306
```

Now ngrok will provide us with a public URL that we can use to access our local database.

![ngrok_url](https://atlasgo.io/uploads/images/ngrok_url.png)

In this example, we can see that ngrok is running our MySQL instance on the URL `tcp://4.tcp.ngrok.io:16496`.
We can connect to our local database with a [database URL](/concepts/url) that looks something like:

```
mysql://root:pass@4.tcp.ngrok.io:16496/db_name
```
:::note
Make sure to remove the 'tcp://' from the URL and write in your local username and password to access the database.
:::

Open the [Explore feature](https://gh.atlasgo.cloud/explore) and click on `import`.

![Explore Page](https://atlasgo.io/uploads/images/explore_page.png)

A modal will open up where we can enter in our database URL.

![Import Modal](https://atlasgo.io/uploads/images/import_modal.png)

Paste in the database URL we just created and click import.

### Wrapping up
In this guide we showed how to easily inspect a database schema using Atlas or ngrok to make using the Explore
feature quick and easy when importing a schema from a local database.

Have questions? Feedback? Feel free to reach out [on our Discord server](https://discord.gg/zZ6sWVg6NT).
