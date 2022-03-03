// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package integration

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"unicode"

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
				"only":    cmdOnly,
				"apply":   t.cmdApply,
				"exist":   t.cmdExist,
				"synced":  t.cmdSynced,
				"cmpshow": t.cmdCmpShow,
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
				"only":    cmdOnly,
				"apply":   t.cmdApply,
				"exist":   t.cmdExist,
				"synced":  t.cmdSynced,
				"cmpshow": t.cmdCmpShow,
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
			"apply":   tt.cmdApply,
			"exist":   tt.cmdExist,
			"synced":  tt.cmdSynced,
			"cmpshow": tt.cmdCmpShow,
		},
	})
}

var keyT struct{}

func (t *myTest) setupScript(env *testscript.Env) error {
	attrs := t.defaultAttrs()
	env.Setenv("version", t.version)
	env.Setenv("charset", attrs[0].(*schema.Charset).V)
	env.Setenv("collate", attrs[1].(*schema.Collation).V)
	return setupScript(t.T, env, t.db, "DROP SCHEMA IF EXISTS %s")
}

func (t *pgTest) setupScript(env *testscript.Env) error {
	env.Setenv("version", t.version)
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
	return nil
}

var (
	keyDB  *sql.DB
	keyDrv *sqlite.Driver
)

func (t *liteTest) setupScript(env *testscript.Env) error {
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", filepath.Base(env.WorkDir)))
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
	return nil
}

// cmdOnly executes only tests that their driver version matches the given pattern.
// For example, "only 8" or "only 8 maria*"
func cmdOnly(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("unsupported: ! only")
	}
	ver := ts.Getenv("version")
	for i := range args {
		re, rerr := regexp.Compile(`(?mi)` + args[i])
		ts.Check(rerr)
		if re.MatchString(ver) {
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
		buf, err := exec.Command("docker", "ps", "-qa", "-f", fmt.Sprintf("name=postgres%s", t.version)).CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("get container id %q: %v", buf, err)
		}
		cmd := exec.Command("docker", "exec", string(bytes.TrimSpace(buf)), "psql", "-U", "postgres", "-d", "test", "-c", fmt.Sprintf(`\d %s.%s`, schema, name))
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
	ts.Check(mysql.UnmarshalHCL([]byte(r.Replace(f)), desired))
	current, err := t.drv.InspectSchema(ctx, desired.Name, nil)
	ts.Check(err)
	desired, err = t.drv.NormalizeSchema(ctx, desired)
	// Normalization and diffing errors should
	// returned back to the caller.
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
	ts.Check(postgres.UnmarshalHCL([]byte(f), desired))
	current, err := t.drv.InspectSchema(ctx, desired.Name, nil)
	ts.Check(err)
	desired, err = t.drv.NormalizeSchema(ctx, desired)
	// Normalization and diffing errors should
	// returned back to the caller.
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
	ts.Check(sqlite.UnmarshalHCL([]byte(f), desired))
	current, err := drv.InspectSchema(context.Background(), desired.Name, nil)
	ts.Check(err)
	changes, err := drv.SchemaDiff(current, desired)
	// Diff errors should returned back to the caller.
	if err != nil {
		return nil, err
	}
	return changes, nil
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

func cmdApply(ts *testscript.TestScript, neg bool, args []string, apply func(context.Context, []schema.Change) error, diff func(*testscript.TestScript, string) ([]schema.Change, error)) {
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
	re, rerr := regexp.Compile(`(?m)` + p)
	ts.Check(rerr)
	if !re.MatchString(err.Error()) {
		ts.Fatalf("mismatched errors: %v != %s", err, p)
	}
}
