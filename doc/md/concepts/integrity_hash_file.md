---
id: sum_file
slug: /concepts/integrity_hash_file
title: Integrity Sum File
---

## Integrity Hash File
When multiple engineers work on the same project in parallel, there is a chance that their migration scripts will conflict with each other and may cause unexpected behavior. For example:
One can retroactively change the content of migrations that have already run.
One can accidentally change the order in which migrations are organized.
One can commit semantically incorrect SQL scripts.

Atlas creates the Integrity Hash File - internally named “atlas.sum” - to tackle these issues. 

The atlas.sum file contains a sum of the whole directory as its first entry, and a checksum for each of the migration files. Its format is inspired by the go.sum file of a Go module, and it would look similar to this:
```
h1:KRFsSi68ZOarsQAJZ1mfSiMSkIOZlMq4RzyF//Pwf8A=
20220318104614_team_A.sql h1:EGknG5Y6GQYrc4W8e/r3S61Aqx2p+NmQyVz/2m8ZNwA=
```

On each run of “atlas migrate” Atlas verifies that your migration directory matches your atlas.sum file. You’ll see an alert in case there’s a discrepancy between the files` signatures and the atlas.sum file.
