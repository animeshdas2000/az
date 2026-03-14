package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindDirsMatchesName(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "projects", "dokkai"), 0755)
	os.MkdirAll(filepath.Join(root, "projects", "scwroll"), 0755)

	got := findDirs("dokkai", []string{root}, 3)
	if len(got) != 1 {
		t.Fatalf("want 1, got %d: %v", len(got), got)
	}
	if filepath.Base(got[0].Path) != "dokkai" {
		t.Errorf("unexpected path: %s", got[0].Path)
	}
}

func TestFindDirsCaseInsensitive(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "MyProject"), 0755)

	got := findDirs("myproject", []string{root}, 2)
	if len(got) != 1 {
		t.Fatalf("want 1, got %d", len(got))
	}
}

func TestFindDirsNoMatch(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "other"), 0755)

	got := findDirs("zzznomatch", []string{root}, 2)
	if len(got) != 0 {
		t.Errorf("want 0, got %d", len(got))
	}
}

func TestFindDirsRespectsDepth(t *testing.T) {
	root := t.TempDir()
	// depth 3 from root → root/a/b/target — should NOT be found at depth 2
	os.MkdirAll(filepath.Join(root, "a", "b", "target"), 0755)

	got := findDirs("target", []string{root}, 2)
	if len(got) != 0 {
		t.Errorf("depth 2 should not reach depth-3 dir, got %d", len(got))
	}

	got = findDirs("target", []string{root}, 3)
	if len(got) != 1 {
		t.Fatalf("depth 3 should find it, got %d", len(got))
	}
}

func TestFindDirsSkipsHidden(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".hidden", "target"), 0755)

	got := findDirs("target", []string{root}, 3)
	if len(got) != 0 {
		t.Errorf("should skip dirs inside hidden folders, got %d", len(got))
	}
}

func TestFindDirsSkipsNodeModules(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "node_modules", "target"), 0755)

	got := findDirs("target", []string{root}, 3)
	if len(got) != 0 {
		t.Errorf("should skip node_modules, got %d", len(got))
	}
}

func TestFindDirsMultipleRoots(t *testing.T) {
	root1 := t.TempDir()
	root2 := t.TempDir()
	os.MkdirAll(filepath.Join(root1, "alpha"), 0755)
	os.MkdirAll(filepath.Join(root2, "alpha"), 0755)

	got := findDirs("alpha", []string{root1, root2}, 2)
	if len(got) != 2 {
		t.Fatalf("want 2 results from 2 roots, got %d", len(got))
	}
}
