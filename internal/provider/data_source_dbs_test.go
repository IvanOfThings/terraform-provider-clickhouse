package clickhouse_provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDataSourceDbs(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		// ProviderFactories: providerFactories,
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceDbs,
				Check: resource.ComposeTestCheckFunc(
					checkDatabases(),
				),
			},
		},
	})
}

func checkDatabases() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState := s.Modules[0].Resources["data.clickhouse_dbs.this"]
		if resourceState == nil {
			return fmt.Errorf("resource not found in state")
		}
		instanceState := resourceState.Primary
		if instanceState == nil {
			return fmt.Errorf("resource has no primary instance")
		}
		if instanceState.ID != "databases_read" {
			return fmt.Errorf("expected ID to be 'databases_read', got %s", instanceState.ID)
		}

		nDbs, err := strconv.Atoi(instanceState.Attributes["dbs.#"])
		if err != nil {
			return fmt.Errorf("expected a number for field 'dbs', got %s", instanceState.Attributes["dbs.#"])
		}
		if nDbs == 0 {
			return fmt.Errorf("expected databases to be greater or equal to 1, got %s", instanceState.Attributes["dbs.#"])
		}
		return nil
	}
}

const testAccDataSourceDbs = `
data "clickhouse_dbs" "this" {}`
