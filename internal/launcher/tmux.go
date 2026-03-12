package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// launchTmux splits the current tmux pane and runs the HUD in the new pane
// while executing codex in the original pane.
func launchTmux(codexArgs []string, split string, hudBinary string) error {
	// Determine split direction flag.
	var dirFlag string
	switch split {
	case "right":
		dirFlag = "-h"
	default: // "bottom" or anything else defaults to vertical split
		dirFlag = "-v"
	}

	// Split the tmux window and run the HUD in the new pane.
	hudCmd := hudWatchCommand(hudBinary)
	tmuxArgs := []string{
		"split-window",
		dirFlag,
		"-l", "30%",
		hudCmd,
	}

	splitCmd := exec.Command("tmux", tmuxArgs...)
	splitCmd.Stdout = os.Stdout
	splitCmd.Stderr = os.Stderr
	if err := splitCmd.Run(); err != nil {
		return fmt.Errorf("tmux split-window failed: %w", err)
	}

	// Run codex in the original (current) pane.
	codexBin, codexCmdArgs := buildCodexCommand(codexArgs)
	c := exec.Command(codexBin, codexCmdArgs...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("codex exited with error: %w", err)
	}
	return nil
}

// launchNewTmuxSession creates a brand-new tmux session with codex in the main
// pane and the HUD in a split pane. Used when tmux is installed but the user
// is not already inside a tmux session.
func launchNewTmuxSession(codexArgs []string, split string, hudBinary string, tmuxPath string) error {
	var dirFlag string
	switch split {
	case "right":
		dirFlag = "-h"
	default:
		dirFlag = "-v"
	}

	hudCmd := hudWatchCommand(hudBinary)
	codexBin, codexCmdArgs := buildCodexCommand(codexArgs)
	codexFullCmd := codexBin
	if len(codexCmdArgs) > 0 {
		codexFullCmd += " " + strings.Join(codexCmdArgs, " ")
	}

	// tmux new-session <codex> \; split-window <hud> \; select-pane -t 0
	args := []string{
		"new-session",
		codexFullCmd,
		";",
		"split-window", dirFlag, "-l", "30%",
		hudCmd,
		";",
		"select-pane", "-t", "0",
	}

	cmd := exec.Command(tmuxPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
