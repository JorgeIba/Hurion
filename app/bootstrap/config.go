package bootstrap

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

const DefaultConfigPath = "config.yaml"

// ClassConfig represents the configuration for a single class.
type ClassConfig struct {
	Code          string   `mapstructure:"code"`
	Name          string   `mapstructure:"name"`
	RoomIDs       []string `mapstructure:"room_ids"`
	RoleIDs       []string `mapstructure:"role_ids"`
	MentorRoleIDs []string `mapstructure:"mentor_role_ids"`
}

// Validate ensures the class configuration is internally consistent.
func (c ClassConfig) Validate() error {
	if strings.TrimSpace(c.Code) == "" {
		return errors.New("class code is required")
	}
	if strings.TrimSpace(c.Name) == "" {
		return errors.New("class name is required")
	}
	for _, roleID := range c.MentorRoleIDs {
		if strings.TrimSpace(roleID) == "" {
			return errors.New("mentor_role_ids must not contain empty role IDs")
		}
	}
	return nil
}

// Config represents the global application configuration.
type Config struct {
	GuildID          string        `mapstructure:"guild_id"`
	AdminRoleID      string        `mapstructure:"admin_role_id"`
	MentorRoleID     string        `mapstructure:"mentor_role_id"`
	Classes          []ClassConfig `mapstructure:"classes"`
	TestMode         bool          `mapstructure:"test_mode"`
	WhitelistUserIDs []string      `mapstructure:"whitelist_user_ids"`
	WhitelistRoleIDs []string      `mapstructure:"whitelist_role_ids"`
}

// Validate ensures the entire configuration state is valid.
func (c Config) Validate() error {
	if strings.TrimSpace(c.AdminRoleID) == "" {
		return errors.New("admin_role_id is required")
	}
	if strings.TrimSpace(c.MentorRoleID) == "" {
		return errors.New("mentor_role_id is required")
	}

	if c.TestMode && len(c.WhitelistUserIDs) == 0 && len(c.WhitelistRoleIDs) == 0 {
		return errors.New("test_mode is enabled but no whitelist_user_ids or whitelist_role_ids are provided")
	}

	for i, class := range c.Classes {
		if err := class.Validate(); err != nil {
			return fmt.Errorf("invalid class at index %d: %w", i, err)
		}
	}

	return nil
}

// LoadConfig reads the configuration strictly from a YAML file.
// Unknown keys in the YAML will trigger an error (Strict Unmarshaling).
func LoadConfig(configPath string) (Config, error) {
	v := viper.New()

	path := strings.TrimSpace(configPath)
	if path == "" {
		path = DefaultConfigPath
	}

	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return Config{}, fmt.Errorf("failed to read config file %q: %w", path, err)
	}

	// We explicitly DO NOT bind environment variables for configuration IDs
	// to ensure the YAML file remains the sole source of truth for the domain.

	var cfg Config
	// UnmarshalExact returns an error if the YAML contains keys that do not match the struct.
	if err := v.UnmarshalExact(&cfg); err != nil {
		return Config{}, fmt.Errorf("failed to unmarshal config (strict mode): %w", err)
	}

	cfg.GuildID = strings.TrimSpace(cfg.GuildID)

	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// MentorRolesByClass preserves absent versus explicit empty overrides.
func (c Config) MentorRolesByClass() map[string][]string {
	roles := make(map[string][]string)
	for _, class := range c.Classes {
		if class.MentorRoleIDs != nil {
			roles[class.Code] = append([]string{}, class.MentorRoleIDs...)
		}
	}
	return roles
}
