package clickhouse_provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceTable() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceTableCreate,
		ReadContext:   resourceTableRead,
		DeleteContext: resourceTableDelete,
		Schema: map[string]*schema.Schema{
			"database": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"comment": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"table_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"cluster": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"engine": &schema.Schema{
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validateOnClusterEngine,
			},
			"engine_params": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				Elem: &schema.Schema{
					Type:     schema.TypeString,
					ForceNew: true,
				},
			},
			"order_by": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Schema{
					Type:     schema.TypeString,
					ForceNew: true,
				},
			},
			"partition_by": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"by": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"partition_function": &schema.Schema{
							Type:             schema.TypeString,
							Optional:         true,
							ValidateDiagFunc: validatePartitionBy,
							Default:          nil,
							ForceNew:         true,
						},
					},
				},
			},
			"columns": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"type": &schema.Schema{
							Type:             schema.TypeString,
							Required:         true,
							ValidateDiagFunc: validateType,
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

	client := meta.(*apiClient)
	conn := client.clickhouseConnection

	database := d.Get("database").(string)
	table_name := d.Get("table_name").(string)
	columns := d.Get("columns").([]interface{})
	order_by := mapArrayInterfaceToArrayOfStrings(d.Get("order_by").([]interface{}))
	mappedColumns := mapColums(columns)
	err := validateParams(mappedColumns, order_by, "order_by")
	if err != nil {
		return diag.FromErr(err)
	}

	if err != nil {
		return diag.FromErr(err)
	}

	data := CHDataBase{
		database:   database,
		table_name: table_name,
	}
	errors := make([]error, 0)
	tables := getTables(conn, &data, &errors)
	if len(errors) > 0 {
		return diag.FromErr(errors[0])
	}
	mappedTables := make([]dataSourceCHTable, 0)
	if len(mappedTables) == 0 {
		return diags
	}

	table, err := mapTableToDatasource(tables[0])
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("database", &table.database); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("table_name", &table.table_name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("engine", &table.engine); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("engine_params", &table.engine_params); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("cluster", &table.cluster); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("columns", &table.columns); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(*table.cluster + ":" + database + ":" + table_name)

	return diags

}

func resourceTableCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {

	var diags diag.Diagnostics

	database := d.Get("database").(string)
	table_name := d.Get("table_name").(string)
	columns := d.Get("columns").([]interface{})
	cluster := d.Get("cluster").(string)
	engine := d.Get("engine").(string)
	comment := d.Get("comment").(string)
	engine_params := mapArrayInterfaceToArrayOfStrings(d.Get("engine_params").([]interface{}))
	order_by := mapArrayInterfaceToArrayOfStrings(d.Get("order_by").([]interface{}))
	mappedColumns := mapColums(columns)
	err := validateParams(mappedColumns, order_by, "order_by")
	if err != nil {
		return diag.FromErr(err)
	}

	partition_by := d.Get("partition_by")

	mappedPartitionBy, err := mapPartitionBy(partition_by, mappedColumns)
	if err != nil {
		return diag.FromErr(err)
	}

	query := buildCreateONClusterSentence(mappedColumns, database, table_name, cluster, engine, order_by, engine_params, mappedPartitionBy, getComment(comment, cluster))

	client := meta.(*apiClient)
	conn := client.clickhouseConnection

	err = conn.Exec(query)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(cluster + ":" + database + ":" + table_name)

	return diags

}

func resourceTableDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var diags diag.Diagnostics

	client := meta.(*apiClient)
	conn := client.clickhouseConnection

	database := d.Get("database").(string)
	table_name := d.Get("table_name").(string)
	cluster := d.Get("cluster").(string)

	err := conn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %v.%v on cluster %v SYNC", database, table_name, cluster))

	if err != nil {
		return diag.FromErr(err)
	}
	return diags
}
