# Issue Documentation

This directory contains detailed documentation for significant issues in the Terraform ClickHouse Provider, both active and resolved.

## Purpose

Issue documentation serves to:
- Provide comprehensive technical details for complex issues
- Document root causes and solutions for future reference
- Help users understand fixes and migration paths
- Capture lessons learned for preventing similar issues

## Documentation Format

Each issue document should include:

### Required Sections
- **Status**: Current state (Active, Resolved, Under Investigation)
- **Severity**: Impact level (Critical, High, Medium, Low)
- **Component**: Affected resource or feature
- **Summary**: Brief description of the issue
- **Problem Description**: Detailed technical explanation
- **Impact**: User and system impact
- **Reproduction**: Minimal reproduction case
- **Solution**: Fix description (if resolved)
- **Verification**: How to verify the fix
- **Migration Guide**: User upgrade instructions

### Optional Sections
- **Related Documentation**: Links to relevant docs
- **Prevention**: How to avoid similar issues
- **Timeline**: Issue lifecycle dates
- **Additional Notes**: Technical context

## Active Issues

**None currently.** All known issues have been resolved.

## Resolved Issues

### High Severity

#### [REMOTE Privilege State Drift](./remote-privilege-state-drift.md)
**Status:** ✅ RESOLVED (v3.9.0)
**Component:** Role Resource
**Description:** Expandable SOURCE privileges (REMOTE, S3, AZURE, HDFS, URL, MYSQL, POSTGRES, MONGO, KAFKA) caused false state drift detection due to ClickHouse privilege expansion. Provider was reading expanded privileges from `system.grants` instead of original grants.

**Fix:** Changed privilege retrieval to use `SHOW GRANTS` command instead of querying `system.grants` table, ensuring original privileges are preserved in Terraform state.

**Coverage:** 54+ test apply cycles, 100% expandable privilege coverage

**See Also:**
- [docs/guides/revoke-fix.md](../../docs/guides/revoke-fix.md) - Detailed fix explanation
- [internal-docs/TESTING.md](../TESTING.md) - Test coverage documentation

## Known Limitations

These are documented design limitations, not bugs:

### Single Database Per Role
**Component:** Role Resource
**Description:** The `clickhouse_role` resource currently supports privileges for only one database per role instance.

**Current Behavior:**
```hcl
resource "clickhouse_role" "example" {
  name       = "my_role"
  database   = "db1"
  privileges = ["SELECT"]
}
# Role has SELECT on db1
# Updating database to "db2" revokes privileges on db1
```

**Desired Future Behavior:**
```hcl
# Multiple role resources accumulating privileges
resource "clickhouse_role" "example_db1" {
  name       = "my_role"
  database   = "db1"
  privileges = ["SELECT"]
}

resource "clickhouse_role" "example_db2" {
  name       = "my_role"
  database   = "db2"
  privileges = ["INSERT"]
}
# Role should have SELECT on db1 AND INSERT on db2
```

**Workaround:** Use separate roles for different databases:
```hcl
resource "clickhouse_role" "db1_reader" {
  name       = "db1_reader"
  database   = "db1"
  privileges = ["SELECT"]
}

resource "clickhouse_role" "db2_writer" {
  name       = "db2_writer"
  database   = "db2"
  privileges = ["INSERT"]
}

resource "clickhouse_user" "app_user" {
  name     = "app_user"
  password = var.password
  roles    = [
    clickhouse_role.db1_reader.name,
    clickhouse_role.db2_writer.name
  ]
}
```

**Documented In:**
- Skipped test: `TestAccResourceRole_ChangeDatabaseAndPrivileges`
- `CLAUDE.md` - Known Limitations section

### Replicated Table Engine Support
**Component:** Table Resource
**Description:** Limited support for replicated table engines due to engine parameter complexity.

**Supported:**
- `ReplicatedMergeTree` with explicit replica paths
- `Distributed` tables

**Limited Support:**
- Complex replication parameters
- Auto-generated replica paths
- Advanced MergeTree variants

**See:** Provider README.md for current engine support status

## Reporting New Issues

### Before Filing an Issue

1. **Check existing documentation:**
   - Review this README
   - Check resolved issues (may already be fixed)
   - Read [internal-docs/TESTING.md](../TESTING.md) for troubleshooting

2. **Verify the issue:**
   - Reproduce with minimal configuration
   - Test with latest provider version
   - Check if it's a known limitation

3. **Gather information:**
   - Provider version
   - ClickHouse version
   - Terraform version
   - Minimal reproduction case
   - Expected vs actual behavior
   - Error messages (full output)

### Filing an Issue

Create a GitHub issue with:

**Title:** Brief, descriptive summary

**Template:**
```markdown
## Environment
- Provider Version: X.Y.Z
- ClickHouse Version: XX.YY.ZZ
- Terraform Version: X.Y.Z
- Operating System: OS/Arch

## Description
[Clear description of the issue]

## Expected Behavior
[What should happen]

## Actual Behavior
[What actually happens]

## Reproduction Steps
1. [First step]
2. [Second step]
3. [...]

## Minimal Configuration
```hcl
[Minimal HCL that reproduces the issue]
```

## Error Output
```
[Full error messages or unexpected output]
```

## Additional Context
[Any other relevant information]
```

## Contributing Issue Documentation

If you're documenting a new issue:

1. **Create new markdown file:** `internal-docs/issues/descriptive-name.md`
2. **Follow the format:** Use resolved issues as template
3. **Be comprehensive:** Include all required sections
4. **Link related docs:** Reference guides, tests, architecture docs
5. **Update this README:** Add entry to Active or Resolved sections

### Documentation Standards

- **Technical Accuracy**: Verify all technical details
- **Reproducibility**: Include working reproduction cases
- **Completeness**: Cover problem, impact, solution, verification
- **Clarity**: Write for both users and developers
- **Cross-references**: Link to related documentation

## Directory Structure

```
internal-docs/issues/
├── README.md                           # This file
├── remote-privilege-state-drift.md     # Resolved: Expandable privilege issue
└── [future-issue].md                   # Future issue documentation
```

## Related Documentation

- **Testing Guide**: [internal-docs/TESTING.md](../TESTING.md)
- **Setup Guide**: [docs/guides/setup.md](../../docs/guides/setup.md)
- **Revoke Fix**: [docs/guides/revoke-fix.md](../../docs/guides/revoke-fix.md)
- **Architecture**: [CLAUDE.md](../../CLAUDE.md)
- **Main README**: [README.md](../../README.md)

## Maintenance

This directory should be reviewed and updated:
- When new issues are discovered
- When issues are resolved
- When provider architecture changes significantly
- During major version releases

**Last Updated:** 2026-01-29
