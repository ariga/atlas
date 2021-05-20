schema "todo" {

}

table "users" {
  schema = schema.todo.name
  column "id" {
    type = "integer"
  }
  column "name" {
    type = "string"
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
