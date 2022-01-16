---
id: atlas-ui-intro
slug: /ui/intro
title: Atlas UI
---

To launch the Atlas UI, run either `schema inspect` or `schema apply` commands with the `-w` command-line
flag:

## Examples

### Inspect
```
atlas schema inspect -d "mysql://root:pass@tcp(localhost:3306)/example" -w
```

### Apply
```
atlas schema apply -d "mysql://root:pass@tcp(localhost:3306)/example" -f atlas.hcl -w
```


