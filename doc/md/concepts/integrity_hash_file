---
id: hash_file
slug: /concepts/integrity_hash_file
title: Integrity Hash File
---

## Version Migration Files
Version migration files are files that describe changes to the databases. Usually these files are prefixed with a number - which describe the version of the database change. The framework that runs these files uses these numbers to determine which changes have been applied to the database.

In Ent, you’ll find these files in ent/migrations/ folder.
In Django, you’ll find these files in project_name/app_name/migrations/ folder.

## Integrity Hash File
Databases hold one of the most valuable assets of the organization. Unfortunately, databases are also quite fragile. Especially when working in teams where developers are working in parallel on the same business logic. Data loss is real and that’s why good practice is so important when working with DBs.

There are 3 issues that can occur with version migration files.
1. One can retroactively change the content of migrations that have already run.
2. One can accidentally change the order in which migrations are organized.
3. One can commit semantically incorrect SQL scripts.

Atlas creates the Integrity Hash File - internally named “atlas.sum” - to tackle these issues. 

The atlas.sum file contains a sum of the whole directory as its first entry, and a checksum for each of the migration files. Its format is inspired by the go.sum file of a Go module, and it would look similar to this:
```
h1:KRFsSi68ZOarsQAJZ1mfSiMSkIOZlMq4RzyF//Pwf8A=
20220318104614_team_A.sql h1:EGknG5Y6GQYrc4W8e/r3S61Aqx2p+NmQyVz/2m8ZNwA=
```

On each run of “atlas migrate” Atlas verifies that your migration directory matches your atlas.sum file. You’ll see an alert in case there’s a discrepancy between the files` signatures and the atlas.sum file.
