---
id: k8s-init-containers
slug: /guides/deploying/k8s-init-container
title: Deploying schema migrations to Kubernetes with Init Containers
---

In [Kubernetes](https://kubernetes.io), [Init Containers](https://kubernetes.io/docs/concepts/workloads/pods/init-containers/)
are specialized containers that run before app containers in a Pod. Init containers
can contain utilities or setup scripts not present in an app image. 

Init containers can be utilized to run schema migrations with Atlas before the
application loads. Because init containers can use a container image different
from the application, developers can use a [purpose-built image](image.md) that
only contains Atlas and the migration scripts to run them.  This way, less 
can be included in the application runtime environment, which reduces
the attack surface from a security perspective. 

Depending on an application's [deployment strategy](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#strategy),
multiple replicas of an init container may run concurrently.  In the case of
schema migrations, this can cause a dangerous race condition with unknown outcomes.
To prevent this, in databases that support advisory locking, Atlas will acquire
a lock on the migration operation before running migrations, making the
operation mutually exclusive.

In this guide, we demonstrate how schema migrations can be integrated into
a Kubernetes deployment using an init container.

Prerequisites to the guide:
1. [A migrations docker image](/guides/deploying/image)
2. [A Kubernetes Deployment manifest](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/) defining
   your application.
3. A running Kubernetes cluster to work against.

## Adding an init container

Suppose our deployment manifest looks similar to this:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.14.2
        ports:
        - containerPort: 80
```

Now, let's say our [migration container image](image.md) which contains the Atlas binary
and our migration scripts is available at `ghcr.io/repo/migrations:v0.1.2`. We would like
to run `migrate apply` against our target database residing at `mysql://root:s3cr37p455@dbhostname.io:3306/db`. 

We will use a Kubernetes [Secret](https://kubernetes.io/docs/concepts/configuration/secret/) to
store a [config file](/atlas-schema/projects) containing the credentials to our database. Create the file:

```hcl title=atlas.hcl
env "k8s" {
  url = "mysql://root:s3cr37p455@dbhostname.io:3306/db"
}
```

Kubernetes accepts secrets encoded as base64 strings. Let's calculate the
base64 string representing our project file:

```text
cat atlas.hcl | base64
```
Copy the result:
```text
ZW52ICJrOHMiIHsKICB1cmwgPSAibXlzcWw6Ly9yb290OnMzY3IzN3A0NTVAZGJob3N0bmFtZS5pbzozMzA2L2RiIgp9Cg==
```

Create the secret manifest:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: atlas-project
type: Opaque
data:
  atlas.hcl: ZW52ICJrOHMiIHsKICB1cmwgPSAibXlzcWw6Ly9yb290OnMzY3IzN3A0NTVAZGJob3N0bmFtZS5pbzozMzA2L2RiIgp9Cg==
```

Apply the secret on the cluster:

```yaml
kubectl apply -f secret.yaml
```

The secret is created:
```text
secret/atlas-project created
```

Next, add a volume to mount the config file and an init container using it to
the deployment manifest: 

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      // highlight-start
      volumes:
      - name: atlas-project
        secret:
          secretName: atlas-project
      initContainers:
      - name: migrate
        image: ghcr.io/repo/migrations:v0.1.2
        imagePullPolicy: Always
        args: ["migrate", "apply", "-c", "file:///etc/atlas/atlas.hcl", "--env", "k8s"]
        volumeMounts:
        - name: atlas-project
          mountPath: "/etc/atlas"
      // highlight-end
      containers:
      - name: nginx
        image: nginx:1.14.2
        ports:
        - containerPort: 80
```

Notice the new configuration blocks we added to our deployment manifest:
* We added our secret `atlas-project` [as a volume](https://kubernetes.io/docs/tasks/configure-pod-container/configure-volume-storage/#configure-a-volume-for-a-pod) to the
  the deployment's PodSpec.
* We add an `initContainer` named `migrate` that runs the `ghcr.io/repo/migrations:v0.1.2` image.
* We mounted the `atlas-project` volume at `/etc/atlas` in our init container.
* We configured our init container to run with these flags: `["migrate", "apply", "-c", "file:///etc/atlas/atlas.hcl", "--env", "k8s"]`

## Wrapping up

That's it! After we apply our new deployment manifest, Kubernetes will first run 
the init container and only then run the application containers.  