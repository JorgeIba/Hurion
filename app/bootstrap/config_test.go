package bootstrap

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		yamlConfig string
		useFile    string
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
				assert.Equal(t, []string{"chan-b-1", "chan-b-2"}, cfg.Classes[0].ChannelIDs)
				assert.Equal(t, []string{"role-b-1"}, cfg.Classes[0].RoleIDs)

				// Check Avanzados
				assert.Equal(t, "avanzados", cfg.Classes[1].Code)
				assert.Equal(t, "Avanzados 2025", cfg.Classes[1].Name)
				assert.Equal(t, []string{"chan-a-1"}, cfg.Classes[1].ChannelIDs)
				assert.Equal(t, []string{"role-a-1", "role-a-2"}, cfg.Classes[1].RoleIDs)

				// Check Intermedios
				assert.Equal(t, "intermedios", cfg.Classes[2].Code)
				assert.Equal(t, "Intermedios 2025", cfg.Classes[2].Name)
				assert.Equal(t, []string{"chan-i-1"}, cfg.Classes[2].ChannelIDs)
				assert.Equal(t, []string{"role-i-1"}, cfg.Classes[2].RoleIDs)
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
			name: "should error when required fields are missing in YAML",
			yamlConfig: `
admin_role_id: "only-admin-present"
`,
			wantErr: true,
		},
		{
			name: "should return error for invalid YAML",
			yamlConfig: `
admin_role_id: "valid"
mentor_role_id: [
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var configPath string
			if tt.useFile != "" {
				configPath = tt.useFile
			} else if tt.yamlConfig != "" {
				tempDir := t.TempDir()
				configPath = filepath.Join(tempDir, "config.yaml")
				err := os.WriteFile(configPath, []byte(tt.yamlConfig), 0644)
				require.NoError(t, err)
			}

			cfg, err := LoadConfig(configPath)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			tt.assertFunc(t, cfg)
		})
	}
}

func TestLoadConfig_NoEnvOverride(t *testing.T) {
	// This test ensures that environment variables do NOT override YAML IDs
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	yamlContent := `
admin_role_id: "yaml-admin"
mentor_role_id: "yaml-mentor"
`
	err := os.WriteFile(configPath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Set environment variables to verify they are ignored for ID configuration
	os.Setenv("DISCORD_ADMIN_ROLE_ID", "env-admin-override")
	defer os.Unsetenv("DISCORD_ADMIN_ROLE_ID")

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)

	// ASSERTION: The YAML value must persist, environment must be ignored for these IDs
	assert.Equal(t, "yaml-admin", cfg.AdminRoleID, "Environment variable should NOT override YAML configuration for IDs")
}
