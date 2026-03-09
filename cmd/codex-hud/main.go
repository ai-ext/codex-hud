// cmd/codex-hud/main.go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ds/codex-hud/internal/config"
	"github.com/ds/codex-hud/internal/tui"
	"github.com/ds/codex-hud/internal/watcher"
	"github.com/spf13/cobra"
)

var version = "0.1.0"

func main() {
	rootCmd := &cobra.Command{
		Use:   "codex-hud",
		Short: "Real-time HUD for OpenAI Codex CLI",
		RunE:  runWatch,
	}

	rootCmd.Flags().String("file", "", "Path to a specific .jsonl session file")
	rootCmd.Flags().Bool("watch", false, "Watch mode: monitor existing session")
	rootCmd.Flags().String("split", "bottom", "Split direction: bottom or right")
	rootCmd.Version = version

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runWatch(cmd *cobra.Command, args []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot find home directory: %w", err)
	}

	configPath := filepath.Join(home, ".codex", "codex-hud.toml")
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	sessionsDir := filepath.Join(home, ".codex", "sessions")

	filePath, _ := cmd.Flags().GetString("file")
	if filePath == "" {
		filePath, err = watcher.FindLatestSession(sessionsDir)
		if err != nil {
			filePath = "" // No session yet, will watch for new ones
		}
	}

	lines := make(chan string, 100)
	stop := make(chan struct{})
	defer close(stop)

	if filePath != "" {
		go func() {
			watcher.TailFile(filePath, lines, stop)
		}()
	} else {
		go func() {
			watcher.WatchForNewSession(sessionsDir, lines, stop)
		}()
	}

	model := tui.NewModel(cfg, lines)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}
