---
title: "Passwordless Schema Migrations on RDS with Atlas"
authors: rotemtam
tags: [migrations, rds, iam, atlas]
---

## TL;DR

Atlas now supports AWS IAM authentication, which enables you to perform passwordless schema migrations on your
RDS databases. To use it with Atlas, add the `aws_rds_token` data source to your `atlas.hcl` configuration file:

```hcl
data "aws_rds_token" "mydb" {
  endpoint = "mydb.123456789012.us-east-1.rds.amazonaws.com:3306"
  username = "atlas"
}
```

To skip the intro and jump straight to the tutorial, click [here](#demo-time).

## Introduction

Passwords have long been the default mechanism for authentication, but they come with a set of known vulnerabilities. In recent years, 
our industry has shifted towards alternative authentication methods due to these weaknesses. For databases, which store
critical data, this security concern is even more important.

Schema migrations require elevated permissions, making it even more essential to approach them with
utmost care in order to prevent security breaches. In this post, we'll show how to use Atlas to perform passwordless schema
migrations on databases hosted in AWS's [RDS](https://aws.amazon.com/rds/) service.

## The Problem with Passwords

Passwords are considered a weak authentication mechanism for humans logging in to systems since
they can be leaked or guessed. For this reason, many services offer more robust authentication
methods, such as [multi-factor authentication](https://en.wikipedia.org/wiki/Multi-factor_authentication)
or [single sign-on](https://en.wikipedia.org/wiki/Single_sign-on).

In this post, we'll focus on the security concerns of passwords (or API Tokens) for automated systems (such as
CI/CD pipelines), which are used to perform schema migrations. Such tokens pose a challenge to securing systems in a few ways:

* **Leaks.** When stored in configuration files, passwords are typically in plain text, increasing the risk of leaks.
* **Granularity.** When passwords are shared among multiple users, it becomes challenging to grant and revoke access for
  individual users based on role changes or emerging security concerns.
* **Visibility.** Because passwords are usually visible to operators and are shared by multiple users, it's hard to track
  who performed which operation once authenticated.
* **Rotation.** Because passwords tend to be long-lived, their rotation becomes a cumbersome task.

## IAM Authentication

IAM, short for [Identity and Access Management](https://en.wikipedia.org/wiki/Identity_management), is a framework that
has been adopted by virtually all cloud providers for managing digital identities and their permissions. Unlike traditional 
password-based systems where credentials are stored and checked, IAM verifies who (or what) is making a request and 
then checks the permissions associated with that identity.

IAM services supply mechanisms for generating short-lived tokens based on the identity of the caller. In addition, 
these services provide a centralized way to manage permissions (by creating granular access policies and grouping them
into roles) and auditing capabilities to track how subjects (users or services) use the system.

Configured correctly, under IAM, every subject can access exactly what it needs and nothing more, without
ever having to use a password or some other token that might be leaked or stolen. When a person leaves your organization (or no 
longer needs access to a particular resource), you can revoke their access by updating their IAM role.

## IAM authentication for Databases

Most databases in use today predate IAM and have developed their own internal mechanisms for authentication
and authorization. In recent years, cloud vendors have worked to create a bridge between IAM and databases, allowing
users to authenticate their identity to databases using IAM credentials. In this post, we'll focus on AWS's implementation of IAM
authentication for RDS databases.

#### How does it work? 
First, enable IAM authentication on your RDS instance. This installs a plugin on the database that
allows it to authenticate users with IAM credentials instead of passwords. Read how to do this in the 
[AWS documentation](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/UsingWithRDS.IAMDBAuth.Enabling.html)

Next, create a database user and grant it permission to authenticate using IAM. 
  
In MySQL, execute a statement like this:
```sql
CREATE USER 'atlas' IDENTIFIED WITH AWSAuthenticationPlugin as 'RDS';
```

In PostgreSQL, execute a statement like this:

```sql
CREATE USER atlas; 
GRANT rds_iam TO atlas;
```

Finally, create an IAM policy that allows subjects to create RDS connection tokens. This policy can then
be attached to roles for developers or services that need to connect to the database. Read how to do this in the
[AWS documentation](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/UsingWithRDS.IAMDBAuth.IAMPolicy.html).

## IAM Authentication with Atlas

Tools that perform schema migrations such as Atlas require elevated permissions to perform their tasks. For example,
they need to be able to inspect the database's information schema tables as well as create and drop resources. For this
reason, any mechanism that can further protect the security of their credentials is essential, making IAM authentication
a great fit. To support this use case, we have recently added support for AWS IAM authentication to Atlas.

## Demo Time!

Let's see how to use Atlas to perform passwordless schema migrations on an RDS database.

For the purpose of this demo, we assume that we have a PostgreSQL database running in RDS with IAM authentication
enabled. We also assume that we have a user named `atlas` that has been granted the `rds_iam` permission and that 
we have created an IAM policy that allows us to generate RDS tokens.

Start by creating a new file named `atlas.hcl` to store our project configuration and add the following content:

```hcl
// Define local variables for the database endpoint and username.
locals {
  endpoint = "atlas-demo.xyzxyz.us-east-1.rds.amazonaws.com:5432"
  username = "atlas"
}

// Use the "aws_rds_token" data source to generate a token for the database.
data "aws_rds_token" "db" {
  endpoint = local.endpoint
  username = local.username
  region = "us-east-1"
}

// Define an environment named "rds" that uses the generated token.
env "rds" {
  url = "postgres://${local.username}:${urlescape(data.aws_rds_token.db)}@${local.endpoint}/postgres"
}
```

Lets break this example down:
* The `locals` block defines two variables – `endpoint` and `username` – that we use to store the database endpoint and
  the username of the user created in the database.
* Next, we define an `aws_rds_token` data source to generate a token for the database. To read more about this
  data source, see the [documentation](/atlas-schema/projects#data-source-aws_rds_token).
* Finally, we define an environment named `rds` that uses the generated token. The `url` property defines the
  [connection URL](/concepts/url) that Atlas will use to connect to the database. Notice that we use the `urlescape`
  function to escape the token before embedding it in the URL.

Now that we have our project configuration, let's use Atlas to inspect the database schema. Run the following command:

```shell
atlas schema inspect -c "file://atlas.hcl" --env rds
```

You should see output similar to the following:

```hcl
schema "public" {
}
```

Amazing! This output indicates that Atlas was able to both connect to the database and inspect the schema without us having
to provide it with any credentials!

## Wrapping up

In this post, we discussed the security concerns around passwords and how IAM authentication can help mitigate them. We
also demonstrated how to use Atlas to perform passwordless schema migrations on an RDS database using IAM authentication.
If you use Atlas to perform schema migrations on RDS databases, we encourage you to give IAM authentication a try!

#### How can we make Atlas better?

We would love to hear from you [on our Discord server](https://discord.gg/zZ6sWVg6NT) :heart:.
