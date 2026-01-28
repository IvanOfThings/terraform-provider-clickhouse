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

// parseGrantStatement parses SHOW GRANTS output to extract privilege information
// Example inputs:
//   "GRANT REMOTE ON *.* TO role_name"
//   "GRANT SELECT, INSERT ON database.* TO role_name"
//   "GRANT SELECT ON database.table TO role_name"
func parseGrantStatement(statement string, roleName string) ([]CHGrant, error) {
	// Remove "GRANT " prefix
	if !strings.HasPrefix(statement, "GRANT ") {
		return nil, fmt.Errorf("invalid grant statement: %s", statement)
	}
	statement = strings.TrimPrefix(statement, "GRANT ")

	// Find " ON " to split privileges from scope
	onIndex := strings.Index(statement, " ON ")
	if onIndex == -1 {
		return nil, fmt.Errorf("missing ' ON ' in grant statement: %s", statement)
	}

	// Extract privileges (e.g., "REMOTE" or "SELECT, INSERT")
	privilegesPart := statement[:onIndex]
	privilegeNames := strings.Split(privilegesPart, ",")

	// Extract scope (everything after " ON " and before " TO ")
	remainder := statement[onIndex+4:] // Skip " ON "
	toIndex := strings.Index(remainder, " TO ")
	if toIndex == -1 {
		return nil, fmt.Errorf("missing ' TO ' in grant statement: %s", statement)
	}
	scope := strings.TrimSpace(remainder[:toIndex])

	// Parse database from scope
	// "*.*" means global (database = "*")
	// "database.*" means database level
	// "database.table" means table level (we currently don't support this)
	var database string
	if scope == "*.*" {
		database = "*"
	} else if strings.HasSuffix(scope, ".*") {
		// Extract database name (remove ".*" suffix)
		database = strings.TrimSuffix(scope, ".*")
	} else {
		// Table-level grants are not supported in our provider
		return nil, fmt.Errorf("table-level grants are not supported: %s", scope)
	}

	// Create CHGrant for each privilege
	var grants []CHGrant
	for _, privilegeName := range privilegeNames {
		grants = append(grants, CHGrant{
			RoleName:   roleName,
			AccessType: strings.TrimSpace(privilegeName),
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
		err := conn.Exec(ctx, fmt.Sprintf("REVOKE %s ON %s.* FROM %s", strings.Join(revokePrivileges, ","), rolePlan.Database, rolePlan.Name))
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
