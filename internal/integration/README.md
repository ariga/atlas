### This directory contains all integration tests for Atlas.

The provided `docker-compose.yaml` file contains images for each database the integration tests are run on. You can
start them by calling:

```shell
docker-compose --project-name atlas-integration up -d
```

The whole integration suite is then run by executing within this directory: 

```shell
go test ./...
```

#### Selectively running tests

Running all integration tests (and keeping all database containers up all the time) consumes time and resources (and
power). You can execute only some of the tests by using the `-run` and `-dialect` flags:

The below examples don't require for you to have all docker containers running, instead only the ones used in the tests
have to be up.

Consider the following test in `mysql_test.go`:

```go
func TestMySQL_Executor(t *testing.T) {
	myRun(t, func(t *myTest) {
		testExecutor(t)
	})
}
```

If you'd wanted to run that test only for mysql56, simply pass its full name into the `-run` flag:

```shell
# Run TestMySQL_Executor for all mysql versions
go test -run='MySQL_Executor' ./... 

# Run TestMySQL_Executor for mysql 5.6 only
go test -run='MySQL_Executor/mysql56' ./...
```

If you'd like to run the above for Postgres 10, change the name respectively:

```shell
# Run TestPostgres_Executor for all postgres versions
go test -run='Postgres_Executor' ./... 

# Run TestPostgres_Executor for postgres 10 only
go test -run='Postgres_Executor/postgres10' ./...
```

If you want to run all tests for one specific dialect, like only TiDB 5, you can use the `-dialect` flag:

```shell
go test -run='TiDB' -dialect='tidb5' ./...
```