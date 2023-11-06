package user_test

import (
	"fmt"
	"github.com/IvanOfThings/terraform-provider-clickhouse/pkg/common"
	"github.com/IvanOfThings/terraform-provider-clickhouse/pkg/resources/role"
	"github.com/IvanOfThings/terraform-provider-clickhouse/pkg/resources/user"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"regexp"
	"strings"
	"testing"

	"github.com/IvanOfThings/terraform-provider-clickhouse/pkg/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

type TestStepData struct {
	userName string
	password string
	roles    []string
}

const userResourceName = "test_user"
const userResource = "clickhouse_user." + userResourceName
const password1 = "test_password_1"
const password2 = "test_password_2"
const userName1 = "test_user_1"
const userName2 = "test_user_2"
const roleName1 = "test_role_1"
const roleName2 = "test_role_2"
const roleName3 = "test_role_3"

var testStepsData = []TestStepData{
	{
		// Create user
		userName: userName1,
		password: password1,
		roles: []string{
			roleName1,
			roleName2,
		},
	},
	{
		// Update user password
		userName: userName1,
		password: password2,
		roles: []string{
			roleName1,
			roleName2,
		},
	},
	{
		// Update roles
		userName: userName1,
		password: password1,
		roles: []string{
			roleName1,
			roleName3,
		},
	},
	{
		// Update user name
		userName: userName2,
		password: password1,
		roles: []string{
			roleName1,
			roleName3,
		},
	},
	{
		// Update all attributes
		userName: userName1,
		password: password2,
		roles: []string{
			roleName2,
		},
	},
}

func generateTestSteps() []resource.TestStep {
	var testSteps []resource.TestStep
	for _, testStepData := range testStepsData {
		testSteps = append(testSteps, resource.TestStep{
			Config: testAccUserResource(
				testStepData.userName,
				testStepData.password,
				testStepData.roles,
			),
			Check: resource.ComposeTestCheckFunc(
				resource.TestMatchResourceAttr(
					userResource,
					"name",
					regexp.MustCompile(testStepData.userName),
				),
				resource.TestMatchResourceAttr(
					userResource,
					"password",
					regexp.MustCompile(testStepData.password),
				),
				testutils.CheckStateSetAttr("roles", userResource, testStepData.roles),
				testAccCheckUserResourceExists(testStepData.userName, testStepData.roles),
			),
		})
	}
	return testSteps
}

func TestAccResourceRole(t *testing.T) {
	// Feature tests
	resource.Test(t, resource.TestCase{
		Providers:    testutils.Provider(),
		CheckDestroy: testAccCheckRoleResourceDestroy([]string{userName1, userName2}),
		Steps:        generateTestSteps(),
	})
}

func testAccUserResource(userName string, password string, roles []string) string {
	databaseResource := fmt.Sprintf(`
	resource "clickhouse_db" "test_db" {
		name = "test_db"
		comment = "db comment"
	}
	resource "clickhouse_role" "%[1]s" {
		name = "%[1]s"
		database = clickhouse_db.test_db.name
		privileges = ["INSERT"]
	}
	resource "clickhouse_role" "%[2]s" {
		name = "%[2]s"
		database = clickhouse_db.test_db.name
		privileges = ["SELECT"]
	}
	resource "clickhouse_role" "%[3]s" {
		name = "%[3]s"
		database = clickhouse_db.test_db.name
		privileges = ["SELECT"]
	}
`, roleName1, roleName2, roleName3)

	roleResourceRefs := make([]string, len(roles))
	for i, role := range roles {
		roleResourceRefs[i] = fmt.Sprintf(`clickhouse_role.%s.name`, role)
	}

	userResourceStr := fmt.Sprintf(`
	resource "clickhouse_user" "test_user" {
		name = "%[1]s"
		password = "%[2]s"
		roles = [%[3]s]
	}
`, userName, password, strings.Join(roleResourceRefs, ","))

	return fmt.Sprintf("%s%s", databaseResource, userResourceStr)
}

func testAccCheckUserResourceExists(userName string, roles []string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		client := testutils.TestAccProvider.Meta().(*common.ApiClient)
		conn := client.ClickhouseConnection
		chUserService := user.CHUserService{CHConnection: conn}

		dbUser, err := chUserService.GetUser(userName)
		if err != nil {
			return fmt.Errorf("get user: %v", err)
		}
		userResource := dbUser.ToUserResource()

		if err != nil {
			return fmt.Errorf("get user: %v", err)
		}
		if dbUser == nil {
			return fmt.Errorf("user %s not found", userName)
		}

		if len(roles) != len(dbUser.Roles) {
			return fmt.Errorf("role privileges length mismatching between db and state")
		}

		for _, role := range roles {
			if userResource.Roles.Contains(role) == false {
				return fmt.Errorf("user role %s not found in db", role)
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
