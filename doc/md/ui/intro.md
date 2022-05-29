---
id: atlas-ui-intro
slug: /ui/intro
title: Management UI
---

:::info
The Management UI is temporarily unavailable in the latest version of Atlas. The UI will be brought back soon, however if
you wish to use the UI in the meantime, download [Atlas v0.3.7](https://github.com/ariga/atlas/releases/tag/v0.3.7).
:::

## Introduction

As many other CLI tools, you can use Atlas to manage your schemas via your terminal.
While this is the common interface for many infrastructure management workflows
we believe that a visual, integrated environment can be beneficial in many use-cases.

### Opening the UI

Get started with the UI by running the command `atlas serve`.
To ensure that your team's schemas, database credentials, migration history and more are not lost, save the data
in the UI by running the command with [persistent storage](https://atlasgo.io/deployment/intro#persistent-storage):
```
atlas serve --storage mysql://root:pass@localhost:3306/storage_db
```

Alternatively, you can run the commands `schema inspect` or `schema apply` with the `-w` command-line
flag:

#### Inspect
```
atlas schema inspect -u "mysql://root:pass@localhost:3306/example" -w
```

#### Apply
```
atlas schema apply -u "mysql://root:pass@localhost:3306/example" -f atlas.hcl -w
```

### Usage
:::note
The following example is based on the schema created in the Getting Started tutorial.
Click [here](http://localhost:3000/cli/getting-started/setting-up) to start the tutorial from the beginning.
:::
When the UI opens, we are taken to the schema page. This screen displays a high-level view of our Atlas schema.

![Atlas Schema Synced](https://atlasgo.io/uploads/images/docs/synced-atlas-schema.png)


On the top-left (1) we can see a box with information about our schema. The "Activity & History" panel (2) holds an audit history for all changes made to the schema. Each log is clickable, allowing us to view the diff between the schema and its previous state. On the bottom-right (3), the schema is described using the Atlas DDL.  On the bottom-left part of the screen (4), the UI displays an ERD (Entity-relation diagram) showing the different tables and the connections between them (via foreign-keys). By clicking the "expand" icon on the top-right corner of the ERD panel, we can open a more detailed view of our schema.

![Schema ERD open](https://atlasgo.io/uploads/images/docs/schema-erd-open.png)

#### Running a migration

Now that we have seen how Atlas can visualize our database schema in the UI, let’s see how to initiate a migration.

First, we will make changes to our schema by clicking on the "Edit Schema" button on the top right-hand corner.

![Edit Schema Button](https://atlasgo.io/uploads/images/docs/edit-schema-button.png)

Next, add the table `likes` to the schema:
```hcl
table "likes" {
    schema = schema.example
    column "id" {
        null = false
        type = int
    }
    column "user_id" {
        null = false
        type = int
    }
    column "blog_post_id" {
        null = false
        type = int
    }
    primary_key {
        columns = [table.likes.column.id]
    }
    foreign_key "like_blog_post" {
        columns     = [table.likes.column.blog_post_id]
        ref_columns = [table.blog_posts.column.id]
        on_update   = RESTRICT
        on_delete   = RESTRICT
    }
    foreign_key "like_user" {
        columns     = [table.likes.column.user_id]
        ref_columns = [table.users.column.id]
        on_update   = RESTRICT
        on_delete   = RESTRICT
    }
    index "like_blog_post" {
        columns = [table.likes.column.blog_post_id]
    }
    index "like_user" {
        columns = [table.likes.column.user_id]
    }
}
```

Click "save" on the top right-hand corner.

![Save Schema Button](https://atlasgo.io/uploads/images/docs/save-schema.png)

Going back to our schema page, we can notice a few changes. First, our schema is now out of sync. This means we have created a change in the desired state that isn't yet reflected in the database. When a schema is out of sync, two separate schemas appear: desired and inspected.
The _desired_ schema is the one being edited in the UI. The _inspected_ schema is the
true, current state of the database. By clicking the toggle and changing the view to "unified", the schema is shown as one with a highlighted diff.

Looking at the ERD, we can see that our table `likes` is green and has a plus-sign (+) next to it, showing that it needs to be added. This means the change has been recorded, but not yet applied.


![Atlas Schema Not Synced Split](https://atlasgo.io/uploads/images/docs/schema-out-of-sync-unified.png)

Now, let’s run a migration!
Click "Migrate Schema" in the Schema Status box to apply the changes we want to make to our schema.

The migration screen will open up, running the setup automatically. After the setup succeeds, we will see the migration plan.
The diff in the schema in HCL on the left pane, and the planned SQL statements on the right.

![Atlas Schema Not Synced Split](https://atlasgo.io/uploads/images/docs/migrate-plan-step.png)

To start the migration, click "Apply". If you ever wish to not go through with a migration, you can click "Abort".

![Atlas Schema Not Synced Split](https://atlasgo.io/uploads/images/docs/migrate-apply-step.png)

Congrats, we have successfully completed a migration!

As expected, after executing our migration plan, our database and desired schema are now synced.

![Atlas Schema Not Synced Split](https://atlasgo.io/uploads/images/docs/synced-schema-after-migration.png)

### Conclusion

In this section we learned how to use the Management UI to inspect our database and run migrations.
