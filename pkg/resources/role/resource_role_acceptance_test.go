package role_test

import (
	"fmt"
	"github.com/IvanOfThings/terraform-provider-clickhouse/pkg/common"
	"github.com/IvanOfThings/terraform-provider-clickhouse/pkg/resources/role"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"regexp"
	"strings"
	"testing"

	"github.com/IvanOfThings/terraform-provider-clickhouse/pkg/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

type TestStepData struct {
	roleName   string
	database   string
	privileges []string
}

const roleResourceName = "test_role"
const roleResource = "clickhouse_role." + roleResourceName
const roleName1 = "test_role_1"
const roleName2 = "test_role_2"
const databaseName1 = "role_role_db_1"
const databaseName2 = "role_role_db_2"

var testStepsData = []TestStepData{
	{
		// Create role
		roleName: roleName1,
		database: databaseName1,
		privileges: []string{
			"SELECT",
			"INSERT",
		},
	},
	{
		// Remove role privileges
		roleName: roleName1,
		database: databaseName1,
		privileges: []string{
			"SELECT",
		},
	},
	{
		// Remove and add role privileges
		roleName: roleName1,
		database: databaseName1,
		privileges: []string{
			"INSERT",
		},
	},
	{
		// Update db name
		roleName: roleName1,
		database: databaseName2,
		privileges: []string{
			"INSERT",
		},
	},
	{
		// Update db name and privileges at the same time
		roleName: roleName1,
		database: databaseName1,
		privileges: []string{
			"INSERT",
			"ALTER",
		},
	},
	{
		// Check all allowed privileges
		roleName:   roleName1,
		database:   databaseName1,
		privileges: role.AllowedPrivileges,
	},
	{
		// Change role name
		roleName:   roleName2,
		database:   databaseName1,
		privileges: role.AllowedPrivileges,
	},
	{
		// Change role name and db
		roleName:   roleName1,
		database:   databaseName2,
		privileges: role.AllowedPrivileges,
	},
	{
		// Change role name, db and privileges
		roleName: roleName2,
		database: databaseName1,
		privileges: []string{
			"INSERT",
		},
	},
}

func generateTestSteps() []resource.TestStep {
	var testSteps []resource.TestStep
	for _, testStepData := range testStepsData {
		testSteps = append(testSteps, resource.TestStep{
			Config: testAccRoleResource(
				testStepData.roleName,
				testStepData.database,
				common.Quote(testStepData.privileges),
			),
			Check: resource.ComposeTestCheckFunc(
				resource.TestMatchResourceAttr(
					roleResource,
					"name",
					regexp.MustCompile(testStepData.roleName),
				),
				resource.TestMatchResourceAttr(
					roleResource,
					"database",
					regexp.MustCompile(testStepData.database),
				),
				testutils.CheckStateSetAttr("privileges", roleResource, testStepData.privileges),
				testAccCheckRoleResourceExists(testStepData.roleName, testStepData.database, testStepData.privileges),
			),
		})
	}
	return testSteps
}

func TestAccResourceRole(t *testing.T) {
	// Feature tests
	resource.Test(t, resource.TestCase{
		//ProviderFactories: testutils.GetProviderFactories(),
		Providers:    testutils.Provider(),
		CheckDestroy: testAccCheckRoleResourceDestroy([]string{roleName1, roleName2}),
		Steps:        generateTestSteps(),
	})
	// Validate privileges on create
	resource.Test(t, resource.TestCase{
		Providers: testutils.Provider(),
		Steps: []resource.TestStep{
			{
				Config: testAccRoleResource(
					roleName1,
					databaseName1,
					common.Quote([]string{"NOT_ALLOWED_PRIVILEGE"}),
				),
				ExpectError: regexp.MustCompile("NOT_ALLOWED_PRIVILEGE isn't in the allowed privileges list"),
			},
		},
	})
	// Validate privileges on update
	resource.Test(t, resource.TestCase{
		Providers:    testutils.Provider(),
		CheckDestroy: testAccCheckRoleResourceDestroy([]string{roleName1}),
		Steps: []resource.TestStep{
			{
				Config: testAccRoleResource(
					roleName1,
					databaseName1,
					common.Quote([]string{"SELECT"}),
				),
			},
			{
				Config: testAccRoleResource(
					roleName1,
					databaseName1,
					common.Quote([]string{"NOT_ALLOWED_PRIVILEGE"}),
				),
				ExpectError: regexp.MustCompile("NOT_ALLOWED_PRIVILEGE isn't in the allowed privileges list"),
			},
		},
	})
}

func testAccRoleResource(roleName string, database string, privileges []string) string {
	databaseComment := "db comment"
	databaseResource := fmt.Sprintf(`
	resource "clickhouse_db" "%[1]s" {
		name = "%[1]s"
		comment = "[3]%s"
	}

	resource "clickhouse_db" "%[2]s" {
		name = "%[2]s"
		comment = "%[3]s"
	}
`, databaseName1, databaseName2, databaseComment)

	roleResource := fmt.Sprintf(`
	resource "clickhouse_role" "test_role" {
		name = "%[1]s"
		database = clickhouse_db.%s.name
		privileges = [%s]
	}
`, roleName, database, strings.Join(privileges, ","))

	return fmt.Sprintf("%s\n%s", databaseResource, roleResource)
}

func testAccCheckRoleResourceExists(roleName string, database string, privileges []string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		client := testutils.TestAccProvider.Meta().(*common.ApiClient)
		conn := client.ClickhouseConnection
		chRoleService := role.CHRoleService{CHConnection: conn}

		dbRole, err := chRoleService.GetRole(roleName)

		if err != nil {
			return fmt.Errorf("get role: %v", err)
		}
		if dbRole == nil {
			return fmt.Errorf("role %s not found", roleName)
		}

		if len(privileges) != len(dbRole.Privileges) {
			return fmt.Errorf("role privileges length mismatching between db and state")
		}

		for _, privilege := range privileges {
			var matchedDbRolePrivilege *role.CHGrant
			for _, dbRolePrivilege := range dbRole.Privileges {
				if privilege == dbRolePrivilege.AccessType {
					matchedDbRolePrivilege = &dbRolePrivilege
					break
				}
			}
			if matchedDbRolePrivilege == nil {
				return fmt.Errorf("role privilege %s not found in db", privilege)
			}
			if matchedDbRolePrivilege.Database != database {
				return fmt.Errorf("role privilege %s database mismatching between db and state", privilege)
			}
		}

		return nil
	}
}

func testAccCheckRoleResourceDestroy(roleNames []string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, roleName := range roleNames {
			client := testutils.TestAccProvider.Meta().(*common.ApiClient)
			conn := client.ClickhouseConnection
			chRoleService := role.CHRoleService{CHConnection: conn}

			dbRole, err := chRoleService.GetRole(roleName)

			if err != nil {
				return fmt.Errorf("get role: %v", err)
			}

			if dbRole != nil {
				return fmt.Errorf("role %s hasn't been deleted", roleName)
			}
		}
		return nil
	}
}
