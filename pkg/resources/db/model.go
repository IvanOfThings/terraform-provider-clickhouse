package resourcedb

import "github.com/IvanOfThings/terraform-provider-clickhouse/pkg/resources/table"

type CHDBResources struct {
	CHTables []resourcetable.CHTable
}
