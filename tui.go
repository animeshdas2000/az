package main

import (
	"fmt"
	"os"
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
	styleHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")).MarginBottom(1)
	styleFooter = lipgloss.NewStyle().Foreground(lipgloss.Color("238")).MarginTop(1)
	styleInput  = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	styleCount  = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	styleAgent  = lipgloss.NewStyle().Foreground(lipgloss.Color("43")).Bold(true)
	styleNoSess = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

// ── helpers ───────────────────────────────────────────────────────────────────

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

func agentLabel(a agent) string {
	if a.Label != "" {
		return a.Label
	}
	return a.Name
}

func lastSegment(path string) string {
	parts := strings.Split(strings.TrimRight(path, "/"), "/")
	if len(parts) == 0 {
		return path
	}
	return parts[len(parts)-1]
}

// ── model ─────────────────────────────────────────────────────────────────────

type model struct {
	activeAgent agent
	all         []session
	filtered    []session
	cursor      int
	input       textinput.Model
	chosen      string
	height      int
	width       int
	fromFS      bool // true when showing filesystem search results, not session history
}

func initialModel(a agent, sessions []session, query string) model {
	ti := textinput.New()
	ti.Placeholder = "filter sessions…"
	ti.Focus()
	ti.PromptStyle = styleInput
	ti.TextStyle = styleNormal
	ti.Prompt = "  "
	ti.SetValue(query)

	return model{
		activeAgent: a,
		all:         sessions,
		filtered:    filterSessions(query, sessions),
		input:       ti,
		height:      24,
		width:       80,
	}
}

func (m model) Init() tea.Cmd { return textinput.Blink }

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
				m.chosen = m.filtered[m.cursor].Path
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
			if m.cursor -= m.listHeight(); m.cursor < 0 {
				m.cursor = 0
			}
			return m, nil
		case tea.KeyPgDown:
			if m.cursor += m.listHeight(); m.cursor >= len(m.filtered) {
				m.cursor = len(m.filtered) - 1
			}
			return m, nil
		}
	}

	prev := m.input.Value()
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	if v := m.input.Value(); v != prev {
		m.filtered = filterSessions(v, m.all)
		m.cursor = 0
	}
	return m, cmd
}

func (m model) listHeight() int {
	if h := m.height - 7; h > 0 {
		return h
	}
	return 1
}

func (m model) View() string {
	var b strings.Builder

	label := agentLabel(m.activeAgent)
	if m.fromFS {
		label += styleDim.Render("  (filesystem search)")
	}
	b.WriteString(styleHeader.Render(fmt.Sprintf("  az  —  %s", styleAgent.Render(label))))
	b.WriteByte('\n')

	b.WriteString(m.input.View())
	b.WriteString(styleCount.Render(fmt.Sprintf(" %d/%d", len(m.filtered), len(m.all))))
	b.WriteByte('\n')
	b.WriteString(styleDim.Render(strings.Repeat("─", m.width)))
	b.WriteByte('\n')

	switch {
	case len(m.all) == 0:
		b.WriteString(styleNoSess.Render(fmt.Sprintf("  no session history for %s", agentLabel(m.activeAgent))))
		b.WriteByte('\n')
	case len(m.filtered) == 0:
		b.WriteString(styleDim.Render("  no matches"))
		b.WriteByte('\n')
	default:
		m.renderList(&b)
	}

	b.WriteString(styleDim.Render(strings.Repeat("─", m.width)))
	b.WriteByte('\n')
	b.WriteString(styleFooter.Render("  ↑↓ navigate   enter select   esc quit   az <keyword> [query] to switch agent"))

	return b.String()
}

func (m model) renderList(b *strings.Builder) {
	maxRows := m.listHeight()
	start := 0
	if m.cursor >= maxRows {
		start = m.cursor - maxRows + 1
	}
	end := start + maxRows
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	home, _ := os.UserHomeDir()

	for i := start; i < end; i++ {
		s := m.filtered[i]
		base := lastSegment(s.Path)
		dir := strings.TrimSuffix(s.Path, "/"+base)
		age := relTime(s.Mtime)
		dirDisplay := strings.Replace(dir, home, "~", 1) + "/"

		agePad := m.width - lipgloss.Width("  "+dirDisplay+base) - lipgloss.Width(age) - 2
		if agePad < 1 {
			agePad = 1
		}

		if i == m.cursor {
			row := fmt.Sprintf("  %s%s%s%s",
				styleDir.Inherit(styleSelected).Render(dirDisplay),
				styleBase.Inherit(styleSelected).Render(base),
				strings.Repeat(" ", agePad),
				styleAge.Inherit(styleSelected).Render(age),
			)
			b.WriteString(styleSelected.Width(m.width).Render(row))
		} else {
			b.WriteString(fmt.Sprintf("  %s%s%s%s",
				styleDim.Render(dirDisplay),
				styleNormal.Render(base),
				strings.Repeat(" ", agePad),
				styleAge.Render(age),
			))
		}
		b.WriteByte('\n')
	}
}
