package datasources

import (
	"context"

	"github.com/IvanOfThings/terraform-provider-clickhouse/pkg/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceDbs() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "Datasource to retrieve all databases set in clickhouse instance",

		ReadContext: dataSourceDbsRead,

		Schema: map[string]*schema.Schema{
			"dbs": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"db_name": &schema.Schema{
							Description: "DB Name",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"engine": &schema.Schema{
							Description: "DB Engine",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"data_path": &schema.Schema{
							Description: "DB Path",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"metadata_path": &schema.Schema{
							Description: "Metadata Path",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"uuid": &schema.Schema{
							Description: "Metadata Path",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"comment": &schema.Schema{
							Description: "Database comment, it will be codified in a json along with come metadata information (like cluster name in case of clustering)",
							Type:        schema.TypeString,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func dataSourceDbsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*common.ApiClient)
	var diags diag.Diagnostics
	conn := client.ClickhouseConnection

	iter, err := conn.Fetch("SELECT name, engine, data_path, metadata_path, uuid, comment FROM system.databases")
	if err != nil {
		return diag.FromErr(err)
	}

	databases := make([]map[string]string, 0)

	for i := 0; iter.Next(); i++ {
		result := iter.Result

		db, err := common.MapDbData(result)
		if err != nil {
			return diag.FromErr(err)
		}
		databases = append(databases, db)

	}
	if err := d.Set("dbs", &databases); err != nil {
		return diag.FromErr(err)
	}
	d.SetId("databases_read")
	return diags
}
