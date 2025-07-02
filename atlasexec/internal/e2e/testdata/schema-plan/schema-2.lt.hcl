schema "public" {
  comment = "This is a test schema"
}

table "t1" {
  schema = schema.public
  column "c1" {
    type = bigint
  }
  column "c2" {
    type = text
  }
}
