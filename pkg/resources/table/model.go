package resourcetable

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Triple-Whale/terraform-provider-clickhouse/pkg/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

// -- begin DB read() types --
type CHTable struct {
	Database   string     `ch:"database"`
	Name       string     `ch:"name"`
	EngineFull string     `ch:"engine_full"`
	Engine     string     `ch:"engine"`
	Comment    string     `ch:"comment"`
	Columns    []CHColumn `ch:"columns"`
	Indexes    []CHIndex  `ch:"indexes"`
}

type CHIndex struct {
	Name        string `ch:"name"`
	Expression  string `ch:"expr"`
	Type        string `ch:"type"`
	Granularity uint64 `ch:"granularity"`
}

type CHColumn struct {
	Database          string `ch:"database"`
	Table             string `ch:"table"`
	Name              string `ch:"name"`
	Type              string `ch:"type"`
	Comment           string `ch:"comment"`
	DefaultKind       string `ch:"default_kind"`
	DefaultExpression string `ch:"default_expression"`
}

// -- end DB read() types --

// -- built from DB read, and from tf definition code --
type TableResource struct {
	Database     string
	Name         string
	EngineFull   string
	Engine       string
	Cluster      string
	Comment      string
	EngineParams []string
	OrderBy      []string
	Columns      []ColumnDefinition
	PartitionBy  []PartitionByResource
	Indexes      []IndexDefinition
	Settings     map[string]string
	TTL          string
}

type IndexDefinition struct {
	Name        string
	Expression  string
	Type        string
	Granularity uint64
}

type ColumnDefinition struct {
	Name              string `json:"name"`
	Type              string `json:"type"`
	Comment           string `json:"comment"`
	DefaultKind       string `json:"default_kind"`
	DefaultExpression string `json:"default_expression"`
}

type PartitionByResource struct {
	By                string
	PartitionFunction string
	Mod               string
}

// -- end parsed types --

func (t *CHTable) IndexesToResource() []IndexDefinition {
	var indexResources []IndexDefinition
	for _, index := range t.Indexes {
		indexResource := IndexDefinition{
			Name:        index.Name,
			Expression:  index.Expression,
			Type:        index.Type,
			Granularity: index.Granularity,
		}
		indexResources = append(indexResources, indexResource)
	}
	return indexResources
}

func (t *CHTable) ColumnsToResource() []ColumnDefinition {
	var columnResources []ColumnDefinition
	for _, column := range t.Columns {
		columnResource := ColumnDefinition{
			Name:              column.Name,
			Type:              column.Type,
			Comment:           column.Comment,
			DefaultKind:       column.DefaultKind,
			DefaultExpression: column.DefaultExpression,
		}
		columnResources = append(columnResources, columnResource)
	}
	return columnResources
}

func (t *CHTable) ToResource() (*TableResource, error) {
	tableResource := TableResource{
		Database:   t.Database,
		Name:       t.Name,
		EngineFull: t.EngineFull,
		Engine:     t.Engine,
		Columns:    t.ColumnsToResource(),
		Indexes:    t.IndexesToResource(),
	}

	engineParams := GetEngineParams(t.EngineFull)
	orderBy := GetOrderBy(t.EngineFull)

	comment, cluster, _, err := common.UnmarshalComment(t.Comment)
	if err != nil {
		return nil, err
	}

	tableResource.OrderBy = orderBy
	tableResource.Cluster = cluster
	tableResource.Comment = comment
	tableResource.EngineParams = removeDefaultParams(engineParams)

	return &tableResource, nil
}

func GetEngineParams(engineFull string) []string {
	r := regexp.MustCompile(`^\w+\(([^)]*)\)`)
	match := r.FindStringSubmatch(engineFull)
	var engineParams []string
	if len(match) > 1 {
		values := strings.Split(match[1], ",")
		for _, value := range values {
			value = strings.TrimSpace(value)
			engineParams = append(engineParams, strings.TrimSpace(value))
		}
	}
	return engineParams
}

func GetOrderBy(engineFull string) []string {
	rMultiple := regexp.MustCompile(`ORDER BY\s*\(([^)]+)\)`)

	match := rMultiple.FindStringSubmatch(engineFull)
	if len(match) == 0 {
		rSingle := regexp.MustCompile(`ORDER BY\s+([^ ]+)`)
		match = rSingle.FindStringSubmatch(engineFull)
	}

	var orderBy []string
	if len(match) > 1 {
		values := strings.Split(match[1], ",")
		for _, value := range values {
			value = strings.TrimSpace(value)
			orderBy = append(orderBy, value)
		}
	}
	return orderBy
}

// without this, terraform sees a diff for Replicated tables
func removeDefaultParams(engineParams []string) []string {
	var newEngineParams []string
	for _, param := range engineParams {
		if param != "'/clickhouse/tables/{uuid}/{shard}'" && param != "'{replica}'" {
			newEngineParams = append(newEngineParams, param)
		}
	}
	return newEngineParams
}

func (t *TableResource) GetColumnsResourceList() []ColumnDefinition {
	var columnResources []ColumnDefinition
	for _, column := range t.Columns {
		columnResources = append(columnResources, ColumnDefinition{
			Name:              column.Name,
			Type:              column.Type,
			Comment:           column.Comment,
			DefaultKind:       column.DefaultKind,
			DefaultExpression: column.DefaultExpression,
		})
	}
	return columnResources
}

func (t *TableResource) SetPartitionBy(partitionBy []interface{}) {
	for _, partitionBy := range partitionBy {
		partitionByResource := PartitionByResource{
			By:                partitionBy.(map[string]interface{})["by"].(string),
			PartitionFunction: partitionBy.(map[string]interface{})["partition_function"].(string),
			Mod:               partitionBy.(map[string]interface{})["mod"].(string),
		}
		t.PartitionBy = append(t.PartitionBy, partitionByResource)
	}
}

func (t *TableResource) HasColumn(columnName string) bool {
	for _, column := range t.GetColumnsResourceList() {
		if column.Name == columnName {
			return true
		}
	}
	return false
}

func (t *TableResource) Validate(diags diag.Diagnostics) {
	t.validateOrderBy(diags)
	t.validatePartitionBy(diags)
}

func (t *TableResource) validateOrderBy(diags diag.Diagnostics) {
	for _, orderField := range t.OrderBy {
		if t.HasColumn(orderField) == false {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "wrong value",
				Detail:   fmt.Sprintf("order by field '%s' is not a column", orderField),
			})
		}
	}
}

func (t *TableResource) validatePartitionBy(diags diag.Diagnostics) {
	for _, partitionBy := range t.PartitionBy {
		if t.HasColumn(partitionBy.By) == false {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "wrong value",
				Detail:   fmt.Sprintf("partition by field '%s' is not a column", partitionBy.By),
			})
		}
	}
}
