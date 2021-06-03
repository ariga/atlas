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
			usersT := &schema.Table{
				Name:   "users",
				Schema: realm.Schemas[0],
				Columns: []*schema.Column{
					{
						Name:  "id",
						Type:  &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}},
						Attrs: []schema.Attr{&mysql.AutoIncrement{}},
					},
				},
				Attrs: defaultAttrs(version),
			}
			usersT.PrimaryKey = &schema.Index{Parts: []*schema.IndexPart{{C: usersT.Columns[0]}}}
			postsT := &schema.Table{
				Name:   "posts",
				Schema: realm.Schemas[0],
				Columns: []*schema.Column{
					{
						Name:  "id",
						Type:  &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint", Size: 8}},
						Attrs: []schema.Attr{&mysql.AutoIncrement{}},
					},
					{
						Name:    "author_id",
						Type:    &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint", Size: 8}, Null: true},
						Default: &schema.RawExpr{X: "10"},
					},
					{
						Name: "ctime",
						Type: &schema.ColumnType{Raw: "timestamp", Type: &schema.TimeType{T: "timestamp"}},
						Default: &schema.RawExpr{
							X: "CURRENT_TIMESTAMP",
						},
						Attrs: []schema.Attr{
							&mysql.OnUpdate{
								A: "CURRENT_TIMESTAMP",
							},
						},
					},
				},
				Attrs: defaultAttrs(version),
			}
			postsT.PrimaryKey = &schema.Index{Parts: []*schema.IndexPart{{C: postsT.Columns[0]}}}
			postsT.Indexes = []*schema.Index{
				{Name: "author_id", Parts: []*schema.IndexPart{{C: postsT.Columns[1]}}},
				{Name: "id_author_id_unique", Unique: true, Parts: []*schema.IndexPart{{C: postsT.Columns[1]}, {C: postsT.Columns[0]}}},
			}
			postsT.ForeignKeys = []*schema.ForeignKey{
				{Symbol: "author_id", Table: postsT, Columns: postsT.Columns[1:2], RefTable: usersT, RefColumns: usersT.Columns, OnDelete: schema.NoAction},
			}
			migrate := mysql.Migrate{Driver: drv}
			err = migrate.Exec(ctx, []schema.Change{
				&schema.AddTable{T: usersT},
				&schema.AddTable{T: postsT},
			})
			require.NoError(t, err)
			defer migrate.Exec(ctx, []schema.Change{
				&schema.DropTable{T: postsT},
				&schema.DropTable{T: usersT},
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
			changes, err = diff.TableDiff(realm.Schemas[0].Tables[1], usersT)
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
