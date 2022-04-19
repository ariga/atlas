package atlas

import (
	"context"

	"ariga.io/atlas/sql"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
)

func Inspect(ctx context.Context, d *sql.Driver, url string, schemas ...string) ([]byte, error) {
	name, err := sql.SchemaNameFromURL(ctx, url)
	if err != nil {
		return nil, err
	}
	if name != "" {
		schemas = append(schemas, name)
	}
	s, err := d.InspectRealm(ctx, &schema.InspectRealmOption{
		Schemas: schemas,
	})
	if err != nil {
		return nil, err
	}
	ddl, err := d.MarshalSpec(s)
	if err != nil {
		return nil, err
	}
	return ddl, nil
}

func Plan(ctx context.Context, fromURL, toURL string) (*migrate.Plan, error) {
	toDriver, err := sql.DefaultMux.OpenAtlas(ctx, toURL)
	if err != nil {
		return nil, err
	}
	changes, err := Diff(ctx, fromURL, toURL)
	if err != nil {
		return nil, err
	}
	return toDriver.PlanChanges(ctx, "plan", changes)
}

func Diff(ctx context.Context, fromURL, toURL string) ([]schema.Change, error) {
	fromDriver, err := sql.DefaultMux.OpenAtlas(ctx, fromURL)
	if err != nil {
		return nil, err
	}
	defer fromDriver.Close()
	toDriver, err := sql.DefaultMux.OpenAtlas(ctx, toURL)
	if err != nil {
		return nil, err
	}
	defer toDriver.Close()
	fromName, err := sql.SchemaNameFromURL(ctx, fromURL)
	if err != nil {
		return nil, err
	}
	toName, err := sql.SchemaNameFromURL(ctx, toURL)
	if err != nil {
		return nil, err
	}
	fromSchema, err := fromDriver.InspectSchema(ctx, fromName, nil)
	if err != nil {
		return nil, err
	}
	toSchema, err := toDriver.InspectSchema(ctx, toName, nil)
	if err != nil {
		return nil, err
	}
	// SchemaDiff checks for name equality which is irrelevant in the case
	// the user wants to compare their contents, if the names are different
	// we reset them to allow the comparison.
	if fromName != toName {
		toSchema.Name = ""
		fromSchema.Name = ""
	}
	return toDriver.SchemaDiff(fromSchema, toSchema)
}
