# REMOTE Privilege Issue Example

This example reproduces the original issue where granting REMOTE privilege to a role with database `"*"` caused a syntax error.

## Original Error

```
Error: resource role update: error granting privileges to role remote:
code: 62, message: Syntax error: failed at position 24 (ON): ON *.*) TO remote.
Expected access type
```

## The Bug

The provider was generating incorrect SQL:
```sql
GRANT CURRENT GRANTS (REMOTE ON *.*) TO remote
```

## The Fix

The corrected SQL should be:
```sql
GRANT REMOTE ON *.* TO remote
```

Global privileges (REMOTE, S3, etc.) should NOT be wrapped in `CURRENT GRANTS()`.

## Prerequisites

1. Start ClickHouse cluster:
   ```bash
   cd ../..
   make docker-up
   ```

2. Build the provider locally:
   ```bash
   make build
   ```

## Running the Example

### Step 1: Initialize Terraform

```bash
cd examples/issue-remote-privilege
terraform init
```

### Step 2: Apply (will FAIL with master branch)

```bash
terraform apply
```

**Expected Result with Buggy Code:**
```
Error: resource role update: error granting privileges to role remote:
code: 62, message: Syntax error
```

### Step 3: Apply Fix

Go back to project root and apply the fix in `pkg/resources/role/service.go`:

```go
func getGrantQuery(roleName string, privileges []string, database string) string {
	// Separate global privileges from database-level privileges
	var globalPrivileges []string
	var dbPrivileges []string

	for _, privilege := range privileges {
		if IsGlobalPrivilege(privilege) {
			globalPrivileges = append(globalPrivileges, privilege)
		} else {
			dbPrivileges = append(dbPrivileges, privilege)
		}
	}

	// Build query parts
	var queries []string

	// Grant global privileges with ON *.* syntax
	if len(globalPrivileges) > 0 {
		queries = append(queries, fmt.Sprintf("GRANT %s ON *.* TO %s", strings.Join(globalPrivileges, ","), roleName))
	}

	// Grant database-level privileges with appropriate syntax
	if len(dbPrivileges) > 0 {
		if database == "system" || database == "*" {
			queries = append(queries, fmt.Sprintf("GRANT CURRENT GRANTS (%s ON %s.*) TO %s", strings.Join(dbPrivileges, ","), database, roleName))
		} else {
			queries = append(queries, fmt.Sprintf("GRANT %s ON %s.* TO %s", strings.Join(dbPrivileges, ","), database, roleName))
		}
	}

	return strings.Join(queries, "; ")
}
```

### Step 4: Rebuild and Apply Again

```bash
cd ../..
make build
cd examples/issue-remote-privilege
terraform apply
```

**Expected Result with Fix:**
```
Apply complete! Resources: 1 added, 0 changed, 0 destroyed.
```

## Verify

Check the role was created in ClickHouse:

```bash
docker-compose exec clickhouse-01 clickhouse-client --query "SELECT name FROM system.roles WHERE name='remote'"
```

Check the privileges (REMOTE expands to READ + WRITE):

```bash
docker-compose exec clickhouse-01 clickhouse-client --query "SELECT role_name, access_type, database FROM system.grants WHERE role_name='remote'"
```

Expected output:
```
remote	READ	\N
remote	WRITE	\N
```

## Cleanup

```bash
terraform destroy
```

## Notes

- REMOTE privilege in ClickHouse expands to READ + WRITE privileges
- This is expected behavior
- The fix ensures the SQL syntax is correct for global privileges
