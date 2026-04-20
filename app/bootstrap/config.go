package bootstrap

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// DefaultConfigPath is the fallback filename used if no path is provided.
const DefaultConfigPath = "config.yaml"

// ClassConfig represents the configuration for a single class.
type ClassConfig struct {
	Code       string   `mapstructure:"code"`
	Name       string   `mapstructure:"name"`
	ChannelIDs []string `mapstructure:"channel_ids"`
	RoleIDs    []string `mapstructure:"role_ids"`
}

// Config represents the global application configuration.
// AdminRoleID and MentorRoleID must be provided in the configuration file.
type Config struct {
	AdminRoleID  string        `mapstructure:"admin_role_id"`
	MentorRoleID string        `mapstructure:"mentor_role_id"`
	Classes      []ClassConfig `mapstructure:"classes"`
}

// LoadConfig reads the configuration strictly from a YAML file.
// Environment variables are not used for these configuration IDs.
// If configPath is empty, it attempts to load from DefaultConfigPath.
func LoadConfig(configPath string) (Config, error) {
	v := viper.New()

	path := strings.TrimSpace(configPath)
	if path == "" {
		path = DefaultConfigPath
	}

	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		// Only return error if a path was explicitly provided.
		if configPath != "" {
			return Config{}, fmt.Errorf("failed to read config file %q: %w", path, err)
		}
		// If using default path and it's missing, we still need to validate fields below
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate required fields (must be present in the YAML)
	if cfg.AdminRoleID == "" {
		return Config{}, fmt.Errorf("admin_role_id is required in the configuration file")
	}
	if cfg.MentorRoleID == "" {
		return Config{}, fmt.Errorf("mentor_role_id is required in the configuration file")
	}

	return cfg, nil
}
