terraform {
  required_providers {
    clickhouse = {
      version = "2.0.0"
      source  = "hashicorp.com/ivanofthings/clickhouse"
    }
  }
}

provider "clickhouse" {
  port = 8123
}


resource "clickhouse_db" "test_db" {
  name    = "database_test"
  comment = "This is a test database"
}


// '{cluster}' is tha way to refer cluster name using macros provided by altinity clickhouse ok8s operator for clickhouse
// Should provide cluster name in case of using this into a clickhouse cluster
// see: https://github.com/Altinity/clickhouse-operator/blob/master/docs/replication_setup.md
resource "clickhouse_db" "test_db_clusterd" {
  name    = "clustered_test_database"
  comment = "This is a clustered test database"
  cluster = "'{cluster}'"
}

