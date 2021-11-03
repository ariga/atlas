// Package client provides abstraction over the usage of Atlas.
// Example usage of client:
//		d := client.NewAtlasDriver(ctx,dsn)
//		defer client.Close()
//		data := client.Inspect(ctx,d)
//
package client
