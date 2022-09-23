package datasources_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/IvanOfThings/terraform-provider-clickhouse/pks/testutils"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var databaseName = "system"
var comment = ""

func TestAccDataSourceDbs(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		PreCheck: func() { testutils.TestAccPreCheck(t) },
		// ProviderFactories: providerFactories,
		Providers: testutils.Provider(),
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

		dbIdx := -1
		for i := 0; i < nDbs; i++ {
			idxName := fmt.Sprintf("dbs.%d.db_name", i)
			if instanceState.Attributes[idxName] == databaseName {
				dbIdx = i
				break
			}
		}
		if dbIdx == -1 {
			return fmt.Errorf("database %s not found", databaseName)
		}

		idxComment := fmt.Sprintf("databases.%d.comment", dbIdx)
		if instanceState.Attributes[idxComment] != comment {
			return fmt.Errorf("expected comment '%s', got '%s'", comment, instanceState.Attributes[idxComment])
		}
		idxEngine := fmt.Sprintf("dbs.%d.engine", dbIdx)
		if instanceState.Attributes[idxEngine] == "" {
			return fmt.Errorf("expected 'engine' to be set")
		}
		idxDataPath := fmt.Sprintf("dbs.%d.data_path", dbIdx)
		if instanceState.Attributes[idxDataPath] == "" {
			return fmt.Errorf("expected 'data_path' to be set")
		}
		idxMetadataPath := fmt.Sprintf("dbs.%d.metadata_path", dbIdx)
		if instanceState.Attributes[idxMetadataPath] == "" {
			return fmt.Errorf("expected 'metadata_path' to be set")
		}
		idxCurrent := fmt.Sprintf("dbs.%d.uuid", dbIdx)
		if instanceState.Attributes[idxCurrent] == "" {
			return fmt.Errorf("expected 'uuid' to be set")
		}
		return nil
	}
}

const testAccDataSourceDbs = `
data "clickhouse_dbs" "this" {}`
