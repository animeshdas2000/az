package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── styles ────────────────────────────────────────────────────────────────────

var (
	styleNormal   = lipgloss.NewStyle()
	styleDim      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	styleSelected = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("255")).
			Background(lipgloss.Color("57"))
	styleDir    = lipgloss.NewStyle().Foreground(lipgloss.Color("75"))
	styleBase   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	styleAge    = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	styleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212")).
			MarginBottom(1)
	styleFooter = lipgloss.NewStyle().
			Foreground(lipgloss.Color("238")).
			MarginTop(1)
	styleInput = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212"))
	styleCount = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
)

// ── data ──────────────────────────────────────────────────────────────────────

type project struct {
	path  string
	mtime int64
}

type sessionsIndex struct {
	OriginalPath string `json:"originalPath"`
}

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
	root := filepath.Join(home, ".claude", "projects")

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
		projects = append(projects, project{
			path:  slugToPath(dir),
			mtime: lastMtime(dir),
		})
	}

	sort.Slice(projects, func(i, j int) bool {
		return projects[i].mtime > projects[j].mtime
	})
	return projects
}

func filterProjects(query string, all []project) []project {
	if query == "" {
		return all
	}
	q := strings.ToLower(query)
	var out []project
	for _, p := range all {
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

func relTime(unix int64) string {
	d := time.Since(time.Unix(unix, 0))
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return time.Unix(unix, 0).Format("Jan 2")
	}
}

func formatPath(path string) (dir string, base string) {
	return filepath.Dir(path) + "/", filepath.Base(path)
}

// ── TUI model ─────────────────────────────────────────────────────────────────

type model struct {
	all      []project
	filtered []project
	cursor   int
	input    textinput.Model
	chosen   string
	height   int
	width    int
}

func initialModel(projects []project, query string) model {
	ti := textinput.New()
	ti.Placeholder = "filter sessions…"
	ti.Focus()
	ti.PromptStyle = styleInput
	ti.TextStyle = styleNormal
	ti.Prompt = "  "
	ti.SetValue(query)

	filtered := filterProjects(query, projects)

	return model{
		all:      projects,
		filtered: filtered,
		input:    ti,
		height:   24,
		width:    80,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyEnter:
			if len(m.filtered) > 0 {
				m.chosen = m.filtered[m.cursor].path
			}
			return m, tea.Quit

		case tea.KeyUp, tea.KeyCtrlP:
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case tea.KeyDown, tea.KeyCtrlN:
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, nil

		case tea.KeyPgUp:
			m.cursor -= m.listHeight()
			if m.cursor < 0 {
				m.cursor = 0
			}
			return m, nil

		case tea.KeyPgDown:
			m.cursor += m.listHeight()
			if m.cursor >= len(m.filtered) {
				m.cursor = len(m.filtered) - 1
			}
			return m, nil
		}
	}

	// Pass remaining keys to the text input.
	prev := m.input.Value()
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	if v := m.input.Value(); v != prev {
		m.filtered = filterProjects(v, m.all)
		m.cursor = 0
	}
	return m, cmd
}

func (m model) listHeight() int {
	// header(2) + input(1) + border(2) + footer(2) = 7 overhead
	h := m.height - 7
	if h < 1 {
		h = 1
	}
	return h
}

func (m model) View() string {
	var b strings.Builder

	// ── header ────────────────────────────────────────────────────────────────
	b.WriteString(styleHeader.Render("  cz  —  claude session jumper"))
	b.WriteByte('\n')

	// ── search input ──────────────────────────────────────────────────────────
	countStr := styleCount.Render(fmt.Sprintf(" %d/%d", len(m.filtered), len(m.all)))
	b.WriteString(m.input.View() + countStr)
	b.WriteByte('\n')
	b.WriteString(styleDim.Render(strings.Repeat("─", m.width)))
	b.WriteByte('\n')

	// ── list ──────────────────────────────────────────────────────────────────
	maxRows := m.listHeight()
	start := 0
	if m.cursor >= maxRows {
		start = m.cursor - maxRows + 1
	}
	end := start + maxRows
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	if len(m.filtered) == 0 {
		b.WriteString(styleDim.Render("  no matches"))
		b.WriteByte('\n')
	}

	for i := start; i < end; i++ {
		p := m.filtered[i]
		dir, base := formatPath(p.path)
		age := relTime(p.mtime)

		// trim home prefix for display
		home, _ := os.UserHomeDir()
		dir = strings.Replace(dir, home, "~", 1)

		line := fmt.Sprintf("  %s%s", dir, base)
		agePad := m.width - lipgloss.Width(line) - lipgloss.Width(age) - 2
		if agePad < 1 {
			agePad = 1
		}

		if i == m.cursor {
			rendered := styleSelected.Width(m.width).Render(
				fmt.Sprintf("  %s%s%s%s",
					styleDir.Inherit(styleSelected).Render(dir),
					styleBase.Inherit(styleSelected).Render(base),
					strings.Repeat(" ", agePad),
					styleAge.Inherit(styleSelected).Render(age),
				),
			)
			b.WriteString(rendered)
		} else {
			b.WriteString(fmt.Sprintf("  %s%s%s%s",
				styleDim.Render(dir),
				styleNormal.Render(base),
				strings.Repeat(" ", agePad),
				styleAge.Render(age),
			))
		}
		b.WriteByte('\n')
	}

	// ── footer ────────────────────────────────────────────────────────────────
	b.WriteString(styleDim.Render(strings.Repeat("─", m.width)))
	b.WriteByte('\n')
	b.WriteString(styleFooter.Render("  ↑↓ navigate   enter select   esc quit"))

	return b.String()
}

// ── entry point ───────────────────────────────────────────────────────────────

func main() {
	args := os.Args[1:]

	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Println(`cz - Claude session jumper

Usage:
  cz [query]   Open Claude session matching <query> (interactive TUI).
  cz --list    Print all session paths sorted by recency and exit.`)
		return
	}

	projects := loadProjects()

	if len(args) > 0 && args[0] == "--list" {
		for _, p := range projects {
			fmt.Println(p.path)
		}
		return
	}

	query := ""
	if len(args) > 0 {
		query = args[0]
	}

	// Run TUI on /dev/tty so stdout stays clean for the shell wrapper.
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cz: cannot open /dev/tty:", err)
		os.Exit(1)
	}

	m := initialModel(projects, query)
	p := tea.NewProgram(m,
		tea.WithInput(tty),
		tea.WithOutput(tty),
		tea.WithAltScreen(),
	)

	result, err := p.Run()
	tty.Close()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cz:", err)
		os.Exit(1)
	}

	final := result.(model)
	if final.chosen == "" {
		os.Exit(1)
	}

	if _, err := os.Stat(final.chosen); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "cz: directory no longer exists: %s\n", final.chosen)
		os.Exit(1)
	}

	// Print chosen path to stdout for the shell wrapper to cd into.
	fmt.Println(final.chosen)

	// If CZ_EXEC=1, exec claude directly in the target directory.
	if os.Getenv("CZ_EXEC") == "1" {
		claudePath, err := exec.LookPath("claude")
		if err != nil {
			fmt.Fprintln(os.Stderr, "cz: claude not found in PATH")
			os.Exit(1)
		}
		os.Chdir(final.chosen)
		syscallExec(claudePath)
	}
}
