package resourcerole

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"strings"
)

var AllowedPrivileges = []string{
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
	"ALTER TABLE",
	"ALTER VIEW",
}

func ValidatePrivileges(privileges *schema.Set) diag.Diagnostics {
	var diagnostics diag.Diagnostics

	for _, privilege := range privileges.List() {
		validatePrivilege(privilege.(string), &diagnostics)
	}
	return diagnostics
}

func validatePrivilege(privilege string, diagnostics *diag.Diagnostics) {
	isAllowed := false
	upperCasePrivilege := strings.ToUpper(privilege)
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
