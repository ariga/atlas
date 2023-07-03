---
title: "Announcing v0.11.0: Manage database schemas with Kubernetes and Atlas"
authors: rotemtam
tags: [schema, migration, kubernetes, operator]
---
:::info TL;DR

You can now use the Atlas Kubernetes Operator to safely manage your database schemas with Atlas from
within your Kubernetes cluster.

[See an example](#demo-time)

:::

## Introduction

Today, we are excited to announce the release of [Atlas v0.11.0](https://github.com/ariga/atlas/releases/tag/v0.11.0),
which introduces the [Atlas Kubernetes Operator](/integrations/kubernetes/operator). This release is a major
milestone in our mission to make Atlas the most robust and modern way to manage your database schemas. With the Atlas Kubernetes
Operator, you can now manage your database schemas with Atlas from within your Kubernetes cluster.

In this release, we also introduce a new concept to Atlas -
["Diff Policies"](https://atlasgo.io/declarative/apply#diff-policy) - which allow you to customize the
way Atlas plans database migrations for you. This concept is directly related to the Kubernetes Operator, and we will explain how
below.

### What are Kubernetes Operators?

Kubernetes has taken the cloud infrastructure world by storm mostly thanks to its declarative API. When working
with Kubernetes, developers provide their cluster's desired configuration to the Kubernetes API, and Kubernetes
is responsible for reconciling the actual state of the cluster with the desired state. This allows developers to
focus on the desired state of their cluster, and let Kubernetes handle the complexities of how to get there.

This works out incredibly well for stateless components, such as containers, network configuration and access
policies. The benefit of stateless components is that they can be replaced at any time, and Kubernetes can
simply create a new instance of the component with the desired configuration. For stateful resources, such as databases,
 this is not the case. Throwing away a running database and creating a new one with the desired configuration
is not an option.

For this reason, reconciling the desired state of a database with its actual state
can be a complex task that requires a lot of domain knowledge. [Kubernetes Operators](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)
were introduced to the Kubernetes ecosystem to help users manage complex stateful resources by codifying
this type of domain knowledge into a Kubernetes controller.

### What is the Atlas Kubernetes Operator?

The Atlas Kubernetes Operator is a Kubernetes controller that uses Atlas to manage
your database schema. The Atlas Kubernetes Operator allows you to define the desired schema
and apply it to your database using the Kubernetes API.

### Declarative schema migrations

![](https://user-images.githubusercontent.com/1522681/236139615-1d10feea-8b19-46a2-905b-b614883c48c8.png)

The Atlas Kubernetes Operator supports [declarative migrations](https://atlasgo.io/concepts/declarative-vs-versioned#declarative-migrations).
In declarative migrations, the desired state of the database is defined by the user and the operator is responsible
for reconciling the desired state with the actual state of the database (planning and executing `CREATE`, `ALTER`
and `DROP` statements).

### Diffing policies

One of the common objections to applying declarative workflows to databases is that there are often
multiple ways to achieve the same desired state. For example, if you are running a Postgres database, you may want
to add an index to a table. Depending on your circumstances, you may want to add this index with or without the `CONCURRENTLY`
option. When using a declarative workflow, you supply _where_ you want to go, but not _how_ to get there.

To address this concern, we have introduced the concept of "diff policies" to Atlas. Diff policies allow you
to customize the way Atlas plans database schema changes for you. For example, you can define a diff policy that
will always add the `CONCURRENTLY` option to `CREATE INDEX` statements. You can also define a diff policy that
will skip certain kinds of changes (for example `DROP COLUMN`) altogether.

Diff policies can be defined in the `atlas.hcl` file you use to configure Atlas. For example:

```hcl
env "local" {
  diff {
    // By default, indexes are not created or dropped concurrently.
    concurrent_index {
        create = true
        drop   = true
      }
  }
}

```

Diff policies are especially valuable when using the Atlas Kubernetes Operator, as they allow you to customize
and constrain the way the operator manages your database to account for your specific needs.  We will see an example
of this below.

### Demo time!

Let's see the Atlas Kubernetes Operator in action. In this demo, we will use the Atlas Kubernetes Operator to manage
a MySQL database running in a Kubernetes cluster.

The Atlas Kubernetes Operator is available as a Helm Chart. To install the chart with the release
name `atlas-operator`:

```bash
helm install atlas-operator oci://ghcr.io/ariga/charts/atlas-operator
```

After installing the
operator, follow these steps to get started:

1. Create a MySQL database and a secret with an [Atlas URL](https://atlasgo.io/concepts/url)
  to the database:

  ```bash
  kubectl apply -f https://raw.githubusercontent.com/ariga/atlas-operator/65dce84761354d1766041c7f286b35cc24ffdddb/config/integration/databases/mysql.yaml
  ```

  Result:

  ```bash
  deployment.apps/mysql created
  service/mysql created
  secret/mysql-credentials created
  ```

2. Create a file named `schema.yaml` containing an `AtlasSchema` resource to define the desired schema:

  ```yaml
  apiVersion: db.atlasgo.io/v1alpha1
  kind: AtlasSchema
  metadata:
    name: atlasschema-mysql
  spec:
    urlFrom:
      secretKeyRef:
        key: url
        name: mysql-credentials
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

3. Apply the schema:

  ```bash
  kubectl apply -f schema.yaml
  ```

  Result:
  ```bash
  atlasschema.db.atlasgo.io/atlasschema-mysql created
  ```

4. Check that our table was created:

  ```bash
  kubectl exec -it $(kubectl get pods -l app=mysql -o jsonpath='{.items[0].metadata.name}') -- mysql -uroot -ppass -e "describe myapp.users"
  ```

  Result:

  ```bash
  +-----------+--------------+------+-----+---------+----------------+
  | Field     | Type         | Null | Key | Default | Extra          |
  +-----------+--------------+------+-----+---------+----------------+
  | id        | int          | NO   | PRI | NULL    | auto_increment |
  | name      | varchar(255) | NO   |     | NULL    |                |
  | email     | varchar(255) | NO   | UNI | NULL    |                |
  | short_bio | varchar(255) | NO   |     | NULL    |                |
  +-----------+--------------+------+-----+---------+----------------+
  ```

  Hooray! We applied our desired schema to our target database.

### Diff policies in action

Now let's see how we can use diffing policies to customize the way the operator manages our database. In this example,
we will demonstrate how we can prevent the operator from dropping columns in our database. Modify the `schema.yaml` file:

```diff
apiVersion: db.atlasgo.io/v1alpha1
kind: AtlasSchema
metadata:
  name: atlasschema-mysql
spec:
  urlFrom:
    secretKeyRef:
      key: url
      name: mysql-credentials
+  policy:
+    diff:
+      skip:
+        drop_column: true
  schema:
    sql: |
      create table users (
        id int not null auto_increment,
        name varchar(255) not null,
        email varchar(255) unique not null,
-        short_bio varchar(255) not null,
        primary key (id)
      );
```
In the example above we added a `policy` section to our `AtlasSchema` resource. In this section, we defined a `diff`
policy that will skip `DROP COLUMN` statements. In addition, we dropped the `short_bio` column from our schema.
Let's apply the updated schema:

```bash
kubectl apply -f schema.yaml
```
Next, wait for the operator to reconcile the desired state with the actual state of the database:

```bash
kubectl wait --for=condition=Ready atlasschema/atlasschema-mysql
```

Finally, let's check that the `short_bio` column was not dropped. Run:

```bash
kubectl exec -it $(kubectl get pods -l app=mysql -o jsonpath='{.items[0].metadata.name}') -- mysql -uroot -ppass -e "describe myapp.users"
```
Result:

```text
+-----------+--------------+------+-----+---------+----------------+
| Field     | Type         | Null | Key | Default | Extra          |
+-----------+--------------+------+-----+---------+----------------+
| id        | int          | NO   | PRI | NULL    | auto_increment |
| name      | varchar(255) | NO   |     | NULL    |                |
| email     | varchar(255) | NO   | UNI | NULL    |                |
| short_bio | varchar(255) | NO   |     | NULL    |                |
+-----------+--------------+------+-----+---------+----------------+
```

As you can see, the `short_bio` column was not dropped. This is because we defined a diffing policy that skips
`DROP COLUMN` statements.

### Linting policies

An alternative way to prevent the operator from dropping columns is to use a linting policy. Linting policies allow
you to define rules that will be used to validate the changes to the schema before they are applied to the database.
Let's see how we can define a policy that prevents the operator from applying destructive changes to the schema.
Edit the `schema.yaml` file:

```diff
```diff
apiVersion: db.atlasgo.io/v1alpha1
kind: AtlasSchema
metadata:
  name: atlasschema-mysql
spec:
  urlFrom:
    secretKeyRef:
      key: url
      name: mysql-credentials
   policy:
+     lint:
+       destructive:
+         error: true
-    diff:
-      skip:
-        drop_column: true
  schema:
    sql: |
      create table users (
        id int not null auto_increment,
        name varchar(255) not null,
        email varchar(255) unique not null,
        primary key (id)
      );
```
In the example above we replaced the `diff` policy with a `lint` policy. In this policy, we defined a `destructive`
rule that will cause the operator to fail if it detects a destructive change to the schema. Notice that the `short_bio`
is not present in the schema (we did this in our previous change).

Let's apply the updated schema:

```bash
kubectl apply -f schema.yaml
```

Next, let's wait for the operator to reconcile the desired state with the actual state of the database:

```bash
kubectl wait --for=condition=Ready atlasschema/atlasschema-mysql --timeout 10s
```
Notice that this time, the operator failed to reconcile the desired state with the actual state of the database:
```text
error: timed out waiting for the condition on atlasschemas/atlasschema-mysql
```
Let's check the reason for this failure:

```bash
kubectl get atlasschema atlasschema-mysql -o jsonpath='{.status.conditions[?(@.type=="Ready")].message}'
```

Result:

```text
destructive changes detected:
- Dropping non-virtual column "short_bio"
```

Hooray! We have successfully prevented the operator from applying destructive changes to our database.

### Conclusion

In this post, we have presented the Atlas Operator and demonstrated how you can use it to manage your database schema.
We also covered diffing and linting policies and showed how you can use them to customize the way the operator manages
your database.

#### How can we make Atlas better?

We would love to hear from you [on our Discord server](https://discord.gg/zZ6sWVg6NT) :heart:.
