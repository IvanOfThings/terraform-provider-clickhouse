package resourcerole

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGrantStatement(t *testing.T) {
	tests := []struct {
		name            string
		statement       string
		roleName        string
		expectedGrants  []CHGrant
		expectError     bool
		errorContains   string
	}{
		{
			name:      "global privilege - REMOTE",
			statement: "GRANT REMOTE ON *.* TO test_role",
			roleName:  "test_role",
			expectedGrants: []CHGrant{
				{RoleName: "test_role", AccessType: "REMOTE", Database: "*"},
			},
			expectError: false,
		},
		{
			name:      "global privilege - S3",
			statement: "GRANT S3 ON *.* TO test_role",
			roleName:  "test_role",
			expectedGrants: []CHGrant{
				{RoleName: "test_role", AccessType: "S3", Database: "*"},
			},
			expectError: false,
		},
		{
			name:      "database privilege - single",
			statement: "GRANT SELECT ON mydb.* TO test_role",
			roleName:  "test_role",
			expectedGrants: []CHGrant{
				{RoleName: "test_role", AccessType: "SELECT", Database: "mydb"},
			},
			expectError: false,
		},
		{
			name:      "database privilege - multiple",
			statement: "GRANT SELECT, INSERT ON mydb.* TO test_role",
			roleName:  "test_role",
			expectedGrants: []CHGrant{
				{RoleName: "test_role", AccessType: "SELECT", Database: "mydb"},
				{RoleName: "test_role", AccessType: "INSERT", Database: "mydb"},
			},
			expectError: false,
		},
		{
			name:      "system database privilege",
			statement: "GRANT SELECT ON system.* TO test_role",
			roleName:  "test_role",
			expectedGrants: []CHGrant{
				{RoleName: "test_role", AccessType: "SELECT", Database: "system"},
			},
			expectError: false,
		},
		{
			name:          "invalid statement - missing GRANT",
			statement:     "SELECT ON *.* TO test_role",
			roleName:      "test_role",
			expectError:   true,
			errorContains: "invalid grant statement",
		},
		{
			name:          "invalid statement - missing ON",
			statement:     "GRANT REMOTE *.* TO test_role",
			roleName:      "test_role",
			expectError:   true,
			errorContains: "missing ' ON '",
		},
		{
			name:          "invalid statement - missing TO",
			statement:     "GRANT REMOTE ON *.*",
			roleName:      "test_role",
			expectError:   true,
			errorContains: "missing ' TO '",
		},
		{
			name:          "table-level grant not supported",
			statement:     "GRANT SELECT ON mydb.mytable TO test_role",
			roleName:      "test_role",
			expectError:   true,
			errorContains: "table-level grants are not supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grants, err := parseGrantStatement(tt.statement, tt.roleName)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				assert.Nil(t, grants)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedGrants, grants)
			}
		})
	}
}

func TestGetGrantQuery(t *testing.T) {
	tests := []struct {
		name       string
		roleName   string
		privileges []string
		database   string
		expected   string
	}{
		{
			name:       "global privilege - REMOTE",
			roleName:   "test_role",
			privileges: []string{"REMOTE"},
			database:   "*",
			expected:   "GRANT REMOTE ON *.* TO test_role",
		},
		{
			name:       "global privilege - S3",
			roleName:   "test_role",
			privileges: []string{"S3"},
			database:   "*",
			expected:   "GRANT S3 ON *.* TO test_role",
		},
		{
			name:       "database privilege - regular database",
			roleName:   "test_role",
			privileges: []string{"SELECT", "INSERT"},
			database:   "mydb",
			expected:   "GRANT SELECT,INSERT ON mydb.* TO test_role",
		},
		{
			name:       "database privilege - system database",
			roleName:   "test_role",
			privileges: []string{"SELECT"},
			database:   "system",
			expected:   "GRANT CURRENT GRANTS (SELECT ON system.*) TO test_role",
		},
		{
			name:       "database privilege - wildcard database",
			roleName:   "test_role",
			privileges: []string{"SELECT"},
			database:   "*",
			expected:   "GRANT CURRENT GRANTS (SELECT ON *.*) TO test_role",
		},
		{
			name:       "mixed privileges - global and database",
			roleName:   "test_role",
			privileges: []string{"REMOTE", "SELECT"},
			database:   "*",
			expected:   "GRANT REMOTE ON *.* TO test_role; GRANT CURRENT GRANTS (SELECT ON *.*) TO test_role",
		},
		{
			name:       "multiple global privileges",
			roleName:   "test_role",
			privileges: []string{"REMOTE", "S3"},
			database:   "*",
			expected:   "GRANT REMOTE,S3 ON *.* TO test_role",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getGrantQuery(tt.roleName, tt.privileges, tt.database)
			assert.Equal(t, tt.expected, result)
		})
	}
}
