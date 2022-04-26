// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package revision

import (
	"fmt"
)

const (
	// Label holds the string label denoting the revision type in the database.
	Label = "revision"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "version"
	// FieldDescription holds the string denoting the description field in the database.
	FieldDescription = "description"
	// FieldExecutionState holds the string denoting the execution_state field in the database.
	FieldExecutionState = "execution_state"
	// FieldExecutedAt holds the string denoting the executed_at field in the database.
	FieldExecutedAt = "executed_at"
	// FieldExecutionTime holds the string denoting the execution_time field in the database.
	FieldExecutionTime = "execution_time"
	// FieldHash holds the string denoting the hash field in the database.
	FieldHash = "hash"
	// FieldOperatorVersion holds the string denoting the operator_version field in the database.
	FieldOperatorVersion = "operator_version"
	// FieldMeta holds the string denoting the meta field in the database.
	FieldMeta = "meta"
	// Table holds the table name of the revision in the database.
	Table = "atlas_schema_revisions"
)

// Columns holds all SQL columns for revision fields.
var Columns = []string{
	FieldID,
	FieldDescription,
	FieldExecutionState,
	FieldExecutedAt,
	FieldExecutionTime,
	FieldHash,
	FieldOperatorVersion,
	FieldMeta,
}

// ValidColumn reports if the column name is valid (part of the table columns).
func ValidColumn(column string) bool {
	for i := range Columns {
		if column == Columns[i] {
			return true
		}
	}
	return false
}

// ExecutionState defines the type for the "execution_state" enum field.
type ExecutionState string

// ExecutionState values.
const (
	ExecutionStateOngoing ExecutionState = "ongoing"
	ExecutionStateOk      ExecutionState = "ok"
	ExecutionStateError   ExecutionState = "error"
)

func (es ExecutionState) String() string {
	return string(es)
}

// ExecutionStateValidator is a validator for the "execution_state" field enum values. It is called by the builders before save.
func ExecutionStateValidator(es ExecutionState) error {
	switch es {
	case ExecutionStateOngoing, ExecutionStateOk, ExecutionStateError:
		return nil
	default:
		return fmt.Errorf("revision: invalid enum value for execution_state field: %q", es)
	}
}
