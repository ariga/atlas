## atlas schema inspect

Inspect an atlas schema

```
atlas schema inspect [flags]
```

### Examples

```

atlas schema inspect -d mysql://user:pass@tcp(localhost:3306)/dbname
atlas schema inspect -d maria://user:pass@tcp(localhost:3306)/dbname
atlas schema inspect --dsn postgres://user:pass@host:port/dbname
atlas schema inspect -d sqlite://file:ex1.db?_fk=1
```

### Options

```
  -d, --dsn string   [driver://username:password@protocol(address)/dbname?param=value] Select data source using the dsn format
  -h, --help         help for inspect
  -w, --web          open in UI server
```

### SEE ALSO

* [atlas schema](atlas_schema.md)	 - Work with atlas schemas

