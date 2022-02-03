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

When opening the UI from the inspect command, the schema is presented. On the bottom-right, the current schema is
described using the Atlas DDL. On the top-right, the “Activity & History” panel holds an audit history for all changes made to the schema. 
Each log is clickable, allowing to view the diff between the schema and its previous state.
On the bottom-left part of the screen, the UI displays an ERD (Entity-relation Diagram) showing the different tables and the connections
between them (via foreign-keys). Opening the ERD to a fullscreen view offers a more in depth view of each table.  

![Schema ERD open](https://atlasgo.io/uploads/images/docs/schema-erd-open.png)

####Editing your schema
![Edit Schema](https://atlasgo.io/uploads/images/docs/edit-schema.png)  
The edit schema screen is where you can make changes to your schema. To learn more about the Atlas DDL syntax, click 
[here](https://atlasgo.io/ddl/intro#hcl).  

####Desired Schema vs. Inspected Schema
![Atlas Schema Not Synced Split](https://atlasgo.io/uploads/images/docs/schema-out-of-sync-split.png)
After editing a schema (in this case after adding the table `likes`, which is shown in green in the ERD) the schema will be out of sync.
When a schema is out of sync, two separate schemas appear: desired and inspected.
The desired schema is the one being edited in the UI. The inspected schema is the
true, current state of the database. 
By clicking the toggle and changing the view to "unified", the schema is shown as one with a highlighted diff.  

![Atlas Schema Not Synced Unified](https://atlasgo.io/uploads/images/docs/schema-out-of-sync-unified.png)

After running a migration, the desired and inspected schemas become one -  the changes have been applied to the database.

For example, `likes` became a part of the schema after running a migration:

![Synced Schema After Migration](https://atlasgo.io/uploads/images/docs/synced-schema-after-migration.png)


