package action

import (
	"context"
	"testing"

	"ariga.io/atlas/schema/schemaspec/schemahcl"
	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/schema"

	"github.com/DATA-DOG/go-sqlmock"
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
	d := mockDriver(t)
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

func mockDriver(t *testing.T) *Driver {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	m.ExpectQuery(".*").
		WillReturnRows(sqlmock.NewRows([]string{"1", "2", "3"}).AddRow("8.0.19", "utf8_general_ci", "utf8"))
	i := &interceptor{ExecQuerier: db}
	drv, err := mysql.Open(i)
	require.NoError(t, err)
	return &Driver{
		Driver:      drv,
		interceptor: i,
	}
}
