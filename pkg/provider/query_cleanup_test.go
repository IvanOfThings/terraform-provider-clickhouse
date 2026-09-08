package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	resourcerole "github.com/IvanOfThings/terraform-provider-clickhouse/pkg/resources/role"
	resourceuser "github.com/IvanOfThings/terraform-provider-clickhouse/pkg/resources/user"
	"github.com/stretchr/testify/require"
)

func TestReadsCloseRowsBeforeContextCancellation(t *testing.T) {
	reads := map[string]func(context.Context, *driver.Conn) error{
		"user": func(ctx context.Context, conn *driver.Conn) error {
			_, err := (&resourceuser.CHUserService{CHConnection: conn}).GetUser(ctx, "test_user")
			return err
		},
		"role": func(ctx context.Context, conn *driver.Conn) error {
			_, err := (&resourcerole.CHRoleService{CHConnection: conn}).GetRole(ctx, "test_role")
			return err
		},
	}

	for name, read := range reads {
		t.Run(name, func(t *testing.T) {
			rows := &cleanupRows{hasRow: true}
			var conn driver.Conn = &cleanupConn{rows: rows}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			require.NoError(t, read(ctx, &conn))
			// The RPC context can be canceled as soon as the read returns.
			require.True(t, rows.closed, "query results must be drained before the read returns")
		})
	}
}

type cleanupConn struct {
	driver.Conn
	rows *cleanupRows
}

func (c *cleanupConn) Query(_ context.Context, query string, _ ...any) (driver.Rows, error) {
	if strings.HasPrefix(query, "SHOW GRANTS") {
		return &cleanupRows{}, nil
	}
	return c.rows, nil
}

type cleanupRows struct {
	driver.Rows
	hasRow bool
	closed bool
}

func (r *cleanupRows) Next() bool {
	if r.hasRow {
		r.hasRow = false
		return true
	}
	r.Close()
	return false
}

func (r *cleanupRows) ScanStruct(any) error { return nil }

func (r *cleanupRows) Close() error {
	r.closed = true
	return nil
}
