package schemaspec_test

import (
	"testing"

	"ariga.io/atlas/sql/internal/schemautil"
	"ariga.io/atlas/sql/schema/schemaspec"
	"github.com/stretchr/testify/require"
)

type OwnerBlock struct {
	FirstName string `spec:"first_name"`
	Born      int    `spec:"born"`
	Active    bool   `spec:"active"`
}

func (*OwnerBlock) Type() string {
	return "owner"
}

func (*OwnerBlock) Name() string {
	return ""
}

func TestExtension(t *testing.T) {
	r := &schemaspec.Resource{
		Type: "owner",
		Attrs: []*schemaspec.Attr{
			schemautil.StrLitAttr("first_name", "tzuri"),
			schemautil.LitAttr("born", "2019"),
			schemautil.LitAttr("active", "true"),
		},
	}
	owner := OwnerBlock{}
	err := r.Scan(&owner)
	require.NoError(t, err)
	require.EqualValues(t, "tzuri", owner.FirstName)
	require.EqualValues(t, 2019, owner.Born)
	require.EqualValues(t, true, owner.Active)

	resource := schemaspec.ExtAsResource(&owner)
	require.EqualValues(t, r, resource)
}
