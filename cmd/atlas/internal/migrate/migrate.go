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
	"time"

	"ariga.io/atlas/cmd/atlas/internal/migrate/ent"
	"ariga.io/atlas/cmd/atlas/internal/migrate/ent/revision"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"
	"ariga.io/atlas/sql/sqltool"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	entschema "entgo.io/ent/dialect/sql/schema"
	"github.com/google/uuid"
)

type (
	// EntRevisions provides implementation for the migrate.RevisionReadWriter interface.
	EntRevisions struct {
		ac     *sqlclient.Client // underlying Atlas client
		ec     *ent.Client       // underlying Ent client
		schema string            // name of the schema the revision table resides in
	}

	// Option allows to configure EntRevisions by using functional arguments.
	Option func(*EntRevisions) error
)

// NewEntRevisions creates a new EntRevisions with the given sqlclient.Client.
func NewEntRevisions(ctx context.Context, ac *sqlclient.Client, opts ...Option) (*EntRevisions, error) {
	r := &EntRevisions{ac: ac}
	for _, opt := range opts {
		if err := opt(r); err != nil {
			return nil, err
		}
	}
	// Create the connection with the underlying migrate.Driver to have it inside a possible transaction.
	entopts := []ent.Option{ent.Driver(sql.NewDriver(r.ac.Name, sql.Conn{ExecQuerier: r.ac.Driver}))}
	// SQLite does not support multiple schema, therefore schema-config is only needed for other dialects.
	if r.ac.Name != dialect.SQLite {
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
	c := ent.NewClient(ent.Driver(sql.OpenDB(r.ac.Name, r.ac.DB)))
	// Ensure the ent client is bound to the requested revision schema. Open a new connection, if not.
	if r.ac.Name != dialect.SQLite && r.ac.URL.Schema != r.schema {
		sc, err := sqlclient.OpenURL(ctx, r.ac.URL.URL, sqlclient.OpenSchema(r.schema))
		if err != nil {
			return err
		}
		defer sc.Close()
		c = ent.NewClient(ent.Driver(sql.OpenDB(sc.Name, sc.DB)))
	}
	if r.ac.Name == dialect.SQLite {
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
	return c.Schema.Create(ctx, entschema.WithDropColumn(true), entschema.WithDiffHook(func(next entschema.Differ) entschema.Differ {
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
					if cs.T.Name == revision.Table {
						return schema.Changes{cs}, nil
					}
				case *schema.ModifyTable:
					if cs.T.Name == revision.Table {
						return schema.Changes{cs}, nil
					}
				}
			}
			return nil, nil
		})
	}))
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
	case FormatAtlas:
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
func Dir(u string, create bool) (migrate.Dir, error) {
	parsed, err := url.Parse(u)
	if err != nil {
		return nil, err
	}
	return DirURL(parsed, create)
}

// DirURL returns a migrate.Dir to use as migration directory. For now only local directories are supported.
func DirURL(u *url.URL, create bool) (migrate.Dir, error) {
	p := filepath.Join(u.Host, u.Path)
	switch u.Scheme {
	case "mem":
		return migrate.OpenMemDir(path.Join(u.Host, u.Path)), nil
	case "file":
		if p == "" {
			p = "migrations"
		}
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
