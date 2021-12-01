package integration

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"ariga.io/atlas/sql/schema"

	entsql "entgo.io/ent/dialect/sql"
	"entgo.io/ent/entc/integration/ent"
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
	applyHcl(spec string)
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

func testEntIntegration(t T, dialect string, db *sql.DB) {
	ctx := context.Background()
	drv := entsql.OpenDB(dialect, db)
	client := ent.NewClient(ent.Driver(drv))
	require.NoError(t, client.Schema.Create(ctx))
	sanity(client)
	realm := t.loadRealm()
	ensureNoChange(t, realm.Schemas[0].Tables...)

	// Drop tables.
	changes := make([]schema.Change, len(realm.Schemas[0].Tables))
	for i, t := range realm.Schemas[0].Tables {
		changes[i] = &schema.DropTable{T: t}
	}
	t.migrate(changes...)

	// Add tables.
	for i, t := range realm.Schemas[0].Tables {
		changes[i] = &schema.AddTable{T: t}
	}
	t.migrate(changes...)
	ensureNoChange(t, realm.Schemas[0].Tables...)
	sanity(client)

	// Drop tables.
	for i, t := range realm.Schemas[0].Tables {
		changes[i] = &schema.DropTable{T: t}
	}
	t.migrate(changes...)
}

func testHCLIntegration(t T, full string, empty string) {
	t.applyHcl(full)
	users := t.loadUsers()
	posts := t.loadPosts()
	t.dropTables(users.Name, posts.Name)
	column, ok := users.Column("id")
	require.True(t, ok, "expected id column")
	require.Equal(t, "users", users.Name)
	column, ok = posts.Column("author_id")
	require.Equal(t, "author_id", column.Name)
	t.applyHcl(empty)
	require.Empty(t, t.realm().Schemas[0].Tables)
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

func sanity(c *ent.Client) {
	ctx := context.Background()
	u := c.User.Create().
		SetName("foo").
		SetAge(20).
		AddPets(
			c.Pet.Create().SetName("pedro").SaveX(ctx),
			c.Pet.Create().SetName("xabi").SaveX(ctx),
		).
		AddFiles(
			c.File.Create().SetName("a").SetSize(10).SaveX(ctx),
			c.File.Create().SetName("b").SetSize(20).SaveX(ctx),
		).
		SaveX(ctx)
	c.Group.Create().
		SetName("Github").
		SetExpire(time.Now()).
		AddUsers(u).
		SetInfo(c.GroupInfo.Create().SetDesc("desc").SaveX(ctx)).
		SaveX(ctx)
}
