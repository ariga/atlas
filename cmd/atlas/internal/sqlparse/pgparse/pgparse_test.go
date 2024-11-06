// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package pgparse_test

import (
	"strconv"
	"testing"

	"ariga.io/atlas/cmd/atlas/internal/sqlparse/pgparse"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/schema"

	"github.com/stretchr/testify/require"
)

func TestFixChange_RenameColumns(t *testing.T) {
	var p pgparse.Parser
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

	changes, err = p.FixChange(
		nil,
		"ALTER TABLE t RENAME COLUMN c1 TO c2",
		schema.Changes{
			&schema.ModifyTable{
				Changes: schema.Changes{
					&schema.DropColumn{C: schema.NewColumn("c1")},
					&schema.AddColumn{C: schema.NewColumn("c2")},
				},
			},
			&schema.ModifyView{
				From: &schema.View{Name: "t", Def: "select c1 from t"},
				To:   &schema.View{Name: "t", Def: "select c2 from t"},
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
			&schema.ModifyView{
				From: &schema.View{Name: "t", Def: "select c1 from t"},
				To:   &schema.View{Name: "t", Def: "select c2 from t"},
			},
		},
		changes,
	)
}

func TestFixChange_RenameIndexes(t *testing.T) {
	var p pgparse.Parser
	changes, err := p.FixChange(
		nil,
		"ALTER INDEX IF EXISTS i1 RENAME TO i2",
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

func TestFixChange_CreateIndexCon(t *testing.T) {
	var p pgparse.Parser
	changes, err := p.FixChange(
		nil,
		"CREATE INDEX i1 ON t1 (c1)",
		schema.Changes{
			&schema.ModifyTable{
				T: schema.NewTable("t1"),
				Changes: schema.Changes{
					&schema.AddIndex{I: schema.NewIndex("i1")},
				},
			},
		},
	)
	require.NoError(t, err)
	// No changes.
	require.Equal(
		t,
		schema.Changes{
			&schema.ModifyTable{
				T: schema.NewTable("t1"),
				Changes: schema.Changes{
					&schema.AddIndex{
						I: schema.NewIndex("i1"),
					},
				},
			},
		},
		changes,
	)

	changes, err = p.FixChange(
		nil,
		"CREATE INDEX CONCURRENTLY i1 ON t1 (c1)",
		schema.Changes{
			&schema.ModifyTable{
				T: schema.NewTable("t1"),
				Changes: schema.Changes{
					&schema.AddIndex{I: schema.NewIndex("i1")},
				},
			},
		},
	)
	require.NoError(t, err)
	// Should add the "Concurrently" clause to the AddIndex command.
	require.Equal(
		t,
		schema.Changes{
			&schema.ModifyTable{
				T: schema.NewTable("t1"),
				Changes: schema.Changes{
					&schema.AddIndex{
						I: schema.NewIndex("i1"),
						Extra: []schema.Clause{
							&postgres.Concurrently{},
						},
					},
				},
			},
		},
		changes,
	)

	changes, err = p.FixChange(
		nil,
		"CREATE INDEX CONCURRENTLY i1 ON t1 (c1)",
		schema.Changes{
			&schema.ModifyTable{
				T: schema.NewTable("t1"),
				Changes: schema.Changes{
					&schema.AddIndex{
						I: schema.NewIndex("i1"),
						Extra: []schema.Clause{
							&postgres.Concurrently{},
						},
					},
				},
			},
		},
	)
	require.NoError(t, err)
	// The "Concurrently" clause should not be added if it already exists.
	require.Equal(
		t,
		schema.Changes{
			&schema.ModifyTable{
				T: schema.NewTable("t1"),
				Changes: schema.Changes{
					&schema.AddIndex{
						I: schema.NewIndex("i1"),
						Extra: []schema.Clause{
							&postgres.Concurrently{},
						},
					},
				},
			},
		},
		changes,
	)
	// Support quoted identifiers.
	changes, err = p.FixChange(
		nil,
		`CREATE INDEX CONCURRENTLY "i1" ON t1 (c1)`,
		schema.Changes{
			&schema.ModifyTable{
				T: schema.NewTable("t1"),
				Changes: schema.Changes{
					&schema.AddIndex{I: schema.NewIndex("i1")},
				},
			},
		},
	)
	require.NoError(t, err)
	m, ok := changes[0].(*schema.ModifyTable)
	require.True(t, ok)
	require.Equal(t, &postgres.Concurrently{}, m.Changes[0].(*schema.AddIndex).Extra[0])

	// Support qualified quoted identifiers.
	changes, err = p.FixChange(
		nil,
		`CREATE INDEX CONCURRENTLY "i1" ON "public".t1 (c1)`,
		schema.Changes{
			&schema.ModifyTable{
				T: schema.NewTable("t1").
					SetSchema(schema.New("public")),
				Changes: schema.Changes{
					&schema.AddIndex{I: schema.NewIndex("i1")},
				},
			},
		},
	)
	require.NoError(t, err)
	m, ok = changes[0].(*schema.ModifyTable)
	require.True(t, ok)
	require.Equal(t, &postgres.Concurrently{}, m.Changes[0].(*schema.AddIndex).Extra[0])
}

func TestFixChange_DropIndexCon(t *testing.T) {
	var p pgparse.Parser
	changes, err := p.FixChange(
		nil,
		"DROP INDEX i1",
		schema.Changes{
			&schema.ModifyTable{
				T: schema.NewTable("t1"),
				Changes: schema.Changes{
					&schema.DropIndex{I: schema.NewIndex("i1")},
				},
			},
		},
	)
	require.NoError(t, err)
	// No changes.
	require.Equal(
		t,
		schema.Changes{
			&schema.ModifyTable{
				T: schema.NewTable("t1"),
				Changes: schema.Changes{
					&schema.DropIndex{
						I: schema.NewIndex("i1"),
					},
				},
			},
		},
		changes,
	)

	changes, err = p.FixChange(
		nil,
		"DROP INDEX CONCURRENTLY i1",
		schema.Changes{
			&schema.ModifyTable{
				T: schema.NewTable("t1"),
				Changes: schema.Changes{
					&schema.DropIndex{I: schema.NewIndex("i1")},
				},
			},
		},
	)
	require.NoError(t, err)
	// Should add the "Concurrently" clause to the DropIndex command.
	require.Equal(
		t,
		schema.Changes{
			&schema.ModifyTable{
				T: schema.NewTable("t1"),
				Changes: schema.Changes{
					&schema.DropIndex{
						I: schema.NewIndex("i1"),
						Extra: []schema.Clause{
							&postgres.Concurrently{},
						},
					},
				},
			},
		},
		changes,
	)

	changes, err = p.FixChange(
		nil,
		"DROP INDEX CONCURRENTLY i1",
		schema.Changes{
			&schema.ModifyTable{
				T: schema.NewTable("t1"),
				Changes: schema.Changes{
					&schema.DropIndex{
						I: schema.NewIndex("i1"),
						Extra: []schema.Clause{
							&postgres.Concurrently{},
						},
					},
				},
			},
		},
	)
	require.NoError(t, err)
	// The "Concurrently" clause should not be added if it already exists.
	require.Equal(
		t,
		schema.Changes{
			&schema.ModifyTable{
				T: schema.NewTable("t1"),
				Changes: schema.Changes{
					&schema.DropIndex{
						I: schema.NewIndex("i1"),
						Extra: []schema.Clause{
							&postgres.Concurrently{},
						},
					},
				},
			},
		},
		changes,
	)
	// Support quoted identifiers.
	changes, err = p.FixChange(
		nil,
		`DROP INDEX CONCURRENTLY "i1"`,
		schema.Changes{
			&schema.ModifyTable{
				T: schema.NewTable("t1"),
				Changes: schema.Changes{
					&schema.DropIndex{I: schema.NewIndex("i1")},
				},
			},
		},
	)
	require.NoError(t, err)
	m, ok := changes[0].(*schema.ModifyTable)
	require.True(t, ok)
	require.Equal(t, &postgres.Concurrently{}, m.Changes[0].(*schema.DropIndex).Extra[0])

	// Support qualified identifiers.
	changes, err = p.FixChange(
		nil,
		`DROP INDEX CONCURRENTLY public.i1`,
		schema.Changes{
			&schema.ModifyTable{
				T: schema.NewTable("t1").
					SetSchema(schema.New("public")),
				Changes: schema.Changes{
					&schema.DropIndex{I: schema.NewIndex("i1")},
				},
			},
		},
	)
	require.NoError(t, err)
	m, ok = changes[0].(*schema.ModifyTable)
	require.True(t, ok)
	require.Equal(t, &postgres.Concurrently{}, m.Changes[0].(*schema.DropIndex).Extra[0])

	// Support qualified quoted identifiers.
	changes, err = p.FixChange(
		nil,
		`DROP INDEX CONCURRENTLY "public"."i1"`,
		schema.Changes{
			&schema.ModifyTable{
				T: schema.NewTable("t1").
					SetSchema(schema.New("public")),
				Changes: schema.Changes{
					&schema.DropIndex{I: schema.NewIndex("i1")},
				},
			},
		},
	)
	require.NoError(t, err)
	m, ok = changes[0].(*schema.ModifyTable)
	require.True(t, ok)
	require.Equal(t, &postgres.Concurrently{}, m.Changes[0].(*schema.DropIndex).Extra[0])

	// Multiple indexes.
	changes, err = p.FixChange(
		nil,
		`DROP INDEX CONCURRENTLY i1, i2`,
		schema.Changes{
			&schema.ModifyTable{
				T: schema.NewTable("t1"),
				Changes: schema.Changes{
					&schema.DropIndex{I: schema.NewIndex("i1")},
					&schema.DropIndex{I: schema.NewIndex("i2")},
				},
			},
		},
	)
	require.NoError(t, err)
	m, ok = changes[0].(*schema.ModifyTable)
	require.True(t, ok)
	require.Equal(t, &postgres.Concurrently{}, m.Changes[0].(*schema.DropIndex).Extra[0])
	require.Equal(t, &postgres.Concurrently{}, m.Changes[1].(*schema.DropIndex).Extra[0])
}

func TestFixChange_RenameTable(t *testing.T) {
	var p pgparse.Parser
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
			file: `UPDATE t SET c = 2;`,
		},
		{
			file: `UPDATE t SET c = 2;`,
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
			var p pgparse.Parser
			stmts, err := migrate.Stmts(tt.file)
			require.NoError(t, err)
			filled, err := p.ColumnFilledBefore(stmts, schema.NewTable("t"), schema.NewColumn("c"), tt.pos)
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
		CREATE VIEW old AS (SELECT * FROM "new");
		`,
			pos:         1,
			wantCreated: true,
		},
		{
			file: `
		ALTER TABLE old RENAME TO new;
		CREATE VIEW old AS (SELECT * FROM "1");
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
			var p pgparse.Parser
			stmts, err := migrate.Stmts(tt.file)
			require.NoError(t, err)
			created, err := p.CreateViewAfter(stmts, "old", "new", tt.pos)
			require.Equal(t, err != nil, tt.wantErr, err)
			require.Equal(t, tt.wantCreated, created)
		})
	}
}
