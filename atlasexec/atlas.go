package atlasexec

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"sync"
)

type (
	// Client is a client for the Atlas CLI.
	Client struct {
		execPath   string
		workingDir string
		env        Environ
		stdout     io.Writer
		stderr     io.Writer
	}
	// LoginParams are the parameters for the `login` command.
	LoginParams struct {
		Token string
	}
	// WhoAmIParams are the parameters for the `whoami` command
	WhoAmIParams struct {
		ConfigURL string
		Env       string
		Vars      VarArgs
	}
	// WhoAmI contains the result of an 'atlas whoami' run.
	WhoAmI struct {
		Org string `json:"Org,omitempty"`
	}
	// Version contains the result of an 'atlas version' run.
	Version struct {
		Version string `json:"Version"`
		SHA     string `json:"SHA,omitempty"`
		Canary  bool   `json:"Canary,omitempty"`
	}
	// VarArgs is a map of variables for the command.
	VarArgs interface {
		// AsArgs returns the variables as arguments.
		AsArgs() []string
	}
	// Vars2 is a map of variables for the command.
	// It supports multiple values for the same key (list).
	Vars2 map[string]any
	// Environ is a map of environment variables.
	Environ map[string]string
	// RunContext is an input type for describing the context of where the
	// command is triggered from. For example, a GitHub Action on the master branch.
	RunContext struct {
		Repo     string  `json:"repo,omitempty"`
		Path     string  `json:"path,omitempty"`
		Branch   string  `json:"branch,omitempty"`
		Commit   string  `json:"commit,omitempty"`
		URL      string  `json:"url,omitempty"`
		Username string  `json:"username,omitempty"` // The username that triggered the event that initiated the command.
		UserID   string  `json:"userID,omitempty"`   // The user ID that triggered the event that initiated the command.
		SCMType  SCMType `json:"scmType,omitempty"`  // Source control management system type.
	}
	// SCMType is a type for the "scm_type" enum field.
	SCMType string
	// DeployRunContext is an input type for describing the context in which
	// `migrate-apply` and `migrate down` were used. For example, a GitHub Action with version v1.2.3
	DeployRunContext struct {
		TriggerType    TriggerType `json:"triggerType,omitempty"`
		TriggerVersion string      `json:"triggerVersion,omitempty"`
	}
	// TriggerType defines the type for the "trigger_type" enum field.
	TriggerType string
	// Vars is a map of variables for the command.
	//
	// Deprecated: Use Vars2 instead.
	Vars map[string]string
)

// TriggerType values.
const (
	TriggerTypeCLI          TriggerType = "CLI"
	TriggerTypeKubernetes   TriggerType = "KUBERNETES"
	TriggerTypeTerraform    TriggerType = "TERRAFORM"
	TriggerTypeGithubAction TriggerType = "GITHUB_ACTION"
	TriggerTypeCircleCIOrb  TriggerType = "CIRCLECI_ORB"
	TriggerTypeGitlab       TriggerType = "GITLAB"
	TriggerTypeBitbucket    TriggerType = "BITBUCKET"
	TriggerTypeAzureDevOps  TriggerType = "AZURE_DEVOPS"
)

// SCMType values.
const (
	SCMTypeGithub      SCMType = "GITHUB"
	SCMTypeGitlab      SCMType = "GITLAB"
	SCMTypeBitbucket   SCMType = "BITBUCKET"
	SCMTypeAzureDevOps SCMType = "AZURE_DEVOPS"
)

// ExecutionOrder values.
const (
	ExecOrderLinear     MigrateExecOrder = "linear" // Default
	ExecOrderLinearSkip MigrateExecOrder = "linear-skip"
	ExecOrderNonLinear  MigrateExecOrder = "non-linear"
)

// NewClient returns a new Atlas client with the given atlas-cli path.
func NewClient(workingDir, execPath string) (_ *Client, err error) {
	if execPath == "" {
		return nil, fmt.Errorf("execPath cannot be empty")
	} else if execPath, err = exec.LookPath(execPath); err != nil {
		return nil, fmt.Errorf("looking up atlas-cli: %w", err)
	}
	if workingDir != "" {
		_, err := os.Stat(workingDir)
		if err != nil {
			return nil, fmt.Errorf("initializing Atlas with working dir %q: %w", workingDir, err)
		}
	}
	return &Client{
		execPath:   execPath,
		workingDir: workingDir,
	}, nil
}

// WithWorkDir creates a new client with the given working directory.
// It is useful to run multiple commands in the multiple directories.
//
// Example:
//
//	client := atlasexec.NewClient("", "atlas")
//	err := client.WithWorkDir("dir1", func(c *atlasexec.Client) error {
//	  _, err := c.MigrateApply(ctx, &atlasexec.MigrateApplyParams{
//	  })
//	  return err
//	})
func (c *Client) WithWorkDir(dir string, fn func(*Client) error) error {
	wd := c.workingDir
	defer func() { c.workingDir = wd }()
	c.workingDir = dir
	return fn(c)
}

// SetEnv allows we override the environment variables for the atlas-cli.
// To append new environment variables to environment from OS, use NewOSEnviron() then add new variables.
func (c *Client) SetEnv(env map[string]string) error {
	for k := range env {
		if _, ok := defaultEnvs[k]; ok {
			return fmt.Errorf("atlasexec: cannot override the default environment variable %q", k)
		}
	}
	c.env = env
	return nil
}

// SetStdout specifies a writer to stream stdout to for every command.
func (c *Client) SetStdout(w io.Writer) {
	c.stdout = w
}

// SetStderr specifies a writer to stream stderr to for every command.
func (c *Client) SetStderr(w io.Writer) {
	c.stderr = w
}

// Login runs the 'login' command.
func (c *Client) Login(ctx context.Context, params *LoginParams) error {
	if params.Token == "" {
		return errors.New("token cannot be empty")
	}
	_, err := c.runCommand(ctx, []string{"login", "--token", params.Token})
	return err
}

// Logout runs the 'logout' command.
func (c *Client) Logout(ctx context.Context) error {
	_, err := c.runCommand(ctx, []string{"logout"})
	return err
}

// WhoAmI runs the 'whoami' command.
func (c *Client) WhoAmI(ctx context.Context, params *WhoAmIParams) (*WhoAmI, error) {
	args := []string{"whoami", "--format", "{{ json . }}"}
	// Global flags
	if params.ConfigURL != "" {
		args = append(args, "--config", params.ConfigURL)
	}
	if params.Env != "" {
		args = append(args, "--env", params.Env)
	}
	if params.Vars != nil {
		args = append(args, params.Vars.AsArgs()...)
	}
	return firstResult(jsonDecode[WhoAmI](c.runCommand(ctx, args)))
}

var reVersion = regexp.MustCompile(`^atlas version v(\d+\.\d+.\d+)-?([a-z0-9]*)?`)

// Version runs the 'version' command.
func (c *Client) Version(ctx context.Context) (*Version, error) {
	r, err := c.runCommand(ctx, []string{"version"})
	if err != nil {
		return nil, err
	}
	out, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	v := reVersion.FindSubmatch(out)
	if v == nil {
		return nil, errors.New("unexpected output format")
	}
	var sha string
	if len(v) > 2 {
		sha = string(v[2])
	}
	return &Version{
		Version: string(v[1]),
		SHA:     sha,
		Canary:  strings.Contains(string(out), "canary"),
	}, nil
}

// var reVersion = regexp.MustCompile(`^atlas version v(\d+\.\d+.\d+)-?([a-z0-9]*)?`)
func (v Version) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "atlas version v%s", v.Version)
	if v.SHA != "" {
		fmt.Fprintf(&b, "-%s", v.SHA)
	}
	if v.Canary {
		b.WriteString("-canary")
	}
	return b.String()
}

// NewOSEnviron returns the current environment variables from the OS.
func NewOSEnviron() Environ {
	env := map[string]string{}
	for _, ev := range os.Environ() {
		parts := strings.SplitN(ev, "=", 2)
		if len(parts) == 0 {
			continue
		}
		k := parts[0]
		v := ""
		if len(parts) == 2 {
			v = parts[1]
		}
		env[k] = v
	}
	return env
}

// ToSlice converts the environment variables to a slice.
func (e Environ) ToSlice() []string {
	env := make([]string, 0, len(e))
	for k, v := range e {
		env = append(env, k+"="+v)
	}
	// Ensure the order of the envs.
	slices.Sort(env)
	return env
}

var defaultEnvs = map[string]string{
	// Disable the update notifier and upgrade suggestions.
	"ATLAS_NO_UPDATE_NOTIFIER":     "1",
	"ATLAS_NO_UPGRADE_SUGGESTIONS": "1",
}

// ErrRequireLogin is returned when a command requires the user to be logged in.
// It exists here to be shared between the different packages that require login.
var ErrRequireLogin = errors.New("command requires 'atlas login'")

// runCommand runs the given command and returns its output.
func (c *Client) runCommand(ctx context.Context, args []string) (io.Reader, error) {
	var stdout, stderr bytes.Buffer
	cmd := c.cmd(ctx, args)
	cmd.Stdout = mergeWriters(&stdout, c.stdout)
	cmd.Stderr = mergeWriters(&stderr, c.stderr)
	if err := c.runErr(cmd.Run(), &stdout, &stderr); err != nil {
		return nil, err
	}
	return &stdout, nil
}

type Stream[T any] interface {
	// Next reads the next item from the stream, making it available by calling Current.
	// It returns false if there are no more items and the stream is closed.
	Next() bool
	// Current returns the current item from the stream.
	Current() (T, error)
	// Err returns the error, if any, that occurred while reading the stream.
	Err() error
}

// runCommandStream runs the given command streams its output split by new-lines.
func (c *Client) runCommandStream(ctx context.Context, args []string) (Stream[string], error) {
	cmd := c.cmd(ctx, args)
	var stderr bytes.Buffer
	cmd.Stderr = mergeWriters(&stderr, c.stderr)
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdout pipe: %w", err)
	}
	if err = cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting command: %w", err)
	}
	var (
		scan   = bufio.NewScanner(out)
		buf    = strings.Builder{}
		ch     = make(chan string)
		s      = &stream{ch: ch}
		stdout = mergeWriters(&buf, c.stdout)
	)
	go func() {
		defer close(ch)
		for scan.Scan() {
			stdout.Write(scan.Bytes())
			ch <- scan.Text()
		}
		s.lock.Lock()
		defer s.lock.Unlock()
		s.err = c.runErr(cmd.Wait(), &buf, &stderr)
	}()
	return s, nil
}

func (c *Client) cmd(ctx context.Context, args []string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, c.execPath, args...)
	cmd.Dir = c.workingDir
	var env Environ
	if c.env == nil {
		// Initialize the environment variables from the OS.
		env = NewOSEnviron()
	} else {
		env = maps.Clone(c.env)
	}
	maps.Copy(env, defaultEnvs)
	cmd.Env = env.ToSlice()
	return cmd
}

func (c *Client) runErr(err error, stdout, stderr interface{ String() string }) error {
	if err == nil {
		return nil
	}
	e := strings.TrimSpace(stderr.String())
	// Explicit check the stderr for the login error.
	if e == "Error: command requires 'atlas login'" {
		return ErrRequireLogin
	}
	return &Error{
		err:    err,
		Stderr: strings.TrimSpace(stderr.String()),
		Stdout: strings.TrimSpace(stdout.String()),
	}
}

type stream struct {
	ch   chan string
	cur  string
	err  error
	lock sync.RWMutex
}

// Next advances the stream to the next item.
func (s *stream) Next() bool {
	s.lock.RLock()
	if s.err != nil || s.ch == nil {
		s.lock.RUnlock()
		return false
	}
	s.lock.RUnlock()
	r, ok := <-s.ch
	if !ok {
		return false
	}
	s.cur = r
	return true
}

// Current returns the current item from the stream.
func (s *stream) Current() (string, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if s.err != nil {
		return "", s.err
	}
	return s.cur, nil
}

// Err returns the error, if any, that occurred while reading the stream.
func (s *stream) Err() error {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.err
}

var _ Stream[string] = (*stream)(nil)

func mergeWriters(writers ...io.Writer) io.Writer {
	var compact []io.Writer
	for _, w := range writers {
		if w != nil {
			compact = append(compact, w)
		}
	}
	switch len(compact) {
	case 1:
		return compact[0]
	case 0:
		return io.Discard
	default:
		return io.MultiWriter(compact...)
	}
}

// Error is an error returned by the atlasexec package,
// when it executes the atlas-cli command.
type Error struct {
	err    error  // The underlying error.
	Stdout string // Stdout of the command.
	Stderr string // Stderr of the command.
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Stderr != "" {
		return e.Stderr
	}
	return e.Stdout
}

// ExitCode returns the exit code of the command.
// If the error is not an exec.ExitError, it returns 1.
func (e *Error) ExitCode() int {
	var exitErr *exec.ExitError
	if errors.As(e.err, &exitErr) {
		return exitErr.ExitCode()
	}
	// Not an exec.ExitError or nil.
	// Return the system default exit code.
	return new(exec.ExitError).ExitCode()
}

// Unwrap returns the underlying error.
func (e *Error) Unwrap() error {
	return e.err
}

// TempFile creates a temporary file with the given content and extension.
func TempFile(content, ext string) (string, func() error, error) {
	f, err := os.CreateTemp("", "atlasexec-*."+ext)
	if err != nil {
		return "", nil, err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	if err != nil {
		return "", nil, err
	}
	return fmt.Sprintf("file://%s", f.Name()), func() error {
		return os.Remove(f.Name())
	}, nil
}

// AsArgs returns the variables as arguments.
func (v Vars2) AsArgs() []string {
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	var args []string
	for _, k := range keys {
		switch reflect.TypeOf(v[k]).Kind() {
		case reflect.Slice, reflect.Array:
			ev := reflect.ValueOf(v[k])
			for i := range ev.Len() {
				args = append(args, "--var", fmt.Sprintf("%s=%v", k, ev.Index(i)))
			}
		default:
			args = append(args, "--var", fmt.Sprintf("%s=%v", k, v[k]))
		}
	}
	return args
}

// AsArgs returns the variables as arguments.
func (v Vars) AsArgs() []string {
	var args []string
	for k, v := range v {
		args = append(args, "--var", fmt.Sprintf("%s=%s", k, v))
	}
	return args
}

func stringVal(r io.Reader, err error) (string, error) {
	if err != nil {
		return "", err
	}
	s, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	return string(s), nil
}

func jsonDecode[T any](r io.Reader, err error) ([]*T, error) {
	if err != nil {
		return nil, err
	}
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var dst []*T
	dec := json.NewDecoder(bytes.NewReader(buf))
	for {
		var m T
		switch err := dec.Decode(&m); err {
		case io.EOF:
			return dst, nil
		case nil:
			dst = append(dst, &m)
		default:
			return nil, &Error{
				err:    fmt.Errorf("decoding JSON from stdout: %w", err),
				Stdout: string(buf),
			}
		}
	}
}

func jsonDecodeErr[T any](fn func([]*T, string) error) func(io.Reader, error) ([]*T, error) {
	return func(r io.Reader, err error) ([]*T, error) {
		if err != nil {
			if cliErr := (&Error{}); errors.As(err, &cliErr) && cliErr.Stdout != "" {
				d, err := jsonDecode[T](strings.NewReader(cliErr.Stdout), nil)
				if err == nil {
					return nil, fn(d, cliErr.Stderr)
				}
				// If the error is not a JSON, return the original error.
			}
			return nil, err
		}
		return jsonDecode[T](r, err)
	}
}

// repeatFlag repeats the flag for each value.
func repeatFlag(flag string, values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values)*2)
	for _, v := range values {
		out = append(out, flag, v)
	}
	return out
}

func listString(args []string) string {
	return strings.Join(args, ",")
}

func firstResult[T ~[]E, E any](r T, err error) (e E, _ error) {
	switch {
	case err != nil:
		return e, err
	case len(r) == 1:
		return r[0], nil
	default:
		return e, errors.New("The command returned more than one result, use Slice function instead")
	}
}

func last[A ~[]E, E any](a A) (_ E) {
	if l := len(a); l > 0 {
		return a[l-1]
	}
	return
}
