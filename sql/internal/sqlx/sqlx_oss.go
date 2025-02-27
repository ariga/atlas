// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package sqlx

import (
	"slices"

	"ariga.io/atlas/sql/schema"
)

// sortViewChanges is an optional function to sort to views by their dependencies.
func sortViewChanges(changes []schema.Change) ([]schema.Change, error) {
	return changes, nil // unimplemented.
}

func (*Diff) triggerDiff(_, _ interface {
	Trigger(string) (*schema.Trigger, bool)
}, _, _ []*schema.Trigger, _ *schema.DiffOptions) ([]schema.Change, error) {
	return nil, nil // unimplemented.
}

// funcDep returns true if f1 depends on f2.
func funcDep(_, _ *schema.Func, _ SortOptions) bool {
	return false // unimplemented.
}

// procDep returns true if p1 depends on p2.
func procDep(_, _ *schema.Proc, _ SortOptions) bool {
	return false // unimplemented.
}

func tableDepFunc(*schema.Table, *schema.Func, SortOptions) bool {
	return false // unimplemented.
}

func (*Diff) askForColumns(_ *schema.Table, changes []schema.Change, _ *schema.DiffOptions) ([]schema.Change, error) {
	return changes, nil // unimplemented.
}

func (*Diff) askForIndexes(_ string, changes []schema.Change, _ *schema.DiffOptions) ([]schema.Change, error) {
	return changes, nil // unimplemented.
}

func (*Diff) fixRenames(changes schema.Changes) schema.Changes {
	return changes // unimplemented.
}

// dependsOn reports if the given change depends on the other change.
func dependsOn(c1, c2 schema.Change, _ SortOptions) bool {
	if dependOnOf(c1, c2) {
		return true
	}
	switch c1 := c1.(type) {
	case *schema.DropSchema:
		switch c2 := c2.(type) {
		case *schema.DropTable:
			// Schema must be dropped after all its tables and references to them.
			return SameSchema(c1.S, c2.T.Schema) || slices.ContainsFunc(c2.T.ForeignKeys, func(fk *schema.ForeignKey) bool {
				return SameSchema(c1.S, fk.RefTable.Schema)
			})
		case *schema.ModifyTable:
			return SameSchema(c1.S, c2.T.Schema) || slices.ContainsFunc(c2.Changes, func(c schema.Change) bool {
				fk, ok := c.(*schema.DropForeignKey)
				return ok && SameSchema(c1.S, fk.F.RefTable.Schema)
			})
		}
	case *schema.AddTable:
		switch c2 := c2.(type) {
		case *schema.AddSchema:
			return c1.T.Schema.Name == c2.S.Name
		case *schema.DropTable:
			// Table recreation.
			return c1.T.Name == c2.T.Name && SameSchema(c1.T.Schema, c2.T.Schema)
		case *schema.AddTable:
			if refTo(c1.T.ForeignKeys, c2.T) {
				return true
			}
			if slices.ContainsFunc(c1.T.Columns, func(c *schema.Column) bool {
				return c.Type != nil && typeDependsOnT(c.Type.Type, c2.T)
			}) {
				return true
			}
		case *schema.ModifyTable:
			if (c1.T.Name != c2.T.Name || !SameSchema(c1.T.Schema, c2.T.Schema)) && refTo(c1.T.ForeignKeys, c2.T) {
				return true
			}
		case *schema.AddObject:
			t, ok := c2.O.(schema.Type)
			if ok && slices.ContainsFunc(c1.T.Columns, func(c *schema.Column) bool {
				return schema.IsType(c.Type.Type, t)
			}) {
				return true
			}
		}
		return depOfAdd(c1.T.Deps, c2)
	case *schema.DropTable:
		// If it is a drop of a table, the change must occur
		// after all resources that rely on it will be dropped.
		switch c2 := c2.(type) {
		case *schema.DropTable:
			// References to this table, must be dropped first.
			if refTo(c2.T.ForeignKeys, c1.T) {
				return true
			}
			if slices.ContainsFunc(c2.T.Columns, func(c *schema.Column) bool {
				return c.Type != nil && typeDependsOnT(c.Type.Type, c1.T)
			}) {
				return true
			}
		case *schema.ModifyTable:
			if slices.ContainsFunc(c2.Changes, func(c schema.Change) bool {
				switch c := c.(type) {
				case *schema.DropForeignKey:
					return refTo([]*schema.ForeignKey{c.F}, c1.T)
				case *schema.DropColumn:
					return c.C.Type != nil && typeDependsOnT(c.C.Type.Type, c1.T)
				}
				return false
			}) {
				return true
			}
		}
		return depOfDrop(c1.T, c2)
	case *schema.ModifyTable:
		switch c2 := c2.(type) {
		case *schema.AddTable:
			// Table modification relies on its creation.
			if c1.T.Name == c2.T.Name && SameSchema(c1.T.Schema, c2.T.Schema) {
				return true
			}
			// Tables need to be created before referencing them.
			if slices.ContainsFunc(c1.Changes, func(c schema.Change) bool {
				switch c := c.(type) {
				case *schema.AddForeignKey:
					return refTo([]*schema.ForeignKey{c.F}, c2.T)
				case *schema.AddColumn:
					return c.C.Type != nil && typeDependsOnT(c.C.Type.Type, c2.T)
				case *schema.ModifyColumn:
					return c.To.Type != nil && typeDependsOnT(c.To.Type.Type, c2.T)
				}
				return false
			}) {
				return true
			}
		case *schema.ModifyTable:
			if c1.T != c2.T {
				addC := make(map[*schema.Column]bool)
				for _, c := range c2.Changes {
					if add, ok := c.(*schema.AddColumn); ok {
						addC[add.C] = true
					}
				}
				return slices.ContainsFunc(c1.Changes, func(c schema.Change) bool {
					fk, ok := c.(*schema.AddForeignKey)
					return ok && refTo([]*schema.ForeignKey{fk.F}, c2.T) && slices.ContainsFunc(fk.F.Columns, func(c *schema.Column) bool { return addC[c] })
				})
			}
		case *schema.AddObject:
			t, ok := c2.O.(schema.Type)
			if ok && slices.ContainsFunc(c1.Changes, func(c schema.Change) bool {
				switch c := c.(type) {
				case *schema.AddColumn:
					return schema.IsType(c.C.Type.Type, t)
				case *schema.ModifyColumn:
					return schema.IsType(c.To.Type.Type, t)
				default:
					return false
				}
			}) {
				return true
			}
		}
		return depOfAdd(c1.T.Deps, c2)
	case *schema.DropObject:
		t, ok := c1.O.(schema.Type)
		if !ok {
			return false
		}
		// Dropping a type must occur after all its usage were dropped.
		switch c2 := c2.(type) {
		case *schema.DropTable:
			// Dropping a table also drops its triggers and might depend on the type.
			if slices.ContainsFunc(c2.T.Triggers, func(tg *schema.Trigger) bool {
				return slices.Contains(tg.Deps, c1.O)
			}) {
				return true
			}
			if slices.ContainsFunc(c2.T.Columns, func(c *schema.Column) bool {
				return schema.IsType(c.Type.Type, t)
			}) {
				return true
			}
		case *schema.ModifyTable:
			return slices.ContainsFunc(c2.Changes, func(c schema.Change) bool {
				d, ok := c.(*schema.DropColumn)
				return ok && schema.IsType(d.C.Type.Type, t)
			})
		}
	}
	return false
}
