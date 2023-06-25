---
id: atlas with argo cd
title: Simplifying Database Management with Atlas Kubernetes Operator and Argo CD
---
# Simplifying Database Management with Atlas Kubernetes Operator and Argo CD
GitOps, powered by Argo CD, simplifies and automates application deployment on Kubernetes. Using Git as the source of truth ensures consistent and traceable deployments.

Integrating Ariga's Atlas Operator enhances this process by seamlessly managing database schemas within the Kubernetes environment. This guide explores the integration of GitOps with Argo CD and the Atlas Operator, streamlining application deployment and database management.

By adopting this approach, teams can achieve efficient and error-free deployments, maintaining a synchronized application and database environment.

## Getting to Know the Key Players

Before delving into the integration process, let's familiarise ourselves with the main components involved in this solution:

1\. **Atlas Kubernetes Operator by Ariga:** This Kubernetes controller utilises Atlas to manage your database schema. It enables you to define and apply your preferred schema to your database via the Kubernetes API.

2\. **Argo CD:** A Kubernetes continuous delivery tool, Argo CD facilitates GitOps through a declarative approach.

## Install ArgoCd

To install Argo run the following command:

```bash
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
```

Wait till all the pods in the argocd namespace are running

```bash
➜ kubectl get pods -n argocd
NAME                                                READY   STATUS    RESTARTS   AGE
argocd-application-controller-0                     1/1     Running   0          57s
argocd-applicationset-controller-79d54f5d64-lvmxc   1/1     Running   0          57s
argocd-dex-server-7f4b99d696-mj67b                  1/1     Running   0          57s
argocd-notifications-controller-cdd87f4f5-ljbg2     1/1     Running   0          57s
argocd-redis-74d77964b-5rzb9                        1/1     Running   0          57s
argocd-repo-server-5589688987-6xmng                 1/1     Running   0          57s
argocd-server-688b758f7d-trc6s                      1/1     Running   0          57s
```

For more information or if you run into some error refer to the [Argo CD Documentation](https://argo-cd.readthedocs.io/en/stable/getting_started/).

## Install Atlas Operator

```bash
helm install atlas-operator oci://ghcr.io/ariga/charts/atlas-operator
```

Wait till the atlas-operator pod is running.

```bash
➜ kubectl get pods
NAME                              READY   STATUS    RESTARTS   AGE
atlas-operator-7449896b68-nzfwq   1/1     Running   0          21s
```

For more information on the installation process refer to the [Atlas Operator Documentation](https://atlasgo.io/integrations/kubernetes/operator#getting-started)

## The basic outline of the project

In our application architecture, we have a database that is connected to our application and managed using Atlas CR (Custom Resource). The database plays a crucial role in storing and retrieving data for the application, while the Atlas CR provides seamless integration and management of the database schema within our Kubernetes environment.

![outline of the project](app.png)

Set up a Git repository to serve as the central storage for all your configuration files. In this example, we’re using the [https://github.com/pratikjagrut/atlas-argocd-demo.git](https://github.com/pratikjagrut/atlas-argocd-demo.git) repository, and the configuration files for the DB, Atlas CR and Nginx are in the ***configs*** sub-directory.

## The challenge in accomplishing deployment flow

Integrating GitOps practices with a database in our application stack poses a unique challenge. 

Argo CD provides a declarative approach to GitOps, allowing us to define an Argo CD application and seamlessly handle the synchronization process. By pushing changes to the database schema or application code to the Git repository, Argo CD automatically syncs those changes to the Kubernetes cluster.

However, ensuring the proper order of deployments is critical. In our scenario, the database deployment must succeed before rolling out the application to ensure its functionality. If the database deployment encounters an issue, it is essential to address it before proceeding with the application deployment. The sync wave functionality of Argo CD enables us to achieve this specific deployment order and streamline the GitOps process.

## SyncWaves

Argo CD provides SyncWaves and Synchooks as features that help to control the order in which manifests are applied within an application. By using annotations with specific order numbers, you can determine the sequence of manifest applications. Lower numbers indicate the earlier application and negative numbers are also allowed.

The diagram shows our application deployment order.

![Flow of the project](app1.png)

To achieve the above order we’ll annotate the configs of DB, Alas CR and Nginx as follows

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

Now, with the implementation of the sync wave annotation, we can ensure that the database is deployed first. Only if the database is in a healthy state, the application will be rolled out. However, there is a potential issue if we make mistakes in the schema and the Atlas object becomes unhealthy. In the current setup, the application will still be rolled out because Argo CD does not consider the health of the custom resource.

## Health Check for Atlas objects

Argo CD has built-in health assessment for standard Kubernetes types, such as deployments and replicaSets, it does not have a built-in health check for the Atlas custom resource. However, Argo CD supports custom health checks written in LUA, allowing us to define our custom health assessment logic for the Atlas custom resource.

To define the custom health logic for the Atlas object in Argo CD, we can add the custom health check configuration to the ***argocd-cm*** ConfigMap. Below is a custom health check for the Atlas object:

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

![Argo Dashboard](dashboard.png)