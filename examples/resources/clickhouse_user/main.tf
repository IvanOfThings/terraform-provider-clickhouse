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
}

resource "clickhouse_db" "awesome_database" {
  name    = "awesome_database"
  comment = "This is an awesome database"
}

resource "clickhouse_role" "awesome_role_1" {
  name       = "awesome_role_1"
  database   = clickhouse_db.awesome_database.name
  privileges = ["SELECT"]
}

resource "clickhouse_role" "awesome_role_2" {
  name       = "awesome_role_2"
  database   = clickhouse_db.awesome_database.name
  privileges = ["INSERT"]
}

resource "clickhouse_user" "awesome_user" {
  name     = "awesome_user"
  password = "awesome_user_password"
  roles    = [clickhouse_role.awesome_role_1.name, clickhouse_role.awesome_role_2.name]
}


