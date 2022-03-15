package docker

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	mysqld "github.com/go-sql-driver/mysql"
)

const defaultPass = "pass"

type (
	// Client is a Go docker client that can run and destroy local docker containers.
	Client struct {
		c *client.Client // underlying connection to docker engine
	}
	// Config is used to configure a container creation.
	Config struct {
		Image string
		Env   []string // env vars to pass to the container
		Port  nat.Port // internal port to expose and connect to
	}
	// A Container is an instance of a created container.
	Container struct {
		c   *Client // starting Client
		cfg Config  // Config used to create this container
		// ID of the container.
		ID string
		// Password of the root user.
		Password string
		// Port this containers service is bound to.
		Port string
	}
	// ConfigOption allows configuring Config with functional arguments.
	ConfigOption func(*Config) error
)

// NewClient returns a new Client.
func NewClient() (*Client, error) {
	c, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}
	return &Client{c: c}, nil
}

// NewConfig returns a new config with the given options applied.
func NewConfig(opts ...ConfigOption) (*Config, error) {
	c := &Config{}
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// MySQL returns a new Config for a MySQL image.
func MySQL(version string) (*Config, error) {
	return NewConfig(
		Image("mysql:"+version),
		Port("3306"),
		Env("MYSQL_ROOT_PASSWORD="+defaultPass),
	)
}

// Image sets the docker image to use.
//
// Example:
//   - Image("mysql")
//   - Image("postgres:13")
func Image(i string) ConfigOption {
	return func(c *Config) error {
		c.Image = i
		return nil
	}
}

// Port sets the port the container services exposes.
//
// Common configuration include:
//   - mysql: Port("3306")
//   - pg:    Port("5432")
func Port(p string) ConfigOption {
	return func(c *Config) (err error) {
		c.Port, err = nat.NewPort("tcp", p)
		return
	}
}

// Env sets the environment variables to pass to the container.
//
// Example:
//   - mysql: Env("MYSQL_ROOT_PASSWORD=password")
//   - pg:    Env("POSTGRES_PASSWORD=password")
func Env(env ...string) ConfigOption {
	return func(c *Config) error {
		c.Env = env
		return nil
	}
}

// Up pulls and starts a new docker container from the given image tag.
func (c *Client) Up(ctx context.Context, cfg *Config) (*Container, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	// Make sure the image is available
	rd, err := c.c.ImagePull(ctx, cfg.Image, types.ImagePullOptions{})
	_, err = io.Copy(os.Stdout, rd)
	if err != nil {
		return nil, err
	}
	// Get a free host TCP port the container can bind its exposed service port on.
	p, err := freePort()
	if err != nil {
		return nil, err
	}
	// Create the container.
	con, err := c.c.ContainerCreate(
		ctx,
		&container.Config{
			Image: cfg.Image,
			Env:   cfg.Env,
		},
		&container.HostConfig{
			PortBindings: nat.PortMap{
				cfg.Port: {{HostIP: "localhost", HostPort: p}},
			},
		},
		nil, nil, "",
	)
	if err != nil {
		return nil, err
	}
	if err := c.c.ContainerStart(ctx, con.ID, types.ContainerStartOptions{}); err != nil {
		return nil, err
	}
	return &Container{
		c:        c,
		cfg:      *cfg,
		ID:       con.ID,
		Password: defaultPass,
		Port:     p,
	}, nil
}

// Down stops and removes a running container.
func (c *Client) Down(ctx context.Context, ct *Container) error {
	// Don't care about the error. ContainerRemove will get rid of the container anyway.
	_ = c.c.ContainerStop(ctx, ct.ID, nil)
	return c.c.ContainerRemove(ctx, ct.ID, types.ContainerRemoveOptions{})
}

// Down stops and removes this container.
func (c *Container) Down(ctx context.Context) error {
	return c.c.Down(ctx, c)
}

// Wait waits for this container to be ready.
func (c *Container) Wait(ctx context.Context, timeout time.Duration) error {
	mysqld.SetLogger(log.New(ioutil.Discard, "", 1))
	defer mysqld.SetLogger(log.New(os.Stderr, "[mysql] ", log.Ldate|log.Ltime|log.Lshortfile))
	if timeout > time.Minute {
		timeout = time.Minute
	}
	done := time.After(timeout)
	for {
		select {
		case <-time.After(100 * time.Millisecond):
			db, err := sql.Open(c.Driver(), c.DSN())
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
func (c *Container) DSN() string {
	return fmt.Sprintf("root:%s@tcp(localhost:%s)/", c.Password, c.Port)
}

// validate that no empty values are given.
func (c *Config) validate() error {
	if c == nil || c.Image == "" || c.Port == "" {
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
	defer l.Close()
	return strconv.Itoa(l.Addr().(*net.TCPAddr).Port), nil
}
