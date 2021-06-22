package schemaspec

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAccessors(t *testing.T) {
	s := &Schema{
		Name: "hello",
		Tables: []*Table{
			{
				Name: "t1",
				Columns: []*Column{
					{Name: "c1", Type: "string"},
					{Name: "c2", Type: "string"},
				},
				Indexes: []*Index{
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
