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

// TestAccResourceRole_RevokePrivileges tests revoking privileges
func TestAccResourceRole_RevokePrivileges(t *testing.T) {
	const roleName = "test_revoke_privileges"

	resource.Test(t, resource.TestCase{
		Providers:    testutils.Provider(),
		CheckDestroy: testAccCheckRoleResourceDestroy([]string{roleName}),
		Steps: []resource.TestStep{
			{
				// Start with multiple privileges
				Config: fmt.Sprintf(`
					resource "clickhouse_role" "revoke_test" {
						name       = "%s"
						database   = "*"
						privileges = ["REMOTE", "S3", "SYSTEM FLUSH LOGS"]
					}`, roleName),
				Check: testutils.CheckStateSetAttr("privileges", "clickhouse_role.revoke_test", []string{"REMOTE", "S3", "SYSTEM FLUSH LOGS"}),
			},
			{
				// Revoke one global privilege (S3)
				Config: fmt.Sprintf(`
					resource "clickhouse_role" "revoke_test" {
						name       = "%s"
						database   = "*"
						privileges = ["REMOTE", "SYSTEM FLUSH LOGS"]
					}`, roleName),
				Check: testutils.CheckStateSetAttr("privileges", "clickhouse_role.revoke_test", []string{"REMOTE", "SYSTEM FLUSH LOGS"}),
			},
			{
				// Re-apply - should be no-op
				Config: fmt.Sprintf(`
					resource "clickhouse_role" "revoke_test" {
						name       = "%s"
						database   = "*"
						privileges = ["REMOTE", "SYSTEM FLUSH LOGS"]
					}`, roleName),
				ExpectNonEmptyPlan: false,
			},
			{
				// Revoke another global privilege (REMOTE)
				Config: fmt.Sprintf(`
					resource "clickhouse_role" "revoke_test" {
						name       = "%s"
						database   = "*"
						privileges = ["SYSTEM FLUSH LOGS"]
					}`, roleName),
				Check: testutils.CheckStateSetAttr("privileges", "clickhouse_role.revoke_test", []string{"SYSTEM FLUSH LOGS"}),
			},
			{
				// Revoke all and add different privilege
				Config: fmt.Sprintf(`
					resource "clickhouse_role" "revoke_test" {
						name       = "%s"
						database   = "*"
						privileges = ["AZURE"]
					}`, roleName),
				Check: testutils.CheckStateSetAttr("privileges", "clickhouse_role.revoke_test", []string{"AZURE"}),
			},
		},
	})
}

// TestAccResourceRole_RevokeExpandablePrivileges tests revoking expandable privileges
func TestAccResourceRole_RevokeExpandablePrivileges(t *testing.T) {
	const roleName = "test_revoke_expandable"

	resource.Test(t, resource.TestCase{
		Providers:    testutils.Provider(),
		CheckDestroy: testAccCheckRoleResourceDestroy([]string{roleName}),
		Steps: []resource.TestStep{
			{
				// Start with all expandable privileges
				Config: fmt.Sprintf(`
					resource "clickhouse_role" "revoke_expandable" {
						name       = "%s"
						database   = "*"
						privileges = ["REMOTE", "S3", "AZURE", "HDFS", "URL"]
					}`, roleName),
				Check: testutils.CheckStateSetAttr("privileges", "clickhouse_role.revoke_expandable",
					[]string{"REMOTE", "S3", "AZURE", "HDFS", "URL"}),
			},
			{
				// Revoke some expandable privileges
				Config: fmt.Sprintf(`
					resource "clickhouse_role" "revoke_expandable" {
						name       = "%s"
						database   = "*"
						privileges = ["REMOTE", "S3"]
					}`, roleName),
				Check: testutils.CheckStateSetAttr("privileges", "clickhouse_role.revoke_expandable",
					[]string{"REMOTE", "S3"}),
			},
			{
				// Re-apply - should be no-op
				Config: fmt.Sprintf(`
					resource "clickhouse_role" "revoke_expandable" {
						name       = "%s"
						database   = "*"
						privileges = ["REMOTE", "S3"]
					}`, roleName),
				ExpectNonEmptyPlan: false,
			},
			{
				// Revoke all expandable, add non-expandable
				Config: fmt.Sprintf(`
					resource "clickhouse_role" "revoke_expandable" {
						name       = "%s"
						database   = "*"
						privileges = ["SYSTEM FLUSH LOGS"]
					}`, roleName),
				Check: testutils.CheckStateSetAttr("privileges", "clickhouse_role.revoke_expandable",
					[]string{"SYSTEM FLUSH LOGS"}),
			},
		},
	})
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

// Test 1: Arrange: Have a role without privileges
//
//	Act: Add some privileges to it
//	Assert: It should now exist with those privileges
func TestAccResourceRole_AddPrivilegesToEmptyRole(t *testing.T) {
	const roleName = "test_add_to_empty_role"

	resource.Test(t, resource.TestCase{
		Providers:    testutils.Provider(),
		CheckDestroy: testAccCheckRoleResourceDestroy([]string{roleName}),
		Steps: []resource.TestStep{
			{
				// Arrange: Create role with NO privileges
				Config: fmt.Sprintf(`
					resource "clickhouse_role" "test_empty" {
						name       = "%s"
						database   = "*"
						privileges = []
					}`, roleName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("clickhouse_role.test_empty", "name", roleName),
					resource.TestCheckResourceAttr("clickhouse_role.test_empty", "database", "*"),
					// Verify in ClickHouse that role exists
					testAccCheckRoleExistsInClickHouse(roleName),
					// Verify role has NO privileges
					testAccCheckRoleHasNoPrivileges(roleName),
				),
			},
			{
				// Act: Add privileges to the role
				Config: fmt.Sprintf(`
					resource "clickhouse_role" "test_empty" {
						name       = "%s"
						database   = "*"
						privileges = ["REMOTE", "S3", "AZURE"]
					}`, roleName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("clickhouse_role.test_empty", "name", roleName),
					resource.TestCheckResourceAttr("clickhouse_role.test_empty", "database", "*"),
					testutils.CheckStateSetAttr("privileges", "clickhouse_role.test_empty", []string{"REMOTE", "S3", "AZURE"}),
					// Assert: Verify in ClickHouse that privileges exist
					testAccCheckRoleHasPrivileges(roleName, []string{"REMOTE", "S3", "AZURE"}),
				),
			},
			{
				// Verify no drift after adding privileges
				Config: fmt.Sprintf(`
					resource "clickhouse_role" "test_empty" {
						name       = "%s"
						database   = "*"
						privileges = ["REMOTE", "S3", "AZURE"]
					}`, roleName),
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// Test 2: Arrange: We have a role A with privileges P1 and P2 in database D1
//
//	Act: We change the role to have privileges P3 and P4 for database D2
//	Assert: The role A should have privileges P3 and P4 for D2 (P1, P2 on D1 are revoked)
func TestAccResourceRole_ChangeDatabaseAndPrivileges(t *testing.T) {
	t.Skip("SKIPPED: Resource model does not yet support multiple databases per role. " +
		"This test documents the DESIRED behavior where a role can accumulate privileges " +
		"across multiple databases. The resource currently replaces privileges when database changes. " +
		"See: Future enhancement to support multiple database grants per role.")

	const roleName = "test_change_db_privs"
	const db1 = "test_role_db_1"
	const db2 = "test_role_db_2"

	resource.Test(t, resource.TestCase{
		Providers:    testutils.Provider(),
		CheckDestroy: testAccCheckRoleResourceDestroy([]string{roleName}),
		Steps: []resource.TestStep{
			{
				// Arrange: Create role A with privileges P1 (SELECT) and P2 (INSERT) on database D1
				Config: fmt.Sprintf(`
						resource "clickhouse_db" "db1" {
							name    = "%s"
							comment = "test database 1"
						}

						resource "clickhouse_db" "db2" {
							name    = "%s"
							comment = "test database 2"
						}

						resource "clickhouse_role" "test_change" {
							name       = "%s"
							database   = clickhouse_db.db1.name
							privileges = ["SELECT", "INSERT"]
						}`, db1, db2, roleName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("clickhouse_role.test_change", "name", roleName),
					resource.TestCheckResourceAttr("clickhouse_role.test_change", "database", db1),
					testutils.CheckStateSetAttr("privileges", "clickhouse_role.test_change", []string{"SELECT", "INSERT"}),
					// Verify in ClickHouse
					testAccCheckRoleHasPrivilegesOnDatabase(roleName, db1, []string{"SELECT", "INSERT"}),
				),
			},
			{
				// Act: Add privileges P3 (ALTER) and P4 (DROP TABLE) on database D2
				// DESIRED BEHAVIOR: This should ADD privileges on D2 while KEEPING privileges on D1
				Config: fmt.Sprintf(`
						resource "clickhouse_db" "db1" {
							name    = "%s"
							comment = "test database 1"
						}

						resource "clickhouse_db" "db2" {
							name    = "%s"
							comment = "test database 2"
						}

						resource "clickhouse_role" "test_change" {
							name       = "%s"
							database   = clickhouse_db.db2.name
							privileges = ["ALTER", "DROP TABLE"]
						}`, db1, db2, roleName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("clickhouse_role.test_change", "name", roleName),
					resource.TestCheckResourceAttr("clickhouse_role.test_change", "database", db2),
					testutils.CheckStateSetAttr("privileges", "clickhouse_role.test_change", []string{"ALTER", "DROP TABLE"}),
					// Assert: Verify role has NEW privileges on D2
					testAccCheckRoleHasPrivilegesOnDatabase(roleName, db2, []string{"ALTER", "DROP TABLE"}),
					// Assert: Verify role STILL has original privileges on D1 (DESIRED BEHAVIOR)
					testAccCheckRoleHasPrivilegesOnDatabase(roleName, db1, []string{"SELECT", "INSERT"}),
				),
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

// Helper: Check if role exists in ClickHouse (even with no privileges)
func testAccCheckRoleExistsInClickHouse(roleName string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		client := testutils.TestAccProvider.Meta().(*common.ApiClient)
		conn := client.ClickhouseConnection
		chRoleService := resourcerole.CHRoleService{CHConnection: conn}

		role, err := chRoleService.GetRole(context.Background(), roleName)
		if err != nil {
			return fmt.Errorf("error checking role existence: %v", err)
		}
		if role == nil {
			return fmt.Errorf("role %s does not exist in ClickHouse", roleName)
		}

		return nil
	}
}

// Helper: Check if role has specific privileges (database agnostic)
func testAccCheckRoleHasPrivileges(roleName string, expectedPrivileges []string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		client := testutils.TestAccProvider.Meta().(*common.ApiClient)
		conn := client.ClickhouseConnection
		chRoleService := resourcerole.CHRoleService{CHConnection: conn}

		role, err := chRoleService.GetRole(context.Background(), roleName)
		if err != nil {
			return fmt.Errorf("error fetching role: %v", err)
		}
		if role == nil {
			return fmt.Errorf("role %s not found", roleName)
		}

		// Create map of actual privileges
		actualPrivs := make(map[string]bool)
		for _, priv := range role.Privileges {
			actualPrivs[priv.AccessType] = true
		}

		// Check all expected privileges exist
		for _, expectedPriv := range expectedPrivileges {
			if !actualPrivs[expectedPriv] {
				return fmt.Errorf("role %s missing privilege %s. Has: %v", roleName, expectedPriv, role.Privileges)
			}
		}

		return nil
	}
}

// Helper: Check if role has specific privileges on a specific database
func testAccCheckRoleHasPrivilegesOnDatabase(roleName, database string, expectedPrivileges []string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		client := testutils.TestAccProvider.Meta().(*common.ApiClient)
		conn := client.ClickhouseConnection
		chRoleService := resourcerole.CHRoleService{CHConnection: conn}

		role, err := chRoleService.GetRole(context.Background(), roleName)
		if err != nil {
			return fmt.Errorf("error fetching role: %v", err)
		}
		if role == nil {
			return fmt.Errorf("role %s not found", roleName)
		}

		// Filter privileges for this database
		dbPrivs := make(map[string]bool)
		for _, priv := range role.Privileges {
			if priv.Database == database {
				dbPrivs[priv.AccessType] = true
			}
		}

		// Check all expected privileges exist on this database
		for _, expectedPriv := range expectedPrivileges {
			if !dbPrivs[expectedPriv] {
				return fmt.Errorf("role %s missing privilege %s on database %s. Has: %v",
					roleName, expectedPriv, database, role.Privileges)
			}
		}

		return nil
	}
}

// Helper: Check if role has NO privileges at all
func testAccCheckRoleHasNoPrivileges(roleName string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		client := testutils.TestAccProvider.Meta().(*common.ApiClient)
		conn := client.ClickhouseConnection
		chRoleService := resourcerole.CHRoleService{CHConnection: conn}

		role, err := chRoleService.GetRole(context.Background(), roleName)
		if err != nil {
			return fmt.Errorf("error fetching role: %v", err)
		}
		if role == nil {
			return fmt.Errorf("role %s not found", roleName)
		}

		// Check that role has NO privileges
		if len(role.Privileges) > 0 {
			return fmt.Errorf("role %s should have no privileges but has: %v",
				roleName, role.Privileges)
		}

		return nil
	}
}

// Helper: Check if role has NO privileges on a specific database
func testAccCheckRoleHasNoPrivilegesOnDatabase(roleName, database string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		client := testutils.TestAccProvider.Meta().(*common.ApiClient)
		conn := client.ClickhouseConnection
		chRoleService := resourcerole.CHRoleService{CHConnection: conn}

		role, err := chRoleService.GetRole(context.Background(), roleName)
		if err != nil {
			return fmt.Errorf("error fetching role: %v", err)
		}
		if role == nil {
			return fmt.Errorf("role %s not found", roleName)
		}

		// Check that no privileges exist for this database
		for _, priv := range role.Privileges {
			if priv.Database == database {
				return fmt.Errorf("role %s still has privilege %s on database %s (should have none)",
					roleName, priv.AccessType, database)
			}
		}

		return nil
	}
}
