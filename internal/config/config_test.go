package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Default()

	// Display defaults
	if cfg.Display.Theme != "default" {
		t.Errorf("Display.Theme = %q, want %q", cfg.Display.Theme, "default")
	}
	if cfg.Display.RefreshMs != 500 {
		t.Errorf("Display.RefreshMs = %d, want %d", cfg.Display.RefreshMs, 500)
	}
	if cfg.Display.ShowRateLimit != true {
		t.Errorf("Display.ShowRateLimit = %v, want %v", cfg.Display.ShowRateLimit, true)
	}
	if cfg.Display.ShowActivity != true {
		t.Errorf("Display.ShowActivity = %v, want %v", cfg.Display.ShowActivity, true)
	}
	if cfg.Display.ShowGit != true {
		t.Errorf("Display.ShowGit = %v, want %v", cfg.Display.ShowGit, true)
	}

	// Git defaults
	if cfg.Git.ShowDirty != true {
		t.Errorf("Git.ShowDirty = %v, want %v", cfg.Git.ShowDirty, true)
	}
	if cfg.Git.ShowAheadBehind != false {
		t.Errorf("Git.ShowAheadBehind = %v, want %v", cfg.Git.ShowAheadBehind, false)
	}
	if cfg.Git.ShowFileStats != false {
		t.Errorf("Git.ShowFileStats = %v, want %v", cfg.Git.ShowFileStats, false)
	}

	// Tmux defaults
	if cfg.Tmux.AutoDetect != true {
		t.Errorf("Tmux.AutoDetect = %v, want %v", cfg.Tmux.AutoDetect, true)
	}
	if cfg.Tmux.Position != "bottom" {
		t.Errorf("Tmux.Position = %q, want %q", cfg.Tmux.Position, "bottom")
	}
	if cfg.Tmux.Size != 30 {
		t.Errorf("Tmux.Size = %d, want %d", cfg.Tmux.Size, 30)
	}
}

func TestLoadConfigMissing(t *testing.T) {
	cfg, err := Load("/tmp/nonexistent-codex-hud-config-87654321.toml")
	if err != nil {
		t.Fatalf("Load() returned error for missing file: %v", err)
	}

	// Should return defaults when file is missing
	expected := Default()
	if cfg.Display.Theme != expected.Display.Theme {
		t.Errorf("Display.Theme = %q, want %q", cfg.Display.Theme, expected.Display.Theme)
	}
	if cfg.Display.RefreshMs != expected.Display.RefreshMs {
		t.Errorf("Display.RefreshMs = %d, want %d", cfg.Display.RefreshMs, expected.Display.RefreshMs)
	}
	if cfg.Git.ShowDirty != expected.Git.ShowDirty {
		t.Errorf("Git.ShowDirty = %v, want %v", cfg.Git.ShowDirty, expected.Git.ShowDirty)
	}
	if cfg.Tmux.Size != expected.Tmux.Size {
		t.Errorf("Tmux.Size = %d, want %d", cfg.Tmux.Size, expected.Tmux.Size)
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	// Create a temporary TOML file with partial overrides
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	content := `
[display]
theme = "dark"
refresh_ms = 1000

[git]
show_ahead_behind = true
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	// Overridden values
	if cfg.Display.Theme != "dark" {
		t.Errorf("Display.Theme = %q, want %q", cfg.Display.Theme, "dark")
	}
	if cfg.Display.RefreshMs != 1000 {
		t.Errorf("Display.RefreshMs = %d, want %d", cfg.Display.RefreshMs, 1000)
	}
	if cfg.Git.ShowAheadBehind != true {
		t.Errorf("Git.ShowAheadBehind = %v, want %v", cfg.Git.ShowAheadBehind, true)
	}

	// Non-overridden values should remain at defaults
	if cfg.Display.ShowRateLimit != true {
		t.Errorf("Display.ShowRateLimit = %v, want %v", cfg.Display.ShowRateLimit, true)
	}
	if cfg.Display.ShowActivity != true {
		t.Errorf("Display.ShowActivity = %v, want %v", cfg.Display.ShowActivity, true)
	}
	if cfg.Display.ShowGit != true {
		t.Errorf("Display.ShowGit = %v, want %v", cfg.Display.ShowGit, true)
	}
	if cfg.Git.ShowDirty != true {
		t.Errorf("Git.ShowDirty = %v, want %v", cfg.Git.ShowDirty, true)
	}
	if cfg.Git.ShowFileStats != false {
		t.Errorf("Git.ShowFileStats = %v, want %v", cfg.Git.ShowFileStats, false)
	}
	if cfg.Tmux.AutoDetect != true {
		t.Errorf("Tmux.AutoDetect = %v, want %v", cfg.Tmux.AutoDetect, true)
	}
	if cfg.Tmux.Position != "bottom" {
		t.Errorf("Tmux.Position = %q, want %q", cfg.Tmux.Position, "bottom")
	}
	if cfg.Tmux.Size != 30 {
		t.Errorf("Tmux.Size = %d, want %d", cfg.Tmux.Size, 30)
	}
}

func TestLoadConfigInvalidTOML(t *testing.T) {
	// Create a temporary file with invalid TOML
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.toml")

	content := `
[display
theme = "dark"
this is not valid toml }{}{}{
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() returned error for invalid TOML: %v", err)
	}

	// Should return defaults when TOML is invalid
	expected := Default()
	if cfg.Display.Theme != expected.Display.Theme {
		t.Errorf("Display.Theme = %q, want %q", cfg.Display.Theme, expected.Display.Theme)
	}
	if cfg.Display.RefreshMs != expected.Display.RefreshMs {
		t.Errorf("Display.RefreshMs = %d, want %d", cfg.Display.RefreshMs, expected.Display.RefreshMs)
	}
	if cfg.Git.ShowDirty != expected.Git.ShowDirty {
		t.Errorf("Git.ShowDirty = %v, want %v", cfg.Git.ShowDirty, expected.Git.ShowDirty)
	}
	if cfg.Tmux.AutoDetect != expected.Tmux.AutoDetect {
		t.Errorf("Tmux.AutoDetect = %v, want %v", cfg.Tmux.AutoDetect, expected.Tmux.AutoDetect)
	}
	if cfg.Tmux.Position != expected.Tmux.Position {
		t.Errorf("Tmux.Position = %q, want %q", cfg.Tmux.Position, expected.Tmux.Position)
	}
	if cfg.Tmux.Size != expected.Tmux.Size {
		t.Errorf("Tmux.Size = %d, want %d", cfg.Tmux.Size, expected.Tmux.Size)
	}
}
