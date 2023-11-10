package datasources

import (
	"context"
	"log"

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
			"dbs": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Description: "DB Name",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"engine": {
							Description: "DB Engine",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"data_path": {
							Description: "DB Path",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"metadata_path": {
							Description: "Metadata Path",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"uuid": {
							Description: "Metadata Path",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"comment": {
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
	conn := *client.ClickhouseConnection

	rows, err := conn.Query(ctx, "SELECT name, engine, data_path, metadata_path, uuid, comment FROM system.databases")
	if err != nil {
		return diag.FromErr(err)
	}

	var dbResources []map[string]interface{}

	for i := 0; rows.Next(); i++ {
		var chDatabase CHDatabase
		if err := rows.ScanStruct(&chDatabase); err != nil {
			log.Fatal(err)
		}
		dbResource := map[string]interface{}{
			"name":          chDatabase.Name,
			"engine":        chDatabase.Engine,
			"data_path":     chDatabase.DataPath,
			"metadata_path": chDatabase.MetadataPath,
			"uuid":          chDatabase.Uuid,
			"comment":       chDatabase.Comment,
		}
		dbResources = append(dbResources, dbResource)
	}
	if err := d.Set("dbs", dbResources); err != nil {
		return diag.FromErr(err)
	}
	d.SetId("databases_read")
	return diags
}
