// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package action

import (
	"context"
	"errors"
	"io/ioutil"
	"strings"

	"ariga.io/atlas/sql/schema"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var (
	// ApplyFlags are the flags used in Apply command.
	ApplyFlags struct {
		URL         string
		DevURL      string
		File        string
		Web         bool
		Addr        string
		DryRun      bool
		Schema      []string
		AutoApprove bool
	}
	// ApplyCmd represents the apply command.
	ApplyCmd = &cobra.Command{
		Use:   "apply",
		Short: "Apply an atlas schema to a target database.",
		// Use 80-columns as max width.
		Long: `'atlas schema apply' plans and executes a database migration to bring a given
database to the state described in the Atlas schema file. Before running the
migration, Atlas will print the migration plan and prompt the user for approval.

If run with the "--dry-run" flag, atlas will exit after printing out the planned
migration.`,
		Run: CmdApplyRun,
		Example: `  atlas schema apply -u "mysql://user:pass@localhost/dbname" -f atlas.hcl
  atlas schema apply -u "mysql://localhost" -f atlas.hcl --schema prod --schema staging
  atlas schema apply -u "mysql://user:pass@localhost:3306/dbname" -f atlas.hcl --dry-run
  atlas schema apply -u "mariadb://user:pass@localhost:3306/dbname" -f atlas.hcl
  atlas schema apply --url "postgres://user:pass@host:port/dbname?sslmode=disable" -f atlas.hcl
  atlas schema apply -u "sqlite://file:ex1.db?_fk=1" -f atlas.hcl`,
	}
)

const (
	answerApply = "Apply"
	answerAbort = "Abort"
)

func init() {
	schemaCmd.AddCommand(ApplyCmd)
	ApplyCmd.Flags().SortFlags = false
	ApplyCmd.Flags().StringVarP(&ApplyFlags.File, "file", "f", "", "[/path/to/file] file containing the HCL schema.")
	ApplyCmd.Flags().StringVarP(&ApplyFlags.URL, "url", "u", "", "URL to the database using the format:\n[driver://username:password@address/dbname?param=value]")
	ApplyCmd.Flags().StringSliceVarP(&ApplyFlags.Schema, "schema", "s", nil, "Set schema names.")
	ApplyCmd.Flags().StringVarP(&ApplyFlags.URL, "dev-url", "", "", "URL for the dev database. Used to validate schemas and calculate diffs\nbefore running migration.")
	ApplyCmd.Flags().BoolVarP(&ApplyFlags.DryRun, "dry-run", "", false, "Dry-run. Print SQL plan without prompting for execution.")
	ApplyCmd.Flags().BoolVarP(&ApplyFlags.AutoApprove, "auto-approve", "", false, "Auto approve. Apply the schema changes without prompting for approval.")
	ApplyCmd.Flags().BoolVarP(&ApplyFlags.Web, "web", "w", false, "Open in a local Atlas UI.")
	ApplyCmd.Flags().StringVarP(&ApplyFlags.Addr, "addr", "", ":5800", "used with -w, local address to bind the server to.")
	cobra.CheckErr(ApplyCmd.MarkFlagRequired("url"))
	cobra.CheckErr(ApplyCmd.MarkFlagRequired("file"))
	dsn2url(ApplyCmd, &ApplyFlags.URL)
}

// CmdApplyRun is the command used when running CLI.
func CmdApplyRun(*cobra.Command, []string) {
	if ApplyFlags.Web {
		schemaCmd.PrintErrln("The Atlas UI is not available in this release.")
		return
	}
	d, err := DefaultMux.OpenAtlas(ApplyFlags.URL)
	cobra.CheckErr(err)
	applyRun(d, ApplyFlags.URL, ApplyFlags.File, ApplyFlags.DryRun, ApplyFlags.AutoApprove)
}

func applyRun(d *Driver, url string, file string, dryRun bool, autoApprove bool) {
	ctx := context.Background()
	schemas := ApplyFlags.Schema
	if n, err := SchemaNameFromURL(url); n != "" {
		cobra.CheckErr(err)
		schemas = append(schemas, n)
	}
	realm, err := d.InspectRealm(ctx, &schema.InspectRealmOption{
		Schemas: schemas,
	})
	cobra.CheckErr(err)
	f, err := ioutil.ReadFile(file)
	cobra.CheckErr(err)
	desired := &schema.Realm{}
	cobra.CheckErr(d.UnmarshalSpec(f, desired))
	if len(schemas) > 0 {
		// Validate all schemas in file were selected by user.
		sm := make(map[string]bool, len(schemas))
		for _, s := range schemas {
			sm[s] = true
		}
		for _, s := range desired.Schemas {
			if !sm[s.Name] {
				schemaCmd.Printf("schema %q from file %q was not selected %q, all schemas defined in file must be selected\n", s.Name, file, schemas)
				return
			}
		}
	}
	if _, ok := d.Driver.(schema.Normalizer); ok && ApplyFlags.DevURL != "" {
		dev, err := DefaultMux.OpenAtlas(ApplyFlags.DevURL)
		cobra.CheckErr(err)
		desired, err = dev.Driver.(schema.Normalizer).NormalizeRealm(ctx, desired)
		cobra.CheckErr(err)
	}
	changes, err := d.RealmDiff(realm, desired)
	cobra.CheckErr(err)
	if len(changes) == 0 {
		schemaCmd.Println("Schema is synced, no changes to be made")
		return
	}
	p, err := d.PlanChanges(ctx, "plan", changes)
	cobra.CheckErr(err)
	schemaCmd.Println("-- Planned Changes:")
	for _, c := range p.Changes {
		if c.Comment != "" {
			schemaCmd.Println("--", strings.ToUpper(c.Comment[:1])+c.Comment[1:])
		}
		schemaCmd.Println(c.Cmd)
	}
	if dryRun {
		return
	}
	if autoApprove || promptUser() {
		cobra.CheckErr(d.ApplyChanges(ctx, changes))
	}
}

func promptUser() bool {
	prompt := promptui.Select{
		Label: "Are you sure?",
		Items: []string{answerApply, answerAbort},
	}
	_, result, err := prompt.Run()
	cobra.CheckErr(err)
	return result == answerApply
}

func dsn2url(cmd *cobra.Command, p *string) {
	cmd.Flags().StringVarP(p, "dsn", "d", "", "")
	cobra.CheckErr(cmd.Flags().MarkHidden("dsn"))
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		dsnF, urlF := cmd.Flag("dsn"), cmd.Flag("url")
		switch {
		case !dsnF.Changed && !urlF.Changed:
			return errors.New(`required flag "url" was not set`)
		case dsnF.Changed && urlF.Changed:
			return errors.New(`both flags "url" and "dsn" were set`)
		case dsnF.Changed && !urlF.Changed:
			urlF.Changed = true
			urlF.Value = dsnF.Value
		}
		return nil
	}
}
