// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"unicode"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlite"

	"github.com/pkg/diff"
	"github.com/rogpeppe/go-internal/testscript"
	"github.com/stretchr/testify/require"
)

func TestMySQL_Script(t *testing.T) {
	myRun(t, func(t *myTest) {
		testscript.Run(t.T, testscript.Params{
			Dir:   "testdata/mysql",
			Setup: t.setupScript,
			Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
				"only":        cmdOnly,
				"apply":       t.cmdApply,
				"exist":       t.cmdExist,
				"synced":      t.cmdSynced,
				"cmphcl":      t.cmdCmpHCL,
				"cmpshow":     t.cmdCmpShow,
				"cmpmig":      t.cmdCmpMig,
				"execsql":     t.cmdExec,
				"atlas":       t.cmdCLI,
				"clearSchema": t.clearSchema,
				"validJSON":   validJSON,
			},
		})
	})
}

func TestPostgres_Script(t *testing.T) {
	pgRun(t, func(t *pgTest) {
		testscript.Run(t.T, testscript.Params{
			Dir:   "testdata/postgres",
			Setup: t.setupScript,
			Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
				"only":        cmdOnly,
				"apply":       t.cmdApply,
				"exist":       t.cmdExist,
				"synced":      t.cmdSynced,
				"cmphcl":      t.cmdCmpHCL,
				"cmpshow":     t.cmdCmpShow,
				"cmpmig":      t.cmdCmpMig,
				"execsql":     t.cmdExec,
				"atlas":       t.cmdCLI,
				"clearSchema": t.clearSchema,
			},
		})
	})
}

func TestSQLite_Script(t *testing.T) {
	tt := &liteTest{T: t}
	testscript.Run(t, testscript.Params{
		Dir:   "testdata/sqlite",
		Setup: tt.setupScript,
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"apply":       tt.cmdApply,
			"exist":       tt.cmdExist,
			"synced":      tt.cmdSynced,
			"cmphcl":      tt.cmdCmpHCL,
			"cmpshow":     tt.cmdCmpShow,
			"cmpmig":      tt.cmdCmpMig,
			"execsql":     tt.cmdExec,
			"atlas":       tt.cmdCLI,
			"clearSchema": tt.clearSchema,
		},
	})
}

var keyT struct{}

func (t *myTest) setupScript(env *testscript.Env) error {
	attrs := t.defaultAttrs()
	env.Setenv("version", t.version)
	env.Setenv("charset", attrs[0].(*schema.Charset).V)
	env.Setenv("collate", attrs[1].(*schema.Collation).V)
	if err := replaceDBURL(env, t.url("")); err != nil {
		return err
	}
	return setupScript(t.T, env, t.db, "DROP SCHEMA IF EXISTS %s")
}

func replaceDBURL(env *testscript.Env, url string) error {
	// Set the workdir in the test atlas.hcl file.
	projectFile := filepath.Join(env.WorkDir, "atlas.hcl")
	if b, err := os.ReadFile(projectFile); err == nil {
		rep := strings.ReplaceAll(string(b), "URL", url)
		return os.WriteFile(projectFile, []byte(rep), 0600)
	}
	return nil
}

func (t *pgTest) setupScript(env *testscript.Env) error {
	env.Setenv("version", t.version)
	u := strings.ReplaceAll(t.url(""), "/test", "/")
	if err := replaceDBURL(env, u); err != nil {
		return err
	}
	return setupScript(t.T, env, t.db, "DROP SCHEMA IF EXISTS %s CASCADE")
}

func setupScript(t *testing.T, env *testscript.Env, db *sql.DB, dropCmd string) error {
	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	name := strings.ReplaceAll(filepath.Base(env.WorkDir), "-", "_")
	env.Setenv("db", name)
	if _, err := conn.ExecContext(ctx, fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", name)); err != nil {
		return err
	}
	env.Defer(func() {
		if _, err := conn.ExecContext(ctx, fmt.Sprintf(dropCmd, name)); err != nil {
			t.Fatal(err)
		}
		if err := conn.Close(); err != nil {
			t.Fatal(err)
		}
	})
	// Store the testscript.T for later use.
	// See "only" function below.
	env.Values[keyT] = env.T()
	if err := setupCLITest(t, env); err != nil {
		return err
	}
	return nil
}

var (
	keyDB  *sql.DB
	keyDrv *sqlite.Driver
)

const atlasPathKey = "cli.atlas"

func (t *liteTest) setupScript(env *testscript.Env) error {
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?cache=shared&_fk=1",
		filepath.Join(env.WorkDir, "atlas.sqlite")))
	require.NoError(t, err)
	env.Defer(func() {
		require.NoError(t, db.Close())
	})
	drv, err := sqlite.Open(db)
	require.NoError(t, err)
	env.Setenv("db", "main")
	// Attach connection and driver to the
	// environment as tests run in parallel.
	env.Values[keyDB] = db
	env.Values[keyDrv] = drv
	if err := setupCLITest(t.T, env); err != nil {
		return err
	}
	// Set the workdir in the test atlas.hcl file.
	projectFile := filepath.Join(env.WorkDir, "atlas.hcl")
	if b, err := os.ReadFile(projectFile); err == nil {
		rep := strings.ReplaceAll(string(b), "URL",
			fmt.Sprintf("sqlite://file:%s/atlas.sqlite?cache=shared&_fk=1", env.WorkDir))
		return os.WriteFile(projectFile, []byte(rep), 0600)
	}
	return nil
}

func setupCLITest(t *testing.T, env *testscript.Env) error {
	path, err := buildCmd(t)
	if err != nil {
		return err
	}
	env.Setenv(atlasPathKey, path)
	return nil
}

// cmdOnly executes only tests that their driver version matches the given pattern.
// For example, "only 8" or "only 8 maria*"
func cmdOnly(ts *testscript.TestScript, neg bool, args []string) {
	ver := ts.Getenv("version")
	for i := range args {
		re, rerr := regexp.Compile(`(?mi)` + args[i])
		ts.Check(rerr)
		if !neg == re.MatchString(ver) {
			return
		}
	}
	// This is not an elegant way to get the created testing.T for the script,
	// but we need some workaround to get it in order to skip specific tests.
	ts.Value(keyT).(testscript.T).Skip("skip version", ver)
}

func (t *myTest) cmdCmpShow(ts *testscript.TestScript, _ bool, args []string) {
	cmdCmpShow(ts, args, func(schema, name string) (string, error) {
		var create string
		if err := t.db.QueryRow(fmt.Sprintf("SHOW CREATE TABLE `%s`.`%s`", schema, name)).Scan(&name, &create); err != nil {
			return "", err
		}
		i := strings.LastIndexByte(create, ')')
		create, opts := create[:i+1], strings.Fields(create[i+1:])
		for _, opt := range opts {
			switch strings.Split(opt, "=")[0] {
			// Keep only options that are relevant for the tests.
			case "AUTO_INCREMENT", "COMMENT":
				create += " " + opt
			}
		}
		return create, nil
	})
}

func (t *pgTest) cmdCmpShow(ts *testscript.TestScript, _ bool, args []string) {
	cmdCmpShow(ts, args, func(schema, name string) (string, error) {
		buf, err := exec.Command("docker", "ps", "-qa", "-f", fmt.Sprintf("publish=%d", t.port)).CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("get container id %q: %v", buf, err)
		}
		buf = bytes.TrimSpace(buf)
		if len(bytes.Split(buf, []byte("\n"))) > 1 {
			return "", fmt.Errorf("multiple container ids found: %q", buf)
		}
		cmd := exec.Command("docker", "exec", string(buf), "psql", "-U", "postgres", "-d", "test", "-c", fmt.Sprintf(`\d %s.%s`, schema, name))
		// Use "cmd.String" to debug command.
		buf, err = cmd.CombinedOutput()
		if err != nil {
			return "", err
		}
		lines := strings.Split(string(buf), "\n")
		for i := range lines {
			lines[i] = strings.TrimRightFunc(lines[i], unicode.IsSpace)
		}
		return strings.Join(lines, "\n"), err
	})
}

func (t *liteTest) cmdCmpShow(ts *testscript.TestScript, _ bool, args []string) {
	cmdCmpShow(ts, args, func(_, name string) (string, error) {
		var (
			stmts []string
			db    = ts.Value(keyDB).(*sql.DB)
		)
		rows, err := db.Query("SELECT sql FROM sqlite_schema where tbl_name = ?", name)
		if err != nil {
			return "", fmt.Errorf("querying schema")
		}
		defer rows.Close()
		for rows.Next() {
			var s string
			if err := rows.Scan(&s); err != nil {
				return "", err
			}
			stmts = append(stmts, s)
		}
		return strings.Join(stmts, "\n"), nil
	})
}

func cmdCmpShow(ts *testscript.TestScript, args []string, show func(schema, name string) (string, error)) {
	if len(args) < 2 {
		ts.Fatalf("invalid number of args to 'cmpshow': %d", len(args))
	}
	var (
		ver   = ts.Getenv("version")
		fname = args[len(args)-1]
		stmts = make([]string, 0, len(args)-1)
	)
	for _, name := range args[:len(args)-1] {
		create, err := show(ts.Getenv("db"), name)
		if err != nil {
			ts.Fatalf("show table %q: %v", name, err)
		}
		stmts = append(stmts, create)
	}

	// Check if there is a file prefixed by database version (1.sql and <version>/1.sql).
	if _, err := os.Stat(ts.MkAbs(filepath.Join(ver, fname))); err == nil {
		fname = filepath.Join(ver, fname)
	}
	t1, t2 := strings.Join(stmts, "\n"), ts.ReadFile(fname)
	if strings.TrimSpace(t1) == strings.TrimSpace(t2) {
		return
	}
	var sb strings.Builder
	ts.Check(diff.Text("show", fname, t1, t2, &sb))
	ts.Fatalf(sb.String())
}

func (t *myTest) cmdCmpHCL(ts *testscript.TestScript, _ bool, args []string) {
	r := strings.NewReplacer("$charset", ts.Getenv("charset"), "$collate", ts.Getenv("collate"), "$db", ts.Getenv("db"))
	cmdCmpHCL(ts, args, func(name string) (string, error) {
		s, err := t.drv.InspectSchema(context.Background(), name, nil)
		ts.Check(err)
		buf, err := mysql.MarshalHCL(s)
		require.NoError(t, err)
		return string(buf), nil
	}, func(s string) string {
		return r.Replace(ts.ReadFile(s))
	})
}

func (t *pgTest) cmdCmpHCL(ts *testscript.TestScript, _ bool, args []string) {
	cmdCmpHCL(ts, args, func(name string) (string, error) {
		s, err := t.drv.InspectSchema(context.Background(), name, nil)
		ts.Check(err)
		buf, err := postgres.MarshalHCL(s)
		require.NoError(t, err)
		return string(buf), nil
	}, func(s string) string {
		return strings.ReplaceAll(ts.ReadFile(s), "$db", ts.Getenv("db"))
	})
}

func (t *liteTest) cmdCmpHCL(ts *testscript.TestScript, _ bool, args []string) {
	cmdCmpHCL(ts, args, func(name string) (string, error) {
		s, err := ts.Value(keyDrv).(migrate.Driver).InspectSchema(context.Background(), "main", nil)
		ts.Check(err)
		buf, err := sqlite.MarshalHCL(s)
		require.NoError(t, err)
		return string(buf), nil
	}, func(s string) string {
		return strings.ReplaceAll(ts.ReadFile(s), "$db", ts.Getenv("db"))
	})
}

func cmdCmpHCL(ts *testscript.TestScript, args []string, inspect func(schema string) (string, error), read ...func(string) string) {
	if len(args) != 1 {
		ts.Fatalf("invalid number of args to 'cmpinspect': %d", len(args))
	}
	if len(read) == 0 {
		read = append(read, ts.ReadFile)
	}
	var (
		fname = args[0]
		ver   = ts.Getenv("version")
	)
	f1, err := inspect(ts.Getenv("db"))
	if err != nil {
		ts.Fatalf("inspect schema %q: %v", ts.Getenv("db"), err)
	}
	// Check if there is a file prefixed by database version (1.sql and <version>/1.sql).
	if _, err := os.Stat(ts.MkAbs(filepath.Join(ver, fname))); err == nil {
		fname = filepath.Join(ver, fname)
	}
	f2 := read[0](fname)
	if strings.TrimSpace(f1) == strings.TrimSpace(f2) {
		return
	}
	var sb strings.Builder
	ts.Check(diff.Text("inspect", fname, f1, f2, &sb))
	ts.Fatalf(sb.String())
}

func (t *myTest) cmdExec(ts *testscript.TestScript, _ bool, args []string) {
	cmdExec(ts, args, t.db)
}

func (t *pgTest) cmdExec(ts *testscript.TestScript, _ bool, args []string) {
	cmdExec(ts, args, t.db)
}

func (t *liteTest) cmdExec(ts *testscript.TestScript, _ bool, args []string) {
	cmdExec(ts, args, ts.Value(keyDB).(*sql.DB))
}

func (t *myTest) cmdCLI(ts *testscript.TestScript, neg bool, args []string) {
	cmdCLI(ts, neg, args, t.url(ts.Getenv("db")), ts.Getenv(atlasPathKey))
}

func (t *pgTest) cmdCLI(ts *testscript.TestScript, neg bool, args []string) {
	cmdCLI(ts, neg, args, t.url(ts.Getenv("db")), ts.Getenv(atlasPathKey))
}

func (t *liteTest) cmdCLI(ts *testscript.TestScript, neg bool, args []string) {
	dbURL := fmt.Sprintf("sqlite://file:%s/atlas.sqlite?cache=shared&_fk=1", ts.Getenv("WORK"))
	cmdCLI(ts, neg, args, dbURL, ts.Getenv(atlasPathKey))
}

func cmdCLI(ts *testscript.TestScript, neg bool, args []string, dbURL, cliPath string) {
	var (
		workDir = ts.Getenv("WORK")
		r       = strings.NewReplacer("URL", dbURL, "$db", ts.Getenv("db"))
	)
	for i, arg := range args {
		args[i] = r.Replace(arg)
	}
	switch l := len(args); {
	// If command was run with a unix redirect-like suffix.
	case l > 1 && args[l-2] == ">":
		outPath := filepath.Join(workDir, args[l-1])
		f, err := os.Create(outPath)
		ts.Check(err)
		defer f.Close()
		cmd := exec.Command(cliPath, args[0:l-2]...)
		cmd.Stdout = f
		stderr := &bytes.Buffer{}
		cmd.Stderr = stderr
		cmd.Dir = workDir
		if err := cmd.Run(); err != nil && !neg {
			ts.Fatalf("\n[stderr]\n%s", stderr)
		}
	default:
		err := ts.Exec(cliPath, args...)
		if !neg {
			ts.Check(err)
		}
		if neg && err == nil {
			ts.Fatalf("expected fail")
		}
	}
}

func (t *myTest) cmdCmpMig(ts *testscript.TestScript, neg bool, args []string) {
	cmdCmpMig(ts, neg, args)
}

func (t *pgTest) cmdCmpMig(ts *testscript.TestScript, neg bool, args []string) {
	cmdCmpMig(ts, neg, args)
}

func (t *liteTest) cmdCmpMig(ts *testscript.TestScript, neg bool, args []string) {
	cmdCmpMig(ts, neg, args)
}

var reLiquibaseChangeset = regexp.MustCompile("--changeset atlas:[0-9]+-[0-9]+")

// cmdCmpMig compares a migration file under migrations with a provided file.
// If the first argument is a filename that does exist, that file is used for comparison.
// If there is no file with that name, the argument is parsed to an integer n and the
// nth sql file is used for comparison. Lexicographic order of
// the files in the directory is used to access the file of interest.
func cmdCmpMig(ts *testscript.TestScript, _ bool, args []string) {
	if len(args) < 2 {
		ts.Fatalf("invalid number of args to 'cmpmig': %d", len(args))
	}
	// Check if there is a file prefixed by database version (1.sql and <version>/1.sql).
	var (
		ver   = ts.Getenv("version")
		fname = args[1]
	)
	if _, err := os.Stat(ts.MkAbs(filepath.Join(ver, fname))); err == nil {
		fname = filepath.Join(ver, fname)
	}
	expected := strings.TrimSpace(ts.ReadFile(fname))
	dir, err := os.ReadDir(ts.MkAbs("migrations"))
	ts.Check(err)
	idx, err := strconv.Atoi(args[0])
	ts.Check(err)
	current := 0
	for _, f := range dir {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".sql") {
			continue
		}
		if current == idx {
			actual := strings.TrimSpace(ts.ReadFile(filepath.Join("migrations", f.Name())))
			exLines, acLines := strings.Split(actual, "\n"), strings.Split(expected, "\n")
			if len(exLines) != len(acLines) {
				var sb strings.Builder
				ts.Check(diff.Text(f.Name(), args[1], expected, actual, &sb))
				ts.Fatalf(sb.String())
			}
			for i := range exLines {
				// Skip liquibase changeset comments since they contain a timestamp.
				if reLiquibaseChangeset.MatchString(acLines[i]) {
					continue
				}
				if exLines[i] != acLines[i] {
					var sb strings.Builder
					ts.Check(diff.Text(f.Name(), args[1], expected, actual, &sb))
					ts.Fatalf(sb.String())
				}
			}
			return
		}
		current++
	}
	ts.Fatalf("could not find the #%d migration", idx)
}

func cmdExec(ts *testscript.TestScript, args []string, db *sql.DB) {
	if len(args) == 0 {
		ts.Fatalf("missing statements for 'execsql'")
	}
	for i := range args {
		s := strings.ReplaceAll(args[i], "$db", ts.Getenv("db"))
		_, err := db.Exec(s)
		ts.Check(err)
	}
}

func (t *myTest) cmdExist(ts *testscript.TestScript, neg bool, args []string) {
	cmdExist(ts, neg, args, func(schema, name string) (bool, error) {
		var b bool
		if err := t.db.QueryRow("SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?", schema, name).Scan(&b); err != nil {
			return false, err
		}
		return b, nil
	})
}

func (t *pgTest) cmdExist(ts *testscript.TestScript, neg bool, args []string) {
	cmdExist(ts, neg, args, func(schema, name string) (bool, error) {
		var b bool
		if err := t.db.QueryRow("SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = $1 AND TABLE_NAME = $2", schema, name).Scan(&b); err != nil {
			return false, err
		}
		return b, nil
	})
}

func (t *liteTest) cmdExist(ts *testscript.TestScript, neg bool, args []string) {
	cmdExist(ts, neg, args, func(_, name string) (bool, error) {
		var (
			b  bool
			db = ts.Value(keyDB).(*sql.DB)
		)
		if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE `type`='table' AND `name` = ?", name).Scan(&b); err != nil {
			return false, err
		}
		return b, nil
	})
}

func cmdExist(ts *testscript.TestScript, neg bool, args []string, exists func(schema, name string) (bool, error)) {
	for _, name := range args {
		b, err := exists(ts.Getenv("db"), name)
		if err != nil {
			ts.Fatalf("failed query table existence %q: %v", name, err)
		}
		if !b != neg {
			ts.Fatalf("table %q existence failed", name)
		}
	}
}

func (t *myTest) cmdSynced(ts *testscript.TestScript, neg bool, args []string) {
	cmdSynced(ts, neg, args, t.hclDiff)
}

func (t *myTest) cmdApply(ts *testscript.TestScript, neg bool, args []string) {
	cmdApply(ts, neg, args, t.drv.ApplyChanges, t.hclDiff)
}

func (t *myTest) hclDiff(ts *testscript.TestScript, name string) ([]schema.Change, error) {
	var (
		desired = &schema.Schema{}
		f       = ts.ReadFile(name)
		ctx     = context.Background()
		r       = strings.NewReplacer("$charset", ts.Getenv("charset"), "$collate", ts.Getenv("collate"), "$db", ts.Getenv("db"))
	)
	ts.Check(mysql.EvalHCLBytes([]byte(r.Replace(f)), desired, nil))
	current, err := t.drv.InspectSchema(ctx, desired.Name, nil)
	ts.Check(err)
	desired, err = t.drv.(schema.Normalizer).NormalizeSchema(ctx, desired)
	// Normalization and diffing errors should
	// be returned to the caller.
	if err != nil {
		return nil, err
	}
	changes, err := t.drv.SchemaDiff(current, desired)
	if err != nil {
		return nil, err
	}
	return changes, nil
}

func (t *pgTest) cmdSynced(ts *testscript.TestScript, neg bool, args []string) {
	cmdSynced(ts, neg, args, t.hclDiff)
}

func (t *pgTest) cmdApply(ts *testscript.TestScript, neg bool, args []string) {
	cmdApply(ts, neg, args, t.drv.ApplyChanges, t.hclDiff)
}

func (t *pgTest) hclDiff(ts *testscript.TestScript, name string) ([]schema.Change, error) {
	var (
		desired = &schema.Schema{}
		ctx     = context.Background()
		f       = strings.ReplaceAll(ts.ReadFile(name), "$db", ts.Getenv("db"))
	)
	ts.Check(postgres.EvalHCLBytes([]byte(f), desired, nil))
	current, err := t.drv.InspectSchema(ctx, desired.Name, nil)
	ts.Check(err)
	desired, err = t.drv.(schema.Normalizer).NormalizeSchema(ctx, desired)
	// Normalization and diffing errors should
	// be returned to the caller.
	if err != nil {
		return nil, err
	}
	changes, err := t.drv.SchemaDiff(current, desired)
	if err != nil {
		return nil, err
	}
	return changes, nil
}

func (t *liteTest) cmdSynced(ts *testscript.TestScript, neg bool, args []string) {
	cmdSynced(ts, neg, args, t.hclDiff)
}

func (t *liteTest) cmdApply(ts *testscript.TestScript, neg bool, args []string) {
	cmdApply(ts, neg, args, ts.Value(keyDrv).(*sqlite.Driver).ApplyChanges, t.hclDiff)
}

func (t *liteTest) hclDiff(ts *testscript.TestScript, name string) ([]schema.Change, error) {
	var (
		desired = &schema.Schema{}
		f       = ts.ReadFile(name)
		drv     = ts.Value(keyDrv).(*sqlite.Driver)
	)
	ts.Check(sqlite.EvalHCLBytes([]byte(f), desired, nil))
	current, err := drv.InspectSchema(context.Background(), desired.Name, nil)
	ts.Check(err)
	changes, err := drv.SchemaDiff(current, desired)
	// Diff errors should return to the caller.
	if err != nil {
		return nil, err
	}
	return changes, nil
}

func (t *myTest) clearSchema(ts *testscript.TestScript, _ bool, args []string) {
	if len(args) == 0 {
		args = append(args, ts.Getenv("db"))
	}
	_, err := t.db.Exec("DROP DATABASE IF EXISTS " + args[0])
	ts.Check(err)
	_, err = t.db.Exec("CREATE DATABASE IF NOT EXISTS " + args[0])
	ts.Check(err)
}

func (t *pgTest) clearSchema(ts *testscript.TestScript, _ bool, args []string) {
	if len(args) == 0 {
		args = append(args, ts.Getenv("db"))
	}
	_, err := t.db.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", args[0]))
	ts.Check(err)
	_, err = t.db.Exec("CREATE SCHEMA IF NOT EXISTS " + args[0])
	ts.Check(err)
}

func (t *liteTest) clearSchema(ts *testscript.TestScript, _ bool, _ []string) {
	for _, stmt := range []string{
		"PRAGMA writable_schema = 1;",
		"DELETE FROM sqlite_master WHERE type IN ('table', 'index', 'trigger');",
		"PRAGMA writable_schema = 0;",
		"VACUUM;",
	} {
		_, err := ts.Value(keyDB).(*sql.DB).Exec(stmt)
		ts.Check(err)
	}
}

func cmdSynced(ts *testscript.TestScript, neg bool, args []string, diff func(*testscript.TestScript, string) ([]schema.Change, error)) {
	if len(args) != 1 {
		ts.Fatalf("unexpected number of args to synced command: %d", len(args))
	}
	switch changes, err := diff(ts, args[0]); {
	case err != nil:
		ts.Fatalf("unexpected diff failure on synced: %v", err)
	case len(changes) > 0 && !neg:
		ts.Fatalf("expect no schema changes, but got: %d", len(changes))
	case len(changes) == 0 && neg:
		ts.Fatalf("expect schema changes, but there are none")
	}
}

func cmdApply(ts *testscript.TestScript, neg bool, args []string, apply func(context.Context, []schema.Change, ...migrate.PlanOption) error, diff func(*testscript.TestScript, string) ([]schema.Change, error)) {
	changes, err := diff(ts, args[0])
	switch {
	case err != nil && !neg:
		ts.Fatalf("diff states: %v", err)
	// If we expect to fail, and there's a specific error to compare.
	case err != nil && len(args) == 2:
		matchErr(ts, err, args[1])
		return
	}
	switch err := apply(context.Background(), changes); {
	case err != nil && !neg:
		ts.Fatalf("apply changes: %v", err)
	case err == nil && neg:
		ts.Fatalf("unexpected apply success")
	// If we expect to fail, and there's a specific error to compare.
	case err != nil && len(args) == 2:
		matchErr(ts, err, args[1])
	// Apply passed. Make sure there is no drift.
	case !neg:
		changes, err := diff(ts, args[0])
		ts.Check(err)
		if len(changes) > 0 {
			ts.Fatalf("unexpected schema changes: %d", len(changes))
		}
	}
}

func matchErr(ts *testscript.TestScript, err error, p string) {
	re, rerr := regexp.Compile(`(?m)` + regexp.QuoteMeta(p))
	ts.Check(rerr)
	if !re.MatchString(err.Error()) {
		ts.Fatalf("mismatched errors: %v != %s", err, p)
	}
}

func validJSON(ts *testscript.TestScript, _ bool, args []string) {
	ts.Check(json.Unmarshal([]byte(ts.ReadFile(args[0])), new(map[string]any)))
}
