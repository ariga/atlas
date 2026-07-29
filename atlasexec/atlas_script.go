// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package atlasexec

import (
	"context"
	"time"
)

type (
	// ScriptExecParams are the parameters for the `script exec` command.
	ScriptExecParams struct {
		ConfigURL string
		Env       string
		Vars      VarArgs

		URL   string   // Database URL to execute against. (required)
		Files []string // URL(s) to script files or directories.
		Match string   // Run only scripts matching regexp.
		Quiet bool     // Output only the script's output.
	}
	// ScriptQueryParams are the parameters for the `script query` command.
	ScriptQueryParams struct {
		ConfigURL string
		Env       string
		Vars      VarArgs

		URL   string   // Database URL to query against. (required)
		Files []string // URL(s) to script files or directories.
		Match string   // Run only scripts matching regexp.
		Quiet bool     // Output only the script's output.
	}
	// ScriptLoopParams are the parameters for the `script loop` command.
	ScriptLoopParams struct {
		ConfigURL string
		Env       string
		Vars      VarArgs

		URL   string   // Database URL to run against. (required)
		Files []string // URL(s) to script files or directories.
		Match string   // Run only scripts matching regexp.
		Quiet bool     // Output only the script's output.
	}
	// ScriptTestParams are the parameters for the `script test` command.
	ScriptTestParams struct {
		ConfigURL string
		Env       string
		Vars      VarArgs
		DevURL    string

		Run   string   // Run only tests matching regexp.
		Paths []string // Paths to the test files/directories.
	}
	// ScriptPushParams are the parameters for the `script push` command.
	ScriptPushParams struct {
		ConfigURL string
		Env       string
		Vars      VarArgs

		Name  string   // Name of the script repository to push to.
		Files []string // URL(s) to script files or directories.
	}
	// ScriptExec is a summary of a `script exec|query|loop` run.
	ScriptExec struct {
		Scripts []*ScriptFile `json:"Scripts,omitempty"` // Executed scripts, in order.
		Start   time.Time     // Run start time.
		End     time.Time     // Run end time.
		Error   string        `json:"Error,omitempty"` // Run-level error, outside any script.
	}
	// ScriptFile is the report of a single executed script.
	ScriptFile struct {
		Name       string             `json:"Name,omitempty"`
		Kind       string             `json:"Kind,omitempty"`
		Conditions []*ScriptCond      `json:"Conditions,omitempty"`
		Asserts    []*ScriptBool      `json:"Asserts,omitempty"`
		Checks     []*ScriptCheck     `json:"Checks,omitempty"`
		Execs      []*ScriptStmt      `json:"Execs,omitempty"`
		Queries    []*ScriptResult    `json:"Queries,omitempty"`
		Iterations []*ScriptIteration `json:"Iterations,omitempty"`
		Messages   []string           `json:"Messages,omitempty"`
		Outputs    []string           `json:"Outputs,omitempty"`
		Start      time.Time
		End        time.Time
		Error      string `json:"Error,omitempty"`
	}
	// ScriptCond is a `condition` guard outcome. Passed is false when the guard
	// was not met and the script stopped gracefully.
	ScriptCond struct {
		Name   string `json:"Name,omitempty"`
		Passed bool   `json:"Passed"`
		Error  string `json:"Error,omitempty"`
	}
	// ScriptBool is an `assert` outcome.
	ScriptBool struct {
		Name   string `json:"Name,omitempty"`
		Passed bool   `json:"Passed"`
		Error  string `json:"Error,omitempty"`
	}
	// ScriptCheck is a `check` outcome. Diff is set on a comparison mismatch.
	ScriptCheck struct {
		Name   string `json:"Name,omitempty"`
		Passed bool   `json:"Passed"`
		Diff   string `json:"Diff,omitempty"`
		Error  string `json:"Error,omitempty"`
	}
	// ScriptStmt is an `exec` statement outcome.
	ScriptStmt struct {
		Name         string    `json:"Name,omitempty"`
		SQL          string    `json:"SQL,omitempty"`
		AffectedRows int64     `json:"AffectedRows,omitempty"`
		Start        time.Time `json:"Start,omitempty"`
		End          time.Time `json:"End,omitempty"`
		Error        string    `json:"Error,omitempty"`
	}
	// ScriptResult is a `query` result.
	ScriptResult struct {
		Index int    `json:"Index"`
		Out   string `json:"Out,omitempty"`
	}
	// ScriptIteration is one loop iteration: the 0-based Index, the batch Size,
	// and how long its do body Took.
	ScriptIteration struct {
		Index int           `json:"Index"`
		Size  int           `json:"Size,omitempty"`
		Took  time.Duration `json:"Took,omitempty"`
	}
	// ScriptPush represents the result of a 'script push' command.
	ScriptPush struct {
		Name    string              `json:"Name,omitempty"`    // Name of the script repository.
		Files   int                 `json:"Files,omitempty"`   // Number of pushed script files.
		Link    string              `json:"Link,omitempty"`    // Web URL of the scripts in the registry.
		Backups []*ScriptPushBackup `json:"Backups,omitempty"` // Backup replication results.
		Error   string              `json:"Error,omitempty"`   // Cloud push error, if any.
	}
	// ScriptPushBackup represents one backup replication attempt during script push.
	ScriptPushBackup struct {
		URL   string `json:"URL,omitempty"`
		Error string `json:"Error,omitempty"`
	}
)

// ScriptExec runs the 'script exec' command.
func (c *Client) ScriptExec(ctx context.Context, params *ScriptExecParams) (*ScriptExec, error) {
	args := []string{"script", "exec", "--format", "{{ json . }}"}
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
	// Flags of the 'script exec' sub-command
	if params.URL != "" {
		args = append(args, "--url", params.URL)
	}
	args = append(args, repeatFlag("--file", params.Files)...)
	if params.Match != "" {
		args = append(args, "--run", params.Match)
	}
	if params.Quiet {
		args = append(args, "--quiet")
	}
	return firstResult(jsonDecode[ScriptExec](c.runCommand(ctx, args)))
}

// ScriptQuery runs the 'script query' command.
func (c *Client) ScriptQuery(ctx context.Context, params *ScriptQueryParams) (*ScriptExec, error) {
	args := []string{"script", "query", "--format", "{{ json . }}"}
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
	// Flags of the 'script query' sub-command
	if params.URL != "" {
		args = append(args, "--url", params.URL)
	}
	args = append(args, repeatFlag("--file", params.Files)...)
	if params.Match != "" {
		args = append(args, "--run", params.Match)
	}
	if params.Quiet {
		args = append(args, "--quiet")
	}
	return firstResult(jsonDecode[ScriptExec](c.runCommand(ctx, args)))
}

// ScriptLoop runs the 'script loop' command.
func (c *Client) ScriptLoop(ctx context.Context, params *ScriptLoopParams) (*ScriptExec, error) {
	args := []string{"script", "loop", "--format", "{{ json . }}"}
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
	// Flags of the 'script loop' sub-command
	if params.URL != "" {
		args = append(args, "--url", params.URL)
	}
	args = append(args, repeatFlag("--file", params.Files)...)
	if params.Match != "" {
		args = append(args, "--run", params.Match)
	}
	if params.Quiet {
		args = append(args, "--quiet")
	}
	return firstResult(jsonDecode[ScriptExec](c.runCommand(ctx, args)))
}

// ScriptTest runs the 'script test' command.
func (c *Client) ScriptTest(ctx context.Context, params *ScriptTestParams) (string, error) {
	args := []string{"script", "test"}
	if params.ConfigURL != "" {
		args = append(args, "--config", params.ConfigURL)
	}
	if params.Env != "" {
		args = append(args, "--env", params.Env)
	}
	if params.DevURL != "" {
		args = append(args, "--dev-url", params.DevURL)
	}
	if params.Run != "" {
		args = append(args, "--run", params.Run)
	}
	if params.Vars != nil {
		args = append(args, params.Vars.AsArgs()...)
	}
	if len(params.Paths) > 0 {
		args = append(args, params.Paths...)
	}
	return stringVal(c.runCommand(ctx, args))
}

// ScriptPush runs the 'script push' command.
func (c *Client) ScriptPush(ctx context.Context, params *ScriptPushParams) (*ScriptPush, error) {
	args := []string{"script", "push", "--format", "{{ json . }}"}
	if params.ConfigURL != "" {
		args = append(args, "--config", params.ConfigURL)
	}
	if params.Env != "" {
		args = append(args, "--env", params.Env)
	}
	if params.Vars != nil {
		args = append(args, params.Vars.AsArgs()...)
	}
	args = append(args, repeatFlag("--file", params.Files)...)
	if params.Name != "" {
		args = append(args, params.Name)
	}
	return firstResult(jsonDecode[ScriptPush](c.runCommand(ctx, args)))
}
