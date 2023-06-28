---
id: k8s-argo
title: Deploying to Kubernetes with the Atlas Operator and Argo CD
slug: /guides/deploying/k8s-argo
---

[GitOps](https://www.gitops.tech/) is a software development and deployment methodology that uses Git as the central repository
for both code and infrastructure configurations, enabling automated and auditable deployments.

[ArgoCD](https://argoproj.github.io/cd/) is a Kubernetes-native continuous delivery tool that implements GitOps principles.
It uses a declarative approach to deploy applications to Kubernetes, ensuring that the desired state of the 
application is always maintained.

[Kubernetes Operators](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/) are software extensions to 
Kubernetes that enable the automation and management of complex, application-specific operational tasks and 
domain-specific knowledge within a Kubernetes cluster.

In this guide, we will demonstrate how to use the [Atlas Kubernetes Operator](/integrations/kubernetes/operator) and 
ArgoCD to achieve a GitOps-based deployment workflow for your database schema.

## Pre-requisites

* A running Kubernetes cluster

For learning purposes, you can use [Minikube](https://minikube.sigs.k8s.io/docs/start/), which is a tool that runs a
single-node Kubernetes cluster inside a VM on your laptop.

## Installation 

### Install ArgoCD

To install ArgoCD run the following commands:

```bash
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
```

Wait until all the pods in the `argocd` namespace are running:

```bash
kubectl wait --for=condition=ready pod --all -n argocd
```

`kubectl` will print something like this:

```bash
pod/argocd-application-controller-0 condition met
pod/argocd-applicationset-controller-69dbc8585c-6qbwr condition met
pod/argocd-dex-server-59f89468dc-xl7rg condition met
pod/argocd-notifications-controller-55565589db-gnjdh condition met
pod/argocd-redis-74cb89f466-gzk4f condition met
pod/argocd-repo-server-68444f6479-mn5gl condition met
pod/argocd-server-579f659dd5-5djb5 condition met
```

For more information or if you run into some error refer to the 
[Argo CD Documentation](https://argo-cd.readthedocs.io/en/stable/getting_started/).

### Install the Atlas Operator

```bash
helm install atlas-operator oci://ghcr.io/ariga/charts/atlas-operator
```

Helm will print something like this:

```bash
Pulled: ghcr.io/ariga/charts/atlas-operator:0.1.9
Digest: sha256:4dfed310f0197827b330d2961794e7fc221aa1da1d1b95736dde65c090e6c714
NAME: atlas-operator
LAST DEPLOYED: Tue Jun 27 16:58:30 2023
NAMESPACE: default
STATUS: deployed
REVISION: 1
TEST SUITE: None
```

Wait until the atlas-operator pod is running:

```bash
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=atlas-operator -n default
```

`kubectl` will print something like this:

```bash
pod/atlas-operator-866dfbc56d-qkkkn condition met
```

For more information on the installation process refer to the [Atlas Operator Documentation](/integrations/kubernetes/operator#getting-started)

## High-level architecture

Before we dive into the details of the deployment flow, let’s take a look at the high-level architecture of our application.

![Application Architecture](https://atlasgo.io/uploads/k8s/argocd/app-diagram.png)

On a high level, our application consists of the following components:

1. A backend application - in our example we will use a plain NGINX server
   as a placeholder for a real backend application.
2. A database - in our example we will use a MySQL Pod for the database.
   In a more realistic scenario, you might want to use a managed database service like AWS RDS or GCP Cloud SQL.
3. An `AtlasSchema`  Custom Resource that defines the database schema and is managed by the Atlas Operator.

In our application architecture, we have a database that is connected to our application and managed using 
Atlas CR (Custom Resource). The database plays a crucial role in storing and retrieving data for the application,
while the Atlas CR provides seamless integration and management of the database schema within our Kubernetes environment.


Set up a Git repository to serve as the central storage for all your configuration files. In this example, we’re using the [https://github.com/pratikjagrut/atlas-argocd-demo.git](https://github.com/pratikjagrut/atlas-argocd-demo.git) repository, and the configuration files for the DB, Atlas CR and Nginx are in the ***configs*** sub-directory.

## Incorporating schema changes into the GitOps workflow

Integrating GitOps practices with a database in our application stack poses a unique challenge. 

Argo CD provides a declarative approach to GitOps, allowing us to define an Argo CD application
and seamlessly handle the synchronization process. By pushing changes to the database schema or 
application code to the Git repository, Argo CD automatically syncs those changes to the Kubernetes cluster.

However, as we discussed in the [introduction](/guides/deploying/intro#running-migrations-as-part-of-deployment-pipelines),
ensuring the proper order of deployments is critical. In our scenario, the database deployment
must succeed before rolling out the application to ensure its functionality. If the database deployment
encounters an issue, it is essential to address it before proceeding with the application deployment. 

In the next section, we will see how the SyncWave functionality of Argo CD enables us to achieve
this specific deployment order and streamline the GitOps process.

## Sync Waves
   
Argo CD provides Sync Waves and Sync Hooks as features that help to control the order in which 
manifests are applied within an application. By using annotations with specific order numbers,
you can determine the sequence of manifest applications. Lower numbers indicate the earlier 
application and negative numbers are also allowed.

The diagram shows our application deployment order.

![Application Architecture](https://atlasgo.io/uploads/k8s/argocd/deployment-flow.png)

To achieve the above order we'll annotate each resource with a sync wave annotation order number:

```yaml
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "<order-number>"
```

For more information refer to the [official documentation](https://argo-cd.readthedocs.io/en/stable/user-guide/sync-waves/).

### **Create database manifest**

In your repository, create a manifest file that includes the service and deployment configurations for the database. Annotate these configurations with a sync wave annotation order number of 0.

***db.yaml***

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "0"
  name: mysql
spec:
  ports:
  - port: 3306
  selector:
    app: mysql
  clusterIP: None
---
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "0"
  name: mysql
spec:
  selector:
    matchLabels:
      app: mysql
  strategy:
    type: Recreate
  template:
    metadata:
      labels:
        app: mysql
    spec:
      containers:
      - image: mysql
        name: mysql
        env:
        - name: MYSQL_ROOT_PASSWORD
          value: pass
        - name: MYSQL_DATABASE
          value: example
        readinessProbe:
            tcpSocket:
              port: 3306
            initialDelaySeconds: 10
            periodSeconds: 10
        livenessProbe:
          tcpSocket:
            port: 3306
          initialDelaySeconds: 15
          periodSeconds: 15
        ports:
        - containerPort: 3306
          name: mysql
```

### Create AtlasSchema CR

Create the AtlasSchema custom resource to define the desired schema for your database, refer to the [Atlas Operator documentation](https://github.com/ariga/atlas-operator/blob/master/charts/atlas-operator/templates/crds/crd.yaml) and determine the specifications, such as the desired database schema, configuration options, and additional parameters.

Here we’re creating a ***users*** table in an ***example*** database and annotating it with a sync wave order number of 1. This annotation informs Argo CD to deploy them after the database has been successfully deployed.

**atlas-cr.yaml**

```yaml
apiVersion: db.atlasgo.io/v1alpha1
kind: AtlasSchema
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "1"
  name: myapp
spec:
  url: mysql://root:pass@mysql:3306/example
  schema:
    sql: |
      create table users (
        id int not null auto_increment,
        name varchar(255) not null,
        email varchar(255) unique not null,
        short_bio varchar(255) not null,
        primary key (id)
      );
```

### Now create a deployment of your backend application

To simulate the process, we are using an Nginx server instead of a real backend server. Annotate the backend deployment with a sync wave order number of 2. This tells Argo CD to deploy the backend application after the Atlas CR is deployed and confirmed to be in good health.

***nginx-dep.yaml***

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "2"
  name: nginx
spec:
  selector:
    matchLabels:
      app: nginx
  replicas: 2
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx
        ports:
        - containerPort: 80
```

## Health Check for Atlas objects

To decide whether a SyncWave is complete and the next SyncWave can be started, Argo CD performs 
a health check on the resources in the current SyncWave. If the health check fails, Argo CD will 
not proceed with the next SyncWave.

Argo CD has built-in health assessment for standard Kubernetes types, such as `Deployment` and `ReplicaSet`,
but it does not have a built-in health check for custom resources such as `AtlasSchema`. 

To bridge this gap, Argo CD supports custom health checks written in [Lua](https://lua.org), 
allowing us to define our custom health assessment logic for the Atlas custom resource.

To define the custom logic for the Atlas object in Argo CD, we can add 
the custom health check configuration to the ***argocd-cm*** ConfigMap. Below is a custom 
health check for the Atlas object:

***argocd-cm.yaml***

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: argocd-cm
  namespace: argocd
  labels:
    app.kubernetes.io/name: argocd-cm
    app.kubernetes.io/part-of: argocd
data:
  resource.customizations: |
    db.atlasgo.io/AtlasSchema:
      health.lua: |
        hs = {}
        if obj.status ~= nil then
          if obj.status.conditions ~= nil then
            for i, condition in ipairs(obj.status.conditions) do
              if condition.type == "Ready" and condition.status == "False" then
                hs.status = "Degraded"
                hs.message = condition.message
                return hs
              end
              if condition.type == "Ready" and condition.status == "True" then
                hs.status = "Healthy"
                hs.message = condition.message
                return hs
              end
            end
          end
        end

        hs.status = "Progressing"
        hs.message = "Waiting for reconciliation"
        return hs
```

```bash
➜ kubectl apply -f argocd-cm.yaml
configmap/argocd-cm created
```

## Create Argo CD Application

In your Git repository, create an Argo CD ***Application.yaml*** and define the argo cd application properties within it:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: atlas-argocd-demo
  namespace: argocd
spec:
  source:
    path: configs
    repoURL: 'https://github.com/pratikjagrut/atlas-argocd-demo'
    targetRevision: main
  destination:
    namespace: default
    server: 'https://kubernetes.default.svc'
  project: default
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    retry:
      limit: 5
      backoff:
        duration: 5s
        maxDuration: 3m0s
        factor: 2
    syncOptions:
      - CreateNamespace=true
```

Replace ***atlas-argocd-demo*** with your preferred application name, and ***'***[***https://github.com/pratikjagrut/atlas-argocd-demo***](https://github.com/pratikjagrut/atlas-argocd-demo)***'*** with the URL of your Git repository and ***path: configs*** where the configuration files reside, if they’re in the root of the repo then ***path: ./***

The current structure of this example looks like this:

```bash
➜  atlas-argocd-demo git:(main) tree
.
├── Application.yaml
├── README.md
├── argocd-cm.yaml
└── configs
    ├── atlas-cr.yaml
    ├── db.yaml
    └── nginx-dep.yaml

1 directory, 6 files
```

Push these files to your Git repository.

Now apply **Application.yaml**.

```bash
➜ kubectl apply -f Application.yaml
application.argoproj.io/atlas-argocd-demo created
```

Once you create an Argo CD application, it automatically pulls the configuration files from your Git repository and applies them to your Kubernetes cluster. As a result, the corresponding resources are created based on the configurations. This streamlined process ensures that the desired state of your application is synchronised with the actual state in the cluster.

Verify if the application is successfully deployed and the resources are healthy.  

```bash
➜ kubectl get -n argocd applications.argoproj.io atlas-argocd-demo -o=jsonpath='{range .status.resources[*]}{"\n"}{.kind}: {"\t"} {.name} {"\t"} ({.status}) {"\t"} ({.health}){end}'

Service: 	     mysql 	 (Synced) 	 ({"status":"Healthy"})
Deployment: 	 mysql 	 (Synced) 	 ({"status":"Healthy"})
Deployment: 	 nginx 	 (Synced) 	 ({"status":"Healthy"})
AtlasSchema: 	 myapp 	 (Synced) 	 ({"message":"The schema has been applied successfully. Apply response: {\"Changes\":{}}","status":"Healthy"})%
```

Here we can see the health and dependency structure of all the deployments.

(dashboard)