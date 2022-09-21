package provider

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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
				"username": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
					DefaultFunc: func() (any, error) {
						return getEnvVar("TF_CLICKHOUSE_USERNAME")
					},
				},
				"password": &schema.Schema{
					Type:      schema.TypeString,
					Optional:  true,
					Sensitive: true,
					DefaultFunc: func() (any, error) {
						return getEnvVar("TF_CLICKHOUSE_PASSWORD")
					},
				},
				"clickhouse_url": &schema.Schema{
					Type:      schema.TypeString,
					Required:  true,
					Sensitive: true,
					DefaultFunc: func() (any, error) {
						return getEnvVar("TF_CLICKHOUSE_URL")
					},
				},
				"port": &schema.Schema{
					Type:     schema.TypeInt,
					Required: true,
				},
			},
			DataSourcesMap: map[string]*schema.Resource{
				"dbs_data_source": dataSourceDbs(),
			},
			ResourcesMap: map[string]*schema.Resource{
				"db_resource": resourceDb(),
			},
		}

		p.ConfigureContextFunc = configure(version, p)

		return p
	}
}

func getEnvVar(envVarName string) (any, error) {
	if v := os.Getenv(envVarName); v != "" {
		return v, nil
	}
	return nil, errors.New(fmt.Sprintf("Env var %v not present", envVarName))

}

type apiClient struct {
	clickhouseConnection *ch.Conn
}

func configure(version string, p *schema.Provider) func(context.Context, *schema.ResourceData) (any, diag.Diagnostics) {
	return func(ctx context.Context, d *schema.ResourceData) (any, diag.Diagnostics) {
		clickhouseUrl := d.Get("clickhouse_url").(string)
		port := d.Get("port").(int)
		username := d.Get("username").(string)
		password := d.Get("password").(string)
		clickhouseConnection := ch.New(clickhouseUrl, port, username, url.QueryEscape(password))

		var diags diag.Diagnostics

		if clickhouseUrl == "" {
			return nil, diag.FromErr(fmt.Errorf("Error retrieving clickhouse uri"))
		}

		return &apiClient{clickhouseConnection: clickhouseConnection}, diags
	}
}
