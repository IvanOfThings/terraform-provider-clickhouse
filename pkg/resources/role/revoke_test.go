package resourcerole

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetRevokeQuery(t *testing.T) {
	tests := []struct {
		name       string
		roleName   string
		privileges []string
		database   string
		expected   string
	}{
		{
			name:       "revoke global privilege - REMOTE",
			roleName:   "test_role",
			privileges: []string{"REMOTE"},
			database:   "*",
			expected:   "REVOKE REMOTE ON *.* FROM test_role",
		},
		{
			name:       "revoke global privilege - S3",
			roleName:   "test_role",
			privileges: []string{"S3"},
			database:   "*",
			expected:   "REVOKE S3 ON *.* FROM test_role",
		},
		{
			name:       "revoke multiple global privileges",
			roleName:   "test_role",
			privileges: []string{"REMOTE", "S3"},
			database:   "*",
			expected:   "REVOKE REMOTE,S3 ON *.* FROM test_role",
		},
		{
			name:       "revoke database privilege - regular database",
			roleName:   "test_role",
			privileges: []string{"SELECT", "INSERT"},
			database:   "mydb",
			expected:   "REVOKE SELECT,INSERT ON mydb.* FROM test_role",
		},
		{
			name:       "revoke database privilege - system database",
			roleName:   "test_role",
			privileges: []string{"SELECT"},
			database:   "system",
			expected:   "REVOKE SELECT ON system.* FROM test_role",
		},
		{
			name:       "revoke mixed privileges - global and database",
			roleName:   "test_role",
			privileges: []string{"REMOTE", "SELECT"},
			database:   "*",
			expected:   "REVOKE REMOTE ON *.* FROM test_role; REVOKE SELECT ON *.* FROM test_role",
		},
		{
			name:       "revoke all expandable privileges",
			roleName:   "test_role",
			privileges: []string{"REMOTE", "S3", "AZURE", "HDFS"},
			database:   "*",
			expected:   "REVOKE REMOTE,S3,AZURE,HDFS ON *.* FROM test_role",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getRevokeQuery(tt.roleName, tt.privileges, tt.database)
			assert.Equal(t, tt.expected, result)
		})
	}
}
