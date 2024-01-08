---
title: "Announcing Schema Monitoring: Get notified about schema drifts"
authors: masseelch
tags: [db, schema, atlas, agent]
---

Managing database schemas is hard. Since they are the (oftentimes only) stateful component in an application, issues
originating in a faulty database state can have wide impact on your products and services. A relatively simple issue, 
that can have a huge impact is when the database schema is not in sync with what downstream consumers expect it to be.
In today's post, we want to present how Atlas can help detect those schema drifts by monitoring your databases and 
comparing their state with what you expect it to be and notifying you, once there is a schema change. 

### Prerequisites

All you need for this is docker installed on your machine. Then start a database container by running the following 
command.

```shell
docker run --name atlas-agent-demo-db --rm -e POSTGRES_PASSWORD=pass -p 5432:5432 -d postgres:15
```

This PostgreSQL container will act as a production database for this post. Since this container is running locally on 
your developer machine, it most likely is not accessible by the internet and therefore is a good mock for an isolated 
production database. In a real scenario, you'll have something like an AWS RDS or GCP Cloud SQL instance running in a
VPC in your cloud.

### Create an Agent

Since our goal is for Atlas to monitor our schemas and ensure they are in sync with what we have defined, Atlas needs
to be able to connect to our database. However, opening up a production database (or any database) to the internet
is not a good idea. A common solution to this is to have a "thing" running in your VPC, that has connection to the
internet and to your database. This "thing" is commonly called an Agent and Atlas now offers such a solution as well.

The first step to establish a connection between Atlas and your database, is telling Atlas about your intention to do
so. Head over to the general settings and click on the Agents tab.

![image](https://github.com/ariga/atlas/assets/12862103/6551a17e-d1df-4ae6-a6d2-051419c900d1)

Pick a name for your agent and hit **Create Agent**.

![image](https://github.com/ariga/atlas/assets/12862103/8c6ea303-db9a-47da-bfef-ea0fa7a22401)

### Connect the Agent

Now that we created an Agent in Atlas, we need to run the Agent binary in our VPC and check if it can access the
cloud. To authenticate against Atlas, a token will be created that you need to provide the Agent with. Store this token
in a secure place, as you won't be able to see it again. However, you can always create a new one.

![image](https://github.com/ariga/atlas/assets/12862103/2aac6477-deea-4c4a-b555-b04253e0dd7a)

Since in this example we are running the agent locally, we will choose the docker image for the Atlas agent. Copy the
command shown to you in the cloud and execute it. It should look similar to the command below:

```shell
docker run -e ATLAS_TOKEN="aci_*****" arigaio/atlas-agent
```

If your machine has access to the internet and can reach Atlas, you should see the below success message.

![image](https://github.com/ariga/atlas/assets/12862103/55d0646f-197f-4881-8f95-7f5124dc3e79)

### Database Credentials

Since we want the Agent to connect to our database on behalf of Atlas, it needs to know how to access it.
For this, we can assign an agent multiple database connections. Either click the **Set up Database Connection** or
select the **Database Connections** tab and hit **Create Connection**.

:::info
You can only create a connection with an actively running Agent. If there is no Agent selectable in the dropdown,
ensure the Agent binary is still running and has access to the cloud.
:::

Fill out the form with the connection details to your database. In this example, we have a locally running PostgreSQL
docker container. 

:::info
Since we are running both the database and the agent using docker, we need to ensure the Agent container can access the 
database container. A relatively simple solution for this is to run the Agent container with `--network host` (linux) 
or use `host.docker.internal` (mac) as the database host.  
:::

![image](https://github.com/ariga/atlas/assets/12862103/b9d17dbe-e5c3-460a-9dd9-864ddb888ff8)

Atlas offers various ways to provide the agent with credentials to the database. The easiest is to provide the database
password via an environment variable. Select the option in the dropdown and put the name of the variable containing the
password. In this case `DATABASE_PASSWORD`. 

:::warning
In a real example we advise to use a secrets manager or IAM authentication to connect to your database.
:::

:::info
Note: Ensure the Agents environment does contain this variable with the  correct password, e.g. by re-running the agent 
container:

```
docker run -e ATLAS_TOKEN="aci_*****" -e DATABASE_PASSWORD=pass arigaio/atlas-agent
```
:::

To ensure the credentials are correct, Atlas will check if the credentials are working before we can save them. Hit
the **Test Connection** button and wait. It can take a few seconds before the Agent will pick up the job and check
the connection to the database.

If all goes well, we should see a message telling us that Atlas was able to connect to out database through the 
agent.

![image](https://github.com/ariga/atlas/assets/12862103/a0371b48-fc1f-45fb-8f26-9ea1cb3a025f)

### Drift Detection

Now that Atlas can connect to our database, it can start monitoring our schema and warn us, if it detects a drift 
between our migration directory and its deployment. Head over to your migration directory and ensure you have at least
one migration file synced. If you don't have a directory yet, there are step-by-step instructions on how to create one
in our previous post about "[GitOps for Databases, Part 1: CI/CD"](2023-12-06-gitops-for-databases-part-1.mdx#local-setup).

Once your migration directory is synced with the cloud, make sure there is at least one deployment available.
Simply click the **Deployments** link in the sidebar and follow the instructions.

![image](https://github.com/ariga/atlas/assets/12862103/f3977c56-6d41-4cf7-bf3c-8f9f14ad1863)

In this example, we have two deployments of the same migration directory called **production** and **staging**.

![image](https://github.com/ariga/atlas/assets/12862103/32a7f163-8ec6-4b48-91f7-2953e50bb85b)

Awesome! We have a migration directory, which defines the desired state of our database schemas, and we have two 
deployment, which are the schemas we want to monitor and check for schema drifts against our migration directory. All
that is left for us to do is enabling drift detection by toggling the switch. You'll be asked which database connection 
the deployments are reachable with.

![image](https://github.com/ariga/atlas/assets/12862103/9c4e567d-c86b-4c7c-8b35-65dd43f2f123)

Select the connection we created previously, then click **Enable Drift Detection**.

![image](https://github.com/ariga/atlas/assets/12862103/1f8ec1ee-cdb9-4936-ac0a-7f327d804739)

Atlas will immediately trigger a drift detection check for all targets reachable on the selected database connection, 
that were deployed from the current migration directory. As you can see in the following image, Atlas is connecting to 
both our **production** and **staging** deployments and compares their schema with the state of the migration directory.

![image](https://github.com/ariga/atlas/assets/12862103/09a2bcb6-f7bb-4e7c-bf0a-12994112c8de)

In our example, I made a small change to my production schema. Now, it would be great to see what the difference between
the production schema and our migration directory is, right? We will have a look at that in a bit.

![image](https://github.com/ariga/atlas/assets/12862103/707a7f0f-279c-4042-9435-8b146c4cbcf7)

:::info
By default, Atlas will run a drift detection check twice a day.
:::

First, since the schemas are now monitored in the background, wouldn't it be great if we would be notified, if there was
a schema drift? Luckily, Atlas has you covered here as well. Just press the **Set up Notification Channel**, which will
lead you to the migration directories webhook settings. Here you can choose between various ways to be notified about
things happening with your database schemas. One of which is the detection of a schema drift. In this example, we'll be 
configuring Atlas to email us, once a schema drift is detected.

![image](https://github.com/ariga/atlas/assets/12862103/3bd8d392-d236-4110-a685-9b9fc3ba441c)

Great! Now, if a schema drift is detected, you will be notified about it via email. Now, you will remember we Atlas did
already detect a drift in the **production** schema? Let's have a look. Head over to the **Databases** tab, open the 
**production** database and click on the **Drift Detection**. You should see one entry. 

![image](https://github.com/ariga/atlas/assets/12862103/cc169581-622f-4131-b7ba-f8c2eda0d9db)

Open it up, and you will be presented with a nice ERD, visualizing the drift and the HCL / SQL representation of the 
desired and current schema.

![image](https://github.com/ariga/atlas/assets/12862103/a48100c1-9f5d-41c9-b184-ffe816f1d623)

### Wrapping up

That's it! I hope you try out (and enjoy) all of these new features and find them useful.
As always, we would love to hear your feedback and suggestions on our [Discord server](https://discord.gg/zZ6sWVg6NT).
