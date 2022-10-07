// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cmdapi

import (
	"errors"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	flagAutoApprove = "auto-approve"
	flagDevURL      = "dev-url"
	flagDryRun      = "dry-run"
	flagDSN         = "dsn" // deprecated in favor of flagURL
	flagExclude     = "exclude"
	flagFile        = "file"
	flagSchema      = "schema"
	flagURL         = "url"
	flagVar         = "var"
)

func addFlagAutoApprove(set *pflag.FlagSet, target *bool) {
	set.BoolVar(target, flagAutoApprove, false, "apply changes without prompting for approval")
}

func addFlagDSN(set *pflag.FlagSet, target *string) {
	set.StringVarP(target, flagDSN, "d", "", "")
	cobra.CheckErr(set.MarkDeprecated(flagDSN, "please use --url instead"))
}

func addFlagDevURL(set *pflag.FlagSet, target *string) {
	set.StringVar(
		target,
		flagDevURL,
		"",
		"[driver://username:password@address/dbname?param=value] select a dev database using the URL format",
	)
}

func addFlagDryRun(set *pflag.FlagSet, target *bool) {
	set.BoolVar(target, flagDryRun, false, "print SQL without executing it")
}

func addFlagExclude(set *pflag.FlagSet, target *[]string) {
	set.StringSliceVar(
		target,
		flagExclude,
		nil,
		"list of glob patterns used to filter resources from applying",
	)
}

func addFlagURL(set *pflag.FlagSet, target *string) {
	set.StringVarP(
		target,
		flagURL, "u",
		"",
		"[driver://username:password@address/dbname?param=value] select a resource using the URL format",
	)
}

func addFlagSchema(set *pflag.FlagSet, target *[]string) {
	set.StringSliceVarP(
		target,
		flagSchema, "s",
		nil,
		"set schema names",
	)
}

func dsn2url(cmd *cobra.Command) error {
	dsnF, urlF := cmd.Flag(flagDSN), cmd.Flag(flagURL)
	switch {
	case dsnF == nil:
	case dsnF.Changed && urlF.Changed:
		return errors.New(`both flags "url" and "dsn" were set`)
	case dsnF.Changed && !urlF.Changed:
		return cmd.Flags().Set(flagURL, dsnF.Value.String())
	}
	return nil
}

// maySetFlag sets the flag with the provided name to envVal if such a flag exists
// on the cmd, it was not set by the user via the command line and if envVal is not
// an empty string.
func maySetFlag(cmd *cobra.Command, name, envVal string) error {
	fl := cmd.Flag(name)
	if fl == nil {
		return nil
	}
	if fl.Changed {
		return nil
	}
	if envVal == "" {
		return nil
	}
	return cmd.Flags().Set(name, envVal)
}
