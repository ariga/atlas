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
		{Cmd: "DROP TABLE t3", Reverse: []string{"CREATE TABLE t1(id int)", "CREATE INDEX idx ON t1(id)"}, Comment: "drop table t3"},
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
-- drop table t3
DROP TABLE t3;
`,
				v + "_tooling-plan.down.sql": `-- reverse: drop table t3
CREATE TABLE t1(id int);
CREATE INDEX idx ON t1(id);
-- reverse: create table t2
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
-- drop table t3
DROP TABLE t3;

-- +goose Down
-- reverse: drop table t3
CREATE TABLE t1(id int);
CREATE INDEX idx ON t1(id);
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
-- drop table t3
DROP TABLE t3;
`,
				"U" + v + "__tooling-plan.sql": `-- reverse: drop table t3
CREATE TABLE t1(id int);
CREATE INDEX idx ON t1(id);
-- reverse: create table t2
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
--changeset atlas:%[1]s-1
--comment: create table t1
CREATE TABLE t1(c int);
--rollback: DROP TABLE t1 IF EXISTS;

--changeset atlas:%[1]s-2
--comment: create table t2
CREATE TABLE t2(c int);
--rollback: DROP TABLE t2;

--changeset atlas:%[1]s-3
--comment: drop table t3
DROP TABLE t3;
--rollback: CREATE TABLE t1(id int);
--rollback: CREATE INDEX idx ON t1(id);
`, v),
			},
		},
		{
			"amacneil/dbmate",
			sqltool.DBMateFormatter,
			map[string]string{
				v + "_tooling-plan.sql": `-- migrate:up
-- create table t1
CREATE TABLE t1(c int);
-- create table t2
CREATE TABLE t2(c int);
-- drop table t3
DROP TABLE t3;

-- migrate:down
-- reverse: drop table t3
CREATE TABLE t1(id int);
CREATE INDEX idx ON t1(id);
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
			pl := migrate.NewPlanner(nil, d, migrate.PlanFormat(tt.fmt), migrate.PlanWithChecksum(false))
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
				{
					"ALTER TABLE post ADD updated_at TIMESTAMP NOT NULL;",
					"CREATE\nOR REPLACE FUNCTION histories_partition_creation( DATE, DATE )\nreturns void AS $$\nDECLARE\ncreate_query text;\nBEGIN\nFOR create_query IN\nSELECT 'CREATE TABLE IF NOT EXISTS histories_'\n           || TO_CHAR(d, 'YYYY_MM')\n           || ' ( CHECK( created_at >= timestamp '''\n           || TO_CHAR(d, 'YYYY-MM-DD 00:00:00')\n           || ''' AND created_at < timestamp '''\n           || TO_CHAR(d + INTERVAL '1 month', 'YYYY-MM-DD 00:00:00')\n           || ''' ) ) inherits ( histories );'\nFROM generate_series($1, $2, '1 month') AS d LOOP\n    EXECUTE create_query;\nEND LOOP;  -- LOOP END\nEND;         -- FUNCTION END\n$$\nlanguage plpgsql;",
				},
			},
		},
		{
			name: "flyway",
			dir: func() migrate.Dir {
				d, err := sqltool.NewFlywayDir("testdata/flyway")
				require.NoError(t, err)
				return d
			}(),
			versions:     []string{"2", "3", ""},
			descriptions: []string{"baseline", "third_migration", "views"},
			stmts: [][]string{
				{
					"CREATE TABLE post\n(\n    id    int NOT NULL,\n    title text,\n    body  text,\n    created_at TIMESTAMP NOT NULL\n    PRIMARY KEY (id)\n);",
					"INSERT INTO post (title, created_at) VALUES (\n'This is\nmy multiline\n\nvalue', NOW());",
				},
				{"ALTER TABLE tbl_2 ADD col_1 INTEGER NOT NULL;"},
				{"CREATE VIEW `my_view` AS SELECT * FROM `post`;"},
			},
		},
		{
			name: "liquibase",
			dir: func() migrate.Dir {
				d, err := sqltool.NewLiquibaseDir("testdata/liquibase")
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
				{"CREATE TABLE tbl_2 (col INT);"},
			},
		},
		{
			name: "dbmate",
			dir: func() migrate.Dir {
				d, err := sqltool.NewDBMateDir("testdata/dbmate")
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
				{"CREATE TABLE tbl_2 (col INT);"},
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

func TestChecksum(t *testing.T) {
	for _, tt := range []struct {
		name  string
		dir   migrate.Dir
		files []string // files expected to be part of the checksum (in order)
	}{
		{
			name: "golang-migrate",
			dir: func() migrate.Dir {
				d, err := sqltool.NewGolangMigrateDir("testdata/golang-migrate")
				require.NoError(t, err)
				return d
			}(),
			files: []string{
				"1_initial.down.sql",
				"1_initial.up.sql",
				"2_second_migration.down.sql",
				"2_second_migration.up.sql",
			},
		},
		{
			name: "goose",
			dir: func() migrate.Dir {
				d, err := sqltool.NewGooseDir("testdata/goose")
				require.NoError(t, err)
				return d
			}(),
			files: []string{
				"1_initial.sql",
				"2_second_migration.sql",
			},
		},
		{
			name: "flyway",
			dir: func() migrate.Dir {
				d, err := sqltool.NewFlywayDir("testdata/flyway")
				require.NoError(t, err)
				return d
			}(),
			files: []string{
				"B2__baseline.sql",
				"R__views.sql",
				"U1__initial.sql",
				"V1__initial.sql",
				"V2__second_migration.sql",
				"V3__third_migration.sql",
			},
		},
		{
			name: "liquibase",
			dir: func() migrate.Dir {
				d, err := sqltool.NewLiquibaseDir("testdata/liquibase")
				require.NoError(t, err)
				return d
			}(),
			files: []string{
				"1_initial.sql",
				"2_second_migration.sql",
			},
		},
		{
			name: "dbmate",
			dir: func() migrate.Dir {
				d, err := sqltool.NewDBMateDir("testdata/dbmate")
				require.NoError(t, err)
				return d
			}(),
			files: []string{
				"1_initial.sql",
				"2_second_migration.sql",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sum, err := tt.dir.Checksum()
			require.NoError(t, err)
			require.Len(t, sum, len(tt.files))
			for i := range tt.files {
				require.Equal(t, tt.files[i], sum[i].N)
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
