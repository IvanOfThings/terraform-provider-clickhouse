
resource "clickhouse_db" "vulnerabilities_db" {
  db_name = "vulnerabilities_test"
  comment = "This is a vulnerabilities database"
}
