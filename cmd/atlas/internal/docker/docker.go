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

const pass = "pass"

type (
	// Config is used to configure container creation.
	Config struct {
		setup []string // contains statements to execute once the service is up
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

// FromURL parses a URL in the format of
// "docker://image/tag" and returns a Config.
func FromURL(u *url.URL) (*Config, error) {
	var (
		tag   string
		opts  []ConfigOption
		parts = strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	)
	if len(parts) > 0 {
		tag = parts[0]
	}
	if len(parts) > 1 {
		opts = append(opts, Database(parts[1]))
	}
	switch u.Host {
	case "mysql":
		if len(parts) > 1 {
			opts = append(opts, Env("MYSQL_DATABASE="+parts[1]), setup(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", parts[1])))
		}
		return MySQL(tag, opts...)
	case "maria", "mariadb":
		if len(parts) > 1 {
			opts = append(opts, Env("MYSQL_DATABASE="+parts[1]), setup(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", parts[1])))
		}
		return MariaDB(tag, opts...)
	case "postgres":
		if len(parts) > 1 {
			opts = append(opts, Env("POSTGRES_DB="+parts[1]))
		}
		return PostgreSQL(tag, opts...)
	case "postgis":
		if len(parts) > 1 {
			opts = append(opts, Env("POSTGRES_DB="+parts[1]))
		}
		return Postgis(tag, opts...)
	default:
		return nil, fmt.Errorf("unsupported docker image %q", u.Host)
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

// Postgis returns a new Config for a Postgis image. This is a Postgres variant
func Postgis(version string, opts ...ConfigOption) (*Config, error) {
	return NewConfig(
		append(
			[]ConfigOption{
				Image("postgis/postgis:" + version),
				Port("5432"),
				Database("postgres"),
				Env("POSTGRES_PASSWORD=" + pass),
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

// setup adds statements to execute once the service is ready. For example:
//
//	setup("CREATE DATABASE IF NOT EXISTS test")
//	setup("DROP SCHEMA IF EXISTS public CASCADE")
func setup(s ...string) ConfigOption {
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
	// Make sure the image is up-to-date.
	out, err := exec.CommandContext(ctx, "docker", "pull", c.Image).CombinedOutput() //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("pulling image: %w: %q", err, out)
	}
	// Run the container.
	args := []string{"docker", "run", "--rm", "--detach"}
	for _, e := range c.Env {
		args = append(args, "-e", e)
	}
	args = append(args, "-p", fmt.Sprintf("%s:%s", p, c.Port), c.Image)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...) //nolint:gosec
	buf := &bytes.Buffer{}
	cmd.Stdout = io.MultiWriter(c.Out, buf)
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return &Container{
		cfg:        *c,
		ID:         strings.TrimSpace(buf.String()),
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
	for {
		select {
		case <-time.After(100 * time.Millisecond):
			// Ping against the root connection.
			u.Path = "/"
			client, err := sqlclient.Open(ctx, u.String())
			if err != nil {
				continue
			}
			db := client.DB
			if err := db.PingContext(ctx); err != nil {
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
			return errors.New("timeout")
		}
	}
}

// URL returns a URL to connect to the Container.
func (c *Container) URL() (*url.URL, error) {
	switch img := path.Base(strings.SplitN(c.cfg.Image, ":", 2)[0]); img {
	case "postgres", "postgis":
		return url.Parse(fmt.Sprintf("postgres://postgres:%s@localhost:%s/%s?sslmode=disable", c.Passphrase, c.Port, c.cfg.Database))
	case "mysql", "mariadb":
		return url.Parse(fmt.Sprintf("%s://root:%s@localhost:%s/%s", img, c.Passphrase, c.Port, c.cfg.Database))
	default:
		return nil, fmt.Errorf("unknown container image: %q", img)
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
	sqlclient.Register("docker", sqlclient.OpenerFunc(client))
}

func client(ctx context.Context, u *url.URL) (client *sqlclient.Client, err error) {
	cfg, err := FromURL(u)
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
