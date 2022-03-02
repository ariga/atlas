package integration

import (
	"bytes"
	"context"
	"database/sql"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"ariga.io/atlas/schema/schemaspec"
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
	loadTable(string) *schema.Table
	dropTables(...string)
	migrate(...schema.Change)
	diff(*schema.Table, *schema.Table) []schema.Change
	applyHcl(spec string)
	applyRealmHcl(spec string)
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

func testImplicitIndexes(t T, db *sql.DB) {
	const (
		name = "implicit_indexes"
		ddl  = "create table implicit_indexes(c1 int unique, c2 int unique, unique(c1,c2), unique(c2,c1))"
	)
	t.dropTables(name)
	_, err := db.Exec(ddl)
	require.NoError(t, err)
	current := t.loadTable(name)
	c1, c2 := schema.NewNullIntColumn("c1", "int"), schema.NewNullIntColumn("c2", "int")
	desired := schema.NewTable(name).
		AddColumns(c1, c2).
		AddIndexes(
			schema.NewUniqueIndex("").AddColumns(c1),
			schema.NewUniqueIndex("").AddColumns(c2),
			schema.NewUniqueIndex("").AddColumns(c1, c2),
			schema.NewUniqueIndex("").AddColumns(c2, c1),
		)
	changes := t.diff(current, desired)
	require.Empty(t, changes)
	desired.AddIndexes(
		schema.NewIndex("c1_key").AddColumns(c1),
		schema.NewIndex("c2_key").AddColumns(c2),
	)
	changes = t.diff(current, desired)
	require.NotEmpty(t, changes)
	t.migrate(&schema.ModifyTable{T: desired, Changes: changes})
	ensureNoChange(t, desired)
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

func testCLISchemaInspect(t T, h string, dsn string, unmarshaler schemaspec.Unmarshaler) {
	// Required to have a clean "stderr" while running first time.
	err := exec.Command("go", "run", "-mod=mod", "ariga.io/atlas/cmd/atlas").Run()
	require.NoError(t, err)
	t.dropTables("users")
	var expected schema.Schema
	err = unmarshaler.UnmarshalSpec([]byte(h), &expected)
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
	err = unmarshaler.UnmarshalSpec(stdout.Bytes(), &actual)
	require.NoError(t, err)
	require.Empty(t, stderr.String())
	require.Equal(t, expected, actual)
}

func testCLIMultiSchemaApply(t T, h string, dsn string, schemas []string, unmarshaler schemaspec.Unmarshaler) {
	// Required to have a clean "stderr" while running first time.
	err := exec.Command("go", "run", "-mod=mod", "ariga.io/atlas/cmd/atlas").Run()
	f := "atlas.hcl"
	err = ioutil.WriteFile(f, []byte(h), 0644)
	require.NoError(t, err)
	t.Cleanup(func() {
		os.Remove("atlas.hcl")
	})
	require.NoError(t, err)
	var expected schema.Realm
	err = unmarshaler.UnmarshalSpec([]byte(h), &expected)
	require.NoError(t, err)
	cmd := exec.Command("go", "run", "ariga.io/atlas/cmd/atlas",
		"schema",
		"apply",
		"-f",
		"atlas.hcl",
		"-d",
		dsn,
		"-s",
		strings.Join(schemas, ","),
	)
	stdout, stderr := bytes.NewBuffer(nil), bytes.NewBuffer(nil)
	cmd.Stderr = stderr
	cmd.Stdout = stdout
	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)
	defer stdin.Close()
	_, err = io.WriteString(stdin, "\n")
	require.NoError(t, cmd.Run(), stderr.String())
	require.Contains(t, stdout.String(), `-- Add new schema named "test2"`)
}

func testCLIMultiSchemaInspect(t T, h string, dsn string, schemas []string, unmarshaler schemaspec.Unmarshaler) {
	// Required to have a clean "stderr" while running first time.
	err := exec.Command("go", "run", "-mod=mod", "ariga.io/atlas/cmd/atlas").Run()
	require.NoError(t, err)
	var expected schema.Realm
	err = unmarshaler.UnmarshalSpec([]byte(h), &expected)
	require.NoError(t, err)
	t.applyRealmHcl(h)
	cmd := exec.Command("go", "run", "ariga.io/atlas/cmd/atlas",
		"schema",
		"inspect",
		"-d",
		dsn,
		"-s",
		strings.Join(schemas, ","),
	)
	stdout, stderr := bytes.NewBuffer(nil), bytes.NewBuffer(nil)
	cmd.Stderr = stderr
	cmd.Stdout = stdout
	require.NoError(t, cmd.Run(), stderr.String())
	var actual schema.Realm
	err = unmarshaler.UnmarshalSpec(stdout.Bytes(), &actual)
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
		"-u",
		dsn,
		"-f",
		f,
		"--dev-url",
		dsn,
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

func testCLISchemaApplyDry(t T, h string, dsn string) {
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
		"-f",
		f,
		"--dry-run",
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
	require.NotContains(t, stdout.String(), "Are you sure?", "dry run should not prompt")
	realm := t.loadRealm()
	_, ok := realm.Schemas[0].Table("users")
	require.False(t, ok, "expected users table not to be created")
}

func testCLISchemaApplyAutoApprove(t T, h string, dsn string) {
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
		"-f",
		f,
		"--auto-approve",
	)
	stdout, stderr := bytes.NewBuffer(nil), bytes.NewBuffer(nil)
	cmd.Stderr = stderr
	cmd.Stdout = stdout
	require.NoError(t, err)
	require.NoError(t, cmd.Run(), stderr.String(), stdout.String())
	require.Empty(t, stderr.String(), stderr.String())
	require.Contains(t, stdout.String(), "-- Planned")
	u := t.loadUsers()
	require.NotNil(t, u)
}

func testCLISchemaDiff(t T, dsn string) {
	// Required to have a clean "stderr" while running first time.
	err := exec.Command("go", "run", "-mod=mod", "ariga.io/atlas/cmd/atlas").Run()

	require.NoError(t, err)
	t.dropTables("users")
	cmd := exec.Command("go", "run", "ariga.io/atlas/cmd/atlas",
		"schema",
		"diff",
		"--from",
		dsn,
		"--to",
		dsn,
	)
	stdout, stderr := bytes.NewBuffer(nil), bytes.NewBuffer(nil)
	cmd.Stderr = stderr
	cmd.Stdout = stdout
	require.NoError(t, cmd.Run(), stderr.String(), stdout.String())
	require.Empty(t, stderr.String(), stderr.String())
	require.Contains(t, stdout.String(), "Schemas are synced, no changes to be made.")
}

func TestCLI_Version(t *testing.T) {
	// Required to have a clean "stderr" while running first time.
	require.NoError(t, exec.Command("go", "run", "-mod=mod", "ariga.io/atlas/cmd/atlas").Run())
	tests := []struct {
		name     string
		cmd      *exec.Cmd
		expected string
	}{
		{
			name: "dev mode",
			cmd: exec.Command("go", "run", "ariga.io/atlas/cmd/atlas",
				"version",
			),
			expected: "atlas CLI version - development\nhttps://github.com/ariga/atlas/releases/latest\n",
		},
		{
			name: "release",
			cmd: exec.Command("go", "run",
				"-ldflags",
				"-X ariga.io/atlas/cmd/action.version=v1.2.3",
				"ariga.io/atlas/cmd/atlas",
				"version",
			),
			expected: "atlas CLI version v1.2.3\nhttps://github.com/ariga/atlas/releases/tag/v1.2.3\n",
		},
		{
			name: "canary",
			cmd: exec.Command("go", "run",
				"-ldflags",
				"-X ariga.io/atlas/cmd/action.version=v0.3.0-6539f2704b5d-canary",
				"ariga.io/atlas/cmd/atlas",
				"version",
			),
			expected: "atlas CLI version v0.3.0-6539f2704b5d-canary\nhttps://github.com/ariga/atlas/releases/latest\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("ATLAS_NO_UPDATE_NOTIFIER", "true")
			stdout := bytes.NewBuffer(nil)
			tt.cmd.Stdout = stdout
			require.NoError(t, tt.cmd.Run())
			require.Equal(t, tt.expected, stdout.String())
		})
	}
}

func ensureNoChange(t T, tables ...*schema.Table) {
	realm := t.loadRealm()
	require.Equal(t, len(realm.Schemas[0].Tables), len(tables))
	for i := range tables {
		tt, ok := realm.Schemas[0].Table(tables[i].Name)
		require.True(t, ok)
		changes := t.diff(tt, tables[i])
		require.Emptyf(t, changes, "changes should be empty for table %s, but instead was %#v", tt.Name, changes)
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
