package role

import (
	"context"
	"fmt"
	"github.com/IvanOfThings/terraform-provider-clickhouse/pkg/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"strings"
)

func ResourceRole() *schema.Resource {
	return &schema.Resource{
		Description:   "Resource to manage Clickhouse roles",
		CreateContext: resourceRoleCreate,
		ReadContext:   resourceRoleRead,
		DeleteContext: resourceRoleDelete,
		UpdateContext: resourceRoleUpdate,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Description: "Role name",
				Type:        schema.TypeString,
				Required:    true,
			},
			"database": &schema.Schema{
				Description: "Database where to grant permissions to the user",
				Type:        schema.TypeString,
				Required:    true,
			},
			"privileges": &schema.Schema{
				Description: "Granted privileges to the role. Privileges will be granted at DB level",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func resourceRoleUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := meta.(*common.ApiClient)
	conn := client.ClickhouseConnection

	stateRoleName, planRoleName := d.GetChange("name")
	planDatabase := d.Get("database").(string)
	planPrivileges := d.Get("privileges").(*schema.Set)

	errors := make([]error, 0)
	role := GetRole(conn, stateRoleName.(string), &errors)
	if len(errors) > 0 {
		return diag.FromErr(errors[0])
	}
	if role == nil {
		return diag.FromErr(fmt.Errorf("role %s not found", planRoleName))
	}

	roleNameHasChange := d.HasChange("name")
	roleDatabaseHasChange := d.HasChange("database")
	rolePrivilegesHasChange := d.HasChange("privileges")

	var grantPrivileges []string
	var revokePrivileges []string
	if rolePrivilegesHasChange {
		for _, planPrivilege := range planPrivileges.List() {
			found := false
			for _, privilege := range role.Privileges {
				if privilege.AccessType == planPrivilege {
					found = true
				}
			}
			if found == false {
				grantPrivileges = append(grantPrivileges, planPrivilege.(string))
			}
		}

		for _, privilege := range role.Privileges {
			if planPrivileges.Contains(privilege.AccessType) == false {
				revokePrivileges = append(revokePrivileges, privilege.AccessType)
			}
		}
	}

	if roleNameHasChange {
		err := conn.Exec(fmt.Sprintf("ALTER ROLE %s RENAME TO %s", role.Name, planRoleName))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if roleDatabaseHasChange {
		err := conn.Exec(fmt.Sprintf("REVOKE ALL ON *.* FROM %s", planRoleName))
		if err != nil {
			return diag.FromErr(err)
		}
		dbPrivileges := role.GetPrivilegesList()
		err = conn.Exec(fmt.Sprintf(
			"GRANT %s ON %s.* TO %s",
			strings.Join(dbPrivileges, ","),
			planDatabase,
			planRoleName,
		))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if len(grantPrivileges) > 0 {
		err := conn.Exec(fmt.Sprintf("GRANT %s ON %s.* TO %s", strings.Join(grantPrivileges, ","), planDatabase, planRoleName))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if len(revokePrivileges) > 0 {
		err := conn.Exec(fmt.Sprintf("REVOKE %s ON %s.* FROM %s", strings.Join(revokePrivileges, ","), planDatabase, planRoleName))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}

func resourceRoleRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var diags diag.Diagnostics

	client := meta.(*common.ApiClient)
	conn := client.ClickhouseConnection

	roleNameState := d.Get("name").(string)

	errors := make([]error, 0)
	role := GetRole(conn, roleNameState, &errors)
	if len(errors) > 0 {
		return diag.FromErr(errors[0])
	}

	var database string
	var privileges []string
	for i := 0; i < len(role.Privileges); i++ {
		if database != "" && role.Privileges[i].Database != "" && role.Privileges[i].Database != database {
			errors = append(errors, fmt.Errorf("role %s has privileges on different databases", roleNameState))
		}
		database = role.Privileges[i].Database
		privileges = append(privileges, role.Privileges[i].AccessType)
	}

	if err := d.Set("name", role.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("database", database); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("privileges", &privileges); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(role.Name)

	return diags
}

func resourceRoleCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var diags diag.Diagnostics

	client := meta.(*common.ApiClient)
	conn := client.ClickhouseConnection

	database := d.Get("database").(string)
	roleName := d.Get("name").(string)
	privileges := d.Get("privileges").(*schema.Set)

	diags = ValidatePrivileges(privileges)

	if diags.HasError() {
		return diags
	}

	err := conn.Exec(fmt.Sprintf("CREATE ROLE %s", roleName))
	if err != nil {
		return diag.FromErr(err)
	}

	for _, privilege := range privileges.List() {
		err = conn.Exec(fmt.Sprintf("GRANT %s ON %s.* TO %s", privilege, database, roleName))
		if err != nil {
			// Rollback
			err2 := conn.Exec(fmt.Sprintf("DROP ROLE %s", roleName))
			if err2 != nil {
				return diag.FromErr(fmt.Errorf("%v, %v", err, err2))
			}
			return diag.FromErr(err)
		}
	}

	d.SetId(roleName)

	return diags

}

func resourceRoleDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var diags diag.Diagnostics

	client := meta.(*common.ApiClient)
	conn := client.ClickhouseConnection

	roleName := d.Get("name").(string)

	err := conn.Exec(fmt.Sprintf("DROP ROLE %s", roleName))

	if err != nil {
		return diag.FromErr(err)
	}
	return diags
}
