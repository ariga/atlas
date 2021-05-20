schema "todo" {

}

table "users" {
  schema = schemas.todo.name
  column "id" {
    type = "integer"
  }
  column "name" {
    type = "string"
  }
}

table "roles" {
  schema = schemas.todo.name
  column "id" {
    type = "integer"
  }
  column "name" {
    type = "string"
  }
}
