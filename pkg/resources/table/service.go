package resourcetable

import (
	"context"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/IvanOfThings/terraform-provider-clickhouse/pkg/common"
)

type CHTableService struct {
	CHConnection *driver.Conn
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
	if err != nil {
		return nil, fmt.Errorf("scanning Clickhouse table row: %v", err)
	}

	chTable.Columns, err = ts.getTableColumns(ctx, database, table)
	if err != nil {
		return nil, fmt.Errorf("getting columns for Clickhouse table: %v", err)
	}

	return &chTable, nil
}

func (ts *CHTableService) getTableColumns(ctx context.Context, database string, table string) ([]CHColumn, error) {
	query := fmt.Sprintf(
		"SELECT database, table, name, type FROM system.columns WHERE database = '%s' AND table = '%s'",
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
	query := fmt.Sprintf("DROP TABLE %s.%s %s SYNC", tableResource.Database, tableResource.Name, common.GetClusterStatement(tableResource.Cluster))
	err := (*ts.CHConnection).Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("deleting Clickhouse table: %v", err)
	}
	return nil
}
