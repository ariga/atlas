schema "todo" {

}

table "tasks" {
  schema = schema.todo.name
  column "id" {
    type = "integer"
  }
  column "text" {
    type = "string"
  }
  primary_key {
    columns = [
      table.tasks.column.id,
    ]
  }
  index "idx_text" {
    columns = [
      table.tasks.column.text
    ]
    unique = true
  }
}

table "group_vals" {
  schema = schema.todo.name
  column "id" {
    type = "integer"
  }
  column "group_id" {
    type = "integer"
  }
  column "value" {
    type = "string"
  }
  primary_key {
    columns = [
      table.group_vals.column.id,
      table.group_vals.column.group_id,
    ]
  }
}

table "resources" {
  schema = schema.todo.name
  column "id" {
    type = "integer"
  }
  column "task_id" {
    type = "integer"
  }
  column "url" {
    type = "string"
  }
  foreign_key "resource_task" {
    columns = [
      table.resources.column.task_id,
    ]
    references = [
      table.tasks.column.id,
    ]
    on_delete = reference_option.CASCADE
  }
}