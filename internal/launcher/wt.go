package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// launchWT opens a Windows Terminal window with codex in the top pane and
// the HUD in the bottom pane (or left/right when split=right).
// Always creates a new WT tab with both panes pre-configured so that codex
// and HUD are guaranteed to be in the same window.
func launchWT(codexArgs []string, split string, sizePercent int, hudBinary string) error {
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

	// wt new-tab <codex> ; split-pane <dir> --size <pct> <hud> --watch --fresh
	// Uses cmd /c so the ";" subcommand separator is passed literally to wt.exe.
	cmdLine := fmt.Sprintf("wt new-tab %s ; split-pane %s --size %s %s --watch --fresh",
		codexFullCmd, dirFlag, sizeStr, hudBinary)

	fmt.Println("Launching codex + HUD in Windows Terminal...")

	cmd := exec.Command("cmd", "/c", cmdLine)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
