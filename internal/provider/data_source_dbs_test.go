package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceDbs(t *testing.T) {

	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceDbs,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"data.clickhouse_dbs.this", "dbs", regexp.MustCompile("[]")),
				),
			},
		},
	})
}

const testAccDataSourceDbs = `
data "clickhouse_dbs" "this" {}`
