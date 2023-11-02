package common

import (
	"fmt"
	"strings"

	ch "github.com/leprosus/golang-clickhouse"
)

func getColumns(conn *ch.Conn, database string, tableName string, errors *[]error) []clickhouseTablesColumn {

	queryRows := `
	SELECT database, table, name, type, position, 
	default_kind ,
	default_expression ,
	data_compressed_bytes ,
	data_uncompressed_bytes	 ,
	marks_bytes	 , comment,   
	is_in_partition_key, is_in_sorting_key, is_in_primary_key, is_in_sampling_key, 
	compression_codec ,
	character_octet_length ,
	numeric_precision,
	numeric_precision_radix,
	numeric_scale ,
	datetime_precision 
	from system.columns
	where database = '%v' AND table = '%v'
	`
	iter, err := conn.Fetch(fmt.Sprintf(queryRows, database, tableName))

	if err != nil {
		*errors = append(*errors, err)
		return nil
	}

	columns := make([]clickhouseTablesColumn, 0)
	for i := 0; iter.Next(); i++ {
		result := iter.Result
		// var errors = make([]error, 0)
		database := ToString(result, "database", errors)
		table := ToString(result, "table", errors)
		column_name := ToString(result, "name", errors)
		data_type := ToString(result, "type", errors)
		position := toUint64(result, "position", errors)
		default_kind := ToString(result, "default_kind", errors)
		default_expression := ToString(result, "default_expression", errors)
		data_compressed_bytes := ToString(result, "data_compressed_bytes", errors)
		data_uncompressed_bytes := ToString(result, "data_uncompressed_bytes", errors)
		marks_bytes := ToString(result, "marks_bytes", errors)
		comment := ToString(result, "comment", errors)
		is_in_partition_key := toUint64(result, "is_in_partition_key", errors)
		is_in_sorting_key := toUint64(result, "is_in_sorting_key", errors)
		is_in_primary_key := toUint64(result, "is_in_primary_key", errors)
		is_in_sampling_key := toUint64(result, "is_in_sampling_key", errors)
		compression_codec := ToString(result, "compression_codec", errors)
		character_octet_length := toUint64(result, "character_octet_length", errors)
		numeric_precision := toUint64(result, "numeric_precision", errors)
		numeric_precision_radix := toUint64(result, "numeric_precision_radix", errors)
		numeric_scale := toUint64(result, "numeric_scale", errors)
		datetime_precision := toUint64(result, "datetime_precision", errors)
		if len(*errors) > 0 {
			return nil
		}
		row := clickhouseTablesColumn{
			database:                *database,
			table:                   *table,
			column_name:             *column_name,
			data_type:               *data_type,
			position:                *position,
			default_kind:            *default_kind,
			default_expression:      *default_expression,
			data_compressed_bytes:   *data_compressed_bytes,
			data_uncompressed_bytes: *data_uncompressed_bytes,
			marks_bytes:             *marks_bytes,
			comment:                 *comment,
			is_in_partition_key:     *is_in_partition_key,
			is_in_sorting_key:       *is_in_sorting_key,
			is_in_primary_key:       *is_in_primary_key,
			is_in_sampling_key:      *is_in_sampling_key,
			compression_codec:       *compression_codec,
			character_octet_length:  character_octet_length,
			numeric_precision:       numeric_precision,
			numeric_precision_radix: numeric_precision_radix,
			numeric_scale:           numeric_scale,
			datetime_precision:      datetime_precision,
		}

		columns = append(columns, row)
		if err != nil {
			*errors = append(*errors, err)
			return nil
		}
	}
	return columns
}

func GetResourceNamesOnDataBases(conn *ch.Conn, databaseName string) (resources *CHDbResources, errors []error) {
	query := fmt.Sprintf("SELECT `name` FROM system.tables where database = '%v'", databaseName)
	errors = make([]error, 0)
	iter, err := conn.Fetch(query)
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	tableNames := make([]string, 0)

	for i := 0; iter.Next(); i++ {
		result := iter.Result

		table := *ToString(result, "name", &errors)
		if len(errors) > 0 {
			return nil, errors
		}

		tableNames = append(tableNames, table)
	}
	return &CHDbResources{
		TableNames: tableNames,
	}, errors
}

func GetTables(conn *ch.Conn, data *CHDataBase, errors *[]error) []clickhouseTable {
	var query string
	if data != nil {
		query = fmt.Sprintf("SELECT `database`, `name`, `engine_full`, `engine`, `comment` FROM system.tables where database = '%v' AND name = '%v'", data.Database, data.Name)
	} else {
		query = fmt.Sprintf("SELECT `database`, `name`, `engine_full`, `engine`, `comment` FROM system.tables")

	}
	iter, err := conn.Fetch(query)
	if err != nil {
		*errors = append(*errors, err)
		return nil
	}

	tables := make([]clickhouseTable, 0)

	for i := 0; iter.Next(); i++ {
		result := iter.Result

		database := *ToString(result, "database", errors)
		table_name := *ToString(result, "name", errors)

		var columns []clickhouseTablesColumn
		columns = getColumns(conn, database, table_name, errors)

		table := clickhouseTable{
			database:    database,
			name:        table_name,
			engine_full: *ToString(result, "engine_full", errors),
			engine:      *ToString(result, "engine", errors),
			comment:     *ToString(result, "comment", errors),
			columns:     columns,
		}
		if len(*errors) > 0 {
			return nil
		}

		tables = append(tables, table)
	}
	return tables
}

func buildColumnSentence(col CHColumn) string {
	createRowScript := "`%name%` %type% %nullability% %special% %compresion_codec% %ttl_expression%"
	rowInProgres := strings.Replace(createRowScript, "%name%", col.name, 1)
	rowInProgres = strings.Replace(rowInProgres, "%type%", col.data_type, 1)

	rowInProgres = strings.Replace(rowInProgres, "%nullability%", col.nullability, 1)
	rowInProgres = strings.Replace(rowInProgres, "%special%", col.special, 1)
	rowInProgres = strings.Replace(rowInProgres, "%compresion_codec%", col.compresion_codec, 1)
	rowInProgres = strings.Replace(rowInProgres, "%ttl_expression%", col.ttl_expression, 1)

	return rowInProgres
}

func buildColumnsSentence(cols []CHColumn) []string {
	outColumn := make([]string, 0)
	for i := 0; i < len(cols); i++ {
		outColumn = append(outColumn, "\t"+buildColumnSentence(cols[i]))
	}
	return outColumn
}

func buildPartitionBySentence(partition_by []TPartitionBy) string {

	partitionBySentence := `PARTITION BY (%partition%)`
	partialPartitionBy := `%partitionTo%(%partial_partition%)`
	clauses := make([]string, 0)
	for _, partitionClause := range partition_by {
		if partitionClause.partition_function == nil || *partitionClause.partition_function == "" {
			clauses = append(clauses, partitionClause.by)
		} else {
			_partialPartitionBy := strings.Replace(partialPartitionBy, "%partial_partition%", partitionClause.by, 1)
			clauses = append(clauses, strings.Replace(_partialPartitionBy, "%partitionTo%", *partitionClause.partition_function, 1))
		}
	}
	return strings.Replace(partitionBySentence, "%partition%", strings.Join(clauses, ", "), 1)
}

func buildOrderBySentence(order_by []string) string {
	orderBySentence := `ORDER BY (%order_by%)`
	return strings.Replace(orderBySentence, "%order_by%", strings.Join(order_by, ", "), 1)
}

func BuildCreateONClusterSentence(mappedColumns []CHColumn, db_name string, table_name string, cluster string, defaultCluster string, engine string, order_by []string, engine_params []string, partition_by *[]TPartitionBy, comment string) (query string, clusterToUse string) {
	columnsStatement := ""
	if len(mappedColumns) > 0 {
		columnsList := buildColumnsSentence(mappedColumns)
		columnsStatement = "(" + strings.Join(columnsList, ",\n") + ")\n"
	}

	clusterStatement, clusterToUse := GetClusterStatement(cluster, defaultCluster)

	createTableScript := `CREATE TABLE %db_name%.%table_name% %cluster% %columns%  ENGINE = %engine%(%engine_params%) `
	query = strings.Replace(createTableScript, "%columns%", "\n"+columnsStatement, 1)
	query = strings.Replace(query, "%db_name%", db_name, 1)
	query = strings.Replace(query, "%table_name%", table_name, 1)
	query = strings.Replace(query, "%cluster%", clusterStatement, 1)
	query = strings.Replace(query, "%engine%", engine, 1)

	query = strings.Replace(query, "%engine_params%", strings.Join(engine_params, ", "), 1)

	if order_by != nil && len(order_by) > 0 {
		query = query + " " + buildOrderBySentence(order_by)
	}
	if partition_by != nil && len(*partition_by) > 0 {
		query = query + " " + buildPartitionBySentence(*partition_by)
	}

	query = strings.Replace(query+" COMMENT '%_comment_%'", "%_comment_%", comment, 1)

	return query, clusterToUse
}
