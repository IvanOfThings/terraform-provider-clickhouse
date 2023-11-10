package common

import (
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type ApiClient struct {
	ClickhouseConnection *driver.Conn
	DefaultCluster       string
}
