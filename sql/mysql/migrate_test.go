package mysql

import (
	"context"
	"testing"

	"ariga.io/atlas/sql/internal/sqltest"
	"ariga.io/atlas/sql/schema"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestMigrate_Exec(t *testing.T) {
	migrate, mk, err := newMigrate("8.0.13")
	require.NoError(t, err)
	mk.ExpectExec(sqltest.Escape("DROP TABLE `users`")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mk.ExpectExec(sqltest.Escape("DROP TABLE `public`.`pets`")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mk.ExpectExec(sqltest.Escape("CREATE TABLE `pets` (`a` int NOT NULL, `b` bigint NOT NULL, `c` bigint NULL, PRIMARY KEY (`a`, `b`), UNIQUE INDEX `b_c_unique` (`b`, `c`) COMMENT 'comment')")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mk.ExpectExec(sqltest.Escape("ALTER TABLE `users` DROP INDEX `id_spouse_id`")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mk.ExpectExec(sqltest.Escape("ALTER TABLE `users` ADD CONSTRAINT `spouse` FOREIGN KEY (`spouse_id`) REFERENCES `users` (`id`) ON DELETE SET NULL, ADD INDEX `id_spouse_id` (`spouse_id`, `id` DESC) COMMENT 'comment'")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mk.ExpectExec(sqltest.Escape("CREATE TABLE `posts` (`id` bigint NOT NULL, `author_id` bigint NULL, CONSTRAINT `author` FOREIGN KEY (`author_id`) REFERENCES `users` (`id`))")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mk.ExpectExec(sqltest.Escape("CREATE TABLE `comments` (`id` bigint NOT NULL, `post_id` bigint NULL, CONSTRAINT `comment` FOREIGN KEY (`post_id`) REFERENCES `posts` (`id`))")).
		WillReturnResult(sqlmock.NewResult(0, 0))
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
					{Name: "b_c_unique", Unique: true, Parts: []*schema.IndexPart{{C: t.Columns[1]}, {C: t.Columns[2]}}, Attrs: []schema.Attr{&schema.Comment{Text: "comment"}}},
				}
				return t
			}(),
		},
	})
	require.NoError(t, err)
	err = migrate.Exec(context.Background(), func() []schema.Change {
		users := &schema.Table{
			Name: "users",
			Columns: []*schema.Column{
				{Name: "id", Type: &schema.ColumnType{Raw: "bigint"}},
				{Name: "spouse_id", Type: &schema.ColumnType{Raw: "bigint", Null: true}},
			},
		}
		posts := &schema.Table{
			Name: "posts",
			Columns: []*schema.Column{
				{Name: "id", Type: &schema.ColumnType{Raw: "bigint"}},
				{Name: "author_id", Type: &schema.ColumnType{Raw: "bigint", Null: true}},
			},
		}
		posts.ForeignKeys = []*schema.ForeignKey{
			{Symbol: "author", Table: posts, Columns: posts.Columns[1:], RefTable: users, RefColumns: users.Columns[:1]},
		}
		comments := &schema.Table{
			Name: "comments",
			Columns: []*schema.Column{
				{Name: "id", Type: &schema.ColumnType{Raw: "bigint"}},
				{Name: "post_id", Type: &schema.ColumnType{Raw: "bigint", Null: true}},
			},
		}
		comments.ForeignKeys = []*schema.ForeignKey{
			{Symbol: "comment", Table: comments, Columns: comments.Columns[1:], RefTable: posts, RefColumns: posts.Columns[:1]},
		}
		return []schema.Change{
			&schema.AddTable{T: posts},
			&schema.AddTable{T: comments},
			&schema.ModifyTable{
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
						To: &schema.Index{
							Name: "id_spouse_id",
							Parts: []*schema.IndexPart{
								{C: users.Columns[1]},
								{C: users.Columns[0], Attrs: []schema.Attr{&schema.Collation{V: "D"}}},
							},
							Attrs: []schema.Attr{
								&schema.Comment{Text: "comment"},
							},
						},
					},
				},
			},
		}
	}())
	require.NoError(t, err)
}

func TestMigrate_DetachCycles(t *testing.T) {
	migrate, mk, err := newMigrate("8.0.13")
	require.NoError(t, err)
	mk.ExpectExec(sqltest.Escape("CREATE TABLE `users` (`id` bigint NOT NULL, `workplace_id` bigint NULL)")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mk.ExpectExec(sqltest.Escape("CREATE TABLE `workplaces` (`id` bigint NOT NULL, `owner_id` bigint NULL)")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mk.ExpectExec(sqltest.Escape("ALTER TABLE `users` ADD CONSTRAINT `workplace` FOREIGN KEY (`workplace_id`) REFERENCES `workplaces` (`id`)")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mk.ExpectExec(sqltest.Escape("ALTER TABLE `workplaces` ADD CONSTRAINT `owner` FOREIGN KEY (`owner_id`) REFERENCES `users` (`id`)")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	err = migrate.Exec(context.Background(), func() []schema.Change {
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
			{Symbol: "workplace", Table: users, Columns: users.Columns[1:], RefTable: workplaces, RefColumns: workplaces.Columns[:1]},
		}
		workplaces.ForeignKeys = []*schema.ForeignKey{
			{Symbol: "owner", Table: workplaces, Columns: workplaces.Columns[1:], RefTable: users, RefColumns: users.Columns[:1]},
		}
		return []schema.Change{
			&schema.AddTable{T: users},
			&schema.AddTable{T: workplaces},
		}
	}())
	require.NoError(t, err)
}

func newMigrate(version string) (schema.Execer, *mock, error) {
	db, m, err := sqlmock.New()
	if err != nil {
		return nil, nil, err
	}
	mk := &mock{m}
	mk.version(version)
	drv, err := Open(db)
	if err != nil {
		return nil, nil, err
	}
	return drv.Migrate(), mk, nil
}
