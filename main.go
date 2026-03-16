package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// lookPath is a thin wrapper around exec.LookPath for testability.
var lookPath = exec.LookPath

func main() {
	args := os.Args[1:]

	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		printHelp()
		return
	}

	cfg := loadConfig()

	if len(args) > 0 && args[0] == "--agents" {
		printAgents(cfg)
		return
	}

	if len(args) > 0 && args[0] == "--list" {
		a, _ := cfg.resolve("")
		printSessions(a)
		return
	}

	activeAgent, query := cfg.parseArgs(args)

	// --list after agent keyword: az o --list
	if query == "--list" {
		printSessions(activeAgent)
		return
	}

	sessions, err := loaderFor(activeAgent).load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "az: load sessions: %v\n", err)
	}
	sessions = dedup(sessions)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Mtime > sessions[j].Mtime
	})

	// If a query was given but produces no session matches, fall back to a
	// live filesystem search so the user always gets a result.
	fromFS := false
	if query != "" && len(filterSessions(query, sessions)) == 0 {
		fsDirs := findDirs(query, defaultSearchRoots(), 4)
		if len(fsDirs) == 1 {
			// Exactly one filesystem match — open immediately, no TUI needed.
			fmt.Println(fsDirs[0].Path)
			return
		}
		if len(fsDirs) > 1 {
			sessions = fsDirs
			fromFS = true
		}
	}

	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, "az: cannot open /dev/tty:", err)
		os.Exit(1)
	}

	m := initialModel(activeAgent, sessions, query)
	m.fromFS = fromFS
	p := tea.NewProgram(m,
		tea.WithInput(tty),
		tea.WithOutput(tty),
		tea.WithAltScreen(),
	)

	result, err := p.Run()
	tty.Close()
	if err != nil {
		fmt.Fprintln(os.Stderr, "az:", err)
		os.Exit(1)
	}

	final := result.(model)
	if final.chosen == "" {
		os.Exit(1)
	}

	if _, err := os.Stat(final.chosen); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "az: directory no longer exists: %s\n", final.chosen)
		os.Exit(1)
	}

	fmt.Println(final.chosen)

	if os.Getenv("AZ_EXEC") == "1" {
		cmdPath, err := lookPath(activeAgent.Cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "az: %s not found in PATH\n", activeAgent.Cmd)
			os.Exit(1)
		}
		os.Chdir(final.chosen)
		syscallExec(cmdPath)
	}
}

func printHelp() {
	fmt.Print(`az - agent z: jump to any agent session by directory name

Usage:
  az [query]             Open default agent (claude) session matching <query>.
  az <keyword> [query]   Open specific agent session.
  az --agents            List all configured agents and their keywords.
  az --list              Print all sessions for default agent and exit.

Examples:
  az dokkai              jump to claude session in ~/…/dokkai
  az o scwroll           jump to opencode session in ~/…/scwroll
  az g                   browse all gemini sessions
  az o                   browse all opencode sessions
`)
}

func printAgents(cfg config) {
	fmt.Printf("%-12s  %-10s  %s\n", "NAME", "KEYWORD", "CMD")
	fmt.Println(strings.Repeat("─", 40))
	for _, a := range cfg.Agents {
		kw := a.Keyword
		if kw == "" {
			kw = "(default)"
		}
		marker := ""
		if a.Name == cfg.Default {
			marker = " ✓"
		}
		fmt.Printf("%-12s  %-10s  %s%s\n", a.Name, kw, a.Cmd, marker)
	}
}

func printSessions(a agent) {
	sessions, err := loaderFor(a).load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "az:", err)
		os.Exit(1)
	}
	for _, s := range dedup(sessions) {
		fmt.Println(s.Path)
	}
}
