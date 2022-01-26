// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package action

import (
	"context"
	"strings"

	"github.com/spf13/cobra"
)

var (
	// DiffFlags are the flags used in the Diff command.
	DiffFlags struct {
		FromDSN string
		ToDSN   string
	}
	// DiffCmd represents the diff command.
	DiffCmd = &cobra.Command{
		Use:   "diff",
		Short: "Calculate and print the diff between two schemas.",
		Long: "`atlas schema diff`" + ` connects to two given databases, inspects
them, calculates the difference in their schemas, and prints a plan of
SQL queries to bring the "from" database to the schema of the "to" database.`,
		Run: CmdDiffRun,
	}
)

func init() {
	schemaCmd.AddCommand(DiffCmd)
	DiffCmd.Flags().StringVarP(&DiffFlags.FromDSN, "from", "", "", "[driver://username:password@protocol(address)/dbname?param=value] Select data source using the dsn format")
	DiffCmd.Flags().StringVarP(&DiffFlags.ToDSN, "to", "", "", "[driver://username:password@protocol(address)/dbname?param=value] Select data source using the dsn format")
	cobra.CheckErr(DiffCmd.MarkFlagRequired("from"))
	cobra.CheckErr(DiffCmd.MarkFlagRequired("to"))
}

// CmdDiffRun connects to the given databases, and prints an SQL plan to get from
// the "from" schema to the "to" schema.
func CmdDiffRun(cmd *cobra.Command, args []string) {
	fromDriver, err := defaultMux.OpenAtlas(DiffFlags.FromDSN)
	cobra.CheckErr(err)
	toDriver, err := defaultMux.OpenAtlas(DiffFlags.ToDSN)
	cobra.CheckErr(err)
	fromName, err := SchemaNameFromDSN(DiffFlags.FromDSN)
	cobra.CheckErr(err)
	toName, err := SchemaNameFromDSN(DiffFlags.ToDSN)
	cobra.CheckErr(err)
	ctx := context.Background()
	fromSchema, err := fromDriver.InspectSchema(ctx, fromName, nil)
	cobra.CheckErr(err)
	toSchema, err := toDriver.InspectSchema(ctx, toName, nil)
	cobra.CheckErr(err)
	// SchemaDiff checks for name equality which is irrelevant in the case
	// the user wants to compare their contents, if the names are different
	// we reset them to allow the comparison.
	if fromName != toName {
		toSchema.Name = ""
		fromSchema.Name = ""
	}
	diff, err := toDriver.SchemaDiff(fromSchema, toSchema)
	cobra.CheckErr(err)
	p, err := toDriver.PlanChanges(ctx, "plan", diff)
	cobra.CheckErr(err)
	if len(p.Changes) == 0 {
		schemaCmd.Println("Schemas are synced, no changes to be made.")
		return
	}
	for _, c := range p.Changes {
		if c.Comment != "" {
			cmd.Println("--", strings.ToUpper(c.Comment[:1])+c.Comment[1:])
		}
		cmd.Println(c.Cmd)
	}
}
