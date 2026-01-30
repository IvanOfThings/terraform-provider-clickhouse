---
page_title: "Privilege Revoke Fix"
subcategory: "Troubleshooting"
description: |-
  Documentation for the expandable privilege state drift fix in the ClickHouse Terraform Provider, including background, solution, and verification steps.
---

# REVOKE Fix Documentation

## Overview

This document describes the fix for privilege revocation issues in the ClickHouse Terraform Provider, specifically related to expandable SOURCE privileges and state drift detection.

## Problem Statement

### Background

ClickHouse v25.7+ introduced "expandable" SOURCE privileges that automatically expand into READ + WRITE variants when granted. For example:

- `GRANT REMOTE ON *.* TO role` → Internally stored as `REMOTE`, `REMOTE_READ`, `REMOTE_WRITE`
- `GRANT S3 ON *.* TO role` → Internally stored as `S3`, `S3_READ`, `S3_WRITE`

### The Issue

When querying `system.grants` table, ClickHouse returns the **expanded** privileges, not the original granted privilege. This caused two critical problems:

1. **False State Drift Detection**: Terraform detected changes on every apply cycle
   - User grants: `REMOTE`
   - Provider reads from `system.grants`: `REMOTE`, `REMOTE_READ`, `REMOTE_WRITE`
   - Terraform sees mismatch and plans to "fix" the state
   - Endless apply cycles

2. **Incorrect REVOKE Statements**: When revoking privileges, provider attempted to revoke expanded forms
   - Tried: `REVOKE REMOTE_READ, REMOTE_WRITE FROM role`
   - Should use: `REVOKE REMOTE FROM role`

### Affected Privileges

All expandable SOURCE privileges were affected:

- `REMOTE`
- `S3`
- `AZURE`
- `HDFS`
- `URL`
- `MYSQL`
- `POSTGRES`
- `MONGO`
- `KAFKA`

## Solution

### Use SHOW GRANTS Instead of system.grants

The fix changed privilege retrieval from querying `system.grants` table to using the `SHOW GRANTS` command.

**Before (Broken):**
```go
// Query system.grants - returns EXPANDED privileges
query := fmt.Sprintf("SELECT access_type, database FROM system.grants WHERE role_name = '%s'", roleName)
// Returns: REMOTE, REMOTE_READ, REMOTE_WRITE
```

**After (Fixed):**
```go
// Use SHOW GRANTS - returns ORIGINAL privileges
query := fmt.Sprintf("SHOW GRANTS FOR %s", roleName)
// Returns: GRANT REMOTE ON *.* TO role_name
```

### Implementation Details

#### 1. New Grant Parsing Logic

Created `parseGrantStatement()` function to parse SHOW GRANTS output:

```go
func parseGrantStatement(statement, expectedRole string) ([]CHGrant, error) {
    // Parses: "GRANT REMOTE, S3 ON *.* TO role_name"
    // Into: []CHGrant{
    //   {RoleName: "role_name", AccessType: "REMOTE", Database: "*"},
    //   {RoleName: "role_name", AccessType: "S3", Database: "*"},
    // }
}
```

Supported formats:
- `GRANT <privileges> ON *.* TO <role>` (global privileges)
- `GRANT <privileges> ON <database>.* TO <role>` (database-level)

#### 2. Updated getRoleGrants()

```go
func (rs *CHRoleService) getRoleGrants(ctx context.Context, roleName string) ([]CHGrant, error) {
    // Use SHOW GRANTS instead of SELECT from system.grants
    query := fmt.Sprintf("SHOW GRANTS FOR %s", roleName)
    rows, err := (*rs.CHConnection).Query(ctx, query)
    // ... parse each grant statement
}
```

#### 3. Preserved Original Privilege Logic

Grant and revoke operations already used the correct original privileges:

```go
// Always grants original privilege
GRANT REMOTE ON *.* TO role_name

// Always revokes original privilege
REVOKE REMOTE FROM role_name
```

The fix ensured that **read** operations match **write** operations.

## Testing

### Comprehensive Test Coverage

Added extensive tests to prevent regression:

#### 1. Individual Privilege Tests
```go
TestAccResourceRole_ExpandablePrivileges
// Tests each of 9 expandable privileges individually
// Verifies: Create, Read, No drift across multiple applies
```

#### 2. Multiple Privilege Tests
```go
TestAccResourceRole_MultipleExpandablePrivileges
// Tests: ["REMOTE", "S3", "HDFS"]
// Ensures multiple expandable privileges work together
```

#### 3. Mixed Privilege Tests
```go
TestAccResourceRole_MixedExpandableAndNonExpandable
// Tests: ["REMOTE", "S3", "SYSTEM FLUSH LOGS"]
// Verifies expandable + non-expandable coexistence
```

#### 4. Stability Tests
```go
TestAccResourceRole_MultipleApplyCycles
// Runs 5 consecutive apply cycles
// Ensures long-term stability
```

#### 5. Revoke Tests
```go
TestAccResourceRole_RevokeExpandablePrivileges
// Tests revoking expandable privileges
// Verifies clean removal without expanded form issues
```

### Test Metrics

- **Total test functions**: 10+ for expandable privileges
- **Total apply cycles**: 54+ specifically testing expandable behavior
- **Coverage**: 100% of all 9 expandable privileges
- **Drift detection**: All tests verify `ExpectNonEmptyPlan: false`

## Verification

### Before Fix

```bash
# Initial apply
terraform apply
# Plan: 1 to add

# Second apply (should be no-op)
terraform apply
# Plan: 1 to change (BUG!)
# Terraform detects drift due to expanded privileges
```

### After Fix

```bash
# Initial apply
terraform apply
# Plan: 1 to add

# Second apply
terraform apply
# Plan: 0 to change ✅
# No drift detected!

# Third apply (stability check)
terraform apply
# Plan: 0 to change ✅
```

## Impact

### What Changed

**Provider Behavior:**
- ✅ No more false state drift for expandable privileges
- ✅ Correct privilege storage in Terraform state
- ✅ Accurate privilege comparison on read
- ✅ Stable across multiple apply cycles

**User Experience:**
- ✅ Single `terraform apply` needed (not endless loops)
- ✅ Predictable behavior matching expectations
- ✅ Clean state file without expanded privilege noise

### What Stayed the Same

**No Breaking Changes:**
- ✅ HCL configuration syntax unchanged
- ✅ Resource schema unchanged
- ✅ Grant/revoke SQL statements unchanged
- ✅ Existing privilege validation unchanged
- ✅ Database-level privileges unaffected

### Compatibility

- **ClickHouse versions**: All versions (SHOW GRANTS is standard SQL)
- **Terraform versions**: All supported versions
- **Existing state**: Automatically fixed on next apply
- **Migration needed**: None - transparent upgrade

## Related Files

### Implementation
- `pkg/resources/role/service.go` - Updated `getRoleGrants()` and `parseGrantStatement()`
- `pkg/resources/role/validators.go` - Defines `ExpandableSourcePrivileges` list

### Tests
- `pkg/resources/role/resource_role_acceptance_test.go` - All expandable privilege tests
- `pkg/resources/role/service_test.go` - Unit tests for parsing logic

### Documentation
- `internal-docs/TESTING.md` - Comprehensive testing guide
- `docs/guides/setup.md` - Setup instructions
- `CLAUDE.md` - Architecture and privilege system details

## Best Practices

### For Provider Development

1. **Always use SHOW GRANTS for privilege retrieval**
   ```go
   // DO THIS
   query := fmt.Sprintf("SHOW GRANTS FOR %s", roleName)

   // NOT THIS
   query := "SELECT * FROM system.grants WHERE role_name = ?"
   ```

2. **Test with multiple apply cycles**
   ```go
   Steps: []resource.TestStep{
       {Config: config, Check: checks},
       {Config: config, ExpectNonEmptyPlan: false},  // Critical!
       {Config: config, ExpectNonEmptyPlan: false},  // Test stability
   }
   ```

3. **Verify no drift detection**
   ```go
   // Always include drift detection test
   {
       Config: sameConfigAsAbove,
       ExpectNonEmptyPlan: false,  // Must be no changes
   }
   ```

### For Users

1. **Upgrading**: Simply upgrade provider version, no action needed
2. **Testing**: Run `terraform plan` after upgrade, should show no changes
3. **Monitoring**: Check for unexpected diffs in global privilege roles

## Future Considerations

### Potential Enhancements

1. **Expanded Privilege Information**: Add computed field showing expanded forms
   ```hcl
   resource "clickhouse_role" "example" {
     privileges = ["REMOTE"]  # User-specified

     # Computed (read-only)
     expanded_privileges = ["REMOTE", "REMOTE_READ", "REMOTE_WRITE"]
   }
   ```

2. **Privilege Validation**: Warn when mixing expandable with explicit expanded
   ```hcl
   # Should warn: redundant
   privileges = ["REMOTE", "REMOTE_READ"]
   ```

3. **Documentation Generation**: Auto-document which privileges are expandable
   ```bash
   terraform providers schema -json | jq '.privileges'
   ```

### Known Limitations

1. **Single Database Per Role**: Resource currently supports one database per role
   - See skipped test: `TestAccResourceRole_ChangeDatabaseAndPrivileges`
   - Future enhancement: Support multiple database grants per role

2. **Table-Level Grants**: Not supported by `parseGrantStatement()`
   - Currently rejects: `GRANT SELECT ON database.table TO role`
   - Only supports: `ON *.*` and `ON database.*`

3. **Role Hierarchies**: Not yet tested with nested role grants
   - Future: Test `GRANT role1 TO role2` scenarios

## References

- [ClickHouse GRANT Documentation](https://clickhouse.com/docs/sql-reference/statements/grant)
- [ClickHouse Privileges Reference](https://clickhouse.com/docs/sql-reference/statements/grant#privileges)
- [Terraform Plugin SDK Testing](https://www.terraform.io/plugin/sdkv2/testing/acceptance-tests)

## Related Issues

See `internal-docs/issues/remote-privilege-state-drift.md` for the original issue that prompted this fix.
