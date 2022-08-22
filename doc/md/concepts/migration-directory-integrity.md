---
id: migration-directory-integrity
slug: /concepts/migration-directory-integrity
title: Migration Directory Integrity File
---

Applying changes to a database schema in the wrong order can be dangerous. For this
reason, Atlas is built on a workflow that enforces a _linear history_ using a 
_migration directory integrity file_.

### The problem

Suppose you have multiple teams working on a system simultaneously, all of which need to make 
changes to the database schema from time to time. Unless they somehow coordinate, they
may end up with a broken migration directory.

Consider what would happen if Team A and B, working in parallel, both merge a migration
that creates a new table named `inventory`. This is illustrated in this diagram:

![Teams committing migrations in parallel. Source: Ent blog](https://entgo.io/images/assets/migrate/no-conflict.svg)

Something like this might happen:
1. Team A creates a feature branch, committing a migration creating the `inventory` table. 
2. Team B creates a second feature branch, also creating a table by the same name. 
3. Both branches pass code-review and continuous integration. 
4. Team A's branch is merged to the mainline branch. 
5. Team B's branch is merged. 
6. When both changes are deployed, the first one to run will succeed and the second will fail.
   This will happen in an arbitrary order (migrations are run in lexicographic order, usually set by the timestamp
   on the developer's workstation when generating them).

### The solution

Recovering from a failed migration is quite a headache, so wouldn't it be great
to prevent this from ever happening?

Code conflicts are usually detected by source-control systems (such as Git) when
the same line in the same file is modified by two different commits. In our case,
no such conflict happens because migrations are typically described in a separate
file for each migration. 

Atlas's engine offers a way to prevent concurrent creation of new migration 
files and guards against accidental changes in the migration history we call
*Migration Directory Integrity File*. This file is simply another file in your
migration directory called `atlas.sum` and looks something like:

```text
h1:KRFsSi68ZOarsQAJZ1mfSiMSkIOZlMq4RzyF//Pwf8A=
20220318104614_team_A.sql h1:EGknG5Y6GQYrc4W8e/r3S61Aqx2p+NmQyVz/2m8ZNwA=
```

The `atlas.sum` file contains the checksum of each migration file (implemented by a 
reverse, one branch merkle hash tree), and a sum of all files. Adding new files 
results in a change to the sum file, which will raise merge conflicts in most 
version control systems. 

### How does this mechanism prevent situations like the one we described above? 

The migration directory integrity file is updated automatically
whenever a new migration is created. Therefore, after Team A merged their changes to
the mainline branch, Team B would not be able to do so without dealing
with the changes landed by Team B.

Because of the merge conflict on the `atlas.sum` file, in order to land
their changes, Team B would need to:
1. Merge the latest changes into their branch
2. Resolve any conflicts on the database schema (and application) level
3. Re-compute the `atlas.sum` file (using the [`atlas migrate hash`](/cli-reference#atlas-migrate-hash))
 command.
4. Merge their changes to the mainline branch. 
