---
id: cloud-sql-via-github-actions
slug: /guides/deploying/cloud-sql-via-github-actions
title: Deploying schema migrations to Google CloudSQL using Atlas
---

## In this article
* [Overview](#overview)
* [What is Cloud SQL?](#what-is-cloud-sql)
* [What is Cloud SQL Auth Proxy?](#what-is-cloud-sql-auth-proxy)
* [What is GitHub Actions?](#what-is-github-actions])
* [Deploying Schema Migrations to Cloud SQL](#deploying-schema-migrations-to-cloud-sql)
* [Prerequisites](#prerequisites)
* [Step-by-Step](#step-by-step)
   1. [Authenticate to Google Cloud](#step-by-step)
   2. [Retrieve your instance connection name](#retrieve-your-instance-connection-name)
   3. [Store your password in GitHub Secrets](#store-your-password-in-github-secrets)
   4. [Setup GitHub Actions](#setup-github-actions)
   5. [Execute your GitHub Actions Workflow](#execute-your-github-actions-workflow)
* [Wrapping Up](#wrapping-up)

## Overview

In this guide, we demonstrate how to handle database schema changes when working with Cloud SQL. Within the framework of this topic, we are going to introduce how to set up a GitHub Actions workflow to automatically deploy database schema changes to a Cloud SQL instance. This approach is meant to enhance automation, version control, CI/CD, DevOps practices, and scalability, contributing to more efficient and reliable database management.

Before diving into the practical implementation, let's first look at some of the underlying technologies that we will be working with.

## What is Cloud SQL?
Cloud SQL is a fully-managed database service that makes it easy to set up, maintain, manage, and administer your relational databases in the cloud. With Cloud SQL, you can deploy your databases in a highly available and scalable manner, with automatic failover and load balancing, so that your applications can handle a large number of concurrent requests and traffic spikes. You can also choose from different machine types and storage sizes to meet your specific performance and storage requirements.

## What is Cloud SQL Auth Proxy?
The Cloud SQL Auth Proxy is a utility for ensuring simple, secure connections to your Cloud SQL instances. It provides a convenient way to control access to your database using Identity and Access Management (IAM) permissions while ensuring a secure connection to your Cloud SQL instance. Like most proxy tools, it serves as the intermediary authority on connection authorizations. Using the Cloud SQL Auth proxy is the recommended method for connecting to a Cloud SQL instance.

## What is GitHub Actions?
GitHub Actions is a continuous integration and continuous delivery (CI/CD) platform that allows you to automate your build, test, and deployment pipeline. You can create workflows that build and test every pull request to your repository, or deploy merged pull requests to production. GitHub Actions goes beyond just DevOps and lets you run workflows when other events happen in your repository. For example, in this guide, you will run a workflow to automatically deploy migrations to a Cloud SQL database whenever someone pushes changes to the main branch in your repository.

## Deploying Schema Migrations to Cloud SQL

### Prerequisites

Prerequisites to the guide:

1. You will need to have the GCP **Project Editor** role. This role grants you full read and write access to resources within your project.
2. Google Cloud SDK installed on your workstation. If you have not installed the SDK, you can find [instructions for installing the SDK from the official documentation](https://cloud.google.com/sdk/docs/install/).
3. A running Cloud SQL instance to work against. If you have not created the instance yet, see [Creating instances at cloud.google.com](https://cloud.google.com/sql/docs/postgres/create-instance).
4. A GitHub repository to create and run a GitHub Actions workflow.

### Step-by-Step
#### 1—Authenticate to Google Cloud
There are two approaches to authenticating with Google Cloud: Authentication via a Google Cloud Service Account Key JSON or authentication via [Workload Identity Federation](https://cloud.google.com/iam/docs/workload-identity-federation).

**Setup Workload Identity Federation** 
Identity federation allows you to grant applications running outside Google Cloud access to Google Cloud resources, without using Service Account Keys. It is recommended over Service Account Keys as it eliminates the maintenance and security burden associated with service account keys and also establishes a trust delegation relationship between a particular GitHub Actions workflow invocation and permissions on Google Cloud.

For authenticating via Workload Identity Federation, you must create and configure a Google Cloud Workload Identity Provider. A Workload Identity Provider is an entity that describes a relationship between Google Cloud and an external identity provider, such as GitHub, AWS, Azure Active Directory, etc.

To create and configure a Workload Identity Provider:

1. Save your project ID as an environment variable. The rest of these steps assume this environment variable is set:

```bash
$ export PROJECT_ID="my-project" # update with your value
```
2. Create a Google Cloud Service Account. If you already have a Service Account, take note of the email address and skip this step.

```bash
$ gcloud iam service-accounts create "my-service-account" \
  --project "${PROJECT_ID}"
```

3. Enable the IAM Credentials API:

```bash
$ gcloud services enable iamcredentials.googleapis.com
```

4. Grant the Google Cloud Service Account permissions to edit Cloud SQL resources.

```bash
$ gcloud projects add-iam-policy-binding [PROJECT_NAME] \
--member serviceAccount:[SERVICE_ACCOUNT_EMAIL] \
--role roles/editor
```

Replace **[PROJECT_NAME]** with the name of your project, and **[SERVICE_ACCOUNT_EMAIL]** with the email address of the service account you want to grant access to.

5. Create a new workload identity pool:

```bash
$ gcloud iam workload-identity-pools create "my-pool" \
  --project="${PROJECT_ID}" \
  --location="global" \
  --display-name="My pool"
```

6. Get the full ID of the Workload Identity Pool:

```bash
$ gcloud iam workload-identity-pools describe "my-pool" \
  --project="${PROJECT_ID}" \
  --location="global" \
  --format="value(name)"
```

Save this value as an environment variable:

```bash
$ export WORKLOAD_IDENTITY_POOL_ID="..." # value from above

# This should look like:
#
#   projects/123456789/locations/global/workloadIdentityPools/my-pool
#
```

7. Create a Workload Identity Provider in that pool:

```bash
$ gcloud iam workload-identity-pools providers create-oidc "my-provider" \
  --project="${PROJECT_ID}" \
  --location="global" \
  --workload-identity-pool="my-pool" \
  --display-name="GitHub provider" \
 --attribute-mapping="google.subject=assertion.sub,attribute.actor=assertion.actor,attribute.repository=assertion.repository" \
  --issuer-uri="https://token.actions.githubusercontent.com"
```

8. Allow authentications from the Workload Identity Provider originating from your repository to impersonate the Service Account created above:

```bash
# Update this value to your GitHub repository.

$ export REPO="username/repo_name" # e.g. "ariga/atlas"

$ gcloud iam service-accounts add-iam-policy-binding "my-service-account@${PROJECT_ID}.iam.gserviceaccount.com" \
  --project="${PROJECT_ID}" \
  --role="roles/iam.workloadIdentityUser" \
  --member="principalSet://iam.googleapis.com/${WORKLOAD_IDENTITY_POOL_ID}/attribute.repository/${REPO}"
```

Note that **$WORKLOAD_IDENTITY_POOL_ID** should be the full Workload Identity Pool resource ID, like:

**projects/123456789/locations/global/workloadIdentityPools/my-pool**

9. Extract the Workload Identity Provider resource name:

```bash
$ gcloud iam workload-identity-pools providers describe "my-provider" \
  --project="${PROJECT_ID}" \
  --location="global" \
  --workload-identity-pool="my-pool" \
  --format="value(name)"
```

Use this value as the **workload_identity_provider** value in your GitHub Actions YAML.

Using the Workload Identity Provider ID and Service Account email, the GitHub Action will mint a GitHub OIDC token and exchange the GitHub token for a Google Cloud access token.

**Note:** It can take up to **5 minutes** from when you configure the Workload Identity Pool mapping until the permissions are available.

#### 2—Retrieve your Instance Connection Name 
The instance connection name is a connection string that identifies a Cloud SQL instance, and you need this string to establish a connection to your database.  The format of the connection name is **projectID:region:instanceID**.

To retrieve the Cloud SQL instance connection name, run the following command:

```bash
$ gcloud sql instances describe <INSTANCE_NAME> --format='value(connectionName)'
```

For example, if your instance name is **"my-instance"**, you can retrieve its connection name using the following command:

```bash
$ gcloud sql instances describe my-instance --format='value(connectionName)' 
```

#### 3—Store your Password in GitHub Secrets
Secrets are a way to store sensitive information securely in a repository, such as passwords, API keys, and access tokens. To use secrets in your workflow, you must first create the secret in your repository's settings by following these steps:

1. Navigate to your repository on GitHub.
2. Click on the **"Settings"** tab.
3. Click on **"Secrets"** in the left sidebar.
4. Click on **"New repository secret"**.
5. Enter **"DB_PASSWORD"** in the **"Name"** field.
6. Enter the actual password in the **"Value"** field.
7. Click on **"Add secret"**.

Once you have added the secret, you can reference it in your workflow using **`${{ secrets.DB_PASSWORD }}`**. The action will retrieve the actual password value from the secret and use it in the **`DB_PASSWORD`** environment variable during the workflow run.

#### 4—Setup GitHub Actions
Here is an example GitHub Actions workflow for authenticating to GCP with workload identity federation and deploying migrations to a Cloud SQL MySQL database using Cloud SQL Proxy:

```yaml
name: Deploy Migrations

on:
  push:
    branches:
      - main

env:
  PROJECT_ID: my-project-id
  INSTANCE_CONNECTION_NAME: my-instance-connection-name
  DB_HOST: 127.0.0.1
  DB_PORT: 3306
  DB_NAME: my-db-name
  DB_USER: my-db-user
  DB_PASSWORD: ${{ secrets.DB_PASSWORD }}

jobs:
  deploy-migrations:
    runs-on: ubuntu-latest
    permissions:
      contents: 'read'
      id-token: 'write'
    steps:
      - name: Checkout Repository
        uses: actions/checkout@v3

      - name: Download and install Atlas CLI
        run: |
          curl -sSf https://atlasgo.sh | sh -s -- -y

      - name: Download wait-for-it.sh
        run: |
          wget https://raw.githubusercontent.com/vishnubob/wait-for-it/master/wait-for-it.sh
          chmod +x wait-for-it.sh

      - id: 'auth'
        uses: 'google-github-actions/auth@v1'
        with:
          workload_identity_provider: 'projects/123456789/locations/global/workloadIdentityPools/my-pool/providers/my-provider'
          service_account: 'my-service-account@my-project.iam.gserviceaccount.com'

      - name: 'Set up Cloud SDK'
        uses: 'google-github-actions/setup-gcloud@v1'
        with:
          version: '>= 416.0.0'

      - name: Download Cloud SQL Proxy
        run: |
          wget https://dl.google.com/cloudsql/cloud_sql_proxy.linux.amd64 -O cloud_sql_proxy
          chmod +x cloud_sql_proxy

      - name: Start Cloud SQL Proxy
        run: ./cloud_sql_proxy -instances=$INSTANCE_CONNECTION_NAME=tcp:3306 &

      - name: Wait for Cloud SQL Proxy to Start
        run: |
          ./wait-for-it.sh $DB_HOST:$DB_PORT -s -t 10 -- echo "Cloud SQL Proxy is running"

      - name: Deploy Migrations
        run: |
          echo -ne '\n' | atlas migrate apply   --url "mysql://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME"   --dir file://migrations

      - name: Stop Cloud SQL Proxy
        run: kill $(ps aux | grep cloud_sql_proxy | grep -v grep | awk '{print $2}')
```

Note that for this workflow to work, you will need to replace the placeholders in the environment variables with your own values. Your migrations directory should be stored in your repository's root directory.

Here's what this workflow does:

1. Sets the name of the workflow to **"Deploy Migrations"**.
2. Triggers on a push to the **main** branch.
3. Sets the environment variables required for the Cloud SQL instance and the database we want to deploy migrations to.
4. Defines a job named **"deploy-migrations"** that runs on the latest version of Ubuntu.
5. Checkout the code.
6. Downloads and installs the Atlas CLI.
7. Uses the Google Cloud Workload Identity Federation to authenticate with Google Cloud. 
8. Configures the [Google Cloud SDK](https://cloud.google.com/sdk/) in the GitHub Actions environment. 
9. Downloads the Cloud SQL Proxy and makes it executable.
10. Starts the Cloud SQL Proxy, to create a secure tunnel between your GitHub Actions runner and your Cloud SQL instance.
11. Wait for Cloud SQL Proxy to start up before proceeding with the subsequent steps.
12. Deploys all pending migration files in the migration directory on a Cloud SQL database.
13. Stops the Cloud SQL Proxy

#### 5—Execute your GitHub Actions Workflow 
To execute this workflow once you commit to the main branch, follow these steps:

1. Create a new file named **atlas_migrate_db.yml** in the **.github/workflows/** directory of your repository.
2. Add the code block we've just discussed to the **atlas_migrate_db.yml** file.
3. Commit the **atlas_migrate_db.yml** file to your repository's **main** branch.

Now, whenever you push changes to the **main** branch, all pending migrations will be executed. You can monitor the progress of the GitHub Action in the "Actions" tab of your repository.

## Wrapping Up
In this guide, you learned how to deploy schema migrations to Cloud SQL using Atlas, while ensuring secure connections via a Cloud SQL Proxy. With this knowledge, you can leverage the power of Atlas and Cloud SQL to manage your database schema changes with ease and confidence.

In addition to the specific steps outlined in this guide, you also gained valuable experience with various concepts and tools that are widely used in database management, such as GitHub Actions, Cloud SQL, Cloud SQL Proxy, and the Google Cloud SDK. We hope that this guide has been helpful in expanding your knowledge and skills.