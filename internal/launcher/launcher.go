// Package launcher detects the terminal environment and launches codex + HUD
// in a split-pane configuration appropriate for that environment.
package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Environment represents the detected terminal environment.
type Environment int

const (
	// EnvTmux indicates the session is running inside tmux.
	EnvTmux Environment = iota
	// EnvWindowsTerminal indicates the session is running inside Windows Terminal.
	EnvWindowsTerminal
	// EnvGeneric indicates an unrecognized terminal.
	EnvGeneric
)

// String returns a human-readable name for the environment.
func (e Environment) String() string {
	switch e {
	case EnvTmux:
		return "tmux"
	case EnvWindowsTerminal:
		return "Windows Terminal"
	default:
		return "generic"
	}
}

// Detect inspects environment variables to determine the current terminal.
func Detect() Environment {
	if os.Getenv("TMUX") != "" {
		return EnvTmux
	}
	// WT_SESSION is set inside Windows Terminal tabs/panes.
	if os.Getenv("WT_SESSION") != "" {
		return EnvWindowsTerminal
	}
	// On Windows, check if wt.exe is available even without WT_SESSION
	// (e.g., running from Git Bash or PowerShell inside Windows Terminal,
	// or Windows Terminal is installed but env var not propagated).
	if runtime.GOOS == "windows" {
		if _, err := exec.LookPath("wt"); err == nil {
			return EnvWindowsTerminal
		}
	}
	return EnvGeneric
}

// Launch opens a split pane (or new window) running the HUD alongside codex.
func Launch(codexArgs []string, split string, sizePercent int, hudBinary string) error {
	env := Detect()
	switch env {
	case EnvTmux:
		return launchTmux(codexArgs, split, sizePercent, hudBinary)
	case EnvWindowsTerminal:
		return launchWT(codexArgs, split, sizePercent, hudBinary)
	default:
		// On non-Windows, try tmux.
		if runtime.GOOS != "windows" {
			if tmuxPath, err := exec.LookPath("tmux"); err == nil {
				return launchNewTmuxSession(codexArgs, split, sizePercent, hudBinary, tmuxPath)
			}
		}
		return launchFallback(codexArgs, hudBinary, runtime.GOOS)
	}
}

// buildCodexCommand returns the codex binary name and its argument list.
func buildCodexCommand(codexArgs []string) (string, []string) {
	bin := "codex"
	if runtime.GOOS == "windows" {
		// Try codex.cmd first (npm global install), then codex.exe.
		if _, err := exec.LookPath("codex.cmd"); err == nil {
			bin = "codex.cmd"
		}
	}
	args := make([]string, len(codexArgs))
	copy(args, codexArgs)
	return bin, args
}

// hudWatchCommand returns the full command string to run the HUD in watch mode.
func hudWatchCommand(hudBinary string) string {
	// Quote the path if it contains spaces (common on Windows).
	if strings.Contains(hudBinary, " ") {
		return fmt.Sprintf("\"%s\" --watch --fresh", hudBinary)
	}
	return fmt.Sprintf("%s --watch --fresh", hudBinary)
}
