# Custom Configuration

The operator generates an `atlas.hcl` for each resource. `config` and `configFrom` add your own
configuration to it: data sources such as `composite_schema`, `external_schema`, `aws_rds_token`, and
`gcp_cloudsql_token`, variables, and env settings
(https://atlasgo.io/integrations/kubernetes/project-configuration).

## Enabling It

1. Install or upgrade the operator with `allowCustomConfig`:

   ```bash
   helm install atlas-operator oci://ghcr.io/ariga/charts/atlas-operator --set allowCustomConfig=true
   ```

   Without it, a resource with `config` fails with
   `install the operator with "--set allowCustomConfig=true" to use custom atlas.hcl config`. Custom
   configuration can run data sources that execute commands and queries inside the operator pod, so
   enable it only where you trust whoever writes the resources.
2. Give the resource a token (`cloud.tokenFrom`): custom configuration is an Atlas Pro feature, and
   fails without one with `command requires 'atlas login'` or
   `login is required to use custom atlas.hcl config`.
3. Set `envName` to the env block the configuration defines. Without it, the resource fails with
   `env name must be set when using custom atlas.hcl config`. If no block has that name, it fails with
   `env block "<name>" is not found`.

## Writing It

`config` holds HCL inline. `configFrom` reads it from a Secret, for configuration that carries
credentials:

```yaml
spec:
  envName: "prod"
  configFrom:
    secretKeyRef:
      key: config
      name: my-secret
```

An unnamed env block takes its name from `envName`, so one configuration serves several environments:

```yaml
spec:
  envName: "prod"  # This value is passed to `atlas.env`
  config: |
    env {
      name = atlas.env  # Resolves to "prod" at runtime
      url  = "postgres://..."
    }
```

- When the env sets `url`, the resource's `url`, `urlFrom`, and `credentials` are ignored. Use one or the
  other.
- When the env sets `dev`, the operator creates no dev database and uses that URL.
- The resource's inline schema is `file://schema.sql` or `file://schema.hcl` inside the configuration,
  and an inline or ConfigMap migration directory is `file://migrations`.
- The operator merges the configuration with the one it generates: in a block of the same type and name,
  your attributes win.
- `AtlasMigration` still requires `spec.dir`.
- One resource targets one database: an env with `for_each` fails with
  `multiple targets are not supported`.

## Variables

`vars` passes values to `variable` blocks, inline or from a Secret or ConfigMap:

```yaml
spec:
  envName: "prod"
  vars: 
    - key: "url"
      valueFrom: 
        secretKeyRef:
          name: secret-creds
          key: url
  config: |
    variable "url" {
      type = string
    }
    env "prod" {
      url = var.url
    }
```

`configMapKeyRef` reads a ConfigMap the same way, and `value` sets the value inline. With Helm, template
the `vars` values from the chart's values file.

- A key that the Secret or ConfigMap does not have reads as an empty string, without an error. A wrong
  key in `configFrom` leaves the configuration empty: check the names when the configuration seems
  ignored.
- The operator does not watch the Secrets and ConfigMaps behind `configFrom` and `vars`. A change to them
  applies at the next reconcile; change the resource's spec to apply it now.

## Combining Schemas

`composite_schema` builds the desired state from several sources:

```yaml
apiVersion: db.atlasgo.io/v1alpha1
kind: AtlasSchema
metadata:
  name: atlasschema-pg
spec:
  urlFrom:
    secretKeyRef:
      key: url
      name: postgres-credentials
  cloud:
    tokenFrom:
      secretKeyRef:
        key: token
        name: atlas-token
  envName: custom-env
  config: |
    data "external_schema" "users" {
      program = [
        "echo",
        "CREATE TABLE users (id INT, name TEXT);"
      ]
    }
    data "external_schema" "posts" {
      program = [
        "echo",
        "CREATE TABLE posts (id INT, user_id INT, title TEXT);"
      ]
    }
    data "composite_schema" "default" {
      schema "public" {
        url = data.external_schema.users.url
      }
      schema "public" {
        url = data.external_schema.posts.url
      }
    }
    env "custom-env" {
      schema {
        src = data.composite_schema.default.url
      }
    }
```

## IAM Authentication

### AWS RDS

Requires IAM authentication on the instance, a database user for it, and an IAM role with
`rds-db:connect` attached to the operator's service account:

```yaml
apiVersion: db.atlasgo.io/v1alpha1
kind: AtlasMigration
metadata:
  name: atlas-migration
spec:
  envName: "prod"
  cloud:
    tokenFrom:
      secretKeyRef:
        name: atlas-token
        key: token
  dir:
    remote:
      name: "myapp"
      tag: "commit-id"
  config: |
    data "aws_rds_token" "db" {
      endpoint = "mydb.xxx.us-east-1.rds.amazonaws.com:5432"
      region   = "us-east-1"
      username = "iamuser"
    }
    env {
      name = atlas.env
      url  = "postgres://iamuser:${urlescape(data.aws_rds_token.db)}@mydb.xxx.us-east-1.rds.amazonaws.com:5432/mydb?sslmode=require"
    }
```

The token holds characters that must be escaped in a URL, hence `urlescape()`. For MySQL, the endpoint
uses port `3306`, and the env's URL is:

```hcl
url  = "mysql://iamuser:${urlescape(data.aws_rds_token.db)}@mydb.xxx.us-east-1.rds.amazonaws.com:3306/mydb?tls=true&allowCleartextPasswords=1"
```

### GCP Cloud SQL

Requires IAM authentication on the instance, a database user for it, and Workload Identity for the
operator's service account:

```yaml
apiVersion: db.atlasgo.io/v1alpha1
kind: AtlasMigration
metadata:
  name: atlas-migration
spec:
  envName: "prod"
  cloud:
    tokenFrom:
      secretKeyRef:
        name: atlas-token
        key: token
  dir:
    remote:
      name: "myapp"
      tag: "commit-id"
  config: |
    data "gcp_cloudsql_token" "db" {}
    env {
      name = atlas.env
      url  = "postgres://iamuser:${urlescape(data.gcp_cloudsql_token.db)}@10.0.0.1:5432/mydb?sslmode=require"
    }
```

For MySQL, Cloud SQL IAM needs `allowCleartextPasswords` and `tls` in the URL:

```hcl
url  = "mysql://iamuser:${urlescape(data.gcp_cloudsql_token.db)}@10.0.0.1:3306/mydb?allowCleartextPasswords=1&tls=skip-verify&parseTime=true"
```
