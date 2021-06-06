package mysql

import (
	"context"
	"testing"

	"ariga.io/atlas/sql/schema"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestMigrate_Exec(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	mk := mock{m}
	mk.version("8.0.13")
	mk.ExpectExec(escape("DROP TABLE `users`")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mk.ExpectExec(escape("DROP TABLE `public`.`pets`")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mk.ExpectExec(escape("CREATE TABLE `pets` (`a` int NOT NULL, `b` bigint NOT NULL, `c` bigint NULL, PRIMARY KEY (`a`, `b`), UNIQUE INDEX `b_c_unique` (`b`, `c`))")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mk.ExpectExec(escape("ALTER TABLE `users` DROP INDEX `id_spouse_id`")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mk.ExpectExec(escape("ALTER TABLE `users` ADD CONSTRAINT `spouse` FOREIGN KEY (`spouse_id`) REFERENCES `users` (`id`) ON DELETE SET NULL, ADD INDEX `id_spouse_id` (`spouse_id`, `id`)")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	drv, err := Open(db)
	require.NoError(t, err)
	migrate := Migrate{Driver: drv}
	err = migrate.Exec(context.Background(), []schema.Change{
		&schema.DropTable{T: &schema.Table{Name: "users"}},
		&schema.DropTable{T: &schema.Table{Name: "pets", Schema: &schema.Schema{Name: "public"}}},
		&schema.AddTable{
			T: func() *schema.Table {
				t := &schema.Table{
					Name: "pets",
					Columns: []*schema.Column{
						{Name: "a", Type: &schema.ColumnType{Raw: "int"}},
						{Name: "b", Type: &schema.ColumnType{Raw: "bigint"}},
						{Name: "c", Type: &schema.ColumnType{Raw: "bigint", Null: true}},
					},
				}
				t.PrimaryKey = &schema.Index{
					Parts: []*schema.IndexPart{{C: t.Columns[0]}, {C: t.Columns[1]}},
				}
				t.Indexes = []*schema.Index{
					{Name: "b_c_unique", Unique: true, Parts: []*schema.IndexPart{{C: t.Columns[1]}, {C: t.Columns[2]}}},
				}
				return t
			}(),
		},
	})
	require.NoError(t, err)
	err = migrate.Exec(context.Background(), []schema.Change{
		func() schema.Change {
			users := &schema.Table{
				Name: "users",
				Columns: []*schema.Column{
					{Name: "id", Type: &schema.ColumnType{Raw: "bigint"}},
					{Name: "spouse_id", Type: &schema.ColumnType{Raw: "bigint", Null: true}},
				},
			}
			return &schema.ModifyTable{
				T: users,
				Changes: []schema.Change{
					&schema.AddForeignKey{
						F: &schema.ForeignKey{
							Symbol:     "spouse",
							Table:      users,
							Columns:    users.Columns[1:],
							RefTable:   users,
							RefColumns: users.Columns[:1],
							OnDelete:   "SET NULL",
						},
					},
					&schema.ModifyIndex{
						From: &schema.Index{Name: "id_spouse_id", Parts: []*schema.IndexPart{{C: users.Columns[0]}, {C: users.Columns[1]}}},
						To:   &schema.Index{Name: "id_spouse_id", Parts: []*schema.IndexPart{{C: users.Columns[1]}, {C: users.Columns[0]}}},
					},
				},
			}
		}(),
	})
	require.NoError(t, err)
}
