// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqltool_test

import (
	"fmt"
	"io/fs"
	"testing"
	"time"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/sqltool"
	"github.com/stretchr/testify/require"
)

var plan = &migrate.Plan{
	Name:       "tooling-plan",
	Reversible: true,
	Changes: []*migrate.Change{
		{Cmd: "CREATE TABLE t1(c int)", Reverse: "DROP TABLE t1 IF EXISTS", Comment: "create table t1"},
		{Cmd: "CREATE TABLE t2(c int)", Reverse: "DROP TABLE t2", Comment: "create table t2"},
	},
}

func TestFormatters(t *testing.T) {
	v := time.Now().UTC().Format("20060102150405")
	for _, tt := range []struct {
		name     string
		fmt      migrate.Formatter
		expected map[string]string
	}{
		{
			"golang-migrate/migrate",
			sqltool.GolangMigrateFormatter,
			map[string]string{
				v + "_tooling-plan.up.sql": `-- create table t1
CREATE TABLE t1(c int);
-- create table t2
CREATE TABLE t2(c int);
`,
				v + "_tooling-plan.down.sql": `-- reverse: create table t2
DROP TABLE t2;
-- reverse: create table t1
DROP TABLE t1 IF EXISTS;
`,
			},
		},
		{
			"pressly/goose",
			sqltool.GooseFormatter,
			map[string]string{
				v + "_tooling-plan.sql": `-- +goose Up
-- create table t1
CREATE TABLE t1(c int);
-- create table t2
CREATE TABLE t2(c int);

-- +goose Down
-- reverse: create table t2
DROP TABLE t2;
-- reverse: create table t1
DROP TABLE t1 IF EXISTS;
`,
			},
		},
		{
			"flyway",
			sqltool.FlywayFormatter,
			map[string]string{
				"V" + v + "__tooling-plan.sql": `-- create table t1
CREATE TABLE t1(c int);
-- create table t2
CREATE TABLE t2(c int);
`,
				"U" + v + "__tooling-plan.sql": `-- reverse: create table t2
DROP TABLE t2;
-- reverse: create table t1
DROP TABLE t1 IF EXISTS;
`,
			},
		},
		{
			"liquibase",
			sqltool.LiquibaseFormatter,
			map[string]string{
				v + "_tooling-plan.sql": fmt.Sprintf(`--liquibase formatted sql
--changeset atlas:%s-1
--comment: create table t1
CREATE TABLE t1(c int);
--rollback: DROP TABLE t1 IF EXISTS;

--changeset atlas:%s-2
--comment: create table t2
CREATE TABLE t2(c int);
--rollback: DROP TABLE t2;
`, v, v),
			},
		},
		{
			"amacneil/dbmate",
			sqltool.DbmateFormatter,
			map[string]string{
				v + "_tooling-plan.sql": `-- migrate:up
-- create table t1
CREATE TABLE t1(c int);
-- create table t2
CREATE TABLE t2(c int);

-- migrate:down
-- reverse: create table t2
DROP TABLE t2;
-- reverse: create table t1
DROP TABLE t1 IF EXISTS;
`,
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			d := dir(t)
			pl := migrate.NewPlanner(nil, d, migrate.WithFormatter(tt.fmt), migrate.DisableChecksum())
			require.NotNil(t, pl)
			require.NoError(t, pl.WritePlan(plan))
			require.Equal(t, len(tt.expected), countFiles(t, d))
			for name, content := range tt.expected {
				requireFileEqual(t, d, name, content)
			}
		})
	}
}

func TestScanners(t *testing.T) {
	d, err := sqltool.NewGolangMigrateDir("testdata/golang-migrate")
	require.NoError(t, err)

	files, err := d.Files()
	require.NoError(t, err)
	require.Len(t, files, 2)
	require.Equal(t, "1_initial.up.sql", files[0].Name())
	require.Equal(t, "2_second_migration.up.sql", files[1].Name())

	first, second := files[0], files[1]

	stmts, err := d.Stmts(first)
	require.NoError(t, err)
	require.Equal(t, []string{"CREATE TABLE tbl\n(\n    col INT\n);"}, stmts)
	stmts, err = d.Stmts(second)
	require.NoError(t, err)
	require.Equal(t, []string{"CREATE TABLE tbl_2 (col INT);"}, stmts)

	v, err := d.Version(first)
	require.NoError(t, err)
	require.Equal(t, "1", v)

	desc, err := d.Desc(first)
	require.NoError(t, err)
	require.Equal(t, "initial", desc)
	desc, err = d.Desc(second)
	require.NoError(t, err)
	require.Equal(t, "second_migration", desc)
}

func dir(t *testing.T) migrate.Dir {
	p := t.TempDir()
	d, err := migrate.NewLocalDir(p)
	require.NoError(t, err)
	return d
}

func countFiles(t *testing.T, d migrate.Dir) int {
	files, err := fs.ReadDir(d, "")
	require.NoError(t, err)
	return len(files)
}

func requireFileEqual(t *testing.T, d migrate.Dir, name, contents string) {
	c, err := fs.ReadFile(d, name)
	require.NoError(t, err)
	require.Equal(t, contents, string(c))
}
