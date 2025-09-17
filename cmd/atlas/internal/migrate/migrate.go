// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migrate

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"ariga.io/atlas/cmd/atlas/internal/migrate/ent"
	"ariga.io/atlas/cmd/atlas/internal/migrate/ent/revision"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"
	"ariga.io/atlas/sql/sqlite"
	"ariga.io/atlas/sql/sqltool"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	entschema "entgo.io/ent/dialect/sql/schema"
	"github.com/google/uuid"
)

type (
	// RevisionReadWriter is a revision read-writer with migration capabilities.
	RevisionReadWriter interface {
		migrate.RevisionReadWriter
		// CurrentRevision returns the current revision in the revisions table.
		CurrentRevision(context.Context) (*migrate.Revision, error)
		// Migrate applies the migration of the revisions table.
		Migrate(context.Context) error
		// ID returns the current target identifier.
		ID(context.Context, string) (string, error)
	}
	// EntRevisions provides implementation for the migrate.RevisionReadWriter interface.
	EntRevisions struct {
		ac     *sqlclient.Client // underlying Atlas client
		ec     *ent.Client       // underlying Ent client
		schema string            // name of the schema the revision table resides in
	}

	// Option allows to configure EntRevisions by using functional arguments.
	Option func(*EntRevisions) error
)

// Dialect returns the "ent dialect" of the Ent client.
func (r *EntRevisions) Dialect() string {
	return EntDialect(r.ac.Name)
}

// EntDialect returns the Ent dialect for the given driver.
func EntDialect(d string) string {
	switch {
	case d == mysql.DriverMaria:
		return dialect.MySQL // Ent does not support "mariadb" as dialect.
	case strings.HasPrefix(d, "libsql"):
		return dialect.SQLite // Ent does not support "libsql" as dialect.
	case d == sqlite.DriverName:
		return dialect.SQLite // Ent does not support "sqlite" as dialect.
	default:
		return d
	}
}

// RevisionsForClient creates a new RevisionReadWriter for the given sqlclient.Client.
func RevisionsForClient(ctx context.Context, ac *sqlclient.Client, schema string) (RevisionReadWriter, error) {
	// If the driver supports the RevisionReadWriter interface, use it.
	if drv, ok := ac.Driver.(interface {
		RevisionsReadWriter(context.Context, string) (migrate.RevisionReadWriter, error)
	}); ok {
		rrw, err := drv.RevisionsReadWriter(ctx, schema)
		if err != nil {
			return nil, err
		}
		if rrw, ok := rrw.(RevisionReadWriter); ok {
			return rrw, nil
		}
		return nil, fmt.Errorf("unexpected revision read-writer type: %T", rrw)
	}
	return NewEntRevisions(ctx, ac, WithSchema(schema))
}

// NewEntRevisions creates a new EntRevisions with the given sqlclient.Client.
func NewEntRevisions(ctx context.Context, ac *sqlclient.Client, opts ...Option) (*EntRevisions, error) {
	r := &EntRevisions{ac: ac}
	for _, opt := range opts {
		if err := opt(r); err != nil {
			return nil, err
		}
	}
	if r.Dialect() == dialect.SQLite && r.schema != "" && r.schema != "main" {
		return nil, fmt.Errorf("cannot store revisions-table in a separate schema (%q) with SQLite driver", r.schema)
	}
	// Create the connection with the underlying migrate.Driver to have it inside a possible transaction.
	entopts := []ent.Option{ent.Driver(sql.NewDriver(r.Dialect(), sql.Conn{ExecQuerier: r.ac.Driver}))}
	// SQLite does not support multiple schema, therefore schema-config is only needed for other dialects.
	if r.Dialect() != dialect.SQLite {
		// Make sure the schema to store the revisions table in does exist.
		_, err := r.ac.InspectSchema(ctx, r.schema, &schema.InspectOptions{Mode: schema.InspectSchemas})
		if err != nil && !schema.IsNotExistError(err) {
			return nil, err
		}
		if schema.IsNotExistError(err) {
			if err := r.ac.ApplyChanges(ctx, []schema.Change{
				&schema.AddSchema{S: &schema.Schema{Name: r.schema}},
			}); err != nil {
				return nil, err
			}
		}
		// Tell Ent to operate on a given schema.
		if r.schema != "" {
			entopts = append(entopts, ent.AlternateSchema(ent.SchemaConfig{Revision: r.schema}))
		}
	}
	// Instantiate the Ent client and migrate the revision schema.
	r.ec = ent.NewClient(entopts...)
	return r, nil
}

// WithSchema configures the schema to use for the revision table.
func WithSchema(s string) Option {
	return func(r *EntRevisions) error {
		r.schema = s
		return nil
	}
}

// Ident returns the table identifier.
func (r *EntRevisions) Ident() *migrate.TableIdent {
	return &migrate.TableIdent{Name: revision.Table, Schema: r.schema}
}

// ReadRevision reads a revision from the revisions table.
//
// ReadRevision will not return results only saved in cache.
func (r *EntRevisions) ReadRevision(ctx context.Context, v string) (*migrate.Revision, error) {
	if v == revisionID {
		return nil, errors.New("cannot read revision-table identifier as revision")
	}
	rev, err := r.ec.Revision.Get(ctx, v)
	if err != nil && !ent.IsNotFound(err) {
		return nil, err
	}
	if ent.IsNotFound(err) {
		return nil, migrate.ErrRevisionNotExist
	}
	return rev.AtlasRevision(), nil
}

// ReadRevisions reads the revisions from the revisions table.
//
// ReadRevisions will not return results only saved to cache.
func (r *EntRevisions) ReadRevisions(ctx context.Context) ([]*migrate.Revision, error) {
	revs, err := r.ec.Revision.Query().
		Where(revision.IDNEQ(revisionID)).
		Order(revision.ByID()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	ret := make([]*migrate.Revision, len(revs))
	for i, rev := range revs {
		ret[i] = rev.AtlasRevision()
	}
	return ret, nil
}

// CurrentRevision returns the current (latest) revision in the revisions table.
func (r *EntRevisions) CurrentRevision(ctx context.Context) (*migrate.Revision, error) {
	rev, err := r.ec.Revision.Query().
		Where(revision.IDNEQ(revisionID)).
		Order(revision.ByID(sql.OrderDesc())).
		First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return nil, err
	}
	if ent.IsNotFound(err) {
		return nil, migrate.ErrRevisionNotExist
	}
	return rev.AtlasRevision(), nil
}

// WriteRevision writes a revision to the revisions table.
func (r *EntRevisions) WriteRevision(ctx context.Context, rev *migrate.Revision) error {
	if rev.Version == revisionID {
		return errors.New("writing the revision-table identifier is not allowed")
	}
	return r.ec.Revision.Create().
		SetRevision(rev).
		OnConflict(sql.ConflictColumns(revision.FieldID)).
		UpdateNewValues().
		Exec(ctx)
}

// DeleteRevision deletes a revision from the revisions table.
func (r *EntRevisions) DeleteRevision(ctx context.Context, v string) error {
	if v == revisionID {
		return errors.New("deleting the revision-table identifier is not allowed")
	}
	return r.ec.Revision.DeleteOneID(v).Exec(ctx)
}

// Migrate attempts to create / update the revisions table. This is separated since Ent attempts to wrap the migration
// execution in a transaction and assumes the underlying connection is of type *sql.DB, which is not true for actually
// reading and writing revisions.
func (r *EntRevisions) Migrate(ctx context.Context) (err error) {
	var (
		opts = []entschema.MigrateOption{
			entschema.WithDropColumn(true),
		}
		c = ent.NewClient(ent.Driver(sql.OpenDB(r.Dialect(), r.ac.DB)))
	)
	switch {
	case r.Dialect() != dialect.SQLite:
		// Ensure the ent client is bound to the requested revision schema. Open a new connection, if not.
		if r.ac.URL.Schema != r.schema {
			sc, err := sqlclient.OpenURL(ctx, r.ac.URL.URL, sqlclient.OpenSchema(r.schema))
			if err != nil {
				return err
			}
			defer sc.Close()
			c = ent.NewClient(ent.Driver(sql.OpenDB(r.Dialect(), sc.DB)))
		}
		// In non-SQLite databases, there can be multiple schemas, and we
		// prefer to pass it explicitly rather than calling to CURRENT_SCHEMA().
		opts = append(opts, entschema.WithSchemaName(r.schema))
	default: // SQLite.
		var on sql.NullBool
		if err := r.ac.DB.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&on); err != nil {
			return err
		}
		if !on.Bool {
			// Ent requires the foreign key checks in SQLite to be enabled for migration. Since Atlas does not,
			// ensure they are set for the migration attempt and restore previous setting afterwards.
			_, err := r.ac.ExecContext(ctx, "PRAGMA foreign_keys = on")
			if err != nil {
				return err
			}
			defer func() {
				_, err2 := r.ac.ExecContext(ctx, "PRAGMA foreign_keys = off")
				if err2 != nil {
					if err != nil {
						err = fmt.Errorf("%v: %w", err2, err)
						return
					}
					err = err2
				}
			}()
		}
	}
	return c.Schema.Create(ctx, append(opts, entschema.WithDiffHook(func(next entschema.Differ) entschema.Differ {
		return entschema.DiffFunc(func(current, desired *schema.Schema) ([]schema.Change, error) {
			changes, err := next.Diff(current, desired)
			if err != nil {
				return nil, err
			}
			// Skip all changes beside revisions
			// table creation or modification.
			for i := range changes {
				switch cs := changes[i].(type) {
				case *schema.AddTable:
					r.maySetSchemaQualifier(cs.T)
					if cs.T.Name == revision.Table {
						return schema.Changes{cs}, nil
					}
				case *schema.ModifyTable:
					r.maySetSchemaQualifier(cs.T)
					if cs.T.Name == revision.Table {
						return schema.Changes{cs}, nil
					}
				}
			}
			return nil, nil
		})
	}))...)
}

// maySetSchemaQualifier sets the schema on the atlas_schema_revisions table
// for cases the migration should use qualified identifiers due to some limitation
// in PostgreSQL services that use "connection pooler" and do not support the "search_path"
// parameter. See: https://github.com/ariga/atlas/issues/2509.
func (r *EntRevisions) maySetSchemaQualifier(t *schema.Table) {
	if r.Dialect() != dialect.Postgres || r.schema == "" || t.Schema != nil {
		return // Not PG, or schema-scope (e.g., public).
	}
	if knownServices := []string{"neon.tech", "supabase.co", "supabase.com"}; slices.ContainsFunc(knownServices, func(s string) bool {
		return strings.HasSuffix(r.ac.URL.Host, s)
	}) {
		t.SetSchema(schema.New(r.schema))
	}
}

// revisionID holds the column "id" ("version") of the revision that holds the identifier of the
// connected revisions table. The "." prefix ensures the is it lower than any other revisions.
const revisionID = ".atlas_cloud_identifier"

// ID returns the identifier of the connected revisions table.
func (r *EntRevisions) ID(ctx context.Context, operatorV string) (string, error) {
	err := r.ec.Revision.Create().
		SetID(revisionID).                // identifier key
		SetDescription(uuid.NewString()). // actual revision identifier
		SetOperatorVersion(operatorV).    // operator version
		SetExecutedAt(time.Now()).        // when it was set
		SetExecutionTime(0).              // dummy values
		SetHash("").
		OnConflict(sql.ConflictColumns(revision.FieldID)).
		Ignore().
		Exec(ctx)
	if err != nil {
		return "", fmt.Errorf("upsert revision-table id: %w", err)
	}
	rev, err := r.ec.Revision.Get(ctx, revisionID)
	if err != nil {
		return "", fmt.Errorf("read revision-table id: %w", err)
	}
	id, err := uuid.Parse(rev.Description)
	if err != nil {
		return "", fmt.Errorf("parse revision-table id: %w", err)
	}
	return id.String(), nil
}

var _ migrate.RevisionReadWriter = (*EntRevisions)(nil)

// List of supported formats.
const (
	FormatAtlas         = "atlas"
	FormatGolangMigrate = "golang-migrate"
	FormatGoose         = "goose"
	FormatFlyway        = "flyway"
	FormatLiquibase     = "liquibase"
	FormatDBMate        = "dbmate"
)

// Formats is the list of supported formats.
var Formats = []string{FormatAtlas, FormatGolangMigrate, FormatGoose, FormatFlyway, FormatLiquibase, FormatDBMate}

// Formatter returns the dir formatter for its URL.
func Formatter(u *url.URL) (migrate.Formatter, error) {
	switch f := u.Query().Get("format"); f {
	case "", FormatAtlas:
		return migrate.DefaultFormatter, nil
	case FormatGolangMigrate:
		return sqltool.GolangMigrateFormatter, nil
	case FormatGoose:
		return sqltool.GooseFormatter, nil
	case FormatFlyway:
		return sqltool.FlywayFormatter, nil
	case FormatLiquibase:
		return sqltool.LiquibaseFormatter, nil
	case FormatDBMate:
		return sqltool.DBMateFormatter, nil
	default:
		return nil, fmt.Errorf("unknown format %q", f)
	}
}

// Dir parses u and calls dirURL.
func Dir(ctx context.Context, u string, create bool) (migrate.Dir, error) {
	parsed, err := url.Parse(u)
	if err != nil {
		return nil, err
	}
	return DirURL(ctx, parsed, create)
}

// Directory types (URL schemes).
const (
	DirTypeMem   = "mem"
	DirTypeFile  = "file"
	DirTypeAtlas = "atlas"
)

// DefaultDirName is the default directory name.
const DefaultDirName = "migrations"

// DirURL returns a migrate.Dir to use as migration directory.
func DirURL(ctx context.Context, u *url.URL, create bool) (migrate.Dir, error) {
	p := filepath.Join(u.Host, u.Path)
	switch u.Scheme {
	case DirTypeMem:
		return migrate.OpenMemDir(path.Join(u.Host, u.Path)), nil
	case DirTypeFile:
		if p == "" {
			p = DefaultDirName
		}
	case DirTypeAtlas:
		return openAtlasDir(ctx, u)
	case "":
		return nil, fmt.Errorf("missing scheme for dir url. Did you mean %q? ", fmt.Sprintf("%s://%s", DirTypeFile, u.Path))
	default:
		return nil, fmt.Errorf("unsupported driver %q", u.Scheme)
	}
	fn := func() (migrate.Dir, error) { return migrate.NewLocalDir(p) }
	switch f := u.Query().Get("format"); f {
	case "", FormatAtlas:
		// this is the default
	case FormatGolangMigrate:
		fn = func() (migrate.Dir, error) { return sqltool.NewGolangMigrateDir(p) }
	case FormatGoose:
		fn = func() (migrate.Dir, error) { return sqltool.NewGooseDir(p) }
	case FormatFlyway:
		fn = func() (migrate.Dir, error) { return sqltool.NewFlywayDir(p) }
	case FormatLiquibase:
		fn = func() (migrate.Dir, error) { return sqltool.NewLiquibaseDir(p) }
	case FormatDBMate:
		fn = func() (migrate.Dir, error) { return sqltool.NewDBMateDir(p) }
	default:
		return nil, fmt.Errorf("unknown dir format %q", f)
	}
	d, err := fn()
	if create && errors.Is(err, fs.ErrNotExist) {
		if err := os.MkdirAll(p, 0755); err != nil {
			return nil, err
		}
		d, err = fn()
		if err != nil {
			return nil, err
		}
	}
	return d, err
}

// ChangesToRealm returns the schema changes for creating the given Realm.
func ChangesToRealm(c *sqlclient.Client, r *schema.Realm) schema.Changes {
	var changes schema.Changes
	for _, o := range r.Objects {
		changes = append(changes, &schema.AddObject{O: o})
	}
	for _, s := range r.Schemas {
		// Generate commands for creating the schemas on realm-mode.
		if c.URL.Schema == "" {
			changes = append(changes, &schema.AddSchema{S: s})
		}
		for _, o := range s.Objects {
			changes = append(changes, &schema.AddObject{O: o})
		}
		for _, t := range s.Tables {
			changes = append(changes, &schema.AddTable{T: t})
			for _, r := range t.Triggers {
				changes = append(changes, &schema.AddTrigger{T: r})
			}
		}
		for _, v := range s.Views {
			changes = append(changes, &schema.AddView{V: v})
			for _, r := range v.Triggers {
				changes = append(changes, &schema.AddTrigger{T: r})
			}
		}
		for _, f := range s.Funcs {
			changes = append(changes, &schema.AddFunc{F: f})
		}
		for _, p := range s.Procs {
			changes = append(changes, &schema.AddProc{P: p})
		}
	}
	return changes
}
