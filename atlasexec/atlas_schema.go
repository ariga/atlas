// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package atlasexec

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"ariga.io/atlas/sql/migrate"
)

type (
	// SchemaPushParams are the parameters for the `schema push` command.
	SchemaPushParams struct {
		ConfigURL string
		Env       string
		Vars      VarArgs
		Context   *RunContext
		DevURL    string

		URL         []string // Desired schema URL(s) to push
		Schema      []string // If set, only the specified schemas are pushed.
		Name        string   // Name of the schema (repo) to push to.
		Tag         string   // Tag to push the schema with
		Version     string   // Version of the schema to push. Defaults to the current timestamp.
		Description string   // Description of the schema changes.
	}
	// SchemaPush represents the result of a 'schema push' command.
	SchemaPush struct {
		Link string
		Slug string
		URL  string
	}
	// SchemaApplyParams are the parameters for the `schema apply` command.
	SchemaApplyParams struct {
		ConfigURL string
		Env       string
		Vars      VarArgs
		DevURL    string

		URL         string
		To          string // TODO: change to []string
		TxMode      string
		Exclude     []string
		Include     []string
		Schema      []string
		DryRun      bool   // If true, --dry-run is set.
		AutoApprove bool   // If true, --auto-approve is set.
		PlanURL     string // URL of the plan in Atlas format (atlas://<repo>/plans/<id>). (optional)
		LockName    string
	}
	// SchemaApply represents the result of a 'schema apply' command.
	SchemaApply struct {
		Env
		// Changes holds the changes applied to the database.
		// Exists for backward compatibility with the old schema
		// apply structure as old SDK versions rely on it.
		Changes Changes      `json:"Changes,omitempty"`
		Error   string       `json:"Error,omitempty"`   // Any error that occurred during execution.
		Start   time.Time    `json:"Start,omitempty"`   // When apply (including plan) started.
		End     time.Time    `json:"End,omitempty"`     // When apply ended.
		Applied *AppliedFile `json:"Applied,omitempty"` // Applied migration file (pre-planned or computed).
		// Plan information might be partially filled. For example, if lint is done
		// during plan-stage, the linting report is available in the Plan field. If
		// the migration is pre-planned migration, the File.URL is set, etc.
		Plan *SchemaPlan `json:"Plan,omitempty"`
	}
	// SchemaApplyError is returned when an error occurred
	// during a schema applying attempt.
	SchemaApplyError struct {
		Result []*SchemaApply
		Stderr string
	}
	// SchemaInspectParams are the parameters for the `schema inspect` command.
	SchemaInspectParams struct {
		ConfigURL string
		Env       string
		Vars      VarArgs
		Format    string
		DevURL    string

		URL     string
		Exclude []string
		Include []string
		Schema  []string
	}
	// SchemaTestParams are the parameters for the `schema test` command.
	SchemaTestParams struct {
		ConfigURL string
		Env       string
		Vars      VarArgs
		DevURL    string

		URL   string
		Run   string
		Paths []string
	}
	// SchemaPlanParams are the parameters for the `schema plan` command.
	SchemaPlanParams struct {
		ConfigURL string
		Env       string
		Vars      VarArgs
		Context   *RunContext
		DevURL    string
		Exclude   []string
		Include   []string
		Schema    []string

		From, To   []string
		Repo       string
		Name       string
		Directives []string
		// The below are mutually exclusive and can be replaced
		// with the 'schema plan' sub-commands instead.
		DryRun     bool // If false, --auto-approve is set.
		Pending    bool
		Push, Save bool
	}
	// SchemaPlanListParams are the parameters for the `schema plan list` command.
	SchemaPlanListParams struct {
		ConfigURL string
		Env       string
		Vars      VarArgs
		Context   *RunContext
		DevURL    string
		Schema    []string
		Exclude   []string
		Include   []string

		From, To []string
		Repo     string
		Pending  bool // If true, only pending plans are listed.
	}
	// SchemaPlanPushParams are the parameters for the `schema plan push` command.
	SchemaPlanPushParams struct {
		ConfigURL string
		Env       string
		Vars      VarArgs
		Context   *RunContext
		DevURL    string
		Schema    []string
		Exclude   []string
		Include   []string

		From, To []string
		Repo     string
		Pending  bool   // Push plan in pending state.
		File     string // File to push. (optional)
	}
	// SchemaPlanPullParams are the parameters for the `schema plan pull` command.
	SchemaPlanPullParams struct {
		ConfigURL string
		Env       string
		Vars      VarArgs
		URL       string // URL to the plan in Atlas format. (required)
	}
	// SchemaPlanLintParams are the parameters for the `schema plan lint` command.
	SchemaPlanLintParams struct {
		ConfigURL string
		Env       string
		Vars      VarArgs
		Context   *RunContext
		DevURL    string
		Schema    []string
		Exclude   []string
		Include   []string

		From, To []string
		Repo     string
		File     string
	}
	// SchemaPlanValidateParams are the parameters for the `schema plan validate` command.
	SchemaPlanValidateParams struct {
		ConfigURL string
		Env       string
		Vars      VarArgs
		Context   *RunContext
		DevURL    string
		Schema    []string
		Exclude   []string
		Include   []string

		From, To []string
		Repo     string
		Name     string
		File     string
	}
	// SchemaPlanApproveParams are the parameters for the `schema plan approve` command.
	SchemaPlanApproveParams struct {
		ConfigURL string
		Env       string
		Vars      VarArgs

		URL string
	}
	// SchemaPlan is the result of a 'schema plan' command.
	SchemaPlan struct {
		Env   Env             `json:"Env,omitempty"`   // Environment info.
		Repo  string          `json:"Repo,omitempty"`  // Repository name.
		Lint  *SummaryReport  `json:"Lint,omitempty"`  // Lint report.
		File  *SchemaPlanFile `json:"File,omitempty"`  // Plan file.
		Error string          `json:"Error,omitempty"` // Any error occurred during planning.
	}
	// SchemaPlanApprove is the result of a 'schema plan approve' command.
	SchemaPlanApprove struct {
		URL    string `json:"URL,omitempty"`    // URL of the plan in Atlas format.
		Link   string `json:"Link,omitempty"`   // Link to the plan in the registry.
		Status string `json:"Status,omitempty"` // Status of the plan in the registry.
	}
	// SchemaPlanFile is a JSON representation of a schema plan file.
	SchemaPlanFile struct {
		Name      string          `json:"Name,omitempty"`      // Name of the plan.
		FromHash  string          `json:"FromHash,omitempty"`  // Hash of the 'from' realm.
		FromDesc  string          `json:"FromDesc,omitempty"`  // Optional description of the 'from' state.
		ToHash    string          `json:"ToHash,omitempty"`    // Hash of the 'to' realm.
		ToDesc    string          `json:"ToDesc,omitempty"`    // Optional description of the 'to' state.
		Migration string          `json:"Migration,omitempty"` // Migration SQL.
		Stmts     []*migrate.Stmt `json:"Stmts,omitempty"`     // Statements in the migration (available only in the JSON output).
		// registry only fields.
		URL    string `json:"URL,omitempty"`    // URL of the plan in Atlas format.
		Link   string `json:"Link,omitempty"`   // Link to the plan in the registry.
		Status string `json:"Status,omitempty"` // Status of the plan in the registry.
	}
	// SchemaCleanParams are the parameters for the `schema clean` command.
	SchemaCleanParams struct {
		ConfigURL string
		Env       string
		Vars      VarArgs

		URL         string // URL of the schema to clean. (required)
		DryRun      bool   // If true, --dry-run is set.
		AutoApprove bool   // If true, --auto-approve is set.
	}
	// SchemaClean represents the result of a 'schema clean' command.
	SchemaClean struct {
		Env
		Start   time.Time    `json:"Start,omitempty"`   // When clean started.
		End     time.Time    `json:"End,omitempty"`     // When clean ended.
		Applied *AppliedFile `json:"Applied,omitempty"` // Applied migration file.
		Error   string       `json:"Error,omitempty"`   // Any error that occurred during execution.
	}
	// SchemaLintParams are the parameters for the `schema lint` command.
	SchemaLintParams struct {
		ConfigURL string
		Env       string
		Vars      VarArgs

		URL    []string // Schema URL(s) to lint
		Schema []string // If set, only the specified schemas are linted.
		Format string
		DevURL string
	}
	// SchemaLintReport holds the results of a schema lint operation
	SchemaLintReport struct {
		Steps []Report `json:"Steps,omitempty"`
	}

	// SchemaStatsParams are the parameters for the `schema stats` command.
	SchemaStatsParams struct {
		ConfigURL string
		Env       string
		Vars      VarArgs

		URL     string
		Exclude []string
		Include []string
		Schema  []string
	}
)

// SchemaPush runs the 'schema push' command.
func (c *Client) SchemaPush(ctx context.Context, params *SchemaPushParams) (*SchemaPush, error) {
	args := []string{"schema", "push", "--format", "{{ json . }}"}
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
	// Hidden flags
	if params.Context != nil {
		buf, err := json.Marshal(params.Context)
		if err != nil {
			return nil, err
		}
		args = append(args, "--context", string(buf))
	}
	// Flags of the 'schema push' sub-commands
	args = append(args, repeatFlag("--url", params.URL)...)
	if params.DevURL != "" {
		args = append(args, "--dev-url", params.DevURL)
	}
	if len(params.Schema) > 0 {
		args = append(args, "--schema", listString(params.Schema))
	}
	if params.Tag != "" {
		args = append(args, "--tag", params.Tag)
	}
	if params.Version != "" {
		args = append(args, "--version", params.Version)
	}
	if params.Description != "" {
		args = append(args, "--desc", params.Description)
	}
	if params.Name != "" {
		args = append(args, params.Name)
	}
	return firstResult(jsonDecode[SchemaPush](c.runCommand(ctx, args)))
}

// SchemaApply runs the 'schema apply' command.
func (c *Client) SchemaApply(ctx context.Context, params *SchemaApplyParams) (*SchemaApply, error) {
	return firstResult(c.SchemaApplySlice(ctx, params))
}

// SchemaApplySlice runs the 'schema apply' command for multiple targets.
func (c *Client) SchemaApplySlice(ctx context.Context, params *SchemaApplyParams) ([]*SchemaApply, error) {
	args := []string{"schema", "apply", "--format", "{{ json . }}"}
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
	// Flags of the 'schema apply' sub-commands
	if params.URL != "" {
		args = append(args, "--url", params.URL)
	}
	if params.To != "" {
		args = append(args, "--to", params.To)
	}
	if params.TxMode != "" {
		args = append(args, "--tx-mode", params.TxMode)
	}
	if params.DevURL != "" {
		args = append(args, "--dev-url", params.DevURL)
	}
	if len(params.Schema) > 0 {
		args = append(args, "--schema", listString(params.Schema))
	}
	if len(params.Exclude) > 0 {
		args = append(args, "--exclude", listString(params.Exclude))
	}
	if len(params.Include) > 0 {
		args = append(args, "--include", listString(params.Include))
	}
	if params.PlanURL != "" {
		args = append(args, "--plan", params.PlanURL)
	}
	if params.LockName != "" {
		args = append(args, "--lock-name", params.LockName)
	}
	switch {
	case params.DryRun:
		args = append(args, "--dry-run")
	case params.AutoApprove:
		args = append(args, "--auto-approve")
	}
	return jsonDecodeErr(newSchemaApplyError)(c.runCommand(ctx, args))
}

// SchemaInspect runs the 'schema inspect' command.
func (c *Client) SchemaInspect(ctx context.Context, params *SchemaInspectParams) (string, error) {
	args := []string{"schema", "inspect"}
	if params.Env != "" {
		args = append(args, "--env", params.Env)
	}
	if params.ConfigURL != "" {
		args = append(args, "--config", params.ConfigURL)
	}
	if params.URL != "" {
		args = append(args, "--url", params.URL)
	}
	if params.DevURL != "" {
		args = append(args, "--dev-url", params.DevURL)
	}
	switch {
	case params.Format == "sql":
		args = append(args, "--format", "{{ sql . }}")
	case params.Format != "":
		args = append(args, "--format", params.Format)
	}
	if len(params.Schema) > 0 {
		args = append(args, "--schema", listString(params.Schema))
	}
	if len(params.Exclude) > 0 {
		args = append(args, "--exclude", listString(params.Exclude))
	}
	if len(params.Include) > 0 {
		args = append(args, "--include", listString(params.Include))
	}
	if params.Vars != nil {
		args = append(args, params.Vars.AsArgs()...)
	}
	return stringVal(c.runCommand(ctx, args))
}

// SchemaTest runs the 'schema test' command.
func (c *Client) SchemaTest(ctx context.Context, params *SchemaTestParams) (string, error) {
	args := []string{"schema", "test"}
	if params.Env != "" {
		args = append(args, "--env", params.Env)
	}
	if params.ConfigURL != "" {
		args = append(args, "--config", params.ConfigURL)
	}
	if params.URL != "" {
		args = append(args, "--url", params.URL)
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

// SchemaPlan runs the `schema plan` command.
func (c *Client) SchemaPlan(ctx context.Context, params *SchemaPlanParams) (*SchemaPlan, error) {
	args := []string{"schema", "plan", "--format", "{{ json . }}"}
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
	// Hidden flags
	if params.Context != nil {
		buf, err := json.Marshal(params.Context)
		if err != nil {
			return nil, err
		}
		args = append(args, "--context", string(buf))
	}
	// Flags of the 'schema plan' sub-commands
	if params.DevURL != "" {
		args = append(args, "--dev-url", params.DevURL)
	}
	if len(params.Schema) > 0 {
		args = append(args, "--schema", listString(params.Schema))
	}
	if len(params.Exclude) > 0 {
		args = append(args, "--exclude", listString(params.Exclude))
	}
	if len(params.Include) > 0 {
		args = append(args, "--include", listString(params.Include))
	}
	if len(params.From) > 0 {
		args = append(args, "--from", listString(params.From))
	}
	if len(params.To) > 0 {
		args = append(args, "--to", listString(params.To))
	}
	if params.Name != "" {
		args = append(args, "--name", params.Name)
	}
	if params.Repo != "" {
		args = append(args, "--repo", params.Repo)
	}
	if params.Save {
		args = append(args, "--save")
	}
	if params.Push {
		args = append(args, "--push")
	}
	if params.Pending {
		args = append(args, "--pending")
	}
	if params.DryRun {
		args = append(args, "--dry-run")
	} else {
		args = append(args, "--auto-approve")
	}
	for _, d := range params.Directives {
		args = append(args, "--directive", strconv.Quote(d))
	}
	// NOTE: This command only support one result.
	return firstResult(jsonDecode[SchemaPlan](c.runCommand(ctx, args)))
}

// SchemaPlanList runs the `schema plan list` command.
func (c *Client) SchemaPlanList(ctx context.Context, params *SchemaPlanListParams) ([]SchemaPlanFile, error) {
	args := []string{"schema", "plan", "list", "--format", "{{ json . }}"}
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
	// Hidden flags
	if params.Context != nil {
		buf, err := json.Marshal(params.Context)
		if err != nil {
			return nil, err
		}
		args = append(args, "--context", string(buf))
	}
	// Flags of the 'schema plan lint' sub-commands
	if params.DevURL != "" {
		args = append(args, "--dev-url", params.DevURL)
	}
	if len(params.Schema) > 0 {
		args = append(args, "--schema", listString(params.Schema))
	}
	if len(params.Exclude) > 0 {
		args = append(args, "--exclude", listString(params.Exclude))
	}
	if len(params.Include) > 0 {
		args = append(args, "--include", listString(params.Include))
	}
	if len(params.From) > 0 {
		args = append(args, "--from", listString(params.From))
	}
	if len(params.To) > 0 {
		args = append(args, "--to", listString(params.To))
	}
	if params.Repo != "" {
		args = append(args, "--repo", params.Repo)
	}
	if params.Pending {
		args = append(args, "--pending")
	}
	args = append(args, "--auto-approve")
	// NOTE: This command only support one result.
	v, err := firstResult(jsonDecode[[]SchemaPlanFile](c.runCommand(ctx, args)))
	if err != nil {
		return nil, err
	}
	return *v, nil
}

// SchemaPlanPush runs the `schema plan push` command.
func (c *Client) SchemaPlanPush(ctx context.Context, params *SchemaPlanPushParams) (string, error) {
	args := []string{"schema", "plan", "push", "--format", "{{ json . }}"}
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
	// Hidden flags
	if params.Context != nil {
		buf, err := json.Marshal(params.Context)
		if err != nil {
			return "", err
		}
		args = append(args, "--context", string(buf))
	}
	// Flags of the 'schema plan push' sub-commands
	if params.DevURL != "" {
		args = append(args, "--dev-url", params.DevURL)
	}
	if len(params.Schema) > 0 {
		args = append(args, "--schema", listString(params.Schema))
	}
	if len(params.Exclude) > 0 {
		args = append(args, "--exclude", listString(params.Exclude))
	}
	if len(params.Include) > 0 {
		args = append(args, "--include", listString(params.Include))
	}
	if len(params.From) > 0 {
		args = append(args, "--from", listString(params.From))
	}
	if len(params.To) > 0 {
		args = append(args, "--to", listString(params.To))
	}
	if params.File != "" {
		args = append(args, "--file", params.File)
	} else {
		return "", &InvalidParamsError{"schema plan push", "missing required flag --file"}
	}
	if params.Repo != "" {
		args = append(args, "--repo", params.Repo)
	}
	if params.Pending {
		args = append(args, "--pending")
	} else {
		args = append(args, "--auto-approve")
	}
	return stringVal(c.runCommand(ctx, args))
}

// SchemaPlanPush runs the `schema plan pull` command.
func (c *Client) SchemaPlanPull(ctx context.Context, params *SchemaPlanPullParams) (string, error) {
	args := []string{"schema", "plan", "pull"}
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
	// Flags of the 'schema plan pull' sub-commands
	if params.URL != "" {
		args = append(args, "--url", params.URL)
	} else {
		return "", &InvalidParamsError{"schema plan pull", "missing required flag --url"}
	}
	return stringVal(c.runCommand(ctx, args))
}

// SchemaPlanLint runs the `schema plan lint` command.
func (c *Client) SchemaPlanLint(ctx context.Context, params *SchemaPlanLintParams) (*SchemaPlan, error) {
	args := []string{"schema", "plan", "lint", "--format", "{{ json . }}"}
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
	// Hidden flags
	if params.Context != nil {
		buf, err := json.Marshal(params.Context)
		if err != nil {
			return nil, err
		}
		args = append(args, "--context", string(buf))
	}
	// Flags of the 'schema plan lint' sub-commands
	if params.DevURL != "" {
		args = append(args, "--dev-url", params.DevURL)
	}
	if len(params.Schema) > 0 {
		args = append(args, "--schema", listString(params.Schema))
	}
	if len(params.Exclude) > 0 {
		args = append(args, "--exclude", listString(params.Exclude))
	}
	if len(params.Include) > 0 {
		args = append(args, "--include", listString(params.Include))
	}
	if len(params.From) > 0 {
		args = append(args, "--from", listString(params.From))
	}
	if len(params.To) > 0 {
		args = append(args, "--to", listString(params.To))
	}
	if params.File != "" {
		args = append(args, "--file", params.File)
	} else {
		return nil, &InvalidParamsError{"schema plan lint", "missing required flag --file"}
	}
	if params.Repo != "" {
		args = append(args, "--repo", params.Repo)
	}
	args = append(args, "--auto-approve")
	// NOTE: This command only support one result.
	return firstResult(jsonDecode[SchemaPlan](c.runCommand(ctx, args)))
}

// SchemaPlanValidate runs the `schema plan validate` command.
func (c *Client) SchemaPlanValidate(ctx context.Context, params *SchemaPlanValidateParams) error {
	args := []string{"schema", "plan", "validate"}
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
	// Hidden flags
	if params.Context != nil {
		buf, err := json.Marshal(params.Context)
		if err != nil {
			return err
		}
		args = append(args, "--context", string(buf))
	}
	// Flags of the 'schema plan validate' sub-commands
	if params.DevURL != "" {
		args = append(args, "--dev-url", params.DevURL)
	}
	if len(params.Schema) > 0 {
		args = append(args, "--schema", listString(params.Schema))
	}
	if len(params.Exclude) > 0 {
		args = append(args, "--exclude", listString(params.Exclude))
	}
	if len(params.Include) > 0 {
		args = append(args, "--include", listString(params.Include))
	}
	if len(params.From) > 0 {
		args = append(args, "--from", listString(params.From))
	}
	if len(params.To) > 0 {
		args = append(args, "--to", listString(params.To))
	}
	if params.File != "" {
		args = append(args, "--file", params.File)
	} else {
		return &InvalidParamsError{"schema plan validate", "missing required flag --file"}
	}
	if params.Name != "" {
		args = append(args, "--name", params.Name)
	}
	if params.Repo != "" {
		args = append(args, "--repo", params.Repo)
	}
	args = append(args, "--auto-approve")
	_, err := stringVal(c.runCommand(ctx, args))
	return err
}

// SchemaPlanApprove runs the `schema plan approve` command.
func (c *Client) SchemaPlanApprove(ctx context.Context, params *SchemaPlanApproveParams) (*SchemaPlanApprove, error) {
	args := []string{"schema", "plan", "approve", "--format", "{{ json . }}"}
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
	// Flags of the 'schema plan approve' sub-commands
	if params.URL != "" {
		args = append(args, "--url", params.URL)
	} else {
		return nil, &InvalidParamsError{"schema plan approve", "missing required flag --url"}
	}
	// NOTE: This command only support one result.
	return firstResult(jsonDecode[SchemaPlanApprove](c.runCommand(ctx, args)))
}

// SchemaClean runs the `schema clean` command.
func (c *Client) SchemaClean(ctx context.Context, params *SchemaCleanParams) (*SchemaClean, error) {
	args := []string{"schema", "clean", "--format", "{{ json . }}"}
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
	// Flags of the 'schema clean' sub-commands
	if params.URL != "" {
		args = append(args, "--url", params.URL)
	}
	switch {
	case params.DryRun:
		args = append(args, "--dry-run")
	case params.AutoApprove:
		args = append(args, "--auto-approve")
	}
	return firstResult(jsonDecode[SchemaClean](c.runCommand(ctx, args)))
}

// SchemaLint runs the 'schema lint' command.
func (c *Client) SchemaLint(ctx context.Context, params *SchemaLintParams) (*SchemaLintReport, error) {
	args, err := params.AsArgs()
	if err != nil {
		return nil, err
	}
	return firstResult(jsonDecode[SchemaLintReport](c.runCommand(ctx, args)))
}

// SchemaStats runs the 'schema stats' command.
func (c *Client) SchemaStats(ctx context.Context, params *SchemaStatsParams) ([]TableSizeMetric, error) {
	args := []string{"schema", "stats"}
	if params.Env != "" {
		args = append(args, "--env", params.Env)
	}
	if params.ConfigURL != "" {
		args = append(args, "--config", params.ConfigURL)
	}
	if params.URL != "" {
		args = append(args, "--url", params.URL)
	}
	if len(params.Schema) > 0 {
		args = append(args, "--schema", listString(params.Schema))
	}
	if len(params.Exclude) > 0 {
		args = append(args, "--exclude", listString(params.Exclude))
	}
	if len(params.Include) > 0 {
		args = append(args, "--include", listString(params.Include))
	}
	if params.Vars != nil {
		args = append(args, params.Vars.AsArgs()...)
	}
	output, err := stringVal(c.runCommand(ctx, args))
	if err != nil {
		return nil, err
	}
	return ParsePrometheusMetrics(output)
}

// AsArgs returns the parameters as arguments.
func (p *SchemaLintParams) AsArgs() ([]string, error) {
	args := []string{"schema", "lint", "--format", "{{ json . }}"}
	if p.Env != "" {
		args = append(args, "--env", p.Env)
	}
	if p.ConfigURL != "" {
		args = append(args, "--config", p.ConfigURL)
	}
	if p.DevURL != "" {
		args = append(args, "--dev-url", p.DevURL)
	}
	args = append(args, repeatFlag("--url", p.URL)...)
	if len(p.Schema) > 0 {
		args = append(args, "--schema", listString(p.Schema))
	}
	if p.Vars != nil {
		args = append(args, p.Vars.AsArgs()...)
	}
	return args, nil
}

// InvalidParamsError is an error type for invalid parameters.
type InvalidParamsError struct {
	cmd string
	msg string
}

// Error returns the error message.
func (e *InvalidParamsError) Error() string {
	return fmt.Sprintf("atlasexec: command %q has invalid parameters: %v", e.cmd, e.msg)
}
func newSchemaApplyError(r []*SchemaApply, stderr string) error {
	return &SchemaApplyError{Result: r, Stderr: stderr}
}

// Error implements the error interface.
func (e *SchemaApplyError) Error() string {
	if e.Stderr != "" {
		return e.Stderr
	}
	return last(e.Result).Error
}
