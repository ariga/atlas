// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlx

import (
	"fmt"
	"testing"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"

	"github.com/stretchr/testify/require"
)

func TestDetachCycles(t *testing.T) {
	users := &schema.Table{
		Name: "users",
		Columns: []*schema.Column{
			{Name: "id", Type: &schema.ColumnType{Raw: "bigint"}},
			{Name: "workplace_id", Type: &schema.ColumnType{Raw: "bigint", Null: true}},
		},
	}
	workplaces := &schema.Table{
		Name: "workplaces",
		Columns: []*schema.Column{
			{Name: "id", Type: &schema.ColumnType{Raw: "bigint"}},
			{Name: "owner_id", Type: &schema.ColumnType{Raw: "bigint", Null: true}},
		},
	}
	users.ForeignKeys = []*schema.ForeignKey{
		{Symbol: "workplace", Table: users, Columns: users.Columns[1:2], RefTable: workplaces, RefColumns: workplaces.Columns[:1]},
	}
	changes := []schema.Change{&schema.AddTable{T: workplaces}, &schema.AddTable{T: users}}
	planned, err := DetachCycles(changes)
	require.NoError(t, err)
	require.Equal(t, changes, planned)

	deletion := []schema.Change{&schema.DropTable{T: users}, &schema.DropTable{T: workplaces}}
	planned, err = DetachCycles(deletion)
	require.NoError(t, err)
	require.Equal(t, deletion, planned)

	// Create a circular reference.
	workplaces.ForeignKeys = []*schema.ForeignKey{
		{Symbol: "owner", Table: workplaces, Columns: workplaces.Columns[1:], RefTable: users, RefColumns: users.Columns[:1]},
	}
	// Add a self-ref foreign-key.
	users.Columns = append(users.Columns, &schema.Column{Name: "spouse_id", Type: &schema.ColumnType{Raw: "bigint", Null: true}})
	users.ForeignKeys = append(users.ForeignKeys, &schema.ForeignKey{Symbol: "spouse", Table: users, Columns: users.Columns[2:], RefTable: users, RefColumns: users.Columns[:1]})

	planned, err = DetachCycles(changes)
	require.NoError(t, err)
	require.Len(t, planned, 4)
	require.Empty(t, planned[0].(*schema.AddTable).T.ForeignKeys)
	require.NotEmpty(t, planned[1].(*schema.AddTable).T.ForeignKeys)
	require.Equal(t, &schema.ModifyTable{
		T: workplaces,
		Changes: []schema.Change{
			&schema.AddForeignKey{
				F: &schema.ForeignKey{Symbol: "owner", Table: workplaces, Columns: workplaces.Columns[1:], RefTable: users, RefColumns: users.Columns[:1]},
			},
		},
	}, planned[2])
	require.Equal(t, &schema.ModifyTable{
		T: users,
		Changes: []schema.Change{
			&schema.AddForeignKey{
				F: &schema.ForeignKey{Symbol: "workplace", Table: users, Columns: users.Columns[1:2], RefTable: workplaces, RefColumns: workplaces.Columns[:1]},
			},
		},
	}, planned[3])

	planned, err = DetachCycles(deletion)
	require.NoError(t, err)
	require.Equal(t, &schema.ModifyTable{
		T: users,
		Changes: []schema.Change{
			&schema.DropForeignKey{
				F: &schema.ForeignKey{Symbol: "workplace", Table: users, Columns: users.Columns[1:2], RefTable: workplaces, RefColumns: workplaces.Columns[:1]},
			},
		},
	}, planned[0])
	require.Equal(t, &schema.ModifyTable{
		T: workplaces,
		Changes: []schema.Change{
			&schema.DropForeignKey{
				F: &schema.ForeignKey{Symbol: "owner", Table: workplaces, Columns: workplaces.Columns[1:], RefTable: users, RefColumns: users.Columns[:1]},
			},
		},
	}, planned[1])
	users.ForeignKeys = nil
	workplaces.ForeignKeys = nil
	require.Equal(t, deletion, planned[2:])

	// Delete associated table and foreign-key.
	users.AddForeignKeys(
		schema.NewForeignKey("workplace").AddColumns(users.Columns[1:2]...).SetRefTable(workplaces).AddRefColumns(workplaces.Columns[:1]...),
	)
	changes = []schema.Change{&schema.DropTable{T: workplaces}, &schema.ModifyTable{
		T: users,
		Changes: []schema.Change{
			&schema.DropForeignKey{F: users.ForeignKeys[0]},
		},
	}}
	planned, err = DetachCycles(changes)
	require.NoError(t, err)
	require.Equal(t, []schema.Change{changes[1], changes[0]}, planned)
}

func TestConsistentOrder(t *testing.T) {
	newT := func(n string) *schema.Table { return schema.NewTable(n).AddColumns(schema.NewIntColumn("id", "int")) }
	t1, t2, t3 := newT("t1"), newT("t2"), newT("t3")
	t2.AddForeignKeys(schema.NewForeignKey("t1").AddColumns(t2.Columns[0]).SetRefTable(t1).AddRefColumns(t1.Columns[0]))
	t3.AddForeignKeys(schema.NewForeignKey("t1").AddColumns(t3.Columns[0]).SetRefTable(t1).AddRefColumns(t1.Columns[0]))
	order := func() string {
		planned, err := DetachCycles([]schema.Change{
			&schema.AddTable{T: t1},
			&schema.AddTable{T: t2},
			&schema.AddTable{T: t3},
		})
		require.NoError(t, err)
		return fmt.Sprintf("%s,%s,%s", planned[0].(*schema.AddTable).T.Name, planned[1].(*schema.AddTable).T.Name, planned[2].(*schema.AddTable).T.Name)
	}
	v := order()
	for i := 0; i < 100; i++ {
		require.Equal(t, v, order(), "inconsistent order")
	}
}

func TestCheckChangesScope(t *testing.T) {
	err := CheckChangesScope(migrate.PlanOptions{}, []schema.Change{
		&schema.AddSchema{},
	})
	require.EqualError(t, err, "*schema.AddSchema is not allowed when migration plan is scoped to one schema")
	err = CheckChangesScope(migrate.PlanOptions{}, []schema.Change{
		&schema.ModifySchema{
			S: schema.New("s1"),
			Changes: []schema.Change{
				&schema.AddAttr{A: &schema.Collation{V: "utf8"}},
			},
		},
	})
	require.EqualError(t, err, "*schema.ModifySchema is not allowed when migration plan is scoped to one schema")
	err = CheckChangesScope(migrate.PlanOptions{}, []schema.Change{
		&schema.DropSchema{},
	})
	require.EqualError(t, err, "*schema.DropSchema is not allowed when migration plan is scoped to one schema")
	err = CheckChangesScope(migrate.PlanOptions{}, []schema.Change{
		&schema.AddTable{T: schema.NewTable("users").SetSchema(schema.New("s1"))},
		&schema.AddTable{T: schema.NewTable("users").SetSchema(schema.New("s2"))},
	})
	require.EqualError(t, err, "found 2 schemas when migration plan is scoped to one: [\"s1\" \"s2\"]")
}

func TestBadTypeComparison(t *testing.T) {
	var o1 struct {
		schema.Type
		schema.Object
		_ []func()
	}
	require.Panics(t, func() { _ = schema.Type(o1) == schema.Type(o1) })
	SortChanges([]schema.Change{
		&schema.DropObject{O: o1},
		&schema.DropTable{T: schema.NewTable("t1").AddColumns(schema.NewColumn("c1").SetType(o1))},
	}, nil)
}

func TestSortChanges(t *testing.T) {
	t1 := &schema.Table{
		Name: "t1",
		Columns: []*schema.Column{
			{Name: "id", Type: &schema.ColumnType{Raw: "int"}},
			{Name: "cid", Type: &schema.ColumnType{Raw: "int"}},
		},
	}
	t1.AddForeignKeys(schema.NewForeignKey("t1").AddColumns(t1.Columns[1:2]...).SetRefTable(t1).AddRefColumns(t1.Columns[:1]...))
	f1 := &schema.Func{Name: "f1", Deps: []schema.Object{t1}}
	e1 := &schema.EnumType{T: "e1", Values: []string{"v1", "v2"}}
	tr1 := &schema.Trigger{Name: "tr", Table: t1, Deps: []schema.Object{f1}}
	tr2 := &schema.Trigger{Name: "tr2", Table: t1, Deps: []schema.Object{e1}}
	changes := []schema.Change{
		&schema.ModifyTable{T: t1},
		&schema.AddTrigger{T: tr1},
		&schema.AddTable{T: t1},
		&schema.AddFunc{F: f1},
	}
	planned := SortChanges(changes, nil)
	require.Equal(t, []schema.Change{changes[2], changes[0], changes[3], changes[1]}, planned)

	changes = []schema.Change{
		&schema.DropTable{T: t1},
		&schema.DropFunc{F: f1},
		&schema.DropTrigger{T: tr1},
	}
	planned = SortChanges(changes, nil)
	require.Equal(t, []schema.Change{changes[2], changes[1], changes[0]}, planned)

	// No changes.
	planned = SortChanges([]schema.Change{
		&schema.DropFunc{F: f1},
		&schema.DropTable{T: t1},
	}, nil)
	require.Equal(t, []schema.Change{&schema.DropFunc{F: f1}, &schema.DropTable{T: t1}}, planned)

	// The table must be dropped before the enum type if one of its triggers depends on the enum type.
	t1.Triggers = []*schema.Trigger{tr2}
	changes = []schema.Change{
		&schema.DropObject{O: e1},
		&schema.DropTable{T: t1},
	}
	planned = SortChanges(changes, nil)
	require.Equal(t, []schema.Change{changes[1], changes[0]}, planned)

	// The table must be dropped before the function if one of its triggers depends on the function.
	t1.Triggers = []*schema.Trigger{tr1}
	planned = SortChanges([]schema.Change{
		&schema.DropFunc{F: f1},
		&schema.DropTable{T: t1},
	}, nil)
	require.Equal(t, []schema.Change{&schema.DropTable{T: t1}, &schema.DropFunc{F: f1}}, planned)

	// Ignore functions that do not reside on the same schema.
	f1.Schema = schema.New("s1")
	f2 := &schema.Func{Name: "f1", Args: []*schema.FuncArg{{Name: "a1"}}}
	planned = SortChanges([]schema.Change{
		&schema.AddFunc{F: f1},
		&schema.DropFunc{F: f2},
	}, nil)
	require.Equal(t, []schema.Change{&schema.AddFunc{F: f1}, &schema.DropFunc{F: f2}}, planned)

	// Order functions that reside on the same schema.
	f2.Schema = schema.New("s1")
	planned = SortChanges([]schema.Change{
		&schema.AddFunc{F: f1},
		&schema.DropFunc{F: f2},
	}, nil)
	require.Equal(t, []schema.Change{&schema.DropFunc{F: f2}, &schema.AddFunc{F: f1}}, planned)

	// Do not change the order in case there is no
	// dependency between the function and the view.
	v1 := schema.NewView("v1", "select 1")
	planned = SortChanges([]schema.Change{
		&schema.AddFunc{F: f1},
		&schema.AddView{V: v1},
	}, nil)
	require.IsType(t, &schema.AddFunc{}, planned[0])
	require.IsType(t, &schema.AddView{}, planned[1])
	// Respect dependencies between views and functions.
	f1.Deps = []schema.Object{v1}
	planned = SortChanges([]schema.Change{
		&schema.AddFunc{F: f1},
		&schema.AddView{V: v1},
	}, nil)
	require.IsType(t, &schema.AddView{}, planned[0])
	require.IsType(t, &schema.AddFunc{}, planned[1])
	// Respect copies.
	planned = SortChanges([]schema.Change{
		&schema.AddFunc{F: f1},
		&schema.AddView{V: P(V(v1))},
	}, nil)
	require.IsType(t, &schema.AddView{}, planned[0])
	require.IsType(t, &schema.AddFunc{}, planned[1])
	// Respect driver-specific logic.
	f1.Deps = nil
	planned = SortChanges([]schema.Change{
		&schema.AddFunc{F: f1},
		&schema.AddView{V: v1},
	}, &SortOptions{FuncDepV: func(*schema.Func, *schema.View) bool { return true }})
	require.IsType(t, &schema.AddView{}, planned[0])
	require.IsType(t, &schema.AddFunc{}, planned[1])

	// CREATE TRIGGER depends on ALTER TABLE.
	planned = SortChanges([]schema.Change{
		&schema.AddTrigger{T: &schema.Trigger{Name: "tr1", Table: t1, Events: []schema.TriggerEvent{schema.TriggerEventUpdateOf(t1.Columns...)}}},
		&schema.AddTable{T: t1},
	}, nil)
	require.IsType(t, &schema.AddTable{}, planned[0])
	require.IsType(t, &schema.AddTrigger{}, planned[1])

	// DROP TRIGGER depends on ALTER TABLE.
	planned = SortChanges([]schema.Change{
		&schema.ModifyTable{T: t1, Changes: []schema.Change{&schema.DropColumn{C: t1.Columns[0]}}},
		&schema.DropTrigger{T: &schema.Trigger{Name: "tr1", Table: t1, Events: []schema.TriggerEvent{schema.TriggerEventUpdateOf(t1.Columns...)}}},
	}, nil)
	require.IsType(t, &schema.DropTrigger{}, planned[0])
	require.IsType(t, &schema.ModifyTable{}, planned[1])

	// No dependency between DROP TRIGGER and ALTER TABLE.
	planned = SortChanges([]schema.Change{
		&schema.ModifyTable{T: t1, Changes: []schema.Change{&schema.DropColumn{C: t1.Columns[0]}}},
		&schema.DropTrigger{T: &schema.Trigger{Name: "tr1", Table: t1, Events: []schema.TriggerEvent{schema.TriggerEventUpdate}}},
	}, nil)
	require.IsType(t, &schema.ModifyTable{}, planned[0])
	require.IsType(t, &schema.DropTrigger{}, planned[1])
}

// RowType is a mocked RowTyper implementation.
type RowType struct {
	schema.Type
	*schema.Table
}

func (r *RowType) RowType() *schema.Table { return r.Table }

func TestSortChanges_RowTyper(t *testing.T) {
	t1, t2 := schema.NewTable("t1"), schema.NewTable("t2")
	t2.AddColumns(schema.NewColumn("t1").SetType(&RowType{Table: t1}))
	// Tables created before other tables that depend on them.
	changes := []schema.Change{&schema.AddTable{T: t2}, &schema.AddTable{T: t1}}
	planned := SortChanges(changes, nil)
	require.Equal(t, []schema.Change{changes[1], changes[0]}, planned)

	// Tables created before other columns that depend on them.
	changes = []schema.Change{
		&schema.ModifyTable{T: t2, Changes: schema.Changes{&schema.AddColumn{C: t2.Columns[0]}}},
		&schema.AddTable{T: t1},
	}
	planned = SortChanges(changes, nil)
	require.Equal(t, []schema.Change{changes[1], changes[0]}, planned)

	// Tables dropped after other tables that depend on them.
	changes = []schema.Change{&schema.DropTable{T: t1}, &schema.DropTable{T: t2}}
	planned = SortChanges(changes, nil)
	require.Equal(t, []schema.Change{changes[1], changes[0]}, planned)

	// Tables dropped after other columns that depend on them.
	changes = []schema.Change{
		&schema.DropTable{T: t1},
		&schema.ModifyTable{T: t2, Changes: schema.Changes{&schema.DropColumn{C: t2.Columns[0]}}},
	}
	planned = SortChanges(changes, nil)
	require.Equal(t, []schema.Change{changes[1], changes[0]}, planned)

	// Functions created after the tables they depend on.
	f1 := &schema.Func{Name: "f1", Ret: &RowType{Table: t1}}
	changes = []schema.Change{&schema.AddFunc{F: f1}, &schema.AddTable{T: t1}}
	planned = SortChanges(changes, nil)
	require.Equal(t, []schema.Change{changes[1], changes[0]}, planned)

	// Functions dropped before the tables they depend on.
	changes = []schema.Change{&schema.DropTable{T: t1}, &schema.DropFunc{F: f1}}
	planned = SortChanges(changes, nil)
	require.Equal(t, []schema.Change{changes[1], changes[0]}, planned)
}

func TestSameTable(t *testing.T) {
	t1 := schema.NewTable("t1")
	require.True(t, SameTable(t1, t1))
	require.True(t, SameTable(t1, schema.NewTable("t1")), "same copy")
	require.False(t, SameTable(t1, schema.NewTable("t2")))

	t1.SetSchema(schema.New("public"))
	require.True(t, SameTable(t1, t1))
	require.True(t, SameTable(t1, schema.NewTable("t1").SetSchema(schema.New("public"))), "same copy")
	require.False(t, SameTable(t1, schema.NewTable("t1")))
	require.False(t, SameTable(t1, schema.NewTable("t1").SetSchema(schema.New("private"))))
}

func TestSameView(t *testing.T) {
	v1 := schema.NewView("v1", "")
	require.True(t, SameView(v1, v1))
	require.True(t, SameView(v1, schema.NewView("v1", "")), "same copy")
	require.False(t, SameView(v1, schema.NewView("v2", "")))

	v1.SetSchema(schema.New("public"))
	require.True(t, SameView(v1, v1))
	require.True(t, SameView(v1, schema.NewView("v1", "").SetSchema(schema.New("public"))), "same copy")
	require.False(t, SameView(v1, schema.NewView("v1", "")))
	require.False(t, SameView(v1, schema.NewView("v1", "").SetSchema(schema.New("private"))))
}
