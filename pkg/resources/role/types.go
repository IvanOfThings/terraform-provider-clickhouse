package role

type CHGrant struct {
	RoleName   string
	AccessType string
	Database   string
}

type CHRole struct {
	Name       string
	Privileges []CHGrant
}

type Resource struct {
	Name       string
	Database   string
	Privileges []string
}

func (r *CHRole) GetPrivilegesList() []string {
	var privileges []string
	for _, privilege := range r.Privileges {
		privileges = append(privileges, privilege.AccessType)
	}
	return privileges
}
