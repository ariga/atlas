schema "todo" {

}

table "tasks" {
  schema = schema.todo.name
  column "uuid" {
    type = "string"
    default = "uuid()"
  }
  column "text" {
    type = "string"
  }
}