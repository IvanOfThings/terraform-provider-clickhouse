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

module "databases" {
  source = "./data-sources/clickhouse_dbs"
}


output "databases" {
  value = module.databases.all_dbs
}
