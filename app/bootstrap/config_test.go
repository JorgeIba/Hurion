package bootstrap

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	// We do NOT use t.Parallel() here because we are modifying environment variables
	// which is a global state and would cause race conditions between tests.

	tests := []struct {
		name       string
		yamlConfig string
		useFile    string
		envVars    map[string]string
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
			name:    "environment variables should override YAML file values",
			useFile: "testdata/config.yaml",
			envVars: map[string]string{
				"DISCORD_ADMIN_ROLE_ID":  "override-admin",
				"DISCORD_MENTOR_ROLE_ID": "override-mentor",
			},
			assertFunc: func(t *testing.T, cfg Config) {
				// Overridden values
				assert.Equal(t, "override-admin", cfg.AdminRoleID)
				assert.Equal(t, "override-mentor", cfg.MentorRoleID)

				// Ensure non-overridden nested values are still present
				require.Len(t, cfg.Classes, 3)
				assert.Equal(t, "basicos", cfg.Classes[0].Code)
			},
		},
		{
			name: "should handle missing config file gracefully (only env vars)",
			envVars: map[string]string{
				"DISCORD_MENTOR_ROLE_ID": "env-mentor-id",
			},
			assertFunc: func(t *testing.T, cfg Config) {
				assert.Equal(t, "env-mentor-id", cfg.MentorRoleID)
				assert.Empty(t, cfg.AdminRoleID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment variables before each test to ensure isolation
			os.Unsetenv("DISCORD_ADMIN_ROLE_ID")
			os.Unsetenv("DISCORD_MENTOR_ROLE_ID")

			var configPath string
			if tt.useFile != "" {
				configPath = tt.useFile
			} else if tt.yamlConfig != "" {
				tempDir := t.TempDir()
				configPath = filepath.Join(tempDir, "config.yaml")
				err := os.WriteFile(configPath, []byte(tt.yamlConfig), 0644)
				require.NoError(t, err)
			}

			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			cfg, err := LoadConfig(configPath)
			require.NoError(t, err)
			tt.assertFunc(t, cfg)
		})
	}
}
