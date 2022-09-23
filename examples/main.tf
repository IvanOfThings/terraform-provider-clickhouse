terraform {
  required_providers {
    clickhouse = {
      version = "0.1.0"
      source  = "hashicorp.com/edu/clickhouse"
    }
  }
}

provider "clickhouse" {
  port           = 8923
  clickhouse_url = "127.0.0.1"
  username       = "default"
  password       = ""
}

module "databases" {
  source = "./data-sources/clickhouse_dbs"
}


output "databases" {
  value = module.databases.all_dbs
}
