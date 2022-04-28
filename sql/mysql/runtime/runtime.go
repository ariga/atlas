package runtime

import (
	"context"
	"database/sql"

	"ariga.io/atlas/sql/internal/migrate/migrate"
	"ariga.io/atlas/sql/mysql"
	entsql "entgo.io/ent/dialect/sql"
	entschema "entgo.io/ent/dialect/sql/schema"
)

// InitEntSchemaMigrator stitches in the Ent migration engine to the mysql.Driver at runtime. This is necessary
// because the Ent migration engine imports atlas and therefore would introduce a cyclic dependency.
func InitEntSchemaMigrator(drv *mysql.Driver, db *sql.DB, dialect string) {
	mgr := migrate.NewSchema(entsql.NewDriver(entsql.Conn{ExecQuerier: db}, dialect))
	drv.InitSchemaMigrator(func(ctx context.Context) error {
		return mgr.Create(ctx, entschema.WithAtlas(true))
	})
}
