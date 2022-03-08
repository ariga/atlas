// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package action

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"text/template"
	"time"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"github.com/spf13/cobra"
)

const (
	migrateFlagDir        = "dir"
	migrateFlagSchema     = "schema"
	migrateFlagForce      = "force"
	migrateDiffFlagDevURL = "dev-url"
	migrateDiffFlagTo     = "to"
)

var (
	// MigrateFlags are the flags used in MigrateCmd (and sub-commands).
	MigrateFlags struct {
		DirURL  string
		DevURL  string
		ToURL   string
		Schemas []string
		Force   bool
	}
	// MigrateCmd represents the migrate command. It wraps several other sub-commands.
	MigrateCmd = &cobra.Command{
		Use:   "migrate",
		Short: "Manage versioned migration files",
		Long:  "'atlas migrate' wraps several sub-commands for migration management.",
		PersistentPreRun: func(*cobra.Command, []string) {
			// Migrate commands will not run on a broken migration directory, unless the force flag is given.
			if !MigrateFlags.Force {
				dir, err := dir()
				cobra.CheckErr(err)
				cobra.CheckErr(migrate.Validate(dir)) // TODO(masseelch): tell the user what's wrong
			}
		},
	}
	// MigrateDiffCmd represents the migrate diff command.
	MigrateDiffCmd = &cobra.Command{
		Use:   "diff",
		Short: "Compute the diff between the migration directory and a connected database and create a new migration file.",
		Long: `'atlas migrate diff' uses the dev-database to re-run all migration files in the migration
directory and compares it to a given desired state and create a new migration file containing SQL statements to migrate 
the migration directory state to the desired schema. The desired state can be another connected database or an HCL file.`,
		Example: `  atlas migrate diff --dev-url mysql://user:pass@localhost:3306/dev --to mysql://user:pass@localhost:3306/dbname
  atlas migrate diff --dev-url mysql://user:pass@localhost:3306/dev --to file://atlas.hcl`,
		Args: cobra.MaximumNArgs(1),
		Run:  CmdMigrateDiffRun,
	}
)

func init() {
	RootCmd.AddCommand(MigrateCmd)
	MigrateCmd.AddCommand(MigrateDiffCmd)
	MigrateCmd.PersistentFlags().StringVarP(&MigrateFlags.DirURL, migrateFlagDir, "", "file://migrations", "select migration directory using DSN format")
	MigrateCmd.PersistentFlags().StringSliceVarP(&MigrateFlags.Schemas, migrateFlagSchema, "", nil, "set schema names")
	MigrateCmd.PersistentFlags().BoolVarP(&MigrateFlags.Force, migrateFlagForce, "", false, "force a command to run on a broken migration directory state")
	MigrateCmd.PersistentFlags().SortFlags = false
	MigrateDiffCmd.Flags().StringVarP(&MigrateFlags.DevURL, migrateDiffFlagDevURL, "", "", "[driver://username:password@address/dbname?param=value] select a data source using the DSN format")
	MigrateDiffCmd.Flags().StringVarP(&MigrateFlags.ToURL, migrateDiffFlagTo, "", "", "[driver://username:password@address/dbname?param=value] select a data source using the DSN format")
	MigrateDiffCmd.Flags().SortFlags = false
	cobra.CheckErr(MigrateDiffCmd.MarkFlagRequired(migrateDiffFlagDevURL))
	cobra.CheckErr(MigrateDiffCmd.MarkFlagRequired(migrateDiffFlagTo))
}

// CmdMigrateDiffRun is the command executed when running the CLI with 'migrate diff' args.
func CmdMigrateDiffRun(cmd *cobra.Command, args []string) {
	// Open a dev driver.
	dev, err := DefaultMux.OpenAtlas(MigrateFlags.DevURL)
	cobra.CheckErr(err)
	// Open the migration directory. For now only local directories are supported.
	dir, err := dir()
	cobra.CheckErr(err)
	// Get a state reader for the desired state.
	desired, err := to(cmd.Context(), dev)
	cobra.CheckErr(err)
	// Only create one file per plan.
	tf, err := migrate.NewTemplateFormatter(nameTmpl, contentTmpl)
	cobra.CheckErr(err)
	// Plan the changes and create a new migration file.
	pl := migrate.NewPlanner(dev, dir, migrate.WithFormatter(tf))
	var name string
	if len(args) > 0 {
		name = args[0]
	}
	plan, err := pl.Plan(cmd.Context(), name, desired)
	cobra.CheckErr(err)
	// Write the plan to a new file.
	cobra.CheckErr(pl.WritePlan(plan))
	// TODO(masseelch): clean up dev after reading the state from migration dir.
}

// dir returns a migrate.Dir to use as migration directory. For now only local directories are supported.
func dir() (migrate.Dir, error) {
	parts := strings.SplitN(MigrateFlags.DirURL, "://", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid dir url %q", MigrateFlags.DirURL)
	}
	switch parts[0] {
	case "file":
		return migrate.NewLocalDir(parts[1])
	default:
		return nil, fmt.Errorf("unsupported driver %q", parts[0])
	}
}

// to returns a migrate.StateReader for the given to flag.
func to(ctx context.Context, d *Driver) (migrate.StateReader, error) {
	parts := strings.SplitN(MigrateFlags.ToURL, "://", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid driver url %q", MigrateFlags.ToURL)
	}
	schemas := MigrateFlags.Schemas
	if n, err := SchemaNameFromURL(ctx, parts[0]); n != "" {
		cobra.CheckErr(err)
		schemas = append(schemas, n)
	}
	switch parts[0] {
	case "file": // hcl file
		f, err := ioutil.ReadFile(parts[1])
		if err != nil {
			return nil, err
		}
		realm := &schema.Realm{}
		if err := d.UnmarshalSpec(f, realm); err != nil {
			return nil, err
		}
		if len(schemas) > 0 {
			// Validate all schemas in file were selected by user.
			sm := make(map[string]bool, len(schemas))
			for _, s := range schemas {
				sm[s] = true
			}
			for _, s := range realm.Schemas {
				if !sm[s.Name] {
					return nil, fmt.Errorf("schema %q from file %q is not requested (all schemas in HCL must be requested)", s.Name, parts[1])
				}
			}
		}
		if norm, ok := d.Driver.(schema.Normalizer); ok {
			realm, err = norm.NormalizeRealm(ctx, realm)
			if err != nil {
				return nil, err
			}
		}
		return migrate.Realm(realm), nil
	default: // database connection
		drv, err := DefaultMux.OpenAtlas(MigrateFlags.ToURL)
		if err != nil {
			return nil, err
		}
		return migrate.Conn(drv, &schema.InspectRealmOption{Schemas: schemas}), nil
	}
}

var (
	funcMap = template.FuncMap{
		"now": func() string { return time.Now().Format("20060102150405") },
		"sem": func(s string) string {
			if !strings.HasSuffix(s, ";") {
				return s + ";"
			}
			return s
		},
	}
	nameTmpl = template.Must(template.New("name").Funcs(funcMap).Parse(
		"{{ now }}{{ with .Name }}_{{ . }}{{ end }}.sql",
	))
	contentTmpl = template.Must(template.New("content").Funcs(funcMap).Parse(
		"{{ range .Changes }}{{ with .Comment }}-- {{ println . }}{{ end }}{{ println (sem .Cmd) }}{{ end }}",
	))
)
