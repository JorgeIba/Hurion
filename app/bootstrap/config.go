package bootstrap

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

const DefaultConfigPath = "config.yaml"

// ClassConfig represents the configuration for a single class.
type ClassConfig struct {
	Code       string   `mapstructure:"code"`
	Name       string   `mapstructure:"name"`
	ChannelIDs []string `mapstructure:"channel_ids"`
	RoleIDs    []string `mapstructure:"role_ids"`
}

// Config represents the global application configuration.
type Config struct {
	AdminRoleID  string        `mapstructure:"admin_role_id"`
	MentorRoleID string        `mapstructure:"mentor_role_id"`
	Classes      []ClassConfig `mapstructure:"classes"`
}

// LoadConfig merges configuration from a YAML file and environment variables.
// Environment variables with the prefix "DISCORD_" override YAML values.
func LoadConfig(configPath string) (Config, error) {
	v := viper.New()

	if strings.TrimSpace(configPath) != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			return Config{}, fmt.Errorf("failed to read config file %q: %w", configPath, err)
		}
	}

	// Environment variable configuration
	v.SetEnvPrefix("DISCORD")
	v.AutomaticEnv()
	// Replace dots with underscores in env vars (e.g., DISCORD_ADMIN_ROLE_ID)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Explicitly bind environment variables to ensure they are picked up during Unmarshal
	// even when no config file is loaded.
	_ = v.BindEnv("admin_role_id")
	_ = v.BindEnv("mentor_role_id")

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return cfg, nil
}
