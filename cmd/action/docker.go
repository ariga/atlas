// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package action

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	mysqld "github.com/go-sql-driver/mysql"
)

const pass = "pass"

type (
	// DockerConfig is used to configure container creation.
	DockerConfig struct {
		Image string
		Env   []string  // env vars to pass to the container
		Port  string    // internal port to expose and connect to
		Out   io.Writer // custom writer to send docker cli output to
	}
	// A Container is an instance of a created container.
	Container struct {
		cfg DockerConfig // DockerConfig used to create this container
		// ID of the container.
		ID string
		// Passphrase of the root user.
		Passphrase string
		// Port on the host this containers service is bound to.
		Port string
	}
	// DockerConfigOption allows configuring DockerConfig with functional arguments.
	DockerConfigOption func(*DockerConfig) error
)

// NewConfig returns a new config with the given options applied.
func NewConfig(opts ...DockerConfigOption) (*DockerConfig, error) {
	c := &DockerConfig{Out: os.Stdout}
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// MySQL returns a new DockerConfig for a MySQL image.
func MySQL(version string, opts ...DockerConfigOption) (*DockerConfig, error) {
	return NewConfig(append(
		[]DockerConfigOption{
			Image("mysql:" + version),
			Port("3306"),
			Env("MYSQL_ROOT_PASSWORD=" + pass),
		},
		opts...,
	)...,
	)
}

// Image sets the docker image to use. For example:
//
//	Image("mysql")
//	Image("postgres:13")
//
func Image(i string) DockerConfigOption {
	return func(c *DockerConfig) error {
		c.Image = i
		return nil
	}
}

// Port sets the port the container services exposes. For example:
//
//	Port("3306")
//	Port("5432")
//
func Port(p string) DockerConfigOption {
	return func(c *DockerConfig) error {
		c.Port = p
		return nil
	}
}

// Env sets the environment variables to pass to the container. For example:
//
// 	Config(Image("mysql"), Env("MYSQL_ROOT_PASSWORD=password"))
// 	Config(Image("postgres"), Env("MYSQL_ROOT_PASSWORD=password"))
//
func Env(env ...string) DockerConfigOption {
	return func(c *DockerConfig) error {
		c.Env = env
		return nil
	}
}

// Out sets an io.Writer to use when running docker commands. For example:
//
// 	buf := new(bytes.Buffer)
// 	NewConfig(Out(buf))
//
func Out(w io.Writer) DockerConfigOption {
	return func(c *DockerConfig) error {
		c.Out = w
		return nil
	}
}

// Run pulls and starts a new docker container from the DockerConfig.
func (c *DockerConfig) Run(ctx context.Context) (*Container, error) {
	// Make sure the configuration is not missing critical values.
	if err := c.validate(); err != nil {
		return nil, err
	}
	// Get a free host TCP port the container can bind its exposed service port on.
	p, err := freePort()
	if err != nil {
		return nil, err
	}
	// Make sure the image is up-to-date.
	cmd := exec.CommandContext(ctx, "docker", "pull", c.Image) //nolint:gosec
	cmd.Stdout = c.Out
	if err := cmd.Run(); err != nil {
		return nil, err
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
	}, nil
}

// Down stops and removes this container.
func (c *Container) Down(ctx context.Context) error {
	return exec.CommandContext(ctx, "docker", "stop", c.ID).Run() //nolint:gosec
}

// Wait waits for this container to be ready.
func (c *Container) Wait(ctx context.Context, timeout time.Duration) error {
	mysqld.SetLogger(log.New(ioutil.Discard, "", 1))
	defer mysqld.SetLogger(log.New(os.Stderr, "[mysql] ", log.Ldate|log.Ltime|log.Lshortfile))
	if timeout > time.Minute {
		timeout = time.Minute
	}
	var (
		done     = time.After(timeout)
		drv      = c.Driver()
		dsn, err = c.DSN()
	)
	if err != nil {
		return err
	}
	for {
		select {
		case <-time.After(100 * time.Millisecond):
			db, err := sql.Open(drv, dsn)
			if err != nil {
				continue
			}
			if err := db.PingContext(ctx); err != nil {
				continue
			}
			return nil
		case <-ctx.Done():
			return ctx.Err()
		case <-done:
			return errors.New("timeout")
		}
	}
}

// Driver returns the database/sql driver name.
func (c *Container) Driver() string {
	return strings.SplitN(c.cfg.Image, ":", 2)[0]
}

// DSN returns a DSN to connect to the Container.
func (c *Container) DSN() (string, error) {
	switch drv := c.Driver(); drv {
	case "mysql", "maria", "mariadb":
		return fmt.Sprintf("root:%s@tcp(localhost:%s)/", c.Passphrase, c.Port), nil
	default:
		return "", fmt.Errorf("unknown driver: %q", drv)
	}
}

// validate that no empty values are given.
func (c *DockerConfig) validate() error {
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
	l.Close()
	return strconv.Itoa(l.Addr().(*net.TCPAddr).Port), nil
}
