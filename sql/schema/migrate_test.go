// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package schema_test

import (
	"fmt"
	"log"
	"strconv"
	"testing"

	"ariga.io/atlas/sql/schema"

	"github.com/stretchr/testify/require"
)

func TestChanges_IndexAddTable(t *testing.T) {
	changes := schema.Changes{
		&schema.AddTable{T: schema.NewTable("users")},
		&schema.DropTable{T: schema.NewTable("posts")},
		&schema.AddTable{T: schema.NewTable("posts")},
		&schema.AddTable{T: schema.NewTable("posts")},
	}
	require.Equal(t, 2, changes.IndexAddTable("posts"))
	require.Equal(t, -1, changes.IndexAddTable("post_tags"))
}

func TestChanges_IndexDropTable(t *testing.T) {
	changes := schema.Changes{
		&schema.DropTable{T: schema.NewTable("users")},
		&schema.AddTable{T: schema.NewTable("posts")},
		&schema.DropTable{T: schema.NewTable("posts")},
	}
	require.Equal(t, 2, changes.IndexDropTable("posts"))
	require.Equal(t, -1, changes.IndexDropTable("post_tags"))
}

func TestChanges_IndexAddColumn(t *testing.T) {
	changes := schema.Changes{
		&schema.AddColumn{C: schema.NewColumn("name")},
		&schema.DropColumn{C: schema.NewColumn("name")},
		&schema.AddColumn{C: schema.NewColumn("name")},
	}
	require.Equal(t, 0, changes.IndexAddColumn("name"))
	require.Equal(t, -1, changes.IndexAddColumn("created_at"))
}

func TestChanges_IndexDropColumn(t *testing.T) {
	changes := schema.Changes{
		&schema.AddColumn{C: schema.NewColumn("name")},
		&schema.DropColumn{C: schema.NewColumn("name")},
		&schema.AddColumn{C: schema.NewColumn("name")},
	}
	require.Equal(t, 1, changes.IndexDropColumn("name"))
	require.Equal(t, -1, changes.IndexDropColumn("created_at"))
}

func TestChanges_IndexAddIndex(t *testing.T) {
	changes := schema.Changes{
		&schema.DropIndex{I: schema.NewIndex("name")},
		&schema.AddIndex{I: schema.NewIndex("created_at")},
		&schema.AddIndex{I: schema.NewIndex("name")},
	}
	require.Equal(t, 2, changes.IndexAddIndex("name"))
	require.Equal(t, -1, changes.IndexAddIndex("age"))
}

func TestChanges_IndexDropIndex(t *testing.T) {
	changes := schema.Changes{
		&schema.AddIndex{I: schema.NewIndex("name")},
		&schema.DropIndex{I: schema.NewIndex("created_at")},
		&schema.DropIndex{I: schema.NewIndex("name")},
	}
	require.Equal(t, 2, changes.IndexDropIndex("name"))
	require.Equal(t, -1, changes.IndexDropIndex("age"))
}

func TestChanges_RemoveIndex(t *testing.T) {
	changes := make(schema.Changes, 0, 5)
	for i := 0; i < 5; i++ {
		changes = append(changes, &schema.AddColumn{C: schema.NewColumn(strconv.Itoa(i))})
	}
	changes.RemoveIndex(0)
	require.Equal(t, 4, len(changes))
	for i := 0; i < 4; i++ {
		require.Equal(t, strconv.Itoa(i+1), changes[i].(*schema.AddColumn).C.Name)
	}
	changes.RemoveIndex(0, 3, 2)
	require.Equal(t, 1, len(changes))
	require.Equal(t, "2", changes[0].(*schema.AddColumn).C.Name)
}

func ExampleChanges_Replace() {
	changes := schema.Changes{
		&schema.AddIndex{I: schema.NewIndex("id")},
		&schema.AddColumn{C: schema.NewColumn("new_name")},
		&schema.AddColumn{C: schema.NewColumn("id")},
		&schema.AddColumn{C: schema.NewColumn("created_at")},
		&schema.DropColumn{C: schema.NewColumn("old_name")},
	}
	i, j := changes.IndexAddColumn("new_name"), changes.IndexDropColumn("old_name")
	if i == -1 || j == -1 {
		log.Fatalln("Unexpected change positions")
	}
	// Replace "add" and "drop" with "rename".
	changes = append(changes, &schema.RenameColumn{From: changes[j].(*schema.DropColumn).C, To: changes[i].(*schema.AddColumn).C})
	changes.RemoveIndex(i, j)
	for _, c := range changes {
		switch c := c.(type) {
		case *schema.AddColumn:
			fmt.Printf("%T(%s)\n", c, c.C.Name)
		case *schema.DropColumn:
			fmt.Printf("%T(%s)\n", c, c.C.Name)
		case *schema.RenameColumn:
			fmt.Printf("%T(%s -> %s)\n", c, c.From.Name, c.To.Name)
		case *schema.AddIndex:
			fmt.Printf("%T(%s)\n", c, c.I.Name)
		}
	}
	// Output:
	// *schema.AddIndex(id)
	// *schema.AddColumn(id)
	// *schema.AddColumn(created_at)
	// *schema.RenameColumn(old_name -> new_name)
}
