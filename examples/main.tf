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
  username = "default"
  password = ""
}

module "databases" {
  source = "./data-sources/clickhouse_dbs"
}


output "databases" {
  value = module.databases.all_dbs
}
