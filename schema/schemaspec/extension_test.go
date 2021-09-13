package schemaspec_test

import (
	"testing"

	"ariga.io/atlas/schema/schemaspec"
	"ariga.io/atlas/schema/schemaspec/internal/schemautil"

	"github.com/stretchr/testify/require"
)

type OwnerBlock struct {
	ID        string `spec:",name"`
	FirstName string `spec:"first_name"`
	Born      int    `spec:"born"`
	Active    bool   `spec:"active"`
}

func (*OwnerBlock) Type() string {
	return "owner"
}

type PetBlock struct {
	ID     string        `spec:",name"`
	Breed  string        `spec:"breed"`
	Born   int           `spec:"born"`
	Owners []*OwnerBlock `spec:"owner"`
}

func (*PetBlock) Type() string {
	return "pet"
}

func TestExtension(t *testing.T) {
	original := &schemaspec.Resource{
		Name: "name",
		Type: "owner",
		Attrs: []*schemaspec.Attr{
			schemautil.StrLitAttr("first_name", "tzuri"),
			schemautil.LitAttr("born", "2019"),
			schemautil.LitAttr("active", "true"),
		},
	}
	owner := OwnerBlock{}
	err := original.As(&owner)
	require.NoError(t, err)
	require.EqualValues(t, "tzuri", owner.FirstName)
	require.EqualValues(t, "name", owner.ID)
	require.EqualValues(t, 2019, owner.Born)
	require.EqualValues(t, true, owner.Active)

	scan := &schemaspec.Resource{}
	err = scan.Scan(&owner)
	require.NoError(t, err)
	require.EqualValues(t, original, scan)
}

func TestNested(t *testing.T) {
	pet := &schemaspec.Resource{
		Name: "donut",
		Type: "pet",
		Attrs: []*schemaspec.Attr{
			schemautil.StrLitAttr("breed", "golden retriever"),
			schemautil.LitAttr("born", "2002"),
		},
		Children: []*schemaspec.Resource{
			{
				Name: "rotemtam",
				Type: "owner",
				Attrs: []*schemaspec.Attr{
					schemautil.StrLitAttr("first_name", "rotem"),
					schemautil.LitAttr("born", "1985"),
					schemautil.LitAttr("active", "true"),
				},
			},
		},
	}
	pb := PetBlock{}
	err := pet.As(&pb)
	require.NoError(t, err)
	require.EqualValues(t, PetBlock{
		ID:    "donut",
		Breed: "golden retriever",
		Born:  2002,
		Owners: []*OwnerBlock{
			{ID: "rotemtam", FirstName: "rotem", Born: 1985, Active: true},
		},
	}, pb)
	scan := &schemaspec.Resource{}
	err = scan.Scan(&pb)
	require.NoError(t, err)
	require.EqualValues(t, pet, scan)
}
