package client

import (
	"testing"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestDriverFail(t *testing.T) {
	_, _, err := NewAtlasDriver("bad utl")
	require.Error(t, err)
}

func TestDriverNoDB(t *testing.T) {
	_, _, err := NewAtlasDriver("root@tcp(localhost:3306)/todo")
	require.Error(t, err)
}

func TestDriverConnect(t *testing.T) {
	_, close, err := NewAtlasDriver("sqlite3://file:ent?mode=memory&cache=shared&_fk=1")
	require.NoError(t, err)
	close()
}
