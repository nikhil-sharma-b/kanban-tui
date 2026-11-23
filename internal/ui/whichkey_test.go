package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhilsharma/kanban-tui/internal/sample"
)

func boardAt(t *testing.T, w, h int, flip bool, keys ...string) (*model, tea.Cmd) {
	t.Helper()
	ws := sample.Workspace(sample.Now)
	if flip {
		ws = sample.PlacementFlip(sample.Now)
	}
	m := NewWithOptions(ws, nil, "", Options{Now: func() time.Time { return sample.Now }}).(*model)
	next, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m = next.(*model)
	var cmd tea.Cmd
	for _, k := range keys {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		switch k {
		case " ":
			msg = tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
		case "esc":
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		case "backspace":
			msg = tea.KeyMsg{Type: tea.KeyBackspace}
		}
		next, cmd = m.Update(msg)
		m = next.(*model)
	}
	return m, cmd
}

func TestWhichKeyPlacement(t *testing.T) {
	tests := []struct {
		name string
		w, h int
		flip bool
		keys []string
		want box
	}{
		{"bottom-right above the keybar", 120, 36, false, []string{"l", "j", " "}, box{x: 74, y: 26, w: 44, h: 7}},
		{"group grows upward", 120, 36, false, []string{"l", "j", " ", "t"}, box{x: 74, y: 24, w: 44, h: 9}},
		{"flips left over the selected card", 120, 36, true, append([]string{"l", "l", "l"}, append(strings.Split(strings.Repeat("j", 10), ""), " ")...), box{x: 2, y: 26, w: 44, h: 7}},
		{"80x24 sits on the keybar", 80, 24, false, []string{"l", "j", " "}, box{x: 38, y: 15, w: 40, h: 7}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, _ := boardAt(t, tt.w, tt.h, tt.flip, tt.keys...)
			got := m.whichKeyBox(len(continuations(strings.Join(m.pendingKeys, " "))))
			if got != tt.want {
				t.Fatalf("box = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestWhichKeyShowsAfterDelayOnly(t *testing.T) {
	m, cmd := boardAt(t, 120, 36, false, " ")
	if m.whichKeyShown || cmd == nil {
		t.Fatalf("shown=%v cmd=%v right after SPC", m.whichKeyShown, cmd)
	}
	m, _ = boardAt(t, 120, 36, false, " ", "t")
	next, _ := m.Update(whichKeyMsg{seq: m.whichKeySeq - 1})
	if next.(*model).whichKeyShown {
		t.Fatal("a stale timer showed the popover")
	}
	next, _ = m.Update(whichKeyMsg{seq: m.whichKeySeq})
	if !next.(*model).whichKeyShown {
		t.Fatal("popover not shown once the delay passed")
	}
}

func TestWhichKeyBackspaceEscAndUnbound(t *testing.T) {
	m, _ := boardAt(t, 120, 36, false, " ", "t", "backspace")
	if got := strings.Join(m.pendingKeys, " "); got != "space" {
		t.Fatalf("after backspace pending = %q, want space", got)
	}
	m, _ = boardAt(t, 120, 36, false, " ", "t", "esc")
	if len(m.pendingKeys) != 0 || m.toast.text != "" {
		t.Fatalf("esc should cancel silently: pending=%v toast=%q", m.pendingKeys, m.toast.text)
	}
	m, _ = boardAt(t, 120, 36, false, " ", "t", "q")
	if len(m.pendingKeys) != 0 || m.toast.text != "SPC t q isn’t bound" {
		t.Fatalf("unbound: pending=%v toast=%q", m.pendingKeys, m.toast.text)
	}
}

func TestContinuationsGroupsFirstInTableOrder(t *testing.T) {
	var keys []string
	for _, e := range continuations(leaderKey) {
		keys = append(keys, e.key)
	}
	if got := strings.Join(keys, " "); got != "t c p a d" {
		t.Fatalf("root = %q", got)
	}
	keys = nil
	for _, e := range continuations("space t") {
		keys = append(keys, e.key)
	}
	if got := strings.Join(keys, " "); got != "n e x a h l k j w y" {
		t.Fatalf("task group = %q", got)
	}
}
