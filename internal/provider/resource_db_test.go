package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceDb(t *testing.T) {

	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceDb,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"clickhouse_db.new_db", "db_name", regexp.MustCompile("^testing_db")),
					resource.TestMatchResourceAttr(
						"clickhouse_db.new_db", "comment", regexp.MustCompile("^This is a testing database")),
				),
			},
		},
	})
}

const testAccResourceDb = `
resource "clickhouse_db" "new_db" {
  db_name = "testing_db"
  comment = "This is a testing database"
}
`
