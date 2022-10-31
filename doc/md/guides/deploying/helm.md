---
id: helm
slug: /guides/deploying/helm
title: Deploying schema migrations to Kubernetes with Helm
---

[Helm](https://helm.sh) is a popular package manager for Kubernetes that allows
developers to package applications into distributable modules called 
[Charts](https://helm.sh/docs/intro/using_helm/#three-big-concepts) that can be
installed, upgraded, uninstalled, and more against a Kubernetes cluster.

Helm is commonly used by software projects as a means for distributing software
in a way that will be simple for developers to manage on their clusters. For example,
[Bitnami](https://bitnami.com/) maintains [hundreds of charts](https://bitnami.com/stacks/helm)
for easily installing many popular applications, such as [MySQL](https://bitnami.com/stack/mysql/helm),
[Apache Kafka](https://bitnami.com/stack/kafka/helm) and others on Kubernetes. 

In addition, many teams ([Ariga](https://github.com/ariga) among them) use Helm
as a way to package internal applications for deployment on Kubernetes. 

In this guide, we demonstrate how schema migrations can be integrated into
Helm charts in such a way that satisfies the principles for deploying
schema migrations which we described in the [introduction](/guides/deploying/intro).

Prerequisites to the guide:
1. [A migrations docker image](/guides/deploying/image) 
2. [A Helm chart](https://helm.sh/docs/chart_template_guide/getting_started/) defining
 your application. 

## Using Helm lifecycle hooks

To satisfy the principle of having migrations run _before_ the new application
version starts, as well as ensure that only one migration job runs concurrently,
we use Helm's [pre-upgrade hooks](https://helm.sh/docs/topics/charts_hooks/) feature.

Helm pre-upgrade hooks are chart hooks that:
> Executes on an upgrade request after templates are rendered, but before any resources are updated

To use a pre-upgrade hook to run migrations with Atlas as part of our chart definition,
we create a template for a [Kubernetes Job](https://kubernetes.io/docs/concepts/workloads/controllers/job/)
and annotate it with the relevant [Helm hook annotations](https://helm.sh/docs/topics/charts_hooks/#the-available-hooks).

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
 [private repository](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/).
  If you are hosting on ghcr.io, see [this guide](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/).
* `image.repository`: the container repository where you pushed your migration image to.
* `image.tag`: the tag of the latest migration image.
* `dburl`: the [URL](/concepts/url) of the database which you want to apply migrations to.

Notice the `annotations` block at the top of the file. This block contains two important
attributes:
1. `"helm.sh/hook": pre-install,pre-upgrade`: configures this job to run as a pre-install
  hook and as a pre-upgrade hook. 
2. `"helm.sh/hook-delete-policy": before-hook-creation,hook-succeeded`: sets the following
  deletion behavior for the jobs created by the hook:
  * `before-hook-creation`: Delete the previous resource before a new hook is launched (default)
  * `hook-succeeded`: Delete the resource after the hook is successfully executed.
  This combination ensures that on the happy path jobs are cleaned after finishing and that 
  in case a job fails, it remains on the cluster for its operators to debug. In addition, it 
  ensures that when you retry a job, its past invocations are also cleaned up. 