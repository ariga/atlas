schema "todo" {

}

table "tasks" {
  schema = schema.todo.name
  column "text" {
    type = "string"
    comment = "comment"
    charset = "charset"
    collation = "collation"
  }
}
