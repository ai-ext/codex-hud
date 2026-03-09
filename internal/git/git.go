// Package git provides lightweight helpers for reading git repository state.
package git

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Status represents the current state of a git working tree.
type Status struct {
	Branch    string
	Dirty     bool
	Modified  int
	Added     int
	Deleted   int
	Untracked int
	Ahead     int
	Behind    int
}

// runGit executes a git command in dir with a 1-second timeout and returns
// its trimmed stdout. It returns an error if the command fails or times out.
func runGit(dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

// GetStatus returns the current status of the git repository at dir.
// It returns an error if dir is not inside a git repository.
func GetStatus(dir string) (*Status, error) {
	// Verify this is a git repo and get the branch name.
	branch, err := runGit(dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("not a git repository (or git not installed): %w", err)
	}

	// Get porcelain status for file-level stats.
	porcelain, err := runGit(dir, "--no-optional-locks", "status", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("git status: %w", err)
	}

	s := &Status{Branch: branch}

	if porcelain == "" {
		return s, nil
	}

	for _, line := range strings.Split(porcelain, "\n") {
		if len(line) < 2 {
			continue
		}

		// Porcelain v1 format: XY filename
		// X = index status, Y = working-tree status.
		x := line[0]
		y := line[1]

		switch {
		case x == '?' && y == '?':
			s.Untracked++
		default:
			// Check both index (x) and working-tree (y) columns.
			if x == 'M' || y == 'M' {
				s.Modified++
			}
			if x == 'A' {
				s.Added++
			}
			if x == 'D' || y == 'D' {
				s.Deleted++
			}
		}
	}

	s.Dirty = s.Modified > 0 || s.Added > 0 || s.Deleted > 0 || s.Untracked > 0

	return s, nil
}
