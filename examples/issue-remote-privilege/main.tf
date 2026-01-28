terraform {
  required_providers {
    clickhouse = {
      source = "registry.terraform.io/ivanofthings/clickhouse"
    }
  }
}

provider "clickhouse" {
  host     = "127.0.0.1"
  port     = 9000
  username = "default"
  password = ""
}

# This reproduces the original issue:
# Error: resource role update: error granting privileges to role remote:
# code: 62, message: Syntax error: failed at position 24 (ON): ON *.*) TO remote.
# Expected access type

resource "clickhouse_role" "remote" {
  name       = "remote"
  database   = "*"
  privileges = ["REMOTE"]
}
