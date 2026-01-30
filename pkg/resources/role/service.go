package resourcerole

import (
	"context"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"strings"
)

type CHRoleService struct {
	CHConnection *driver.Conn
}

func getRevokeQuery(roleName string, privileges []string, database string) string {
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

	var queries []string

	// Revoke global privileges with ON *.* syntax
	if len(globalPrivileges) > 0 {
		queries = append(queries, fmt.Sprintf("REVOKE %s ON *.* FROM %s",
			strings.Join(globalPrivileges, ","), roleName))
	}

	// Revoke database-level privileges with appropriate syntax
	if len(dbPrivileges) > 0 {
		queries = append(queries, fmt.Sprintf("REVOKE %s ON %s.* FROM %s",
			strings.Join(dbPrivileges, ","), database, roleName))
	}

	return strings.Join(queries, "; ")
}

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

	var queries []string

	// Grant global privileges with ON *.* syntax (no CURRENT GRANTS wrapper)
	if len(globalPrivileges) > 0 {
		queries = append(queries, fmt.Sprintf("GRANT %s ON *.* TO %s",
			strings.Join(globalPrivileges, ","), roleName))
	}

	// Grant database-level privileges with appropriate syntax
	if len(dbPrivileges) > 0 {
		if database == "system" || database == "*" {
			queries = append(queries, fmt.Sprintf("GRANT CURRENT GRANTS (%s ON %s.*) TO %s",
				strings.Join(dbPrivileges, ","), database, roleName))
		} else {
			queries = append(queries, fmt.Sprintf("GRANT %s ON %s.* TO %s",
				strings.Join(dbPrivileges, ","), database, roleName))
		}
	}

	return strings.Join(queries, "; ")
}

func (rs *CHRoleService) getRoleGrants(ctx context.Context, roleName string) ([]CHGrant, error) {
	// Use SHOW GRANTS instead of SELECT from system.grants
	// SHOW GRANTS returns the original granted privileges (e.g., REMOTE)
	// system.grants returns expanded privileges (e.g., READ + WRITE)
	query := fmt.Sprintf("SHOW GRANTS FOR %s", roleName)
	rows, err := (*rs.CHConnection).Query(ctx, query)

	if err != nil {
		return nil, fmt.Errorf("error fetching role grants: %w", err)
	}

	var privileges []CHGrant
	for rows.Next() {
		var grantStatement string
		err := rows.Scan(&grantStatement)
		if err != nil {
			return nil, fmt.Errorf("error scanning grant statement: %w", err)
		}

		// Parse SHOW GRANTS output
		// Example: "GRANT REMOTE ON *.* TO role_name"
		// Example: "GRANT SELECT, INSERT ON database.* TO role_name"
		grants, err := parseGrantStatement(grantStatement, roleName)
		if err != nil {
			return nil, fmt.Errorf("error parsing grant statement '%s': %w", grantStatement, err)
		}
		privileges = append(privileges, grants...)
	}

	return privileges, nil
}

// parseGrantStatement parses a single ClickHouse SHOW GRANTS statement
// and returns the corresponding CHGrant entries.
//
// Supported forms:
//   - GRANT <privileges> ON *.* TO <role>
//   - GRANT <privileges> ON <database>.* TO <role>
//
// Notes:
//   - The role in the statement must match expectedRole
//   - Table-level grants (<database>.<table>) are currently rejected
//   - Privileges are split on commas and normalized via trimming
func parseGrantStatement(statement, expectedRole string) ([]CHGrant, error) {
	stmt := strings.TrimSpace(statement)

	// Normalize keyword case for parsing, but keep original tokens
	upper := strings.ToUpper(stmt)

	if !strings.HasPrefix(upper, "GRANT ") {
		return nil, fmt.Errorf("invalid grant statement (missing GRANT): %q", statement)
	}

	// Split into GRANT <privileges> ON <scope> TO <role>
	grantBody := stmt[len("GRANT "):]
	upperBody := upper[len("GRANT "):]

	onIdx := strings.Index(upperBody, " ON ")
	if onIdx == -1 {
		return nil, fmt.Errorf("missing ' ON ' in grant statement: %s", statement)
	}

	privPart := strings.TrimSpace(grantBody[:onIdx])
	rest := grantBody[onIdx+4:]
	upperRest := upperBody[onIdx+4:]

	toIdx := strings.Index(upperRest, " TO ")
	if toIdx == -1 {
		return nil, fmt.Errorf("missing ' TO ' in grant statement: %s", statement)
	}

	scopePart := strings.TrimSpace(rest[:toIdx])
	rolePart := strings.TrimSpace(rest[toIdx+4:])

	if rolePart != expectedRole {
		return nil, fmt.Errorf(
			"grant role mismatch: expected %q, got %q",
			expectedRole, rolePart,
		)
	}

	// Parse privileges
	rawPrivileges := strings.Split(privPart, ",")
	privileges := make([]string, 0, len(rawPrivileges))
	for _, p := range rawPrivileges {
		p = strings.TrimSpace(p)
		if p == "" {
			return nil, fmt.Errorf("empty privilege in grant: %q", statement)
		}
		privileges = append(privileges, p)
	}

	// Parse scope
	var database string
	switch {
	case scopePart == "*.*":
		database = "*"

	case strings.HasSuffix(scopePart, ".*"):
		database = strings.TrimSuffix(scopePart, ".*")
		if database == "" {
			return nil, fmt.Errorf("invalid database scope: %q", scopePart)
		}

	default:
		return nil, fmt.Errorf("table-level grants are not supported: %s", scopePart)
	}

	grants := make([]CHGrant, 0, len(privileges))
	for _, p := range privileges {
		grants = append(grants, CHGrant{
			RoleName:   expectedRole,
			AccessType: p,
			Database:   database,
		})
	}

	return grants, nil
}


func (rs *CHRoleService) GetRole(ctx context.Context, roleName string) (*CHRole, error) {
	roleQuery := fmt.Sprintf("SELECT name FROM system.roles WHERE name = '%s'", roleName)

	rows, err := (*rs.CHConnection).Query(ctx, roleQuery)
	if err != nil {
		return nil, fmt.Errorf("error fetching role: %s", err)
	}
	if rows.Next() == false {
		return nil, nil
	}

	privileges, err := rs.getRoleGrants(ctx, roleName)
	if err != nil {
		return nil, fmt.Errorf("error fetching role grants: %s", err)
	}

	return &CHRole{
		Name:       roleName,
		Privileges: privileges,
	}, nil
}

func (rs *CHRoleService) UpdateRole(ctx context.Context, rolePlan RoleResource, resourceData *schema.ResourceData) (*CHRole, error) {
	stateRoleName, _ := resourceData.GetChange("name")
	chRole, err := rs.GetRole(ctx, stateRoleName.(string))

	if err != nil {
		return nil, fmt.Errorf("error fetching role: %s", err)
	}
	if chRole == nil {
		return nil, fmt.Errorf("role %s not found", rolePlan.Name)
	}

	roleNameHasChange := resourceData.HasChange("name")
	roleDatabaseHasChange := resourceData.HasChange("database")
	rolePrivilegesHasChange := resourceData.HasChange("privileges")

	var grantPrivileges []string
	var revokePrivileges []string
	if rolePrivilegesHasChange {
		for _, planPrivilege := range rolePlan.Privileges.List() {
			found := false
			for _, privilege := range chRole.Privileges {
				if privilege.AccessType == planPrivilege {
					found = true
				}
			}
			if found == false {
				grantPrivileges = append(grantPrivileges, planPrivilege.(string))
			}
		}

		for _, privilege := range chRole.Privileges {
			if rolePlan.Privileges.Contains(privilege.AccessType) == false {
				revokePrivileges = append(revokePrivileges, privilege.AccessType)
			}
		}
	}

	conn := *rs.CHConnection

	if roleNameHasChange {
		err := conn.Exec(ctx, fmt.Sprintf("ALTER ROLE %s RENAME TO %s", chRole.Name, rolePlan.Name))
		if err != nil {
			return nil, fmt.Errorf("error renaming role %s to %s: %v", chRole.Name, rolePlan.Name, err)
		}
	}

	if roleDatabaseHasChange {
		err := conn.Exec(ctx, fmt.Sprintf("REVOKE ALL ON *.* FROM %s", rolePlan.Name))
		if err != nil {
			return nil, fmt.Errorf("error revoking all privileges from role %s: %v", chRole.Name, err)
		}
		dbPrivileges := chRole.GetPrivilegesList()
		err = conn.Exec(ctx, getGrantQuery(
			rolePlan.Name,
			dbPrivileges,
			rolePlan.Database,
		))
		if err != nil {
			return nil, fmt.Errorf("error granting privileges to role %s: %v", chRole.Name, err)
		}
	}

	if len(grantPrivileges) > 0 {
		err := conn.Exec(ctx, getGrantQuery(rolePlan.Name, grantPrivileges, rolePlan.Database))
		if err != nil {
			return nil, fmt.Errorf("error granting privileges to role %s: %v", chRole.Name, err)
		}
	}

	if len(revokePrivileges) > 0 {
		err := conn.Exec(ctx, getRevokeQuery(rolePlan.Name, revokePrivileges, rolePlan.Database))
		if err != nil {
			return nil, fmt.Errorf("error revoking privileges from role %s: %v", chRole.Name, err)
		}
	}

	return rs.GetRole(ctx, rolePlan.Name)
}

func (rs *CHRoleService) CreateRole(ctx context.Context, name string, database string, privileges []string) (*CHRole, error) {
	conn := *rs.CHConnection
	err := conn.Exec(ctx, fmt.Sprintf("CREATE ROLE %s", name))
	if err != nil {
		return nil, fmt.Errorf("error creating role: %s", err)
	}

	var chPrivileges []CHGrant

	for _, privilege := range privileges {
		err = conn.Exec(ctx, getGrantQuery(name, []string{privilege}, database))
		if err != nil {
			// Rollback
			err2 := conn.Exec(ctx, fmt.Sprintf("DROP ROLE %s", name))
			if err2 != nil {
				return nil, fmt.Errorf("error creating role: %s:%s", err, err2)
			}
			return nil, fmt.Errorf("error creating role: %s", err)
		}
		chPrivileges = append(chPrivileges, CHGrant{RoleName: name, AccessType: privilege, Database: database})
	}
	return &CHRole{Name: name, Privileges: chPrivileges}, nil
}

func (rs *CHRoleService) DeleteRole(ctx context.Context, name string) error {
	return (*rs.CHConnection).Exec(ctx, fmt.Sprintf("DROP ROLE %s", name))
}
