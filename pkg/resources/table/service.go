package resourcetable

import (
	"context"
	"fmt"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/Triple-Whale/terraform-provider-clickhouse/pkg/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type CHTableService struct {
	CHConnection *driver.Conn
}

// err := chTableService.UpdateComment(context.Background(), tableResource)
//
//	func (ts *CHTableService) UpdateComment(ctx context.Context, tableResource TableResource) error {
//		chTableService.UpdateColumns(context.Background(), addColumns, dropColumns)

func (ts *CHTableService) UpdateColumns(ctx context.Context, addColumns []interface{}, dropColumns []interface{}) error {
	// init empty erray of queries
	// querys := []string{}
	// // for each addColumn
	// // for _, addColumn := range addColumns {
	// // 	query := fmt.Sprintf("ALTER TABLE %s.%s ADD COLUMN %s", addColumn[0][

	// if len(addColumns) > 0 {
	// 	query := fmt.Sprintf("ALTER TABLE %s.%s ADD COLUMN %s", addColumns[0], addColumns[1], addColumns[2])
	// 	// fmt.
	// 	// err := (*ts.CHConnection).Exec(ctx, query)
	// 	// if err != nil {
	// 	// 	return fmt.Errorf("adding columns to Clickhouse table: %v", err)
	// 	// }
	// }
	// if len(dropColumns) > 0 {
	// 	query := fmt.Sprintf("ALTER TABLE %s.%s DROP COLUMN %s", dropColumns[0], dropColumns[1], dropColumns[2])
	// 	// err := (*ts.CHConnection).Exec(ctx, query)
	// 	// if err != nil {
	// 	// 	return fmt.Errorf("dropping columns from Clickhouse table: %v", err)
	// 	// }
	// }
	return nil
}

func copyToMap(iface interface{}) map[string]interface{} {
	mapCopy := iface.(map[string]interface{})
	mapNew := make(map[string]interface{})
	for k, v := range mapCopy {
		mapNew[k] = v
	}
	return mapNew
}

func (ts *CHTableService) UpdateTable(ctx context.Context, table TableResource, resourceData *schema.ResourceData) error {
	clusterStatement := common.GetClusterStatement(table.Cluster)
	if resourceData.HasChange("comment") {
		query := fmt.Sprintf("ALTER TABLE %s.%s %s MODIFY COMMENT '%s'", table.Database, table.Name, clusterStatement, table.Comment)
		err := (*ts.CHConnection).Exec(ctx, query)
		if err != nil {
			return err
		}
	}
	if resourceData.HasChange("ttl") {
		if table.TTL != "" {
			query := fmt.Sprintf("ALTER TABLE %s.%s %s MODIFY TTL %s", table.Database, table.Name, clusterStatement, table.TTL)
			err := (*ts.CHConnection).Exec(ctx, query)
			if err != nil {
				return fmt.Errorf("modifying TTL in Clickhouse table: %v", err)
			}
		} else {
			query := fmt.Sprintf("ALTER TABLE %s.%s %s REMOVE TTL", table.Database, table.Name, clusterStatement)
			err := (*ts.CHConnection).Exec(ctx, query)
			if err != nil {
				return fmt.Errorf("removing TTL from Clickhouse table: %v", err)
			}
		}
	}
	if resourceData.HasChange("column") {
		old, new := resourceData.GetChange("column")
		oldColumns := old.([]interface{})
		newColumns := new.([]interface{})

		// lookup map
		oldColumnsMap := make(map[string]map[string]interface{})
		for _, column := range oldColumns {
			columnMap := column.(map[string]interface{})
			columnName := columnMap["name"].(string)
			oldColumnsMap[columnName] = columnMap
		}

		newColumnsMap := make(map[string]map[string]interface{})
		for _, column := range newColumns {
			columnMap := column.(map[string]interface{})
			columnName := columnMap["name"].(string)
			newColumnsMap[columnName] = columnMap
		}

		location := "FIRST"
		for _, column := range newColumns {
			columnMap := copyToMap(column)
			columnMap["location"] = location

			if _, exists := oldColumnsMap[columnMap["name"].(string)]; !exists {
				query := fmt.Sprintf("ALTER TABLE %s.%s %s ADD COLUMN %s %s %s %s %s %s", table.Database, table.Name, clusterStatement, columnMap["name"], columnMap["type"], columnMap["default_kind"], columnMap["default_expression"], getComment(columnMap["comment"].(string)), columnMap["location"])
				err := (*ts.CHConnection).Exec(ctx, query)
				if err != nil {
					return fmt.Errorf("adding columns to Clickhouse table: %v", err)
				}
			} else {
				oldColumn := oldColumnsMap[columnMap["name"].(string)]
				if oldColumn["type"] != columnMap["type"] {
					query := fmt.Sprintf("ALTER TABLE %s.%s %s MODIFY COLUMN %s %s", table.Database, table.Name, clusterStatement, columnMap["name"], columnMap["type"])
					err := (*ts.CHConnection).Exec(ctx, query)
					if err != nil {
						return fmt.Errorf("modifying columns in Clickhouse table: %v", err)
					}
				}
				if oldColumn["comment"] != columnMap["comment"] {
					query := fmt.Sprintf("ALTER TABLE %s.%s %s COMMENT COLUMN %s '%s'", table.Database, table.Name, clusterStatement, columnMap["name"], columnMap["comment"])
					err := (*ts.CHConnection).Exec(ctx, query)
					if err != nil {
						return fmt.Errorf("modifying columns in Clickhouse table: %v", err)
					}
				}
				if oldColumn["default_kind"] != columnMap["default_kind"] || oldColumn["default_expression"] != columnMap["default_expression"] {
					query := fmt.Sprintf("ALTER TABLE %s.%s %s MODIFY COLUMN %s %s %s", table.Database, table.Name, clusterStatement, columnMap["name"], columnMap["default_kind"], columnMap["default_expression"])
					err := (*ts.CHConnection).Exec(ctx, query)
					if err != nil {
						return fmt.Errorf("modifying columns in Clickhouse table: %v", err)
					}
				}
			}

			location = "AFTER " + columnMap["name"].(string)
		}

		for _, column := range oldColumns {
			columnMap := column.(map[string]interface{})
			if _, exists := newColumnsMap[columnMap["name"].(string)]; !exists {
				query := fmt.Sprintf("ALTER TABLE %s.%s %s DROP COLUMN %s", table.Database, table.Name, clusterStatement, columnMap["name"])
				err := (*ts.CHConnection).Exec(ctx, query)
				if err != nil {
					return fmt.Errorf("dropping columns from Clickhouse table: %v", err)
				}

			}
		}

	}
	return nil
}

func (ts *CHTableService) GetDBTables(ctx context.Context, database string) ([]CHTable, error) {
	query := fmt.Sprintf("SELECT database, name FROM system.tables where database = '%s'", database)
	rows, err := (*ts.CHConnection).Query(ctx, query)

	if err != nil {
		return nil, fmt.Errorf("reading tables from Clickhouse: %v", err)
	}

	var tables []CHTable
	for rows.Next() {
		var table CHTable
		err := rows.ScanStruct(&table)
		if err != nil {
			return nil, fmt.Errorf("scanning Clickhouse table row: %v", err)
		}
		tables = append(tables, table)
	}

	return tables, nil
}

func (ts *CHTableService) GetTable(ctx context.Context, database string, table string) (*CHTable, error) {
	query := fmt.Sprintf("SELECT database, name, engine_full, engine, comment FROM system.tables where database = '%s' and name = '%s'", database, table)
	row := (*ts.CHConnection).QueryRow(ctx, query)

	if row.Err() != nil {
		return nil, fmt.Errorf("reading table from Clickhouse: %v", row.Err())
	}

	var chTable CHTable
	err := row.ScanStruct(&chTable)
	if err != nil && strings.Contains(err.Error(), "no rows in result set") {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scanning Clickhouse table row: %v", err)
	}

	chTable.Columns, err = ts.getTableColumns(ctx, database, table)
	if err != nil {
		return nil, fmt.Errorf("getting columns for Clickhouse table: %v", err)
	}

	chTable.Indexes, err = ts.getTableIndexes(ctx, database, table)
	if err != nil {
		return nil, fmt.Errorf("getting indexes for Clickhouse table: %v", err)
	}

	return &chTable, nil
}

func (ts *CHTableService) getTableIndexes(ctx context.Context, database string, table string) ([]CHIndex, error) {
	query := fmt.Sprintf(
		"SELECT name, expr, type, granularity FROM system.data_skipping_indices WHERE database = '%s' AND table = '%s'",
		database,
		table,
	)
	rows, err := (*ts.CHConnection).Query(ctx, query)

	if err != nil {
		return nil, fmt.Errorf("reading indexes from Clickhouse: %v", err)
	}

	var chIndexes []CHIndex
	for rows.Next() {
		var index CHIndex
		err := rows.ScanStruct(&index)
		if err != nil {
			return nil, fmt.Errorf("scanning Clickhouse index row: %v", err)
		}
		chIndexes = append(chIndexes, index)
	}
	return chIndexes, nil
}

func (ts *CHTableService) getTableColumns(ctx context.Context, database string, table string) ([]CHColumn, error) {
	query := fmt.Sprintf(
		"SELECT database, table, name, type, comment, default_kind, default_expression FROM system.columns WHERE database = '%s' AND table = '%s'",
		database,
		table,
	)
	rows, err := (*ts.CHConnection).Query(ctx, query)

	if err != nil {
		return nil, fmt.Errorf("reading columns from Clickhouse: %v", err)
	}

	var chColumns []CHColumn
	for rows.Next() {
		var column CHColumn
		err := rows.ScanStruct(&column)
		if err != nil {
			return nil, fmt.Errorf("scanning Clickhouse column row: %v", err)
		}
		chColumns = append(chColumns, column)
	}
	return chColumns, nil
}

func (ts *CHTableService) CreateTable(ctx context.Context, tableResource TableResource) error {
	query := buildCreateOnClusterSentence(tableResource)
	err := (*ts.CHConnection).Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("creating Clickhouse table: %v", err)
	}
	return nil
}

func (ts *CHTableService) DeleteTable(ctx context.Context, tableResource TableResource) error {
	query := fmt.Sprintf("DROP TABLE if exists %s.%s %s", tableResource.Database, tableResource.Name, common.GetClusterStatement(tableResource.Cluster))
	err := (*ts.CHConnection).Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("deleting Clickhouse table: %v", err)
	}
	return nil
}
