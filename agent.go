package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type agent struct {
	Name    string `json:"name"`
	Keyword string `json:"keyword"`
	Cmd     string `json:"cmd"`
	// Display label shown in TUI header; falls back to Name if empty.
	Label string `json:"label,omitempty"`
}

type config struct {
	Default string  `json:"default"`
	Agents  []agent `json:"agents"`
}

// defaultConfig returns built-in agents. Only agents whose binary is found
// in PATH are included, plus claude which is always present as default.
func defaultConfig() config {
	all := []agent{
		{Name: "claude", Keyword: "", Cmd: "claude", Label: "Claude Code"},
		{Name: "opencode", Keyword: "o", Cmd: "opencode", Label: "OpenCode"},
		{Name: "gemini", Keyword: "g", Cmd: "gemini", Label: "Gemini CLI"},
		{Name: "codex", Keyword: "c", Cmd: "codex", Label: "Codex"},
		{Name: "amp", Keyword: "amp", Cmd: "amp", Label: "Amp"},
	}

	var agents []agent
	for _, a := range all {
		if _, err := findExec(a.Cmd); err == nil {
			agents = append(agents, a)
		}
	}
	// Always include claude as the baseline default even if not found.
	if len(agents) == 0 {
		agents = []agent{all[0]}
	}

	return config{Default: "claude", Agents: agents}
}

// resolve returns the agent matching keyword (by Keyword field or full Name).
// An empty keyword returns the default agent.
func (c config) resolve(keyword string) (agent, bool) {
	if keyword == "" {
		for _, a := range c.Agents {
			if a.Name == c.Default {
				return a, true
			}
		}
		if len(c.Agents) > 0 {
			return c.Agents[0], true
		}
		return agent{}, false
	}

	kw := strings.ToLower(keyword)
	for _, a := range c.Agents {
		if strings.ToLower(a.Keyword) == kw || strings.ToLower(a.Name) == kw {
			return a, true
		}
	}
	return agent{}, false
}

// parseArgs interprets os.Args[1:] as [keyword] [query].
// If the first argument matches a known agent keyword or name, it is consumed
// as the agent selector; otherwise the default agent is used and the first
// argument becomes the query.
func (c config) parseArgs(args []string) (agent, string) {
	def, _ := c.resolve("")

	if len(args) == 0 {
		return def, ""
	}

	if a, ok := c.resolve(args[0]); ok && args[0] != "" {
		query := ""
		if len(args) > 1 {
			query = args[1]
		}
		return a, query
	}

	// First arg is not a keyword — treat as query for default agent.
	return def, args[0]
}

// marshal serialises the config to JSON.
func (c config) marshal() ([]byte, error) {
	return json.MarshalIndent(c, "", "  ")
}

func unmarshalConfig(data []byte) (config, error) {
	var c config
	return c, json.Unmarshal(data, &c)
}

// loadConfig reads ~/.config/az/config.json; returns defaultConfig on any error.
func loadConfig() config {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".config", "az", "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return defaultConfig()
	}
	cfg, err := unmarshalConfig(data)
	if err != nil {
		return defaultConfig()
	}
	return cfg
}

// findExec looks up a binary in PATH (wraps exec.LookPath for testability).
func findExec(name string) (string, error) {
	return lookPath(name)
}
