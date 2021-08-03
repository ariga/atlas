// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package schemaspec_test

import (
	"testing"

	"ariga.io/atlas/sql/internal/schemautil"
	"ariga.io/atlas/sql/schema/schemaspec"
	"github.com/stretchr/testify/require"
)

func TestAccessors(t *testing.T) {
	s := &schemaspec.Schema{
		Name: "hello",
		Tables: []*schemaspec.Table{
			{
				Name: "t1",
				Columns: []*schemaspec.Column{
					{Name: "c1", Type: "string"},
					{Name: "c2", Type: "string"},
				},
				Indexes: []*schemaspec.Index{
					{Name: "idx"},
				},
			},
			{
				Name: "t2",
			},
		},
	}
	t1, ok := s.Table("t1")
	require.True(t, ok)
	require.EqualValues(t, t1, s.Tables[0])

	t2, ok := s.Table("t2")
	require.EqualValues(t, t2, s.Tables[1])
	require.True(t, ok)

	_, ok = s.Table("t3")
	require.False(t, ok)

	c1, ok := t1.Column("c1")
	require.True(t, ok)
	require.EqualValues(t, t1.Columns[0], c1)
	c2, ok := t1.Column("c2")
	require.True(t, ok)
	require.EqualValues(t, t1.Columns[1], c2)
	_, ok = t1.Column("c3")
	require.False(t, ok)
	idx, ok := t1.Index("idx")
	require.True(t, ok)
	require.EqualValues(t, t1.Indexes[0], idx)
	_, ok = t1.Index("idx2")
	require.False(t, ok)
}

func TestColumn_Overrides(t *testing.T) {
	col := &schemaspec.Column{}
	col.Overrides = []*schemaspec.Override{
		{
			Dialect: "mysql",
			Resource: schemaspec.Resource{
				Attrs: []*schemaspec.Attr{
					{
						K: "type",
						V: &schemaspec.LiteralValue{V: "varchar(100)"},
					},
				},
			},
		},
		{
			Dialect: "mysql",
			Version: "10",
			Resource: schemaspec.Resource{
				Attrs: []*schemaspec.Attr{
					{
						K: "type",
						V: &schemaspec.LiteralValue{V: "varchar(100)"},
					},
				},
			},
		},
		{
			Dialect: "postgres",
			Resource: schemaspec.Resource{
				Attrs: []*schemaspec.Attr{
					{
						K: "type",
						V: &schemaspec.LiteralValue{V: "varchar(200)"},
					},
				},
			},
		},
	}
	myo := col.Override("mysql", "mysql 10")
	require.EqualValues(t, myo.Attrs[0], col.Overrides[0].Attrs[0])
	pgo := col.Override("postgres")
	require.EqualValues(t, pgo.Attrs[0], col.Overrides[2].Attrs[0])
}

func TestResource_Merge(t *testing.T) {
	r := &schemaspec.Resource{
		Attrs: []*schemaspec.Attr{
			schemautil.StrLitAttr("existing", "old_val"),
			schemautil.StrLitAttr("other", "val"),
			schemautil.StrLitAttr("to_bool", "val"),
		},
	}
	other := &schemaspec.Resource{
		Attrs: []*schemaspec.Attr{
			schemautil.StrLitAttr("new", "val"),
			schemautil.StrLitAttr("existing", "val"),
			schemautil.LitAttr("to_bool", "true"),
		},
	}
	r.Merge(other)
	require.EqualValues(t, "val", getStrAttr(t, r, "new"))
	require.EqualValues(t, "val", getStrAttr(t, r, "existing"))
	require.EqualValues(t, "val", getStrAttr(t, r, "other"))
	require.EqualValues(t, true, getBoolAttr(t, r, "to_bool"))
}

func getStrAttr(t *testing.T, r *schemaspec.Resource, name string) string {
	v, ok := r.Attr(name)
	require.True(t, ok)
	s, err := v.String()
	require.NoError(t, err)
	return s
}

func getBoolAttr(t *testing.T, r *schemaspec.Resource, name string) bool {
	v, ok := r.Attr(name)
	require.True(t, ok)
	b, err := v.Bool()
	require.NoError(t, err)
	return b
}
