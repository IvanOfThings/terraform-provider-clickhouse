package resourcetable

import (
	"context"
	"fmt"

	"github.com/Triple-Whale/terraform-provider-clickhouse/pkg/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceTable() *schema.Resource {
	return &schema.Resource{
		Description: "Resource to manage tables",

		CreateContext: resourceTableCreate,
		ReadContext:   resourceTableRead,
		DeleteContext: resourceTableDelete,
		UpdateContext: resourceTableUpdate,
		Schema: map[string]*schema.Schema{
			"database": {
				Description: "DB Name where the table will bellow",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"comment": {
				Description: "Database comment, it will be codified in a json along with come metadata information (like cluster name in case of clustering)",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"name": {
				Description: "Table Name",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"cluster": {
				Description: "Cluster Name, it is required for Replicated or Distributed tables and forbidden in other case",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
			},
			"engine": {
				Description: "Table engine type (Supported types so far: Distributed, ReplicatedReplacingMergeTree, ReplacingMergeTree)",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"engine_params": {
				Description: "Engine params in case the engine type requires them",
				Type:        schema.TypeList,
				Optional:    true,
				ForceNew:    true,
				Elem: &schema.Schema{
					Type:     schema.TypeString,
					ForceNew: true,
				},
			},
			"order_by": {
				Description: "Order by columns to use as sorting key",
				Type:        schema.TypeList,
				Optional:    true,
				ForceNew:    true,
				Elem: &schema.Schema{
					Type:     schema.TypeString,
					ForceNew: true,
				},
			},
			"partition_by": {
				Description: "Partition Key to split data",
				Type:        schema.TypeList,
				Optional:    true,
				ForceNew:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"by": {
							Description: "Column to use as part of the partition key",
							Type:        schema.TypeString,
							Required:    true,
							ForceNew:    true,
						},
						"partition_function": {
							Description: "Partition function, could be empty or one of following: toYYYYMM, toYYYYMMDD or toYYYYMMDDhhmmss",
							Type:        schema.TypeString,
							Optional:    true,
							Default:     nil,
							ForceNew:    true,
						},
						"mod": {
							Description: "Modulo to apply to the partition function",
							Type:        schema.TypeString,
							Optional:    true,
							ForceNew:    true,
						},
					},
				},
			},
			"column": {
				Description: "Column",
				Type:        schema.TypeList,
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Description: "Column Name",
							Type:        schema.TypeString,
							Required:    true,
						},
						"type": {
							Description: "Column Type",
							Type:        schema.TypeString,
							Required:    true,
						},
						"comment": {
							Description: "Column Comment",
							Type:        schema.TypeString,
							Optional:    true,
						},
						"default_kind": {
							Description: "Column Default Kind",
							Type:        schema.TypeString,
							Optional:    true,
						},
						"default_expression": {
							Description: "Column Default Expression",
							Type:        schema.TypeString,
							Optional:    true,
						},
					},
				},
			},
		"settings": {
			Description: "Table settings",
			Type:        schema.TypeMap,
			Optional:    true,
			ForceNew:    true,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		"ttl": {
			Description: "Table TTL expression, e.g. 'event_date + INTERVAL 30 DAY'",
			Type:        schema.TypeString,
			Optional:    true,
		},
			"index": {
				Description: "Index",
				Type:        schema.TypeList,
				Optional:    true,
				ForceNew:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Description: "Index Name",
							Type:        schema.TypeString,
							Required:    true,
							ForceNew:    true,
						},
						"expression": {
							Description: "Index Expression",
							Type:        schema.TypeString,
							Required:    true,
							ForceNew:    true,
						},
						"type": {
							Description: "Index Type",
							Type:        schema.TypeString,
							Required:    true,
							ForceNew:    true,
						},
						"granularity": {
							Description: "Index Granularity",
							Type:        schema.TypeInt,
							Optional:    true,
							ForceNew:    true,
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
	tableName := d.Get("name").(string)

	chTableService := CHTableService{CHConnection: conn}
	chTable, err := chTableService.GetTable(ctx, database, tableName)
	if chTable == nil && err == nil {
		d.SetId("")
		return nil
	}

	if err != nil {
		return diag.FromErr(fmt.Errorf("reading Clickhouse table: %v", err))
	}

	tableResource, err := chTable.ToResource()
	if err != nil {
		return diag.FromErr(fmt.Errorf("transforming Clickhouse table to resource: %v", err))
	}

	if err := d.Set("database", tableResource.Database); err != nil {
		return diag.FromErr(fmt.Errorf("setting database: %v", err))
	}
	if tableResource.Comment != "" {
		if err := d.Set("comment", tableResource.Comment); err != nil {
			return diag.FromErr(fmt.Errorf("setting comment: %v", err))
		}
	}
	if err := d.Set("name", tableResource.Name); err != nil {
		return diag.FromErr(fmt.Errorf("setting name: %v", err))
	}
	if tableResource.Cluster != "" {
		if err := d.Set("cluster", tableResource.Cluster); err != nil {
			return diag.FromErr(fmt.Errorf("setting cluster: %v", err))
		}
	}
	if err := d.Set("engine", tableResource.Engine); err != nil {
		return diag.FromErr(fmt.Errorf("setting engine: %v", err))
	}
	if tableResource.EngineParams != nil {
		if err := d.Set("engine_params", tableResource.EngineParams); err != nil {
			return diag.FromErr(fmt.Errorf("setting engine_params: %v", err))
		}
	}
	if tableResource.OrderBy != nil {
		if err := d.Set("order_by", tableResource.OrderBy); err != nil {
			return diag.FromErr(fmt.Errorf("setting order_by: %v", err))
		}
	}
	// not set - partition_by
	if err := d.Set("column", getColumns(tableResource.Columns)); err != nil {
		return diag.FromErr(fmt.Errorf("setting column: %v", err))
	}
	if tableResource.Indexes != nil {
		if err := d.Set("index", getIndexes(tableResource.Indexes)); err != nil {
			return diag.FromErr(fmt.Errorf("setting indexes: %v", err))
		}
	}
	// not set - settings

	d.SetId(tableResource.Cluster + ":" + database + ":" + tableName)

	return diags
}

func getIndexes(indexes []IndexDefinition) []map[string]interface{} {
	var ret []map[string]interface{}

	for _, index := range indexes {
		ret = append(ret, map[string]interface{}{
			"name":        index.Name,
			"expression":  index.Expression,
			"type":        index.Type,
			"granularity": index.Granularity,
		})
	}
	return ret
}

func getColumns(columns []ColumnDefinition) []map[string]interface{} {
	var ret []map[string]interface{}

	for _, column := range columns {
		ret = append(ret, map[string]interface{}{
			"name":               column.Name,
			"type":               column.Type,
			"comment":            column.Comment,
			"default_kind":       column.DefaultKind,
			"default_expression": column.DefaultExpression,
		})
	}
	return ret
}

func (t *TableResource) setColumns(columns []interface{}) {
	for _, column := range columns {
		columnDefinition := ColumnDefinition{
			Name:              column.(map[string]interface{})["name"].(string),
			Type:              column.(map[string]interface{})["type"].(string),
			Comment:           column.(map[string]interface{})["comment"].(string),
			DefaultKind:       column.(map[string]interface{})["default_kind"].(string),
			DefaultExpression: column.(map[string]interface{})["default_expression"].(string),
		}
		t.Columns = append(t.Columns, columnDefinition)
	}
}

func (t *TableResource) setIndexes(indexes []interface{}) {
	for _, index := range indexes {
		indexDefinition := IndexDefinition{
			Name:        index.(map[string]interface{})["name"].(string),
			Expression:  index.(map[string]interface{})["expression"].(string),
			Type:        index.(map[string]interface{})["type"].(string),
			Granularity: uint64(index.(map[string]interface{})["granularity"].(int)),
		}
		t.Indexes = append(t.Indexes, indexDefinition)
	}
}

func resourceTableCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var diags diag.Diagnostics

	client := meta.(*common.ApiClient)
	conn := client.ClickhouseConnection
	tableResource := TableResource{}
	chTableService := CHTableService{CHConnection: conn}

	tableResource.Cluster = d.Get("cluster").(string)
	tableResource.Database = d.Get("database").(string)
	tableResource.Name = d.Get("name").(string)
	tableResource.setColumns(d.Get("column").([]interface{}))
	tableResource.setIndexes(d.Get("index").([]interface{}))
	tableResource.Engine = d.Get("engine").(string)
	tableResource.Comment = common.GetComment(d.Get("comment").(string), tableResource.Cluster, nil)
	tableResource.EngineParams = common.MapArrayInterfaceToArrayOfStrings(d.Get("engine_params").([]interface{}))
	tableResource.OrderBy = common.MapArrayInterfaceToArrayOfStrings(d.Get("order_by").([]interface{}))
	tableResource.SetPartitionBy(d.Get("partition_by").([]interface{}))
	tableResource.Settings = common.MapInterfaceToMapOfString(d.Get("settings").(map[string]interface{}))
	tableResource.TTL = d.Get("ttl").(string)

	if tableResource.Cluster == "" {
		tableResource.Cluster = client.DefaultCluster
	}

	tableResource.Validate(diags)
	if diags.HasError() {
		return diags
	}

	err := chTableService.CreateTable(ctx, tableResource)

	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(tableResource.Cluster + ":" + tableResource.Database + ":" + tableResource.Name)

	return diags
}

func resourceTableDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var diags diag.Diagnostics
	client := meta.(*common.ApiClient)
	conn := client.ClickhouseConnection
	chTableService := CHTableService{CHConnection: conn}

	var tableResource TableResource
	tableResource.Database = d.Get("database").(string)
	tableResource.Name = d.Get("name").(string)
	tableResource.Cluster = d.Get("cluster").(string)
	if tableResource.Cluster == "" {
		tableResource.Cluster = client.DefaultCluster
	}

	err := chTableService.DeleteTable(ctx, tableResource)

	if err != nil {
		return diag.FromErr(err)
	}
	return diags
}

func resourceTableUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var diags diag.Diagnostics

	client := meta.(*common.ApiClient)
	conn := client.ClickhouseConnection
	chTableService := CHTableService{CHConnection: conn}

	tableResource := TableResource{}

	tableResource.Database = d.Get("database").(string)
	tableResource.Name = d.Get("name").(string)
	tableResource.Cluster = d.Get("cluster").(string)
	tableResource.setColumns(d.Get("column").([]interface{}))
	tableResource.Comment = common.GetComment(d.Get("comment").(string), tableResource.Cluster, nil)
	tableResource.TTL = d.Get("ttl").(string)

	err := chTableService.UpdateTable(ctx, tableResource, d)
	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}
