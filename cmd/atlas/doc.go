// Package atlas provides abstraction for cli/lib.
// Example usage:
//		d, _ := driver.NewAtlas(dsn)
//		defer d.Close()
//		data := schema.Inspect(ctx, d.Inspector, "schemaName")
//
package atlas
