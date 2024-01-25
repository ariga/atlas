---
id: hcl-types
title: HCL Column Types
slug: /atlas-schema/hcl-types
---

The following guide describes the column types supported by Atlas HCL, and how to use them.

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

### Domain

The `domain` type is a user-defined data type that is based on an existing data type but with optional constraints
and default values. Learn more about it in the [PostgreSQL website](https://www.postgresql.org/docs/current/domains.html).

```hcl
domain "us_postal_code" {
  schema = schema.public
  type   = text
  null   = true
  check "us_postal_code_check" {
    expr = "((VALUE ~ '^\\d{5}$'::text) OR (VALUE ~ '^\\d{5}-\\d{4}$'::text))"
  }
}

domain "username" {
  schema = schema.public
  type    = text
  null    = false
  default = "anonymous"
  check "username_length" {
    expr = "(length(VALUE) > 3)"
  }
}

table "users" {
  schema = schema.public
  column "name" {
    type = domain.username
  }
  column "zip" {
    type = domain.us_postal_code
  }
}
```

### Enum

The `enum` type allows storing a set of enumerated values. Learn more about it in the [PostgreSQL website](https://www.postgresql.org/docs/current/datatype-enum.html).

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

## SQL Server

### Bit

The `bit` type allows creating [BIT](https://learn.microsoft.com/en-us/sql/t-sql/data-types/bit-transact-sql) columns.

```hcl
table "t" {
  schema = schema.dbo
  column "c1" {
    type = bit
  }
}
```

### Binary strings

The `varbinary` and `binary` types allow storing binary byte strings.

```hcl
table "t" {
  schema = schema.dbo
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
  column "c4" {
    // Max length: 8,000 bytes.
    type = varbinary(MAX)
  }
}
```

### Date and Time

Atlas supports the standard SQL Server types for storing date and time values: `date`, `datetime`, `datetime2`, `datetimeoffset`, `smalldatetime` and `time`.
The document on Microsoft website has more information on [date and time types](https://learn.microsoft.com/en-us/sql/t-sql/data-types/data-types-transact-sql#date-and-time).

```hcl
table "t" {
  schema = schema.dbo
  column "c1" {
    type = date
  }
  column "c2" {
    type = datetime
  }
  column "c3" {
    type = datetime2
  }
  column "c4" {
    type = datetimeoffset
  }
  column "c5" {
    type = smalldatetime
  }
  column "c6" {
    // Equals to time(7).
    type = time
  }
  column "c7" {
    type = time(1)
  }
  column "c8" {
    type = time(2)
  }
  column "c9" {
    type = time(3)
  }
  column "c10" {
    type = time(4)
  }
  column "c11" {
    type = time(5)
  }
  column "c12" {
    type = time(6)
  }
}
```

### Integer

The `int`, `bigint`, `smallint`, and `tinyint` integer types are support by Atlas.
See document on Microsoft website for more information on [integer types](https://learn.microsoft.com/en-us/sql/t-sql/data-types/int-bigint-smallint-and-tinyint-transact-sql).

```hcl
table "t" {
  schema = schema.dbo
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
    type = bigint
  }
}
```

#### Integer Blocks

The [`identity`](https://learn.microsoft.com/en-us/sql/t-sql/functions/identity-function-transact-sql) block can be used to create an identity column.

```hcl
table "t" {
  schema = schema.dbo
  column "c1" {
    type = tinyint
  }
  column "c2" {
    type = bigint
    identity {
      seed      = 701
      increment = 1000
    }
  }
  primary_key {
    columns = [column.c2]
  }
}
```

### Fixed Point (Decimal)

The [`decimal` and `numeric`](https://learn.microsoft.com/en-us/sql/t-sql/data-types/decimal-and-numeric-transact-sql) types are supported for storing exact numeric values. Note that in SQL Server the two types are identical.

```hcl
table "t" {
  schema = schema.dbo
  column "c1" {
    // Equals to decimal(18, 0) as the
    // default precision is 18.
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
    type = numeric
  }
}
```

### Floating Point (Float)

The `float` and `real` types are supported for storing approximate numeric values.
The document on Microsoft website has more information on [float types](https://learn.microsoft.com/en-us/sql/t-sql/data-types/float-and-real-transact-sql).

```hcl
table "t" {
  schema = schema.dbo
  column "c1" {
    // Equals to float(53).
    type = float
  }
  column "c2" {
    // float(n) is between 1 and 53.
    type = float(12)
  }
  column "c3" {
    // The ISO synonym for real is `float(24)`.
    type = real
  }
}
```

### Money

The [`money` and `smallmoney`](https://learn.microsoft.com/en-us/sql/t-sql/data-types/money-and-smallmoney-transact-sql) data types allows creating columns for storing currency amount with a fixed fractional precision.

```hcl
table "t" {
  schema = schema.dbo
  column "c1" {
    type = money
  }
  column "c2" {
    type = smallmoney
  }
}
```

### Character strings

The `char`, and `varchar` types allow creating string columns. The document on Microsoft website has more information on [string types](https://learn.microsoft.com/en-us/sql/t-sql/data-types/char-and-varchar-transact-sql).

```hcl
table "t" {
  schema = schema.dbo
  column "c1" {
    // Equals to varchar(1).
    type = varchar
  }
  column "c2" {
    type = varchar(255)
  }
  column "c3" {
    type = varchar(MAX)
  }
  column "c4" {
    // Equals to char(1).
    type = char
  }
  column "c5" {
    type = char(5)
  }
}
```

### Unicode character strings

The `nchar`, and `nvarchar` types allow creating string columns. The document on Microsoft website has more information on [unicode string types](https://learn.microsoft.com/en-us/sql/t-sql/data-types/nchar-and-nvarchar-transact-sql).

```hcl
table "t" {
  schema = schema.dbo
  column "c1" {
    // Equals to nvarchar(1).
    type = nvarchar
  }
  column "c2" {
    type = nvarchar(255)
  }
  column "c3" {
    type = nvarchar(MAX)
  }
  column "c4" {
    // Equals to nchar(1).
    type = nchar
  }
  column "c5" {
    type = nchar(5)
  }
}
```

### `ntext`, `text` and `image`

Atlas supports some deprecated types for backward compatibility. The document on Microsoft website has more information on [ntext, text and image types](https://learn.microsoft.com/en-us/sql/t-sql/data-types/ntext-text-and-image-transact-sql).

```hcl
table "t" {
  schema = schema.dbo
  column "c1" {
    type = ntext
  }
  column "c2" {
    type = text
  }
  column "c3" {
    type = image
  }
}
```

## ClickHouse

### Array
Atlas supports defining ClickHouse array types using the `sql` function.

```hcl
table "t" {
  schema = schema.test
  engine = Memory
  column "c1" {
    type = sql("Array(Int32)")
  }
  column "c2" {
    type = sql("Array(String)")
  }
  column "c3" {
    type = sql("Array(Array(Int32))")
  }
}
```
### Boolean
The `Bool` type allows creating standard SQL boolean columns.

```hcl
table "t" {
  schema = schema.test
  engine = Memory
  column "c1" {
    type = Bool
  }
  column "c2" {
    type    = Bool
    default = true
  }
}
```

### Date and Time
Atlas supports the standard ClickHouse types for creating date and time columns: `Date`, `DateTime`, `DateTime32` `DateTime64`.

```hcl
table "t" {
  schema = schema.test
  engine = Memory
  column "c1" {
    null     = false
    type     = Date
  }
  column "c2" {
    null    = false
    type    = Date32
  }
  column "c3" {
    null    = false
    type    = DateTime
  }
  column "c4" {
    null    = false
    type    = DateTime("America/New_York")
  }
  column "c5" {
    null    = false
    type    = DateTime
  }
  column "c6" {
    null    = false
    type    = DateTime32("America/New_York")
  }
  column "c7" {
    null    = false
    type    = DateTime64(3)
  }
  column "c8" {
    null    = false
    type    = DateTime64(3, "America/New_York")
  }
}
```

### Fixed Point (Decimal)
The `Decimal` type allows creating columns for storing exact numeric values.
The precision and scale are specified as below.
- `Decimal` Precision: 9, Scale: 0
- `Decimal32(Scale)` Precision: 9, Scale: Scale
- `Decimal64(Scale)` Precision: 18, Scale: Scale
- `Decimal128(Scale)` Precision: 38, Scale: Scale
- `Decimal256(Scale)` Precision: 76, Scale: Scale
- `Decimal(Precision, Scale)` Precision: Precision, Scale: Scale 

```hcl
table "t" {
  schema = schema.test
  engine = Memory
  column "c1" {
    null = false
    type = Decimal
  }
  column "c2" {
    null = false
    type = Decimal32(2)
  }
  column "c3" {
    null = false
    type = Decimal64(2)
  }
  column "c4" {
    null = false
    type = Decimal128(2)
  }
  column "c5" {
    null = false
    type = Decimal256(2)
  }
  column "c6" {
    null = false
    type = Decimal(11, 2)
  }
}
```

### Enum
The `Enum` type allows storing a set of enumerated values and supports defining ClickHouse enum types using the `sql` function.

```hcl
table "t" {
  schema = schema.test
  engine = Memory
  column "c1" {
    null = false
    type = Enum("a", "b")
  }
  column "c2" {
    null = false
    type = Enum8("a", "b")
  }
  column "c3" {
    null = false
    type = Enum16("a", "b")
  }
}
```

### Fixed String
The `FixedString` type allows creating columns for storing fixed-length string values.

```hcl
table "t" {
  schema = schema.test
  engine = Memory
  column "c1" {
    null = false
    type = FixedString(10)
  }
}
```

### Floating Point (Float)
The `Float32` and `Float64` types are supported for storing approximate numeric values.
The aliases for these types are `Float` and `Double`.

```hcl
table "t" {
  schema = schema.test
  engine = Memory
  column "c1" {
    null = false
    type = Float
  }
  column "c2" {
    null = false
    type = Double
  }
}
```

### Integer
The `Int8`, `Int16`, `Int32`, `Int64`, `Int128`, `Int256` types allow creating integer types.
The aliases for these types are `Tinyint`, `Smallint`, `Int`, `Bigint`.

```hcl
table "t" {
  schema = schema.test
  engine = Memory
  column "c1" {
    null = false
    type = Tinyint
  }
  column "c2" {
    null = false
    type = Smallint
  }
  column "c3" {
    null = false
    type = Int
  }
  column "c4" {
    null = false
    type = Bigint
  }
  column "c5" {
    null = false
    type = Int128
  }
  column "c6" {
    null = false
    type = Int256
  }
}
```

#### Integer Attributes

The `Unsigned` attribute is also supported by integer types.

```hcl
table "t" {
  schema = schema.test
  engine = Memory
  column "c1" {
    null     = false
    type     = Int
    unsigned = true
  }
}
```

### IPv4 and IPv6
The `IPv4` and `IPv6` types allow creating columns for storing IPv4 and IPv6 addresses.

```hcl
table "t" {
  schema = schema.test
  engine = Memory
  column "c1" {
    null = false
    type = IPv4
  }
  column "c2" {
    null = false
    type = IPv6
  }
}
```

### Spatial
Atlas supports the standard ClickHouse types for creating spatial columns.

```hcl
table "t" {
  schema = schema.test
  engine = Memory
  column "c1" {
    null = false
    type = Point
  }
  column "c2" {
    null = false
    type = Polygon
  }
  column "c3" {
    null = false
    type = MultiPolygon
  }
}
```

### Ring
The `Ring` type allows creating columns for storing ring values.

```hcl
table "t" {
  schema = schema.test
  engine = Memory
  column "c1" {
    null = false
    type = Ring
  }
}
```

### String
The `String` type allows creating columns for storing string values.

```hcl
table "t" {
  schema = schema.test
  engine = Memory
  column "c1" {
    null = false
    type = String
  }
}
```

### UUID
The `UUID` type allows creating columns for storing Universally Unique Identifiers (UUID).

```hcl
table "t" {
  schema = schema.test
  engine = Memory
  column "c1" {
    null = false
    type = UUID
  }
}
```

### Tuple
Atlas supports defining ClickHouse tuple types using the `sql` function.

```hcl 
table "t" {
  schema = schema.test
  engine = Memory
  column "c1" {
    null = false
    type = sql("Tuple(Int32, String)")
  }
}
```

### LowCardinality
Atlas supports defining ClickHouse low cardinality types using the `sql` function.

```hcl
table "t" {
  schema = schema.test
  engine = Memory
  column "c1" {
    null = false
    type = sql("LowCardinality(String)")
  }
}
```

### Nullable
Atlas supports defining ClickHouse nullable types using the `sql` function.
`Null` attribute is needed to be set to `true` for nullable types.

```hcl
table "t" {
  schema = schema.test
  engine = Memory
  column "c1" {
    null = true
    type = sql("Nullable(String)")
  }
}
```

### JSON
The `JSON` type allows creating columns for storing JSON objects.

```hcl
table "t" {
  schema = schema.test
  engine = Memory
  column "c1" {
    type = JSON
  }
}
```
