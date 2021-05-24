schema "todo" {

}

table "users" {
  schema = schema.todo.name
  column "id" {
    type = "integer"
    null = false
    size = 1
    unsigned = true
  }
  column "name" {
    type = "string"
    null = true
    size = 255
  }
}

table "roles" {
  schema = schema.todo.name
  column "id" {
    type = "integer"
  }
  column "name" {
    type = "string"
  }
}

table "todos" {
  schema = schema.todo.name
  column "id" {
    type = "integer"
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
  }