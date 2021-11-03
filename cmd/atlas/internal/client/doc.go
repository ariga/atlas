// Package client provides abstraction over the usage of Atlas.
// Example usage of client:
//		d,close,_ := client.NewAtlasDriver(dsn)
//		defer close()
//		data := client.Inspect(ctx, d)
//
package client
