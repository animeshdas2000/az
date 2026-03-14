package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
)

const projectsDir = ".claude/projects"

type project struct {
	path  string
	mtime int64
}

// sessionsIndex mirrors the sessions-index.json structure.
type sessionsIndex struct {
	OriginalPath string `json:"originalPath"`
}

// slugToPath recovers the filesystem path from a project slug.
// It first checks sessions-index.json for an authoritative originalPath,
// then falls back to a greedy filesystem-aware reconstruction.
func slugToPath(projectDir string) string {
	indexFile := filepath.Join(projectDir, "sessions-index.json")
	if data, err := os.ReadFile(indexFile); err == nil {
		var idx sessionsIndex
		if json.Unmarshal(data, &idx) == nil && idx.OriginalPath != "" {
			return idx.OriginalPath
		}
	}
	return greedyPath(filepath.Base(projectDir))
}

// greedyPath reconstructs a path from a slug like -Users-foo-bar-baz
// by greedily matching hyphen-separated parts against real filesystem entries.
func greedyPath(slug string) string {
	parts := strings.Split(strings.TrimPrefix(slug, "-"), "-")
	result := buildPath(parts, "/")
	if result != "" {
		return result
	}
	// naive fallback: replace all - with /
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

// lastMtime returns the most recent mtime of any .jsonl file in dir,
// falling back to the directory mtime.
func lastMtime(dir string) int64 {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
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

func loadProjects() []project {
	home, _ := os.UserHomeDir()
	root := filepath.Join(home, projectsDir)

	entries, err := os.ReadDir(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cz: cannot read %s: %v\n", root, err)
		os.Exit(1)
	}

	var projects []project
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(root, e.Name())
		path := slugToPath(dir)
		projects = append(projects, project{
			path:  path,
			mtime: lastMtime(dir),
		})
	}

	sort.Slice(projects, func(i, j int) bool {
		return projects[i].mtime > projects[j].mtime
	})
	return projects
}

func matchProjects(query string, projects []project) []project {
	q := strings.ToLower(query)
	var out []project
	for _, p := range projects {
		parts := strings.Split(p.path, string(os.PathSeparator))
		for _, part := range parts {
			if strings.Contains(strings.ToLower(part), q) {
				out = append(out, p)
				break
			}
		}
	}
	return out
}

func pick(projects []project) string {
	if len(projects) == 0 {
		return ""
	}
	if len(projects) == 1 {
		return projects[0].path
	}

	// Try fzf first.
	if path, err := pickWithFzf(projects); err == nil {
		return path
	}

	// Fallback: numbered list printed to stderr so the caller can read stdout.
	fmt.Fprintln(os.Stderr, "Multiple matches:")
	for i, p := range projects {
		fmt.Fprintf(os.Stderr, "  %d) %s\n", i+1, p.path)
	}
	fmt.Fprint(os.Stderr, "Choose: ")
	var choice int
	fmt.Scan(&choice)
	if choice < 1 || choice > len(projects) {
		return ""
	}
	return projects[choice-1].path
}

func pickWithFzf(projects []project) (string, error) {
	fzfPath, err := exec.LookPath("fzf")
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for _, p := range projects {
		sb.WriteString(p.path)
		sb.WriteByte('\n')
	}

	cmd := exec.Command(fzfPath, "--prompt=cz> ", "--height=40%", "--reverse", "--no-sort")
	cmd.Stdin = strings.NewReader(sb.String())
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func usage() {
	fmt.Println(`cz - Claude session jumper

Usage:
  cz [query]   Open the most recent Claude session matching <query>.
               With no query, shows all sessions.
  cz --list    Print all sessions (path, sorted by recency) and exit.

When multiple directories match, an interactive picker is shown (fzf if
available, otherwise a numbered list).`)
}

func main() {
	args := os.Args[1:]

	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		usage()
		return
	}

	projects := loadProjects()

	// --list: just print paths and exit (used by the shell wrapper).
	if len(args) > 0 && args[0] == "--list" {
		for _, p := range projects {
			fmt.Println(p.path)
		}
		return
	}

	if len(args) > 0 {
		projects = matchProjects(args[0], projects)
	}

	if len(projects) == 0 {
		query := ""
		if len(args) > 0 {
			query = args[0]
		}
		if query != "" {
			fmt.Fprintf(os.Stderr, "cz: no sessions matching %q\n", query)
		} else {
			fmt.Fprintln(os.Stderr, "cz: no sessions found")
		}
		os.Exit(1)
	}

	target := pick(projects)
	if target == "" {
		os.Exit(1)
	}

	if _, err := os.Stat(target); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "cz: directory no longer exists: %s\n", target)
		os.Exit(1)
	}

	// Print the chosen path to stdout so the shell wrapper can cd to it.
	fmt.Println(target)

	// Also exec claude directly if CZ_EXEC=1 (set by the shell wrapper after cd).
	if os.Getenv("CZ_EXEC") == "1" {
		claudePath, err := exec.LookPath("claude")
		if err != nil {
			fmt.Fprintln(os.Stderr, "cz: claude not found in PATH")
			os.Exit(1)
		}
		if err := os.Chdir(target); err != nil {
			fmt.Fprintf(os.Stderr, "cz: chdir failed: %v\n", err)
			os.Exit(1)
		}
		syscall.Exec(claudePath, []string{"claude"}, os.Environ())
	}
}
