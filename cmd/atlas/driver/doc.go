// Package driver provides abstraction over the usage of Atlas.
// Example usage of client:
//		d,_ := client.NewAtlasDriver(dsn)
//		defer d.close()
//		data := schema.Inspect(ctx, d.Inspector)
//
package driver
