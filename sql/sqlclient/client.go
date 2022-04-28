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

// Client provides the common functionalities for working with Atlas from different
// applications (e.g. CLI and TF). Note, the Client is dialect specific and should
// be instantiated using a call to Open.
type Client struct {
	// DB used for creating the client.
	DB *sql.DB

	// A migration driver for the attached dialect.
	migrate.Driver

	// Marshal and Unmarshal functions for decoding
	// and encoding the schema documents.
	schemaspec.Marshaler
	schemahcl.Evaluator
}

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

	namedOpener struct {
		Opener
		name string
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
	return v.(namedOpener).Open(ctx, u)
}

type (
	registerOptions struct {
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
func DriverOpener(open func(schema.ExecQuerier) (migrate.Driver, error), dsn func(*url.URL) string) Opener {
	return OpenerFunc(func(ctx context.Context, u *url.URL) (*Client, error) {
		v, ok := drivers.Load(u.Scheme)
		if !ok {
			return nil, fmt.Errorf("sql/sqlclient: unexpected missing opener %q", u.Scheme)
		}
		db, err := sql.Open(v.(namedOpener).name, dsn(u))
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
			Driver: drv,
		}, nil
	})
}

// Register registers a client Opener (i.e. creator) with the given name.
func Register(name string, opener Opener, opts ...RegisterOption) {
	if opener == nil {
		panic("sql/sqlclient: Register opener is nil")
	}
	opt := &registerOptions{}
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
		drivers.Store(f, namedOpener{
			name:   name,
			Opener: opener,
		})
	}
}
