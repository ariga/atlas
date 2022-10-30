---
id: deploy
slug: /versioned/deploy
title: Deploying schema changes to production
---

## Schema changes as Deployments 

Changes to database schemas  rarely happen in isolation. Most commonly, changes to the database 
schema are related to some change in the application code. Because incompatibility between
the database schema and the application can cause serious problems it is advised to give careful thought
to how these changes are rolled out.

Based on our  experience, we have come to the conclusion that changes
to the database schema should be thought of part of the  _deployment_ sequence, alongside
changing the application version, provisioning infrastructure or applying
configuration changes. 

This guide describes some strategies teams can employ to incorporate schema
changes into their deployment sequence. 

### Running migrations on server initialization

In many cases, we have seen teams that run schema migration logic as part
of the application code: when servers start, before listening for traffic, 
code that ensures that the database schema is up-to-date is invoked. 
This is especially common for teams using ORMs that support an "auto-migration"
flow. 

In our experience, this strategy may work for simple use-cases, but may
cause issues in larger, more established projects. Some downsides of running
migrations on boot are: 
* If multiple replicas of the server code are deployed concurrently
  to avoid dangerous race conditions, some form of synchronization must be
  employed to make sure only one instance tries to run the migration.
* If migrations fail, the server crashes, often entering a crash-loop,
  which may reduce the overall capacity of the system to handle traffic.
* If migrations are driven by a dedicated tool (such as Atlas, Liquibase, Flyway, etc.)
  the tool needs to be packaged into the same deployment artifact. This is both
  cumbersome to invoke and goes against security best practices to reduce attack surface
  by including only the bare minimum into runtime containers. 

### Running migrations as part of deployment pipelines

Instead of running migrations on server init, we suggest using a deployment
strategy that follows these principles: 

1. Schema migrations are deployed as a discrete step in the deployment pipeline
   preceding application version changes.
2. If a migration fails, the whole deployment pipeline should halt.
3. Measures should be taken to ensure that only one instance of the migration
   script runs concurrently. 

## An example setup

To demonstrate these principles, we will now share how schema changes are deployment
as part of [Ariga's](https://ariga.io) deployment pipeline. This setup consists of the following
steps:

1. Build a `migrations` docker image per version
2. Apply migrations as a Helm pre-upgrade hook 

### Packaging migrations to a dedicated container image

As part of our build process we create a dedicated Docker image that
includes Atlas and all relevant migrations files. To do so, we start
with the official [Atlas Docker image](https://hub.docker.com/r/arigaio/atlas)
and add the migration files to it. The final `Dockerfile` looks something
like: 

```dockerfile title=Dockerfile
FROM arigaio/atlas:latest

COPY service/ent/migrations /src/
```

Since we use GitHub Actions heavily for our continuous integration pipe, we added 
a workflow that runs on each push to our mainline branch to build a docker image
and push it to the GitHub container repo (ghcr.io):

```yaml title=.github/workflows/push-docker.yaml
name: Push Docker
on:
  push:
    branches:
      - master
  workflow_dispatch:
jobs:
  docker-push:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v2
      - name: Login to GitHub Container Registry
        uses: docker/login-action@v2
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - name: Build and push
        uses: docker/build-push-action@v3
        with:
          push: true
          file: ${{ matrix.file }}
          tags: ghcr.io/ariga/<repo name>:${{ github.sha }}
          cache-from: type=gha
          cache-to: type=gha,mode=max
```

### Apply migrations as a Helm pre-upgrade hook

At Ariga, we use [Helm](https://helm.sh/) to package our applications
and deploy them to our [Kubernetes](https://kubernetes.io) cluster. 

To satisfy the principle of having migrations run _before_ the new application
version starts as well as ensure that only one migration job runs concurrently,
we use Helm's [pre-upgrade hooks](https://helm.sh/docs/topics/charts_hooks/) feature.

Helm pre-upgrade hooks are chart hooks that:
> Executes on an upgrade request after templates are rendered, but before any resources are updated

To use a pre-upgrade hook to run migrations with Atlas as part of our chart definition
we include a file similar to this:

```helm
apiVersion: batch/v1
kind: Job
metadata:
  # job name should include a unix timestamp to make sure it's unique
  name: "{{ .Release.Name }}-migrate-{{ now | unixEpoch }}"
  labels:
    helm.sh/chart: "{{ .Chart.Name }}-{{ .Chart.Version }}"
  // highlight-start
  annotations:
    "helm.sh/hook": pre-install,pre-upgrade
    "helm.sh/hook-delete-policy": before-hook-creation,hook-succeeded
  // highlight-end
spec:
  template:
    metadata:
      name: "{{ .Release.Name }}-create-tables"
      labels:
        app.kubernetes.io/managed-by: {{ .Release.Service | quote }}
        app.kubernetes.io/instance: {{ .Release.Name | quote }}
        helm.sh/chart: "{{ .Chart.Name }}-{{ .Chart.Version }}"
    spec:
      restartPolicy: Never
      imagePullSecrets:
        - name: {{ .Values.imagePullSecret }}
      containers:
        - name: atlas-migrate
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
          args:
            - migrate
            - apply
            - -u
            - {{ .Values.dburl }}
            - --dir
            - file:///src/
```
Be sure to pass the following [values](https://helm.sh/docs/chart_template_guide/values_files/):

* `imagePullSecret` - secret containing credentials to a 
 [private repository](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/)
  if you are hosting on ghcr.io, see [this guide](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/).
* `image.repository`: the container repository where you pushed your migration image to.
* `image.tag`: the tag of the latest migration image.
* `dburl` the [/concepts/url](URL) of the database which you want to apply migrations to.

Notice the `annotations` block at the top of the file. This block contains two important
attributes:
* `"helm.sh/hook": pre-install,pre-upgrade`: configures this job to run as a pre-install
  hook and as a pre-upgrade hook. 
* `"helm.sh/hook-delete-policy": before-hook-creation,hook-succeeded`, sets the following
  deletion behavior for the jobs created by the hook:
  * `before-hook-creation`: Delete the previous resource before a new hook is launched (default)
  * `hook-succeeded`: Delete the resource after the hook is successfully executed.
  This combination ensures that on the happy path jobs are cleaned after finishing and that 
  in case a job fails it remains on the cluster for its operators to debug. In addition, it 
  ensures that when you retry a job, its past invocations are also cleaned up. 