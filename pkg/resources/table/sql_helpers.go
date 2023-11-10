package resourcetable

import (
	"fmt"
	"github.com/IvanOfThings/terraform-provider-clickhouse/pkg/common"
	"strings"
)

func buildColumnsSentence(cols []ColumnResource) []string {
	outColumn := make([]string, 0)
	for _, col := range cols {
		outColumn = append(outColumn, fmt.Sprintf("\t %s %s", col.Name, col.Type))
	}
	return outColumn
}

func buildPartitionBySentence(partitionBy []PartitionByResource) string {
	if len(partitionBy) > 0 {
		partitionBySentenceItems := make([]string, 0)
		for _, partitionByItem := range partitionBy {
			if partitionByItem.PartitionFunction == "" {
				partitionBySentenceItems = append(partitionBySentenceItems, partitionByItem.By)
			} else {
				partitionBySentenceItems = append(partitionBySentenceItems, fmt.Sprintf("%v(%v)", partitionByItem.PartitionFunction, partitionByItem.By))
			}
		}
		return fmt.Sprintf("PARTITION BY %v", strings.Join(partitionBySentenceItems, ", "))
	}
	return ""
}

func buildOrderBySentence(orderBy []string) string {
	if len(orderBy) > 0 {
		return fmt.Sprintf("ORDER BY %v", strings.Join(orderBy, ", "))
	}
	return ""
}

func buildCreateOnClusterSentence(resource TableResource) (query string) {
	columnsStatement := ""
	if len(resource.Columns) > 0 {
		columnsList := buildColumnsSentence(resource.GetColumnsResourceList())
		columnsStatement = "(" + strings.Join(columnsList, ",\n") + ")\n"
	}

	clusterStatement := common.GetClusterStatement(resource.Cluster)

	return fmt.Sprintf(
		"CREATE TABLE %v.%v %v %v ENGINE = %v(%v) %s %s COMMENT '%s'",
		resource.Database,
		resource.Name,
		clusterStatement,
		columnsStatement,
		resource.Engine,
		strings.Join(resource.EngineParams, ", "),
		buildOrderBySentence(resource.OrderBy),
		buildPartitionBySentence(resource.PartitionBy),
		resource.Comment,
	)
}
