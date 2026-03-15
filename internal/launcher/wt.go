package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// launchWT handles Windows Terminal split-pane launch.
// When already inside WT (WT_SESSION set), it splits the current tab.
// When outside WT, it opens a new WT window with codex + HUD pre-configured.
func launchWT(codexArgs []string, split string, sizePercent int, hudBinary string) error {
	if os.Getenv("WT_SESSION") != "" {
		return launchWTSplit(codexArgs, split, sizePercent, hudBinary)
	}
	return launchWTNew(codexArgs, split, sizePercent, hudBinary)
}

// wtDirFlag returns the WT split direction flag.
// WT flags are opposite of tmux:
//
//	-H = horizontal split = pane below (top/bottom)
//	-V = vertical split   = pane to right (side by side)
func wtDirFlag(split string) string {
	if split == "right" {
		return "-V"
	}
	return "-H"
}

// launchWTSplit splits the current Windows Terminal tab (when already inside WT).
// Uses "-w 0" to explicitly target the current window.
func launchWTSplit(codexArgs []string, split string, sizePercent int, hudBinary string) error {
	dirFlag := wtDirFlag(split)
	sizeStr := fmt.Sprintf("%.2f", float64(sizePercent)/100.0)

	// -w 0 = target the current (most recent) window, not a new one.
	wtArgs := []string{
		"-w", "0",
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

// launchWTNew opens a new Windows Terminal window with codex + HUD already
// split. Used when codex-hud is run from outside WT (e.g. plain PowerShell,
// Git Bash, CMD).
// Builds the full command line as a string and runs via cmd /c so that the
// ";" subcommand separator is not escaped by Go's exec.Command.
func launchWTNew(codexArgs []string, split string, sizePercent int, hudBinary string) error {
	dirFlag := wtDirFlag(split)
	sizeStr := fmt.Sprintf("%.2f", float64(sizePercent)/100.0)

	codexBin, codexCmdArgs := buildCodexCommand(codexArgs)
	codexFullCmd := codexBin
	if len(codexCmdArgs) > 0 {
		codexFullCmd += " " + strings.Join(codexCmdArgs, " ")
	}

	// Build the full wt command line as a single string.
	// cmd /c ensures the ";" separator is passed literally to wt.exe.
	cmdLine := fmt.Sprintf("wt new-tab %s ; split-pane %s --size %s %s --watch --fresh",
		codexFullCmd, dirFlag, sizeStr, hudBinary)

	fmt.Println("Launching codex + HUD in Windows Terminal...")

	cmd := exec.Command("cmd", "/c", cmdLine)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
