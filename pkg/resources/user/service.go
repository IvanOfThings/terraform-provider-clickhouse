package user

import (
	"encoding/json"
	"fmt"
	"github.com/IvanOfThings/terraform-provider-clickhouse/pkg/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ch "github.com/leprosus/golang-clickhouse"
	"strings"
)

type CHUserService struct {
	CHConnection *ch.Conn
}

func (us *CHUserService) GetUser(userName string) (*CHUser, error) {
	roleQuery := fmt.Sprintf("SELECT name, default_roles_list FROM system.users WHERE name = '%s'", userName)

	roleIt, err := us.CHConnection.Fetch(roleQuery)
	if err != nil {
		return nil, fmt.Errorf("error fetching user: %s", err)
	}
	if roleIt.Next() == false {
		return nil, nil
	}
	name, err := roleIt.Result.String("name")
	if err != nil {
		return nil, fmt.Errorf("error retrieving user 'name': %s", err)
	}

	rolesStr, err := roleIt.Result.String("default_roles_list")
	if err != nil {
		return nil, fmt.Errorf("error retrieving user 'default_roles_list': %s", err)
	}
	var rolesList []string
	err = json.Unmarshal([]byte(strings.ReplaceAll(rolesStr, "'", "\"")), &rolesList)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling user 'default_roles_list': %s", err)
	}

	return &CHUser{
		Name:  name,
		Roles: rolesList,
	}, nil
}

func (us *CHUserService) CreateUser(userPlan UserResource) (*CHUser, error) {
	var rolesList []string

	for _, role := range userPlan.Roles.List() {
		rolesList = append(rolesList, role.(string))
	}
	query := fmt.Sprintf(
		"CREATE USER %s IDENTIFIED WITH sha256_password BY '%s'",
		userPlan.Name,
		userPlan.Password,
	)

	if len(rolesList) > 0 {
		query = fmt.Sprintf("%s DEFAULT ROLE %s", query, strings.Join(rolesList, ","))
	}
	err := us.CHConnection.Exec(query)
	if err != nil {
		return nil, fmt.Errorf("error creating user: %s", err)
	}
	return us.GetUser(userPlan.Name)
}

func (us *CHUserService) UpdateUser(userPlan UserResource, resourceData *schema.ResourceData) (*CHUser, error) {
	stateUserName, _ := resourceData.GetChange("name")
	user, err := us.GetUser(stateUserName.(string))
	if err != nil {
		return nil, fmt.Errorf("error fetching user: %s", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user %s not found", userPlan.Name)
	}

	userNameHasChange := resourceData.HasChange("name")
	userPasswordHasChange := resourceData.HasChange("password")
	userRolesHasChange := resourceData.HasChange("roles")

	var grantRoles []string
	var revokeRoles []string
	if userRolesHasChange {
		for _, planRole := range userPlan.Roles.List() {
			found := false
			for _, role := range user.Roles {
				if role == planRole {
					found = true
				}
			}
			if found == false {
				grantRoles = append(grantRoles, planRole.(string))
			}
		}

		for _, role := range user.Roles {
			if userPlan.Roles.Contains(role) == false {
				revokeRoles = append(revokeRoles, role)
			}
		}
	}

	if len(grantRoles) > 0 {
		err := us.CHConnection.Exec(fmt.Sprintf("GRANT %s TO %s", strings.Join(grantRoles, ","), stateUserName))
		if err != nil {
			return nil, fmt.Errorf("error granting roles to user: %s", err)
		}
	}

	if len(revokeRoles) > 0 {
		err := us.CHConnection.Exec(fmt.Sprintf("REVOKE %s FROM %s", strings.Join(revokeRoles, ","), stateUserName))
		if err != nil {
			return nil, fmt.Errorf("error revoking roles from user: %s", err)
		}
	}

	var changeNameClause string
	var changePasswordClause string

	if userNameHasChange {
		changeNameClause = fmt.Sprintf(" RENAME TO %s", userPlan.Name)
	}

	if userPasswordHasChange {
		changePasswordClause = fmt.Sprintf(" IDENTIFIED with sha256_password BY '%s'", userPlan.Password)
	}

	// After modify original role grants, we need to update default roles
	query := fmt.Sprintf(
		"ALTER USER %s%s%s DEFAULT ROLE %s",
		stateUserName,
		changeNameClause,
		changePasswordClause,
		strings.Join(common.StringSetToList(userPlan.Roles), ","),
	)
	err = us.CHConnection.Exec(query)
	if err != nil {
		return nil, fmt.Errorf("error updating user: %s", err)
	}

	return us.GetUser(userPlan.Name)
}

func (us *CHUserService) DeleteUser(name string) error {
	return us.CHConnection.Exec(fmt.Sprintf("DROP USER %s", name))
}
