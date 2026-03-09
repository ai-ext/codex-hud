package launcher

import (
	"fmt"
	"os"
	"os/exec"
)

// launchWT splits the current Windows Terminal tab and runs the HUD in the new
// pane while executing codex in the original pane.
func launchWT(codexArgs []string, split string, hudBinary string) error {
	// Determine split direction flag.
	// Windows Terminal uses -V for vertical (top/bottom) and -H for horizontal (left/right).
	var dirFlag string
	switch split {
	case "right":
		dirFlag = "-H"
	default: // "bottom"
		dirFlag = "-V"
	}

	hudCmd := hudWatchCommand(hudBinary)
	wtArgs := []string{
		"split-pane",
		dirFlag,
		"--size", "0.3",
		hudCmd,
	}

	splitCmd := exec.Command("wt", wtArgs...)
	splitCmd.Stdout = os.Stdout
	splitCmd.Stderr = os.Stderr
	if err := splitCmd.Run(); err != nil {
		return fmt.Errorf("wt split-pane failed: %w", err)
	}

	// Run codex in the current pane.
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
