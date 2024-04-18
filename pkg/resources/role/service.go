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
	if database == "system" {
		return fmt.Sprintf("GRANT CURRENT GRANTS (%s ON %s.*) TO %s", strings.Join(privileges, ","), database, roleName)
	}
	return fmt.Sprintf("GRANT %s ON %s.* TO %s", strings.Join(privileges, ","), database, roleName)
}

func (rs *CHRoleService) getRoleGrants(ctx context.Context, roleName string) ([]CHGrant, error) {
	query := fmt.Sprintf("SELECT role_name, access_type, database FROM system.grants WHERE role_name = '%s'", roleName)
	rows, err := (*rs.CHConnection).Query(ctx, query)

	if err != nil {
		return nil, fmt.Errorf("error fetching role grants: %s", err)
	}

	var privileges []CHGrant
	for rows.Next() {
		var privilege CHGrant
		err := rows.ScanStruct(&privilege)
		if err != nil {
			return nil, fmt.Errorf("error scanning role grant: %s", err)
		}
		privileges = append(privileges, privilege)
	}

	return privileges, nil
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
