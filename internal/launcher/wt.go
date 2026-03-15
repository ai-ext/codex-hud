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

// launchWTSplit splits the current Windows Terminal tab (when already inside WT).
func launchWTSplit(codexArgs []string, split string, sizePercent int, hudBinary string) error {
	// WT flags are opposite of tmux:
	//   -H = horizontal split = pane below (top/bottom)
	//   -V = vertical split   = pane to right (side by side)
	var dirFlag string
	switch split {
	case "right":
		dirFlag = "-V"
	default:
		dirFlag = "-H"
	}

	sizeStr := fmt.Sprintf("%.2f", float64(sizePercent)/100.0)

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

// launchWTNew opens a new Windows Terminal window with codex + HUD already
// split. Used when codex-hud is run from outside WT (e.g. plain PowerShell,
// Git Bash, CMD).
func launchWTNew(codexArgs []string, split string, sizePercent int, hudBinary string) error {
	var dirFlag string
	switch split {
	case "right":
		dirFlag = "-V"
	default:
		dirFlag = "-H"
	}

	sizeStr := fmt.Sprintf("%.2f", float64(sizePercent)/100.0)

	codexBin, codexCmdArgs := buildCodexCommand(codexArgs)
	codexFullCmd := codexBin
	if len(codexCmdArgs) > 0 {
		codexFullCmd += " " + strings.Join(codexCmdArgs, " ")
	}

	// wt new-tab <codex> ; split-pane <flags> <hud>
	// The ";" separator tells wt to chain subcommands in one window.
	wtArgs := []string{
		"new-tab", codexFullCmd,
		";",
		"split-pane", dirFlag, "--size", sizeStr,
		hudBinary, "--watch", "--fresh",
	}

	fmt.Println("Launching codex + HUD in Windows Terminal...")

	cmd := exec.Command("wt", wtArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
