package bootstrap

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	// We do NOT use t.Parallel() here because some test cases use t.Setenv(),
	// and Go does not allow parallel execution when environment variables are being modified.

	tests := []struct {
		name       string
		yamlConfig string
		useFile    string
		envVars    map[string]string // Used to verify environment isolation
		wantErr    bool
		assertFunc func(t *testing.T, cfg Config)
	}{
		{
			name:    "should load complex configuration from file",
			useFile: "testdata/config.yaml",
			assertFunc: func(t *testing.T, cfg Config) {
				assert.Equal(t, "admin-role-123", cfg.AdminRoleID)
				assert.Equal(t, "mentor-role-456", cfg.MentorRoleID)
				require.Len(t, cfg.Classes, 3)

				// Check Basicos
				assert.Equal(t, "basicos", cfg.Classes[0].Code)
				assert.Equal(t, "Basicos 2025", cfg.Classes[0].Name)
				assert.Equal(t, []string{"chan-b-1", "chan-b-2"}, cfg.Classes[0].RoomIDs)
				assert.Equal(t, []string{"role-b-1"}, cfg.Classes[0].RoleIDs)

				// Check Avanzados
				assert.Equal(t, "avanzados", cfg.Classes[1].Code)
				assert.Equal(t, "Avanzados 2025", cfg.Classes[1].Name)
				assert.Equal(t, []string{"chan-a-1"}, cfg.Classes[1].RoomIDs)
				assert.Equal(t, []string{"role-a-1", "role-a-2"}, cfg.Classes[1].RoleIDs)

				// Check Intermedios
				assert.Equal(t, "intermedios", cfg.Classes[2].Code)
				assert.Equal(t, "Intermedios 2025", cfg.Classes[2].Name)
				assert.Equal(t, []string{"chan-i-1"}, cfg.Classes[2].RoomIDs)
				assert.Equal(t, []string{"role-i-1"}, cfg.Classes[2].RoleIDs)

				// Check Test Mode & Whitelist
				assert.True(t, cfg.TestMode)
				assert.Equal(t, []string{"user-1", "user-2"}, cfg.WhitelistUserIDs)
				assert.Equal(t, []string{"role-1", "role-2"}, cfg.WhitelistRoleIDs)
			},
		},
		{
			name: "should load inline YAML configuration",
			yamlConfig: `
admin_role_id: "inline-admin"
mentor_role_id: "inline-mentor"
classes: []
`,
			assertFunc: func(t *testing.T, cfg Config) {
				assert.Equal(t, "inline-admin", cfg.AdminRoleID)
				assert.Equal(t, "inline-mentor", cfg.MentorRoleID)
				assert.Empty(t, cfg.Classes)
			},
		},
		{
			name: "environment variables should NOT override YAML configuration",
			yamlConfig: `
admin_role_id: "original-yaml-admin"
mentor_role_id: "original-yaml-mentor"
`,
			envVars: map[string]string{
				"DISCORD_ADMIN_ROLE_ID": "malicious-env-override",
			},
			assertFunc: func(t *testing.T, cfg Config) {
				assert.Equal(t, "original-yaml-admin", cfg.AdminRoleID)
			},
		},
		{
			name: "should error on unknown keys (strict unmarshaling)",
			yamlConfig: `
admin_role_id: "val"
mentor_role_id: "val"
unknown_field: "this should trigger an error"
`,
			wantErr: true,
		},
		{
			name: "should error when required fields are missing in YAML",
			yamlConfig: `
admin_role_id: "only-admin-present"
`,
			wantErr: true,
		},
		{
			name: "should return error for invalid YAML syntax",
			yamlConfig: `
admin_role_id: "valid"
mentor_role_id: [
`,
			wantErr: true,
		},
		{
			name: "should error if class configuration is invalid (missing code)",
			yamlConfig: `
admin_role_id: "val"
mentor_role_id: "val"
classes:
  - name: "Missing Code"
`,
			wantErr: true,
		},
		{
			name:    "should error when explicitly provided file is missing",
			useFile: "non_existent.yaml",
			wantErr: true,
		},
		{
			name: "should error if test_mode is enabled but whitelists are empty",
			yamlConfig: `
admin_role_id: "val"
mentor_role_id: "val"
test_mode: true
whitelist_user_ids: []
whitelist_role_ids: []
`,
			wantErr: true,
		},
		{
			name: "should NOT error if test_mode is disabled and whitelists are empty",
			yamlConfig: `
admin_role_id: "val"
mentor_role_id: "val"
test_mode: false
`,
			assertFunc: func(t *testing.T, cfg Config) {
				assert.False(t, cfg.TestMode)
				assert.Empty(t, cfg.WhitelistUserIDs)
				assert.Empty(t, cfg.WhitelistRoleIDs)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var configPath string
			if tt.useFile != "" {
				configPath = tt.useFile
			} else if tt.yamlConfig != "" {
				tempDir := t.TempDir()
				configPath = filepath.Join(tempDir, "config.yaml")
				err := os.WriteFile(configPath, []byte(tt.yamlConfig), 0644)
				require.NoError(t, err)
			}

			// Apply environment variables to verify isolation
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			cfg, err := LoadConfig(configPath)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.assertFunc != nil {
				tt.assertFunc(t, cfg)
			}
		})
	}
}
