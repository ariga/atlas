---
id: getting-started
title: Getting Started with Atlas Cloud
sidebar_label: Getting Started
---
import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';
import Discord from '../../website/src/assets/icons/discord-white.svg'

[Atlas Cloud](https://atlasgo.cloud) is an online platform that allows developers to view their Atlas projects and track
their [Atlas GitHub Action](/integrations/github-actions) CI runs. The cloud platform gives full visibility into your
Atlas schema, by displaying an entity relation diagram (ERD) that shows the schema changes, as well as the entire
database schema.

### Signing Up

![Login](https://atlasgo.io/uploads/images/login-new.png)
1. To get started with [Atlas Cloud](https://atlasgo.cloud/), create an account by clicking 'sign up' on the homepage.
2. In the sign up screen, you can either sign up with GitHub or enter your work email and choose a password.
3. Next, you will receive an email to verify your account. From your email, [Atlas Cloud](https://atlasgo.cloud/)
will open in a new tab, and you will need to sign in to access your account.
4. Once signed in, you will be asked to confirm the service agreement in order to continue with the login process.
5. Next, you will be prompted to create an organization. After creating the organization, you will be able
to invite team members to join. Choose a meaningful name for the organization, as it will also be your subdomain.
For example, "Acme Corp" will be available at "acme-corp.atlasgo.cloud".

### Choosing an organization
![Choosing organization](https://atlasgo.io/uploads/images/choose-org-new.png)
Atlas Cloud allows you to be a part of multiple organizations.
After logging in (not including the first login), you will see all the organizations you have access to. Choose the organization you would to log in to.
If you would like to create a new organization, click 'create a new organization'.
While logged in to one organization, you would be able to switch to another by clicking on your avatar in the bottom left corner and choose the organization you would like to switch to.

![Switching organization](https://atlasgo.io/uploads/images/switch-org.png)

### Connecting to the Atlas GitHub action
In order to connect to the Atlas GitHub Action, we will create a bot with permission to report CI run results. Bots are used to manage permissions and make sure dedicated actions, such as reporting CI results, are executed only by the entity with the desired permission.
At first you will notice that your projects and CI runs pages are empty. In order to connect the organization
to your GitHub repository, you will need to setup the Atlas GitHub action on your repository by following these steps:

:::note
If you already have the Atlas GitHub action set up, you may skip step 4. In step 5, only add
`ariga-token: ${{ secrets.ARIGA_TOKEN }}` to your yaml file.
:::

1. From the Settings page, under 'Bots', click on "Create Bot" to create a new CI bot.  
The CI bot will be used to connect your GitHub and will allow Atlas Cloud to lint your pull requests.
2. After saving your CI bot, click on "Create Token", and generate a token for your bot.
3. On your GitHub repo, under the 'Settings' section, click on 'Secrets' > 'Actions' to create a new repository secret.
![GitHub Secrets](https://release.ariga.io/images/assets/github-secrets.png)
:::note
If you do not see this on your GitHub repository, ask your repository owner for access or help.
:::
4. Name your secret (for example, ARIGA_TOKEN) and paste the generated token from step 2.
5. Install the Atlas GitHub Action by adding a file named `.github/workflows/atlas-ci.yaml` to your repo.
6. Based on the type of database you are using, copy the following code into the workflow definition file. Set up
the `ariga-token` input parameter to the secret name you chose in the previous step, and ensure your
mainline branch and migration directory path are configured correctly:

<Tabs
defaultValue="mysql"
values={[
{label: 'MySQL', value: 'mysql'},
{label: 'PostgreSQL', value: 'postgres'},
{label: 'MariaDB', value: 'maria'},
]}>
<TabItem value="mysql">

```yaml
name: Atlas CI
on:
  # Run whenever code is changed in the master branch,
  # change this to your root branch.
  push:
    branches:
    // highlight-next-line
      - master
  # Run on PRs where something changed under the `path/to/migration/dir/` directory.
  pull_request:
    paths:
    // highlight-next-line
      - 'path/to/migration/dir/*'
jobs:
  lint:
    services:
      # Spin up a mysql:8.0.29 container to be used as the dev-database for analysis.
      mysql:
        image: mysql:8.0.29
        env:
          MYSQL_ROOT_PASSWORD: pass
          MYSQL_DATABASE: test
        ports:
          - "3307:3306"
        options: >-
          --health-cmd "mysqladmin ping -ppass"
          --health-interval 10s
          --health-start-period 10s
          --health-timeout 5s
          --health-retries 10
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3.0.1
        with:
          fetch-depth: 0 # Mandatory unless "latest" is set below.
      - uses: ariga/atlas-action@v0
        with:
        // highlight-next-line
          dir: 'path/to/migrations'
          dir-format: atlas # Or: golang-migrate, goose, dbmate, flyway, liquibase
          dev-url: mysql://root:pass@localhost:3307/test
          // highlight-next-line
          ariga-token: ${{ secrets.ARIGA_TOKEN }}
```


</TabItem>
<TabItem value="postgres">

```yaml
name: Atlas CI
on:
  # Run whenever code is changed in the master branch,
  # change this to your root branch.
  push:
    branches:
    // highlight-next-line
      - master
  # Run on PRs where something changed under the `path/to/migration/dir/` directory.
  pull_request:
    paths:
    // highlight-next-line
      - 'path/to/migration/dir/*'
jobs:
  lint:
    services:
      # Spin up a postgres:14 container to be used as the dev-database for analysis.
      postgres14:
        image: postgres:14
        env:
          POSTGRES_DB: test
          POSTGRES_PASSWORD: pass
        ports:
          - 5430:5432
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3.0.1
        with:
          fetch-depth: 0 # Mandatory unless "latest" is set below.
      - uses: ariga/atlas-action@v0
        with:
        // highlight-next-line
          dir: 'path/to/migrations'
          dir-format: atlas # Or: golang-migrate, goose, dbmate, flyway, liquibase
          dev-url: postgres://postgres:pass@localhost:5430/test?sslmode=disable
        // highlight-next-line
          ariga-token: ${{ secrets.ARIGA_TOKEN }}
```

</TabItem>
<TabItem value="maria">

```yaml
name: Atlas CI
on:
  # Run whenever code is changed in the master branch,
  # change this to your root branch.
  push:
    branches:
    // highlight-next-line
      - master
  # Run on PRs where something changed under the `path/to/migration/dir/` directory.
  pull_request:
    paths:
    // highlight-next-line
      - 'path/to/migration/dir/*'
jobs:
  lint:
    services:
      # Spin up a maria:10.7 container to be used as the dev-database for analysis.
      maria107:
        image: mariadb:10.7
        env:
          MYSQL_DATABASE: test
          MYSQL_ROOT_PASSWORD: pass
        ports:
          - 4306:3306
        options: >-
          --health-cmd "mysqladmin ping -ppass"
          --health-interval 10s
          --health-start-period 10s
          --health-timeout 5s
          --health-retries 10
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3.0.1
        with:
          fetch-depth: 0 # Mandatory unless "latest" is set below.
      - uses: ariga/atlas-action@v0
        with:
        // highlight-next-line
          dir: 'path/to/migrations'
          dir-format: atlas # Or: golang-migrate, goose, dbmate, flyway, liquibase
          dev-url: maria://root:pass@localhost:4306/test
        // highlight-next-line
          ariga-token: ${{ secrets.ARIGA_TOKEN }}
```
</TabItem>

</Tabs>

6. After merging the workflow to your mainline branch, the workflow will be triggered.
7. Refresh Atlas Cloud and your project will appear!

![Setup Projects](https://atlasgo.io/uploads/images/new-project.png)

### Viewing CI Runs
In the system, you can view all the CI runs that were triggered by the Atlas GitHub workflow.

Each run includes the following:
- A summary of the run.
- SQL statements that were analyzed.
- An ERD that shows the changes made to the schema, as well as a full view.

A run can complete in one of three ways:

<Tabs
defaultValue="successful"
values={[
{label: 'Successful', value: 'successful'},
{label: 'Issues Found', value: 'issues'},
{label: 'Failed', value: 'failed'},
]}>

<TabItem value="successful">

![Successful Run](https://atlasgo.io/uploads/images/successful-ci.png)

CI ran successfully and no errors or issues were found in your SQL statements or Atlas sum file.

</TabItem>

<TabItem value = "issues">

![Issues Found Run](https://atlasgo.io/uploads/images/issues-found-ci.png)

In cases where your SQL statements _might_ cause a failure in production, the CI run will be labeled as 'issues
found'. In this example, we can see that the column `name` was created as non-nullable. The CI is letting us know that
this has a chance of causing a failure, because if there is a row that exists in this table that has a null `name`
value, this migration will for a fact fail in production.
The report also makes sure to reference the specific data-dependent check that was found
[MF103](https://atlasgo.io/lint/analyzers#MF103), in this example).
</TabItem>

<TabItem value = "failed">

![Failed Run](https://atlasgo.io/uploads/images/failed-ci.png)
The CI run can fail for multiple reasons: incorrect SQL statements, wrong configuration, and more.
In this example, we can see the CI has failed due to an SQL statement that attempts to drop a table. Because this
is dangerous and will result in loss of data, the CI will automatically fail any `drop` statements.
However, users can disable this by configuring the destructive analyzer in the
[`atlas.hcl`](../atlas-schema/projects.mdx) file:

```hcl title="atlas.hcl"
lint {
  //highlight-start
  destructive {
    error = false
  }
  //highlight-end
}
```
</TabItem>
</Tabs>

### Inviting Members
Under 'Settings' > 'Members', you can invite team members to your organization.
These members will receive an email with a link to Atlas Cloud, and will be required to sign up with the same email
in order to access the organization.

### Deleting Bots and Tokens
It is possible to delete a bot, however once you do all of its tokens will be **invalidated**.  
If you would like to generate a new bot with a new token, make sure to update your GitHub project's 'Secrets' with the new token.

:::info
For more help, reach out to us on our [Discord server](https://discord.gg/zZ6sWVg6NT).
:::
