// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type DefaultContainer struct {
	ent.Schema
}

func (DefaultContainer) Fields() []ent.Field {
	return []ent.Field{
		field.String("stringdef").
			Default("default"),
		field.Int("int").
			Default(1),
		field.Bool("bool").
			Default(true),
		field.Enum("enum").
			Values("1", "2").
			Default("1"),
		field.Float("float").
			Default(1.5),
	}
}
