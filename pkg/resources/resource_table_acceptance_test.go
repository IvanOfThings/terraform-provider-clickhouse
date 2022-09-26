package resources_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/IvanOfThings/terraform-provider-clickhouse/pkg/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const testResourceTableDatabaseName = "test_database"
const testResourceTableTableName = "replicated_table_test"

func TestAccResourceTable(t *testing.T) {

	resource.UnitTest(t, resource.TestCase{
		PreCheck: func() { testutils.TestAccPreCheck(t) },
		//ProviderFactories: ProviderFactories,
		Providers: testutils.Provider(),
		Steps: []resource.TestStep{
			{
				Config: tableConfigWithName(testResourceTableDatabaseName, testResourceTableTableName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"clickhouse_db.new_db_resource", "db_name", regexp.MustCompile("^"+testResourceTableDatabaseName)),
					resource.TestMatchResourceAttr(
						"clickhouse_db.new_db_resource", "comment", regexp.MustCompile("^this is a comment")),

					resource.TestCheckResourceAttr("clickhouse_table.table", "table_name", testResourceTableTableName),
					resource.TestCheckResourceAttr("clickhouse_table.table", "database", testResourceTableDatabaseName),
					resource.TestCheckNoResourceAttr("clickhouse_table.table", "cluster"),
					resource.TestCheckResourceAttr("clickhouse_table.table", "comment", "This is just a new table"),
					resource.TestCheckResourceAttr("clickhouse_table.table", "engine", "ReplacingMergeTree"),
					resource.TestCheckResourceAttr("clickhouse_table.table", "engine_params.#", "1"),
					resource.TestCheckResourceAttr("clickhouse_table.table", "engine_params.0", "eventTime"),
					resource.TestCheckResourceAttr("clickhouse_table.table", "order_by.#", "1"),
					resource.TestCheckResourceAttr("clickhouse_table.table", "order_by.0", "key"),
					resource.TestCheckResourceAttr("clickhouse_table.table", "column.#", "3"),
					resource.TestCheckResourceAttr("clickhouse_table.table", "column.0.name", "key"),
					resource.TestCheckResourceAttr("clickhouse_table.table", "column.0.type", "Int64"),
					resource.TestCheckResourceAttr("clickhouse_table.table", "column.1.name", "someCol"),
					resource.TestCheckResourceAttr("clickhouse_table.table", "column.1.type", "String"),
					resource.TestCheckResourceAttr("clickhouse_table.table", "column.2.name", "eventTime"),
					resource.TestCheckResourceAttr("clickhouse_table.table", "column.2.type", "DateTime"),
				),
			},
		},
	})
}

func tableConfigWithName(database string, tableName string) string {
	s := `
	resource "clickhouse_db" "new_db_resource" {
		db_name = "%_database_%"
		comment = "this is a comment"
	}
	
	resource "clickhouse_table" "table" {
		database = clickhouse_db.new_db_resource.db_name
		table_name = "%_tableName_%"
		engine = "ReplacingMergeTree"
		engine_params = ["eventTime"]
		order_by = ["key"]
		column  {
			name= "key"
			type= "Int64"
		}
		column {
			name= "someCol"
			type= "String"
		}
		column {
			name= "eventTime"
			type= "DateTime"
		}
		partition_by {
			by = "eventTime"
			partition_function = "toYYYYMM"
		}
		comment = "This is just a new table"
}`

	s = strings.Replace(s, "%_database_%", database, -1)
	s = strings.Replace(s, "%_tableName_%", tableName, -1)
	return s
}
