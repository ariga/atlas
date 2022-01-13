## atlas schema inspect

Inspect an atlas schema

```
atlas schema inspect [flags]
```

### Examples

```

atlas schema inspect -d "mysql://user:pass@tcp(localhost:3306)/dbname"
atlas schema inspect -d "mariadb://user:pass@tcp(localhost:3306)/dbname"
atlas schema inspect --dsn "postgres://user:pass@host:port/dbname"
atlas schema inspect -d "sqlite://file:ex1.db?_fk=1"
```

### Options

```
      --addr string   used with -w, local address to bind the server to (default "127.0.0.1:5800")
  -d, --dsn string    [driver://username:password@protocol(address)/dbname?param=value] Select data source using the dsn format
  -h, --help          help for inspect
  -w, --web           Open in a local Atlas UI
```

### SEE ALSO

* [atlas schema](atlas_schema.md)	 - Work with atlas schemas

