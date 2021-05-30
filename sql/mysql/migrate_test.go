package mysql

import (
	"context"
	"testing"

	"ariga.io/atlas/sql/schema"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestMigrate_Exec(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	mk := mock{m}
	mk.version("8.0.13")
	mk.ExpectExec(escape("DROP TABLE `users`")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mk.ExpectExec(escape("DROP TABLE `public`.`pets`")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mk.ExpectExec(escape("CREATE TABLE `pets` ( `a` int NOT NULL, `b` bigint NOT NULL, `c` bigint NULL, PRIMARY KEY (`a`, `b`) )")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	drv, err := Open(db)
	require.NoError(t, err)
	migrate := Migrate{Driver: drv}
	err = migrate.Exec(context.Background(), []schema.Change{
		&schema.DropTable{T: &schema.Table{Name: "users"}},
		&schema.DropTable{T: &schema.Table{Name: "pets", Schema: &schema.Schema{Name: "public"}}},
		&schema.AddTable{
			T: func() *schema.Table {
				t := &schema.Table{
					Name: "pets",
					Columns: []*schema.Column{
						{Name: "a", Type: &schema.ColumnType{Raw: "int"}},
						{Name: "b", Type: &schema.ColumnType{Raw: "bigint"}},
						{Name: "c", Type: &schema.ColumnType{Raw: "bigint", Null: true}},
					},
				}
				t.PrimaryKey = t.Columns[:2]
				return t
			}(),
		},
	})
	require.NoError(t, err)
}
