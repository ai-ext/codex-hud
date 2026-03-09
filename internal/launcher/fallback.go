package launcher

import (
	"fmt"
	"os"
	"os/exec"
)

// launchFallback opens a new terminal window for the HUD when neither tmux nor
// Windows Terminal is detected. It uses OS-specific mechanisms.
func launchFallback(codexArgs []string, hudBinary string, goos string) error {
	hudCmd := hudWatchCommand(hudBinary)

	var err error
	switch goos {
	case "darwin":
		err = launchDarwin(hudCmd)
	case "linux":
		err = launchLinux(hudCmd)
	case "windows":
		err = launchWindows(hudCmd)
	default:
		return fmt.Errorf("unsupported OS for fallback launcher: %s", goos)
	}
	if err != nil {
		return fmt.Errorf("failed to open HUD window: %w", err)
	}

	// Run codex in the current terminal.
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

// launchDarwin opens macOS Terminal.app with the HUD command.
func launchDarwin(hudCmd string) error {
	script := fmt.Sprintf(`tell application "Terminal" to do script "%s"`, hudCmd)
	return exec.Command("osascript", "-e", script).Run()
}

// launchLinux tries common terminal emulators in order of preference.
func launchLinux(hudCmd string) error {
	terminals := []struct {
		bin  string
		args []string
	}{
		{"x-terminal-emulator", []string{"-e", hudCmd}},
		{"gnome-terminal", []string{"--", "sh", "-c", hudCmd}},
		{"xterm", []string{"-e", hudCmd}},
	}

	for _, t := range terminals {
		path, err := exec.LookPath(t.bin)
		if err != nil {
			continue
		}
		cmd := exec.Command(path, t.args...)
		if err := cmd.Start(); err != nil {
			continue
		}
		return nil
	}
	return fmt.Errorf("no suitable terminal emulator found (tried x-terminal-emulator, gnome-terminal, xterm)")
}

// launchWindows opens a new cmd.exe window with the HUD command.
func launchWindows(hudCmd string) error {
	return exec.Command("cmd", "/c", "start", "cmd", "/k", hudCmd).Run()
}
