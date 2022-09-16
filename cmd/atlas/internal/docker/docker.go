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
	"io/ioutil"
	"log"
	"net"
	"net/url"
	"os"
	"os/exec"
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
	c := &Config{Out: ioutil.Discard}
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// MySQL returns a new Config for a MySQL image.
func MySQL(version string, opts ...ConfigOption) (*Config, error) {
	return NewConfig(
		append(
			[]ConfigOption{
				Image("mysql:" + version),
				Port("3306"),
				Env("MYSQL_ROOT_PASSWORD=" + pass),
			},
			opts...,
		)...,
	)
}

// MariaDB returns a new Config for a MariaDB image.
func MariaDB(version string, opts ...ConfigOption) (*Config, error) {
	return MySQL(version, append([]ConfigOption{Image("mariadb:" + version)}, opts...)...)
}

// PostgreSQL returns a new Config for a PostgreSQL image.
func PostgreSQL(version string, opts ...ConfigOption) (*Config, error) {
	return NewConfig(
		append(
			[]ConfigOption{
				Image("postgres:" + version),
				Port("5432"),
				Env("POSTGRES_PASSWORD=" + pass),
			},
			opts...,
		)...,
	)
}

// Image sets the docker image to use. For example:
//
//	Image("mysql")
//	Image("postgres:13")
func Image(i string) ConfigOption {
	return func(c *Config) error {
		c.Image = strings.TrimSuffix(i, ":")
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
		c.Env = env
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
//	setup("DROP SCHEMA IF EXISTS public CASCADE;")
func setup(s ...string) ConfigOption {
	return func(c *Config) error {
		c.setup = s
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
	cmd := exec.CommandContext(ctx, "docker", "pull", c.Image) //nolint:gosec
	cmd.Stdout = c.Out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pulling image: %w", err)
	}
	// Run the container.
	args := []string{"docker", "run", "--rm", "--detach"}
	for _, e := range c.Env {
		args = append(args, "-e", e)
	}
	args = append(args, "-p", fmt.Sprintf("%s:%s", p, c.Port), c.Image)
	cmd = exec.CommandContext(ctx, args[0], args[1:]...) //nolint:gosec
	out := &bytes.Buffer{}
	cmd.Stdout = io.MultiWriter(c.Out, out)
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return &Container{
		cfg:        *c,
		ID:         strings.TrimSpace(out.String()),
		Passphrase: pass,
		Port:       p,
		out:        c.Out,
	}, nil
}

// Close stops and removes this container.
func (c *Container) Close() error {
	return exec.Command("docker", "stop", c.ID).Run() //nolint:gosec
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
			client, err := sqlclient.Open(ctx, u)
			if err != nil {
				continue
			}
			db := client.DB
			if err := db.PingContext(ctx); err != nil {
				continue
			}
			for _, s := range c.cfg.setup {
				if _, err := db.ExecContext(ctx, s); err != nil {
					_ = db.Close()
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
func (c *Container) URL() (string, error) {
	switch img := strings.SplitN(c.cfg.Image, ":", 2)[0]; img {
	case "postgres":
		return fmt.Sprintf("postgres://postgres:%s@localhost:%s/postgres?sslmode=disable", c.Passphrase, c.Port), nil
	case "mysql", "mariadb":
		return fmt.Sprintf("%s://root:%s@localhost:%s/", img, c.Passphrase, c.Port), nil
	default:
		return "", fmt.Errorf("unknown container image: %q", img)
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
	var cfg *Config
	switch img, tag := u.Host, strings.TrimPrefix(u.Path, "/"); img {
	case "mysql":
		cfg, err = MySQL(tag)
	case "mariadb":
		cfg, err = MariaDB(tag)
	case "postgres":
		cfg, err = PostgreSQL(tag)
	default:
		return nil, fmt.Errorf("unsupported docker image %q", img)
	}
	if err != nil {
		return nil, err
	}
	if u.Query().Has("v") || u.Query().Has("verbose") {
		if err := Out(os.Stdout)(cfg); err != nil {
			return nil, err
		}
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
	if client, err = sqlclient.Open(ctx, u1); err != nil {
		return nil, err
	}
	client.AddClosers(c)
	return client, nil
}
