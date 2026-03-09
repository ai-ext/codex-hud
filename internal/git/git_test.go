package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// initTestRepo creates a temporary git repo with an initial commit and returns its path.
func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "init"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("setup %v failed: %v\n%s", args, err, out)
		}
	}
	return dir
}

func TestGetStatusClean(t *testing.T) {
	dir := initTestRepo(t)

	s, err := GetStatus(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if s.Branch == "" {
		t.Error("expected non-empty branch name")
	}
	if s.Dirty {
		t.Error("expected clean repo, got dirty")
	}
	if s.Modified != 0 {
		t.Errorf("expected Modified=0, got %d", s.Modified)
	}
	if s.Added != 0 {
		t.Errorf("expected Added=0, got %d", s.Added)
	}
	if s.Deleted != 0 {
		t.Errorf("expected Deleted=0, got %d", s.Deleted)
	}
	if s.Untracked != 0 {
		t.Errorf("expected Untracked=0, got %d", s.Untracked)
	}
	if s.Ahead != 0 {
		t.Errorf("expected Ahead=0, got %d", s.Ahead)
	}
	if s.Behind != 0 {
		t.Errorf("expected Behind=0, got %d", s.Behind)
	}
}

func TestGetStatusDirty(t *testing.T) {
	dir := initTestRepo(t)

	// Create an untracked file.
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("hello"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	// Create a tracked file, commit it, then modify it.
	tracked := filepath.Join(dir, "tracked.txt")
	if err := os.WriteFile(tracked, []byte("original"), 0644); err != nil {
		t.Fatalf("write tracked: %v", err)
	}
	for _, args := range [][]string{
		{"git", "add", "tracked.txt"},
		{"git", "commit", "-m", "add tracked"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("setup %v: %v\n%s", args, err, out)
		}
	}
	// Modify the tracked file (working-tree change).
	if err := os.WriteFile(tracked, []byte("modified"), 0644); err != nil {
		t.Fatalf("modify tracked: %v", err)
	}

	s, err := GetStatus(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !s.Dirty {
		t.Error("expected dirty repo")
	}
	if s.Untracked < 1 {
		t.Errorf("expected Untracked>=1, got %d", s.Untracked)
	}
	if s.Modified < 1 {
		t.Errorf("expected Modified>=1, got %d", s.Modified)
	}
}

func TestGetStatusNotGitRepo(t *testing.T) {
	dir := t.TempDir() // plain directory, no git init

	_, err := GetStatus(dir)
	if err == nil {
		t.Fatal("expected error for non-git directory, got nil")
	}
}
