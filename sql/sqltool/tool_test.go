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
	for _, tt := range []struct {
		name                   string
		dir                    migrate.Dir
		versions, descriptions []string
		stmts                  [][]string
	}{
		{
			name: "golang-migrate",
			dir: func() migrate.Dir {
				d, err := sqltool.NewGolangMigrateDir("testdata/golang-migrate")
				require.NoError(t, err)
				return d
			}(),
			versions:     []string{"1", "2"},
			descriptions: []string{"initial", "second_migration"},
			stmts: [][]string{
				{"CREATE TABLE tbl\n(\n    col INT\n);"},
				{"CREATE TABLE tbl_2 (col INT);"},
			},
		},
		{
			name: "goose",
			dir: func() migrate.Dir {
				d, err := sqltool.NewGooseDir("testdata/goose")
				require.NoError(t, err)
				return d
			}(),
			versions:     []string{"1", "2"},
			descriptions: []string{"initial", "second_migration"},
			stmts: [][]string{
				{
					"CREATE TABLE post\n(\n    id    int NOT NULL,\n    title text,\n    body  text,\n    PRIMARY KEY (id)\n);",
					"ALTER TABLE post ADD created_at TIMESTAMP NOT NULL;",
					"INSERT INTO post (title) VALUES (\n'This is\nmy multiline\n\nvalue');",
				},
				{"CREATE\nOR REPLACE FUNCTION histories_partition_creation( DATE, DATE )\nreturns void AS $$\nDECLARE\ncreate_query text;\nBEGIN\nFOR create_query IN\nSELECT 'CREATE TABLE IF NOT EXISTS histories_'\n           || TO_CHAR(d, 'YYYY_MM')\n           || ' ( CHECK( created_at >= timestamp '''\n           || TO_CHAR(d, 'YYYY-MM-DD 00:00:00')\n           || ''' AND created_at < timestamp '''\n           || TO_CHAR(d + INTERVAL '1 month', 'YYYY-MM-DD 00:00:00')\n           || ''' ) ) inherits ( histories );'\nFROM generate_series($1, $2, '1 month') AS d LOOP\n    EXECUTE create_query;\nEND LOOP;  -- LOOP END\nEND;         -- FUNCTION END\n$$\nlanguage plpgsql;"},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			files, err := tt.dir.Files()
			require.NoError(t, err)
			require.Len(t, files, len(tt.versions))
			for i := range tt.versions {
				require.Equal(t, tt.versions[i], files[i].Version())
				require.Equal(t, tt.descriptions[i], files[i].Desc())
				stmts, err := files[i].Stmts()
				require.NoError(t, err)
				require.Len(t, stmts, len(tt.stmts[i]))
				for j, stmt := range stmts {
					require.Equal(t, tt.stmts[i][j], stmt)
				}
			}
		})
	}
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
