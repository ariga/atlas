---
title: DSNs
id: cli-dsn
slug: /cli/dsn
---
## What is a DSN?

Data Source Names, or DSNs, are a standardized method for specifying a database connection.
All the component parts of a DSN are shown below:

```
driver://[username[:password]@][protocol[(address)]]/dbname[?param1=value1&...&paramN=valueN]
```

An example might look like:

```
mysql://user:pass@tcp(localhost:3306)/dbname
```

The above example contains the following information:

- Driver: `mysql`
- Username: `user`
- Password: `pass`
- Protocol: `tcp`
- Host: `localhost`
- Port: `3306`
- Database: `dbname`

## SSL/TLS Mode

The default SSL mode for Postgres is `required`. Please follow the 
[Postgres documentation](https://www.postgresql.org/docs/current/libpq-ssl.html)
for configuring your SSL connection for your database, or set SSL mode to `disable`
with the search parameter `?sslmode=disable`. For local databases,
disabling SSL is appropriate when inspecting and applying schema changes.

MySQL does not require TSL by default. However, you can require TLS
with the `?tls=true` search parameter.
