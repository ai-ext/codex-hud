// cmd/codex-hud/main.go
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ds/codex-hud/internal/config"
	"github.com/ds/codex-hud/internal/launcher"
	"github.com/ds/codex-hud/internal/tui"
	"github.com/ds/codex-hud/internal/watcher"
	"github.com/spf13/cobra"
)

var version = "0.4.0"

func main() {
	rootCmd := &cobra.Command{
		Use:   "codex-hud",
		Short: "Real-time HUD for OpenAI Codex CLI",
		Long: `Real-time HUD for OpenAI Codex CLI.

Default: launches codex + HUD together in a split pane.
Use --watch to run the HUD panel only (monitor an existing session).`,
		RunE: run,
	}

	rootCmd.Flags().String("file", "", "Path to a specific .jsonl session file")
	rootCmd.Flags().Bool("watch", false, "Watch mode: HUD panel only (run codex separately)")
	rootCmd.Flags().Bool("fresh", false, "Skip pre-loading old session data (used by wrapper mode)")
	rootCmd.Flags().String("split", "bottom", "Split direction: bottom or right")
	rootCmd.Flags().Int("size", 40, "HUD pane size in percent (default 40)")
	rootCmd.Version = version

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// run dispatches to watch mode or wrapper mode based on flags.
func run(cmd *cobra.Command, args []string) error {
	watch, _ := cmd.Flags().GetBool("watch")
	if watch {
		return runWatch(cmd, args)
	}
	// Default: wrapper mode (codex + HUD together)
	return runWrapper(cmd, args)
}

// runWrapper launches codex + HUD in a split pane.
func runWrapper(cmd *cobra.Command, args []string) error {
	split, _ := cmd.Flags().GetString("split")
	size, _ := cmd.Flags().GetInt("size")
	if size < 10 || size > 80 {
		size = 40
	}
	self, err := os.Executable()
	if err != nil {
		self = "codex-hud"
	}
	// Resolve symlinks and clean the path for cross-platform compatibility.
	if resolved, e := filepath.EvalSymlinks(self); e == nil {
		self = resolved
	}
	self = filepath.Clean(self)
	return launcher.Launch(args, split, size, self)
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

	fresh, _ := cmd.Flags().GetBool("fresh")
	filePath, _ := cmd.Flags().GetString("file")

	// In fresh mode (wrapper launching a new codex session), skip old data.
	if !fresh && filePath == "" {
		filePath, err = watcher.FindLatestSession(sessionsDir)
		if err != nil {
			filePath = "" // No session yet, will watch for new ones
		}
	}

	// --- Phase 1: Pre-load existing session data BEFORE starting TUI ---
	// This prevents the "jumpy" startup where lines are processed one by one.
	// Skipped in fresh mode since the old session is irrelevant.
	lines := make(chan string, 100)
	model := tui.NewModel(cfg, lines)

	if filePath != "" {
		existingLines, _ := watcher.ReadExistingLines(filePath)
		for _, line := range existingLines {
			tui.ProcessLine(model.State, line)
		}
		if len(existingLines) > 0 {
			model.Waiting = false
		}
	}

	// --- Phase 2: Start tailing for NEW lines only ---
	stop := make(chan struct{})
	defer close(stop)

	if filePath != "" {
		// Tail from end since we already pre-loaded existing content.
		go func() {
			watcher.TailFileFromEnd(filePath, lines, stop)
		}()
	}

	// Always watch for new sessions so the HUD automatically switches when
	// codex starts a new session (e.g. in wrapper mode where codex launches
	// slightly after the HUD). ApplySessionMeta resets per-session state
	// when the session ID changes.
	//
	// In fresh mode, pass the current time so the poll ignores old sessions.
	var minModTime time.Time
	if fresh {
		minModTime = time.Now()
	}
	go func() {
		watcher.WatchForNewSession(sessionsDir, lines, stop, minModTime)
	}()

	// --- Phase 3: Start TUI with pre-populated state ---
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}
