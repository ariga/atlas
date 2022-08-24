---
title: Prevent destructive changes to your database with the Atlas GitHub Action
authors: rotemtam
tags: [ci, github-actions, announcement]
image: https://atlasgo.io/uploads/images/actions-share-image.png
---

Losing data is painful for almost all organizations. This is one of the reasons teams are very
cautious when it comes to making changes to their databases. In fact, many teams set explicit
policies on what kinds of changes to the database are allowed, often completely prohibiting
any change that is destructive.

Destructive changes are changes to a database schema that result in loss of data. For instance,
consider a statement such as:

```sql
ALTER TABLE `users` DROP COLUMN `email_address`;
```

This statement is considered destructive because whatever data is stored in the `email_address`
column will be deleted from disk, with no way to recover it.

Suppose you were in charge of a team that decided to prohibit destructive changes,
how would you go about enforcing such a policy? From our experience, most teams enforce
policies relating to schema migrations in code-review: a human engineer, preferably with
some expertise in operating databases, manually reviews any proposed database migration
scripts and rejects them if they contain destructive changes.

Relying on a human reviewer to enforce such a policy is both expensive (it takes time and mental energy)
and error-prone. Just like manual QA is slowly being replaced with automated testing, and manual
code style reviews are being replaced with linters, isn't it time that we automate the process
of ensuring that changes to database schemas are safe?

### Announcing the Atlas GitHub Action

Today, we're happy to announce the release of the official [Atlas GitHub Action](https://github.com/ariga/atlas-action)
which can be used to apply [migration directory linting](2022-07-14-announcing-atlas-lint.md) for a bunch of popular
database migration tools. [golang-migrate](https://github.com/golang-migrate/migrate),
[goose](https://github.com/pressly/goose), [dbmate](https://github.com/amacneil/dbmate) and Atlas itself are already supported,
and Flyway and Liquibase are coming soon.

If you're using GitHub to manage your source code, you're in luck. By adding a short configuration
file to your repository, you can start linting your schema migration scripts today! Let's see a
short example.

### Setting up

Suppose we are running a website for an e-commerce business. To store the data for our website
we use a MySQL database. Because the data in this database is _everything_ to us, we use
a careful _versioned migrations_ approach where each change to the database schema is
described in an SQL script and stored in our Git repository. To execute these scripts we use
a popular tool called [golang-migrate](https://github.com/golang-migrate/migrate).

The source code for this example can be found in [rotemtam/atlas-action-demo](https://github.com/rotemtam/atlas-action-demo).

Initially, our schema contains two tables: `users` and `orders`, documented in the first few migration files:

Create the `users` table:
```sql title=migrations/20220819060736.up.sql
-- create "users" table
CREATE TABLE `users` (
    `id` int NOT NULL,
    `name` varchar(100) NULL,
    PRIMARY KEY (`id`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
```

Add a unique `email` column:
```sql title=migrations/20220819061056.up.sql
ALTER TABLE `users` ADD COLUMN `email` varchar(255) NOT NULL, ADD UNIQUE INDEX `email_unique` (`email`);
```

Create the `orders` table, with a foreign-key referencing the `users` table:
```sql title=migrations/20220819075145.up.sql
-- create "orders" table
CREATE TABLE `orders` (
    `id` int NOT NULL,
    `user_id` int NOT NULL,
    `total` decimal(10) NOT NULL,
    PRIMARY KEY (`id`),
    INDEX `user_orders` (`user_id`),
    CONSTRAINT `user_orders` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
```

### Installing the Atlas Action

To make sure we never accidentally delete data during schema changes, we enact a policy that prohibits
destructive changes to the database. To enforce this policy, we invoke the `atlas-action` GitHub Action
from within our continuous integration flow by adding a workflow file name `.github/workflows/atlas-ci.yaml`:

```yaml title=.github/workflows/atlas-ci.yaml
name: Atlas CI
on:
  # Run whenever code is changed in the master branch,
  # change this to your root branch.
  push:
    branches:
      - master
  # Run on PRs where something changed under the `path/to/migration/dir/` directory.
  pull_request:
    paths:
      - 'migrations/*'
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
          - "3306:3306"
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
          dir: migrations/
          dir-format: golang-migrate # Or: atlas, goose, dbmate
          dev-url: mysql://root:pass@localhost:3306/test
```

### Detecting a destructive change

Next, let's see what happens when a developer accidentally proposes a destructive change,
to drop a column in the `orders` table:

```sql title=migrations/20220819081116.up.sql
-- modify "orders" table
ALTER TABLE `orders` DROP COLUMN `total`;
```

This change is proposed in [PR #1](https://github.com/rotemtam/atlas-action-demo/pull/1/files)
in our example repo. Because we have previously set up the Atlas GitHub Action to
[lint our migration directory](https://github.com/rotemtam/atlas-action-demo/blob/master/.github/workflows/atlas-ci.yaml),
whenever a file changes under the `migrations/` directory, a workflow is triggered.

After letting our workflow complete, observe that GitHub informs us that the `Atlas CI / lint`
check has failed:

![](https://atlasgo.io/uploads/images/workflow-summary-01.png)

Clicking on the ["details" link](https://github.com/rotemtam/atlas-action-demo/runs/7960178186?check_suite_focus=true) we find a detailed explanation on the causes for
the failure:

![](https://atlasgo.io/uploads/images/workflow-summary-02.png)

Examining the Action [run summary](https://github.com/rotemtam/atlas-action-demo/actions/runs/2906742450) we find the following annotation:

![](https://atlasgo.io/uploads/images/workflow-summary-03.png)

As you can see, Atlas has detected the destructive change we proposed to apply 
to our database and failed our build!

### Wrapping up

In this post we discussed why many teams set policies to prevent destructive
changes to database schemas. We further showed how such policies can be enforced
in an automated way using the official [Atlas GitHub Action](https://atlasgo.io/integrations/github-actions).


### Further reading

To learn more about CI for database schema changes:
* Read the [docs for `atlas migrate lint`](/versioned/lint)
* Learn about [migration analyzers](/lint/analyzers)
* [CLI Command Reference](/cli-reference#atlas-migrate-lint)

Have questions? Feedback? Find our team [on our Discord server](https://discord.gg/zZ6sWVg6NT).
