package clickhouse_provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
				Description: "Cluster name, not mandatory but should be provided if creating a db in a clustered server",
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

	storedComment, _ := result.String("comment")
	storedComment = strings.Replace(storedComment, "\\'", "'", -1)

	byteStreamComment := []byte(storedComment)

	var dat map[string]interface{}

	if err := json.Unmarshal(byteStreamComment, &dat); err != nil {
		return diag.FromErr(err)
	}
	comment := dat["comment"].(string)
	cluster := dat["cluster"].(string)

	newObject := fmt.Sprintf(
		`{
					"db_name":         "%v",
					"engine":       "%v",
					"data_path":     "%v",
					"metadata_path": "%v",
					"uuid":        "%v",
					"comment":    "%v",
					"cluster":    "%v"}`, name, engine, data_path, metadata_path, uuid, comment, cluster)
	input := []byte(newObject)
	var db map[string]string
	err = json.Unmarshal(input, &db)
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("db_name", name)
	d.Set("engine", engine)
	d.Set("data_path", data_path)
	d.Set("metadata_path", metadata_path)
	d.Set("uuid", uuid)
	d.Set("comment", comment)
	d.Set("cluster", cluster)

	d.SetId(cluster + ":" + database_name)

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

	clusterStatement := ""
	if cluster != "" {
		clusterStatement = "ON CLUSTER " + cluster
	}
	storingComment := fmt.Sprintf(`{"comment":"%v","cluster":"%v"}`, comment, cluster)
	storingComment = strings.Replace(storingComment, "'", "\\'", -1)

	query := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %v %v COMMENT '%v'", database_name, clusterStatement, storingComment)

	err := conn.Exec(query)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(cluster + ":" + database_name)

	return diags
}

func resourceDbDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {

	client := meta.(*apiClient)
	var diags diag.Diagnostics
	conn := client.clickhouseConnection

	database_name := d.Get("db_name").(string)
	cluster, _ := d.Get("cluster").(string)
	clusterStatement := ""
	if cluster != "" {
		clusterStatement = "ON CLUSTER " + cluster
	}
	err := conn.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %v %v SYNC", database_name, clusterStatement))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return diags
}
