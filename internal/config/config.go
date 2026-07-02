// Package config resolves xocode paths and default model settings. Everything
// has a sensible hardcoded default so xocode runs with zero configuration.
package config

import (
	"os"
	"path/filepath"
)

// Config holds runtime settings. Currently sourced from defaults + env overrides;
// a config file can be layered on later without changing callers.
type Config struct {
	// ClaudeModel is the model alias/id passed to `claude --model`.
	ClaudeModel string
	// ClaudeEffort is passed to `claude --effort`.
	ClaudeEffort string
	// CursorModel is the model id passed to `cursor-agent --model`.
	CursorModel string
	// Editor is the command used for editing the plan document.
	Editor string
	// PlanDir is where plan documents are written (project-local by default).
	PlanDir string
}

// Load builds a Config from defaults and environment overrides. projectDir is
// the directory xocode was launched in (used for the project-local plan dir).
func Load(projectDir string) Config {
	c := Config{
		ClaudeModel:  "opus", // alias resolves to the latest Opus (4.8)
		ClaudeEffort: "high",
		CursorModel:  "composer-2.5",
		Editor:       firstNonEmpty(os.Getenv("VISUAL"), os.Getenv("EDITOR"), "vi"),
		PlanDir:      filepath.Join(projectDir, ".xocode", "plans"),
	}
	if v := os.Getenv("XOCODE_PLAN_DIR"); v != "" {
		c.PlanDir = v
	}
	if v := os.Getenv("XOCODE_CLAUDE_MODEL"); v != "" {
		c.ClaudeModel = v
	}
	if v := os.Getenv("XOCODE_CURSOR_MODEL"); v != "" {
		c.CursorModel = v
	}
	return c
}

// StateDir returns the per-user directory for xocode state (e.g. onboarding
// completion marker). Follows XDG when set, else ~/.config/xocode.
func StateDir() string {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "xocode")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".xocode"
	}
	return filepath.Join(home, ".config", "xocode")
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
