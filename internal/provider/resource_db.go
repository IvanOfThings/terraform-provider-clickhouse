package clickhouse_provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceDb() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "Resource to handle clickhouse databases.",

		CreateContext: resourceDbCreate,
		ReadContext:   resourceDbRead,
		DeleteContext: resourceDbDelete,

		Schema: map[string]*schema.Schema{
			"cluster": &schema.Schema{
				Description: "Cluster name",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
			},
			"db_name": &schema.Schema{
				Description: "Database name",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"engine": &schema.Schema{
				Description: "Database engine",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"data_path": &schema.Schema{
				Description: "Database internal path",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"metadata_path": &schema.Schema{
				Description: "Database internal metadata path",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"uuid": &schema.Schema{
				Description: "Database UUID",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"comment": &schema.Schema{
				Description: "Comment about the database",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				ForceNew:    true,
			},
		},
	}
}

func resourceDbRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	// use the meta value to retrieve your client from the provider configure method
	// client := meta.(*apiClient)
	client := meta.(*apiClient)
	var diags diag.Diagnostics
	conn := client.clickhouseConnection

	database_name := d.Get("db_name").(string)
	iter, err := conn.Fetch(fmt.Sprintf("SELECT name, engine, data_path, metadata_path, uuid, comment FROM system.databases where name = '%v'", database_name))
	if err != nil {
		return diag.FromErr(err)
	}

	iter.Next()
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
	err = json.Unmarshal(input, &db)
	if err != nil {
		return diag.FromErr(err)
	}
	println(name, data_path, engine, comment)

	d.Set("db_name", name)
	d.Set("engine", engine)
	d.Set("data_path", data_path)
	d.Set("metadata_path", metadata_path)
	d.Set("uuid", uuid)
	d.Set("comment", comment)

	d.SetId(database_name)

	tflog.Trace(ctx, "DB resource created.")

	return diags
}

func resourceDbCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {

	client := meta.(*apiClient)
	var diags diag.Diagnostics
	conn := client.clickhouseConnection

	database_name := d.Get("db_name").(string)
	comment := d.Get("comment").(string)
	cluster, _ := d.Get("cluster").(string)
	if cluster != "" {
		err := conn.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %v ON CLUSTER %v COMMENT '%v'", database_name, cluster, comment))

		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		err := conn.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %v COMMENT '%v'", database_name, comment))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(database_name)

	return diags
}

func resourceDbDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {

	client := meta.(*apiClient)
	var diags diag.Diagnostics
	conn := client.clickhouseConnection

	database_name := d.Get("db_name").(string)
	cluster, _ := d.Get("cluster").(string)
	if cluster != "" {
		err := conn.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %v ON CLUSTER %v SYNC", database_name, cluster))
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		err := conn.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %v SYNC", database_name))
		if err != nil {
			return diag.FromErr(err)
		}
	}
	d.SetId("")
	return diags
}
