package config

import (
	"errors"
	"os"

	"github.com/BurntSushi/toml"
)

// Config holds the top-level application configuration.
type Config struct {
	Display DisplayConfig `toml:"display"`
	Git     GitConfig     `toml:"git"`
	Tmux    TmuxConfig    `toml:"tmux"`
}

// DisplayConfig holds display-related settings.
type DisplayConfig struct {
	Theme         string `toml:"theme"`
	RefreshMs     int    `toml:"refresh_ms"`
	ShowRateLimit bool   `toml:"show_rate_limit"`
	ShowActivity  bool   `toml:"show_activity"`
	ShowGit       bool   `toml:"show_git"`
}

// GitConfig holds git-related settings.
type GitConfig struct {
	ShowDirty       bool `toml:"show_dirty"`
	ShowAheadBehind bool `toml:"show_ahead_behind"`
	ShowFileStats   bool `toml:"show_file_stats"`
}

// TmuxConfig holds tmux-related settings.
type TmuxConfig struct {
	AutoDetect bool   `toml:"auto_detect"`
	Position   string `toml:"position"`
	Size       int    `toml:"size"`
}

// Default returns a Config with all default values.
func Default() *Config {
	return &Config{
		Display: DisplayConfig{
			Theme:         "default",
			RefreshMs:     500,
			ShowRateLimit: true,
			ShowActivity:  true,
			ShowGit:       true,
		},
		Git: GitConfig{
			ShowDirty:       true,
			ShowAheadBehind: false,
			ShowFileStats:   false,
		},
		Tmux: TmuxConfig{
			AutoDetect: true,
			Position:   "bottom",
			Size:       30,
		},
	}
}

// Load reads a TOML config file from the given path.
// If the file is missing or contains invalid TOML, defaults are returned with no error.
func Load(path string) (*Config, error) {
	cfg := Default()

	_, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}

	if _, err := toml.DecodeFile(path, cfg); err != nil {
		// Invalid TOML or other read error: return defaults
		return Default(), nil
	}

	return cfg, nil
}
