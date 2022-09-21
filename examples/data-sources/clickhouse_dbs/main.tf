terraform {
  required_providers {
    clickhouse = {
      version = "0.1.0"
      source  = "hashicorp.com/edu/clickhouse"
    }
  }
}


data "clickhouse_dbs" "this" {}

output "all_dbs" {
  value = data.clickhouse_dbs.this.dbs
}
