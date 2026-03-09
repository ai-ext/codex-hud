// Package launcher detects the terminal environment and launches codex + HUD
// in a split-pane configuration appropriate for that environment.
package launcher

import (
	"fmt"
	"os"
	"runtime"
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
	if os.Getenv("WT_SESSION") != "" {
		return EnvWindowsTerminal
	}
	return EnvGeneric
}

// Launch opens a split pane (or new window) running the HUD alongside codex.
//
// codexArgs are the arguments forwarded to the codex CLI.
// split is the pane direction ("bottom" or "right").
// hudBinary is the path to the codex-hud executable.
func Launch(codexArgs []string, split string, hudBinary string) error {
	env := Detect()
	switch env {
	case EnvTmux:
		return launchTmux(codexArgs, split, hudBinary)
	case EnvWindowsTerminal:
		return launchWT(codexArgs, split, hudBinary)
	default:
		return launchFallback(codexArgs, hudBinary, runtime.GOOS)
	}
}

// buildCodexCommand returns the codex binary name and its argument list.
func buildCodexCommand(codexArgs []string) (string, []string) {
	bin := "codex"
	args := make([]string, len(codexArgs))
	copy(args, codexArgs)
	return bin, args
}

// hudWatchCommand returns the full command string to run the HUD in watch mode.
func hudWatchCommand(hudBinary string) string {
	return fmt.Sprintf("%s --watch", hudBinary)
}
