// Package config resolves xocode paths and default model settings. Values layer
// as defaults < settings file < environment, so xocode runs with zero config but
// remembers changes made in the settings screen.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds runtime settings.
type Config struct {
	// ClaudeModel is the model alias/id passed to `claude --model`.
	ClaudeModel string `json:"claude_model"`
	// ClaudeEffort is passed to `claude --effort`.
	ClaudeEffort string `json:"claude_effort"`
	// CursorModel is the model id passed to `cursor-agent --model`.
	CursorModel string `json:"cursor_model"`
	// Editor is the command used for editing the plan document.
	Editor string `json:"-"`
	// PlanDir is where plan documents are written (project-local by default).
	PlanDir string `json:"-"`
}

// persisted is the subset of Config written to the settings file.
type persisted struct {
	ClaudeModel  string `json:"claude_model"`
	ClaudeEffort string `json:"claude_effort"`
	CursorModel  string `json:"cursor_model"`
}

// Load builds a Config from defaults, then the settings file, then environment
// overrides. projectDir is the directory xocode was launched in.
func Load(projectDir string) Config {
	c := Config{
		ClaudeModel:  "opus", // alias resolves to the latest Opus (4.8)
		ClaudeEffort: "high",
		CursorModel:  "composer-2.5",
		Editor:       firstNonEmpty(os.Getenv("VISUAL"), os.Getenv("EDITOR"), "vi"),
		PlanDir:      filepath.Join(projectDir, ".xocode", "plans"),
	}

	// Settings file layer.
	if b, err := os.ReadFile(settingsPath()); err == nil {
		var p persisted
		if json.Unmarshal(b, &p) == nil {
			c.ClaudeModel = firstNonEmpty(p.ClaudeModel, c.ClaudeModel)
			c.ClaudeEffort = firstNonEmpty(p.ClaudeEffort, c.ClaudeEffort)
			c.CursorModel = firstNonEmpty(p.CursorModel, c.CursorModel)
		}
	}

	// Environment layer (wins).
	if v := os.Getenv("XOCODE_PLAN_DIR"); v != "" {
		c.PlanDir = v
	}
	if v := os.Getenv("XOCODE_CLAUDE_MODEL"); v != "" {
		c.ClaudeModel = v
	}
	if v := os.Getenv("XOCODE_CLAUDE_EFFORT"); v != "" {
		c.ClaudeEffort = v
	}
	if v := os.Getenv("XOCODE_CURSOR_MODEL"); v != "" {
		c.CursorModel = v
	}
	return c
}

// Save persists the model/effort settings to the per-user settings file.
func Save(c Config) error {
	dir := StateDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(persisted{
		ClaudeModel:  c.ClaudeModel,
		ClaudeEffort: c.ClaudeEffort,
		CursorModel:  c.CursorModel,
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath(), b, 0o644)
}

func settingsPath() string {
	return filepath.Join(StateDir(), "settings.json")
}

// StateDir returns the per-user directory for xocode state. Follows XDG when
// set, else ~/.config/xocode.
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
