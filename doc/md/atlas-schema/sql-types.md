---
id: sql-types
title: SQL Column Types
slug: /atlas-schema/sql-types
---

## MySQL

### Bit

The `bit` type allows creating [BIT](https://dev.mysql.com/doc/refman/8.0/en/bit-type.html) columns.
An optional size attribute allows controlling the number of bits stored in a column, ranging from 1 to 64.

```hcl
table "t" {
  schema = schema.test
  column "c1" {
    type = bit
  }
  column "c2" {
    type = bit(4)
  }
}
```

### Binary

The `varbinary` and `binary` types allow storing binary byte strings.

```hcl
table "t" {
  schema = schema.test
  column "c1" {
    // Equals to binary(1).
    type = binary
  }
  column "c2" {
    type = binary(10)
  }
  column "c3" {
    type = varbinary(255)
  }
}
```

### Blob

The `tinyblob`, `mediumblob`, `blob` and `longblob` types allow storing binary large objects.

```hcl
table "t" {
  schema = schema.test
  column "c1" {
    type = tinyblob
  }
  column "c2" {
    type = mediumblob
  }
  column "c3" {
    type = blob
  }
  column "c4" {
    type = longblob
  }
}
```

### Boolean

The `bool` and `boolean` types are mapped to `tinyint(1)` in MySQL. Still, Atlas allows maintaining columns of type `bool`
in the schema for simplicity reasons.

```hcl
table "t" {
  schema = schema.test
  column "c1" {
    type = bool
  }
  column "c2" {
    type = boolean
  }
}
```

Learn more about the motivation for these types in the
[MySQL website](https://dev.mysql.com/doc/refman/8.0/en/other-vendor-data-types.html).

### Date and Time

Atlas supports the standard MySQL types for storing date and time values: `time`, `timestamp`, `date`, `datetime`,
and `year`.

```hcl
table "t" {
  schema = schema.test
  column "c1" {
    type = time
  }
  column "c2" {
    type = timestamp
  }
  column "c3" {
    type = date
  }
  column "c4" {
    type = datetime
  }
  column "c5" {
    type = year
  }
  column "c6" {
    type = time(1)
  }
  column "c7" {
    type = timestamp(2)
  }
  column "c8" {
    type = datetime(4)
  }
}
```

### Fixed Point (Decimal)

The `decimal` and `numeric` types are supported for storing exact numeric values. Note that in MySQL the two types
are identical.

```hcl
table "t" {
  schema = schema.test
  column "c1" {
    // Equals to decimal(10) as the
    // default precision is 10.
    type = decimal
  }
  column "c2" {
    // Equals to decimal(5,0).
    type = decimal(5)
  }
  column "c3" {
    type = decimal(5,2)
  }
  column "c4" {
    type     = numeric
    unsigned = true
  }
}
```

### Floating Point (Float)

The `float` and `double` types are supported for storing approximate numeric values.

```hcl
table "t" {
  schema = schema.test
  column "c1" {
    type = float
  }
  column "c2" {
    type = double
  }
  column "c3" {
    type     = float
    unsigned = true
  }
  column "c4" {
    type     = double
    unsigned = true
  }
}
```

### Enum

The `enum` type allows storing a set of enumerated values.

```hcl
table "t" {
  schema = schema.test
  column "c1" {
    type = enum("a", "b")
  }
  column "c2" {
    type = enum(
      "c",
      "d",
    )
  }
}
```

### Integer

The `tinyint`, `smallint`, `int`, `mediumint`, `bigint` integer types are support by Atlas.

```hcl
table "t" {
  schema = schema.test
  column "c1" {
    type = int
  }
  column "c2" {
    type = tinyint
  }
  column "c3" {
    type = smallint
  }
  column "c4" {
    type = mediumint
  }
  column "c5" {
    type = bigint
  }
}
```

#### Integer Attributes

The [`auto_increment`](https://dev.mysql.com/doc/refman/8.0/en/numeric-type-attributes.html), and `unsigned` attributes
are also supported by integer types.

```hcl
table "t" {
  schema = schema.test
  column "c1" {
    type     = tinyint
    unsigned = true
  }
  column "c2" {
    type           = smallint
    auto_increment = true
  }
  primary_key {
    columns = [column.c2]
  }
}
```

### JSON

The `json` type allows storing JSON objects.

```hcl
table "t" {
  schema = schema.test
  column "c1" {
    type = json
  }
}
```

### Set

The `set` type allows storing a set of values.

```hcl
table "t" {
  schema = schema.test
  column "c1" {
    type = set("a", "b")
  }
  column "c2" {
    type = set(
      "c",
      "d",
    )
  }
}
```

### String

Atlas supports the standard MySQL types for storing string values. `varchar`, `char`, `tinytext`, `mediumtext`, `text`
and `longtext`.

```hcl
table "t" {
  schema = schema.test
  column "c1" {
    type = varchar(255)
  }
  column "c2" {
    type = char(1)
  }
  column "c3" {
    type = tinytext
  }
  column "c4" {
    type = mediumtext
  }
  column "c5" {
    type = text
  }
  column "c6" {
    type = longtext
  }
}
```

### Spatial

The `geometry`, `point`, `multipoint`, `linestring` and the rest of the
[MySQL spatial types](https://dev.mysql.com/doc/refman/8.0/en/spatial-type-overview.html) are supported by Atlas.

```hcl
table "t" {
  schema = schema.test
  column "c1" {
    type = geometry
  }
  column "c2" {
    type = point
  }
  column "c3" {
    type = multipoint
  }
  column "c4" {
    type = linestring
  }
}
```

## PostgreSQL

### Array

Atlas supports defining PostgreSQL array types using the `sql` function.

```hcl
table "t" {
  schema = schema.test
  column "c1" {
    type = sql("int[]")
  }
  column "c2" {
    type = sql("text[]")
  }
  column "c3" {
    type = sql("int ARRAY")
  }
  column "c4" {
    type = sql("varchar(255)[]")
  }
  column "c5" {
    // The current PostgreSQL implementation
    // ignores any supplied array size limits.
    type = sql("point[4][4]")
  }
}
```

### Bit

The `bit` and `bit varying` types allow creating
[bit string](https://www.postgresql.org/docs/current/datatype-bit.html) columns.

```hcl
table "t" {
  schema = schema.test
  column "c1" {
    // Equals to bit(1).
    type = bit
  }
  column "c2" {
    type = bit(2)
  }
  column "c3" {
    // Unlimited length.
    type = bit_varying
  }
  column "c4" {
    type = bit_varying(1)
  }
}

```

### Boolean

The `boolean` type allows creating standard SQL boolean columns.

```hcl
table "t" {
  schema = schema.test
  column "c1" {
    type = boolean
  }
  column "c2" {
    type    = boolean
    default = true
  }
}
```

### Binary

The `bytea` type allows creating binary string columns.

```hcl
table "t" {
  schema = schema.test
  column "c1" {
    type = bytea
  }
}
```

### Date, Time and Interval

Atlas supports the standard PostgreSQL types for creating date, time and interval columns.

```hcl
table "t" {
  schema = schema.test
  column "c1" {
    type = date
  }
  column "c2" {
    // Equals to "time without time zone".
    type = time
  }
  column "c3" {
    // Equals to "time with time zone".
    type = timetz
  }
  column "c4" {
    // Equals "timestamp without time zone".
    type = timestamp
  }
  column "c5" {
    // Equals "timestamp with time zone".
    type = timestamptz
  }
  column "c6" {
    type = timestamp(4)
  }
  column "c7" {
    type = interval
  }
}
```

### Enum

The `enum` type allows storing a set of enumerated values.

```hcl
enum "status" {
  schema = schema.test
  values = ["on", "off"]
}

table "t1" {
  schema = schema.test
  column "c1" {
    type = enum.status
  }
}

table "t2" {
  schema = schema.test
  column "c1" {
    type = enum.status
  }
}
```

### Fixed Point (Decimal)

The `decimal` and `numeric` types are supported for storing exact numeric values. Note that in PostgreSQL the two types
are identical.

```hcl
table "t" {
  schema = schema.test
  column "c1" {
    // Equals to decimal.
    type = numeric
  }
  column "c2" {
    // Equals to decimal(5).
    type = numeric(5)
  }
  column "c3" {
    // Equals to decimal(5,2).
    type = numeric(5,2)
  }
}
```


### Floating Point (Float)

The `real` and `double_precision` types are supported for storing
[approximate numeric values](https://www.postgresql.org/docs/current/datatype-numeric.html#DATATYPE-FLOAT).

```hcl
table "t" {
  schema = schema.test
  column "c1" {
    type = real
  }
  column "c2" {
    type = double_precision
  }
  column "c3" {
    // Equals to real when precision is between 1 to 24.
    type = float(10)
  }
  column "c2" {
    // Equals to double_precision when precision is between 1 to 24.
    type = float(30)
  }
}
```

### Geometric

Atlas supports the standard PostgreSQL types for creating geometric columns.

```hcl
table "t" {
  schema = schema.test
  column "c1" {
    type = circle
  }
  column "c2" {
    type = line
  }
  column "c3" {
    type = lseg
  }
  column "c4" {
      type = box
  }
  column "c5" {
      type = path
  }
  column "c6" {
      type = polygon
  }
  column "c7" {
      type = point
  }
}
```

### Integer

The `smallint`, `integer` / `int`, `bigint` types allow creating integer types.

```hcl
table "t" {
  schema = schema.test
  column "c1" {
    type = smallint
  }
  column "c2" {
    type = integer
  }
  column "c3" {
    type = int
  }
  column "c4" {
    type    = bigint
    default = 1
  }
}
```

### JSON

The `json` and `jsonb` types allow creating columns for storing JSON objects.

```hcl
table "t" {
  schema = schema.test
  column "c1" {
    type = json
  }
  column "c2" {
    type = jsonb
  }
}
```

### Money

The `money` data type allows creating columns for storing currency amount with a fixed fractional precision.

```hcl
table "t" {
  schema = schema.test
  column "c1" {
    type = money
  }
}
```

### Network Address

The `inet`, `cidr`, `macaddr` and `macaddr8` types allow creating network address columns.

```hcl
table "t" {
  schema = schema.test
  column "c1" {
    type = inet
  }
  column "c2" {
    type = cidr
  }
  column "c3" {
    type = macaddr
  }
  column "c4" {
    type = macaddr8
  }
}
```

### Range

PostgreSQL supports the creation of range types for storing range of values of some element type.
Learn more about them in the [PostgreSQL website](https://www.postgresql.org/docs/current/rangetypes.html).


```hcl
table "t" {
  schema = schema.test
  column "r1" {
    type = int4range
  }
  column "r2" {
    type = int8range
  }
  column "r3" {
    type = numrange
  }
  column "r4" {
    type = tsrange
  }
  column "r5" {
    type = tstzrange
  }
  column "r6" {
    type = daterange
  }
  column "r7" {
    type = int4multirange
  }
  column "r8" {
    type = int8multirange
  }
  column "r9" {
    type = nummultirange
  }
  column "r10" {
    type = tsmultirange
  }
  column "r11" {
    type = tstzmultirange
  }
  column "r12" {
    type = datemultirange
  }
}
```

### Serial

PostgreSQL supports creating columns of types `smallserial`, `serial`, and `bigserial`. Note that these types are not
_actual_ types, but more like "macros" for creating non-nullable integer columns with sequences attached.

```hcl
table "t" {
  schema = schema.test
  column "c1" {
      type = smallserial
  }
  column "c2" {
      type = serial
  }
  column "c3" {
      type = bigserial
  }
}
```

### String

The `varchar`, `char`, `character_varying`, `character` and `text` types allow creating string columns.

```hcl
table "t" {
  schema = schema.test
  column "c1" {
    // Unlimited length.
    type = varchar
  }
  column "c2" {
    // Alias to character_varying(255).
    type = varchar(255)
  }
  column "c3" {
    // Equals to char(1).
    type = char
  }
  column "c4" {
    // Alias to character(5).
    type = char(5)
  }
  column "c5" {
    type = text
  }
}
```

### Text Search

The `tsvector` and `tsquery` data types are designed to store and query full text search. Learn more about them in the
[PostgreSQL website](https://www.postgresql.org/docs/current/datatype-textsearch.html).

```hcl
table "t" {
  schema = schema.test
  column "tsv" {
    type = tsvector
  }
  column "tsq" {
    type = tsquery
  }
}
```

### UUID

The `uuid` data type allows creating columns for storing Universally Unique Identifiers (UUID).

```hcl
table "t" {
  schema = schema.test
  column "c1" {
    type = uuid
  }
  column "c2" {
    type    = uuid
    default = sql("gen_random_uuid()")
  }
}
```

### XML

The `xml` data type allows creating columns for storing XML data.

```hcl
table "t" {
  schema = schema.test
  column "c1" {
    type = xml
  }
}
```

## SQLite

Values in SQLite are stored in one of the four native types: `BLOB`, `INTEGER`, `NULL`, `TEXT` and `REAL`. Still, Atlas
supports variety of data types that are commonly used by ORMs. These types are mapped to column affinities based on
the rules described in [SQLite website](https://www.sqlite.org/datatype3.html#type_affinity).

### Blob

The `blob` data type allows creating columns with `BLOB` type affinity.

```hcl
table "t" {
  schema = schema.main
  column "c" {
    type = blob
  }
}
```

### Integer

The `int` and `integer` data types allow creating columns with `INTEGER` type affinity.

```hcl
table "t" {
  schema = schema.main
  column "c" {
    type = int
  }
}
```

### Numeric

The `numeric` and `decimal` data types allow creating columns with `NUMERIC` type affinity.

```hcl
table "t" {
  schema = schema.main
  column "c" {
    type = decimal
  }
}
```

### Text

The `text`, `varchar`, `clob`, `character` and `varying_character` data types allow creating columns with `text` type
affinity. i.e. stored as text strings.

```hcl
table "t" {
  schema = schema.main
  column "c" {
    type = text
  }
}
```

### Real

The `real`, `double`, `double_precision`, and `float` data types allow creating columns with `real` type
affinity.

```hcl
table "t" {
  schema = schema.main
  column "c" {
    type = real
  }
}
```

### Additional Types

As mentioned above, Atlas supports variety of data types that are commonly used by ORMs. e.g. [Ent](https://entgo.io).

```hcl
table "t" {
  schema = schema.main
  column "c1" {
    type = bool
  }
  column "c2" {
    type = date
  }
  column "c3" {
    type = datetime
  }
  column "c4" {
    type = uuid
  }
  column "c5" {
    type = json
  }
}
```
