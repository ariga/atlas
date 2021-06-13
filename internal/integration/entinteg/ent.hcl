table "activities" {
  column "id" {
    type = "int"
  }
  column "name" {
    type = "string"
  }
  primary_key {
    columns = [table.activities.column.id, ]
  }
}
table "default_containers" {
  column "id" {
    type = "int"
  }
  column "stringdef" {
    type    = "string"
    default = "\"default\""
  }
  column "int" {
    type    = "int"
    default = "1"
  }
  column "bool" {
    type    = "boolean"
    default = "true"
  }
  column "enum" {
    type    = "enum"
    default = "\"1\""
    values  = ["1", "2", ]
  }
  column "float" {
    type    = "float"
    default = "1.5"
  }
  primary_key {
    columns = [table.default_containers.column.id, ]
  }
}
table "groups" {
  column "id" {
    type = "int"
  }
  primary_key {
    columns = [table.groups.column.id, ]
  }
}
table "users" {
  column "id" {
    type = "int"
  }
  column "name" {
    type = "string"
  }
  column "optional" {
    type = "string"
    null = true
  }
  column "int" {
    type = "int"
  }
  column "uint" {
    type = "uint"
  }
  column "uint64" {
    type = "uint64"
  }
  column "int64" {
    type = "int64"
  }
  column "time" {
    type = "time"
  }
  column "bool" {
    type = "boolean"
  }
  column "enum" {
    type   = "enum"
    values = ["1", "2", "3", ]
  }
  column "named_enum" {
    type   = "enum"
    values = ["1", "2", "3", ]
  }
  column "uuid" {
    type = "binary"
    size = 16
  }
  column "bytes" {
    type = "binary"
  }
  column "group_id" {
    type = "int"
    null = true
  }
  primary_key {
    columns = [table.users.column.id, ]
  }
  index "uuid" {
    unique  = true
    columns = [table.users.column.uuid, ]
  }
  index "user_time" {
    columns = [table.users.column.time, ]
  }
  foreign_key "users_groups_group" {
    columns    = [table.users.column.group_id, ]
    references = [table.groups.column.id, ]
    on_delete  = reference_option.set_null
  }
}
table "user_activities" {
  column "user_id" {
    type = "int"
  }
  column "activity_id" {
    type = "int"
  }
  primary_key {
    columns = []
  }
  foreign_key "user_activities_user_id" {
    columns    = [table.user_activities.column.user_id, ]
    references = [table.users.column.id, ]
    on_delete  = reference_option.cascade
  }
  foreign_key "user_activities_activity_id" {
    columns    = [table.user_activities.column.activity_id, ]
    references = [table.activities.column.id, ]
    on_delete  = reference_option.cascade
  }
}
