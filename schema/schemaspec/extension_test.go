package schemaspec_test

import (
	"testing"

	"ariga.io/atlas/schema/schemaspec"
	"ariga.io/atlas/schema/schemaspec/internal/schemautil"

	"github.com/stretchr/testify/require"
)

type OwnerBlock struct {
	schemaspec.DefaultExtension
	ID        string                   `spec:",name"`
	FirstName string                   `spec:"first_name"`
	Born      int                      `spec:"born"`
	Active    bool                     `spec:"active"`
	Lit       *schemaspec.LiteralValue `spec:"lit"`
}

type PetBlock struct {
	schemaspec.DefaultExtension
	ID        string        `spec:",name"`
	Breed     string        `spec:"breed"`
	Born      int           `spec:"born"`
	Owners    []*OwnerBlock `spec:"owner"`
	RoleModel *PetBlock     `spec:"role_model"`
}

func TestInvalidExt(t *testing.T) {
	r := &schemaspec.Resource{}
	err := r.As(1)
	require.EqualError(t, err, "schemaspec: expected target to be a pointer")
	var p *string
	err = r.As(p)
	require.EqualError(t, err, "schemaspec: expected target to be a pointer to a struct")
}

func TestExtension(t *testing.T) {
	schemaspec.Register("owner", &OwnerBlock{})
	original := &schemaspec.Resource{
		Name: "name",
		Type: "owner",
		Attrs: []*schemaspec.Attr{
			schemautil.StrLitAttr("first_name", "tzuri"),
			schemautil.LitAttr("born", "2019"),
			schemautil.LitAttr("active", "true"),
			schemautil.LitAttr("lit", "1000"),
			schemautil.LitAttr("extra", "true"),
		},
		Children: []*schemaspec.Resource{
			{
				Name: "extra",
				Type: "extra",
			},
		},
	}
	owner := OwnerBlock{}
	err := original.As(&owner)
	require.NoError(t, err)
	require.EqualValues(t, "tzuri", owner.FirstName)
	require.EqualValues(t, "name", owner.ID)
	require.EqualValues(t, 2019, owner.Born)
	require.EqualValues(t, true, owner.Active)
	require.EqualValues(t, schemautil.LitAttr("lit", "1000").V, owner.Lit)
	attr, ok := owner.Remain().Attr("extra")
	require.True(t, ok)
	eb, err := attr.Bool()
	require.NoError(t, err)
	require.True(t, eb)

	scan := &schemaspec.Resource{}
	err = scan.Scan(&owner)
	require.NoError(t, err)
	require.EqualValues(t, original, scan)
}

func TestNested(t *testing.T) {
	schemaspec.Register("pet", &PetBlock{})
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
			{
				Name: "gonnie",
				Type: "role_model",
				Attrs: []*schemaspec.Attr{
					schemautil.StrLitAttr("breed", "golden retriever"),
					schemautil.LitAttr("born", "1998"),
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
		RoleModel: &PetBlock{
			ID:     "gonnie",
			Breed:  "golden retriever",
			Born:   1998,
			Owners: []*OwnerBlock{},
		},
	}, pb)
	scan := &schemaspec.Resource{}
	err = scan.Scan(&pb)
	require.NoError(t, err)
	require.EqualValues(t, pet, scan)
}

func TestRef(t *testing.T) {
	type A struct {
		Name string          `spec:",name"`
		User *schemaspec.Ref `spec:"user"`
	}
	schemaspec.Register("res", &A{})
	resource := &schemaspec.Resource{
		Name: "x",
		Type: "res",
		Attrs: []*schemaspec.Attr{
			{
				K: "user",
				V: &schemaspec.Ref{V: "$user.rotemtam"},
			},
		},
	}
	tgt := A{}
	err := resource.As(&tgt)
	require.NoError(t, err)
	require.EqualValues(t, &schemaspec.Ref{V: "$user.rotemtam"}, tgt.User)
	scan := &schemaspec.Resource{}
	err = scan.Scan(&tgt)
	require.NoError(t, err)
	require.EqualValues(t, resource, scan)
}

func TestListRef(t *testing.T) {
	type A struct {
		Name  string            `spec:",name"`
		Users []*schemaspec.Ref `spec:"users"`
	}
	schemaspec.Register("res", &A{})
	resource := &schemaspec.Resource{
		Name: "x",
		Type: "res",
		Attrs: []*schemaspec.Attr{
			{
				K: "users",
				V: &schemaspec.ListValue{
					V: []schemaspec.Value{
						&schemaspec.Ref{V: "$user.a8m"},
						&schemaspec.Ref{V: "$user.rotemtam"},
					},
				},
			},
		},
	}

	tgt := A{}
	err := resource.As(&tgt)
	require.NoError(t, err)
	require.Len(t, tgt.Users, 2)
	require.EqualValues(t, []*schemaspec.Ref{
		{V: "$user.a8m"},
		{V: "$user.rotemtam"},
	}, tgt.Users)
	scan := &schemaspec.Resource{}
	err = scan.Scan(&tgt)
	require.NoError(t, err)
	require.EqualValues(t, resource, scan)
}
