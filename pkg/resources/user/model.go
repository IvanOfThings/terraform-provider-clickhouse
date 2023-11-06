package user

import (
	"github.com/IvanOfThings/terraform-provider-clickhouse/pkg/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type CHUser struct {
	Name  string
	Roles []string
}

type UserResource struct {
	Name     string
	Password string
	Roles    *schema.Set
}

func (u *CHUser) ToUserResource() *UserResource {
	return &UserResource{
		Name:  u.Name,
		Roles: common.StringListToSet(u.Roles),
	}
}
