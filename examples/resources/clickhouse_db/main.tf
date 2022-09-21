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

/*
resource "clickhouse_db" "test_db" {
  db_name = "database_test_3"
  comment = "This is a test database"
}
*/

// '{cluster}' is tha way to refer cluster name using macros provided by altinity clickhouse ok8s operator for clickhouse
// Should provide cluster name in case of using this into a clickhouse cluster
// see: https://github.com/Altinity/clickhouse-operator/blob/master/docs/replication_setup.md
resource "clickhouse_db" "test_db_clusterd" {
  db_name = "database_test_clustered_ojete"
  comment = "This is a test database"
  cluster = "'{cluster}'"
}

