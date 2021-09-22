package schemaspec_test

import (
	"fmt"
	"strconv"
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
	Favorite  *Food         `spec:"favorite_food"`
	Allergies []*Food       `spec:"allergy"`
	Owners    []*OwnerBlock `spec:"owner"`
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
			schemautil.StrLitAttr("favorite_food", "pasta"),
			schemautil.ListAttr("allergy", "peanut", "chocolate"),
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
		Favorite: &Food{
			Name: "pasta",
		},
		Allergies: []*Food{
			{Name: "peanut"},
			{Name: "chocolate"},
		},
		Born: 2002,
		Owners: []*OwnerBlock{
			{ID: "rotemtam", FirstName: "rotem", Born: 1985, Active: true},
		},
	}, pb)
	scan := &schemaspec.Resource{}
	err = scan.Scan(&pb)
	require.NoError(t, err)
	require.EqualValues(t, pet, scan)
}

type Food struct {
	Name string
}

func (f *Food) Scan(value schemaspec.Value) error {
	v, ok := value.(*schemaspec.LiteralValue)
	if !ok {
		return fmt.Errorf("expected value to be literal")
	}
	s, err := strconv.Unquote(v.V)
	if err != nil {
		return fmt.Errorf("expected attr %q to be convertible to string", v.V)
	}
	f.Name = s
	return nil
}

func (f *Food) Value() schemaspec.Value {
	return &schemaspec.LiteralValue{V: strconv.Quote(f.Name)}
}
