// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package docker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"ariga.io/atlas/sql/sqlclient"

	"github.com/go-sql-driver/mysql"
)

const (
	pass          = "pass"
	passSQLServer = "P@ssw0rd0995"
)

type (
	// Config is used to configure container creation.
	Config struct {
		driver string   // driver to open connections with.
		setup  []string // contains statements to execute once the service is up
		// Image is the name of the image to pull and run.
		Image string
		// Env vars to pass to the docker container.
		Env []string
		// Internal Port to expose anc connect to.
		Port string
		// Database name to create and connect on init.
		Database string
		// Out is a custom writer to send docker cli output to.
		Out io.Writer
	}
	// A Container is an instance of a created container.
	Container struct {
		cfg Config    // Config used to create this container
		out io.Writer // custom write to log status messages to
		// ID of the container.
		ID string
		// Passphrase of the root user.
		Passphrase string
		// Port on the host this containers service is bound to.
		Port string
	}
	// ConfigOption allows configuring Config with functional arguments.
	ConfigOption func(*Config) error
)

// NewConfig returns a new config with the given options applied.
func NewConfig(opts ...ConfigOption) (*Config, error) {
	c := &Config{Out: io.Discard}
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// Well-known DB drivers
const (
	DriverMySQL      = "mysql"
	DriverMariaDB    = "mariadb"
	DriverPostgres   = "postgres"
	DriverSQLServer  = "sqlserver"
	DriverClickHouse = "clickhouse"
)

// FromURL parses a URL in the format of
// "docker://driver/tag[/dbname]" and returns a Config.
func FromURL(u *url.URL, opts ...ConfigOption) (*Config, error) {
	var (
		parts  = strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
		idxTag = len(parts) - 1
		dbName string
	)
	// Check if the last part is a tag or a database name.
	if idxTag > 0 && !strings.ContainsRune(parts[idxTag], ':') {
		// The last part is not a tag, so it must be a database name.
		dbName, idxTag = parts[idxTag], idxTag-1
	}
	var baseOpts []ConfigOption
	var tag string
	// Support docker+driver://<image>[:<tag>]
	driver, customImage := strings.CutPrefix(u.Scheme, "docker+")
	if customImage {
		// The image is fully specified in the URL.
		img := path.Join(parts[:idxTag+1]...)
		if u.Host != "" && u.Host != "_" {
			img = path.Join(u.Host, img)
		}
		baseOpts = append(baseOpts, Image(img))
	} else {
		driver = u.Host
		if idxTag >= 0 {
			tag = parts[idxTag]
		}
	}
	var (
		err error
		cfg *Config
	)
	switch driver {
	case DriverMySQL:
		if dbName != "" {
			baseOpts = append(baseOpts,
				Database(dbName),
				Env("MYSQL_DATABASE="+dbName),
				Setup(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", dbName)),
			)
		}
		cfg, err = MySQL(tag, append(baseOpts, opts...)...)
	case "maria":
		driver = DriverMariaDB
		fallthrough
	case DriverMariaDB:
		if dbName != "" {
			baseOpts = append(baseOpts,
				Database(dbName),
				Env("MYSQL_DATABASE="+dbName),
				Setup(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", dbName)),
			)
		}
		cfg, err = MariaDB(tag, append(baseOpts, opts...)...)
	case "postgis":
		baseOpts = append(baseOpts, Image("postgis/postgis:"+tag))
		if dbName != "" && dbName != "postgres" {
			// Create manually the PostgreSQL database instead of using the POSTGRES_DB because
			// PostGIS automatically creates and install the following extensions and schemas:
			// Schemas: tiger, tiger_data, topology.
			// Extensions: postgis, postgis_topology, postgis_tiger_geocoder.
			baseOpts = append(
				baseOpts, Database(dbName), Setup(fmt.Sprintf("CREATE DATABASE %q", dbName)),
			)
		}
		driver = DriverPostgres
		cfg, err = PostgreSQL(tag, append(baseOpts, opts...)...)
	case DriverPostgres:
		if dbName != "" {
			baseOpts = append(baseOpts, Database(dbName), Env("POSTGRES_DB="+dbName))
		}
		cfg, err = PostgreSQL(tag, append(baseOpts, opts...)...)
	case DriverSQLServer:
		if dbName != "" && dbName != "master" {
			baseOpts = append(baseOpts,
				Database(dbName),
				Setup(fmt.Sprintf("CREATE DATABASE [%s]", dbName)),
			)
		}
		cfg, err = SQLServer(tag, append(baseOpts, opts...)...)
	case DriverClickHouse:
		if dbName != "" {
			baseOpts = append(baseOpts,
				Database(dbName),
				Env("CLICKHOUSE_DB="+dbName),
				Setup(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", dbName)),
			)
		}
		cfg, err = ClickHouse(tag, append(baseOpts, opts...)...)
	default:
		return nil, fmt.Errorf("unsupported docker image %q", driver)
	}
	if err != nil {
		return nil, err
	}
	cfg.driver = driver
	return cfg, nil
}

// ImageURL returns the base URL for the given driver and image.
func ImageURL(driver string, image string) (*url.URL, error) {
	switch {
	case driver == "" && image == "":
		return nil, errors.New("driver and image cannot be empty")
	case driver == "":
		return nil, errors.New("driver cannot be empty")
	case image == "":
		return nil, errors.New("image cannot be empty")
	default:
		u := &url.URL{Scheme: "docker+" + driver, Host: "_", Path: image}
		if idx := strings.IndexByte(image, '/'); idx != -1 {
			u.Host, u.Path = image[:idx], image[idx+1:]
		}
		return u, nil
	}
}

// Atlas DockerHub user contains the MySQL
// and MariaDB images with faster boot times.
const hubUser = "arigaio"

// MySQL returns a new Config for a MySQL image.
func MySQL(version string, opts ...ConfigOption) (*Config, error) {
	return NewConfig(
		append(
			[]ConfigOption{
				Image(hubUser, "mysql:"+version),
				Port("3306"),
				Env("MYSQL_ROOT_PASSWORD=" + pass),
			},
			opts...,
		)...,
	)
}

// MariaDB returns a new Config for a MariaDB image.
func MariaDB(version string, opts ...ConfigOption) (*Config, error) {
	return MySQL(version, append([]ConfigOption{Image(hubUser, "mariadb:"+version)}, opts...)...)
}

// PostgreSQL returns a new Config for a PostgreSQL image.
func PostgreSQL(version string, opts ...ConfigOption) (*Config, error) {
	return NewConfig(
		append(
			[]ConfigOption{
				Image("postgres:" + version),
				Port("5432"),
				Database("postgres"),
				Env("POSTGRES_PASSWORD=" + pass),
			},
			opts...,
		)...,
	)
}

// SQLServer returns a new Config for a SQLServer image.
func SQLServer(version string, opts ...ConfigOption) (*Config, error) {
	return NewConfig(
		append(
			[]ConfigOption{
				Image("mcr.microsoft.com/mssql/server:" + version),
				Port("1433"),
				Database("master"),
				Env(
					"ACCEPT_EULA=Y",
					"MSSQL_PID=Developer",
					"MSSQL_SA_PASSWORD="+passSQLServer,
				),
			},
			opts...,
		)...,
	)
}

// ClickHouse returns a new Config for a ClickHouse image.
func ClickHouse(version string, opts ...ConfigOption) (*Config, error) {
	return NewConfig(
		append(
			[]ConfigOption{
				Image("clickhouse/clickhouse-server:" + version),
				Port("9000"),
				Env("CLICKHOUSE_PASSWORD=" + pass),
			},
			opts...,
		)...,
	)
}

// Image sets the docker image to use. For example:
//
//	Image("mysql")
//	Image("arigaio", "mysql")
//	Image("postgres:13")
func Image(parts ...string) ConfigOption {
	return func(c *Config) error {
		if len(parts) == 0 || len(parts) > 2 {
			return errors.New("image path must be in the format of 'image' or 'user/image'")
		}
		c.Image = strings.TrimSuffix(path.Join(parts...), ":")
		return nil
	}
}

// Port sets the port the container services exposes. For example:
//
//	Port("3306")
//	Port("5432")
func Port(p string) ConfigOption {
	return func(c *Config) error {
		c.Port = p
		return nil
	}
}

// Env sets the environment variables to pass to the container. For example:
//
//	Config(Image("mysql"), Env("MYSQL_ROOT_PASSWORD=password"))
//	Config(Image("postgres"), Env("MYSQL_ROOT_PASSWORD=password"))
func Env(env ...string) ConfigOption {
	return func(c *Config) error {
		c.Env = append(c.Env, env...)
		return nil
	}
}

// Database sets the database name to connect to. For example:
func Database(name string) ConfigOption {
	return func(c *Config) error {
		c.Database = name
		return nil
	}
}

// Out sets an io.Writer to use when running docker commands. For example:
//
//	buf := new(bytes.Buffer)
//	NewConfig(Out(buf))
func Out(w io.Writer) ConfigOption {
	return func(c *Config) error {
		c.Out = w
		return nil
	}
}

// Setup adds statements to execute once the service is ready. For example:
//
//	setup("CREATE DATABASE IF NOT EXISTS test")
//	setup("DROP SCHEMA IF EXISTS public CASCADE")
func Setup(s ...string) ConfigOption {
	return func(c *Config) error {
		c.setup = append(c.setup, s...)
		return nil
	}
}

// Run pulls and starts a new docker container from the Config.
func (c *Config) Run(ctx context.Context) (*Container, error) {
	// Make sure the configuration is not missing critical values.
	if err := c.validate(); err != nil {
		return nil, err
	}
	// Get a free host TCP port the container can bind its exposed service port on.
	p, err := freePort()
	if err != nil {
		return nil, fmt.Errorf("getting open port: %w", err)
	}
	// Run the container.
	args := []string{"docker", "run", "--rm", "--detach"}
	for _, e := range c.Env {
		args = append(args, "-e", e)
	}
	args = append(args, "-p", fmt.Sprintf("%s:%s", p, c.Port), c.Image)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...) //nolint:gosec
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	cmd.Stdout, cmd.Stderr = io.MultiWriter(c.Out, stdout), stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			err = errors.New(strings.TrimSpace(stderr.String()))
		}
		return nil, err
	}
	return &Container{
		cfg:        *c,
		ID:         strings.TrimSpace(stdout.String()),
		Passphrase: pass,
		Port:       p,
		out:        c.Out,
	}, nil
}

// Close stops and removes this container.
func (c *Container) Close() error {
	return exec.Command("docker", "kill", c.ID).Run() //nolint:gosec
}

// Wait waits for this container to be ready.
func (c *Container) Wait(ctx context.Context, timeout time.Duration) error {
	fmt.Fprintln(c.out, "Waiting for service to be ready ... ")
	mysql.SetLogger(log.New(io.Discard, "", 1))
	defer mysql.SetLogger(log.New(os.Stderr, "[mysql] ", log.Ldate|log.Ltime|log.Lshortfile))
	if timeout > time.Minute {
		timeout = time.Minute
	}
	var (
		done   = time.After(timeout)
		u, err = c.URL()
	)
	if err != nil {
		return err
	}
	pingURL := c.PingURL(*u)
	for {
		select {
		case <-time.After(100 * time.Millisecond):
			var client *sqlclient.Client
			// Ping against the root connection.
			client, err = sqlclient.Open(ctx, pingURL)
			if err != nil {
				continue
			}
			db := client.DB
			if err = db.PingContext(ctx); err != nil {
				continue
			}
			for _, s := range c.cfg.setup {
				if _, err := db.ExecContext(ctx, s); err != nil {
					err = errors.Join(err, db.Close())
					return fmt.Errorf("%q: %w", s, err)
				}
			}
			_ = db.Close()
			fmt.Fprintln(c.out, "Service is ready to connect!")
			return nil
		case <-ctx.Done():
			return ctx.Err()
		case <-done:
			if err != nil {
				return fmt.Errorf("timeout: %w", err)
			}
			return errors.New("timeout")
		}
	}
}

// URL returns a URL to connect to the Container.
func (c *Container) URL() (*url.URL, error) {
	host := "localhost"
	// Check if the DOCKER_HOST env var is set.
	// If it is, use the host from the URL.
	if h := os.Getenv("DOCKER_HOST"); h != "" {
		u, err := url.Parse(h)
		if err != nil {
			return nil, err
		}
		host = u.Hostname()
	}
	switch c.cfg.driver {
	case DriverClickHouse:
		return url.Parse(fmt.Sprintf("clickhouse://:%s@%s:%s/%s", c.Passphrase, host, c.Port, c.cfg.Database))
	case DriverSQLServer:
		return url.Parse(fmt.Sprintf("sqlserver://sa:%s@%s:%s?database=%s", passSQLServer, host, c.Port, c.cfg.Database))
	case DriverPostgres:
		return url.Parse(fmt.Sprintf("postgres://postgres:%s@%s:%s/%s?sslmode=disable", c.Passphrase, host, c.Port, c.cfg.Database))
	case DriverMySQL, DriverMariaDB:
		return url.Parse(fmt.Sprintf("%s://root:%s@%s:%s/%s", c.cfg.driver, c.Passphrase, host, c.Port, c.cfg.Database))
	default:
		return nil, fmt.Errorf("unknown driver: %q", c.cfg.driver)
	}
}

// PingURL returns a URL to ping the Container.
func (c *Container) PingURL(u url.URL) string {
	switch c.cfg.driver {
	case DriverSQLServer:
		q := u.Query()
		q.Del("database")
		u.RawQuery = q.Encode()
		return u.String()
	default:
		u.Path = "/"
		return u.String()
	}
}

// validate that no empty values are given.
func (c *Config) validate() error {
	if c == nil || c.Image == "" || c.Port == "" || c.Out == nil {
		return fmt.Errorf("invalid configuration %q", c)
	}
	return nil
}

func freePort() (string, error) {
	a, err := net.ResolveTCPAddr("tcp", ":0")
	if err != nil {
		return "", err
	}
	l, err := net.ListenTCP("tcp", a)
	if err != nil {
		return "", err
	}
	if err := l.Close(); err != nil {
		return "", err
	}
	return strconv.Itoa(l.Addr().(*net.TCPAddr).Port), nil
}

func init() {
	sqlclient.Register(
		"docker",
		sqlclient.OpenerFunc(Open),
		sqlclient.RegisterFlavours(
			"docker+postgres", "docker+mysql", "docker+maria", "docker+mariadb",
			"docker+sqlserver", "docker+clickhouse",
		),
	)
}

// Open a new docker client.
func Open(ctx context.Context, u *url.URL) (client *sqlclient.Client, err error) {
	return OpenWithOpts(ctx, u)
}

// OpenWithOpts open a new docker client
func OpenWithOpts(ctx context.Context, u *url.URL, opts ...ConfigOption) (client *sqlclient.Client, err error) {
	cfg, err := FromURL(u, opts...)
	if err != nil {
		return nil, err
	}
	qr := u.Query()
	if qr.Has("v") || qr.Has("verbose") {
		if err := Out(os.Stdout)(cfg); err != nil {
			return nil, err
		}
		qr.Del("v")
		qr.Del("verbose")
	}
	c, err := cfg.Run(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			if cerr := c.Close(); err != nil {
				err = fmt.Errorf("%w: %v", err, cerr)
			}
		}
	}()
	if err = c.Wait(ctx, time.Minute); err != nil {
		return nil, err
	}
	u1, err := c.URL()
	if err != nil {
		return nil, err
	}
	for k, v := range u1.Query() {
		qr[k] = v
	}
	u1.RawQuery = qr.Encode()
	if client, err = sqlclient.Open(ctx, u1.String()); err != nil {
		return nil, err
	}
	client.AddClosers(c)
	return client, nil
}
