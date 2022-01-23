package integration

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"ariga.io/atlas/schema/schemaspec"
	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/schema"

	entsql "entgo.io/ent/dialect/sql"
	"entgo.io/ent/entc/integration/ent"
	"github.com/pkg/diff"
	"github.com/rogpeppe/go-internal/testscript"
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
			expected: "atlas CLI version - development\nhttps://github.com/ariga/atlas/releases/tag/latest\n",
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
			expected: "atlas CLI version v0.3.0-6539f2704b5d-canary\nhttps://github.com/ariga/atlas/releases/tag/latest\n",
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

func TestMySQLScript(t *testing.T) {
	myRun(t, func(t *myTest) {
		var (
			attrs            = t.defaultAttrs()
			charset, collate = attrs[0].(*schema.Charset).V, attrs[1].(*schema.Collation).V
		)
		testscript.Run(t.T, testscript.Params{
			Dir: "testdata/mysql",
			Setup: func(env *testscript.Env) error {
				ctx := context.Background()
				conn, err := t.db.Conn(ctx)
				if err != nil {
					return err
				}
				name := strings.ReplaceAll(filepath.Base(env.WorkDir), "-", "_")
				env.Setenv("db", name)
				if _, err := conn.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", name)); err != nil {
					return err
				}
				if _, err := conn.ExecContext(ctx, fmt.Sprintf("USE `%s`", name)); err != nil {
					return err
				}
				env.Defer(func() {
					if _, err := conn.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", name)); err != nil {
						t.Fatal(err)
					}
					if _, err := conn.ExecContext(ctx, "USE test"); err != nil {
						t.Fatal(err)
					}
					if err := conn.Close(); err != nil {
						t.Fatal(err)
					}
				})
				return nil
			},
			Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
				"apply": func(ts *testscript.TestScript, neg bool, args []string) {
					var (
						desired schema.Schema
						f       = ts.ReadFile(args[0])
						r       = strings.NewReplacer("$charset", charset, "$collate", collate, "$db", ts.Getenv("db"))
					)
					ts.Check(mysql.UnmarshalHCL([]byte(r.Replace(f)), &desired))
					current, err := t.drv.InspectSchema(context.Background(), desired.Name, nil)
					ts.Check(err)
					changes, err := t.drv.SchemaDiff(current, &desired)
					ts.Check(err)
					switch err := t.drv.ApplyChanges(context.Background(), changes); {
					case err != nil && !neg:
						ts.Fatalf("apply changes: %v", err)
					case err == nil && neg:
						ts.Fatalf("unexpected apply success")
					// If we expect to fail, and there's a specific error to compare.
					case err != nil && len(args) == 2:
						re, rerr := regexp.Compile(`(?m)` + args[1])
						ts.Check(rerr)
						if !re.MatchString(err.Error()) {
							t.Fatalf("mismatched errors: %v != %s", err, args[1])
						}
					}
				},
				"exist": func(ts *testscript.TestScript, neg bool, args []string) {
					for _, name := range args {
						var b bool
						if err := t.db.QueryRow("SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?", ts.Getenv("db"), name).Scan(&b); err != nil {
							ts.Fatalf("failed query table existence %q: %v", name, err)
						}
						if !b != neg {
							ts.Fatalf("table %q existence failed", name)
						}
					}
				},
				"cmpshow": func(ts *testscript.TestScript, neg bool, args []string) {
					if len(args) < 2 {
						ts.Fatalf("invalid number of args to 'cmpshow': %d", len(args))
					}
					var (
						fname = args[len(args)-1]
						stmts = make([]string, 0, len(args)-1)
					)
					for _, name := range args[:len(args)-1] {
						var create string
						if err := t.db.QueryRow(fmt.Sprintf("SHOW CREATE TABLE `%s`.`%s`", ts.Getenv("db"), name)).Scan(&name, &create); err != nil {
							ts.Fatalf("show table %q: %v", name, err)
						}
						// Trim the "table_options" if it was not requested explicitly.
						stmts = append(stmts, create[:strings.LastIndexByte(create, ')')+1])
					}

					// Check if there is a file prefixed by database version (1.sql and <version>/1.sql).
					if _, err := os.Stat(ts.MkAbs(filepath.Join(t.version, fname))); err == nil {
						fname = filepath.Join(t.version, fname)
					}
					t1, t2 := strings.Join(stmts, "\n"), ts.ReadFile(fname)
					if strings.TrimSpace(t1) == strings.TrimSpace(t2) {
						return
					}
					var sb strings.Builder
					ts.Check(diff.Text("show", fname, t1, t2, &sb))
					ts.Fatalf(sb.String())
				},
			},
		})
	})
}
