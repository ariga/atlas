// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlclient

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/url"
	"sync"

	"ariga.io/atlas/schema/schemaspec"
	"ariga.io/atlas/schema/schemaspec/schemahcl"
	"ariga.io/atlas/sql/schema"

	"ariga.io/atlas/sql/migrate"
)

type (
	// Client provides the common functionalities for working with Atlas from different
	// applications (e.g. CLI and TF). Note, the Client is dialect specific and should
	// be instantiated using a call to Open.
	Client struct {
		// DB used for creating the client.
		DB *sql.DB
		// URL holds an enriched url.URL.
		URL *URL

		// A migration driver for the attached dialect.
		migrate.Driver

		// Marshal and Evaluator functions for decoding
		// and encoding the schema documents.
		schemaspec.Marshaler
		schemahcl.Evaluator
	}

	// URL extends the standard url.URL with additional
	// connection information attached by the Opener (if any).
	URL struct {
		*url.URL

		// The DSN used for opening the connection.
		DSN string

		// The Schema this client is connected to.
		Schema string
	}
)

// Close closes the underlying database connection and the migration
// driver in case it implements the io.Closer interface.
func (c *Client) Close() (err error) {
	if c, ok := c.Driver.(io.Closer); ok {
		err = c.Close()
	}
	if cerr := c.DB.Close(); cerr != nil {
		if err != nil {
			cerr = fmt.Errorf("%w: %v", err, cerr)
		}
		err = cerr
	}
	return err
}

type (
	// Opener opens a migration driver by the given URL.
	Opener interface {
		Open(ctx context.Context, u *url.URL) (*Client, error)
	}

	// OpenerFunc allows using a function as an Opener.
	OpenerFunc func(context.Context, *url.URL) (*Client, error)

	driver struct {
		Opener
		name  string
		parse func(*url.URL) *URL
	}
)

// Open calls f(ctx, u).
func (f OpenerFunc) Open(ctx context.Context, u *url.URL) (*Client, error) {
	return f(ctx, u)
}

var drivers sync.Map

// Open opens an Atlas client by its provided url string.
func Open(ctx context.Context, s string) (*Client, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("sql/sqlclient: parse open url: %w", err)
	}
	v, ok := drivers.Load(u.Scheme)
	if !ok {
		return nil, fmt.Errorf("sql/sqlclient: no opener was register with name %q", u.Scheme)
	}
	client, err := v.(driver).Open(ctx, u)
	if err != nil {
		return nil, err
	}
	if client.URL == nil {
		client.URL = v.(driver).parse(u)
	}
	return client, nil
}

type (
	registerOptions struct {
		parse    func(*url.URL) *URL
		flavours []string
		codec    interface {
			schemaspec.Marshaler
			schemahcl.Evaluator
		}
	}
	// RegisterOption allows configuring the Opener
	// registration using functional options.
	RegisterOption func(*registerOptions)
)

// RegisterFlavours allows registering additional flavours
// (i.e. names), accepted by Atlas to open clients.
func RegisterFlavours(flavours ...string) RegisterOption {
	return func(opts *registerOptions) {
		opts.flavours = flavours
	}
}

// RegisterURLParser allows registering a function for parsing
// the url.URL and attach additional info to the extended URL.
func RegisterURLParser(p func(*url.URL) *URL) RegisterOption {
	return func(opts *registerOptions) {
		opts.parse = p
	}
}

// RegisterCodec registers static codec for attaching into
// the client after it is opened.
func RegisterCodec(m schemaspec.Marshaler, e schemahcl.Evaluator) RegisterOption {
	return func(opts *registerOptions) {
		opts.codec = struct {
			schemaspec.Marshaler
			schemahcl.Evaluator
		}{
			Marshaler: m,
			Evaluator: e,
		}
	}
}

// DriverOpener is a helper Opener creator for sharing between all drivers.
func DriverOpener(open func(schema.ExecQuerier) (migrate.Driver, error)) Opener {
	return OpenerFunc(func(ctx context.Context, u *url.URL) (*Client, error) {
		v, ok := drivers.Load(u.Scheme)
		if !ok {
			return nil, fmt.Errorf("sql/sqlclient: unexpected missing opener %q", u.Scheme)
		}
		ur := v.(driver).parse(u)
		db, err := sql.Open(v.(driver).name, ur.DSN)
		if err != nil {
			return nil, err
		}
		drv, err := open(db)
		if err != nil {
			if cerr := db.Close(); cerr != nil {
				err = fmt.Errorf("%w: %v", err, cerr)
			}
			return nil, err
		}
		return &Client{
			DB:     db,
			URL:    ur,
			Driver: drv,
		}, nil
	})
}

// Register registers a client Opener (i.e. creator) with the given name.
func Register(name string, opener Opener, opts ...RegisterOption) {
	if opener == nil {
		panic("sql/sqlclient: Register opener is nil")
	}
	opt := &registerOptions{
		// Default URL parser uses the URL as the DSN.
		parse: func(u *url.URL) *URL { return &URL{URL: u, DSN: u.String()} },
	}
	for i := range opts {
		opts[i](opt)
	}
	if opt.codec != nil {
		f := opener
		opener = OpenerFunc(func(ctx context.Context, u *url.URL) (*Client, error) {
			c, err := f.Open(ctx, u)
			if err != nil {
				return nil, err
			}
			c.Marshaler, c.Evaluator = opt.codec, opt.codec
			return c, nil
		})
	}
	for _, f := range append(opt.flavours, name) {
		if _, ok := drivers.Load(f); ok {
			panic("sql/sqlclient: Register called twice for " + f)
		}
		drivers.Store(f, driver{
			name:   name,
			parse:  opt.parse,
			Opener: opener,
		})
	}
}
