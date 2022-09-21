package clickhouse_provider

type clickhouseTablesColumn struct {
	database                string
	table                   string
	column_name             string
	data_type               string
	position                uint64
	default_kind            string
	default_expression      string
	data_compressed_bytes   string
	data_uncompressed_bytes string
	marks_bytes             string
	comment                 string
	is_in_partition_key     uint64
	is_in_sorting_key       uint64
	is_in_primary_key       uint64
	is_in_sampling_key      uint64
	compression_codec       string
	character_octet_length  *uint64
	numeric_precision       *uint64
	numeric_precision_radix *uint64
	numeric_scale           *uint64
	datetime_precision      *uint64
}

type clickhouseTable struct {
	database    string
	table_name  string
	engine      string
	engine_full string
	comment     string
	columns     []clickhouseTablesColumn
}

type CHDataBase struct {
	database   string
	table_name string
}

type dataSourceClickhouseColumn struct {
	name      string
	data_type string
}

type dataSourceCHTable struct {
	database      string
	table_name    string
	engine_full   string
	engine        string
	cluster       *string
	comment       string
	engine_params *[]string
	columns       []dataSourceClickhouseColumn
}

type CHColumn struct {
	name             string
	data_type        string
	nullability      string
	special          string
	compresion_codec string
	ttl_expression   string
}

type TPartitionBy struct {
	by                 string
	partition_function *string
}
