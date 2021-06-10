schema "todo" {

}

table "tasks" {
  schema = schema.todo
  column "text" {
    type = "string"
    comment = "comment"
    charset = "charset"
    collation = "collation"
  }
}
