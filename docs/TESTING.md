# Testing Guide

This document provides comprehensive guidance for running and writing tests for the Terraform ClickHouse Provider.

## Table of Contents

- [Test Suite Overview](#test-suite-overview)
- [Running Tests](#running-tests)
- [Test Coverage](#test-coverage)
- [Expandable Privilege Testing](#expandable-privilege-testing)
- [Test Implementation Details](#test-implementation-details)
- [Coverage Metrics](#coverage-metrics)
- [Troubleshooting](#troubleshooting)

## Test Suite Overview

The provider includes two types of tests:

### Unit Tests
- **Purpose**: Fast, isolated tests that don't require ClickHouse
- **Location**: `*_test.go` files (excluding `*_acceptance_test.go`)
- **Coverage**: Service layer logic, parsing functions, validation
- **Execution**: Run in parallel without external dependencies

### Acceptance Tests
- **Purpose**: End-to-end integration tests with real ClickHouse instances
- **Location**: `*_acceptance_test.go` files
- **Coverage**: Full CRUD operations, state management, drift detection
- **Execution**: Require `TF_ACC=1` environment variable and ClickHouse connection

## Running Tests

### Prerequisites

All tests require:
- Go 1.19 or later
- ClickHouse server running (for acceptance tests)

**Quick Setup (Docker):**
```bash
# Standalone ClickHouse instance
docker run -d -p 9000:9000 -p 8123:8123 clickhouse/clickhouse-server

# OR use docker-compose for clustered setup (see docs/guides/SETUP.md)
docker-compose up -d
```

### Running Acceptance Tests

**Full Test Suite:**
```bash
# Using Makefile (recommended)
make testacc

# Manual execution with explicit environment variables
TF_ACC=1 \
TF_CLICKHOUSE_HOST="127.0.0.1" \
TF_CLICKHOUSE_USERNAME="default" \
TF_CLICKHOUSE_PASSWORD="" \
TF_CLICKHOUSE_PORT=9000 \
go test ./... -v -timeout 120m
```

**Single Test:**
```bash
# Run specific test by pattern
TF_ACC=1 \
TF_CLICKHOUSE_HOST=127.0.0.1 \
TF_CLICKHOUSE_PORT=9000 \
TF_CLICKHOUSE_USERNAME=default \
TF_CLICKHOUSE_PASSWORD="" \
go test -v -run "TestAccResourceRole_AddPrivilegesToEmptyRole" ./pkg/resources/role/... -timeout 30m
```

**Test by Resource:**
```bash
# Test only role resource
TF_ACC=1 go test -v ./pkg/resources/role/... -timeout 30m

# Test only database resource
TF_ACC=1 go test -v ./pkg/resources/db/... -timeout 30m

# Test only table resource
TF_ACC=1 go test -v ./pkg/resources/table/... -timeout 30m

# Test only user resource
TF_ACC=1 go test -v ./pkg/resources/user/... -timeout 30m
```

### Running Unit Tests

```bash
# Using Makefile
make test

# OR manual execution
go test ./... -v -timeout 30s
```

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `TF_ACC` | Yes (acceptance) | - | Must be `1` to run acceptance tests |
| `TF_CLICKHOUSE_HOST` | Yes | - | ClickHouse server hostname/IP |
| `TF_CLICKHOUSE_PORT` | Yes | - | Native protocol port (usually 9000) |
| `TF_CLICKHOUSE_USERNAME` | Yes | `default` | ClickHouse username |
| `TF_CLICKHOUSE_PASSWORD` | Yes | `` | ClickHouse password (empty for default user) |

**Important:** The provider uses the **native protocol port (9000)**, not the HTTP port (8123).

## Test Coverage

### Resource Coverage

The test suite covers all provider resources:

| Resource | Test File | Coverage |
|----------|-----------|----------|
| `clickhouse_db` | `pkg/resources/db/resource_db_acceptance_test.go` | Create, Read, Update, Delete, Cluster support, Comment handling |
| `clickhouse_table` | `pkg/resources/table/resource_table_acceptance_test.go` | Various engines, Columns, Partitions, Order by, Cluster support |
| `clickhouse_role` | `pkg/resources/role/resource_role_acceptance_test.go` | Privileges, Database-level, Global privileges, Expandable privileges |
| `clickhouse_user` | `pkg/resources/user/resource_user_acceptance_test.go` | User creation, Password management, Role assignment |

### Data Source Coverage

| Data Source | Test File | Coverage |
|-------------|-----------|----------|
| `clickhouse_dbs` | `pkg/datasources/data_source_dbs_acceptance_test.go` | Database listing |

## Expandable Privilege Testing

### Overview

ClickHouse v25.7+ introduced "expandable" SOURCE privileges that automatically expand into READ + WRITE variants. For example:
- `REMOTE` → `REMOTE` + `REMOTE_READ` + `REMOTE_WRITE`
- `S3` → `S3` + `S3_READ` + `S3_WRITE`

**Critical Implementation Detail:** The provider uses `SHOW GRANTS` (not `system.grants`) to retrieve the **original** granted privileges, preventing false state drift detection.

### Expandable Privileges

The following privileges are expandable and require special handling:

1. **REMOTE** - Remote table engine access
2. **S3** - S3 storage integration
3. **AZURE** - Azure Blob Storage
4. **HDFS** - Hadoop Distributed File System
5. **URL** - URL table engine
6. **MYSQL** - MySQL table engine
7. **POSTGRES** - PostgreSQL table engine
8. **MONGO** - MongoDB table engine
9. **KAFKA** - Kafka table engine

### Test Coverage for Expandable Privileges

The test suite includes **100% coverage** for all expandable privileges:

#### Individual Privilege Tests
`TestAccResourceRole_ExpandablePrivileges` - Tests each privilege individually across multiple apply cycles:

```go
// Tests: REMOTE, S3, AZURE, HDFS, URL, MYSQL, POSTGRES, MONGO, KAFKA
// Steps:
// 1. Create role with single expandable privilege
// 2. Verify state contains original privilege (not expanded)
// 3. Re-apply same config - expect no changes (ExpectNonEmptyPlan: false)
```

#### Multiple Privilege Tests
`TestAccResourceRole_MultipleExpandablePrivileges` - Tests combinations:
```go
// Tests: ["REMOTE", "S3", "HDFS"]
// Ensures multiple expandable privileges don't interfere with each other
```

#### Mixed Privilege Tests
`TestAccResourceRole_MixedExpandableAndNonExpandable` - Tests with global non-expandable:
```go
// Tests: ["REMOTE", "S3", "SYSTEM FLUSH LOGS"]
// Verifies expandable and non-expandable coexist correctly
```

#### Complex Scenarios
`TestAccResourceRole_ComplexGlobalPrivilegeMix` - Comprehensive stress test:
```go
// Tests: ["REMOTE", "S3", "AZURE", "HDFS", "SYSTEM FLUSH LOGS",
//         "SYSTEM RELOAD DICTIONARY", "CREATE TEMPORARY TABLE", "CREATE FUNCTION"]
// Steps: 4 consecutive apply cycles to test long-term stability
```

#### Multiple Apply Cycles
`TestAccResourceRole_MultipleApplyCycles` - Tests stability across 5+ apply cycles for:
- Single expandable privilege
- Multiple expandable privileges
- Mixed privilege sets
- All 9 expandable privileges together

#### Revoke Testing
- `TestAccResourceRole_RevokeExpandablePrivileges` - Tests revoking expandable privileges
- `TestAccResourceRole_PrivilegeUpdates` - Tests updating from one expandable to another

### Total Apply Cycles: 54+

The test suite executes **54+ terraform apply cycles** specifically testing expandable privilege behavior, ensuring zero state drift across all scenarios.

## Test Implementation Details

### Acceptance Test Pattern

All acceptance tests follow the Terraform SDK test framework pattern:

```go
func TestAccResourceX(t *testing.T) {
    resource.Test(t, resource.TestCase{
        Providers:    testutils.Provider(),
        CheckDestroy: testAccCheckXResourceDestroy([]string{"resource_name"}),
        Steps: []resource.TestStep{
            {
                // Step 1: Create resource
                Config: `resource "clickhouse_x" "test" { ... }`,
                Check: resource.ComposeTestCheckFunc(
                    resource.TestCheckResourceAttr("clickhouse_x.test", "name", "value"),
                    testAccCheckXExistsInClickHouse("expected_name"),
                ),
            },
            {
                // Step 2: Update resource
                Config: `resource "clickhouse_x" "test" { ... updated ... }`,
                Check: resource.ComposeTestCheckFunc(...),
            },
            {
                // Step 3: Verify no drift
                Config: `resource "clickhouse_x" "test" { ... same as step 2 ... }`,
                ExpectNonEmptyPlan: false,  // Critical: Must be no changes
            },
        },
    })
}
```

### Custom Check Functions

Tests include helper functions that query ClickHouse directly:

```go
// Check if role exists (even with no privileges)
func testAccCheckRoleExistsInClickHouse(roleName string) resource.TestCheckFunc

// Check if role has specific privileges
func testAccCheckRoleHasPrivileges(roleName string, expectedPrivileges []string) resource.TestCheckFunc

// Check if role has privileges on specific database
func testAccCheckRoleHasPrivilegesOnDatabase(roleName, database string, expectedPrivileges []string) resource.TestCheckFunc

// Check if role has NO privileges
func testAccCheckRoleHasNoPrivileges(roleName string) resource.TestCheckFunc
```

### Test Utilities

Located in `pkg/testutils/testutils.go`:

```go
// Check set/list attributes (order-independent for sets)
testutils.CheckStateSetAttr("privileges", "clickhouse_role.test", []string{"SELECT", "INSERT"})

// Get provider for tests
testutils.Provider()

// Pre-check for acceptance tests
testutils.TestAccPreCheck(t)
```

### Skipped Tests

Some tests are skipped to document desired future behavior:

**Example: `TestAccResourceRole_ChangeDatabaseAndPrivileges`**
```go
t.Skip("SKIPPED: Resource model does not yet support multiple databases per role. " +
    "This test documents the DESIRED behavior where a role can accumulate privileges " +
    "across multiple databases...")
```

Skipped tests serve as:
1. Documentation of known limitations
2. Specification for future enhancements
3. Ready-to-run tests once features are implemented

## Coverage Metrics

### Privilege Coverage
- **Database-level privileges**: 14 privileges fully tested
- **Global privileges**: 11 privileges fully tested (including expandable)
- **Expandable privileges**: 9 privileges with 100% coverage
- **Total apply cycles**: 54+ specifically for expandable privilege stability

### Resource Coverage
- **Databases**: Create, Read, Update, Delete, Cluster support
- **Tables**: Multiple engines (MergeTree, ReplicatedMergeTree, Distributed)
- **Roles**: All privilege types, database changes, name changes
- **Users**: Creation, password management, role assignment

### Test Execution Metrics
- **Acceptance tests**: 120m timeout (full suite)
- **Unit tests**: 30s timeout (per package)
- **Parallel execution**: Yes (for unit tests)

## Troubleshooting

### Common Issues

#### Test Skipped: "TF_ACC not set"
**Problem:** Acceptance tests require `TF_ACC=1` environment variable.
```
=== RUN   TestAccResourceRole
--- SKIP: TestAccResourceRole (0.00s)
```

**Solution:**
```bash
export TF_ACC=1
# OR prepend to test command
TF_ACC=1 go test ./...
```

#### Connection Refused
**Problem:** ClickHouse not running or wrong port.
```
error: dial tcp 127.0.0.1:9000: connect: connection refused
```

**Solution:**
```bash
# Verify ClickHouse is running
docker ps | grep clickhouse

# Start ClickHouse
docker run -d -p 9000:9000 clickhouse/clickhouse-server

# Verify port 9000 (native protocol), not 8123 (HTTP)
nc -zv 127.0.0.1 9000
```

#### Authentication Failed
**Problem:** Wrong username/password.
```
error: authentication failed
```

**Solution:**
```bash
# Default user has empty password
TF_CLICKHOUSE_USERNAME=default TF_CLICKHOUSE_PASSWORD="" go test ./...

# For custom credentials
TF_CLICKHOUSE_USERNAME=myuser TF_CLICKHOUSE_PASSWORD=mypass go test ./...
```

#### State Drift Detected
**Problem:** Test fails with `ExpectNonEmptyPlan: false` but plan is not empty.

**Investigation:**
```bash
# Enable verbose output
TF_ACC=1 TF_LOG=DEBUG go test -v -run "TestName" ./...

# Check if using SHOW GRANTS vs system.grants
# Provider MUST use SHOW GRANTS for expandable privileges
```

**Common Causes:**
1. Using `system.grants` instead of `SHOW GRANTS` (returns expanded privileges)
2. Case sensitivity in privilege names
3. Database scope mismatch (*.* vs database.*)

#### Timeout Errors
**Problem:** Tests timeout before completion.
```
panic: test timed out after 2m0s
```

**Solution:**
```bash
# Increase timeout for acceptance tests
go test -timeout 120m ./...

# For single slow test
go test -timeout 30m -run "TestAccResourceRole_ComplexGlobalPrivilegeMix" ./pkg/resources/role/...
```

### Debugging Tips

1. **Use Test Focus:**
```bash
# Run single test function
go test -run "^TestAccResourceRole_ExpandablePrivileges$" ./pkg/resources/role/...

# Run single subtest
go test -run "TestAccResourceRole_ExpandablePrivileges/REMOTE" ./pkg/resources/role/...
```

2. **Enable Detailed Logging:**
```bash
# Full Terraform logging
TF_ACC=1 TF_LOG=TRACE go test -v ./...

# Provider-specific logging
TF_ACC=1 TF_LOG_PROVIDER=TRACE go test -v ./...
```

3. **Inspect ClickHouse State:**
```bash
# Connect to ClickHouse during test
docker exec -it clickhouse-server clickhouse-client

# Check roles
SHOW ROLES;
SHOW GRANTS FOR role_name;

# Check databases
SHOW DATABASES;
```

4. **Isolate Failures:**
```bash
# Run tests sequentially to avoid race conditions
go test -p 1 -parallel 1 ./...
```

### Getting Help

If you encounter issues not covered here:

1. Check `CLAUDE.md` for provider architecture details
2. Review test implementation in `*_acceptance_test.go` files
3. Examine service layer in `service.go` for SQL generation
4. File an issue with:
   - Test output (with `-v` flag)
   - ClickHouse version
   - Provider version
   - Full test command used
