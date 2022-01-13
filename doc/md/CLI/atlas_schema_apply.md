## atlas schema apply

Apply an atlas schema to a data source

```
atlas schema apply [flags]
```

### Examples

```

atlas schema apply -d "mysql://user:pass@tcp(localhost:3306)/dbname" -f atlas.hcl
atlas schema apply -d "mariadb://user:pass@tcp(localhost:3306)/dbname" -f atlas.hcl
atlas schema apply --dsn "postgres://user:pass@host:port/dbname" -f atlas.hcl
atlas schema apply -d "sqlite://file:ex1.db?_fk=1" -f atlas.hcl
```

### Options

```
      --addr string   used with -w, local address to bind the server to (default "127.0.0.1:5800")
  -d, --dsn string    [driver://username:password@protocol(address)/dbname?param=value] Select data source using the dsn format
  -f, --file string   [/path/to/file] file containing schema
  -h, --help          help for apply
  -w, --web           Open in a local Atlas UI
```

### SEE ALSO

* [atlas schema](atlas_schema.md)	 - Work with atlas schemas

