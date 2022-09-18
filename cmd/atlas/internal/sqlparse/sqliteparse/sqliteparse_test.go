// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqliteparse_test

import (
	"strconv"
	"testing"

	"ariga.io/atlas/cmd/atlas/internal/sqlparse/sqliteparse"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"

	"github.com/stretchr/testify/require"
)

func TestFixChange_RenameColumns(t *testing.T) {
	var p sqliteparse.FileParser
	_, err := p.FixChange(
		nil,
		"ALTER TABLE t RENAME COLUMN c1 TO c2",
		nil,
	)
	require.Error(t, err)

	_, err = p.FixChange(
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
}

func TestFixChange_RenameTable(t *testing.T) {
	var p sqliteparse.FileParser
	changes, err := p.FixChange(
		nil,
		"ALTER TABLE t1 RENAME TO t2",
		schema.Changes{
			&schema.DropTable{T: schema.NewTable("t1")},
			&schema.AddTable{T: schema.NewTable("t2")},
			&schema.AddTable{T: schema.NewTable("t3")},
		},
	)
	require.NoError(t, err)
	require.Equal(
		t,
		schema.Changes{
			&schema.RenameTable{From: schema.NewTable("t1"), To: schema.NewTable("t2")},
			&schema.AddTable{T: schema.NewTable("t3")},
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
			file:       "UPDATE `t` SET c = 2;",
			pos:        100,
			wantFilled: true,
		},
		{
			file:       `UPDATE t SET c = 2 WHERE c IS NULL;`,
			pos:        100,
			wantFilled: true,
		},
		{
			file:       "UPDATE `t` SET `c` = 2 WHERE `c` IS NULL;",
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
UPDATE t1 SET c = 2 WHERE c IS NULL;
UPDATE t SET c = 2 WHERE c IS NULL;
`,
			pos:        2,
			wantFilled: false,
		},
		{
			file: `
UPDATE t SET c = 2 WHERE c IS NULL;
UPDATE t1 SET c = 2 WHERE c IS NULL;
`,
			pos:        30,
			wantFilled: true,
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var (
				p sqliteparse.FileParser
				f = migrate.NewLocalFile("file", []byte(tt.file))
			)
			filled, err := p.ColumnFilledBefore(f, schema.NewTable("t"), schema.NewColumn("c"), tt.pos)
			require.Equal(t, err != nil, tt.wantErr, err)
			require.Equal(t, filled, tt.wantFilled)
		})
	}
}
