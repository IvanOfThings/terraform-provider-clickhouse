package resourcerole

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"strings"
)

var AllowedDbLevelPrivileges = []string{
	"SELECT",
	"INSERT",
	"ALTER",
	"CREATE DATABASE",
	"CREATE TABLE",
	"CREATE VIEW",
	"CREATE DICTIONARY",
	"DROP DATABASE",
	"DROP TABLE",
	"SHOW TABLES",
}

var AllowedGlobalPrivileges = []string{
	"REMOTE",
}

var AllowedPrivileges = append(AllowedDbLevelPrivileges, AllowedGlobalPrivileges...)

func IsGlobalPrivilege(privilege string) bool {
	for _, globalPrivilege := range AllowedGlobalPrivileges {
		if privilege == globalPrivilege {
			return true
		}
	}
	return false
}

func ValidatePrivileges(database string, privileges *schema.Set) diag.Diagnostics {
	var diagnostics diag.Diagnostics

	for _, privilege := range privileges.List() {
		validatePrivilege(database, privilege.(string), &diagnostics)
	}
	return diagnostics
}

func validatePrivilege(database string, privilege string, diagnostics *diag.Diagnostics) {
	isAllowed := false
	upperCasePrivilege := strings.ToUpper(privilege)

	if IsGlobalPrivilege(privilege) && database != "*" {
		diagnostic := diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "wrong value",
			Detail: fmt.Sprintf(
				"Global privilege %s is only allowed for database '*'",
				privilege),
		}
		*diagnostics = append(*diagnostics, diagnostic)
		return
	}
	for _, allowedPrivilege := range AllowedPrivileges {
		if upperCasePrivilege == allowedPrivilege {
			isAllowed = true
			break
		}
	}
	if isAllowed == false {
		diagnostic := diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "wrong value",
			Detail: fmt.Sprintf(
				"%s isn't in the allowed privileges list: [%s]",
				privilege,
				strings.Join(AllowedPrivileges, ", ")),
		}
		*diagnostics = append(*diagnostics, diagnostic)
	}
}
