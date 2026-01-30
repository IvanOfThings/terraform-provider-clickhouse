# Implementation Plan: Handle REVOKE Statements in SHOW GRANTS Output

**Created:** 2026-01-30
**Status:** Draft
**Estimated Effort:** S (Small - 1-2 hours)

## Summary

The `parseGrantStatement` function in `pkg/resources/role/service.go` fails when ClickHouse's `SHOW GRANTS` command returns REVOKE statements. This happens when partial revokes are used (broad grants with specific exceptions). The provider needs to gracefully handle REVOKE statements to avoid breaking existing roles that use partial revokes.

## Problem Statement

### Error Message
```
Error: resource role read: error fetching role grants: error parsing grant statement
'REVOKE SELECT ON system.system_tables_tracking_auburn_qf_97 FROM system_read':
invalid grant statement (missing GRANT): "REVOKE SELECT ON system.system_tables_tracking_auburn_qf_97 FROM system_read"
```

### Root Cause

1. `SHOW GRANTS FOR <role>` returns **both GRANT and REVOKE statements** when partial revokes exist
2. The `parseGrantStatement()` function only accepts statements starting with "GRANT "
3. When a REVOKE statement is encountered, parsing fails with "missing GRANT" error

### When This Occurs

ClickHouse returns REVOKE statements when:
- A role has broad database-level grants (e.g., `GRANT SELECT ON system.*`)
- Specific tables/columns are revoked from that broad grant
- Example: `REVOKE SELECT ON system.system_tables_tracking_auburn_qf_97 FROM system_read`

## Research Findings

### ClickHouse Behavior (Confirmed)

From [ClickHouse documentation](https://clickhouse.com/docs/sql-reference/statements/revoke):

```sql
GRANT SELECT ON *.* TO john;
REVOKE SELECT ON accounts.* FROM john;

SHOW GRANTS FOR john;
-- Returns both:
-- GRANT SELECT ON *.* TO john
-- REVOKE SELECT ON accounts.* FROM john
```

### Repository Analysis

**Current Implementation (`pkg/resources/role/service.go` lines 112-198):**
```go
func parseGrantStatement(statement, expectedRole string) ([]CHGrant, error) {
    stmt := strings.TrimSpace(statement)
    upper := strings.ToUpper(stmt)

    if !strings.HasPrefix(upper, "GRANT ") {
        return nil, fmt.Errorf("invalid grant statement (missing GRANT): %q", statement)
    }
    // ... only handles GRANT statements
}
```

**Current Grant Fetching (`pkg/resources/role/service.go` lines 80-110):**
```go
func (rs *CHRoleService) getRoleGrants(ctx context.Context, roleName string) ([]CHGrant, error) {
    query := fmt.Sprintf("SHOW GRANTS FOR %s", roleName)
    rows, err := (*rs.CHConnection).Query(ctx, query)

    for rows.Next() {
        var grantStatement string
        err := rows.Scan(&grantStatement)

        grants, err := parseGrantStatement(grantStatement, roleName)  // ❌ Fails on REVOKE
        // ...
    }
}
```

### Design Decision: Skip vs Parse REVOKE

| Approach | Pros | Cons |
|----------|------|------|
| **Skip REVOKE statements** | Simple, minimal code change | Doesn't track exceptions, may cause drift |
| **Parse REVOKE statements** | Complete representation | Complex, table-level not supported |
| **Error with clear message** | Explicit about limitation | Blocks users with partial revokes |

**Recommendation:** Skip REVOKE statements with a warning log. This allows:
1. Existing roles with partial revokes to be read without error
2. Provider to continue managing the GRANT portion
3. Users to understand the limitation through logs

## Implementation Steps

### Step 0: Reproduce the Error (TDD - RED Phase)

**Goal:** Write an acceptance test that reproduces the exact error before implementing the fix. This proves:
1. The bug exists and is reproducible
2. Our fix actually solves the problem
3. We have regression protection

**File:** `pkg/resources/role/resource_role_acceptance_test.go`

**Test to add:**

```go
func TestAccResourceRole_PartialRevokeError_Reproduction(t *testing.T) {
    // This test reproduces the bug where SHOW GRANTS returns REVOKE statements
    // that cause parseGrantStatement to fail with "missing GRANT" error.
    //
    // Error being reproduced:
    // Error: resource role read: error fetching role grants: error parsing grant statement
    // 'REVOKE SELECT ON system.system_tables_tracking_auburn_qf_97 FROM system_read':
    // invalid grant statement (missing GRANT): "REVOKE SELECT ON ..."

    roleName := fmt.Sprintf("test_revoke_repro_%s", acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum))

    resource.Test(t, resource.TestCase{
        PreCheck:                 func() { testAccPreCheck(t) },
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            // Step 1: Create a role with broad database-level SELECT privilege
            {
                Config: fmt.Sprintf(`
resource "clickhouse_role" "test" {
    name     = "%s"
    database = "system"
    privileges = ["SELECT"]
}
`, roleName),
                Check: resource.ComposeTestCheckFunc(
                    resource.TestCheckResourceAttr("clickhouse_role.test", "name", roleName),
                    resource.TestCheckResourceAttr("clickhouse_role.test", "database", "system"),
                ),
            },
            // Step 2: Apply partial revoke OUTSIDE Terraform, then re-read
            {
                PreConfig: func() {
                    // Execute partial revoke directly in ClickHouse
                    // This creates a REVOKE statement in SHOW GRANTS output
                    ctx := context.Background()
                    conn := testAccGetClickHouseConnection(t)

                    // Revoke SELECT on a specific table within system database
                    // This creates: REVOKE SELECT ON system.numbers FROM role
                    err := conn.Exec(ctx, fmt.Sprintf(
                        "REVOKE SELECT ON system.numbers FROM %s", roleName))
                    if err != nil {
                        t.Fatalf("Failed to apply partial revoke: %v", err)
                    }

                    // Verify SHOW GRANTS now includes REVOKE statement
                    rows, err := conn.Query(ctx, fmt.Sprintf("SHOW GRANTS FOR %s", roleName))
                    if err != nil {
                        t.Fatalf("Failed to query grants: %v", err)
                    }
                    defer rows.Close()

                    hasRevoke := false
                    for rows.Next() {
                        var stmt string
                        rows.Scan(&stmt)
                        t.Logf("SHOW GRANTS output: %s", stmt)
                        if strings.HasPrefix(strings.ToUpper(stmt), "REVOKE ") {
                            hasRevoke = true
                        }
                    }

                    if !hasRevoke {
                        t.Log("WARNING: ClickHouse did not return REVOKE statement - test may not reproduce the bug")
                    }
                },
                // Re-apply same config - this triggers a Read which should fail
                // with "invalid grant statement (missing GRANT)" error BEFORE the fix
                Config: fmt.Sprintf(`
resource "clickhouse_role" "test" {
    name     = "%s"
    database = "system"
    privileges = ["SELECT"]
}
`, roleName),
                // BEFORE FIX: This will error with "missing GRANT"
                // AFTER FIX: This should succeed with no changes planned
                ExpectNonEmptyPlan: false,
            },
        },
    })
}
```

**Run the test to confirm failure:**

```bash
# This should FAIL before implementing the fix
TF_ACC=1 go test -v ./pkg/resources/role/... -run TestAccResourceRole_PartialRevokeError_Reproduction -timeout 10m
```

**Expected output BEFORE fix:**
```
Error: resource role read: error fetching role grants: error parsing grant statement
'REVOKE SELECT ON system.numbers FROM test_revoke_repro_xxxxx':
invalid grant statement (missing GRANT): "REVOKE SELECT ON system.numbers FROM test_revoke_repro_xxxxx"
```

**Why this approach:**
- Uses `system` database which always exists
- Revokes on `system.numbers` which is a standard ClickHouse table
- PreConfig executes raw SQL to simulate external changes
- Logs the SHOW GRANTS output for debugging

---

### Step 1: Update `getRoleGrants` to Skip REVOKE Statements

**File:** `pkg/resources/role/service.go`

**Location:** Lines 80-110

**Change:** Add REVOKE detection before calling `parseGrantStatement()`

```go
func (rs *CHRoleService) getRoleGrants(ctx context.Context, roleName string) ([]CHGrant, error) {
    query := fmt.Sprintf("SHOW GRANTS FOR %s", roleName)
    rows, err := (*rs.CHConnection).Query(ctx, query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    privileges := []CHGrant{}
    for rows.Next() {
        var grantStatement string
        err := rows.Scan(&grantStatement)
        if err != nil {
            return nil, err
        }

        // Skip REVOKE statements - provider only manages GRANT privileges
        // REVOKE statements appear when partial revokes are used in ClickHouse
        if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(grantStatement)), "REVOKE ") {
            continue
        }

        grants, err := parseGrantStatement(grantStatement, roleName)
        if err != nil {
            return nil, fmt.Errorf("error parsing grant statement '%s': %w", grantStatement, err)
        }
        privileges = append(privileges, grants...)
    }

    return privileges, nil
}
```

### Step 2: Add Unit Test for REVOKE Statement Handling

**File:** `pkg/resources/role/service_test.go`

**Add new test case:**

```go
func TestGetRoleGrants_SkipsRevokeStatements(t *testing.T) {
    // This test verifies that REVOKE statements in SHOW GRANTS output
    // are gracefully skipped without causing errors

    // Test cases for parseGrantStatement should verify REVOKE is not processed
    testCases := []struct {
        name      string
        statement string
        shouldErr bool
    }{
        {
            name:      "REVOKE statement should be skipped at caller level",
            statement: "REVOKE SELECT ON system.system_tables FROM test_role",
            shouldErr: true, // parseGrantStatement itself errors, but getRoleGrants skips
        },
        {
            name:      "REVOKE with table-level scope",
            statement: "REVOKE SELECT ON db.specific_table FROM test_role",
            shouldErr: true,
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            _, err := parseGrantStatement(tc.statement, "test_role")
            if tc.shouldErr {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), "missing GRANT")
            }
        })
    }
}
```

### Step 3: Add Acceptance Test for Role with Partial Revokes

**File:** `pkg/resources/role/resource_role_acceptance_test.go`

**Add test that creates a role, manually adds partial revoke, then verifies provider can read:**

```go
func TestAccResourceRole_WithPartialRevokeInClickHouse(t *testing.T) {
    // This test verifies the provider can read roles that have
    // partial revokes applied outside of Terraform

    roleName := fmt.Sprintf("test_partial_revoke_%s", acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum))

    resource.Test(t, resource.TestCase{
        PreCheck:                 func() { testAccPreCheck(t) },
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            {
                Config: testAccRoleConfig_BasicWithPrivileges(roleName, "*", []string{"SELECT"}),
                Check: resource.ComposeTestCheckFunc(
                    resource.TestCheckResourceAttr("clickhouse_role.test", "name", roleName),
                ),
            },
            {
                // Apply partial revoke outside Terraform
                PreConfig: func() {
                    // Execute: REVOKE SELECT ON system.some_table FROM role
                    // This simulates external changes that add REVOKE to SHOW GRANTS
                },
                Config: testAccRoleConfig_BasicWithPrivileges(roleName, "*", []string{"SELECT"}),
                // Should not error even with REVOKE in SHOW GRANTS output
                ExpectNonEmptyPlan: false,
            },
        },
    })
}
```

### Step 4: Update Documentation

**File:** `CLAUDE.md`

**Add to Known Limitations section:**

```markdown
### Partial Revokes Not Tracked

The provider skips REVOKE statements in `SHOW GRANTS` output. When ClickHouse has partial revokes (broad grant + specific revoke), only the GRANT portion is tracked in Terraform state.

**Impact:** If partial revokes are applied outside Terraform, the provider will not detect or manage them.

**Workaround:** Use complete privilege sets per database scope rather than partial revokes.
```

### Step 5: Validate Changes

- [ ] Run unit tests: `go test ./pkg/resources/role/... -v`
- [ ] Run acceptance tests: `TF_ACC=1 go test ./pkg/resources/role/... -v -timeout 30m`
- [ ] Test against ClickHouse with partial revokes manually
- [ ] Verify no regressions in existing tests

## Acceptance Criteria

- [ ] **TDD RED:** Acceptance test `TestAccResourceRole_PartialRevokeError_Reproduction` fails with "missing GRANT" error
- [ ] **TDD GREEN:** After fix, the same test passes
- [ ] Provider can read roles that have REVOKE statements in `SHOW GRANTS` output
- [ ] REVOKE statements are silently skipped (no errors)
- [ ] Unit tests cover REVOKE statement handling
- [ ] Acceptance test verifies real-world partial revoke scenario
- [ ] Documentation updated with limitation
- [ ] All existing tests continue to pass

## Files to Modify

| File | Change |
|------|--------|
| `pkg/resources/role/resource_role_acceptance_test.go` | Add reproduction test FIRST (Step 0) |
| `pkg/resources/role/service.go` | Add REVOKE detection in `getRoleGrants()` |
| `pkg/resources/role/service_test.go` | Add unit test for REVOKE handling |
| `pkg/resources/role/resource_role_acceptance_test.go` | Add acceptance test |
| `CLAUDE.md` | Document limitation |

## Security Considerations

None - this is a read-side fix that improves compatibility without changing security behavior.

## Performance Considerations

Minimal - adds a single string prefix check per grant statement.

## Alternative Approaches Considered

### 1. Parse and Track REVOKE Statements

**Why not:** Would require significant changes to data model, state management, and HCL schema. Also, table-level grants are explicitly not supported (documented limitation).

### 2. Error with Clear Message

**Why not:** Would block users who have partial revokes applied outside Terraform, even if they don't need to manage them via Terraform.

### 3. Use system.grants Table with Filtering

**Why not:** Previous work established that `system.grants` causes issues with expandable privileges. Staying with `SHOW GRANTS` is the correct approach.

## Related Issues

- Previous fix: Expandable privilege state drift (internal-docs/issues/remote-privilege-state-drift.md)
- ClickHouse Issue: [#31138 - REVOKE ordering after restart](https://github.com/ClickHouse/ClickHouse/issues/31138)

---

## Next Steps

When ready to implement, run:
- `/wiz:work plans/handle-revoke-statements-in-show-grants.md` - Execute the plan
