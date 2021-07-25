schema "todo" {

}

table "user" {
  schema = schema.todo
  column "name" {
    type = "string"
    dialect "mysql" {
      type = "varchar(255)"
      version = "8"
    }
    dialect "mysql" {
      type = "varchar(100)"
      version = "9"
    }
  }
}