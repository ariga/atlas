// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mysql

import (
	"testing"

	"ariga.io/atlas/sql/schema"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestDiff_TableDiff(t *testing.T) {
	type testcase struct {
		name        string
		from, to    *schema.Table
		wantChanges []schema.Change
		wantErr     bool
	}
	tests := []testcase{
		{
			name:    "mismatched names",
			from:    &schema.Table{Name: "users"},
			to:      &schema.Table{Name: "pets"},
			wantErr: true,
		},
		{
			name: "no changes",
			from: &schema.Table{Name: "users", Schema: &schema.Schema{Name: "public"}},
			to:   &schema.Table{Name: "users"},
		},
		{
			name: "no changes",
			from: &schema.Table{Name: "users", Schema: &schema.Schema{Name: "public"}, Columns: []*schema.Column{{Name: "enum", Default: &schema.RawExpr{X: "'A'"}, Type: &schema.ColumnType{Type: &schema.EnumType{Values: []string{"A"}}}}}},
			to:   &schema.Table{Name: "users", Columns: []*schema.Column{{Name: "enum", Default: &schema.RawExpr{X: `"A"`}, Type: &schema.ColumnType{Type: &schema.EnumType{Values: []string{"A"}}}}}},
		},
		{
			name: "modify counter",
			from: &schema.Table{Name: "users", Schema: &schema.Schema{Name: "public"}, Attrs: []schema.Attr{&AutoIncrement{V: 1}}},
			to:   &schema.Table{Name: "users", Attrs: []schema.Attr{&AutoIncrement{V: 100}}},
			wantChanges: []schema.Change{
				&schema.ModifyAttr{
					From: &AutoIncrement{V: 1},
					To:   &AutoIncrement{V: 100},
				},
			},
		},
		// Attributes are specified and the same.
		{
			name: "no engine changes",
			from: &schema.Table{Name: "users", Schema: &schema.Schema{Name: "public"}, Attrs: []schema.Attr{&Engine{V: EngineInnoDB}}},
			to:   &schema.Table{Name: "users", Schema: &schema.Schema{Name: "public"}, Attrs: []schema.Attr{&Engine{V: EngineInnoDB}}},
		},
		// Attributes are specified and represent the same engine.
		{
			name: "no engine changes",
			from: &schema.Table{Name: "users", Schema: &schema.Schema{Name: "public"}, Attrs: []schema.Attr{&Engine{V: EngineInnoDB}}},
			to:   &schema.Table{Name: "users", Schema: &schema.Schema{Name: "public"}, Attrs: []schema.Attr{&Engine{V: "INNODB"}}},
		},
		// Attribute was dropped from the desired state, but the current state if the default (assumed one).
		{
			name: "no engine changes",
			from: &schema.Table{Name: "users", Schema: &schema.Schema{Name: "public"}, Attrs: []schema.Attr{&Engine{V: EngineInnoDB}}},
			to:   &schema.Table{Name: "users", Schema: &schema.Schema{Name: "public"}},
		},
		// Attribute was dropped from the desired state, but the current state if the default (explicitly specific as default).
		{
			name: "no engine changes",
			from: &schema.Table{Name: "users", Schema: &schema.Schema{Name: "public"}, Attrs: []schema.Attr{&Engine{V: EngineMyISAM, Default: true}}},
			to:   &schema.Table{Name: "users", Schema: &schema.Schema{Name: "public"}},
		},
		// The current state has no engine specified (unlikely case), and the desired state is set to the default (assumed one).
		{
			name: "no engine changes",
			from: &schema.Table{Name: "users", Schema: &schema.Schema{Name: "public"}},
			to:   &schema.Table{Name: "users", Schema: &schema.Schema{Name: "public"}, Attrs: []schema.Attr{&Engine{V: EngineInnoDB, Default: true}}},
		},
		// Attributes are specified, but they do not represent the same engine.
		{
			name: "engine changed",
			from: &schema.Table{Name: "users", Schema: &schema.Schema{Name: "public"}, Attrs: []schema.Attr{&Engine{V: EngineInnoDB}}},
			to:   &schema.Table{Name: "users", Schema: &schema.Schema{Name: "public"}, Attrs: []schema.Attr{&Engine{V: EngineMyISAM}}},
			wantChanges: []schema.Change{
				&schema.ModifyAttr{
					From: &Engine{V: EngineInnoDB},
					To:   &Engine{V: EngineMyISAM},
				},
			},
		},
		// The desired state has no engine specified, and the current state is not the default.
		{
			name: "engine changed",
			from: &schema.Table{Name: "users", Schema: &schema.Schema{Name: "public"}, Attrs: []schema.Attr{&Engine{V: EngineMyISAM}}},
			to:   &schema.Table{Name: "users", Schema: &schema.Schema{Name: "public"}},
			wantChanges: []schema.Change{
				&schema.ModifyAttr{
					From: &Engine{V: EngineMyISAM},
					To:   &Engine{V: EngineInnoDB, Default: true}, // Assume InnoDB is the default.
				},
			},
		},
		// The current state has no engine specified (unlikely case), and the desired state is not the default.
		{
			name: "engine changed",
			from: &schema.Table{Name: "users", Schema: &schema.Schema{Name: "public"}},
			to:   &schema.Table{Name: "users", Schema: &schema.Schema{Name: "public"}, Attrs: []schema.Attr{&Engine{V: EngineMyISAM}}},
			wantChanges: []schema.Change{
				&schema.ModifyAttr{
					From: &Engine{V: EngineInnoDB, Default: true}, // Assume InnoDB is the default.
					To:   &Engine{V: EngineMyISAM},
				},
			},
		},
		{
			name: "add collation",
			from: &schema.Table{Name: "users", Schema: &schema.Schema{Name: "public"}, Attrs: []schema.Attr{&schema.Charset{V: "latin1"}}},
			to:   &schema.Table{Name: "users", Attrs: []schema.Attr{&schema.Collation{V: "latin1_bin"}}},
			wantChanges: []schema.Change{
				&schema.AddAttr{
					A: &schema.Collation{V: "latin1_bin"},
				},
			},
		},
		{
			name: "drop collation means modify",
			from: &schema.Table{Name: "users", Schema: &schema.Schema{Name: "public", Attrs: []schema.Attr{&schema.Collation{V: "utf8mb4_0900_ai_ci"}}}, Attrs: []schema.Attr{&schema.Collation{V: "utf8mb4_bin"}}},
			to:   &schema.Table{Name: "users"},
			wantChanges: []schema.Change{
				&schema.ModifyAttr{
					From: &schema.Collation{V: "utf8mb4_bin"},
					To:   &schema.Collation{V: "utf8mb4_0900_ai_ci"},
				},
			},
		},
		{
			name: "modify collation",
			from: &schema.Table{Name: "users", Schema: &schema.Schema{Name: "public"}, Attrs: []schema.Attr{&schema.Charset{V: "latin1"}, &schema.Collation{V: "latin1_swedish_ci"}}},
			to:   &schema.Table{Name: "users", Attrs: []schema.Attr{&schema.Collation{V: "latin1_bin"}}},
			wantChanges: []schema.Change{
				&schema.ModifyAttr{
					From: &schema.Collation{V: "latin1_swedish_ci"},
					To:   &schema.Collation{V: "latin1_bin"},
				},
			},
		},
		{
			name: "drop charset means modify",
			from: &schema.Table{Name: "users", Schema: &schema.Schema{Name: "public", Attrs: []schema.Attr{&schema.Charset{V: "hebrew"}}}, Attrs: []schema.Attr{&schema.Charset{V: "hebrew_bin"}}},
			to:   &schema.Table{Name: "users"},
			wantChanges: []schema.Change{
				&schema.ModifyAttr{
					From: &schema.Charset{V: "hebrew_bin"},
					To:   &schema.Charset{V: "hebrew"},
				},
			},
		},
		{
			name: "modify charset",
			from: &schema.Table{Name: "users", Schema: &schema.Schema{Name: "public"}, Attrs: []schema.Attr{&schema.Charset{V: "utf8"}, &schema.Collation{V: "utf8_general_ci"}}},
			to:   &schema.Table{Name: "users", Attrs: []schema.Attr{&schema.Charset{V: "utf8mb4"}}},
			wantChanges: []schema.Change{
				&schema.ModifyAttr{
					From: &schema.Charset{V: "utf8"},
					To:   &schema.Charset{V: "utf8mb4"},
				},
				&schema.ModifyAttr{
					From: &schema.Collation{V: "utf8_general_ci"},
					To:   &schema.Collation{V: "utf8mb4_0900_ai_ci"},
				},
			},
		},
		{
			name: "add check",
			from: &schema.Table{Name: "t1", Schema: &schema.Schema{Name: "public"}},
			to:   &schema.Table{Name: "t1", Attrs: []schema.Attr{&schema.Check{Name: "users_chk1_c1", Expr: "(`c1` <>_latin1\\'foo\\')"}}},
			wantChanges: []schema.Change{
				&schema.AddCheck{
					C: &schema.Check{Name: "users_chk1_c1", Expr: "(`c1` <>_latin1\\'foo\\')"},
				},
			},
		},
		{
			name: "drop check",
			from: &schema.Table{Name: "t1", Schema: &schema.Schema{Name: "public"}, Attrs: []schema.Attr{&schema.Check{Name: "users_chk1_c1", Expr: "(`c1` <>_latin1\\'foo\\')"}}},
			to:   &schema.Table{Name: "t1"},
			wantChanges: []schema.Change{
				&schema.DropCheck{
					C: &schema.Check{Name: "users_chk1_c1", Expr: "(`c1` <>_latin1\\'foo\\')"},
				},
			},
		},
		{
			name: "modify check",
			from: &schema.Table{Name: "t1", Schema: &schema.Schema{Name: "public"}, Attrs: []schema.Attr{&schema.Check{Name: "users_chk1_c1", Expr: "(`c1` <>_latin1\\'foo\\')", Attrs: []schema.Attr{&Enforced{V: false}}}}},
			to:   &schema.Table{Name: "t1", Attrs: []schema.Attr{&schema.Check{Name: "users_chk1_c1", Expr: "(`c1` <>_latin1\\'foo\\')", Attrs: []schema.Attr{&Enforced{V: true}}}}},
			wantChanges: []schema.Change{
				&schema.ModifyCheck{
					From: &schema.Check{Name: "users_chk1_c1", Expr: "(`c1` <>_latin1\\'foo\\')", Attrs: []schema.Attr{&Enforced{V: false}}},
					To:   &schema.Check{Name: "users_chk1_c1", Expr: "(`c1` <>_latin1\\'foo\\')", Attrs: []schema.Attr{&Enforced{V: true}}},
				},
			},
		},
		{
			name: "add comment",
			from: &schema.Table{Name: "t1", Schema: &schema.Schema{Name: "public"}},
			to:   &schema.Table{Name: "t1", Attrs: []schema.Attr{&schema.Comment{Text: "t1"}}},
			wantChanges: []schema.Change{
				&schema.AddAttr{
					A: &schema.Comment{Text: "t1"},
				},
			},
		},
		{
			name: "drop comment",
			from: &schema.Table{Name: "t1", Schema: &schema.Schema{Name: "public"}, Attrs: []schema.Attr{&schema.Comment{Text: "t1"}}},
			to:   &schema.Table{Name: "t1"},
			wantChanges: []schema.Change{
				&schema.ModifyAttr{
					From: &schema.Comment{Text: "t1"},
					To:   &schema.Comment{Text: ""},
				},
			},
		},
		{
			name: "modify comment",
			from: &schema.Table{Name: "t1", Schema: &schema.Schema{Name: "public"}, Attrs: []schema.Attr{&schema.Comment{Text: "t1"}}},
			to:   &schema.Table{Name: "t1", Attrs: []schema.Attr{&schema.Comment{Text: "t1!"}}},
			wantChanges: []schema.Change{
				&schema.ModifyAttr{
					From: &schema.Comment{Text: "t1"},
					To:   &schema.Comment{Text: "t1!"},
				},
			},
		},
		func() testcase {
			var (
				from = &schema.Table{
					Name: "t1",
					Schema: &schema.Schema{
						Name: "public",
					},
					Columns: []*schema.Column{
						{Name: "c1", Type: &schema.ColumnType{Raw: "json", Type: &schema.JSONType{T: "json"}}},
						{Name: "c2", Type: &schema.ColumnType{Raw: "tinyint", Type: &schema.IntegerType{T: "tinyint"}}},
						{Name: "c4", Type: &schema.ColumnType{Type: &schema.FloatType{T: "float"}}, Default: &schema.Literal{V: "0.00"}},
					},
				}
				to = &schema.Table{
					Name: "t1",
					Columns: []*schema.Column{
						{
							Name:    "c1",
							Type:    &schema.ColumnType{Raw: "json", Type: &schema.JSONType{T: "json"}, Null: true},
							Default: &schema.RawExpr{X: "{}"},
							Attrs:   []schema.Attr{&schema.Comment{Text: "json comment"}},
						},
						{Name: "c3", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}},
						{Name: "c4", Type: &schema.ColumnType{Type: &schema.FloatType{T: "float"}}, Default: &schema.Literal{V: "0.00"}},
					},
				}
			)
			return testcase{
				name: "columns",
				from: from,
				to:   to,
				wantChanges: []schema.Change{
					&schema.ModifyColumn{
						From:   from.Columns[0],
						To:     to.Columns[0],
						Change: schema.ChangeNull | schema.ChangeComment | schema.ChangeDefault,
					},
					&schema.DropColumn{C: from.Columns[1]},
					&schema.AddColumn{C: to.Columns[1]},
				},
			}
		}(),
		// Custom CHARSET was dropped.
		func() testcase {
			var (
				from = schema.NewTable("t1").
					SetSchema(schema.New("public")).
					SetCharset("utf8").
					SetCollation("utf8_general_ci").
					AddColumns(schema.NewStringColumn("c1", "text").SetCharset("latin1"))
				to = schema.NewTable("t1").
					SetSchema(schema.New("public")).
					SetCharset("utf8").
					AddColumns(schema.NewStringColumn("c1", "text"))
			)
			return testcase{
				name: "columns",
				from: from,
				to:   to,
				wantChanges: []schema.Change{
					&schema.ModifyColumn{
						From:   from.Columns[0],
						To:     to.Columns[0],
						Change: schema.ChangeCharset,
					},
				},
			}
		}(),
		// Custom CHARSET was added.
		func() testcase {
			var (
				from = schema.NewTable("t1").
					SetSchema(schema.New("public")).
					SetCharset("utf8").
					SetCollation("utf8_general_ci").
					AddColumns(schema.NewStringColumn("c1", "text"))
				to = schema.NewTable("t1").
					SetSchema(schema.New("public")).
					SetCharset("utf8").
					AddColumns(schema.NewStringColumn("c1", "text").SetCharset("latin1"))
			)
			return testcase{
				name: "columns",
				from: from,
				to:   to,
				wantChanges: []schema.Change{
					&schema.ModifyColumn{
						From:   from.Columns[0],
						To:     to.Columns[0],
						Change: schema.ChangeCharset | schema.ChangeCollate,
					},
				},
			}
		}(),
		// Custom CHARSET was changed.
		func() testcase {
			var (
				from = schema.NewTable("t1").
					SetSchema(schema.New("public")).
					SetCharset("utf8").
					SetCollation("utf8_general_ci").
					AddColumns(schema.NewStringColumn("c1", "text").SetCharset("hebrew"))
				to = schema.NewTable("t1").
					SetSchema(schema.New("public")).
					SetCharset("utf8").
					SetCollation("utf8_general_ci").
					AddColumns(schema.NewStringColumn("c1", "text").SetCharset("latin1"))
			)
			return testcase{
				name: "columns",
				from: from,
				to:   to,
				wantChanges: []schema.Change{
					&schema.ModifyColumn{
						From:   from.Columns[0],
						To:     to.Columns[0],
						Change: schema.ChangeCharset | schema.ChangeCollate,
					},
				},
			}
		}(),
		// Nop CHARSET change.
		func() testcase {
			var (
				from = schema.NewTable("t1").
					SetSchema(schema.New("public")).
					SetCharset("utf8").
					SetCollation("utf8_general_ci").
					AddColumns(
						schema.NewStringColumn("c1", "text").SetCharset("utf8"),
						schema.NewStringColumn("c2", "text"),
					)
				to = schema.NewTable("t1").
					SetSchema(schema.New("public")).
					SetCollation("utf8_general_ci").
					AddColumns(
						schema.NewStringColumn("c1", "text"),
						schema.NewStringColumn("c2", "text").SetCharset("utf8"),
					)
			)
			return testcase{
				name: "columns",
				from: from,
				to:   to,
			}
		}(),
		func() testcase {
			var (
				s    = schema.New("public")
				from = schema.NewTable("t1").
					SetSchema(s).
					AddColumns(
						schema.NewIntColumn("c1", "int"),
						schema.NewIntColumn("c2", "int").
							SetGeneratedExpr(&schema.GeneratedExpr{Expr: "1", Type: "STORED"}),
						schema.NewIntColumn("c3", "int").
							SetGeneratedExpr(&schema.GeneratedExpr{Expr: "1"}),
						schema.NewIntColumn("c4", "int").
							SetGeneratedExpr(&schema.GeneratedExpr{Expr: "1", Type: "VIRTUAL"}),
						schema.NewIntColumn("c5", "int").
							SetGeneratedExpr(&schema.GeneratedExpr{Expr: "1", Type: "PERSISTENT"}),
						schema.NewIntColumn("c6", "int").
							SetGeneratedExpr(&schema.GeneratedExpr{Expr: "1", Type: "PERSISTENT"}),
					)
				to = schema.NewTable("t1").
					SetSchema(s).
					AddColumns(
						// Add generated expression.
						schema.NewIntColumn("c1", "int").
							SetGeneratedExpr(&schema.GeneratedExpr{Expr: "1", Type: "STORED"}),
						// Drop generated expression.
						schema.NewIntColumn("c2", "int"),
						// Modify generated expression.
						schema.NewIntColumn("c3", "int").
							SetGeneratedExpr(&schema.GeneratedExpr{Expr: "2"}),
						// No change.
						schema.NewIntColumn("c4", "int").
							SetGeneratedExpr(&schema.GeneratedExpr{Expr: "1"}),
						schema.NewIntColumn("c5", "int").
							SetGeneratedExpr(&schema.GeneratedExpr{Expr: "1", Type: "STORED"}),
						schema.NewIntColumn("c6", "int").
							SetGeneratedExpr(&schema.GeneratedExpr{Expr: "(1)", Type: "PERSISTENT"}),
					)
			)
			return testcase{
				name: "modify column generated",
				from: from,
				to:   to,
				wantChanges: []schema.Change{
					&schema.ModifyColumn{From: from.Columns[0], To: to.Columns[0], Change: schema.ChangeGenerated},
					&schema.ModifyColumn{From: from.Columns[1], To: to.Columns[1], Change: schema.ChangeGenerated},
					&schema.ModifyColumn{From: from.Columns[2], To: to.Columns[2], Change: schema.ChangeGenerated},
				},
			}
		}(),
		func() testcase {
			var (
				from = &schema.Table{
					Name: "t1",
					Schema: &schema.Schema{
						Name: "public",
					},
					Columns: []*schema.Column{
						{Name: "c1", Type: &schema.ColumnType{Raw: "json", Type: &schema.JSONType{T: "json"}}},
						{Name: "c2", Type: &schema.ColumnType{Raw: "tinyint", Type: &schema.IntegerType{T: "tinyint"}}},
						{Name: "c3", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}},
						{Name: "c4", Type: &schema.ColumnType{Raw: "text", Type: &schema.StringType{T: "text"}}},
					},
				}
				to = &schema.Table{
					Name: "t1",
					Schema: &schema.Schema{
						Name: "public",
					},
					Columns: []*schema.Column{
						{Name: "c1", Type: &schema.ColumnType{Raw: "json", Type: &schema.JSONType{T: "json"}}},
						{Name: "c2", Type: &schema.ColumnType{Raw: "tinyint", Type: &schema.IntegerType{T: "tinyint"}}},
						{Name: "c3", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}},
						{Name: "c4", Type: &schema.ColumnType{Raw: "text", Type: &schema.StringType{T: "text"}}},
					},
				}
			)
			from.Indexes = []*schema.Index{
				{Name: "c1_index", Unique: true, Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[0]}}},
				{Name: "c2_unique", Unique: true, Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1]}}},
				{Name: "c1_prefix", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1], Attrs: []schema.Attr{&SubPart{Len: 50}}}}},
				{Name: "c1_desc", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1]}}},
				{Name: "parser", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[3]}}, Attrs: []schema.Attr{&IndexType{T: IndexTypeFullText}, &IndexParser{P: "ngram"}}},
			}
			to.Indexes = []*schema.Index{
				{Name: "c1_index", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[0]}}},
				{Name: "c3_unique", Unique: true, Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: to.Columns[1]}}},
				{Name: "c1_prefix", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[0], Attrs: []schema.Attr{&SubPart{Len: 100}}}}},
				{Name: "c1_desc", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1], Desc: true}}},
				{Name: "parser", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[3]}}, Attrs: []schema.Attr{&IndexType{T: IndexTypeFullText}}},
			}
			return testcase{
				name: "indexes",
				from: from,
				to:   to,
				wantChanges: []schema.Change{
					&schema.ModifyIndex{From: from.Indexes[0], To: to.Indexes[0], Change: schema.ChangeUnique},
					&schema.DropIndex{I: from.Indexes[1]},
					&schema.ModifyIndex{From: from.Indexes[2], To: to.Indexes[2], Change: schema.ChangeParts},
					&schema.ModifyIndex{From: from.Indexes[3], To: to.Indexes[3], Change: schema.ChangeParts},
					&schema.ModifyIndex{From: from.Indexes[4], To: to.Indexes[4], Change: schema.ChangeAttr},
					&schema.AddIndex{I: to.Indexes[1]},
				},
			}
		}(),
		func() testcase {
			from := schema.NewTable("t1").
				SetSchema(schema.New("test")).
				AddColumns(schema.NewIntColumn("id", "int"))
			to := schema.NewTable("t1").
				SetSchema(schema.New("test")).
				AddColumns(schema.NewIntColumn("id", "int"))
			to.SetPrimaryKey(schema.NewPrimaryKey(to.Columns...))
			return testcase{
				name: "add primary-key",
				from: from,
				to:   to,
				wantChanges: []schema.Change{
					&schema.AddPrimaryKey{P: to.PrimaryKey},
				},
			}
		}(),
		func() testcase {
			from := schema.NewTable("t1").
				SetSchema(schema.New("test")).
				AddColumns(schema.NewIntColumn("id", "int"))
			from.SetPrimaryKey(schema.NewPrimaryKey(from.Columns...))
			to := schema.NewTable("t1").
				SetSchema(schema.New("test")).
				AddColumns(schema.NewIntColumn("id", "int"))
			return testcase{
				name: "drop primary-key",
				from: from,
				to:   to,
				wantChanges: []schema.Change{
					&schema.DropPrimaryKey{P: from.PrimaryKey},
				},
			}
		}(),
		func() testcase {
			from := schema.NewTable("t1").
				SetSchema(schema.New("test")).
				AddColumns(schema.NewStringColumn("id", "varchar(255)"))
			from.SetPrimaryKey(schema.NewPrimaryKey(from.Columns...))
			to := schema.NewTable("t1").
				SetSchema(schema.New("test")).
				AddColumns(schema.NewStringColumn("id", "varchar(255)"))
			to.SetPrimaryKey(
				schema.NewPrimaryKey(from.Columns...).
					AddAttrs(&IndexType{T: IndexTypeHash}),
			)
			return testcase{
				name: "modify primary-key",
				from: from,
				to:   to,
				wantChanges: []schema.Change{
					&schema.ModifyPrimaryKey{
						From:   from.PrimaryKey,
						To:     to.PrimaryKey,
						Change: schema.ChangeAttr,
					},
				},
			}
		}(),
		func() testcase {
			var (
				ref = &schema.Table{
					Name: "t2",
					Schema: &schema.Schema{
						Name: "public",
					},
					Columns: []*schema.Column{
						{Name: "id", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}},
						{Name: "ref_id", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}},
					},
				}
				from = &schema.Table{
					Name: "t1",
					Schema: &schema.Schema{
						Name: "public",
					},
					Columns: []*schema.Column{
						{Name: "t2_id", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}},
					},
				}
				to = &schema.Table{
					Name: "t1",
					Schema: &schema.Schema{
						Name: "public",
					},
					Columns: []*schema.Column{
						{Name: "t2_id", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}},
					},
				}
			)
			from.ForeignKeys = []*schema.ForeignKey{
				{Table: from, Columns: from.Columns, RefTable: ref, RefColumns: ref.Columns[:1]},
			}
			to.ForeignKeys = []*schema.ForeignKey{
				{Table: to, Columns: to.Columns, RefTable: ref, RefColumns: ref.Columns[1:]},
			}
			return testcase{
				name: "foreign-keys",
				from: from,
				to:   to,
				wantChanges: []schema.Change{
					&schema.ModifyForeignKey{
						From:   from.ForeignKeys[0],
						To:     to.ForeignKeys[0],
						Change: schema.ChangeRefColumn,
					},
				},
			}
		}(),
	}
	for _, tt := range tests {
		db, m, err := sqlmock.New()
		require.NoError(t, err)
		mock{m}.version("8.0.19")
		drv, err := Open(db)
		require.NoError(t, err)
		t.Run(tt.name, func(t *testing.T) {
			changes, err := drv.TableDiff(tt.from, tt.to)
			require.Equalf(t, tt.wantErr, err != nil, "error: %q", err)
			require.EqualValues(t, tt.wantChanges, changes)
		})
	}
}

func TestDiff_UnsupportedChecks(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	mock{m}.version("5.6.35")
	drv, err := Open(db)
	require.NoError(t, err)
	s := schema.New("public")
	changes, err := drv.TableDiff(
		schema.NewTable("t").SetSchema(s),
		schema.NewTable("t").SetSchema(s).AddChecks(schema.NewCheck()),
	)
	require.Nil(t, changes)
	require.EqualError(t, err, `version "5.6.35" does not support CHECK constraints`)
}

func TestDiff_SchemaDiff(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	mock{m}.version("8.0.19")
	drv, err := Open(db)
	require.NoError(t, err)

	changes, err := drv.SchemaDiff(&schema.Schema{Name: "public"}, &schema.Schema{Name: "test"})
	require.Error(t, err)
	require.Nil(t, changes)

	from := &schema.Schema{
		Realm: &schema.Realm{
			Attrs: []schema.Attr{
				&schema.Collation{V: "latin1"},
			},
		},
		Tables: []*schema.Table{
			{Name: "users"},
			{Name: "pets"},
		},
		Attrs: []schema.Attr{
			&schema.Collation{V: "latin1"},
		},
	}
	to := &schema.Schema{
		Tables: []*schema.Table{
			{
				Name: "users",
				Columns: []*schema.Column{
					{Name: "t2_id", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}},
				},
			},
			{Name: "groups"},
		},
		Attrs: []schema.Attr{
			&schema.Collation{V: "utf8"},
		},
	}
	from.Tables[0].Schema = from
	from.Tables[1].Schema = from
	changes, err = drv.SchemaDiff(from, to)
	require.NoError(t, err)
	require.EqualValues(t, []schema.Change{
		&schema.ModifySchema{S: to, Changes: []schema.Change{&schema.ModifyAttr{From: from.Attrs[0], To: to.Attrs[0]}}},
		&schema.ModifyTable{T: to.Tables[0], Changes: []schema.Change{&schema.AddColumn{C: to.Tables[0].Columns[0]}}},
		&schema.DropTable{T: from.Tables[1]},
		&schema.AddTable{T: to.Tables[1]},
	}, changes)
}

func TestDiff_LowerCaseMode(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	mock{m}.lcmode("8.0.19", "1")
	drv, err := Open(db)
	require.NoError(t, err)
	changes, err := drv.SchemaDiff(
		schema.New("public").AddTables(schema.NewTable("t"), schema.NewTable("S")),
		schema.New("public").AddTables(schema.NewTable("T"), schema.NewTable("s")),
	)
	require.NoError(t, err)
	require.Nil(t, changes)

	changes, err = drv.SchemaDiff(
		schema.New("public").AddTables(schema.NewTable("t")),
		schema.New("public").AddTables(schema.NewTable("T"), schema.NewTable("t")),
	)
	require.EqualError(t, err, `2 matches found for table "t"`)
	require.Nil(t, changes)

	changes, err = drv.SchemaDiff(
		schema.New("public").AddTables(schema.NewTable("t")),
		schema.New("public").AddTables(schema.NewTable("s"), schema.NewTable("S")),
	)
	require.NoError(t, err)
	require.Len(t, changes, 3)

	mock{m}.lcmode("8.0.19", "2")
	drv, err = Open(db)
	require.NoError(t, err)
	changes, err = drv.SchemaDiff(
		schema.New("public").AddTables(schema.NewTable("t"), schema.NewTable("S")),
		schema.New("public").AddTables(schema.NewTable("T"), schema.NewTable("s")),
	)
	require.NoError(t, err)
	require.Nil(t, changes)

	mock{m}.lcmode("8.0.19", "3")
	drv, err = Open(db)
	require.NoError(t, err)
	changes, err = drv.SchemaDiff(
		schema.New("public").AddTables(schema.NewTable("t"), schema.NewTable("S")),
		schema.New("public").AddTables(schema.NewTable("T"), schema.NewTable("s")),
	)
	require.EqualError(t, err, `unsupported 'lower_case_table_names' mode: 3`)
	require.Nil(t, changes)
}

func TestDiff_RealmDiff(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	mock{m}.version("8.0.19")
	drv, err := Open(db)
	require.NoError(t, err)
	from := &schema.Realm{
		Schemas: []*schema.Schema{
			{
				Name: "public",
				Tables: []*schema.Table{
					{Name: "users"},
					{Name: "pets"},
				},
				Attrs: []schema.Attr{
					&schema.Collation{V: "latin1"},
				},
			},
			{
				Name: "internal",
				Tables: []*schema.Table{
					{Name: "pets"},
				},
			},
		},
	}
	to := &schema.Realm{
		Schemas: []*schema.Schema{
			{
				Name: "public",
				Tables: []*schema.Table{
					{
						Name: "users",
						Columns: []*schema.Column{
							{Name: "t2_id", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}},
						},
					},
					{Name: "pets"},
				},
				Attrs: []schema.Attr{
					&schema.Collation{V: "utf8"},
				},
			},
			{
				Name: "test",
				Tables: []*schema.Table{
					{Name: "pets"},
				},
			},
		},
	}
	from.Schemas[0].Realm = from
	from.Schemas[0].Tables[0].Schema = from.Schemas[0]
	from.Schemas[0].Tables[1].Schema = from.Schemas[0]
	to.Schemas[0].Realm = to
	to.Schemas[0].Tables[0].Schema = to.Schemas[0]
	changes, err := drv.RealmDiff(from, to)
	require.NoError(t, err)
	require.EqualValues(t, []schema.Change{
		&schema.ModifySchema{S: to.Schemas[0], Changes: []schema.Change{&schema.ModifyAttr{From: from.Schemas[0].Attrs[0], To: to.Schemas[0].Attrs[0]}}},
		&schema.ModifyTable{T: to.Schemas[0].Tables[0], Changes: []schema.Change{&schema.AddColumn{C: to.Schemas[0].Tables[0].Columns[0]}}},
		&schema.DropSchema{S: from.Schemas[1]},
		&schema.AddSchema{S: to.Schemas[1]},
		&schema.AddTable{T: to.Schemas[1].Tables[0]},
	}, changes)
}

func TestDefaultDiff(t *testing.T) {
	changes, err := DefaultDiff.SchemaDiff(
		schema.New("public").
			AddTables(
				schema.NewTable("users").AddColumns(schema.NewIntColumn("id", "int")),
			),
		schema.New("public"),
	)
	require.NoError(t, err)
	require.Len(t, changes, 1)
	require.IsType(t, &schema.DropTable{}, changes[0])
}

func TestSkipChanges(t *testing.T) {
	t.Run("DropSchema", func(t *testing.T) {
		from, to := schema.NewRealm(schema.New("public")), schema.NewRealm()
		changes, err := DefaultDiff.RealmDiff(from, to)
		require.NoError(t, err)
		require.Len(t, changes, 1)
		changes, err = DefaultDiff.RealmDiff(from, to, schema.DiffSkipChanges(&schema.DropSchema{}))
		require.NoError(t, err)
		require.Empty(t, changes)
	})

	t.Run("DropTable", func(t *testing.T) {
		from, to := schema.NewRealm(schema.New("public").AddTables(schema.NewTable("users"))), schema.NewRealm(schema.New("public"))
		changes, err := DefaultDiff.RealmDiff(from, to)
		require.NoError(t, err)
		require.Len(t, changes, 1)
		changes, err = DefaultDiff.SchemaDiff(from.Schemas[0], to.Schemas[0])
		require.NoError(t, err)
		require.Len(t, changes, 1)

		changes, err = DefaultDiff.RealmDiff(from, to, schema.DiffSkipChanges(&schema.DropTable{}))
		require.NoError(t, err)
		require.Empty(t, changes)
		changes, err = DefaultDiff.SchemaDiff(from.Schemas[0], to.Schemas[0], schema.DiffSkipChanges(&schema.DropTable{}))
		require.NoError(t, err)
		require.Empty(t, changes)
	})

	t.Run("ModifyTable", func(t *testing.T) {
		from := schema.NewRealm(
			schema.New("public").AddTables(
				schema.NewTable("users").
					AddColumns(schema.NewIntColumn("id", "int")).
					AddIndexes(schema.NewIndex("users_id_idx").AddColumns(schema.NewIntColumn("id", "int"))),
			),
		)
		to := schema.NewRealm(schema.New("public").AddTables(schema.NewTable("users")))
		changes, err := DefaultDiff.RealmDiff(from, to)
		require.NoError(t, err)
		require.Len(t, changes, 1)
		require.Len(t, changes[0].(*schema.ModifyTable).Changes, 2)
		changes, err = DefaultDiff.SchemaDiff(from.Schemas[0], to.Schemas[0])
		require.NoError(t, err)
		require.Len(t, changes, 1)
		require.Len(t, changes[0].(*schema.ModifyTable).Changes, 2)

		changes, err = DefaultDiff.RealmDiff(from, to, schema.DiffSkipChanges(&schema.ModifyTable{}))
		require.NoError(t, err)
		require.Empty(t, changes)
		changes, err = DefaultDiff.SchemaDiff(from.Schemas[0], to.Schemas[0], schema.DiffSkipChanges(&schema.ModifyTable{}))
		require.NoError(t, err)
		require.Empty(t, changes)

		changes, err = DefaultDiff.RealmDiff(from, to, schema.DiffSkipChanges(&schema.DropColumn{}, &schema.DropIndex{}))
		require.NoError(t, err)
		require.Empty(t, changes)
		changes, err = DefaultDiff.SchemaDiff(from.Schemas[0], to.Schemas[0], schema.DiffSkipChanges(&schema.DropColumn{}, &schema.DropIndex{}))
		require.NoError(t, err)
		require.Empty(t, changes)

		// Ignore partial table changes.
		changes, err = DefaultDiff.RealmDiff(from, to, schema.DiffSkipChanges(&schema.DropColumn{}))
		require.NoError(t, err)
		require.Len(t, changes, 1)
		require.Len(t, changes[0].(*schema.ModifyTable).Changes, 1)
		require.IsType(t, &schema.DropIndex{}, changes[0].(*schema.ModifyTable).Changes[0])
		changes, err = DefaultDiff.SchemaDiff(from.Schemas[0], to.Schemas[0], schema.DiffSkipChanges(&schema.DropIndex{}))
		require.NoError(t, err)
		require.Len(t, changes, 1)
		require.Len(t, changes[0].(*schema.ModifyTable).Changes, 1)
		require.IsType(t, &schema.DropColumn{}, changes[0].(*schema.ModifyTable).Changes[0])
	})
}
