package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// launchFallback is used when no split-capable environment is found.
// On Windows, it opens a new console window for the HUD.
// On other platforms, it prints guidance and runs in watch mode.
func launchFallback(codexArgs []string, hudBinary string, goos string) error {
	if goos == "windows" {
		return launchWindowsFallback(codexArgs, hudBinary)
	}

	fmt.Println("┌─────────────────────────────────────────────────────┐")
	fmt.Println("│  codex-hud: split-pane environment not detected     │")
	fmt.Println("│                                                     │")
	switch goos {
	case "darwin":
		fmt.Println("│  Install tmux for the best experience:              │")
		fmt.Println("│    brew install tmux                                │")
	case "linux":
		fmt.Println("│  Install tmux for the best experience:              │")
		fmt.Println("│    sudo apt install tmux                            │")
	}
	fmt.Println("│                                                     │")
	fmt.Println("│  Starting HUD in watch mode...                      │")
	fmt.Println("│  Run 'codex' in another terminal to see stats.      │")
	fmt.Println("└─────────────────────────────────────────────────────┘")
	fmt.Println()

	cmd := exec.Command(hudBinary, "--watch")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// launchWindowsFallback opens the HUD in a new console window and runs codex
// in the current window. Works on any Windows terminal (CMD, PowerShell, Git Bash).
func launchWindowsFallback(codexArgs []string, hudBinary string) error {
	// "start" opens a new console window on Windows.
	// Syntax: cmd /c start "title" <command> <args...>
	// The first quoted string after "start" is treated as the window title.
	var startCmd *exec.Cmd

	if runtime.GOOS == "windows" {
		startCmd = exec.Command("cmd", "/c", "start", "codex-hud", hudBinary, "--watch", "--fresh")
	} else {
		// Shouldn't reach here, but just in case.
		startCmd = exec.Command(hudBinary, "--watch", "--fresh")
	}
	startCmd.Stdout = os.Stdout
	startCmd.Stderr = os.Stderr
	if err := startCmd.Run(); err != nil {
		// If new window fails, fall back to watch mode in current terminal.
		fmt.Println("Could not open new window. Starting HUD in current terminal.")
		fmt.Println("Run 'codex' in another terminal to see stats.")
		fmt.Println()
		cmd := exec.Command(hudBinary, "--watch")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	// Run codex in the current window.
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
