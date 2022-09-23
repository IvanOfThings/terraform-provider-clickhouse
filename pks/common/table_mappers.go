package common

import (
	"regexp"
)

func mapToPtrString(input interface{}) *string {
	var _value *string = nil
	if input != nil && input.(string) != "" {
		_value = &make([]string, 1)[0]
		*_value = input.(string)
	}
	return _value
}

func MapPartitionBy(partitionBy interface{}, mappedColumns []CHColumn) (*[]TPartitionBy, error) {
	if partitionBy == nil {
		return nil, nil
	}

	mappedPartitionBy := make([]TPartitionBy, 0)
	data := partitionBy.([]interface{})
	for _, _data := range data {
		_mappedData := _data.(map[string]interface{})

		_by := _mappedData["by"].(string)
		err := ValidateParams(mappedColumns, []string{_by}, "partition_by")
		if err != nil {
			return nil, err
		}
		_mappedPartitionBy := TPartitionBy{
			by:                 _by,
			partition_function: mapToPtrString(_mappedData["partition_function"]),
		}
		mappedPartitionBy = append(mappedPartitionBy, _mappedPartitionBy)
	}
	return &mappedPartitionBy, nil
}

func MapColumns(dataColumns []interface{}) []CHColumn {
	columns := make([]CHColumn, 0)
	for i := 0; i < len(dataColumns); i++ {
		data := dataColumns[i].(map[string]interface{})
		column := CHColumn{

			name:      data["name"].(string),
			data_type: data["type"].(string),
		}
		columns = append(columns, column)
	}
	return columns
}

func MapTableToDatasource(table clickhouseTable) (*DataSourceCHTable, error) {
	columns := make([]dataSourceClickhouseColumn, 0)
	for i := 0; i < len(table.columns); i++ {
		column := dataSourceClickhouseColumn{
			name:      table.columns[i].column_name,
			data_type: table.columns[i].data_type,
		}
		columns = append(columns, column)
	}

	r, _ := regexp.Compile("MergeTree\\((?P<engine_params>[^)]*)\\)")
	matches := r.FindStringSubmatch(table.engine_full)
	engine_params_index := r.SubexpIndex("engine_params")
	engine_params := make([]string, 0)
	// var cluster []string = nil
	if engine_params_index != -1 {

		regex := regexp.MustCompile("[, ]+")
		params := regex.Split(matches[r.SubexpIndex("engine_params")], -1)
		for _, p := range params {
			engine_params = append(engine_params, p)
		}
		// cluster = make([]string, 1)
		// cluster[0] = strings.Split(engine_params[0], "/")[2]
	}

	comment, cluster, err := UnmarshalComment(table.comment)
	if err != nil {
		return nil, err
	}

	return &DataSourceCHTable{
		Database:      table.database,
		Table_name:    table.table_name,
		Engine_full:   table.engine_full,
		Engine:        table.engine,
		Cluster:       &cluster,
		Comment:       comment,
		Engine_params: &engine_params,
		Columns:       columns,
	}, nil
}

func MapArrayInterfaceToArrayOfStrings(in []interface{}) []string {
	ret := make([]string, 0)
	for _, s := range in {
		ret = append(ret, s.(string))
	}
	return ret
}
