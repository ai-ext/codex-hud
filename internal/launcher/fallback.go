package launcher

import (
	"fmt"
	"os"
	"os/exec"
)

// launchFallback is the last resort when no split-capable environment is found
// and tmux is not installed. It prints guidance and starts the HUD in watch
// mode so the user at least sees something useful.
func launchFallback(codexArgs []string, hudBinary string, goos string) error {
	fmt.Println("┌─────────────────────────────────────────────────────┐")
	fmt.Println("│  codex-hud: split-pane environment not detected     │")
	fmt.Println("│                                                     │")
	fmt.Println("│  Install tmux for the best experience:              │")
	switch goos {
	case "darwin":
		fmt.Println("│    brew install tmux                                │")
	case "linux":
		fmt.Println("│    sudo apt install tmux                            │")
	default:
		fmt.Println("│    (install tmux for your platform)                 │")
	}
	fmt.Println("│                                                     │")
	fmt.Println("│  Starting HUD in watch mode...                      │")
	fmt.Println("│  Run 'codex' in another terminal tab to see stats.  │")
	fmt.Println("└─────────────────────────────────────────────────────┘")
	fmt.Println()

	// Fall back to running the HUD in watch mode in the current terminal.
	cmd := exec.Command(hudBinary, "--watch")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
