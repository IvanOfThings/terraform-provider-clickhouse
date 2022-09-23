package clickhouse_provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDbs() *schema.Resource {
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
							Type:     schema.TypeString,
							Computed: true,
						},
						"engine": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"data_path": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"metadata_path": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"uuid": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"comment": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceDbsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*apiClient)
	var diags diag.Diagnostics
	conn := client.clickhouseConnection

	iter, err := conn.Fetch("SELECT name, engine, data_path, metadata_path, uuid, comment FROM system.databases")
	if err != nil {
		return diag.FromErr(err)
	}

	databases := make([]map[string]string, 0)
	id := ""

	for i := 0; iter.Next(); i++ {
		result := iter.Result

		name, _ := result.String("name")
		engine, _ := result.String("engine")
		data_path, _ := result.String("data_path")
		metadata_path, _ := result.String("metadata_path")
		uuid, _ := result.String("uuid")
		comment, _ := result.String("comment")
		newObject := fmt.Sprintf(
			`{
				"db_name":         "%v",
				"engine":       "%v",
				"data_path":     "%v",
				"metadata_path": "%v",
				"uuid":        "%v",
				"comment":    "%v"}`, name, engine, data_path, metadata_path, uuid, comment)

		input := []byte(newObject)
		var db map[string]string
		err := json.Unmarshal(input, &db)

		if err != nil {
			return diag.FromErr(err)
		}
		databases = append(databases, db)
		id = id + ":" + name

	}
	if err := d.Set("dbs", databases); err != nil {
		return diag.FromErr(err)
	}
	d.SetId("databases_read")
	return diags
}
