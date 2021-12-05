package integration

import (
	"bytes"
	"context"
	"database/sql"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
	"time"

	"ariga.io/atlas/schema/schemaspec"
	"ariga.io/atlas/schema/schemaspec/schemahcl"
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

func testCLISchemaInspect(t T, h string, dsn string, unmarshalSpec func(data []byte, unmarshaler schemaspec.Unmarshaler, v interface{}) error) {
	// Required to have a clean "stderr" while running first time.
	err := exec.Command("go", "run", "-mod=mod", "ariga.io/atlas/cmd/atlas").Run()
	require.NoError(t, err)
	t.dropTables("users")
	var expected schema.Schema
	err = unmarshalSpec([]byte(h), schemahcl.Unmarshal, &expected)
	require.NoError(t, err)
	t.applyHcl(h)
	cmd := exec.Command("go", "run", "ariga.io/atlas/cmd/atlas",
		"schema",
		"inspect",
		"-d",
		dsn,
	)
	stdout, stderr := bytes.NewBuffer(nil), bytes.NewBuffer(nil)
	cmd.Stderr = stderr
	cmd.Stdout = stdout
	require.NoError(t, cmd.Run(), stderr.String())
	var actual schema.Schema
	err = unmarshalSpec(stdout.Bytes(), schemahcl.Unmarshal, &actual)
	require.NoError(t, err)
	require.Empty(t, stderr.String())
	require.Equal(t, expected, actual)
}

func testCLISchemaApply(t T, h string, dsn string) {
	// Required to have a clean "stderr" while running first time.
	err := exec.Command("go", "run", "-mod=mod", "ariga.io/atlas/cmd/atlas").Run()
	require.NoError(t, err)
	t.dropTables("users")
	f := "atlas.hcl"
	err = ioutil.WriteFile(f, []byte(h), 0644)
	require.NoError(t, err)
	defer os.Remove(f)
	cmd := exec.Command("go", "run", "ariga.io/atlas/cmd/atlas",
		"schema",
		"apply",
		"-d",
		dsn,
		dsn,
		"-f",
		f,
	)
	stdout, stderr := bytes.NewBuffer(nil), bytes.NewBuffer(nil)
	cmd.Stderr = stderr
	cmd.Stdout = stdout
	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)
	defer stdin.Close()
	_, err = io.WriteString(stdin, "\n")
	require.NoError(t, err)
	require.NoError(t, cmd.Run(), stderr.String(), stdout.String())
	require.Empty(t, stderr.String(), stderr.String())
	require.Contains(t, stdout.String(), "-- Planned")
	u := t.loadUsers()
	require.NotNil(t, u)
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
