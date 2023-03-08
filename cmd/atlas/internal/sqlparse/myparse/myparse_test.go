// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package myparse_test

import (
	"strconv"
	"testing"

	"ariga.io/atlas/cmd/atlas/internal/sqlparse/myparse"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"

	"github.com/stretchr/testify/require"
)

func TestFixChange_RenameColumns(t *testing.T) {
	var p myparse.Parser
	_, err := p.FixChange(
		nil,
		"ALTER TABLE t RENAME COLUMN c1 TO c2",
		schema.Changes{&schema.AddTable{}},
	)
	require.Error(t, err)

	changes, err := p.FixChange(
		nil,
		"ALTER TABLE t RENAME COLUMN c1 TO c2",
		schema.Changes{
			&schema.ModifyTable{
				Changes: schema.Changes{
					&schema.DropColumn{C: schema.NewColumn("c1")},
					&schema.AddColumn{C: schema.NewColumn("c2")},
				},
			},
		},
	)
	require.NoError(t, err)
	require.Equal(
		t,
		schema.Changes{
			&schema.ModifyTable{
				Changes: schema.Changes{
					&schema.RenameColumn{From: schema.NewColumn("c1"), To: schema.NewColumn("c2")},
				},
			},
		},
		changes,
	)

	changes, err = p.FixChange(
		nil,
		"ALTER TABLE t ADD INDEX i(id), RENAME COLUMN c1 TO c2, ADD COLUMN c3 int, DROP COLUMN c4",
		schema.Changes{
			&schema.ModifyTable{
				Changes: schema.Changes{
					&schema.AddIndex{I: schema.NewIndex("i").AddColumns(schema.NewColumn("id"))},
					&schema.DropColumn{C: schema.NewColumn("c1")},
					&schema.AddColumn{C: schema.NewColumn("c2")},
					&schema.AddColumn{C: schema.NewColumn("c3")},
					&schema.AddColumn{C: schema.NewColumn("c4")},
				},
			},
		},
	)
	require.NoError(t, err)
	require.Equal(
		t,
		schema.Changes{
			&schema.ModifyTable{
				Changes: schema.Changes{
					&schema.AddIndex{I: schema.NewIndex("i").AddColumns(schema.NewColumn("id"))},
					&schema.RenameColumn{From: schema.NewColumn("c1"), To: schema.NewColumn("c2")},
					&schema.AddColumn{C: schema.NewColumn("c3")},
					&schema.AddColumn{C: schema.NewColumn("c4")},
				},
			},
		},
		changes,
	)
}

func TestFixChange_RenameIndexes(t *testing.T) {
	var p myparse.Parser
	changes, err := p.FixChange(
		nil,
		"ALTER TABLE t RENAME Index i1 TO i2",
		schema.Changes{
			&schema.ModifyTable{
				Changes: schema.Changes{
					&schema.DropIndex{I: schema.NewIndex("i1")},
					&schema.AddIndex{I: schema.NewIndex("i2")},
				},
			},
		},
	)
	require.NoError(t, err)
	require.Equal(
		t,
		schema.Changes{
			&schema.ModifyTable{
				Changes: schema.Changes{
					&schema.RenameIndex{From: schema.NewIndex("i1"), To: schema.NewIndex("i2")},
				},
			},
		},
		changes,
	)
}

func TestFixChange_RenameTable(t *testing.T) {
	var p myparse.Parser
	changes, err := p.FixChange(
		nil,
		"RENAME TABLE t1 TO t2",
		schema.Changes{
			&schema.DropTable{T: schema.NewTable("t1")},
			&schema.AddTable{T: schema.NewTable("t2")},
		},
	)
	require.NoError(t, err)
	require.Equal(
		t,
		schema.Changes{
			&schema.RenameTable{From: schema.NewTable("t1"), To: schema.NewTable("t2")},
		},
		changes,
	)
	changes, err = p.FixChange(
		nil,
		"RENAME TABLE `s1`.`t1` TO `s1`.`t2`;",
		schema.Changes{
			&schema.DropTable{T: schema.NewTable("t1")},
			&schema.AddTable{T: schema.NewTable("t2")},
		},
	)
	require.NoError(t, err)
	require.Equal(
		t,
		schema.Changes{
			&schema.RenameTable{From: schema.NewTable("t1"), To: schema.NewTable("t2")},
		},
		changes,
	)
	changes, err = p.FixChange(
		nil,
		"RENAME TABLE t1 TO t2, t3 TO t4",
		schema.Changes{
			&schema.DropTable{T: schema.NewTable("t1")},
			&schema.AddTable{T: schema.NewTable("t2")},
			&schema.DropTable{T: schema.NewTable("t3")},
			&schema.AddTable{T: schema.NewTable("t4")},
		},
	)
	require.NoError(t, err)
	require.Equal(
		t,
		schema.Changes{
			&schema.RenameTable{From: schema.NewTable("t1"), To: schema.NewTable("t2")},
			&schema.RenameTable{From: schema.NewTable("t3"), To: schema.NewTable("t4")},
		},
		changes,
	)
}

func TestFixChange_AlterAndRename(t *testing.T) {
	var (
		p   myparse.Parser
		drv = &mockDriver{}
	)
	drv.changes = append(drv.changes, &schema.AddColumn{C: schema.NewIntColumn("c2", "int")})
	changes, err := p.FixChange(
		drv,
		"ALTER TABLE t1 RENAME TO t2, ADD COLUMN c2 int",
		schema.Changes{
			&schema.DropTable{T: schema.NewTable("t1").AddColumns(schema.NewIntColumn("c1", "int"))},
			&schema.AddTable{T: schema.NewTable("t2").AddColumns(schema.NewIntColumn("c1", "int"), schema.NewIntColumn("c2", "int"))},
		},
	)
	require.NoError(t, err)
	require.Equal(
		t,
		schema.Changes{
			&schema.ModifyTable{
				T: schema.NewTable("t1").AddColumns(schema.NewIntColumn("c1", "int"), schema.NewIntColumn("c2", "int")),
				Changes: schema.Changes{
					&schema.AddColumn{C: schema.NewIntColumn("c2", "int")},
				},
			},
			&schema.RenameTable{
				From: schema.NewTable("t1").AddColumns(schema.NewIntColumn("c1", "int"), schema.NewIntColumn("c2", "int")),
				To:   schema.NewTable("t2").AddColumns(schema.NewIntColumn("c1", "int"), schema.NewIntColumn("c2", "int")),
			},
		},
		changes,
	)
}

func TestColumnFilledBefore(t *testing.T) {
	for i, tt := range []struct {
		file       string
		pos        int
		wantFilled bool
		wantErr    bool
	}{
		{
			file: `UPDATE t SET c = NULL;`,
			pos:  100,
		},
		{
			file:       `UPDATE t SET c = 2;`,
			pos:        100,
			wantFilled: true,
		},
		{
			file:       `UPDATE t SET c = 2 WHERE c IS NULL;`,
			pos:        100,
			wantFilled: true,
		},
		{
			file:       `UPDATE t SET c = 2 WHERE c IS NOT NULL;`,
			pos:        100,
			wantFilled: false,
		},
		{
			file:       `UPDATE t SET c = 2 WHERE c <> NULL`,
			pos:        100,
			wantFilled: false,
		},
		{
			file: `
ALTER TABLE t MODIFY COLUMN c INT NOT NULL;
UPDATE t SET c = 2 WHERE c IS NULL;
`,
			pos:        2,
			wantFilled: false,
		},
		{
			file: `
UPDATE t SET c = 2 WHERE c IS NULL;
ALTER TABLE t MODIFY COLUMN c INT NOT NULL;
`,
			pos:        30,
			wantFilled: true,
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var (
				p myparse.Parser
				f = migrate.NewLocalFile("file", []byte(tt.file))
			)
			filled, err := p.ColumnFilledBefore(f, schema.NewTable("t"), schema.NewColumn("c"), tt.pos)
			require.Equal(t, err != nil, tt.wantErr, err)
			require.Equal(t, filled, tt.wantFilled)
		})
	}
}

func TestColumnFilledAfter(t *testing.T) {
	for i, tt := range []struct {
		file       string
		pos        int
		matchValue any
		wantFilled bool
		wantErr    bool
	}{
		{
			file: `
ALTER TABLE t MODIFY COLUMN c varchar(255) NOT NULL;
UPDATE t SET c = CONCAT('tenant_', d) WHERE c = "";
`,
			matchValue: `""`,
			pos:        30,
			wantFilled: true,
		},
		{
			file: `
ALTER TABLE t MODIFY COLUMN c varchar(255) NOT NULL;
UPDATE t SET c = CONCAT('tenant_', d) WHERE c = "";
`,
			matchValue: "",
			pos:        30,
			wantFilled: true,
		},
		{
			file: `
ALTER TABLE t MODIFY COLUMN c varchar(255) NOT NULL;
UPDATE t SET c = CONCAT('tenant_', d) WHERE c = '';
`,
			matchValue: "",
			pos:        30,
			wantFilled: true,
		},
		{
			file: `
ALTER TABLE t MODIFY COLUMN c varchar(255) NOT NULL;
UPDATE t SET c = CONCAT('tenant_', d) WHERE c = 1;
`,
			matchValue: 1,
			pos:        30,
			wantFilled: true,
		},
		{
			file: `
ALTER TABLE t MODIFY COLUMN c varchar(255) NOT NULL;
UPDATE t SET c = CONCAT('tenant_', d) WHERE c = 0;
`,
			matchValue: "0",
			pos:        30,
			wantFilled: true,
		},
		{
			file: `
ALTER TABLE t MODIFY COLUMN c varchar(255) NOT NULL;
UPDATE t SET c = CONCAT('tenant_', d) WHERE c = 0;
`,
			matchValue: "0",
			pos:        100,
			wantFilled: false,
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var (
				p myparse.Parser
				f = migrate.NewLocalFile("file", []byte(tt.file))
			)
			filled, err := p.ColumnFilledAfter(f, schema.NewTable("t"), schema.NewColumn("c"), tt.pos, tt.matchValue)
			require.Equal(t, err != nil, tt.wantErr, err)
			require.Equal(t, filled, tt.wantFilled)
		})
	}
}

func TestCreateViewAfter(t *testing.T) {
	for i, tt := range []struct {
		file        string
		pos         int
		wantCreated bool
		wantErr     bool
	}{
		{
			file: `
ALTER TABLE old RENAME TO new;
CREATE VIEW old AS SELECT * FROM new;
`,
			pos:         1,
			wantCreated: true,
		},
		{
			file: `
ALTER TABLE old RENAME TO new;
CREATE VIEW old AS SELECT * FROM users;
`,
			pos: 1,
		},
		{
			file: `
ALTER TABLE old RENAME TO new;
CREATE VIEW old AS SELECT * FROM new JOIN new;
`,
			pos: 1,
		},
		{
			file: `
ALTER TABLE old RENAME TO new;
CREATE VIEW old AS SELECT * FROM new;
`,
			pos: 100,
		},
		{
			file: `
ALTER TABLE old RENAME TO new;
CREATE VIEW old AS SELECT a, b, c FROM new;
`,
			wantCreated: true,
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var (
				p myparse.Parser
				f = migrate.NewLocalFile("file", []byte(tt.file))
			)
			created, err := p.CreateViewAfter(f, "old", "new", tt.pos)
			require.Equal(t, err != nil, tt.wantErr, err)
			require.Equal(t, created, tt.wantCreated)
		})
	}
}

func TestColumnHasReferences(t *testing.T) {
	for i, tt := range []struct {
		stmt    string
		column  string
		wantHas bool
		wantErr bool
	}{
		{
			stmt:    "CREATE TABLE t(c int REFERENCES t(c));",
			column:  "c",
			wantHas: true,
		},
		{
			stmt:   "CREATE TABLE t(c int REFERENCES t(c));",
			column: "d",
		},
		{
			stmt:   "CREATE TABLE t(c int REFERENCES t(c), d int);",
			column: "d",
		},
		{
			stmt:    "ALTER TABLE t ADD COLUMN c int REFERENCES t(c);",
			column:  "c",
			wantHas: true,
		},
		{
			stmt:   "ALTER TABLE t ADD COLUMN c int REFERENCES t(c);",
			column: "d",
		},
		{
			stmt:   "ALTER TABLE t ADD COLUMN c int REFERENCES t(c), ADD d int;",
			column: "d",
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var p myparse.Parser
			hasR, err := p.ColumnHasReferences(&migrate.Stmt{Text: tt.stmt}, schema.NewColumn(tt.column))
			require.Equal(t, err != nil, tt.wantErr, err)
			require.Equal(t, hasR, tt.wantHas)
		})
	}
}

type mockDriver struct {
	migrate.Driver
	changes schema.Changes
}

func (d mockDriver) TableDiff(_, _ *schema.Table) ([]schema.Change, error) {
	return d.changes, nil
}
