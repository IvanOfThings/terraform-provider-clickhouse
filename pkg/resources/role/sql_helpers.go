package role

import (
	"fmt"
	"github.com/IvanOfThings/terraform-provider-clickhouse/pkg/common"
	ch "github.com/leprosus/golang-clickhouse"
)

func getRoleGrants(conn *ch.Conn, roleName string, errors *[]error) []CHGrant {
	query := fmt.Sprintf("SELECT role_name, access_type, database, table, column FROM system.grants WHERE role_name = '%s'", roleName)
	grantsIt, err := conn.Fetch(query)

	if err != nil {
		*errors = append(*errors, err)
		return nil
	}

	var privileges []CHGrant

	for i := 0; grantsIt.Next(); i++ {
		result := grantsIt.Result

		roleName := *common.ToString(result, "role_name", errors)
		accessType := *common.ToString(result, "access_type", errors)
		database := *common.ToString(result, "database", errors)
		if len(*errors) > 0 {
			return nil
		}

		privilege := CHGrant{
			RoleName:   roleName,
			Database:   database,
			AccessType: accessType,
		}

		privileges = append(privileges, privilege)
	}

	return privileges
}

func GetRole(conn *ch.Conn, roleName string, errors *[]error) *CHRole {
	roleQuery := fmt.Sprintf("SELECT name FROM system.roles WHERE name = '%s'", roleName)

	roleIt, err := conn.Fetch(roleQuery)
	if err != nil {
		*errors = append(*errors, err)
		return nil
	}
	if roleIt.Next() == false {
		return nil
	}

	privileges := getRoleGrants(conn, roleName, errors)
	if len(*errors) > 0 {
		return nil
	}

	return &CHRole{
		Name:       *common.ToString(roleIt.Result, "name", errors),
		Privileges: privileges,
	}
}
