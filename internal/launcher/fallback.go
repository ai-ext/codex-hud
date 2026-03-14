package launcher

import (
	"fmt"
	"os"
	"os/exec"
)

// launchFallback is used when no split-capable environment is found.
func launchFallback(codexArgs []string, hudBinary string, goos string) error {
	if goos == "windows" {
		return launchWindowsFallback(hudBinary)
	}

	fmt.Println("┌─────────────────────────────────────────────────────┐")
	fmt.Println("│  codex-hud: split-pane environment not detected     │")
	fmt.Println("│                                                     │")
	switch goos {
	case "darwin":
		fmt.Println("│  Install tmux for split-pane:                       │")
		fmt.Println("│    brew install tmux                                │")
	case "linux":
		fmt.Println("│  Install tmux for split-pane:                       │")
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

// launchWindowsFallback tells the user to install Windows Terminal.
// Split-pane on Windows requires Windows Terminal — no alternative fallback.
func launchWindowsFallback(hudBinary string) error {
	fmt.Println()
	fmt.Println("  codex-hud requires Windows Terminal for split-pane display.")
	fmt.Println()

	// Try auto-install via winget
	if _, err := exec.LookPath("winget"); err == nil {
		fmt.Println("  Windows Terminal is not installed.")
		fmt.Print("  Install now via winget? [Y/n]: ")

		var answer string
		fmt.Scanln(&answer)
		if answer == "" || answer == "y" || answer == "Y" {
			fmt.Println()
			fmt.Println("  Installing Windows Terminal...")
			installCmd := exec.Command("winget", "install",
				"--id", "Microsoft.WindowsTerminal",
				"--accept-source-agreements",
				"--accept-package-agreements",
			)
			installCmd.Stdout = os.Stdout
			installCmd.Stderr = os.Stderr
			if err := installCmd.Run(); err != nil {
				fmt.Println()
				fmt.Println("  Installation failed. Please install manually:")
				fmt.Println("    https://aka.ms/terminal")
				return fmt.Errorf("Windows Terminal installation failed: %w", err)
			}
			fmt.Println()
			fmt.Println("  Windows Terminal installed successfully!")
			fmt.Println("  Please restart your terminal, then run 'codex-hud' again.")
			return nil
		}
	}

	fmt.Println("  Please install Windows Terminal:")
	fmt.Println("    winget install Microsoft.WindowsTerminal")
	fmt.Println("    or: https://aka.ms/terminal")
	fmt.Println()
	fmt.Println("  After installing, restart your terminal and run 'codex-hud' again.")
	return fmt.Errorf("Windows Terminal required for split-pane mode")
}
