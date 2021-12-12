package action

import (
	"context"
	"testing"

	"ariga.io/atlas/schema/schemaspec/schemahcl"
	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/schema"
	"github.com/stretchr/testify/require"
)

func TestDiffPrint(t *testing.T) {

	beforeHcl := `
schema "default" {
}
table "users" {
	schema = schema.default
	column "id" {
		type = "int"
	}
}
`
	afterHcl := `
schema "default" {
}
table "users" {
	schema = schema.default
	column "id" {
		type = "string"
	}
	column "name" {
		type = "string"
	}
}

table "groups" {
	schema = schema.default
	column "id" {
		type = "int"
	}
}
`
	ctx := context.Background()
	d := dummyDriver()
	changes := diff(t, d, beforeHcl, afterHcl)
	var changeDescs []*changeDesc
	for _, ch := range changes {
		descriptor, err := changeDescriptor(ctx, ch, d)
		require.NoError(t, err)
		changeDescs = append(changeDescs, descriptor)
	}
	require.EqualValues(t, []*changeDesc{
		{
			typ:     "Modify Table",
			subject: "users",
			queries: []string{
				"ALTER TABLE `default`.`users` MODIFY COLUMN `id` varchar(255) NOT NULL, ADD COLUMN `name` varchar(255) NOT NULL",
			},
		},
		{
			typ:     "Add Table",
			subject: "groups",
			queries: []string{
				"CREATE TABLE `default`.`groups` (`id` int NOT NULL)",
			},
		},
	}, changeDescs)
}

func diff(t *testing.T, d *Driver, beforeHcl, afterHcl string) []schema.Change {
	before, after := schema.Schema{}, schema.Schema{}
	err := mysql.UnmarshalSpec([]byte(beforeHcl), schemahcl.Unmarshal, &before)
	require.NoError(t, err)
	err = mysql.UnmarshalSpec([]byte(afterHcl), schemahcl.Unmarshal, &after)
	require.NoError(t, err)
	before.Realm = &schema.Realm{}
	diff, err := d.SchemaDiff(&before, &after)
	require.NoError(t, err)
	return diff
}

func dummyDriver() *Driver {
	i := &interceptor{}
	drv := &mysql.Driver{ExecQuerier: i}
	return &Driver{
		Execer:      drv.Migrate(),
		Differ:      drv.Diff(),
		interceptor: i,
	}
}
