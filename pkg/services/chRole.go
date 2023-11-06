package services

import (
	"fmt"
	"github.com/IvanOfThings/terraform-provider-clickhouse/pkg/model"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ch "github.com/leprosus/golang-clickhouse"
	"strings"
)

type CHRoleService struct {
	CHConnection *ch.Conn
}

func (rs *CHRoleService) getRoleGrants(roleName string) ([]model.CHGrant, error) {
	query := fmt.Sprintf("SELECT role_name, access_type, database, table, column FROM system.grants WHERE role_name = '%s'", roleName)
	grantsIt, err := rs.CHConnection.Fetch(query)

	if err != nil {
		return nil, fmt.Errorf("error fetching role grants: %s", err)
	}

	var privileges []model.CHGrant

	for i := 0; grantsIt.Next(); i++ {
		result := grantsIt.Result

		roleName, err := result.String("role_name")
		if err != nil {
			return nil, fmt.Errorf("error retrieving role 'role_name': %s", err)
		}
		accessType, err := result.String("access_type")
		if err != nil {
			return nil, fmt.Errorf("error retrieving role 'access_type': %s", err)
		}
		database, err := result.String("database")
		if err != nil {
			return nil, fmt.Errorf("error retrieving role 'database': %s", err)
		}

		privilege := model.CHGrant{
			RoleName:   roleName,
			Database:   database,
			AccessType: accessType,
		}

		privileges = append(privileges, privilege)
	}

	return privileges, nil
}

func (rs *CHRoleService) GetRole(roleName string) (*model.CHRole, error) {
	roleQuery := fmt.Sprintf("SELECT name FROM system.roles WHERE name = '%s'", roleName)

	roleIt, err := rs.CHConnection.Fetch(roleQuery)
	if err != nil {
		return nil, fmt.Errorf("error fetching role: %s", err)
	}
	if roleIt.Next() == false {
		return nil, nil
	}

	privileges, err := rs.getRoleGrants(roleName)
	if err != nil {
		return nil, fmt.Errorf("error fetching role grants: %s", err)
	}

	return &model.CHRole{
		Name:       roleName,
		Privileges: privileges,
	}, nil
}

func (rs *CHRoleService) UpdateRole(rolePlan model.RoleResource, resourceData *schema.ResourceData) (*model.CHRole, error) {
	stateRoleName, _ := resourceData.GetChange("name")
	chRole, err := rs.GetRole(stateRoleName.(string))
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

	if roleNameHasChange {
		err := rs.CHConnection.Exec(fmt.Sprintf("ALTER ROLE %s RENAME TO %s", chRole.Name, rolePlan.Name))
		if err != nil {
			return nil, fmt.Errorf("error renaming role %s to %s: %v", chRole.Name, rolePlan.Name, err)
		}
	}

	if roleDatabaseHasChange {
		err := rs.CHConnection.Exec(fmt.Sprintf("REVOKE ALL ON *.* FROM %s", rolePlan.Name))
		if err != nil {
			return nil, fmt.Errorf("error revoking all privileges from role %s: %v", chRole.Name, err)
		}
		dbPrivileges := chRole.GetPrivilegesList()
		err = rs.CHConnection.Exec(fmt.Sprintf(
			"GRANT %s ON %s.* TO %s",
			strings.Join(dbPrivileges, ","),
			rolePlan.Database,
			rolePlan.Name,
		))
		if err != nil {
			return nil, fmt.Errorf("error granting privileges to role %s: %v", chRole.Name, err)
		}
	}

	if len(grantPrivileges) > 0 {
		err := rs.CHConnection.Exec(fmt.Sprintf("GRANT %s ON %s.* TO %s", strings.Join(grantPrivileges, ","), rolePlan.Database, rolePlan.Name))
		if err != nil {
			return nil, fmt.Errorf("error granting privileges to role %s: %v", chRole.Name, err)
		}
	}

	if len(revokePrivileges) > 0 {
		err := rs.CHConnection.Exec(fmt.Sprintf("REVOKE %s ON %s.* FROM %s", strings.Join(revokePrivileges, ","), rolePlan.Database, rolePlan.Name))
		if err != nil {
			return nil, fmt.Errorf("error revoking privileges from role %s: %v", chRole.Name, err)
		}
	}

	return rs.GetRole(rolePlan.Name)
}

func (rs *CHRoleService) CreateRole(name string, database string, privileges []string) (*model.CHRole, error) {
	err := rs.CHConnection.Exec(fmt.Sprintf("CREATE ROLE %s", name))
	if err != nil {
		return nil, fmt.Errorf("error creating role: %s", err)
	}

	var chPrivileges []model.CHGrant

	for _, privilege := range privileges {
		err = rs.CHConnection.Exec(fmt.Sprintf("GRANT %s ON %s.* TO %s", privilege, database, name))
		if err != nil {
			// Rollback
			err2 := rs.CHConnection.Exec(fmt.Sprintf("DROP ROLE %s", name))
			if err2 != nil {
				return nil, fmt.Errorf("error creating role: %s:%s", err, err2)
			}
			return nil, fmt.Errorf("error creating role: %s", err)
		}
		chPrivileges = append(chPrivileges, model.CHGrant{RoleName: name, AccessType: privilege, Database: database})
	}
	return &model.CHRole{Name: name, Privileges: chPrivileges}, nil
}

func (rs *CHRoleService) DeleteRole(name string) error {
	return rs.CHConnection.Exec(fmt.Sprintf("DROP ROLE %s", name))
}
