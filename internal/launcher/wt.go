package launcher

import (
	"fmt"
	"os"
	"os/exec"
)

// launchWT splits the current Windows Terminal tab and runs the HUD in the new
// pane while executing codex in the original pane.
func launchWT(codexArgs []string, split string, sizePercent int, hudBinary string) error {
	var dirFlag string
	switch split {
	case "right":
		dirFlag = "-H"
	default:
		dirFlag = "-V"
	}

	sizeStr := fmt.Sprintf("%.2f", float64(sizePercent)/100.0)

	// Pass the HUD binary and its flags as separate arguments to wt.
	// wt split-pane expects: wt split-pane [options] <command> [args...]
	wtArgs := []string{
		"split-pane",
		dirFlag,
		"--size", sizeStr,
		hudBinary, "--watch", "--fresh",
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
