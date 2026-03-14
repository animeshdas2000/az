package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ── claude loader ─────────────────────────────────────────────────────────────

func TestClaudeLoaderWithSessionsIndex(t *testing.T) {
	root := t.TempDir()

	// Project with sessions-index.json
	proj := filepath.Join(root, "-Users-test-myproject")
	os.MkdirAll(proj, 0755)
	writeJSON(t, filepath.Join(proj, "sessions-index.json"), map[string]any{
		"originalPath": "/Users/test/myproject",
	})
	writeFile(t, filepath.Join(proj, "session1.jsonl"), "data")

	loader := &claudeLoader{projectsDir: root}
	sessions, err := loader.load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("want 1 session, got %d", len(sessions))
	}
	if sessions[0].Path != "/Users/test/myproject" {
		t.Errorf("path: got %q", sessions[0].Path)
	}
}

func TestClaudeLoaderFallbackSlugDecode(t *testing.T) {
	root := t.TempDir()

	// Project without sessions-index.json — slug decode
	proj := filepath.Join(root, "-tmp-workspace")
	os.MkdirAll(proj, 0755)
	os.MkdirAll("/tmp/workspace", 0755) // ensure path exists for greedy match
	writeFile(t, filepath.Join(proj, "s.jsonl"), "x")

	loader := &claudeLoader{projectsDir: root}
	sessions, err := loader.load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("want 1 session, got %d", len(sessions))
	}
	// Path should be /tmp/workspace (greedy match found it)
	if sessions[0].Path != "/tmp/workspace" {
		t.Errorf("path: got %q, want /tmp/workspace", sessions[0].Path)
	}
}

func TestClaudeLoaderMtimeFromJsonl(t *testing.T) {
	root := t.TempDir()
	proj := filepath.Join(root, "-Users-test-project")
	os.MkdirAll(proj, 0755)
	writeJSON(t, filepath.Join(proj, "sessions-index.json"), map[string]any{
		"originalPath": "/Users/test/project",
	})

	old := filepath.Join(proj, "old.jsonl")
	recent := filepath.Join(proj, "recent.jsonl")
	writeFile(t, old, "x")
	writeFile(t, recent, "x")

	past := time.Now().Add(-24 * time.Hour)
	os.Chtimes(old, past, past)

	loader := &claudeLoader{projectsDir: root}
	sessions, _ := loader.load()
	if len(sessions) == 0 {
		t.Fatal("no sessions loaded")
	}
	// mtime should be that of the most recent .jsonl
	recentInfo, _ := os.Stat(recent)
	if sessions[0].Mtime != recentInfo.ModTime().Unix() {
		t.Errorf("mtime: got %d, want %d", sessions[0].Mtime, recentInfo.ModTime().Unix())
	}
}

// ── opencode loader ───────────────────────────────────────────────────────────

func TestOpenCodeLoaderBasic(t *testing.T) {
	root := t.TempDir()

	projDir := filepath.Join(root, "session", "proj1")
	os.MkdirAll(projDir, 0755)

	writeJSON(t, filepath.Join(projDir, "ses_abc.json"), map[string]any{
		"id":        "ses_abc",
		"directory": "/Users/test/myapp",
		"time": map[string]any{
			"created": 1700000000000,
			"updated": 1700010000000,
		},
	})

	loader := &openCodeLoader{storageDir: root}
	sessions, err := loader.load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("want 1, got %d", len(sessions))
	}
	if sessions[0].Path != "/Users/test/myapp" {
		t.Errorf("path: got %q", sessions[0].Path)
	}
	// updated is ms, we store seconds
	if sessions[0].Mtime != 1700010000 {
		t.Errorf("mtime: got %d, want 1700010000", sessions[0].Mtime)
	}
}

func TestOpenCodeLoaderDeduplicatesByPath(t *testing.T) {
	root := t.TempDir()

	projDir := filepath.Join(root, "session", "proj1")
	os.MkdirAll(projDir, 0755)

	// Two sessions for the same directory — keep most recent
	writeJSON(t, filepath.Join(projDir, "ses_old.json"), map[string]any{
		"id": "ses_old", "directory": "/Users/test/app",
		"time": map[string]any{"updated": int64(1700000000000)},
	})
	writeJSON(t, filepath.Join(projDir, "ses_new.json"), map[string]any{
		"id": "ses_new", "directory": "/Users/test/app",
		"time": map[string]any{"updated": int64(1700010000000)},
	})

	loader := &openCodeLoader{storageDir: root}
	sessions, _ := loader.load()
	if len(sessions) != 1 {
		t.Fatalf("want 1 deduplicated session, got %d", len(sessions))
	}
	if sessions[0].Mtime != 1700010000 {
		t.Errorf("kept wrong session: mtime=%d", sessions[0].Mtime)
	}
}

func TestOpenCodeLoaderSkipsGlobalDir(t *testing.T) {
	root := t.TempDir()

	// global dir should be skipped
	globalDir := filepath.Join(root, "session", "global")
	os.MkdirAll(globalDir, 0755)
	writeJSON(t, filepath.Join(globalDir, "ses_x.json"), map[string]any{
		"id": "ses_x", "directory": "/should/skip",
		"time": map[string]any{"updated": int64(1700000000000)},
	})

	loader := &openCodeLoader{storageDir: root}
	sessions, _ := loader.load()
	if len(sessions) != 0 {
		t.Errorf("global dir should be skipped, got %d sessions", len(sessions))
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
