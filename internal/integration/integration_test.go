package integration

import (
	"testing"

	"ariga.io/atlas/sql/schema"
	"github.com/stretchr/testify/require"
)

// T holds the elements common between dialect tests.
type T interface {
	testing.TB
	realm() *schema.Realm
	loadRealm() *schema.Realm
	users() *schema.Table
	loadUsers() *schema.Table
	posts() *schema.Table
	loadPosts() *schema.Table
	dropTables(...string)
	migrate(...schema.Change)
	diff(*schema.Table, *schema.Table) []schema.Change
}

func testAddDrop(t T) {
	usersT := t.users()
	postsT := t.posts()
	petsT := &schema.Table{
		Name:   "pets",
		Schema: usersT.Schema,
		Columns: []*schema.Column{
			{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}},
			{Name: "owner_id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}, Null: true}},
		},
	}
	petsT.PrimaryKey = &schema.Index{Parts: []*schema.IndexPart{{C: postsT.Columns[0]}}}
	petsT.ForeignKeys = []*schema.ForeignKey{
		{Symbol: "owner_id", Table: petsT, Columns: petsT.Columns[1:], RefTable: usersT, RefColumns: usersT.Columns[:1]},
	}
	t.dropTables(postsT.Name, usersT.Name, petsT.Name)
	t.migrate(&schema.AddTable{T: petsT}, &schema.AddTable{T: usersT}, &schema.AddTable{T: postsT})
	ensureNoChange(t, usersT, petsT, postsT)
	t.migrate(&schema.DropTable{T: usersT}, &schema.DropTable{T: postsT}, &schema.DropTable{T: petsT})
	// Ensure the realm is empty.
	require.EqualValues(t, t.realm(), t.loadRealm())
}

func testRelation(t T) {
	usersT, postsT := t.users(), t.posts()
	t.dropTables(postsT.Name, usersT.Name)
	t.migrate(
		&schema.AddTable{T: usersT},
		&schema.AddTable{T: postsT},
	)
	ensureNoChange(t, postsT, usersT)
}

func ensureNoChange(t T, tables ...*schema.Table) {
	realm := t.loadRealm()
	require.Equal(t, len(realm.Schemas[0].Tables), len(tables))
	for i := range tables {
		tt, ok := realm.Schemas[0].Table(tables[i].Name)
		require.True(t, ok)
		changes := t.diff(tt, tables[i])
		require.Empty(t, changes)
	}
}
