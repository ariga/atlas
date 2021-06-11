schema "todo" {

}

table "tasks" {
  schema = schema.todo
  column "uuid" {
    type = "string"
    default = "uuid()"
  }
  column "text" {
    type = "string"
  }
}
