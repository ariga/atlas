schema "todo" {

}

table "users" {
  schema = schema.todo
  column "id" {
    type = "uint"
    null = false
  }
  column "name" {
    type = "string"
    null = true
    size = 255
  }
}

table "roles" {
  schema = schema.todo
  column "id" {
    type = "int"
  }
  column "name" {
    type = "string"
  }
}

table "todos" {
  schema = schema.todo
  column "id" {
    type = "int"
  }
  column "content" {
    type = "string"
  }
  column "status" {
    type = "enum"
    values = [
      "pending",
      "in_progress",
      "done",
    ]
  }
  column "signature" {
    type = "binary"
    size = 128
  }
  column "visible" {
    type = "boolean"
  }
  column "decimal_col" {
    type = "decimal"
    precision = 2
    scale = 5
  }
  column "float_col" {
    type = "float"
    precision = 2
  }
  column "created" {
    type = "time"
  }
  column "json_col" {
    type = "json"
  }
  column "int8" {
    type = "int8"
  }
  column "uint8" {
    type = "uint8"
  }
  column "int16" {
    type = "int16"
  }
  column "int" {
    type = "int"
  }
  column "int64" {
    type = "int64"
  }
  column "uint64" {
    type = "uint64"
  }
}
