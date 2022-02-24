// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mysql

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
)

// A planApply provides migration capabilities for schema elements.
type planApply struct{ conn }

// PlanChanges returns a migration plan for the given schema changes.
func (p *planApply) PlanChanges(_ context.Context, name string, changes []schema.Change) (*migrate.Plan, error) {
	s := &state{
		conn: p.conn,
		Plan: migrate.Plan{
			Name: name,
			// A plan is reversible, if all
			// its changes are reversible.
			Reversible: true,
			// All statements generated by state will cause implicit commit.
			// https://dev.mysql.com/doc/refman/8.0/en/implicit-commit.html
			Transactional: false,
		},
	}
	if err := s.plan(changes); err != nil {
		return nil, err
	}
	for _, c := range s.Changes {
		if c.Reverse == "" {
			s.Reversible = false
			break
		}
	}
	return &s.Plan, nil
}

// ApplyChanges applies the changes on the database. An error is returned
// if the driver is unable to produce a plan to it, or one of the statements
// is failed or unsupported.
func (p *planApply) ApplyChanges(ctx context.Context, changes []schema.Change) error {
	return sqlx.ApplyChanges(ctx, changes, p)
}

// state represents the state of a planning. It is not part of
// planApply so that multiple planning/applying can be called
// in parallel.
type state struct {
	conn
	migrate.Plan
}

// plan builds the migration plan for applying the
// given changes on the attached connection.
func (s *state) plan(changes []schema.Change) error {
	planned, err := s.topLevel(changes)
	if err != nil {
		return err
	}
	planned, err = sqlx.DetachCycles(planned)
	if err != nil {
		return err
	}
	for _, c := range planned {
		switch c := c.(type) {
		case *schema.AddTable:
			if err := s.addTable(c); err != nil {
				return err
			}
		case *schema.DropTable:
			s.dropTable(c)
		case *schema.ModifyTable:
			if err := s.modifyTable(c); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported change %T", c)
		}
	}
	return nil
}

// topLevel appends first the changes for creating or dropping schemas (top-level schema elements).
func (s *state) topLevel(changes []schema.Change) ([]schema.Change, error) {
	planned := make([]schema.Change, 0, len(changes))
	for _, c := range changes {
		switch c := c.(type) {
		case *schema.AddSchema:
			b := Build("CREATE DATABASE").Ident(c.S.Name)
			if sqlx.Has(c.Extra, &schema.IfNotExists{}) {
				b.P("IF NOT EXISTS")
			}
			// Schema was created with CHARSET and it is not the default database character set.
			if a := (schema.Charset{}); sqlx.Has(c.S.Attrs, &a) && a.V != "" && a.V != s.charset {
				b.P("CHARSET", a.V)
			}
			// Schema was created with COLLATE and it is not the default database collation.
			if a := (schema.Collation{}); sqlx.Has(c.S.Attrs, &a) && a.V != "" && a.V != s.collate {
				b.P("COLLATE", a.V)
			}
			s.append(&migrate.Change{
				Cmd:     b.String(),
				Source:  c,
				Reverse: Build("DROP DATABASE").Ident(c.S.Name).String(),
				Comment: fmt.Sprintf("add new schema named %q", c.S.Name),
			})
		case *schema.DropSchema:
			b := Build("DROP DATABASE").Ident(c.S.Name)
			if sqlx.Has(c.Extra, &schema.IfExists{}) {
				b.P("IF EXISTS")
			}
			s.append(&migrate.Change{
				Cmd:     b.String(),
				Source:  c,
				Comment: fmt.Sprintf("drop schema named %q", c.S.Name),
			})
		case *schema.ModifySchema:
			if err := s.modifySchema(c); err != nil {
				return nil, err
			}
		default:
			planned = append(planned, c)
		}
	}
	return planned, nil
}

// modifySchema builds and appends the migrate.Changes for bringing
// the schema into its modified state.
func (s *state) modifySchema(modify *schema.ModifySchema) error {
	b, r := Build(""), Build("")
	for _, change := range modify.Changes {
		switch change := change.(type) {
		// Add schema attributes to an existing schema only if
		// it is different from the default server configuration.
		case *schema.AddAttr:
			switch a := change.A.(type) {
			case *schema.Charset:
				if a.V != "" && a.V != s.charset {
					b.P("CHARSET", a.V)
					r.P("CHARSET", s.charset)
				}
			case *schema.Collation:
				if a.V != "" && a.V != s.collate {
					b.P("COLLATE", a.V)
					r.P("COLLATE", s.collate)
				}
			default:
				return fmt.Errorf("unexpected schema AddAttr: %T", a)
			}
		case *schema.ModifyAttr:
			switch to := change.To.(type) {
			case *schema.Charset:
				from, ok := change.From.(*schema.Charset)
				if !ok {
					return fmt.Errorf("mismatch ModifyAttr attributes: %T != %T", change.To, change.From)
				}
				b.P("CHARSET", to.V)
				r.P("CHARSET", from.V)
			case *schema.Collation:
				from, ok := change.From.(*schema.Collation)
				if !ok {
					return fmt.Errorf("mismatch ModifyAttr attributes: %T != %T", change.To, change.From)
				}
				b.P("COLLATE", to.V)
				r.P("COLLATE", from.V)
			default:
				return fmt.Errorf("unexpected schema ModifyAttr: %T", change)
			}
		default:
			return fmt.Errorf("unsupported ModifySchema change %T", change)
		}
	}
	if b.Len() > 0 {
		bs := Build("ALTER DATABASE").Ident(modify.S.Name)
		rs := bs.Clone()
		bs.WriteString(b.String())
		rs.WriteString(r.String())
		s.append(&migrate.Change{
			Cmd:     bs.String(),
			Reverse: rs.String(),
			Source:  modify,
			Comment: fmt.Sprintf("modify %q schema", modify.S.Name),
		})
	}
	return nil
}

// addTable builds and appends the migrate.Change
// for creating a table in a schema.
func (s *state) addTable(add *schema.AddTable) error {
	var (
		errors []string
		b      = Build("CREATE TABLE").Table(add.T)
	)
	if sqlx.Has(add.Extra, &schema.IfNotExists{}) {
		b.P("IF NOT EXISTS")
	}
	b.Wrap(func(b *sqlx.Builder) {
		b.MapComma(add.T.Columns, func(i int, b *sqlx.Builder) {
			if err := s.column(b, add.T, add.T.Columns[i]); err != nil {
				errors = append(errors, err.Error())
			}
		})
		if pk := add.T.PrimaryKey; pk != nil {
			b.Comma().P("PRIMARY KEY")
			indexParts(b, pk.Parts)
		}
		if len(add.T.Indexes) > 0 {
			b.Comma()
		}
		b.MapComma(add.T.Indexes, func(i int, b *sqlx.Builder) {
			idx := add.T.Indexes[i]
			index(b, idx)
		})
		if len(add.T.ForeignKeys) > 0 {
			b.Comma()
			if err := s.fks(b, add.T.ForeignKeys...); err != nil {
				errors = append(errors, err.Error())
			}
		}
		for _, attr := range add.T.Attrs {
			if c, ok := attr.(*schema.Check); ok {
				b.Comma()
				s.check(b, c)
			}
		}
	})
	if len(errors) > 0 {
		return fmt.Errorf("create table %q: %s", add.T.Name, strings.Join(errors, ", "))
	}
	s.tableAttr(b, add, add.T.Attrs...)
	s.append(&migrate.Change{
		Cmd:     b.String(),
		Source:  add,
		Reverse: Build("DROP TABLE").Table(add.T).String(),
		Comment: fmt.Sprintf("create %q table", add.T.Name),
	})
	return nil
}

// dropTable builds and appends the migrate.Change
// for dropping a table from a schema.
func (s *state) dropTable(drop *schema.DropTable) {
	b := Build("DROP TABLE").Table(drop.T)
	if sqlx.Has(drop.Extra, &schema.IfExists{}) {
		b.P("IF EXISTS")
	}
	s.append(&migrate.Change{
		Cmd:     b.String(),
		Source:  drop,
		Comment: fmt.Sprintf("drop %q table", drop.T.Name),
	})
}

// modifyTable builds and appends the migrate.Changes for bringing
// the table into its modified state.
func (s *state) modifyTable(modify *schema.ModifyTable) error {
	var changes [2][]schema.Change
	for _, change := range skipAutoChanges(modify.Changes) {
		switch change := change.(type) {
		// Constraints should be dropped before dropping columns, because if a column
		// is a part of multi-column constraints (like, unique index), ALTER TABLE
		// might fail if the intermediate state violates the constraints.
		case *schema.DropIndex:
			changes[0] = append(changes[0], change)
		case *schema.ModifyForeignKey:
			// Foreign-key modification is translated into 2 steps.
			// Dropping the current foreign key and creating a new one.
			changes[0] = append(changes[0], &schema.DropForeignKey{
				F: change.From,
			})
			// Drop the auto-created index for referenced if the reference was changed.
			if change.Change.Is(schema.ChangeRefTable | schema.ChangeRefColumn) {
				changes[0] = append(changes[0], &schema.DropIndex{
					I: &schema.Index{
						Name:  change.From.Symbol,
						Table: modify.T,
					},
				})
			}
			changes[1] = append(changes[1], &schema.AddForeignKey{
				F: change.To,
			})
		// Index modification requires rebuilding the index.
		case *schema.ModifyIndex:
			changes[0] = append(changes[0], &schema.DropIndex{
				I: change.From,
			})
			changes[1] = append(changes[1], &schema.AddIndex{
				I: change.To,
			})
		case *schema.DropAttr:
			return fmt.Errorf("unsupported change type: %v", change.A)
		default:
			changes[1] = append(changes[1], change)
		}
	}
	for i := range changes {
		if len(changes[i]) > 0 {
			if err := s.alterTable(modify.T, changes[i]); err != nil {
				return err
			}
		}
	}
	return nil
}

// alterTable modifies the given table by executing on it a list of
// changes in one SQL statement.
func (s *state) alterTable(t *schema.Table, changes []schema.Change) error {
	var (
		errors     []string
		b          = Build("ALTER TABLE").Table(t)
		reverse    = Build("")
		reversible = true
	)
	b.MapComma(changes, func(i int, b *sqlx.Builder) {
		switch change := changes[i].(type) {
		case *schema.AddColumn:
			b.P("ADD COLUMN")
			if err := s.column(b, t, change.C); err != nil {
				errors = append(errors, err.Error())
			}
			reverse.Comma().P("DROP COLUMN").Ident(change.C.Name)
		case *schema.ModifyColumn:
			b.P("MODIFY COLUMN")
			if err := s.column(b, t, change.To); err != nil {
				errors = append(errors, err.Error())
			}
			reverse.Comma().P("MODIFY COLUMN")
			if err := s.column(reverse, t, change.From); err != nil {
				errors = append(errors, err.Error())
			}
		case *schema.DropColumn:
			b.P("DROP COLUMN").Ident(change.C.Name)
			reversible = false
		case *schema.AddIndex:
			b.P("ADD")
			index(b, change.I)
			reverse.Comma().P("DROP INDEX").Ident(change.I.Name)
		case *schema.DropIndex:
			b.P("DROP INDEX").Ident(change.I.Name)
			reverse.Comma().P("ADD")
			index(reverse, change.I)
			reversible = true
		case *schema.AddForeignKey:
			b.P("ADD")
			if err := s.fks(b, change.F); err != nil {
				errors = append(errors, err.Error())
			}
			reverse.Comma().P("DROP FOREIGN KEY").Ident(change.F.Symbol)
		case *schema.DropForeignKey:
			b.P("DROP FOREIGN KEY").Ident(change.F.Symbol)
			reverse.Comma().P("ADD")
			if err := s.fks(reverse, change.F); err != nil {
				errors = append(errors, err.Error())
			}
		case *schema.AddAttr:
			s.tableAttr(b, change, change.A)
			// Unsupported reverse operation.
			reversible = false
		case *schema.ModifyAttr:
			s.tableAttr(b, change, change.To)
			s.tableAttr(reverse.Comma(), change, change.From)
		case *schema.AddCheck:
			s.check(b.P("ADD"), change.C)
			// Reverse operation is supported if
			// the constraint name is not generated.
			if reversible = change.C.Name != ""; reversible {
				reverse.Comma().P("DROP CONSTRAINT").Ident(change.C.Name)
			}
		case *schema.DropCheck:
			b.P("DROP CONSTRAINT").Ident(change.C.Name)
			s.check(reverse.Comma().P("ADD"), change.C)
		case *schema.ModifyCheck:
			switch {
			case change.From.Name == "":
				errors = append(errors, "cannot modify unnamed check constraint")
			case change.From.Name != change.To.Name:
				errors = append(errors, fmt.Sprintf("mismatch check constraint names: %q != %q", change.From.Name, change.To.Name))
			// Enforcement added.
			case s.supportsEnforceCheck() && sqlx.Has(change.From.Attrs, &Enforced{}) && !sqlx.Has(change.To.Attrs, &Enforced{}):
				b.P("ALTER CHECK").Ident(change.From.Name).P("ENFORCED")
				reverse.Comma().P("ALTER CHECK").Ident(change.From.Name).P("NOT ENFORCED")
			// Enforcement dropped.
			case s.supportsEnforceCheck() && !sqlx.Has(change.From.Attrs, &Enforced{}) && sqlx.Has(change.To.Attrs, &Enforced{}):
				b.P("ALTER CHECK").Ident(change.From.Name).P("NOT ENFORCED")
				reverse.Comma().P("ALTER CHECK").Ident(change.From.Name).P("ENFORCED")
			// Expr was changed.
			case change.From.Expr != change.To.Expr:
				b.P("DROP CHECK").Ident(change.From.Name).Comma().P("ADD")
				s.check(b, change.To)
				reverse.Comma().P("DROP CHECK").Ident(change.To.Name).Comma().P("ADD")
				s.check(reverse, change.From)
			default:
				errors = append(errors, "unknown check constraints change")
			}
		}
	})
	if len(errors) > 0 {
		return fmt.Errorf("alter table %q: %s", t.Name, strings.Join(errors, ", "))
	}
	change := &migrate.Change{
		Cmd: b.String(),
		Source: &schema.ModifyTable{
			T:       t,
			Changes: changes,
		},
		Comment: fmt.Sprintf("modify %q table", t.Name),
	}
	if reversible {
		b := Build("ALTER TABLE").Table(t)
		if _, err := b.ReadFrom(reverse); err != nil {
			return fmt.Errorf("unexpected buffer read: %w", err)
		}
		change.Reverse = b.String()
	}
	s.append(change)
	return nil
}

func (s *state) column(b *sqlx.Builder, t *schema.Table, c *schema.Column) error {
	typ, err := FormatType(c.Type.Type)
	if err != nil {
		return fmt.Errorf("format type for column %q: %w", c.Name, err)
	}
	b.Ident(c.Name).P(typ)
	if !c.Type.Null {
		b.P("NOT")
	}
	b.P("NULL")
	s.columnDefault(b, c)
	// Add manually the JSON_VALID constraint for older
	// versions < 10.4.3. See Driver.checks for full info.
	if _, ok := c.Type.Type.(*schema.JSONType); ok && s.mariadb() && s.ltV("10.4.3") && !sqlx.Has(c.Attrs, &schema.Check{}) {
		b.P("CHECK").Wrap(func(b *sqlx.Builder) {
			b.WriteString(fmt.Sprintf("json_valid(`%s`)", c.Name))
		})
	}
	for _, a := range c.Attrs {
		switch a := a.(type) {
		case *schema.Charset:
			if !supportsCharset(c.Type.Type) {
				return fmt.Errorf("column %q of type %T does not support the CHARSE attribute", c.Name, c.Type.Type)
			}
			// Define the charset explicitly
			// in case it is not the default.
			if s.character(t) != a.V {
				b.P("CHARSET", a.V)
			}
		case *schema.Collation:
			if !supportsCharset(c.Type.Type) {
				return fmt.Errorf("column %q of type %T does not support the COLLATE attribute", c.Name, c.Type.Type)
			}
			// Define the collation explicitly
			// in case it is not the default.
			if s.collation(t) != a.V {
				b.P("COLLATE", a.V)
			}
		case *OnUpdate:
			b.P("ON UPDATE", a.A)
		case *AutoIncrement:
			b.P("AUTO_INCREMENT")
			// Auto increment with value should be configured on table options.
			if a.V > 0 && !sqlx.Has(t.Attrs, &AutoIncrement{}) {
				t.Attrs = append(t.Attrs, a)
			}
		default:
			s.attr(b, a)
		}
	}
	return nil
}

func index(b *sqlx.Builder, idx *schema.Index) {
	var t IndexType
	if sqlx.Has(idx.Attrs, &t) {
		t.T = strings.ToUpper(t.T)
	}
	switch {
	case idx.Unique:
		b.P("UNIQUE")
	case t.T == IndexTypeFullText || t.T == IndexTypeSpatial:
		b.P(t.T)
	}
	b.P("INDEX").Ident(idx.Name)
	// Skip BTREE as it is the default type.
	if t.T == IndexTypeHash {
		b.P("USING", t.T)
	}
	indexParts(b, idx.Parts)
	if c := (schema.Comment{}); sqlx.Has(idx.Attrs, &c) {
		b.P("COMMENT", quote(c.Text))
	}
}

func indexParts(b *sqlx.Builder, parts []*schema.IndexPart) {
	b.Wrap(func(b *sqlx.Builder) {
		b.MapComma(parts, func(i int, b *sqlx.Builder) {
			switch part := parts[i]; {
			case part.C != nil:
				b.Ident(part.C.Name)
			case part.X != nil:
				b.WriteString(part.X.(*schema.RawExpr).X)
			}
			if s := (&SubPart{}); sqlx.Has(parts[i].Attrs, s) {
				b.WriteString(fmt.Sprintf("(%d)", s.Len))
			}
			// Ignore default collation (i.e. "ASC")
			if parts[i].Desc {
				b.P("DESC")
			}
		})
	})
}

func (s *state) fks(b *sqlx.Builder, fks ...*schema.ForeignKey) error {
	return b.MapCommaErr(fks, func(i int, b *sqlx.Builder) error {
		fk := fks[i]
		if fk.Symbol != "" {
			b.P("CONSTRAINT").Ident(fk.Symbol)
		}
		b.P("FOREIGN KEY")
		b.Wrap(func(b *sqlx.Builder) {
			b.MapComma(fk.Columns, func(i int, b *sqlx.Builder) {
				b.Ident(fk.Columns[i].Name)
			})
		})
		b.P("REFERENCES").Table(fk.RefTable)
		b.Wrap(func(b *sqlx.Builder) {
			b.MapComma(fk.RefColumns, func(i int, b *sqlx.Builder) {
				b.Ident(fk.RefColumns[i].Name)
			})
		})
		if fk.OnUpdate != "" {
			b.P("ON UPDATE", string(fk.OnUpdate))
		}
		if fk.OnDelete != "" {
			b.P("ON DELETE", string(fk.OnDelete))
		}
		if fk.OnUpdate == schema.SetNull || fk.OnDelete == schema.SetNull {
			for _, c := range fk.Columns {
				if !c.Type.Null {
					return fmt.Errorf("foreign key constraint was %[1]q SET NULL, but column %[1]q is NOT NULL", c.Name)
				}
			}
		}
		return nil
	})
}

// tableAttr writes the given table attribute to the SQL
// statement builder when a table is created or altered.
func (s *state) tableAttr(b *sqlx.Builder, c schema.Change, attrs ...schema.Attr) {
	for _, a := range attrs {
		switch a := a.(type) {
		case *CreateOptions:
			b.P(a.V)
		case *AutoIncrement:
			// Update the AUTO_INCREMENT if it is an update change or it is not the default.
			if _, ok := c.(*schema.ModifyAttr); ok || a.V > 1 {
				b.P("AUTO_INCREMENT", strconv.FormatInt(a.V, 10))
			}
		case *schema.Check:
			// Ignore CHECK constraints as they are not real attributes,
			// and handled on CREATE or ALTER.
		case *schema.Charset:
			b.P("CHARSET", a.V)
		case *schema.Collation:
			b.P("COLLATE", a.V)
		case *schema.Comment:
			b.P("COMMENT", quote(a.Text))
		}
	}
}

// character returns the table character-set from its attributes
// or from the default defined in the schema or the database.
func (s *state) character(t *schema.Table) string {
	var c schema.Charset
	if sqlx.Has(t.Attrs, &c) || t.Schema != nil && sqlx.Has(t.Schema.Attrs, &c) {
		return c.V
	}
	return s.charset
}

// collation returns the table collation from its attributes
// or from the default defined in the schema or the database.
func (s *state) collation(t *schema.Table) string {
	var c schema.Collation
	if sqlx.Has(t.Attrs, &c) || t.Schema != nil && sqlx.Has(t.Schema.Attrs, &c) {
		return c.V
	}
	return s.collate
}

func (s *state) append(c *migrate.Change) {
	s.Changes = append(s.Changes, c)
}

func (*state) attr(b *sqlx.Builder, attrs ...schema.Attr) {
	for _, a := range attrs {
		switch a := a.(type) {
		case *schema.Collation:
			b.P("COLLATE", a.V)
		case *schema.Comment:
			b.P("COMMENT", quote(a.Text))
		}
	}
}

// columnDefault writes the default value of column to the builder.
func (s *state) columnDefault(b *sqlx.Builder, c *schema.Column) {
	switch x := c.Default.(type) {
	case *schema.Literal:
		v := x.V
		if !hasNumericDefault(c.Type.Type) && !isHex(v) {
			v = quote(v)
		}
		b.P("DEFAULT", v)
	case *schema.RawExpr:
		v := x.X
		// For backwards compatibility, quote raw expressions that are not wrapped
		// with parens for non-numeric column types (i.e. literals).
		switch t := c.Type.Type; {
		case isHex(v), hasNumericDefault(t), strings.HasPrefix(v, "(") && strings.HasSuffix(v, ")"):
		default:
			if _, ok := t.(*schema.TimeType); !ok || !strings.HasPrefix(strings.ToLower(v), currentTS) {
				v = quote(v)
			}
		}
		b.P("DEFAULT", v)
	}
}

// Build instantiates a new builder and writes the given phrase to it.
func Build(phrase string) *sqlx.Builder {
	b := &sqlx.Builder{QuoteChar: '`'}
	return b.P(phrase)
}

// skipAutoChanges filters unnecessary changes that are automatically
// happened by the database when ALTER TABLE is executed.
func skipAutoChanges(changes []schema.Change) []schema.Change {
	var (
		dropC   = make(map[string]bool)
		planned = make([]schema.Change, 0, len(changes))
	)
	for _, c := range changes {
		if c, ok := c.(*schema.DropColumn); ok {
			dropC[c.C.Name] = true
		}
	}
	for i, c := range changes {
		// Simple case for skipping key dropping, if its columns are dropped.
		// https://dev.mysql.com/doc/refman/8.0/en/alter-table.html#alter-table-add-drop-column
		c, ok := c.(*schema.DropIndex)
		if !ok {
			planned = append(planned, changes[i])
			continue
		}
		for _, p := range c.I.Parts {
			if p.C == nil || !dropC[p.C.Name] {
				planned = append(planned, c)
				break
			}
		}
	}
	return planned
}

// checks writes the CHECK constraint to the builder.
func (s *state) check(b *sqlx.Builder, c *schema.Check) {
	expr := c.Expr
	// Expressions should be wrapped with parens.
	if t := strings.TrimSpace(expr); !strings.HasPrefix(t, "(") || !strings.HasSuffix(t, ")") {
		expr = "(" + t + ")"
	}
	if c.Name != "" {
		b.P("CONSTRAINT").Ident(c.Name)
	}
	b.P("CHECK", expr)
	if s.supportsEnforceCheck() && sqlx.Has(c.Attrs, &Enforced{}) {
		b.P("ENFORCED")
	}
}

// supportsCharset reports if the given type supports the CHARSET and COLLATE
// clauses. See: https://dev.mysql.com/doc/refman/8.0/en/charset-column.html
func supportsCharset(t schema.Type) bool {
	switch t.(type) {
	case *schema.StringType, *schema.EnumType, *SetType:
		return true
	default:
		return false
	}
}

func quote(s string) string {
	if sqlx.IsQuoted(s, '"', '\'') {
		return s
	}
	return strconv.Quote(s)
}
