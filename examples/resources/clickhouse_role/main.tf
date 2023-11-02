terraform {
  required_providers {
    clickhouse = {
      version = "2.0.0"
      source  = "hashicorp.com/ivanofthings/clickhouse"
    }
  }
}

provider "clickhouse" {
  port     = 8123
  host     = "127.0.0.1"
  username = "root"
  password = "root"
}

resource "clickhouse_db" "awesome_database" {
  name    = "awesome_database"
  comment = "This is an awesome database"
}

resource "clickhouse_role" "awesome_role" {
  name       = "awesome_role"
  database   = clickhouse_db.awesome_database.name
  privileges = ["INSERT"]
}


