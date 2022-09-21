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

resource "clickhouse_db" "test_db_clustered" {
  db_name = "database_test_10"
  comment = "This is a test database"
  cluster = "'{cluster}'"
}



resource "clickhouse_table" "replicated_table" {
  database      = clickhouse_db.test_db_clustered.db_name
  table_name    = "replicated_table"
  cluster       = "'{cluster}'"
  engine        = "ReplicatedMergeTree"
  engine_params = ["'/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}'", "'{replica}'"]
  order_by      = ["event_date", "event_type"]
  columns {
    name = "event_date"
    type = "Date"
  }
  columns {
    name = "event_type"
    type = "Int32"
  }
  columns {
    name = "article_id"
    type = "Int32"
  }
  columns {
    name = "title"
    type = "String"
  }
  partition_by {
    by = "event_type"
  }
  partition_by {
    by                 = "event_date"
    partition_function = "toYYYYMM"
  }
}


resource "clickhouse_table" "distributed_table" {
  database      = clickhouse_db.test_db_clustered.db_name
  table_name    = "t1_dist_6"
  cluster       = "'{cluster}'"
  engine        = "Distributed"
  engine_params = ["'{cluster}'", clickhouse_db.test_db_clustered.db_name, clickhouse_table.replicated_table.table_name, "rand()"]
}



