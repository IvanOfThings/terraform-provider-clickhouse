package resourcedb

import (
	"context"
	"fmt"
	resourcetable "github.com/IvanOfThings/terraform-provider-clickhouse/pkg/resources/table"

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
	conn := *client.ClickhouseConnection
	defaultCluster := client.DefaultCluster

	database_name := d.Get("name").(string)
	row := conn.QueryRow(ctx, fmt.Sprintf("SELECT name, engine, data_path, metadata_path, uuid, comment FROM system.databases where name = '%v'", database_name))

	if row.Err() != nil {
		return diag.FromErr(fmt.Errorf("reading database from Clickhouse: %v", row.Err()))
	}

	var name, engine, dataPath, metadataPath, uuid, storedComment string

	err := row.Scan(&name, &engine, &dataPath, &metadataPath, &uuid, &storedComment)
	if err != nil {
		return diag.FromErr(fmt.Errorf("scanning Clickhouse DB row: %v", err))
	}

	if name == "" {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Database %v not found", database_name),
			Detail:   "Not possible to retrieve db from server. Could you be performing operation in a cluster? If so try configuring default cluster name on you provider configuration.",
		})
		return diags
	}

	comment, cluster, err := common.UnmarshalComment(storedComment)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("Unable to unmarshal comments for db %q", name),
			Detail:   "Unable to unmarshal comments in order to retrieve cluster information for the table, so that default cluster is going to be used instead.",
		})
		comment, cluster = storedComment, defaultCluster
	}

	err = d.Set("name", name)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Unable to set name for db %q", name),
		})
	}
	err = d.Set("engine", engine)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Unable to set engine for db %q", name),
		})
	}
	err = d.Set("data_path", dataPath)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Unable to set data_path for db %q", name),
		})
	}
	err = d.Set("metadata_path", metadataPath)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Unable to set metadata_path for db %q", name),
		})
	}
	err = d.Set("uuid", uuid)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Unable to set uuid for db %q", name),
		})
	}
	err = d.Set("comment", comment)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Unable to set comment for db %q", name),
		})
	}
	err = d.Set("cluster", cluster)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Unable to set cluster for db %q", name),
		})
	}

	d.SetId(cluster + ":" + database_name)

	tflog.Trace(ctx, "DB resource created.")

	return diags
}

func resourceDbCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*common.ApiClient)
	var diags diag.Diagnostics
	conn := *client.ClickhouseConnection

	cluster, _ := d.Get("cluster").(string)
	if cluster == "" {
		cluster = client.DefaultCluster
	}
	clusterStatement := common.GetClusterStatement(cluster)
	databaseName := d.Get("name").(string)
	comment := d.Get("comment").(string)

	query := fmt.Sprintf("CREATE DATABASE %v %v COMMENT '%v'", databaseName, clusterStatement, common.GetComment(comment, cluster))

	err := conn.Exec(ctx, query)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(cluster + ":" + databaseName)

	return diags
}

func resourceDbDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {

	client := meta.(*common.ApiClient)
	var diags diag.Diagnostics
	conn := client.ClickhouseConnection

	databaseName := d.Get("name").(string)

	if databaseName == "" {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Database name not found",
			Detail:   "Not possible to destroy resource as the database name was not retrieved succesfully. Could you be performing operation in a cluster? If so try configuring default cluster name on you provider configuration.",
		})
		return diags
	}

	chTableService := resourcetable.CHTableService{CHConnection: conn}
	chDBService := CHDBService{CHConnection: conn, CHTableService: &chTableService}
	dbResources, err := chDBService.GetDBResources(ctx, databaseName)

	if err != nil {
		return diag.FromErr(fmt.Errorf("resource db delete: %v", err))
	}

	cluster, _ := d.Get("cluster").(string)
	if cluster == "" {
		cluster = client.DefaultCluster
	}
	clusterStatement := common.GetClusterStatement(cluster)

	query := fmt.Sprintf("DROP DATABASE %v %v SYNC", databaseName, clusterStatement)

	err = (*conn).Exec(ctx, query)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return diags
}
