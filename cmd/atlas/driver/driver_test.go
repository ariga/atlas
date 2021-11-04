package driver

import (
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestDriverFail(t *testing.T) {
	_, err := NewAtlasDriver("bad utl")
	require.Error(t, err)
}

func TestDriverNoDB(t *testing.T) {
	_,  err := NewAtlasDriver("root@tcp(localhost:3306)/todo")
	require.Error(t, err)
}

func TestDriverConnect(t *testing.T) {
	d, err := NewAtlasDriver("sqlite3://file:ent?mode=memory&cache=shared&_fk=1")
	require.NoError(t, err)
	d.Close()
}
