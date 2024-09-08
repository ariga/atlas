---
title: Deploying Versioned Migrations to Kubernetes from Atlas Schema Registry
slug: /guides/deploying/k8s-cloud-versioned
---
This guide will walk you through deploying versioned migrations to Kubernetes from Atlas Schema Registry.

Use this setup if: 

* You are using the Atlas Kubernetes Operator with the versioned migrations flow (e.g using `AtlasMigration` CRDs).
* You have a CI/CD pipelines pushing your migration directory to the Atlas Schema Registry.

## Prerequisites

* An Atlas Cloud account with a project on the Atlas Schema Registry
* An Atlas Cloud Bot Token (see [Creating a Bot Token](/cloud/bots#creating)
* A Kubernetes cluster
* Helm and Kubectl installed

## Steps

1. Create a Kubernetes Secret with your Atlas Cloud Bot Token

  ```shell
  kubectl create secret generic atlas-registry-secret --from-literal=token=<your token>
  ```
2. Create a Kubernetes Secret with your database credentials.

  ```shell
  kubectl create secret generic db-credentials --from-literal=url="mysql://root:pass@localhost:3306/myapp"
  ```
  Replace the `url` value with your database credentials.

3. Install the Atlas Operator

  ```shell
  helm install atlas-operator oci://ghcr.io/ariga/charts/atlas-operator
  ```

4. Locate your Cloud project name in the Atlas Schema Registry

  ![Atlas Schema Registry](https://atlasgo.io/uploads/k8sver/cloud-project-name.png)

  Open the Project Information pane on the right and locate the project slug (e.g `project-name`)
  in the URL.

4. Create an file named `migration.yaml` with the following content:

  ```yaml title="migration.yaml"
  apiVersion: db.atlasgo.io/v1alpha1
  kind: AtlasMigration
  metadata:
    name: atlasmigration
  spec:
    urlFrom:
      secretKeyRef:
        key: url
        name: db-credentials
    cloud:
      tokenFrom:
        secretKeyRef:
          key: token
          name: atlas-registry-secret
    dir:
      remote:
        name: "project-name" # Migration directory name in your atlas cloud project
        tag: "latest"
  ```
  Replace `project-name` with the name of your migration directory in the Atlas Schema Registry.
  
  If you would like to deploy a specific version of the migrations, replace `latest` with the version tag.
  
5. Apply the AtlasMigration CRD manifest

  ```shell
  kubectl apply -f migration.yaml
  ```

6. Check the status of the AtlasMigration CRD:

  ```shell
  kubectl get atlasmigration
  ```

  `kubectl` will output the status of the migration:

  ```
  NAME             READY   REASON
  atlasmigration   True    Applied
  ```

7. Observe the reported migration logs on your Cloud project in the Atlas Schema Registry:

   ![Atlas Schema Registry](https://atlasgo.io/uploads/k8sver/k8s-cloud-logs.png)