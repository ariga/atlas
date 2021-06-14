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
    default = "default"
  }
  column "int" {
    type = "int"
    default = 1
  }
  column "float" {
    type = "float"
    default = 1.1
  }
  column "enum" {
    type = "enum"
    values = ["1", "2"]
    default = "1"
  }
  column "bool" {
    type = "boolean"
    default = true
  }
}
