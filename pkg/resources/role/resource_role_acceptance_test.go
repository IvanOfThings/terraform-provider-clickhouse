package resourcerole_test

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/IvanOfThings/terraform-provider-clickhouse/pkg/common"
	resourcerole "github.com/IvanOfThings/terraform-provider-clickhouse/pkg/resources/role"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

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

var test1StepsData = []TestStepData{
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
		privileges: resourcerole.AllowedDbLevelPrivileges,
	},
	{
		// Change role name
		roleName:   roleName2,
		database:   databaseName1,
		privileges: resourcerole.AllowedDbLevelPrivileges,
	},
	{
		// Change role name and db
		roleName:   roleName1,
		database:   databaseName2,
		privileges: resourcerole.AllowedDbLevelPrivileges,
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

var test2StepsData = []TestStepData{
	{
		// Create role
		roleName: roleName1,
		database: "system",
		privileges: []string{
			"SELECT",
			"INSERT",
		},
	},
	{
		// Remove role privileges
		roleName: roleName1,
		database: "system",
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
		// Check all allowed privileges
		roleName:   roleName1,
		database:   "system",
		privileges: resourcerole.AllowedDbLevelPrivileges,
	},
	{
		// Change role name
		roleName:   roleName2,
		database:   "system",
		privileges: resourcerole.AllowedDbLevelPrivileges,
	},
}

func generateTestSteps(testStepsData []TestStepData) []resource.TestStep {
	var testSteps []resource.TestStep
	for _, testStepData := range testStepsData {
		var databaseRegex *regexp.Regexp
		if testStepData.database == "*" {
			databaseRegex = regexp.MustCompile("\\*")
		} else {
			databaseRegex = regexp.MustCompile(testStepData.database)
		}
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
					databaseRegex,
				),
				testutils.CheckStateSetAttr("privileges", roleResource, testStepData.privileges),
				testAccCheckRoleResourceExists(testStepData.roleName, testStepData.database, testStepData.privileges),
			),
		})
	}
	return testSteps
}

// TestAccResourceRole_ExpandablePrivileges tests expandable SOURCE privileges
// that get expanded by ClickHouse into READ + WRITE variants.
// This test ensures that the provider correctly handles the expansion and
// doesn't detect false state drift.
//
// Expandable privileges include: REMOTE, S3, AZURE, HDFS, URL, MYSQL, POSTGRES, MONGO, KAFKA
// See: https://clickhouse.com/docs/sql-reference/statements/grant#privileges
func TestAccResourceRole_ExpandablePrivileges(t *testing.T) {
	// Test each expandable privilege individually
	expandablePrivileges := []string{"REMOTE", "S3", "AZURE", "HDFS", "URL", "MYSQL", "POSTGRES", "MONGO", "KAFKA"}

	for _, privilege := range expandablePrivileges {
		t.Run(privilege, func(t *testing.T) {
			roleName := fmt.Sprintf("test_%s_role", strings.ToLower(privilege))

			resource.Test(t, resource.TestCase{
				Providers:    testutils.Provider(),
				CheckDestroy: testAccCheckRoleResourceDestroy([]string{roleName}),
				Steps: []resource.TestStep{
					{
						// Step 1: Create role with expandable privilege
						Config: fmt.Sprintf(`
resource "clickhouse_role" "expandable_test" {
	name       = "%s"
	database   = "*"
	privileges = ["%s"]
}`, roleName, privilege),
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr(
								"clickhouse_role.expandable_test",
								"name",
								roleName,
							),
							resource.TestCheckResourceAttr(
								"clickhouse_role.expandable_test",
								"database",
								"*",
							),
							// Verify state contains the original privilege (not expanded form)
							testutils.CheckStateSetAttr("privileges", "clickhouse_role.expandable_test", []string{privilege}),
						),
					},
					{
						// Step 2: Re-apply same configuration - should be no-op
						Config: fmt.Sprintf(`
resource "clickhouse_role" "expandable_test" {
	name       = "%s"
	database   = "*"
	privileges = ["%s"]
}`, roleName, privilege),
						// No changes should be detected
						ExpectNonEmptyPlan: false,
						Check: resource.ComposeTestCheckFunc(
							// Verify privileges are still in original form
							testutils.CheckStateSetAttr("privileges", "clickhouse_role.expandable_test", []string{privilege}),
						),
					},
				},
			})
		})
	}
}

// TestAccResourceRole_MultipleExpandablePrivileges tests multiple expandable privileges together
func TestAccResourceRole_MultipleExpandablePrivileges(t *testing.T) {
	const roleName = "test_multiple_expandable"

	resource.Test(t, resource.TestCase{
		Providers:    testutils.Provider(),
		CheckDestroy: testAccCheckRoleResourceDestroy([]string{roleName}),
		Steps: []resource.TestStep{
			{
				// Create role with multiple expandable privileges
				Config: fmt.Sprintf(`
resource "clickhouse_role" "multi_expandable" {
	name       = "%s"
	database   = "*"
	privileges = ["REMOTE", "S3", "HDFS"]
}`, roleName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"clickhouse_role.multi_expandable",
						"name",
						roleName,
					),
					testutils.CheckStateSetAttr("privileges", "clickhouse_role.multi_expandable", []string{"REMOTE", "S3", "HDFS"}),
				),
			},
			{
				// Re-apply - should be no-op
				Config: fmt.Sprintf(`
resource "clickhouse_role" "multi_expandable" {
	name       = "%s"
	database   = "*"
	privileges = ["REMOTE", "S3", "HDFS"]
}`, roleName),
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// TestAccResourceRole_MixedExpandableAndNonExpandable tests mixing expandable and non-expandable privileges
func TestAccResourceRole_MixedExpandableAndNonExpandable(t *testing.T) {
	const roleName = "test_mixed_privileges"

	resource.Test(t, resource.TestCase{
		Providers:    testutils.Provider(),
		CheckDestroy: testAccCheckRoleResourceDestroy([]string{roleName}),
		Steps: []resource.TestStep{
			{
				// Create role with both expandable (REMOTE, S3) and non-expandable (SYSTEM FLUSH LOGS) privileges
				Config: fmt.Sprintf(`
resource "clickhouse_role" "mixed_test" {
	name       = "%s"
	database   = "*"
	privileges = ["REMOTE", "S3", "SYSTEM FLUSH LOGS"]
}`, roleName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"clickhouse_role.mixed_test",
						"name",
						roleName,
					),
					testutils.CheckStateSetAttr("privileges", "clickhouse_role.mixed_test", []string{"REMOTE", "S3", "SYSTEM FLUSH LOGS"}),
				),
			},
			{
				// Re-apply - should be no-op
				Config: fmt.Sprintf(`
resource "clickhouse_role" "mixed_test" {
	name       = "%s"
	database   = "*"
	privileges = ["REMOTE", "S3", "SYSTEM FLUSH LOGS"]
}`, roleName),
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// TestAccResourceRole_ComplexGlobalPrivilegeMix tests complex combinations of multiple global privileges
func TestAccResourceRole_ComplexGlobalPrivilegeMix(t *testing.T) {
	const roleName = "test_complex_mix"

	resource.Test(t, resource.TestCase{
		Providers:    testutils.Provider(),
		CheckDestroy: testAccCheckRoleResourceDestroy([]string{roleName}),
		Steps: []resource.TestStep{
			{
				// Create role with complex mix: multiple expandable + multiple non-expandable
				Config: fmt.Sprintf(`
resource "clickhouse_role" "complex_mix" {
	name       = "%s"
	database   = "*"
	privileges = ["REMOTE", "S3", "AZURE", "HDFS", "SYSTEM FLUSH LOGS", "SYSTEM RELOAD DICTIONARY", "CREATE TEMPORARY TABLE", "CREATE FUNCTION"]
}`, roleName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"clickhouse_role.complex_mix",
						"name",
						roleName,
					),
					testutils.CheckStateSetAttr("privileges", "clickhouse_role.complex_mix",
						[]string{"REMOTE", "S3", "AZURE", "HDFS", "SYSTEM FLUSH LOGS", "SYSTEM RELOAD DICTIONARY", "CREATE TEMPORARY TABLE", "CREATE FUNCTION"}),
				),
			},
			{
				// Re-apply cycle 1 - should be no-op
				Config: fmt.Sprintf(`
resource "clickhouse_role" "complex_mix" {
	name       = "%s"
	database   = "*"
	privileges = ["REMOTE", "S3", "AZURE", "HDFS", "SYSTEM FLUSH LOGS", "SYSTEM RELOAD DICTIONARY", "CREATE TEMPORARY TABLE", "CREATE FUNCTION"]
}`, roleName),
				ExpectNonEmptyPlan: false,
			},
			{
				// Re-apply cycle 2 - should still be no-op
				Config: fmt.Sprintf(`
resource "clickhouse_role" "complex_mix" {
	name       = "%s"
	database   = "*"
	privileges = ["REMOTE", "S3", "AZURE", "HDFS", "SYSTEM FLUSH LOGS", "SYSTEM RELOAD DICTIONARY", "CREATE TEMPORARY TABLE", "CREATE FUNCTION"]
}`, roleName),
				ExpectNonEmptyPlan: false,
			},
			{
				// Re-apply cycle 3 - should still be no-op (testing long-term stability)
				Config: fmt.Sprintf(`
resource "clickhouse_role" "complex_mix" {
	name       = "%s"
	database   = "*"
	privileges = ["REMOTE", "S3", "AZURE", "HDFS", "SYSTEM FLUSH LOGS", "SYSTEM RELOAD DICTIONARY", "CREATE TEMPORARY TABLE", "CREATE FUNCTION"]
}`, roleName),
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// TestAccResourceRole_MultipleApplyCycles tests that privileges remain stable across multiple apply cycles
func TestAccResourceRole_MultipleApplyCycles(t *testing.T) {
	const roleName = "test_multiple_cycles"

	// Test with different privilege combinations
	testCases := []struct {
		name       string
		privileges []string
	}{
		{
			name:       "single_expandable",
			privileges: []string{"REMOTE"},
		},
		{
			name:       "multiple_expandable",
			privileges: []string{"REMOTE", "S3", "MYSQL"},
		},
		{
			name:       "mixed_privileges",
			privileges: []string{"REMOTE", "S3", "SYSTEM FLUSH LOGS", "CREATE FUNCTION"},
		},
		{
			name:       "all_expandable",
			privileges: []string{"REMOTE", "S3", "AZURE", "HDFS", "URL", "MYSQL", "POSTGRES", "MONGO", "KAFKA"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testRoleName := fmt.Sprintf("%s_%s", roleName, tc.name)
			privilegesStr := fmt.Sprintf(`["%s"]`, strings.Join(tc.privileges, `", "`))

			var steps []resource.TestStep

			// Initial creation
			steps = append(steps, resource.TestStep{
				Config: fmt.Sprintf(`
resource "clickhouse_role" "cycle_test" {
	name       = "%s"
	database   = "*"
	privileges = %s
}`, testRoleName, privilegesStr),
				Check: testutils.CheckStateSetAttr("privileges", "clickhouse_role.cycle_test", tc.privileges),
			})

			// Add 4 re-apply cycles to ensure stability
			for i := 1; i <= 4; i++ {
				steps = append(steps, resource.TestStep{
					Config: fmt.Sprintf(`
resource "clickhouse_role" "cycle_test" {
	name       = "%s"
	database   = "*"
	privileges = %s
}`, testRoleName, privilegesStr),
					ExpectNonEmptyPlan: false,
				})
			}

			resource.Test(t, resource.TestCase{
				Providers:    testutils.Provider(),
				CheckDestroy: testAccCheckRoleResourceDestroy([]string{testRoleName}),
				Steps:        steps,
			})
		})
	}
}

// TestAccResourceRole_PrivilegeUpdates tests updating privileges from one set to another
func TestAccResourceRole_PrivilegeUpdates(t *testing.T) {
	const roleName = "test_privilege_updates"

	resource.Test(t, resource.TestCase{
		Providers:    testutils.Provider(),
		CheckDestroy: testAccCheckRoleResourceDestroy([]string{roleName}),
		Steps: []resource.TestStep{
			{
				// Start with REMOTE
				Config: fmt.Sprintf(`
resource "clickhouse_role" "update_test" {
	name       = "%s"
	database   = "*"
	privileges = ["REMOTE"]
}`, roleName),
				Check: testutils.CheckStateSetAttr("privileges", "clickhouse_role.update_test", []string{"REMOTE"}),
			},
			{
				// Update to S3
				Config: fmt.Sprintf(`
resource "clickhouse_role" "update_test" {
	name       = "%s"
	database   = "*"
	privileges = ["S3"]
}`, roleName),
				Check: testutils.CheckStateSetAttr("privileges", "clickhouse_role.update_test", []string{"S3"}),
			},
			{
				// Re-apply S3 - should be no-op
				Config: fmt.Sprintf(`
resource "clickhouse_role" "update_test" {
	name       = "%s"
	database   = "*"
	privileges = ["S3"]
}`, roleName),
				ExpectNonEmptyPlan: false,
			},
			{
				// Update to multiple expandable
				Config: fmt.Sprintf(`
resource "clickhouse_role" "update_test" {
	name       = "%s"
	database   = "*"
	privileges = ["REMOTE", "S3", "AZURE"]
}`, roleName),
				Check: testutils.CheckStateSetAttr("privileges", "clickhouse_role.update_test", []string{"REMOTE", "S3", "AZURE"}),
			},
			{
				// Re-apply multiple - should be no-op
				Config: fmt.Sprintf(`
resource "clickhouse_role" "update_test" {
	name       = "%s"
	database   = "*"
	privileges = ["REMOTE", "S3", "AZURE"]
}`, roleName),
				ExpectNonEmptyPlan: false,
			},
			{
				// Update to mixed expandable + non-expandable
				Config: fmt.Sprintf(`
resource "clickhouse_role" "update_test" {
	name       = "%s"
	database   = "*"
	privileges = ["REMOTE", "SYSTEM FLUSH LOGS", "CREATE FUNCTION"]
}`, roleName),
				Check: testutils.CheckStateSetAttr("privileges", "clickhouse_role.update_test", []string{"REMOTE", "SYSTEM FLUSH LOGS", "CREATE FUNCTION"}),
			},
			{
				// Re-apply mixed - should be no-op
				Config: fmt.Sprintf(`
resource "clickhouse_role" "update_test" {
	name       = "%s"
	database   = "*"
	privileges = ["REMOTE", "SYSTEM FLUSH LOGS", "CREATE FUNCTION"]
}`, roleName),
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccResourceRole(t *testing.T) {
	// Feature tests, user database
	resource.Test(t, resource.TestCase{
		//ProviderFactories: testutils.GetProviderFactories(),
		Providers:    testutils.Provider(),
		CheckDestroy: testAccCheckRoleResourceDestroy([]string{roleName1, roleName2}),
		Steps:        generateTestSteps(test1StepsData),
	})
	// Feature tests, system database
	resource.Test(t, resource.TestCase{
		Providers:    testutils.Provider(),
		CheckDestroy: testAccCheckRoleResourceDestroy([]string{roleName1, roleName2}),
		Steps:        generateTestSteps(test2StepsData),
	})
	// Feature tests, global privileges (including expandable privileges like REMOTE, S3, etc.)
	resource.Test(t, resource.TestCase{
		Providers:    testutils.Provider(),
		CheckDestroy: testAccCheckRoleResourceDestroy([]string{roleName1, roleName2}),
		Steps: generateTestSteps([]TestStepData{
			{
				// Create role
				roleName:   roleName1,
				database:   "*",
				privileges: resourcerole.AllowedGlobalPrivileges,
			}}),
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
				ExpectError: regexp.MustCompile("NOT_ALLOWED_PRIVILEGE is not in the allowed privileges list"),
			},
		},
	})
	resource.Test(t, resource.TestCase{
		Providers: testutils.Provider(),
		Steps: []resource.TestStep{
			{
				Config: testAccRoleResource(
					roleName1,
					databaseName1,
					common.Quote([]string{"REMOTE"}),
				),
				ExpectError: regexp.MustCompile("Global privilege REMOTE is only allowed for database '\\*'"),
			},
		},
	})
	resource.Test(t, resource.TestCase{
		Providers: testutils.Provider(),
		Steps: []resource.TestStep{
			{
				Config: testAccRoleResource(
					roleName1,
					databaseName1,
					common.Quote([]string{"SYSTEM RELOAD DICTIONARY"}),
				),
				ExpectError: regexp.MustCompile("Global privilege SYSTEM RELOAD DICTIONARY is only allowed for database '\\*'"),
			},
		},
	})
	resource.Test(t, resource.TestCase{
		Providers: testutils.Provider(),
		Steps: []resource.TestStep{
			{
				Config: testAccRoleResource(
					roleName1,
					databaseName1,
					common.Quote([]string{"SYSTEM FLUSH LOGS"}),
				),
				ExpectError: regexp.MustCompile("Global privilege SYSTEM FLUSH LOGS is only allowed for database '\\*'"),
			},
		},
	})
	resource.Test(t, resource.TestCase{
		Providers: testutils.Provider(),
		Steps: []resource.TestStep{
			{
				Config: testAccRoleResource(
					roleName1,
					databaseName1,
					common.Quote([]string{"S3"}),
				),
				ExpectError: regexp.MustCompile("Global privilege S3 is only allowed for database '\\*'"),
			},
		},
	})
	resource.Test(t, resource.TestCase{
		Providers: testutils.Provider(),
		Steps: []resource.TestStep{
			{
				Config: testAccRoleResource(
					roleName1,
					databaseName1,
					common.Quote([]string{"CREATE TEMPORARY TABLE"}),
				),
				ExpectError: regexp.MustCompile("Global privilege CREATE TEMPORARY TABLE is only allowed for database '\\*'"),
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
				ExpectError: regexp.MustCompile("NOT_ALLOWED_PRIVILEGE is not in the allowed privileges list"),
			},
		},
	})
}

func testAccRoleResource(roleName string, database string, privileges []string) string {
	if database == "system" {
		return fmt.Sprintf(`
	resource "clickhouse_role" "test_role" {
		name = "%s"
		database = "system"
		privileges = [%s]
	}`, roleName, strings.Join(privileges, ","))
	}

	if database == "*" {
		return fmt.Sprintf(`
	resource "clickhouse_role" "test_role" {
		name = "%s"
		database = "*"
		privileges = [%s]
	}`, roleName, strings.Join(privileges, ","))
	}

	databaseComment := "db comment"
	databaseResource := fmt.Sprintf(`
	resource "clickhouse_db" "%[1]s" {
		name = "%[1]s"
		comment = "%[3]s"
	}

	resource "clickhouse_db" "%[2]s" {
		name = "%[2]s"
		comment = "%[3]s"
	}
`, databaseName1, databaseName2, databaseComment)

	roleResource := fmt.Sprintf(`
	resource "clickhouse_role" "test_role" {
		name = "%s"
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
		chRoleService := resourcerole.CHRoleService{CHConnection: conn}

		dbRole, err := chRoleService.GetRole(context.Background(), roleName)

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
			var matchedDbRolePrivilege *resourcerole.CHGrant
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
			chRoleService := resourcerole.CHRoleService{CHConnection: conn}

			dbRole, err := chRoleService.GetRole(context.Background(), roleName)

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
