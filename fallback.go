package main

import (
	"os"
	"path/filepath"
	"strings"
)

// skipDirs are directory names we never descend into.
var skipDirs = map[string]bool{
	"node_modules": true,
	".git":         true,
	"vendor":       true,
	".venv":        true,
	"venv":         true,
	"__pycache__":  true,
	"dist":         true,
	"build":        true,
	".next":        true,
}

// findDirs searches for directories whose base name contains query
// (case-insensitive) within roots up to maxDepth levels deep.
// Hidden directories (name starts with '.') and skipDirs are not descended.
func findDirs(query string, roots []string, maxDepth int) []session {
	q := strings.ToLower(query)
	var results []session

	for _, root := range roots {
		walk(root, q, 0, maxDepth, &results)
	}
	return results
}

func walk(dir, query string, depth, maxDepth int, results *[]session) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		// skip hidden and known noisy dirs
		if strings.HasPrefix(name, ".") || skipDirs[name] {
			continue
		}

		full := filepath.Join(dir, name)

		if strings.Contains(strings.ToLower(name), query) {
			info, err := e.Info()
			mtime := int64(0)
			if err == nil {
				mtime = info.ModTime().Unix()
			}
			*results = append(*results, session{Path: full, Mtime: mtime})
		}

		if depth+1 < maxDepth {
			walk(full, query, depth+1, maxDepth, results)
		}
	}
}

// defaultSearchRoots returns the directories az will search when there are no
// session history matches.
func defaultSearchRoots() []string {
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, "Desktop"),
		filepath.Join(home, "Documents"),
		filepath.Join(home, "Projects"),
		filepath.Join(home, "code"),
		filepath.Join(home, "src"),
		home,
	}
	var roots []string
	seen := map[string]bool{}
	for _, c := range candidates {
		if seen[c] {
			continue
		}
		seen[c] = true
		if _, err := os.Stat(c); err == nil {
			roots = append(roots, c)
		}
	}
	return roots
}
