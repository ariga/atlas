// Package client provides abstraction over the usage of Atlas.
// Example usage of client:
//		d,closer,_ := client.NewAtlasDriver(dsn)
//		defer closer()
//		data := client.Inspect(ctx, d)
//
package client
