package sqlx

import (
	"sort"

	"ariga.io/atlas/sql/schema"
)

// DetachCycles takes a list of schema changes, and postpone
// the foreign keys creation if there's at least one circular
// reference in the changeset.
func DetachCycles(changes []schema.Change) []schema.Change {
	sorted, hasCycle := sortMap(changes)
	if hasCycle {
		return detachReferences(changes)
	}
	planned := make([]schema.Change, len(changes))
	copy(planned, changes)
	sort.Slice(planned, func(i, j int) bool {
		return sorted[table(planned[i])] < sorted[table(planned[j])]
	})
	return planned
}

// detachReferences detaches all table references.
func detachReferences(changes []schema.Change) []schema.Change {
	var planned, deferred []schema.Change
	for _, change := range changes {
		switch change := change.(type) {
		case *schema.AddTable:
			var fks []schema.Change
			for _, fk := range change.T.ForeignKeys {
				if fk.RefTable != change.T {
					fks = append(fks, &schema.AddForeignKey{F: fk})
				}
			}
			if len(fks) > 0 {
				deferred = append(deferred, &schema.ModifyTable{T: change.T, Changes: fks})
				t := *change.T
				t.ForeignKeys = nil
				change = &schema.AddTable{T: &t, Extra: change.Extra}
			}
			planned = append(planned, change)
		case *schema.ModifyTable:
			var fks, rest []schema.Change
			for _, c := range change.Changes {
				switch c := c.(type) {
				case *schema.AddForeignKey:
					fks = append(fks, c)
				default:
					rest = append(rest, c)
				}
			}
			if len(fks) > 0 {
				deferred = append(deferred, &schema.ModifyTable{T: change.T, Changes: fks})
			}
			if len(rest) > 0 {
				planned = append(planned, &schema.ModifyTable{T: change.T, Changes: rest})
			}
		default:
			planned = append(planned, change)
		}
	}
	return append(planned, deferred...)
}

// sortMap returns an index-map indicates the position of table in a topological
// sort in reversed order based on its references, and a boolean indicate if there
// is a non-self loop.
func sortMap(changes []schema.Change) (map[*schema.Table]int, bool) {
	var (
		visit      func(*schema.Table) bool
		references = references(changes)
		sorted     = make(map[*schema.Table]int)
		progress   = make(map[*schema.Table]bool)
	)
	visit = func(node *schema.Table) bool {
		if _, done := sorted[node]; done {
			return false
		}
		if progress[node] {
			return true
		}
		progress[node] = true
		for _, ref := range references[node] {
			if visit(ref) {
				return true
			}
		}
		delete(progress, node)
		sorted[node] = len(sorted)
		return false
	}
	for node := range references {
		if visit(node) {
			return nil, true
		}
	}
	return sorted, false
}

// references returns an adjacency list of all child tables and their references in parent tables.
func references(changes []schema.Change) map[*schema.Table][]*schema.Table {
	refs := make(map[*schema.Table][]*schema.Table)
	for _, change := range changes {
		switch change := change.(type) {
		case *schema.AddTable:
			for _, fk := range change.T.ForeignKeys {
				if fk.RefTable != change.T {
					refs[change.T] = append(refs[change.T], fk.RefTable)
				}
			}
		case *schema.ModifyTable:
			for _, c := range change.Changes {
				switch c := c.(type) {
				case *schema.AddForeignKey:
					if c.F.RefTable != change.T {
						refs[change.T] = append(refs[change.T], c.F.RefTable)
					}
				case *schema.ModifyForeignKey:
					if c.To.RefTable != change.T {
						refs[change.T] = append(refs[change.T], c.To.RefTable)
					}
				}
			}
		}
	}
	return refs
}

// table extracts a table from the given change.
func table(change schema.Change) (t *schema.Table) {
	switch change := change.(type) {
	case *schema.AddTable:
		t = change.T
	case *schema.ModifyTable:
		t = change.T
	}
	return
}
