// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package schemahcl_test

import (
	"testing"

	"ariga.io/atlas/schemahcl"

	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

type OwnerBlock struct {
	schemahcl.DefaultExtension
	ID        string    `spec:",name"`
	FirstName string    `spec:"first_name"`
	Born      int       `spec:"born"`
	Active    bool      `spec:"active"`
	BoolPtr   *bool     `spec:"bool_ptr"`
	OmitBool1 bool      `spec:"omit_bool1,omitempty"`
	OmitBool2 bool      `spec:"omit_bool2,omitempty"`
	Lit       cty.Value `spec:"lit"`
}

type PetBlock struct {
	schemahcl.DefaultExtension
	ID        string        `spec:",name"`
	Breed     string        `spec:"breed"`
	Born      int           `spec:"born"`
	Owners    []*OwnerBlock `spec:"owner"`
	RoleModel *PetBlock     `spec:"role_model"`
}

func TestInvalidExt(t *testing.T) {
	r := &schemahcl.Resource{}
	err := r.As(1)
	require.EqualError(t, err, "schemahcl: expected target to be a pointer")
	var p *string
	err = r.As(p)
	require.EqualError(t, err, "schemahcl: expected target to be a pointer to a struct")
}

func TestExtension(t *testing.T) {
	schemahcl.Register("owner", &OwnerBlock{})
	original := &schemahcl.Resource{
		Name: "name",
		Type: "owner",
		Attrs: []*schemahcl.Attr{
			schemahcl.StringAttr("first_name", "tzuri"),
			schemahcl.IntAttr("born", 2019),
			schemahcl.BoolAttr("active", true),
			schemahcl.BoolAttr("bool_ptr", true),
			schemahcl.BoolAttr("omit_bool1", true),
			schemahcl.StringAttr("lit", "1000"),
			schemahcl.BoolAttr("extra", true),
		},
		Children: []*schemahcl.Resource{
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
	require.NotNil(t, owner.BoolPtr)
	require.EqualValues(t, true, *owner.BoolPtr)
	require.EqualValues(t, cty.StringVal("1000"), owner.Lit)
	attr, ok := owner.Remain().Attr("extra")
	require.True(t, ok)
	eb, err := attr.Bool()
	require.NoError(t, err)
	require.True(t, eb)

	scan := &schemahcl.Resource{}
	err = scan.Scan(&owner)
	require.NoError(t, err)
	require.EqualValues(t, original, scan)
}

func TestNested(t *testing.T) {
	schemahcl.Register("pet", &PetBlock{})
	pet := &schemahcl.Resource{
		Name: "donut",
		Type: "pet",
		Attrs: []*schemahcl.Attr{
			schemahcl.StringAttr("breed", "golden retriever"),
			schemahcl.IntAttr("born", 2002),
		},
		Children: []*schemahcl.Resource{
			{
				Name: "rotemtam",
				Type: "owner",
				Attrs: []*schemahcl.Attr{
					schemahcl.StringAttr("first_name", "rotem"),
					schemahcl.IntAttr("born", 1985),
					schemahcl.BoolAttr("active", true),
				},
			},
			{
				Name: "gonnie",
				Type: "role_model",
				Attrs: []*schemahcl.Attr{
					schemahcl.StringAttr("breed", "golden retriever"),
					schemahcl.IntAttr("born", 1998),
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
			ID:    "gonnie",
			Breed: "golden retriever",
			Born:  1998,
		},
	}, pb)
	scan := &schemahcl.Resource{}
	err = scan.Scan(&pb)
	require.NoError(t, err)
	require.EqualValues(t, pet, scan)
	name, err := pet.FinalName()
	require.NoError(t, err)
	require.EqualValues(t, "donut", name)
}

func TestRef(t *testing.T) {
	type A struct {
		Name string         `spec:",name"`
		User *schemahcl.Ref `spec:"user"`
	}
	schemahcl.Register("a", &A{})
	resource := &schemahcl.Resource{
		Name: "x",
		Type: "a",
		Attrs: []*schemahcl.Attr{
			schemahcl.RefAttr("user", &schemahcl.Ref{V: "$user.rotemtam"}),
		},
	}
	var a A
	err := resource.As(&a)
	require.NoError(t, err)
	require.EqualValues(t, &schemahcl.Ref{V: "$user.rotemtam"}, a.User)
	scan := &schemahcl.Resource{}
	err = scan.Scan(&a)
	require.NoError(t, err)
	require.EqualValues(t, resource, scan)
}

func TestListRef(t *testing.T) {
	type B struct {
		Name  string           `spec:",name"`
		Users []*schemahcl.Ref `spec:"users"`
	}
	schemahcl.Register("b", &B{})
	resource := &schemahcl.Resource{
		Name: "x",
		Type: "b",
		Attrs: []*schemahcl.Attr{
			schemahcl.RefsAttr(
				"users",
				&schemahcl.Ref{V: "$user.a8m"},
				&schemahcl.Ref{V: "$user.rotemtam"},
			),
		},
	}

	var b B
	err := resource.As(&b)
	require.NoError(t, err)
	require.Len(t, b.Users, 2)
	require.EqualValues(t, []*schemahcl.Ref{
		{V: "$user.a8m"},
		{V: "$user.rotemtam"},
	}, b.Users)
	scan := &schemahcl.Resource{}
	err = scan.Scan(&b)
	require.NoError(t, err)
	require.EqualValues(t, resource, scan)
}

func TestNameAttr(t *testing.T) {
	type Named struct {
		Name string `spec:"name,name"`
	}
	schemahcl.Register("named", &Named{})
	resource := &schemahcl.Resource{
		Name: "id",
		Type: "named",
		Attrs: []*schemahcl.Attr{
			schemahcl.StringAttr("name", "atlas"),
		},
	}
	var tgt Named
	err := resource.As(&tgt)
	require.NoError(t, err)
	require.EqualValues(t, "atlas", tgt.Name)
	name, err := resource.FinalName()
	require.NoError(t, err)
	require.EqualValues(t, name, "atlas")
}
