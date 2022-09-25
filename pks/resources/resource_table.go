package resources

import (
	"context"
	"fmt"

	"github.com/IvanOfThings/terraform-provider-clickhouse/pks/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceTable() *schema.Resource {
	return &schema.Resource{
		Description: "Resource to manage tables",

		CreateContext: resourceTableCreate,
		ReadContext:   resourceTableRead,
		DeleteContext: resourceTableDelete,
		Schema: map[string]*schema.Schema{

			"database": &schema.Schema{
				Description: "DB Name where the table will bellow",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"comment": &schema.Schema{
				Description: "Database comment, it will be codified in a json along with come metadata information (like cluster name in case of clustering)",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
			},
			"table_name": &schema.Schema{
				Description: "Table Name",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"cluster": &schema.Schema{
				Description: "Cluster Name, it is required for Replicated or Distributed tables and forbidden in other case",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
			},
			"engine": &schema.Schema{
				Description:      "Table engine type (Supported types so far: Distributed, ReplicatedReplacingMergeTree, ReplacingMergeTree)",
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: common.ValidateOnClusterEngine,
			},
			"engine_params": &schema.Schema{
				Description: "Engine params in case the engine type requires them",
				Type:        schema.TypeList,
				Required:    true,
				ForceNew:    true,
				Elem: &schema.Schema{
					Type:     schema.TypeString,
					ForceNew: true,
				},
			},
			"order_by": &schema.Schema{
				Description: "Order by columns to use as sorting key",
				Type:        schema.TypeList,
				Optional:    true,
				ForceNew:    true,
				Elem: &schema.Schema{
					Type:     schema.TypeString,
					ForceNew: true,
				},
			},
			"partition_by": &schema.Schema{
				Description: "Partition Key to split data",
				Type:        schema.TypeList,
				Optional:    true,
				ForceNew:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"by": &schema.Schema{
							Description: "Column to use as part of the partition key",
							Type:        schema.TypeString,
							Required:    true,
							ForceNew:    true,
						},
						"partition_function": &schema.Schema{
							Description:      "Partition function, could be empty or one of following: toYYYYMM, toYYYYMMDD or toYYYYMMDDhhmmss",
							Type:             schema.TypeString,
							Optional:         true,
							ValidateDiagFunc: common.ValidatePartitionBy,
							Default:          nil,
							ForceNew:         true,
						},
					},
				},
			},
			"column": &schema.Schema{
				Description: "Column",
				Type:        schema.TypeList,
				Optional:    true,
				ForceNew:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Description: "Column Name",
							Type:        schema.TypeString,
							Required:    true,
							ForceNew:    true,
						},
						"type": &schema.Schema{
							Description:      "Column Type",
							Type:             schema.TypeString,
							Required:         true,
							ValidateDiagFunc: common.ValidateType,
							ForceNew:         true,
						},
					},
				},
			},
		},
	}
}

func resourceTableRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var diags diag.Diagnostics

	client := meta.(*common.ApiClient)
	conn := client.ClickhouseConnection

	database := d.Get("database").(string)
	table_name := d.Get("table_name").(string)
	columns := d.Get("column").([]interface{})
	order_by := common.MapArrayInterfaceToArrayOfStrings(d.Get("order_by").([]interface{}))
	mappedColumns := common.MapColumns(columns)
	err := common.ValidateParams(mappedColumns, order_by, "order_by")
	if err != nil {
		return diag.FromErr(err)
	}

	if err != nil {
		return diag.FromErr(err)
	}

	data := common.CHDataBase{
		Database:   database,
		Table_name: table_name,
	}

	errors := make([]error, 0)
	tables := common.GetTables(conn, &data, &errors)
	if len(errors) > 0 {
		return diag.FromErr(errors[0])
	}
	mappedTables := make([]common.DataSourceCHTable, 0)
	if len(mappedTables) == 0 {
		return diags
	}

	table, err := common.MapTableToDatasource(tables[0])
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("database", &table.Database); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("table_name", &table.Table_name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("engine", &table.Engine); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("engine_params", &table.Engine_params); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("cluster", table.Cluster); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("column", &table.Columns); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(*table.Cluster + ":" + database + ":" + table_name)

	return diags

}

func resourceTableCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {

	var diags diag.Diagnostics

	client := meta.(*common.ApiClient)
	conn := client.ClickhouseConnection

	cluster := d.Get("cluster").(string)
	database := d.Get("database").(string)
	table_name := d.Get("table_name").(string)
	columns := d.Get("column").([]interface{})
	engine := d.Get("engine").(string)
	comment := d.Get("comment").(string)
	engine_params := common.MapArrayInterfaceToArrayOfStrings(d.Get("engine_params").([]interface{}))
	order_by := common.MapArrayInterfaceToArrayOfStrings(d.Get("order_by").([]interface{}))
	mappedColumns := common.MapColumns(columns)
	err := common.ValidateParams(mappedColumns, order_by, "order_by")
	if err != nil {
		return diag.FromErr(err)
	}

	partition_by := d.Get("partition_by")

	mappedPartitionBy, err := common.MapPartitionBy(partition_by, mappedColumns)
	if err != nil {
		return diag.FromErr(err)
	}

	query, clusterToUse := common.BuildCreateONClusterSentence(mappedColumns, database, table_name, cluster, client.DefaultCluster, engine, order_by, engine_params, mappedPartitionBy, common.GetComment(comment, cluster))

	err = conn.Exec(query)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(clusterToUse + ":" + database + ":" + table_name)

	return diags

}

func resourceTableDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var diags diag.Diagnostics

	client := meta.(*common.ApiClient)
	conn := client.ClickhouseConnection

	database := d.Get("database").(string)
	table_name := d.Get("table_name").(string)
	cluster := d.Get("cluster").(string)
	clusterStatement, _ := common.GetClusterStatement(cluster, client.DefaultCluster)

	err := conn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %v.%v "+clusterStatement, database, table_name))

	if err != nil {
		return diag.FromErr(err)
	}
	return diags
}
