package bootstrap

import (
	"fmt"
)

type OperatingMode int

const (
	ModeOnlineDaemon OperatingMode = iota
	ModeOnlineScript
	ModeMockScript
)

func (m OperatingMode) String() string {
	switch m {
	case ModeOnlineDaemon:
		return "Online Daemon"
	case ModeOnlineScript:
		return "Online Script"
	case ModeMockScript:
		return "Mock Script"
	default:
		return "Unknown Mode"
	}
}

type RunConfig struct {
	Mode       OperatingMode
	ScriptPath string
}

func NewRunConfig(mockDiscord bool, scriptPath string) (RunConfig, error) {
	hasScript := scriptPath != ""

	if mockDiscord && !hasScript {
		return RunConfig{}, fmt.Errorf("FATAL: --mock-discord flag can only be used in combination with --script")
	}

	cfg := RunConfig{
		ScriptPath: scriptPath,
	}

	switch {
	case mockDiscord && hasScript:
		cfg.Mode = ModeMockScript
	case !mockDiscord && hasScript:
		cfg.Mode = ModeOnlineScript
	case !mockDiscord && !hasScript:
		cfg.Mode = ModeOnlineDaemon
	}

	return cfg, nil
}

func (c RunConfig) UsesConsoleSender() bool {
	return c.Mode != ModeOnlineDaemon
}

func (c RunConfig) UsesMockProvider() bool {
	return c.Mode == ModeMockScript
}
