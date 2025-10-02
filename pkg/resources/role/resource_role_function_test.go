package resourcerole_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/IvanOfThings/terraform-provider-clickhouse/pkg/common"
	resourcerole "github.com/IvanOfThings/terraform-provider-clickhouse/pkg/resources/role"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/IvanOfThings/terraform-provider-clickhouse/pkg/testutils"
)

const functionRoleName = "test_function_role"
const functionRoleResource = "clickhouse_role.test_function_role"

func TestAccResourceRole_FunctionPrivileges(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    testutils.Provider(),
		CheckDestroy: testAccCheckRoleResourceDestroy([]string{functionRoleName}),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "clickhouse_role" "test_function_role" {
						name       = "test_function_role"
						database   = "*"
						privileges = ["CREATE FUNCTION", "DROP FUNCTION"]
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						functionRoleResource,
						"name",
						regexp.MustCompile(functionRoleName),
					),
					resource.TestMatchResourceAttr(
						functionRoleResource,
						"database",
						regexp.MustCompile("\\*"),
					),
					testutils.CheckStateSetAttr("privileges", functionRoleResource, []string{"CREATE FUNCTION", "DROP FUNCTION"}),
					testAccCheckFunctionPrivilegesExist(functionRoleName, []string{"CREATE FUNCTION", "DROP FUNCTION"}),
				),
			},
		},
	})
}

func TestAccResourceRole_ValidationCreateFunction(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: testutils.Provider(),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "clickhouse_db" "test_validation_db" {
						name    = "test_validation_db"
						comment = "test db"
					}

					resource "clickhouse_role" "test_role" {
						name       = "test_role"
						database   = clickhouse_db.test_validation_db.name
						privileges = ["CREATE FUNCTION"]
					}
				`,
				ExpectError: regexp.MustCompile("Global privilege CREATE FUNCTION is only allowed for database '\\*'"),
			},
		},
	})
}

func TestAccResourceRole_ValidationDropFunction(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: testutils.Provider(),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "clickhouse_db" "test_validation_db" {
						name    = "test_validation_db"
						comment = "test db"
					}

					resource "clickhouse_role" "test_role" {
						name       = "test_role"
						database   = clickhouse_db.test_validation_db.name
						privileges = ["DROP FUNCTION"]
					}
				`,
				ExpectError: regexp.MustCompile("Global privilege DROP FUNCTION is only allowed for database '\\*'"),
			},
		},
	})
}

func testAccCheckFunctionPrivilegesExist(roleName string, expectedPrivileges []string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		client := testutils.TestAccProvider.Meta().(*common.ApiClient)
		conn := client.ClickhouseConnection
		chRoleService := resourcerole.CHRoleService{CHConnection: conn}

		dbRole, err := chRoleService.GetRole(context.Background(), roleName)
		if err != nil {
			return fmt.Errorf("get role: %v", err)
		}
		if dbRole == nil {
			return fmt.Errorf("role %s not found", roleName)
		}

		if len(expectedPrivileges) != len(dbRole.Privileges) {
			return fmt.Errorf("expected %d privileges, got %d", len(expectedPrivileges), len(dbRole.Privileges))
		}

		privilegeMap := make(map[string]bool)
		for _, priv := range dbRole.Privileges {
			privilegeMap[priv.AccessType] = true
		}

		for _, expected := range expectedPrivileges {
			if !privilegeMap[expected] {
				return fmt.Errorf("privilege %s not found in role", expected)
			}
		}

		return nil
	}
}
