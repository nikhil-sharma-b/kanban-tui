package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhilsharma/kanban-tui/internal/sample"
)

func confirmModel(t *testing.T) *model {
	t.Helper()
	m := NewWithOptions(sample.Workspace(sample.Now), nil, "", Options{}).(*model)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	return next.(*model)
}

func pressKeys(t *testing.T, m *model, keys ...tea.KeyMsg) *model {
	t.Helper()
	for _, k := range keys {
		next, _ := m.Update(k)
		m = next.(*model)
	}
	return m
}

func typed(s string) []tea.KeyMsg {
	var keys []tea.KeyMsg
	for _, r := range s {
		if r == ' ' {
			keys = append(keys, tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
			continue
		}
		keys = append(keys, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return keys
}

var (
	enterKey = tea.KeyMsg{Type: tea.KeyEnter}
	escKey   = tea.KeyMsg{Type: tea.KeyEsc}
)

func TestTierOneEnterIsInert(t *testing.T) {
	m := confirmModel(t)
	m = pressKeys(t, m, typed("lj ta")...)
	if m.mode != modeConfirm {
		t.Fatalf("mode = %v, want confirm", m.mode)
	}
	task := m.boardSelectedTask()
	m = pressKeys(t, m, enterKey)
	if m.mode != modeConfirm || task.Archived() {
		t.Fatal("enter acted in a tier 1 confirmation")
	}
	m = pressKeys(t, m, typed("n")...)
	if m.mode != modeConfirm {
		t.Fatal("n kept a tier 1 confirmation; only esc keeps")
	}
	m = pressKeys(t, m, typed("y")...)
	if m.mode != modeBoard || !task.Archived() {
		t.Fatal("y did not archive")
	}
}

func TestEscKeepsInBothTiers(t *testing.T) {
	m := confirmModel(t)
	m = pressKeys(t, m, typed("lj tx")...)
	m = pressKeys(t, m, escKey)
	if m.mode != modeBoard || m.board.Tasks[sample.DetailTaskID] == nil {
		t.Fatal("esc did not keep the task")
	}
	m = pressKeys(t, m, typed(" cdtodo")...)
	if m.mode != modeConfirm {
		t.Fatalf("mode = %v, want confirm", m.mode)
	}
	m = pressKeys(t, m, escKey)
	if m.mode != modeBoard || len(m.board.Columns) != 4 {
		t.Fatal("esc did not keep the column")
	}
}

func TestTierTwoActsOnlyAfterExactMatch(t *testing.T) {
	m := confirmModel(t)
	m = pressKeys(t, m, typed("l cd")...)
	for _, attempt := range []string{"", "tod", "Todo", "todo!"} {
		m.confirmInput.SetValue(attempt)
		m = pressKeys(t, m, enterKey)
		if m.mode != modeConfirm || len(m.board.Columns) != 4 {
			t.Fatalf("enter with %q deleted the column", attempt)
		}
	}
	m = pressKeys(t, m, typed("y")...)
	if m.mode != modeConfirm || len(m.board.Columns) != 4 {
		t.Fatal("y acted in a tier 2 confirmation")
	}
	m.confirmInput.SetValue(" todo ")
	m = pressKeys(t, m, enterKey)
	if m.mode != modeBoard || len(m.board.Columns) != 3 {
		t.Fatal("exact (trimmed) match did not delete the column")
	}
}

func TestEmptyColumnStaysTierOne(t *testing.T) {
	m := NewWithOptions(sample.EmptyColumn(sample.Now), nil, "", Options{}).(*model)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	m = pressKeys(t, next.(*model), typed("ll cd")...)
	if m.confirm.name != "" {
		t.Fatal("empty column asked for a typed name")
	}
	m = pressKeys(t, m, typed("y")...)
	if len(m.board.Columns) != 3 {
		t.Fatal("y did not delete the empty column")
	}
}
