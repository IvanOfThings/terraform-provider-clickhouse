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

resource "clickhouse_db" "test_db_clustered" {
  name    = "awesome_database"
  comment = "This is an awesome database"
  cluster = "'{cluster}'"
}


resource "clickhouse_table" "replicated_table" {
  database      = clickhouse_db.test_db_clustered.name
  name          = "replicated_table"
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
  database = clickhouse_db.test_db_clustered.name
  name     = "distributed_table"
  cluster  = "'{cluster}'"
  engine   = "Distributed"
  engine_params = [
    "'{cluster}'",
    clickhouse_db.test_db_clustered.name,
    clickhouse_table.replicated_table.name,
  "rand()"]
}




