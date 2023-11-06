package user

import (
	"context"
	"fmt"
	"github.com/IvanOfThings/terraform-provider-clickhouse/pkg/common"
	"github.com/IvanOfThings/terraform-provider-clickhouse/pkg/model"
	"github.com/IvanOfThings/terraform-provider-clickhouse/pkg/services"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceUser() *schema.Resource {
	return &schema.Resource{
		Description:   "Resource to manage Clickhouse users",
		CreateContext: resourceUserCreate,
		UpdateContext: resourceUserUpdate,
		ReadContext:   resourceUserRead,
		DeleteContext: resourceUserDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Description: "User name",
				Type:        schema.TypeString,
				Required:    true,
			},
			"password": {
				Description: "User password",
				Type:        schema.TypeString,
				Required:    true,
			},
			"roles": {
				Description: "User role",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			//"profile": &schema.Schema{
			//	Description: "Order by columns to use as sorting key",
			//	Type:        schema.TypeString,
			//	Optional:    true,
			//},
		},
	}
}

func resourceUserRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var diags diag.Diagnostics

	client := meta.(*common.ApiClient)
	conn := client.ClickhouseConnection
	chUserService := services.CHUserService{CHConnection: conn}

	userName := d.Get("name").(string)
	user, err := chUserService.GetUser(userName)
	if err != nil {
		return diag.FromErr(fmt.Errorf("resource user read: %v", err))
	}

	if err := d.Set("name", user.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("roles", &user.Roles); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(user.Name)

	return diags
}

func resourceUserCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var diags diag.Diagnostics

	client := meta.(*common.ApiClient)
	conn := client.ClickhouseConnection

	userName := d.Get("name").(string)
	password := d.Get("password").(string)
	rolesSet := d.Get("roles").(*schema.Set)
	chUserService := services.CHUserService{CHConnection: conn}
	chUser, err := chUserService.CreateUser(model.UserResource{
		Name:     userName,
		Password: password,
		Roles:    rolesSet,
	})
	if err != nil {
		return diag.FromErr(fmt.Errorf("resource user create: %v", err))
	}

	d.SetId(chUser.Name)

	return diags
}

func resourceUserUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := meta.(*common.ApiClient)
	conn := client.ClickhouseConnection
	chUserService := services.CHUserService{CHConnection: conn}

	planUserName := d.Get("name").(string)
	planPassword := d.Get("password").(string)
	planRoles := d.Get("roles").(*schema.Set)

	// After modify original role grants, we need to update default roles
	chUser, err := chUserService.UpdateUser(model.UserResource{
		Name:     planUserName,
		Password: planPassword,
		Roles:    planRoles,
	}, d)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(chUser.Name)

	return diags
}

func resourceUserDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var diags diag.Diagnostics

	client := meta.(*common.ApiClient)
	conn := client.ClickhouseConnection

	userName := d.Get("name").(string)

	err := conn.Exec(fmt.Sprintf("DROP USER %s", userName))

	if err != nil {
		return diag.FromErr(err)
	}
	return diags
}
