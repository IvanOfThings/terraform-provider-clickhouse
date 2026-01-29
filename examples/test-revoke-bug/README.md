# Test Revoke Bug Example

This example demonstrates manual testing of the revoke bug with global privileges.

> **Note:** For comprehensive bug documentation, see [docs/issues/remote-privilege-state-drift.md](../../docs/issues/remote-privilege-state-drift.md)
>
> For automated test coverage, see [docs/TESTING.md](../../docs/TESTING.md)

## Quick Test

This example reproduces the revoke functionality with global privileges (REMOTE, S3, AZURE).

### Prerequisites

```bash
# Start ClickHouse
cd ../..
docker run -d -p 9000:9000 -p 8123:8123 clickhouse/clickhouse-server

# Build provider
make build
```

### Run Example

```bash
cd examples/test-revoke-bug

# Initialize and apply
terraform init
terraform apply

# Verify role created
docker exec <container> clickhouse-client --query "SHOW GRANTS FOR test_revoke_bug"
```

Expected output:
```
GRANT REMOTE ON *.* TO test_revoke_bug
GRANT S3 ON *.* TO test_revoke_bug
GRANT AZURE ON *.* TO test_revoke_bug
```

### Test Revoke

Edit `main.tf` to remove a privilege (e.g., remove `"AZURE"`), then apply:

```bash
terraform apply
```

Verify privilege removed:
```bash
docker exec <container> clickhouse-client --query "SHOW GRANTS FOR test_revoke_bug"
```

Should now show only REMOTE and S3.

### Cleanup

```bash
terraform destroy
```

## What This Tests

- Granting global privileges (REMOTE, S3, AZURE)
- Revoking global privileges
- Correct SQL syntax for global vs database privileges
- Consistency between grant and revoke operations

## Documentation

- **Bug Details:** [docs/issues/remote-privilege-state-drift.md](../../docs/issues/remote-privilege-state-drift.md)
- **Automated Tests:** [docs/TESTING.md](../../docs/TESTING.md)
- **Revoke Fix:** [docs/guides/REVOKE_FIX.md](../../docs/guides/REVOKE_FIX.md)
- **Setup Guide:** [docs/guides/SETUP.md](../../docs/guides/SETUP.md)
