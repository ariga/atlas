// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migrate

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ariga.io/atlas/sql/internal/migrate/revision"
	"entgo.io/ent/dialect/sql"
)

// Revision is the model entity for the Revision schema.
type Revision struct {
	config `json:"-"`
	// ID of the ent.
	ID string `json:"id,omitempty"`
	// Description holds the value of the "description" field.
	Description string `json:"description,omitempty"`
	// ExecutionState holds the value of the "execution_state" field.
	ExecutionState revision.ExecutionState `json:"execution_state,omitempty"`
	// ExecutedAt holds the value of the "executed_at" field.
	ExecutedAt time.Time `json:"executed_at,omitempty"`
	// ExecutionTime holds the value of the "execution_time" field.
	ExecutionTime time.Duration `json:"execution_time,omitempty"`
	// Hash holds the value of the "hash" field.
	Hash string `json:"hash,omitempty"`
	// OperatorVersion holds the value of the "operator_version" field.
	OperatorVersion string `json:"operator_version,omitempty"`
	// Meta holds the value of the "meta" field.
	Meta map[string]string `json:"meta,omitempty"`
}

// scanValues returns the types for scanning values from sql.Rows.
func (*Revision) scanValues(columns []string) ([]interface{}, error) {
	values := make([]interface{}, len(columns))
	for i := range columns {
		switch columns[i] {
		case revision.FieldMeta:
			values[i] = new([]byte)
		case revision.FieldExecutionTime:
			values[i] = new(sql.NullInt64)
		case revision.FieldID, revision.FieldDescription, revision.FieldExecutionState, revision.FieldHash, revision.FieldOperatorVersion:
			values[i] = new(sql.NullString)
		case revision.FieldExecutedAt:
			values[i] = new(sql.NullTime)
		default:
			return nil, fmt.Errorf("unexpected column %q for type Revision", columns[i])
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the Revision fields.
func (r *Revision) assignValues(columns []string, values []interface{}) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case revision.FieldID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value.Valid {
				r.ID = value.String
			}
		case revision.FieldDescription:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field description", values[i])
			} else if value.Valid {
				r.Description = value.String
			}
		case revision.FieldExecutionState:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field execution_state", values[i])
			} else if value.Valid {
				r.ExecutionState = revision.ExecutionState(value.String)
			}
		case revision.FieldExecutedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field executed_at", values[i])
			} else if value.Valid {
				r.ExecutedAt = value.Time
			}
		case revision.FieldExecutionTime:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field execution_time", values[i])
			} else if value.Valid {
				r.ExecutionTime = time.Duration(value.Int64)
			}
		case revision.FieldHash:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field hash", values[i])
			} else if value.Valid {
				r.Hash = value.String
			}
		case revision.FieldOperatorVersion:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field operator_version", values[i])
			} else if value.Valid {
				r.OperatorVersion = value.String
			}
		case revision.FieldMeta:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field meta", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &r.Meta); err != nil {
					return fmt.Errorf("unmarshal field meta: %w", err)
				}
			}
		}
	}
	return nil
}

// Update returns a builder for updating this Revision.
// Note that you need to call Revision.Unwrap() before calling this method if this Revision
// was returned from a transaction, and the transaction was committed or rolled back.
func (r *Revision) Update() *RevisionUpdateOne {
	return (&RevisionClient{config: r.config}).UpdateOne(r)
}

// Unwrap unwraps the Revision entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (r *Revision) Unwrap() *Revision {
	tx, ok := r.config.driver.(*txDriver)
	if !ok {
		panic("migrate: Revision is not a transactional entity")
	}
	r.config.driver = tx.drv
	return r
}

// String implements the fmt.Stringer.
func (r *Revision) String() string {
	var builder strings.Builder
	builder.WriteString("Revision(")
	builder.WriteString(fmt.Sprintf("id=%v", r.ID))
	builder.WriteString(", description=")
	builder.WriteString(r.Description)
	builder.WriteString(", execution_state=")
	builder.WriteString(fmt.Sprintf("%v", r.ExecutionState))
	builder.WriteString(", executed_at=")
	builder.WriteString(r.ExecutedAt.Format(time.ANSIC))
	builder.WriteString(", execution_time=")
	builder.WriteString(fmt.Sprintf("%v", r.ExecutionTime))
	builder.WriteString(", hash=")
	builder.WriteString(r.Hash)
	builder.WriteString(", operator_version=")
	builder.WriteString(r.OperatorVersion)
	builder.WriteString(", meta=")
	builder.WriteString(fmt.Sprintf("%v", r.Meta))
	builder.WriteByte(')')
	return builder.String()
}

// Revisions is a parsable slice of Revision.
type Revisions []*Revision

func (r Revisions) config(cfg config) {
	for _i := range r {
		r[_i].config = cfg
	}
}
