package cmdapi

import (
	"ariga.io/atlas/cmd/atlas/internal/cmdapi"
)

type (
	// MigrateApplyFlags is the flags for the migrate apply command.
	MigrateApplyFlags = cmdapi.MigrateApplyFlags
)

var (
	// MigrateApply is the run function for the migrate apply command.
	MigrateApply = cmdapi.MigrateApplyRun
)
