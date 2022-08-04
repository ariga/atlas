---
id: new
slug: /versioned/new
title: Manual Migrations
---
In some cases it is desirable to add a migration file manually. This could be done
to provision resources that Atlas does not yet capture in its DDL (such as triggers and
views) or to seed data with `INSERT` statements. 

To manually add a new migration file to the directory use the `migrate new` command. 

### Flags
When using `migrate new` to create a new migration file users may supply the following flags:
* `--dir` the URL of the migration directory, by default it is `file://migrations`, e.g a
  directory named `migrations` in the current working directory.

### Migration name
Users may optionally add a final positional argument to set the name of the migration
file.  This name will be appended to the migration version number in the filename as such:
`<version>_<name>.sql`.

### Recalculating the directory hash
Atlas maintains a file named `atlas.sum` in the migration directory. This file  is used to 
ensure the integrity of the migration directory and force developers to deal with 
situations where migration order or contents was modified after the fact. 

After manually editing the contents of a newly created migration file, the checksums for
the directory must be recalculated. This can be done by running `atlas migrate hash --force`
command.

### Examples

Creating a new migration file:
```text
atlas migrate new
```

Creating a new migration file named "add_user":
```text
atlas migrate new add_user
```

Creating a new migration file in a specific directory:
```text
atlas migrate new --dir file://custom/path/to/dir
```


