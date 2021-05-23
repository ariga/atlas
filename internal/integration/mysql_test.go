package integration

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/schema"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

func TestMySQLInspect(t *testing.T) {
	for version, port := range map[string]int{"57": 3307, "8": 3308} {
		t.Run(version, func(t *testing.T) {
			db, err := sql.Open("mysql", fmt.Sprintf("root:pass@tcp(localhost:%d)/test?parseTime=True", port))
			require.NoError(t, err)
			defer db.Close()
			drv, err := mysql.Open(db)
			require.NoError(t, err)

			ctx := context.Background()
			t.Log("inspecting empty realm")
			realm, err := drv.Realm(ctx, &schema.InspectRealmOption{
				Schemas: []string{"test"},
			})
			require.NoError(t, err)
			require.EqualValues(t, &schema.Realm{
				Schemas: []*schema.Schema{
					{
						Name:   "test",
						Attrs:  defaultAttrs(version),
						Tables: []*schema.Table{},
					},
				},
				Attrs: defaultAttrs(version),
			}, realm)
			t.Log("adding table")
			postsT := &schema.Table{
				Name: "posts",
				Columns: []*schema.Column{
					{Name: "id", Type: &schema.ColumnType{Raw: "bigint"}},
					{Name: "author_id", Type: &schema.ColumnType{Raw: "bigint", Null: true}},
				},
			}
			postsT.PrimaryKey = postsT.Columns[:1]
			err = drv.Exec(ctx, []schema.Change{
				&schema.AddTable{T: postsT},
			})
			require.NoError(t, err)
			defer drv.Exec(ctx, []schema.Change{
				&schema.DropTable{T: postsT},
			})
		})
	}
}

// defaultConfig returns the default charset and
// collation configuration based on the MySQL version.
func defaultAttrs(version string) []schema.Attr {
	var (
		charset   = "latin1"
		collation = "latin1_swedish_ci"
	)
	if version == "8" {
		charset = "utf8mb4"
		collation = "utf8mb4_0900_ai_ci"
	}
	return []schema.Attr{
		&schema.Charset{
			V: charset,
		},
		&schema.Collation{
			V: collation,
		},
	}
}
