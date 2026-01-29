# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Terraform Provider for ClickHouse database management, built on Terraform Plugin SDK v2. It manages databases, tables, roles, and users in ClickHouse clusters via the native protocol (port 9000).

**Tech Stack:**
- Go 1.19+ with Terraform Plugin SDK v2
- ClickHouse Go driver v2 (github.com/ClickHouse/clickhouse-go/v2)
- Native ClickHouse protocol (TCP port 9000, not HTTP 8123)

## Development Commands

### Testing

```bash
# Run all acceptance tests (requires ClickHouse running locally)
make testacc

# Run acceptance tests with custom environment
TF_ACC=1 TF_CLICKHOUSE_HOST="127.0.0.1" TF_CLICKHOUSE_USERNAME="default" TF_CLICKHOUSE_PASSWORD="" TF_CLICKHOUSE_PORT=9000 go test ./... -v -timeout 120m

# Run single test
TF_ACC=1 TF_CLICKHOUSE_HOST=127.0.0.1 TF_CLICKHOUSE_PORT=9000 TF_CLICKHOUSE_USERNAME=default TF_CLICKHOUSE_PASSWORD="" go test -v -run "TestAccResourceRole_AddPrivilegesToEmptyRole" ./pkg/resources/role/... -timeout 30m

# Run unit tests (parallel, no ClickHouse needed)
make test
```

**Important:** `TF_ACC=1` environment variable is REQUIRED to run acceptance tests. Without it, tests are skipped.

### Building

```bash
# Build binary
make build

# Install locally for Terraform development
make install              # Linux
make install-darwin       # macOS

# Generate documentation (auto-updates docs/ from code annotations)
make doc
# OR
go generate ./...
```

### Local Development Setup

The provider expects ClickHouse running on `127.0.0.1:9000` for tests. Use Docker:

```bash
docker run -d -p 9000:9000 -p 8123:8123 clickhouse/clickhouse-server
```

## Architecture

### Resource Structure Pattern

Every resource follows this consistent pattern:

```
pkg/resources/{resource_name}/
├── resource_{name}.go              # Terraform schema + CRUD operations
├── service.go                      # ClickHouse database operations
├── model.go                        # Data structures and transformations
├── validators.go                   # Custom validation logic
├── resource_{name}_acceptance_test.go  # Acceptance tests
└── service_test.go                 # Unit tests for service layer
```

**Example: Role Resource Flow**

1. **resource_role.go**: Defines Terraform schema (name, database, privileges), implements CRUD callbacks
2. **service.go**: Contains `CHRoleService` with methods like `CreateRole()`, `GetRole()`, `UpdateRole()`
3. **model.go**: Transforms between ClickHouse model (`CHRole`) and Terraform model (`RoleResource`)
4. **validators.go**: Custom validation (e.g., `ValidatePrivileges()` separates global vs database-level privileges)

### Provider Configuration

Provider establishes a SINGLE connection to ClickHouse that's shared across all resources:

```go
// pkg/provider/provider.go
// Creates connection with native protocol
conn, err := clickhouse.Open(&clickhouse.Options{
    Addr: []string{fmt.Sprintf("%s:%d", host, port)},
    Auth: clickhouse.Auth{...},
})

// Returns ApiClient shared by all resources
return &common.ApiClient{
    ClickhouseConnection: &conn,
    DefaultCluster: defaultCluster,
}
```

All resources receive this `ApiClient` via the `meta` parameter in CRUD functions.

### Cluster Support

The provider supports both standalone and clustered ClickHouse:

1. **Provider-level default**: `default_cluster` parameter applies to all resources unless overridden
2. **Resource-level override**: Each resource can specify its own `cluster` attribute
3. **ON CLUSTER syntax**: Automatically injected by `common.GetClusterStatement(cluster)` in SQL queries
4. **Metadata persistence**: Cluster name stored in resource comments as JSON for state tracking

**Altinity Operator Support**: Macros like `'{cluster}'`, `'{installation}'`, `'{replica}'` are supported for Kubernetes deployments.

### Comment Encoding Pattern

Resources store metadata in ClickHouse comments as JSON:

```go
// pkg/common/utils.go
comment := common.GetComment("user comment", "cluster_name")
// Produces: {"comment":"user comment","cluster":"cluster_name"}

// On read:
userComment, cluster, err := common.UnmarshalComment(storedComment)
```

This pattern is used by `clickhouse_db` and `clickhouse_table` to track cluster association across Terraform operations.

### Privilege System (Roles)

ClickHouse has TWO types of privileges that require different SQL syntax:

**Global privileges** (require `ON *.*` syntax):
- REMOTE, S3, AZURE, HDFS, URL, MYSQL, POSTGRES, MONGO, KAFKA
- These are "expandable" - ClickHouse expands them (e.g., REMOTE → REMOTE + REMOTE_READ + REMOTE_WRITE)
- Provider MUST use `SHOW GRANTS` (not `system.grants`) to get original privileges, not expanded forms

**Database-level privileges** (require `ON database.*` syntax):
- SELECT, INSERT, ALTER, DROP TABLE, etc.
- May require `GRANT CURRENT GRANTS (...)` wrapper for system database

**Critical Implementation Details:**

```go
// pkg/resources/role/validators.go
func IsGlobalPrivilege(privilege string) bool {
    // Check if privilege needs ON *.* syntax
}

// pkg/resources/role/service.go
func getGrantQuery(roleName, privileges, database) string {
    // Separates global vs database privileges
    // Generates appropriate GRANT statements
}
```

**Known Bug (documented in tests):** The resource currently supports only ONE database per role. `TestAccResourceRole_ChangeDatabaseAndPrivileges` is skipped with `t.Skip()` documenting the DESIRED behavior (accumulate privileges across multiple databases). This is a future enhancement.

### State Drift Fixes

**Empty Privileges Bug (FIXED):** When a role has no privileges, ClickHouse doesn't store database association. The fix in `resource_role.go:93-96` preserves the database field from Terraform state:

```go
if roleResource.Database == "" && len(roleResource.Privileges.List()) == 0 {
    roleResource.Database = d.Get("database").(string)
}
```

## Testing Guidelines

### Test-Driven Development (TDD)

This project follows strict TDD methodology (enforced by `/wiz:test-driven-development` skill):

1. **RED**: Write failing test that reproduces the bug/feature
2. **GREEN**: Write minimal code to make test pass
3. **REFACTOR**: Clean up while keeping tests green

**Critical Rule:** NEVER fix a bug without first writing a test that reproduces it.

### Acceptance Test Pattern

```go
func TestAccResourceX(t *testing.T) {
    // Optional: Skip for known limitations
    // t.Skip("reason explaining why skipped and desired behavior")

    resource.Test(t, resource.TestCase{
        Providers:    testutils.Provider(),
        CheckDestroy: testAccCheckXResourceDestroy([]string{"resource_name"}),
        Steps: []resource.TestStep{
            {
                // Step 1: Create resource
                Config: fmt.Sprintf(`
                    resource "clickhouse_x" "test" {
                        name = "%s"
                        // ...
                    }`, resourceName),
                Check: resource.ComposeTestCheckFunc(
                    resource.TestCheckResourceAttr("clickhouse_x.test", "name", resourceName),
                    // Custom checks that query ClickHouse directly
                    testAccCheckXExistsInClickHouse(resourceName),
                ),
            },
            {
                // Step 2: Update resource
                Config: fmt.Sprintf(`...updated config...`),
                Check: resource.ComposeTestCheckFunc(...),
            },
            {
                // Step 3: Verify no drift
                Config: fmt.Sprintf(`...same as step 2...`),
                ExpectNonEmptyPlan: false,  // Should be no changes
            },
        },
    })
}
```

**Helper Functions:** Create custom check functions that query ClickHouse directly to verify state:

```go
func testAccCheckRoleHasPrivileges(roleName string, expectedPrivileges []string) resource.TestCheckFunc {
    return func(s *terraform.State) error {
        // Connect to ClickHouse
        // Run SHOW GRANTS FOR roleName
        // Verify privileges match expected
        return nil
    }
}
```

### Test Utilities

```go
// pkg/testutils/testutils.go

// Check set/list attributes (order-independent for sets)
testutils.CheckStateSetAttr("privileges", "clickhouse_role.test", []string{"SELECT", "INSERT"})

// Get provider for tests
testutils.Provider()

// Pre-check for acceptance tests
testutils.TestAccPreCheck(t)
```

## Common Patterns

### ID Format

Resources use `{cluster}:{resource_name}` format for IDs:

```go
d.SetId(fmt.Sprintf("%s:%s", cluster, name))

// Extract on read:
idParts := strings.Split(d.Id(), ":")
cluster := idParts[0]
name := idParts[1]
```

### Service Layer Pattern

All database operations go through service structs:

```go
type CHRoleService struct {
    CHConnection *driver.Conn
}

func (s *CHRoleService) CreateRole(ctx context.Context, name, database string, privileges []string) (*CHRole, error) {
    conn := *s.CHConnection
    err := conn.Exec(ctx, fmt.Sprintf("CREATE ROLE %s", name))
    // ...
}
```

### Error Wrapping

Always wrap errors with context using `fmt.Errorf` with `%w`:

```go
if err != nil {
    return diag.FromErr(fmt.Errorf("resource role create: %w", err))
}
```

## Important Files

- **Provider entry**: `pkg/provider/provider.go` - Registers all resources and data sources
- **Common utilities**: `pkg/common/utils.go` - Comment encoding, cluster statements, set/list conversions
- **Test utilities**: `pkg/testutils/testutils.go` - Shared test helpers
- **Examples**: `examples/` - Terraform configurations for manual testing

## Resources Implemented

| Resource | File Location | Description |
|----------|---------------|-------------|
| `clickhouse_db` | `pkg/resources/db/` | Manages databases with comments |
| `clickhouse_table` | `pkg/resources/table/` | Manages tables (MergeTree, Distributed, etc.) |
| `clickhouse_role` | `pkg/resources/role/` | Manages roles with privileges |
| `clickhouse_user` | `pkg/resources/user/` | Manages users with passwords and role assignments |

## Data Sources

| Data Source | File Location | Description |
|-------------|---------------|-------------|
| `clickhouse_dbs` | `pkg/datasources/` | Lists all databases |

## Known Limitations

1. **Roles**: Currently support only ONE database per role (see skipped test `TestAccResourceRole_ChangeDatabaseAndPrivileges`)
2. **Table Engines**: Limited replicated table engine support (documented in README)
3. **Expandable Privileges**: Provider must use `SHOW GRANTS` not `system.grants` to avoid privilege expansion issues
