terraform {
  required_providers {
    clickhouse = {
      version = "0.1.0"
      source  = "hashicorp.com/edu/clickhouse"
    }
  }
}

provider "clickhouse" {
  port = 8123
}


resource "clickhouse_db" "test_db" {
  db_name = "database_test_3"
  comment = "This is a test database"
  cluster = "'{cluster}'"
}

