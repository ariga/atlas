// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migrate

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"ariga.io/atlas/sql/schema"
)

type (
	// A Plan defines a planned changeset that its execution brings the database to
	// the new desired state. Additional information is calculated by the different
	// drivers to indicate if the changeset is transactional (can be rolled-back) and
	// reversible (a down file can be generated to it).
	Plan struct {
		// Version and Name of the plan. Provided by the user or auto-generated.
		Version, Name string

		// Reversible describes if the changeset is reversible.
		Reversible bool

		// Transactional describes if the changeset is transactional.
		Transactional bool

		// Changes defines the list of changeset in the plan.
		Changes []*Change
	}

	// A Change of migration.
	Change struct {
		// Cmd or statement to execute.
		Cmd string

		// Args for placeholder parameters in the statement above.
		Args []any

		// A Comment describes the change.
		Comment string

		// Reverse contains the "reversed" statement(s) if
		// the command is reversible.
		Reverse any // string | []string

		// The Source that caused this change, or nil.
		Source schema.Change
	}
)

// ReverseStmts returns the reverse statements of a Change, if any.
func (c *Change) ReverseStmts() (cmd []string, err error) {
	switch r := c.Reverse.(type) {
	case nil:
	case string:
		cmd = []string{r}
	case []string:
		cmd = r
	default:
		err = fmt.Errorf("sql/migrate: unexpected type %T for reverse commands", r)
	}
	return
}

type (
	// The Driver interface must be implemented by the different dialects to support database
	// migration authoring/planning and applying. ExecQuerier, Inspector and Differ, provide
	// basic schema primitives for inspecting database schemas, calculate the difference between
	// schema elements, and executing raw SQL statements. The PlanApplier interface wraps the
	// methods for generating migration plan for applying the actual changes on the database.
	Driver interface {
		schema.Differ
		schema.ExecQuerier
		schema.Inspector
		PlanApplier
	}

	// PlanApplier wraps the methods for planning and applying changes
	// on the database.
	PlanApplier interface {
		// PlanChanges returns a migration plan for applying the given changeset.
		PlanChanges(context.Context, string, []schema.Change, ...PlanOption) (*Plan, error)

		// ApplyChanges is responsible for applying the given changeset.
		// An error may return from ApplyChanges if the driver is unable
		// to execute a change.
		ApplyChanges(context.Context, []schema.Change, ...PlanOption) error
	}

	// PlanOptions holds the migration plan options to be used by PlanApplier.
	PlanOptions struct {
		// PlanWithSchemaQualifier allows setting a custom schema to prefix
		// tables and other resources. An empty string indicates no qualifier.
		SchemaQualifier *string
		// Indent is the string to use for indentation.
		// If empty, no indentation is used.
		Indent string
		// Mode represents the migration planning mode to be used. If not specified, the driver picks its default.
		// This is useful to indicate to the driver whether the context is a live database, an empty one, or the
		// versioned migration workflow.
		Mode PlanMode
	}

	// PlanMode defines the plan mode to use.
	PlanMode uint8

	// PlanOption allows configuring a drivers' plan using functional arguments.
	PlanOption func(*PlanOptions)

	// StateReader wraps the method for reading a database/schema state.
	// The types below provides a few builtin options for reading a state
	// from a migration directory, a static object (e.g. a parsed file).
	StateReader interface {
		ReadState(ctx context.Context) (*schema.Realm, error)
	}

	// The StateReaderFunc type is an adapter to allow the use of
	// ordinary functions as state readers.
	StateReaderFunc func(ctx context.Context) (*schema.Realm, error)
)

// ReadState calls f(ctx).
func (f StateReaderFunc) ReadState(ctx context.Context) (*schema.Realm, error) {
	return f(ctx)
}

// List of migration planning modes.
const (
	PlanModeUnset    PlanMode = iota // Driver default.
	PlanModeInPlace                  // Changes are applied inplace (e.g., 'schema diff').
	PlanModeDeferred                 // Changes are planned for future applying (e.g., 'migrate diff').
	PlanModeDump                     // Schema creation dump (e.g., 'schema inspect').
)

// Is reports whether m is match the given mode.
func (m PlanMode) Is(m1 PlanMode) bool {
	return m == m1 || m&m1 != 0
}

// ErrNoPlan is returned by Plan when there is no change between the two states.
var ErrNoPlan = errors.New("sql/migrate: no plan for matched states")

// Realm returns a StateReader for the static Realm object.
func Realm(r *schema.Realm) StateReader {
	return StateReaderFunc(func(context.Context) (*schema.Realm, error) {
		return r, nil
	})
}

// Schema returns a StateReader for the static Schema object.
func Schema(s *schema.Schema) StateReader {
	return StateReaderFunc(func(context.Context) (*schema.Realm, error) {
		r := &schema.Realm{Schemas: []*schema.Schema{s}}
		if s.Realm != nil {
			r.Attrs = s.Realm.Attrs
		}
		s.Realm = r
		return r, nil
	})
}

// RealmConn returns a StateReader for a Driver connected to a database.
func RealmConn(drv Driver, opts *schema.InspectRealmOption) StateReader {
	return StateReaderFunc(func(ctx context.Context) (*schema.Realm, error) {
		return drv.InspectRealm(ctx, opts)
	})
}

// SchemaConn returns a StateReader for a Driver connected to a schema.
func SchemaConn(drv Driver, name string, opts *schema.InspectOptions) StateReader {
	return StateReaderFunc(func(ctx context.Context) (*schema.Realm, error) {
		s, err := drv.InspectSchema(ctx, name, opts)
		if err != nil {
			return nil, err
		}
		return Schema(s).ReadState(ctx)
	})
}

type (
	// Planner can plan the steps to take to migrate from one state to another. It uses the enclosed Dir to
	// those changes to versioned migration files.
	Planner struct {
		drv      Driver              // driver to use
		dir      Dir                 // where migration files are stored and read from
		fmt      Formatter           // how to format a plan to migration files
		sum      bool                // whether to create a sum file for the migration directory
		planOpts []PlanOption        // plan options
		diffOpts []schema.DiffOption // diff options
	}

	// PlannerOption allows managing a Planner using functional arguments.
	PlannerOption func(*Planner)

	// A RevisionReadWriter wraps the functionality for reading and writing migration revisions in a database table.
	RevisionReadWriter interface {
		// Ident returns an object identifies this history table.
		Ident() *TableIdent
		// ReadRevisions returns all revisions.
		ReadRevisions(context.Context) ([]*Revision, error)
		// ReadRevision returns a revision by version.
		// Returns ErrRevisionNotExist if the version does not exist.
		ReadRevision(context.Context, string) (*Revision, error)
		// WriteRevision saves the revision to the storage.
		WriteRevision(context.Context, *Revision) error
		// DeleteRevision deletes a revision by version from the storage.
		DeleteRevision(context.Context, string) error
	}

	// A Revision denotes an applied migration in a deployment. Used to track migration executions state of a database.
	Revision struct {
		Version         string        `json:"Version"`             // Version of the migration.
		Description     string        `json:"Description"`         // Description of this migration.
		Type            RevisionType  `json:"Type"`                // Type of the migration.
		Applied         int           `json:"Applied"`             // Applied amount of statements in the migration.
		Total           int           `json:"Total"`               // Total amount of statements in the migration.
		ExecutedAt      time.Time     `json:"ExecutedAt"`          // ExecutedAt is the starting point of execution.
		ExecutionTime   time.Duration `json:"ExecutionTime"`       // ExecutionTime of the migration.
		Error           string        `json:"Error,omitempty"`     // Error of the migration, if any occurred.
		ErrorStmt       string        `json:"ErrorStmt,omitempty"` // ErrorStmt is the statement that raised Error.
		Hash            string        `json:"-"`                   // Hash of migration file.
		PartialHashes   []string      `json:"-"`                   // PartialHashes is the hashes of applied statements.
		OperatorVersion string        `json:"OperatorVersion"`     // OperatorVersion that executed this migration.
	}

	// RevisionType defines the type of the revision record in the history table.
	RevisionType uint

	// Executor is responsible to manage and execute a set of migration files against a database.
	Executor struct {
		drv         Driver             // The Driver to access and manage the database.
		dir         Dir                // The Dir with migration files to use.
		rrw         RevisionReadWriter // The RevisionReadWriter to read and write database revisions to.
		log         Logger             // The Logger to use.
		order       ExecOrder          // The order to execute the migration files.
		baselineVer string             // Start the first migration after the given baseline version.
		allowDirty  bool               // Allow start working on a non-clean database.
		operator    string             // Revision.OperatorVersion
	}

	// ExecutorOption allows configuring an Executor using functional arguments.
	ExecutorOption func(*Executor) error
)

const (
	// RevisionTypeUnknown represents an unknown revision type.
	// This type is unexpected and exists here to only ensure
	// the type is not set to the zero value.
	RevisionTypeUnknown RevisionType = 0

	// RevisionTypeBaseline represents a baseline revision. Note that only
	// the first record can represent a baseline migration and most of its
	// fields are set to the zero value.
	RevisionTypeBaseline RevisionType = 1 << (iota - 1)

	// RevisionTypeExecute represents a migration that was executed.
	RevisionTypeExecute

	// RevisionTypeResolved represents a migration that was resolved. A migration
	// script that was script executed and then resolved should set its Type to
	// RevisionTypeExecute | RevisionTypeResolved.
	RevisionTypeResolved
)

// Has returns if the given flag is set.
func (r RevisionType) Has(f RevisionType) bool {
	return r&f != 0
}

// String implements fmt.Stringer.
func (r RevisionType) String() string {
	switch r {
	case RevisionTypeBaseline:
		return "baseline"
	case RevisionTypeExecute:
		return "applied"
	case RevisionTypeResolved:
		return "manually set"
	case RevisionTypeExecute | RevisionTypeResolved:
		return "applied + manually set"
	default:
		return fmt.Sprintf("unknown (%04b)", r)
	}
}

// MarshalText implements encoding.TextMarshaler.
func (r RevisionType) MarshalText() ([]byte, error) {
	return []byte(r.String()), nil
}

// NewPlanner creates a new Planner.
func NewPlanner(drv Driver, dir Dir, opts ...PlannerOption) *Planner {
	p := &Planner{drv: drv, dir: dir, sum: true}
	for _, opt := range opts {
		opt(p)
	}
	if p.fmt == nil {
		p.fmt = DefaultFormatter
	}
	return p
}

// PlanWithSchemaQualifier allows setting a custom schema to prefix tables and
// other resources. An empty string indicates no prefix.
//
// Note, this options require the changes to be scoped to one
// schema and returns an error otherwise.
func PlanWithSchemaQualifier(q string) PlannerOption {
	return func(p *Planner) {
		p.planOpts = append(p.planOpts, func(o *PlanOptions) {
			o.SchemaQualifier = &q
		})
	}
}

// PlanWithIndent allows generating SQL statements with indentation.
// An empty string indicates no indentation.
func PlanWithIndent(indent string) PlannerOption {
	return func(p *Planner) {
		p.planOpts = append(p.planOpts, func(o *PlanOptions) {
			o.Indent = indent
		})
	}
}

// PlanWithMode allows setting a custom plan mode.
func PlanWithMode(m PlanMode) PlannerOption {
	return func(p *Planner) {
		p.planOpts = append(p.planOpts, func(o *PlanOptions) {
			o.Mode = m
		})
	}
}

// PlanWithDiffOptions allows setting custom diff options.
func PlanWithDiffOptions(opts ...schema.DiffOption) PlannerOption {
	return func(p *Planner) {
		p.diffOpts = append(p.diffOpts, opts...)
	}
}

// PlanFormat sets the Formatter of a Planner.
func PlanFormat(fmt Formatter) PlannerOption {
	return func(p *Planner) {
		p.fmt = fmt
	}
}

// PlanWithChecksum allows setting if the hash-sum functionality
// for the migration directory is enabled or not.
func PlanWithChecksum(b bool) PlannerOption {
	return func(p *Planner) {
		p.sum = b
	}
}

var (
	// WithFormatter calls PlanFormat.
	// Deprecated: use PlanFormat instead.
	WithFormatter = PlanFormat
	// DisableChecksum calls PlanWithChecksum(false).
	// Deprecated: use PlanWithoutChecksum instead.
	DisableChecksum = func() PlannerOption { return PlanWithChecksum(false) }
)

// Plan calculates the migration Plan required for moving the current state (from) state to
// the next state (to). A StateReader can be a directory, static schema elements or a Driver connection.
func (p *Planner) Plan(ctx context.Context, name string, to StateReader) (*Plan, error) {
	return p.plan(ctx, name, to, true)
}

// PlanSchema is like Plan but limits its scope to the schema connection.
// Note, the operation fails in case the connection was not set to a schema.
func (p *Planner) PlanSchema(ctx context.Context, name string, to StateReader) (*Plan, error) {
	return p.plan(ctx, name, to, false)
}

func (p *Planner) plan(ctx context.Context, name string, to StateReader, realmScope bool) (*Plan, error) {
	current, err := p.current(ctx, realmScope)
	if err != nil {
		return nil, err
	}
	desired, err := to.ReadState(ctx)
	if err != nil {
		return nil, err
	}
	var changes []schema.Change
	switch {
	case realmScope:
		changes, err = p.drv.RealmDiff(current, desired, p.diffOpts...)
	default:
		switch n, m := len(current.Schemas), len(desired.Schemas); {
		case n == 0:
			return nil, errors.New("no schema was found in current state after replaying migration directory")
		case n > 1:
			return nil, fmt.Errorf("%d schemas were found in current state after replaying migration directory", len(current.Schemas))
		case m == 0:
			return nil, errors.New("no schema was found in desired state")
		case m > 1:
			return nil, fmt.Errorf("%d schemas were found in desired state; expect 1", len(desired.Schemas))
		default:
			s1, s2 := *current.Schemas[0], *desired.Schemas[0]
			// Avoid comparing schema names when scope is limited to one schema,
			// and the schema qualifier is controlled by the caller.
			if s1.Name != s2.Name {
				s1.Name = s2.Name
			}
			changes, err = p.drv.SchemaDiff(&s1, &s2, p.diffOpts...)
		}
	}
	if err != nil {
		return nil, err
	}
	if len(changes) == 0 {
		return nil, ErrNoPlan
	}
	return p.drv.PlanChanges(ctx, name, changes, p.planOpts...)
}

// Checkpoint calculate the current state of the migration directory by executing its files,
// and return a migration (checkpoint) Plan that represents its states.
func (p *Planner) Checkpoint(ctx context.Context, name string) (*Plan, error) {
	return p.checkpoint(ctx, name, true)
}

// CheckpointSchema is like Checkpoint but limits its scope to the schema connection.
// Note, the operation fails in case the connection was not set to a schema.
func (p *Planner) CheckpointSchema(ctx context.Context, name string) (*Plan, error) {
	return p.checkpoint(ctx, name, false)
}

func (p *Planner) checkpoint(ctx context.Context, name string, realmScope bool) (*Plan, error) {
	current, err := p.current(ctx, realmScope)
	if err != nil {
		return nil, err
	}
	var changes []schema.Change
	switch {
	case realmScope:
		changes, err = p.drv.RealmDiff(schema.NewRealm(), current, p.diffOpts...)
	default:
		switch n := len(current.Schemas); {
		case n == 0:
			return nil, errors.New("no schema was found in current state after replaying migration directory")
		case n > 1:
			return nil, fmt.Errorf("%d schemas were found in current state after replaying migration directory", len(current.Schemas))
		default:
			s1 := current.Schemas[0]
			s2 := schema.New(s1.Name).AddAttrs(s1.Attrs...)
			changes, err = p.drv.SchemaDiff(s2, s1, p.diffOpts...)
		}
	}
	if err != nil {
		return nil, err
	}
	// No changes mean an empty checkpoint.
	if len(changes) == 0 {
		return &Plan{Name: name}, nil
	}
	return p.drv.PlanChanges(ctx, name, changes, p.planOpts...)
}

// current returns the current realm state.
func (p *Planner) current(ctx context.Context, realmScope bool) (*schema.Realm, error) {
	from, err := NewExecutor(p.drv, p.dir, NopRevisionReadWriter{})
	if err != nil {
		return nil, err
	}
	return from.Replay(ctx, func() StateReader {
		if realmScope {
			return RealmConn(p.drv, nil)
		}
		// In case the scope is the schema connection,
		// inspect it and return its connected realm.
		return SchemaConn(p.drv, "", nil)
	}())
}

// WritePlan writes the given Plan to the Dir based on the configured Formatter.
func (p *Planner) WritePlan(plan *Plan) error {
	// Format the plan into files.
	files, err := p.fmt.Format(plan)
	if err != nil {
		return err
	}
	// Store the files in the migration directory.
	for _, f := range files {
		if err := p.dir.WriteFile(f.Name(), f.Bytes()); err != nil {
			return err
		}
	}
	return p.writeSum()
}

// WriteCheckpoint writes the given Plan as a checkpoint file to the Dir based on the configured Formatter.
func (p *Planner) WriteCheckpoint(plan *Plan, tag string) error {
	ck, ok := p.dir.(CheckpointDir)
	if !ok {
		return fmt.Errorf("checkpoint is not supported by %T", p.dir)
	}
	// Format the plan into files.
	files, err := p.fmt.Format(plan)
	if err != nil {
		return err
	}
	if len(files) != 1 {
		return fmt.Errorf("expected one checkpoint file, got %d", len(files))
	}
	if err := ck.WriteCheckpoint(files[0].Name(), tag, files[0].Bytes()); err != nil {
		return err
	}
	return p.writeSum()
}

// writeSum writes the sum file to the Dir, if enabled.
func (p *Planner) writeSum() error {
	if !p.sum {
		return nil
	}
	sum, err := p.dir.Checksum()
	if err != nil {
		return err
	}
	return WriteSumFile(p.dir, sum)
}

var (
	// ErrNoPendingFiles is returned if there are no pending migration files to execute on the managed database.
	ErrNoPendingFiles = errors.New("sql/migrate: no pending migration files")
	// ErrSnapshotUnsupported is returned if there is no Snapshoter given.
	ErrSnapshotUnsupported = errors.New("sql/migrate: driver does not support taking a database snapshot")
	// ErrCleanCheckerUnsupported is returned if there is no CleanChecker given.
	ErrCleanCheckerUnsupported = errors.New("sql/migrate: driver does not support checking if database is clean")
	// ErrRevisionNotExist is returned if the requested revision is not found in the storage.
	ErrRevisionNotExist = errors.New("sql/migrate: revision not found")
)

// MissingMigrationError is returned if a revision is partially applied but
// the matching migration file is not found in the migration directory.
type MissingMigrationError struct{ Version, Description string }

// Error implements error.
func (e MissingMigrationError) Error() string {
	return fmt.Sprintf(
		"sql/migrate: missing migration: revision %q is partially applied but migration file was not found",
		fmt.Sprintf("%s_%s.sql", e.Version, e.Description),
	)
}

// NewExecutor creates a new Executor with default values.
func NewExecutor(drv Driver, dir Dir, rrw RevisionReadWriter, opts ...ExecutorOption) (*Executor, error) {
	if drv == nil {
		return nil, errors.New("sql/migrate: no driver given")
	}
	if dir == nil {
		return nil, errors.New("sql/migrate: no dir given")
	}
	if rrw == nil {
		return nil, errors.New("sql/migrate: no revision storage given")
	}
	ex := &Executor{drv: drv, dir: dir, rrw: rrw}
	for _, opt := range opts {
		if err := opt(ex); err != nil {
			return nil, err
		}
	}
	if ex.log == nil {
		ex.log = NopLogger{}
	}
	if _, ok := drv.(Snapshoter); !ok {
		return nil, ErrSnapshotUnsupported
	}
	if _, ok := drv.(CleanChecker); !ok {
		return nil, ErrCleanCheckerUnsupported
	}
	if ex.baselineVer != "" && ex.allowDirty {
		return nil, errors.New("sql/migrate: baseline and allow-dirty are mutually exclusive")
	}
	return ex, nil
}

// WithAllowDirty defines if we can start working on a non-clean database
// in the first migration execution.
func WithAllowDirty(b bool) ExecutorOption {
	return func(ex *Executor) error {
		ex.allowDirty = b
		return nil
	}
}

// WithBaselineVersion allows setting the baseline version of the database on the
// first migration. Hence, all versions up to and including this version are skipped.
func WithBaselineVersion(v string) ExecutorOption {
	return func(ex *Executor) error {
		ex.baselineVer = v
		return nil
	}
}

// WithLogger sets the Logger of an Executor.
func WithLogger(log Logger) ExecutorOption {
	return func(ex *Executor) error {
		ex.log = log
		return nil
	}
}

// ExecOrder defines the execution order to use.
type ExecOrder uint

const (
	// ExecOrderLinear is the default execution order mode.
	// It expects a linear history and fails if it encounters files that were
	// added out of order. For example, a new file was added with version lower
	// than the last applied revision.
	ExecOrderLinear ExecOrder = iota

	// ExecOrderLinearSkip is a softer version of ExecOrderLinear.
	// This means that if a new file is added with a version lower than the last
	// applied revision, it will be skipped.
	ExecOrderLinearSkip

	// ExecOrderNonLinear executes migration files that were added out of order.
	ExecOrderNonLinear
)

// WithExecOrder sets the execution order to use.
func WithExecOrder(o ExecOrder) ExecutorOption {
	return func(ex *Executor) error {
		ex.order = o
		return nil
	}
}

// WithOperatorVersion sets the operator version to save on the revisions
// when executing migration files.
func WithOperatorVersion(v string) ExecutorOption {
	return func(ex *Executor) error {
		ex.operator = v
		return nil
	}
}

// Pending returns all pending (not fully applied) migration files in the migration directory.
func (e *Executor) Pending(ctx context.Context) ([]File, error) {
	// Don't operate with a broken migration directory.
	if err := e.ValidateDir(ctx); err != nil {
		return nil, err
	}
	// Read all applied database revisions.
	revs, err := e.rrw.ReadRevisions(ctx)
	if err != nil {
		return nil, fmt.Errorf("sql/migrate: read revisions: %w", err)
	}
	all, err := e.dir.Files()
	if err != nil {
		return nil, fmt.Errorf("sql/migrate: read migration directory files: %w", err)
	}
	migrations := SkipCheckpointFiles(all)
	var pending []File
	switch {
	// If it is the first time we run.
	case len(revs) == 0:
		var cerr *NotCleanError
		if err = e.drv.(CleanChecker).CheckClean(ctx, e.rrw.Ident()); err != nil && !errors.As(err, &cerr) {
			return nil, err
		}
		// In case the workspace is not clean one of the flags is required.
		if cerr != nil && !e.allowDirty && e.baselineVer == "" {
			return nil, fmt.Errorf("%w. baseline version or allow-dirty is required", cerr)
		}
		if e.baselineVer != "" {
			baseline := FilesLastIndex(migrations, func(f File) bool {
				return f.Version() == e.baselineVer
			})
			if baseline == -1 {
				return nil, fmt.Errorf("baseline version %q not found", e.baselineVer)
			}
			f := migrations[baseline]
			// Write the first revision in the database as a baseline revision.
			if err := e.writeRevision(ctx, &Revision{Version: f.Version(), Description: f.Desc(), Type: RevisionTypeBaseline}); err != nil {
				return nil, err
			}
			pending = migrations[baseline+1:]
			// In case the "allow-dirty" option was set, or the database is clean,
			// the starting-point is the first migration file or the last checkpoint.
		} else if pending, err = FilesFromLastCheckpoint(e.dir); err != nil {
			return nil, err
		}
	// In case we applied a checkpoint, but it was only partially applied.
	case revs[len(revs)-1].Applied != revs[len(revs)-1].Total && len(all) > 0:
		if idx, found := slices.BinarySearchFunc(all, revs[len(revs)-1], func(f File, r *Revision) int {
			return strings.Compare(f.Version(), r.Version)
		}); found {
			if f, ok := all[idx].(CheckpointFile); ok && f.IsCheckpoint() {
				// There can only be one checkpoint file and it must be the first one applied.
				// Thus, we can consider all migrations following the checkpoint to be pending.
				return append([]File{f}, SkipCheckpointFiles(all[idx:])...), nil
			}
		}
		if len(migrations) == 0 {
			break // don't fall through the next case if there are no migrations
		}
		fallthrough // proceed normally
	// In case we applied/marked revisions in the past, and there is work to do.
	case len(migrations) > 0:
		var (
			last      = revs[len(revs)-1]
			partially = last.Applied != last.Total
			fn        = func(f File) bool { return f.Version() <= last.Version }
		)
		if partially {
			// If the last file is partially applied, we need to find the matching migration file in order to
			// continue execution at the correct statement.
			fn = func(f File) bool { return f.Version() == last.Version }
		}
		// Consider all migration files having a version < the latest revision version as pending. If the
		// last revision is partially applied, it is considered pending as well.
		idx := FilesLastIndex(migrations, fn)
		if idx == -1 {
			// If we cannot find the matching migration version for a partially applied migration,
			// error out since we cannot determine how to proceed from here.
			if partially {
				return nil, &MissingMigrationError{last.Version, last.Description}
			}
			// All migrations have a higher version than the latest revision. Take every migration file as pending.
			return migrations, nil
		}
		// If this file was not partially applied, take the next one.
		if last.Applied == last.Total {
			idx++
		}
		pending = migrations[idx:]
		// Capture all files (versions) between first and last revisions and ensure they
		// were actually applied. Then, error or execute according to the execution order.
		// Note, "first" is computed as it can be set to the first checkpoint, which may
		// not be the first migration file.
		if first := slices.IndexFunc(migrations[:idx], func(f File) bool {
			return f.Version() >= revs[0].Version
		}); first != -1 && first < idx && e.order != ExecOrderLinearSkip {
			var skipped []File
			for _, f := range migrations[first:idx] {
				if _, found := slices.BinarySearchFunc(revs, f, func(r *Revision, f File) int {
					return strings.Compare(r.Version, f.Version())
				}); !found {
					skipped = append(skipped, f)
				}
			}
			switch {
			case len(skipped) == 0:
			case e.order == ExecOrderNonLinear:
				pending = append(skipped, pending...)
			case e.order == ExecOrderLinear:
				return nil, &HistoryNonLinearError{OutOfOrder: skipped, Pending: pending}
			}
		}
	}
	if len(pending) == 0 {
		return nil, ErrNoPendingFiles
	}
	return pending, nil
}

// Execute executes the given migration file on the database. If it sees a file, that has been partially applied, it
// will continue with the next statement in line.
func (e *Executor) Execute(ctx context.Context, m File) (err error) {
	hf, err := e.dir.Checksum()
	if err != nil {
		return fmt.Errorf("sql/migrate: compute hash: %w", err)
	}
	hash, err := hf.SumByName(m.Name())
	if err != nil {
		return fmt.Errorf("sql/migrate: scanning checksum from %q: %w", m.Name(), err)
	}
	stmts, err := e.fileStmts(m)
	if err != nil {
		return fmt.Errorf("sql/migrate: scanning statements from %q: %w", m.Name(), err)
	}
	// Create checksums for the statements.
	var (
		sums = make([]string, len(stmts))
		h    = sha256.New()
	)
	for i, stmt := range stmts {
		if _, err := h.Write([]byte(stmt)); err != nil {
			return err
		}
		sums[i] = base64.StdEncoding.EncodeToString(h.Sum(nil))
	}
	version := m.Version()
	// If there already is a revision with this version in the database,
	// and it is partially applied, continue where the last attempt was left off.
	r, err := e.rrw.ReadRevision(ctx, version)
	if err != nil && !errors.Is(err, ErrRevisionNotExist) {
		return fmt.Errorf("sql/migrate: read revision: %w", err)
	}
	if errors.Is(err, ErrRevisionNotExist) {
		// Haven't seen this file before, create a new revision.
		r = &Revision{
			Version:     version,
			Description: m.Desc(),
			Type:        RevisionTypeExecute,
			Total:       len(stmts),
			Hash:        hash,
		}
	}
	// Save once to mark as started in the database.
	if err = e.writeRevision(ctx, r); err != nil {
		return err
	}
	// Make sure to store the Revision information.
	defer func(ctx context.Context, e *Executor, r *Revision) {
		if err2 := e.writeRevision(ctx, r); err2 != nil {
			err = errors.Join(err, err2)
		}
	}(ctx, e, r)
	if r.Applied > 0 {
		// If the file has been applied partially before, check if the
		// applied statements have not changed.
		for i := 0; i < r.Applied; i++ {
			if i > len(sums) || sums[i] != strings.TrimPrefix(r.PartialHashes[i], "h1:") {
				err = HistoryChangedError{m.Name(), i + 1}
				e.log.Log(LogError{Error: err})
				return err
			}
		}
	}
	e.log.Log(LogFile{m, r.Version, r.Description, r.Applied})
	if err := e.fileChecks(ctx, m, r); err != nil {
		e.log.Log(LogError{Error: err})
		r.done()
		r.Error = err.Error()
		return err
	}
	for _, stmt := range stmts[r.Applied:] {
		e.log.Log(LogStmt{stmt})
		if _, err = e.drv.ExecContext(ctx, stmt); err != nil {
			e.log.Log(LogError{SQL: stmt, Error: err})
			r.done()
			r.ErrorStmt = stmt
			r.Error = err.Error()
			return fmt.Errorf("sql/migrate: executing statement %q from version %q: %w", stmt, r.Version, err)
		}
		r.PartialHashes = append(r.PartialHashes, "h1:"+sums[r.Applied])
		r.Applied++
		if err = e.writeRevision(ctx, r); err != nil {
			return err
		}
	}
	r.done()
	return
}

func (e *Executor) writeRevision(ctx context.Context, r *Revision) error {
	r.ExecutedAt = time.Now()
	r.OperatorVersion = e.operator
	if err := e.rrw.WriteRevision(ctx, r); err != nil {
		return fmt.Errorf("sql/migrate: write revision: %w", err)
	}
	return nil
}

// HistoryChangedError is returned if between two execution attempts already applied statements of a file have changed.
type HistoryChangedError struct {
	File string
	Stmt int
}

func (e HistoryChangedError) Error() string {
	return fmt.Sprintf("sql/migrate: history changed: statement %d from file %q changed", e.Stmt, e.File)
}

// HistoryNonLinearError is returned if the migration history is not linear. Means, a file was added out of order.
// The executor can be configured to ignore this error and continue execution. See WithExecOrder for details.
type HistoryNonLinearError struct {
	// OutOfOrder are the files that were added out of order.
	OutOfOrder []File
	// Pending are valid files that are still pending for execution.
	Pending []File
}

func (e HistoryNonLinearError) Error() string {
	names := make([]string, len(e.OutOfOrder))
	for i := range e.OutOfOrder {
		names[i] = e.OutOfOrder[i].Name()
	}
	f := fmt.Sprintf("files %s were", strings.Join(names, ", "))
	if len(e.OutOfOrder) == 1 {
		f = fmt.Sprintf("file %s was", names[0])
	}
	return fmt.Sprintf("migration %s added out of order. See: https://atlasgo.io/versioned/apply#non-linear-error", f)
}

// ExecuteN executes n pending migration files. If n<=0 all pending migration files are executed.
func (e *Executor) ExecuteN(ctx context.Context, n int) (err error) {
	pending, err := e.Pending(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		if n >= len(pending) {
			n = len(pending)
		}
		pending = pending[:n]
	}
	return e.exec(ctx, pending)
}

// ExecuteTo executes all pending migration files up to and including version.
func (e *Executor) ExecuteTo(ctx context.Context, version string) (err error) {
	pending, err := e.Pending(ctx)
	if err != nil {
		return err
	}
	// Strip pending files greater given version.
	switch idx := FilesLastIndex(pending, func(file File) bool {
		return file.Version() == version
	}); idx {
	case -1:
		return fmt.Errorf("sql/migrate: migration with version %q not found", version)
	default:
		pending = pending[:idx+1]
	}
	return e.exec(ctx, pending)
}

func (e *Executor) exec(ctx context.Context, files []File) error {
	revs, err := e.rrw.ReadRevisions(ctx)
	if err != nil {
		return fmt.Errorf("sql/migrate: read revisions: %w", err)
	}
	LogIntro(e.log, revs, files)
	for _, m := range files {
		if err := e.Execute(ctx, m); err != nil {
			return err
		}
	}
	e.log.Log(LogDone{})
	return err
}

type (
	replayConfig struct {
		version string // to which version to replay (inclusive)
	}
	// ReplayOption configures a migration directory replay behavior.
	ReplayOption func(*replayConfig)
)

// ReplayToVersion configures the last version to apply when replaying the migration directory.
func ReplayToVersion(v string) ReplayOption {
	return func(c *replayConfig) {
		c.version = v
	}
}

// Replay the migration directory and invoke the state to get back the inspection result.
func (e *Executor) Replay(ctx context.Context, r StateReader, opts ...ReplayOption) (_ *schema.Realm, err error) {
	c := &replayConfig{}
	for _, opt := range opts {
		opt(c)
	}
	// Clean up after ourselves.
	restore, err := e.drv.(Snapshoter).Snapshot(ctx)
	if err != nil {
		return nil, fmt.Errorf("sql/migrate: taking database snapshot: %w", err)
	}
	defer func() {
		if err2 := restore(ctx); err2 != nil {
			err = errors.Join(err, err2)
		}
	}()
	// Replay the migration directory on the database.
	switch {
	case c.version != "":
		err = e.ExecuteTo(ctx, c.version)
	default:
		err = e.ExecuteN(ctx, 0)
	}
	if err != nil && !errors.Is(err, ErrNoPendingFiles) {
		return nil, fmt.Errorf("sql/migrate: read migration directory state: %w", err)
	}
	return r.ReadState(ctx)
}

type (
	// Snapshoter wraps the Snapshot method.
	Snapshoter interface {
		// Snapshot takes a snapshot of the current database state and returns a function that can be called to restore
		// that state. Snapshot should return an error, if the current state can not be restored completely, e.g. if
		// there is a table already containing some rows.
		Snapshot(context.Context) (RestoreFunc, error)
	}

	// RestoreFunc is returned by the Snapshoter to explicitly restore the database state.
	RestoreFunc func(context.Context) error

	// TableIdent describes a table identifier returned by the revisions table.
	TableIdent struct {
		Name   string // name of the table.
		Schema string // optional schema.
	}

	// CleanChecker wraps the single CheckClean method.
	CleanChecker interface {
		// CheckClean checks if the connected realm or schema does not contain any resources besides the
		// revision history table. A NotCleanError is returned in case the connection is not-empty.
		CheckClean(context.Context, *TableIdent) error
	}

	// NotCleanError is returned when the connected dev-db is not in a clean state (aka it has schemas and tables).
	// This check is done to ensure no data is lost by overriding it when working on the dev-db.
	NotCleanError struct {
		Reason string        // reason why the database is considered not clean
		State  *schema.Realm // the state the dev-connection is in
	}
)

func (e *NotCleanError) Error() string {
	return "sql/migrate: connected database is not clean: " + e.Reason
}

// NopRevisionReadWriter is a RevisionReadWriter that does nothing.
// It is useful for one-time replay of the migration directory.
type NopRevisionReadWriter struct{}

// Ident implements RevisionsReadWriter.TableIdent.
func (NopRevisionReadWriter) Ident() *TableIdent {
	return nil
}

// ReadRevisions implements RevisionsReadWriter.ReadRevisions.
func (NopRevisionReadWriter) ReadRevisions(context.Context) ([]*Revision, error) {
	return nil, nil
}

// ReadRevision implements RevisionsReadWriter.ReadRevision.
func (NopRevisionReadWriter) ReadRevision(context.Context, string) (*Revision, error) {
	return nil, ErrRevisionNotExist
}

// WriteRevision implements RevisionsReadWriter.WriteRevision.
func (NopRevisionReadWriter) WriteRevision(context.Context, *Revision) error {
	return nil
}

// DeleteRevision implements RevisionsReadWriter.DeleteRevision.
func (NopRevisionReadWriter) DeleteRevision(context.Context, string) error {
	return nil
}

var _ RevisionReadWriter = (*NopRevisionReadWriter)(nil)

// done computes and sets the ExecutionTime.
func (r *Revision) done() {
	r.ExecutionTime = time.Now().Sub(r.ExecutedAt)
}

type (
	// A Logger logs migration execution.
	Logger interface {
		Log(LogEntry)
	}

	// LogEntry marks several types of logs to be passed to a Logger.
	LogEntry interface {
		logEntry()
	}

	// LogExecution is sent once when execution of multiple migration files has been started.
	// It holds the filenames of the pending migration files.
	LogExecution struct {
		// From what version.
		From string
		// To what version.
		To string
		// Migration Files to be executed.
		Files []File
	}

	// LogFile is sent if a new migration file is executed.
	LogFile struct {
		// The File being executed.
		File File
		// Version executed.
		// Deprecated: Use File.Version() instead.
		Version string
		// Desc of migration executed.
		// Deprecated: Use File.Desc() instead.
		Desc string
		// Skip holds the number of stmts of this file that will be skipped.
		// This happens, if a migration file was only applied partially and will now continue to be applied.
		Skip int
	}

	// LogStmt is sent if a new SQL statement is executed.
	LogStmt struct {
		SQL string
	}

	// LogDone is sent if the execution is done.
	LogDone struct{}

	// LogError is sent if there is an error while execution.
	LogError struct {
		SQL   string // Set, if Error was caused by a SQL statement.
		Error error
	}

	// LogChecks is sent before the execution of a group of check statements.
	LogChecks struct {
		Name  string   // Optional name.
		Stmts []string // Check statements.
	}

	// LogCheck is sent after a specific check statement was executed.
	LogCheck struct {
		Stmt  string // Check statement.
		Error error  // Check error.
	}

	// LogChecksDone is sent after the execution of a group of checks
	// together with some text message and error if the group failed.
	LogChecksDone struct {
		Error error // Optional error.
	}

	// NopLogger is a Logger that does nothing.
	// It is useful for one-time replay of the migration directory.
	NopLogger struct{}
)

func (LogExecution) logEntry()  {}
func (LogFile) logEntry()       {}
func (LogStmt) logEntry()       {}
func (LogCheck) logEntry()      {}
func (LogChecks) logEntry()     {}
func (LogChecksDone) logEntry() {}
func (LogDone) logEntry()       {}
func (LogError) logEntry()      {}

// Log implements the Logger interface.
func (NopLogger) Log(LogEntry) {}

// LogIntro gathers some meta information from the migration files and stored
// revisions to log some general information prior to actual execution.
func LogIntro(l Logger, revs []*Revision, files []File) {
	e := LogExecution{Files: files}
	if len(revs) > 0 {
		e.From = revs[len(revs)-1].Version
	}
	if len(files) > 0 {
		e.To = files[len(files)-1].Version()
	}
	l.Log(e)
}

// LogNoPendingFiles starts a new LogExecution and LogDone
// to indicate that there are no pending files to be executed.
func LogNoPendingFiles(l Logger, revs []*Revision) {
	LogIntro(l, revs, nil)
	l.Log(LogDone{})
}
