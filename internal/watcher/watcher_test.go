package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFindLatestSession(t *testing.T) {
	dir := t.TempDir()

	// Create two .jsonl files with different modification times.
	older := filepath.Join(dir, "older.jsonl")
	newer := filepath.Join(dir, "newer.jsonl")

	if err := os.WriteFile(older, []byte(`{"msg":"old"}`+"\n"), 0644); err != nil {
		t.Fatalf("write older: %v", err)
	}

	// Ensure the newer file has a strictly later mtime.
	pastTime := time.Now().Add(-10 * time.Second)
	if err := os.Chtimes(older, pastTime, pastTime); err != nil {
		t.Fatalf("chtimes older: %v", err)
	}

	if err := os.WriteFile(newer, []byte(`{"msg":"new"}`+"\n"), 0644); err != nil {
		t.Fatalf("write newer: %v", err)
	}

	got, err := FindLatestSession(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != newer {
		t.Errorf("expected %s, got %s", newer, got)
	}
}

func TestFindLatestSessionNested(t *testing.T) {
	dir := t.TempDir()

	// Create a nested structure mimicking Codex date-based dirs.
	subdir := filepath.Join(dir, "2026-03-09")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	older := filepath.Join(dir, "root.jsonl")
	nested := filepath.Join(subdir, "session.jsonl")

	if err := os.WriteFile(older, []byte(`{"msg":"old"}`+"\n"), 0644); err != nil {
		t.Fatalf("write older: %v", err)
	}
	pastTime := time.Now().Add(-10 * time.Second)
	if err := os.Chtimes(older, pastTime, pastTime); err != nil {
		t.Fatalf("chtimes older: %v", err)
	}

	if err := os.WriteFile(nested, []byte(`{"msg":"nested"}`+"\n"), 0644); err != nil {
		t.Fatalf("write nested: %v", err)
	}

	got, err := FindLatestSession(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != nested {
		t.Errorf("expected %s, got %s", nested, got)
	}
}

func TestFindLatestSessionEmpty(t *testing.T) {
	dir := t.TempDir()

	_, err := FindLatestSession(dir)
	if err == nil {
		t.Fatal("expected error for empty directory, got nil")
	}
}

func TestWatchForNewSession_ResumeDetection(t *testing.T) {
	dir := t.TempDir()
	lines := make(chan string, 100)
	stop := make(chan struct{})

	// Start watching with no minModTime (like --watch mode without --fresh).
	go WatchForNewSession(dir, lines, stop, time.Time{})

	// Wait for the watcher to start.
	time.Sleep(500 * time.Millisecond)

	// Create first session file (simulates codex starting).
	fileA := filepath.Join(dir, "session_a.jsonl")
	if err := os.WriteFile(fileA, []byte(`{"type":"session_meta","payload":{"id":"a"}}`+"\n"), 0644); err != nil {
		t.Fatalf("write session A: %v", err)
	}

	// Should receive session A's line.
	select {
	case line := <-lines:
		if line != `{"type":"session_meta","payload":{"id":"a"}}` {
			t.Errorf("session A: unexpected line: %s", line)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for session A")
	}

	// Wait a moment, then create second session file (simulates /resume).
	time.Sleep(500 * time.Millisecond)
	// Ensure fileB has a strictly newer mtime.
	pastTime := time.Now().Add(-5 * time.Second)
	os.Chtimes(fileA, pastTime, pastTime)

	fileB := filepath.Join(dir, "session_b.jsonl")
	if err := os.WriteFile(fileB, []byte(`{"type":"session_meta","payload":{"id":"b"}}`+"\n"), 0644); err != nil {
		t.Fatalf("write session B: %v", err)
	}

	// Should receive session B's line (the /resume session).
	select {
	case line := <-lines:
		if line != `{"type":"session_meta","payload":{"id":"b"}}` {
			t.Errorf("session B: unexpected line: %s", line)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for session B (/resume detection failed)")
	}

	close(stop)
}

func TestWatchForNewSession_FreshModeResume(t *testing.T) {
	dir := t.TempDir()
	lines := make(chan string, 100)
	stop := make(chan struct{})

	// Create an old session file that should be ignored in fresh mode.
	oldFile := filepath.Join(dir, "old_session.jsonl")
	os.WriteFile(oldFile, []byte(`{"type":"session_meta","payload":{"id":"old"}}`+"\n"), 0644)
	pastTime := time.Now().Add(-10 * time.Second)
	os.Chtimes(oldFile, pastTime, pastTime)

	// Start watching in fresh mode (minModTime = now).
	minModTime := time.Now()
	go WatchForNewSession(dir, lines, stop, minModTime)

	// Wait for the watcher to start.
	time.Sleep(500 * time.Millisecond)

	// The old session should NOT be detected.
	select {
	case line := <-lines:
		t.Fatalf("should not receive old session, got: %s", line)
	case <-time.After(1 * time.Second):
		// Good — no old session detected.
	}

	// Create a new session file (simulates codex starting after HUD).
	fileA := filepath.Join(dir, "new_session.jsonl")
	os.WriteFile(fileA, []byte(`{"type":"session_meta","payload":{"id":"new"}}`+"\n"), 0644)

	select {
	case line := <-lines:
		if line != `{"type":"session_meta","payload":{"id":"new"}}` {
			t.Errorf("unexpected line: %s", line)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for new session in fresh mode")
	}

	// Now simulate /resume: create another file after the first.
	time.Sleep(500 * time.Millisecond)
	pastTime2 := time.Now().Add(-3 * time.Second)
	os.Chtimes(fileA, pastTime2, pastTime2)

	fileB := filepath.Join(dir, "resume_session.jsonl")
	os.WriteFile(fileB, []byte(`{"type":"session_meta","payload":{"id":"resume"}}`+"\n"), 0644)

	select {
	case line := <-lines:
		if line != `{"type":"session_meta","payload":{"id":"resume"}}` {
			t.Errorf("resume: unexpected line: %s", line)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for resume session in fresh mode")
	}

	close(stop)
}

func TestWatchForNewSession_NestedDirResume(t *testing.T) {
	// Simulate the real Codex sessions directory structure: sessions/YYYY/MM/DD/
	dir := t.TempDir()
	subdir := filepath.Join(dir, "2026", "03", "15")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	lines := make(chan string, 100)
	stop := make(chan struct{})

	// Start in fresh mode.
	minModTime := time.Now()
	go WatchForNewSession(dir, lines, stop, minModTime)

	// Wait for the watcher to start and register subdirectories.
	time.Sleep(500 * time.Millisecond)

	// Create first session file in the nested directory.
	fileA := filepath.Join(subdir, "rollout-session-a.jsonl")
	os.WriteFile(fileA, []byte(`{"type":"session_meta","payload":{"id":"a"}}`+"\n"), 0644)

	select {
	case line := <-lines:
		if line != `{"type":"session_meta","payload":{"id":"a"}}` {
			t.Errorf("session A: unexpected line: %s", line)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for session A in nested dir")
	}

	// Simulate /resume: make fileA old, create fileB.
	time.Sleep(500 * time.Millisecond)
	pastTime := time.Now().Add(-5 * time.Second)
	os.Chtimes(fileA, pastTime, pastTime)

	fileB := filepath.Join(subdir, "rollout-session-b.jsonl")
	os.WriteFile(fileB, []byte(
		`{"type":"session_meta","payload":{"id":"b"}}`+"\n"+
			`{"type":"turn_context","payload":{"model":"o4-mini"}}`+"\n",
	), 0644)

	// Should receive both lines from the /resume session.
	received := 0
	timeout := time.After(5 * time.Second)
	for received < 2 {
		select {
		case <-lines:
			received++
		case <-timeout:
			t.Fatalf("timed out waiting for session B lines, got %d/2", received)
		}
	}

	close(stop)
}

func TestTailFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")

	// Write 2 initial lines.
	if err := os.WriteFile(path, []byte(`{"line":1}`+"\n"+`{"line":2}`+"\n"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	lines := make(chan string, 100)
	stop := make(chan struct{})
	errCh := make(chan error, 1)

	go func() {
		errCh <- TailFile(path, lines, stop)
	}()

	// Collect the 2 existing lines with a timeout.
	timeout := time.After(5 * time.Second)
	var received []string
	for i := 0; i < 2; i++ {
		select {
		case line := <-lines:
			received = append(received, line)
		case <-timeout:
			t.Fatalf("timed out waiting for initial lines, got %d", len(received))
		}
	}

	if len(received) != 2 {
		t.Fatalf("expected 2 initial lines, got %d", len(received))
	}
	if received[0] != `{"line":1}` {
		t.Errorf("line 0: expected {\"line\":1}, got %s", received[0])
	}
	if received[1] != `{"line":2}` {
		t.Errorf("line 1: expected {\"line\":2}, got %s", received[1])
	}

	// Give the watcher a moment to register with the OS before appending.
	time.Sleep(200 * time.Millisecond)

	// Append a 3rd line.
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("open for append: %v", err)
	}
	if _, err := f.WriteString(`{"line":3}` + "\n"); err != nil {
		f.Close()
		t.Fatalf("append: %v", err)
	}
	f.Close()

	// Read the 3rd line.
	timeout = time.After(5 * time.Second)
	select {
	case line := <-lines:
		if line != `{"line":3}` {
			t.Errorf("line 2: expected {\"line\":3}, got %s", line)
		}
	case <-timeout:
		t.Fatal("timed out waiting for appended line")
	}

	// Stop tailing.
	close(stop)

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("TailFile returned unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("TailFile did not return after stop")
	}
}
