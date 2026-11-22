package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/nikhilsharma/kanban-tui/internal/sample"
)

func TestRailSideCollapsesThreeOrMore(t *testing.T) {
	cases := []struct {
		names []string
		left  bool
		want  string
	}{
		{[]string{"backlog"}, true, "backlog"},
		{[]string{"doing", "done"}, false, "doing · done"},
		{[]string{"doing", "review", "done"}, false, "doing +2"},
		{[]string{"backlog", "ideas", "todo"}, true, "+2 todo"},
	}
	for _, c := range cases {
		if got := railSide(c.names, c.left); got != c.want {
			t.Errorf("railSide(%v, %v) = %q, want %q", c.names, c.left, got, c.want)
		}
	}
}

func TestResizeSwitchesLayoutAndKeepsSelection(t *testing.T) {
	var m tea.Model = NewWithOptions(sample.Workspace(sample.Now), nil, "", Options{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	for _, k := range []string{"l", "j", "l"} {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
	}
	selected := m.(*model).selectedTask().ID

	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	wide := ansi.Strip(m.View())
	if rows := strings.Split(wide, "\n"); len(rows) != 36 {
		t.Fatalf("120×36 view has %d rows", len(rows))
	}
	if strings.Contains(wide, "‹ backlog") || !strings.Contains(wide, "backlog · 5") {
		t.Fatalf("120×36 view should show all four columns and no rail:\n%s", wide)
	}
	if got := m.(*model).selectedTask().ID; got != selected {
		t.Fatalf("selection changed on resize: %s → %s", selected, got)
	}

	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	narrow := ansi.Strip(m.View())
	if rows := strings.Split(narrow, "\n"); len(rows) != 24 {
		t.Fatalf("80×24 view has %d rows", len(rows))
	}
	if !strings.Contains(narrow, "‹ backlog") || !strings.Contains(narrow, "done ›") {
		t.Fatalf("80×24 view should name hidden columns in the rail:\n%s", narrow)
	}
	if got := m.(*model).selectedTask().ID; got != selected {
		t.Fatalf("selection changed on resize: %s → %s", selected, got)
	}
}
