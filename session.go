package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type session struct {
	Path  string
	Mtime int64 // unix seconds
}

// loader is the interface for per-agent session discovery.
type loader interface {
	load() ([]session, error)
}

// loaderFor returns the appropriate loader for a given agent.
func loaderFor(a agent) loader {
	home, _ := os.UserHomeDir()
	switch a.Name {
	case "claude":
		return &claudeLoader{
			projectsDir: filepath.Join(home, ".claude", "projects"),
		}
	case "opencode":
		return &openCodeLoader{
			storageDir: filepath.Join(home, ".local", "share", "opencode", "storage"),
		}
	default:
		return &noopLoader{}
	}
}

// ── claude ────────────────────────────────────────────────────────────────────

type claudeLoader struct {
	projectsDir string
}

type claudeSessionsIndex struct {
	OriginalPath string `json:"originalPath"`
}

func (l *claudeLoader) load() ([]session, error) {
	entries, err := os.ReadDir(l.projectsDir)
	if err != nil {
		return nil, err
	}

	var sessions []session
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(l.projectsDir, e.Name())
		path := claudeResolvePath(dir)
		sessions = append(sessions, session{
			Path:  path,
			Mtime: jsonlMtime(dir),
		})
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Mtime > sessions[j].Mtime
	})
	return sessions, nil
}

func claudeResolvePath(projectDir string) string {
	indexFile := filepath.Join(projectDir, "sessions-index.json")
	if data, err := os.ReadFile(indexFile); err == nil {
		var idx claudeSessionsIndex
		if json.Unmarshal(data, &idx) == nil && idx.OriginalPath != "" {
			return idx.OriginalPath
		}
	}
	return greedyPath(filepath.Base(projectDir))
}

// greedyPath reconstructs a filesystem path from a Claude project slug like
// -Users-animeshdas-Desktop-realfast-exo-code-server by greedily matching
// hyphen-separated parts against real filesystem entries.
func greedyPath(slug string) string {
	parts := strings.Split(strings.TrimPrefix(slug, "-"), "-")
	if result := buildPath(parts, "/"); result != "" {
		return result
	}
	return strings.ReplaceAll(slug, "-", "/")
}

func buildPath(parts []string, current string) string {
	if len(parts) == 0 {
		return current
	}
	for n := 1; n <= len(parts); n++ {
		component := strings.Join(parts[:n], "-")
		candidate := filepath.Join(current, component)
		if _, err := os.Stat(candidate); err == nil {
			if result := buildPath(parts[n:], candidate); result != "" {
				return result
			}
		}
	}
	return ""
}

func jsonlMtime(dir string) int64 {
	entries, _ := os.ReadDir(dir)
	var best int64
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		if info, err := e.Info(); err == nil {
			if t := info.ModTime().Unix(); t > best {
				best = t
			}
		}
	}
	if best > 0 {
		return best
	}
	if info, err := os.Stat(dir); err == nil {
		return info.ModTime().Unix()
	}
	return 0
}

// ── opencode ──────────────────────────────────────────────────────────────────

type openCodeLoader struct {
	storageDir string
}

type openCodeSession struct {
	ID        string `json:"id"`
	Directory string `json:"directory"`
	Time      struct {
		Updated int64 `json:"updated"` // milliseconds
	} `json:"time"`
}

func (l *openCodeLoader) load() ([]session, error) {
	sessionRoot := filepath.Join(l.storageDir, "session")
	projects, err := os.ReadDir(sessionRoot)
	if err != nil {
		return nil, err
	}

	// path → most recent mtime (ms → s)
	best := map[string]int64{}

	for _, proj := range projects {
		if !proj.IsDir() || proj.Name() == "global" {
			continue
		}
		projDir := filepath.Join(sessionRoot, proj.Name())
		files, err := os.ReadDir(projDir)
		if err != nil {
			continue
		}
		for _, f := range files {
			if !strings.HasSuffix(f.Name(), ".json") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(projDir, f.Name()))
			if err != nil {
				continue
			}
			var s openCodeSession
			if err := json.Unmarshal(data, &s); err != nil || s.Directory == "" {
				continue
			}
			mtime := s.Time.Updated / 1000 // ms → s
			if mtime > best[s.Directory] {
				best[s.Directory] = mtime
			}
		}
	}

	sessions := make([]session, 0, len(best))
	for path, mtime := range best {
		sessions = append(sessions, session{Path: path, Mtime: mtime})
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Mtime > sessions[j].Mtime
	})
	return sessions, nil
}

// ── noop (agents without session tracking) ───────────────────────────────────

type noopLoader struct{}

func (l *noopLoader) load() ([]session, error) {
	return nil, nil
}
