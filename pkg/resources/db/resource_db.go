package resourcedb

import (
	"context"
	"fmt"

	"github.com/IvanOfThings/terraform-provider-clickhouse/pkg/common"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceDb() *schema.Resource {
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
			"name": &schema.Schema{
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

	client := meta.(*common.ApiClient)
	var diags diag.Diagnostics
	conn := client.ClickhouseConnection
	defaultCluster := client.DefaultCluster

	database_name := d.Get("name").(string)
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

	if name == "" {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Database %v not found", database_name),
			Detail:   "Not possible to retrieve db from server. Could you be performing operation in a cluster? If so try configuring default cluster name on you provider configuration.",
		})
		return diags
	}

	storedComment, _ := result.String("comment")
	comment, cluster, err := common.UnmarshalComment(storedComment)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("Unable to unmarshal comments for db %q", name),
			Detail:   "Unable to unmarshal comments in order to retrieve cluster information for the table, so that default cluster is going to be used instead.",
		})
		comment, cluster = storedComment, defaultCluster
	}

	d.Set("name", name)
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

	client := meta.(*common.ApiClient)
	var diags diag.Diagnostics
	conn := client.ClickhouseConnection

	cluster, _ := d.Get("cluster").(string)

	clusterStatement, clusterToUse := common.GetClusterStatement(cluster, client.DefaultCluster)

	database_name := d.Get("name").(string)
	comment := d.Get("comment").(string)

	query := fmt.Sprintf("CREATE DATABASE %v %v COMMENT '%v'", database_name, clusterStatement, common.GetComment(comment, cluster))

	// diags = append(diags, diag.Diagnostic{
	// 	Severity: diag.Warning,
	// 	Summary:  fmt.Sprintf("Query"),
	// 	Detail:   fmt.Sprintf("Query %q", query),
	// })

	err := conn.Exec(query)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(clusterToUse + ":" + database_name)

	return diags
}

func resourceDbDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {

	client := meta.(*common.ApiClient)
	var diags diag.Diagnostics
	conn := client.ClickhouseConnection

	database_name := d.Get("name").(string)

	if database_name == "" {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Database name not found",
			Detail:   "Not possible to destroy resource as the database name was not retrieved succesfully. Could you be performing operation in a cluster? If so try configuring default cluster name on you provider configuration.",
		})
		return diags
	}

	dbResources, errors := common.GetResourceNamesOnDataBases(conn, database_name)

	// diags = append(diags, diag.Diagnostic{
	// 	Severity: diag.Warning,
	// 	Summary:  fmt.Sprintf("database_name"),
	// 	Detail:   fmt.Sprintf("database_name %q", database_name),
	// })

	if len(errors) > 0 {
		return diag.FromErr(errors[0])
	}
	if len(dbResources.TableNames) > 0 {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Unable to delete db resource %q", database_name),
			Detail:   fmt.Sprintf("DB resource is used by another resources and is not possible to delete it. Tables: %v.", dbResources.TableNames),
		})
		return diags
	}

	cluster, _ := d.Get("cluster").(string)
	clusterStatement, _ := common.GetClusterStatement(cluster, client.DefaultCluster)

	query := fmt.Sprintf("DROP DATABASE %v %v SYNC", database_name, clusterStatement)

	// diags = append(diags, diag.Diagnostic{
	// 	Severity: diag.Warning,
	// 	Summary:  fmt.Sprintf("Query"),
	// 	Detail:   fmt.Sprintf("Query %q", query),
	// })
	err := conn.Exec(query)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return diags
}
