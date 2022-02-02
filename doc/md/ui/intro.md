---
id: atlas-ui-intro
slug: /ui/intro
title: Management UI
---

## Introduction

As many other CLI tools, you can use Atlas to manage your schemas via your terminal.
While this is the common interface for many infrastructure management workflows
we believe that a visual, integrated environment can be beneficial in many use-cases.

### Usage

To launch the Atlas UI, run either `schema inspect` or `schema apply` commands with the `-w` command-line
flag:

#### Inspect
```
atlas schema inspect -d "mysql://root:pass@tcp(localhost:3306)/example" -w
```

#### Apply
```
atlas schema apply -d "mysql://root:pass@tcp(localhost:3306)/example" -f atlas.hcl -w
```

## Features

### Manage your schema
![Atlas Schema Synced](https://atlasgo.io/uploads/images/docs/synced-atlas-schema.png)

When opening the UI from the inspect command, you can view your schema and manage it. On the bottom-right, the current schema is
described using the Atlas DDL. On the top-right, the “Activity & History” panel that holds an audit history for all changes to your schema. 
Each log is clickable, and you can view the diff between the schema and its previous state.
On the bottom-left part of the screen, the UI displays an ERD (Entity-relation Diagram) showing the different tables and the connections
between them (via foreign-keys). When opening the ERD to a fullscreen view, you will be able to see a more in depth view of each table.  

![Schema ERD open](https://atlasgo.io/uploads/images/docs/schema-erd-open.png)

####Editing your schema
![Edit Schema](https://atlasgo.io/uploads/images/docs/edit-schema.png)  
By clicking the 'edit schema' button, an editor opens up where you can make changes to your schema.

####Desired Schema vs. Inspected Schema
![Atlas Schema Not Synced Split](https://atlasgo.io/uploads/images/docs/schema-out-of-sync-split.png)
After editing your schema (in this case we added the table `likes`, which you can see in green in the ERD) the schema will be out of sync.
When a schema is out of sync, you will see two separate schemas: desired and inspected.
The desired schema is the one you are making changes to and editing in the UI. The inspected schema is the
true, current state of your database. 
By clicking the toggle and changing the view to "unified", you can view the schema as one and see the diff.  

![Atlas Schema Not Synced Unified](https://atlasgo.io/uploads/images/docs/schema-out-of-sync-unified.png)

Once you run a migration, the desired and inspected schemas become one - you have applied the changes
that you made in Atlas to your database. 

Here we can see that `likes` became a part of our schema after running a migration:

![Synced Schema After Migration](https://atlasgo.io/uploads/images/docs/synced-schema-after-migration.png)


