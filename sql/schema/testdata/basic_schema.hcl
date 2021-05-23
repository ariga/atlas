schema "todo" {

}

table "users" {
  schema = schema.todo.name
  column "id" {
    type = "integer"
    null = false
    attributes {
      size = 1
      unsigned = true
    }
  }
  column "name" {
    type = "string"
    null = true
    attributes {
      size = 255
    }
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
    attributes {
      values = [
        "pending",
        "in_progress",
        "done",
      ]
    }
  }
}