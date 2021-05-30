package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"

	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/schema"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

func TestMySQL(t *testing.T) {
	for version, port := range map[string]int{"57": 3307, "8": 3308} {
		t.Run(version, func(t *testing.T) {
			db, err := sql.Open("mysql", fmt.Sprintf("root:pass@tcp(localhost:%d)/test?parseTime=True", port))
			require.NoError(t, err)
			defer db.Close()
			drv, err := mysql.Open(db)
			require.NoError(t, err)

			ctx := context.Background()
			t.Log("inspecting empty realm")
			realm, err := drv.InspectRealm(ctx, &schema.InspectRealmOption{
				Schemas: []string{"test"},
			})
			require.NoError(t, err)
			require.EqualValues(t, func() *schema.Realm {
				r := &schema.Realm{
					Schemas: []*schema.Schema{
						{
							Name:  "test",
							Attrs: defaultAttrs(version),
						},
					},
					Attrs: defaultAttrs(version),
				}
				r.Schemas[0].Realm = r
				return r
			}(), realm)
			t.Log("adding table")
			postsT := &schema.Table{
				Name: "posts",
				Columns: []*schema.Column{
					{Name: "id", Type: &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}}},
					{Name: "author_id", Type: &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}, Null: true}},
				},
			}
			postsT.PrimaryKey = postsT.Columns[:1]
			migrate := mysql.Migrate{Driver: drv}
			err = migrate.Exec(ctx, []schema.Change{
				&schema.AddTable{T: postsT},
			})
			require.NoError(t, err)
			defer migrate.Exec(ctx, []schema.Change{
				&schema.DropTable{T: postsT},
			})

			t.Log("comparing tables")
			realm, err = drv.InspectRealm(ctx, &schema.InspectRealmOption{
				Schemas: []string{"test"},
			})
			require.NoError(t, err)
			diff := mysql.Diff{Driver: drv}
			changes, err := diff.TableDiff(realm.Schemas[0].Tables[0], postsT)
			require.NoError(t, err)
			require.Empty(t, changes)
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

// printChanges for debug purpose. Do not remove.
func printChanges(c []schema.Change) {
	fmt.Printf("%T{\n", c)
	for i := range c {
		b, err := json.MarshalIndent(c[i], "\t", "\t")
		if err != nil {
			panic(err)
		}
		fmt.Printf("\t%T%s\n", c[i], string(b))
	}
	fmt.Println("}")
}
