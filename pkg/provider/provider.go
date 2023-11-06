package provider

import (
	"context"
	"errors"
	"fmt"
	"github.com/IvanOfThings/terraform-provider-clickhouse/pkg/resources/db"
	"github.com/IvanOfThings/terraform-provider-clickhouse/pkg/resources/role"
	"github.com/IvanOfThings/terraform-provider-clickhouse/pkg/resources/table"
	"github.com/IvanOfThings/terraform-provider-clickhouse/pkg/resources/user"
	"net/url"
	"os"

	"github.com/IvanOfThings/terraform-provider-clickhouse/pkg/common"
	"github.com/IvanOfThings/terraform-provider-clickhouse/pkg/datasources"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/joho/godotenv"
	ch "github.com/leprosus/golang-clickhouse"
)

func init() {
	// Set descriptions to support markdown syntax, this will be used in document generation
	// and the language server.
	schema.DescriptionKind = schema.StringMarkdown

	// Customize the content of descriptions when output. For example you can add defaults on
	// to the exported descriptions if present.
	// schema.SchemaDescriptionBuilder = func(s *schema.Schema) string {
	// 	desc := s.Description
	// 	if s.Default != nil {
	// 		desc += fmt.Sprintf(" Defaults to `%v`.", s.Default)
	// 	}
	// 	return strings.TrimSpace(desc)
	// }
}

func New(version string) func() *schema.Provider {
	return func() *schema.Provider {
		p := &schema.Provider{
			Schema: map[string]*schema.Schema{
				"default_cluster": &schema.Schema{
					Description: "Default cluster, if provided will be used when no cluster is provided",
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
				},
				"username": &schema.Schema{
					Description: "Clickhouse username with admin privileges",
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: func() (any, error) {
						return getEnvVar("TF_CLICKHOUSE_USERNAME")
					},
				},
				"password": &schema.Schema{
					Description: "Clickhouse user password with admin privileges",
					Type:        schema.TypeString,
					Optional:    true,
					Sensitive:   true,
					DefaultFunc: func() (any, error) {
						if password, _ := getEnvVar("TF_CLICKHOUSE_PASSWORD"); password != nil {
							return password, nil
						}
						return "", nil
					},
				},
				"host": &schema.Schema{
					Description: "Clickhouse server url",
					Type:        schema.TypeString,
					Required:    true,
					Sensitive:   true,
					DefaultFunc: func() (any, error) {
						return getEnvVar("TF_CLICKHOUSE_HOST")
					},
				},
				"port": &schema.Schema{
					Description: "Clickhouse server port",
					Type:        schema.TypeInt,
					Required:    true,
					DefaultFunc: func() (any, error) {
						return getEnvVar("TF_CLICKHOUSE_PORT")
					},
				},
			},
			DataSourcesMap: map[string]*schema.Resource{
				"clickhouse_dbs": datasources.DataSourceDbs(),
			},
			ResourcesMap: map[string]*schema.Resource{
				"clickhouse_db":    resourcedb.ResourceDb(),
				"clickhouse_table": resourcetable.ResourceTable(),
				"clickhouse_role":  resourcerole.ResourceRole(),
				"clickhouse_user":  resourceuser.ResourceUser(),
			},
			ConfigureContextFunc: configure(),
		}

		return p
	}
}

func getEnvVar(envVarName string) (any, error) {

	godotenv.Load(".env")
	if v := os.Getenv(envVarName); v != "" {
		return v, nil
	}
	return nil, errors.New(fmt.Sprintf("Env var %v not present", envVarName))

}

func configure() func(context.Context, *schema.ResourceData) (any, diag.Diagnostics) {
	return func(ctx context.Context, d *schema.ResourceData) (any, diag.Diagnostics) {

		clickhouseUrl := d.Get("host").(string)
		port := d.Get("port").(int)
		username := d.Get("username").(string)
		defaultCluster := d.Get("default_cluster").(string)
		password := d.Get("password").(string)
		clickhouseConnection := ch.New(clickhouseUrl, port, username, url.QueryEscape(password))

		var diags diag.Diagnostics

		if clickhouseUrl == "" {
			return nil, diag.FromErr(fmt.Errorf("Error retrieving clickhouse uri"))
		}

		return &common.ApiClient{ClickhouseConnection: clickhouseConnection, DefaultCluster: defaultCluster}, diags
	}
}
