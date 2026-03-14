package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// launchTmux splits the current tmux pane and runs the HUD in the new pane
// while executing codex in the original pane.
func launchTmux(codexArgs []string, split string, sizePercent int, hudBinary string) error {
	var dirFlag string
	switch split {
	case "right":
		dirFlag = "-h"
	default:
		dirFlag = "-v"
	}

	hudCmd := hudWatchCommand(hudBinary)
	sizeStr := fmt.Sprintf("%d%%", sizePercent)
	tmuxArgs := []string{
		"split-window",
		dirFlag,
		"-l", sizeStr,
		hudCmd,
	}

	splitCmd := exec.Command("tmux", tmuxArgs...)
	splitCmd.Stdout = os.Stdout
	splitCmd.Stderr = os.Stderr
	if err := splitCmd.Run(); err != nil {
		return fmt.Errorf("tmux split-window failed: %w", err)
	}

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
// pane and the HUD in a split pane.
func launchNewTmuxSession(codexArgs []string, split string, sizePercent int, hudBinary string, tmuxPath string) error {
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

	sizeStr := fmt.Sprintf("%d%%", sizePercent)
	args := []string{
		"new-session",
		codexFullCmd,
		";",
		"split-window", dirFlag, "-l", sizeStr,
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
