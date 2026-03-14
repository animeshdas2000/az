package main

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()
	if cfg.Default == "" {
		t.Fatal("default agent must not be empty")
	}
	if len(cfg.Agents) == 0 {
		t.Fatal("default config must include at least one agent")
	}
}

func TestDefaultConfigHasClaude(t *testing.T) {
	cfg := defaultConfig()
	a, ok := cfg.resolve("")
	if !ok {
		t.Fatal("default keyword '' must resolve to an agent")
	}
	if a.Name != "claude" {
		t.Errorf("default agent: want claude, got %s", a.Name)
	}
}

func TestResolveByKeyword(t *testing.T) {
	cfg := config{
		Default: "claude",
		Agents: []agent{
			{Name: "claude", Keyword: "", Cmd: "claude"},
			{Name: "opencode", Keyword: "o", Cmd: "opencode"},
			{Name: "gemini", Keyword: "g", Cmd: "gemini"},
		},
	}

	tests := []struct {
		keyword  string
		wantName string
		wantOK   bool
	}{
		{"", "claude", true},
		{"o", "opencode", true},
		{"g", "gemini", true},
		{"opencode", "opencode", true}, // full name also works
		{"unknown", "", false},
	}

	for _, tt := range tests {
		a, ok := cfg.resolve(tt.keyword)
		if ok != tt.wantOK {
			t.Errorf("resolve(%q): ok=%v, want %v", tt.keyword, ok, tt.wantOK)
			continue
		}
		if ok && a.Name != tt.wantName {
			t.Errorf("resolve(%q): name=%q, want %q", tt.keyword, a.Name, tt.wantName)
		}
	}
}

func TestParseArgs(t *testing.T) {
	cfg := config{
		Default: "claude",
		Agents: []agent{
			{Name: "claude", Keyword: "", Cmd: "claude"},
			{Name: "opencode", Keyword: "o", Cmd: "opencode"},
		},
	}

	tests := []struct {
		args      []string
		wantAgent string
		wantQuery string
	}{
		// no args → default agent, no filter
		{[]string{}, "claude", ""},
		// only query → default agent
		{[]string{"dokkai"}, "claude", "dokkai"},
		// keyword + query
		{[]string{"o", "scwroll"}, "opencode", "scwroll"},
		// keyword only, no query
		{[]string{"o"}, "opencode", ""},
		// unknown first arg → treat as query for default agent
		{[]string{"boardroom"}, "claude", "boardroom"},
	}

	for _, tt := range tests {
		a, q := cfg.parseArgs(tt.args)
		if a.Name != tt.wantAgent {
			t.Errorf("parseArgs(%v): agent=%q, want %q", tt.args, a.Name, tt.wantAgent)
		}
		if q != tt.wantQuery {
			t.Errorf("parseArgs(%v): query=%q, want %q", tt.args, q, tt.wantQuery)
		}
	}
}

func TestConfigRoundTrip(t *testing.T) {
	original := config{
		Default: "opencode",
		Agents: []agent{
			{Name: "claude", Keyword: "", Cmd: "claude"},
			{Name: "opencode", Keyword: "o", Cmd: "opencode"},
		},
	}

	data, err := original.marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	parsed, err := unmarshalConfig(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if parsed.Default != original.Default {
		t.Errorf("default: got %q, want %q", parsed.Default, original.Default)
	}
	if len(parsed.Agents) != len(original.Agents) {
		t.Errorf("agents len: got %d, want %d", len(parsed.Agents), len(original.Agents))
	}
}
