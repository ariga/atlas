---
id: aws-ecs-fargate
slug: /guides/deploying/aws-ecs-fargate
title: Deploying to ECS/Fargate
---
[AWS Elastic Container Service (ECS)](https://aws.amazon.com/ecs/) is a popular way to deploy containerized applications
to AWS. ECS is a managed service that allows you to run containers on a cluster of EC2
instances, or on AWS Fargate, a serverless compute engine for containers.

In this guide, we will demonstrate how to deploy schema migrations to ECS/Fargate using
Atlas.  As deploying to ECS/Fargate is a vast topic that is beyond the scope of this
guide, we will focus on the migration part only.

Because of its operational simplicity, we will discuss deployment to ECS where tasks
are run on Fargate, but the techniques discussed here are relevant to any ECS deployment.

## Prerequisites

Prerequisites to the guide:


1. [A service running on ECS/Fargate](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/ECS_AWSCLI_Fargate.html)
   defining your application.
2. A connection string to a database that is accessible from the ECS task. (e.g. An RDS running in the same VPC as the
   ECS cluster with a security group allowing access from the ECS task)
3. Atlas and AWS CLIs installed and configured on your machine.

## Storing database credentials in Secrets Manager

In order to run migrations, Atlas needs a connection string to the database. In order to avoid storing the database
credentials in plain text in the ECS task definition, we will use AWS Secrets Manager
to store the database credentials and pass them to the migration container as environment
variables.

Let's start by creating a secret in AWS Secrets Manager that contains the database credentials:

```bash
aws secretsmanager create-secret --name mydb --secret-string 'postgres://user:password@host:port/dbname'
```

The CLI responds with the details about the created secret, which we will use later:

```json
{
    "ARN": "arn:aws:secretsmanager:us-east-1:<account id>:secret:mydb-gxZ0Qe",
    "Name": "mydb",
    "VersionId": "ab6d1fc0-d1a0-49c8-9bfb-5fd9922ffc37"
}
```

To make sure that the ECS task has access to the secrets, we will need to add to the
task's IAM role a policy that allows it to access the secrets. This will look something
similar to:

```json
{
   "Statement": [
      {
         "Action": [
            "secretsmanager:GetSecretValue",
            "secretsmanager:DescribeSecret"
         ],
         "Effect": "Allow",
         "Resource": "arn:aws:secretsmanager:us-east-2:<account id>:secret:mydb-<random suffix>",
         "Sid": ""
      }
   ],
   "Version": "2012-10-17"
}
```

## Reading secrets during deployment

To read our secret value during deployment we can use the [`runtimevar`](https://atlasgo.io/atlas-schema/projects#data-source-runtimevar)
data source. To use this, create a project file named `atlas.hcl`:

```hcl 
data "runtimevar" "url" {
  url = "awssecretsmanager://mydb?region=us-east-2"
}
env "deploy" {
  url = "${data.runtimevar.url}"
}
```

Be sure to replace `mydb` with the name of your secret and to set the correct region in the query parameter.

Next, create a Dockerfile that will include your migration directory and project file. This is a variation
of the baseline example we introduced in the ["Creating container images for migrations"](image.md) guide:

```dockerfile
FROM arigaio/atlas:latest

COPY migrations /migrations

COPY atlas.hcl .
```

This image should be built and pushed to ECR (or another container registry) as part of your CI
process. 

### Running migrations before the application starts

In order to make sure that migrations run successfully before the application starts, we will need to update the
ECS task definition to make the main application container depend on the migration container running to completion.
This way, when you deploy a new version of the application, ECS will first run the migration container and only
start the application container once the migration container exits successfully.

Notice that when running migrations for a distributed application, you will need to make sure that only one
actor in our system tries to run the migrations at any given time to avoid race conditions with unknown
outcomes. Luckily, Atlas supports this behavior out of the box. When running migrations, Atlas will
first acquire a lock in the database (using advisory locking, in databases that support it) and then begin execution.

To achieve this, your task definition should look something similar to: 

```js
{
   "family":"fargate-demo-task-dev",
   "taskRoleArn":"arn:aws:iam::<account id>:role/fargate-demo-ecsTaskRole",
   "executionRoleArn":"arn:aws:iam::<account id>:role/fargate-demo-ecsTaskExecutionRole",
   "networkMode":"awsvpc",
   "requiresCompatibilities":[
      "FARGATE"
   ],
   "cpu":"256",
   "memory":"512",
   "containerDefinitions":[
      {
         "name":"atlas",
         "image":"<account id>.dkr.ecr.us-east-2.amazonaws.com/fargate-demo:v5",
         // highlight-start
         "essential":false,
         "command":[
            "migrate",
            "apply",
            "--env",
            "deploy"
         ]
         // highlight-end
      },
      {
         "name":"fargate-demo-container-dev",
         "image":"nginx:latest",
         "portMappings":[
            {
               "containerPort":80,
               "hostPort":80,
               "protocol":"tcp"
            }
         ],
         "essential":true,
         // highlight-start
         "dependsOn":[
            {
               "containerName":"atlas",
               "condition":"SUCCESS"
            }
         ],
         // highlight-end
      }
   ]
}
```
Notice a few points of interest in the above task definition:
1. We define two containers: one for running Atlas migrations, named "atlas" and one for running the application, "app".
For the sake of the example, our application container is only running the latest version of `nginx`, but in a realistic
  scenario it will contain your application code.
2. The `app` container has a `dependsOn` clause that makes it depend on the `atlas` container. This means that ECS will
  only start the `app` container once the `atlas` container exits successfully.
3. The `atlas` container is not marked as `essential`. This is required for containers that aren't expected to keep 
   running through the task's lifecycle, ideal for use cases like running a setup script before the application starts.
4. The `atlas` container is configured to run the `migrate apply` command. This will run all pending migrations and then exit.
   We provide this command with the `--env deploy` flag to make sure that it uses the `deploy` environment defined 
   in our project file.