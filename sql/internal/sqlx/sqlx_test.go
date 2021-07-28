package sqlx

import (
	"testing"

	"ariga.io/atlas/sql/schema"

	"github.com/stretchr/testify/require"
)

func TestVersionNames(t *testing.T) {
	names := VersionPermutations("mysql", "1.2.3")
	require.EqualValues(t, []string{"mysql", "mysql 1", "mysql 1.2", "mysql 1.2.3"}, names)

	names = VersionPermutations("postgres", "11.3 nightly")
	require.EqualValues(t, []string{"postgres", "postgres 11", "postgres 11.3", "postgres 11.3.nightly"}, names)
}

func TestBuilder(t *testing.T) {
	var (
		b       = &Builder{QuoteChar: '"'}
		columns = []string{"a", "b", "c"}
	)
	b.P("CREATE TABLE").
		Table(&schema.Table{Name: "users"}).
		Wrap(func(b *Builder) {
			b.MapComma(columns, func(i int, b *Builder) {
				b.Ident(columns[i]).P("int").P("NOT NULL")
			})
			b.Comma().P("PRIMARY KEY").Wrap(func(b *Builder) {
				b.MapComma(columns, func(i int, b *Builder) {
					b.Ident(columns[i])
				})
			})
		})
	require.Equal(t, `CREATE TABLE "users" ("a" int NOT NULL, "b" int NOT NULL, "c" int NOT NULL, PRIMARY KEY ("a", "b", "c"))`, b.String())
}
