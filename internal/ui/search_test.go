package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
	"github.com/nikhilsharma/kanban-tui/internal/sample"
)

type countingStore struct{ saves int }

func (s *countingStore) Load() (*domain.Workspace, error) { return nil, nil }
func (s *countingStore) Save(*domain.Workspace) error     { s.saves++; return nil }

func searchModel(t *testing.T, keys ...string) (*model, *countingStore) {
	t.Helper()
	s := &countingStore{}
	var m tea.Model = NewWithOptions(sample.SearchFilter(sample.Now), s, "", Options{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	for _, k := range keys {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		switch k {
		case "esc":
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		case "enter":
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		case "backspace":
			msg = tea.KeyMsg{Type: tea.KeyBackspace}
		case "space":
			msg = tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
		}
		m, _ = m.Update(msg)
	}
	return m.(*model), s
}

func TestSearchAppliesEvenWithNoMatches(t *testing.T) {
	m, _ := searchModel(t, "/", "z", "z", "z", "enter")
	if m.mode != modeBoard || m.filter != "zzz" {
		t.Fatalf("mode %v filter %q, want board with zzz applied", m.mode, m.filter)
	}
	if matches, total := m.filterCounts(); matches != 0 || total != 14 {
		t.Fatalf("counts %d of %d, want 0 of 14", matches, total)
	}
}

func TestSearchMatchesDescriptionsCaseInsensitively(t *testing.T) {
	m, _ := searchModel(t, "/", "r", "e", "p", "a", "i", "n", "t")
	if matches, _ := m.filterCounts(); matches != 1 {
		t.Fatalf("matches = %d, want the one task whose description says Repaint", matches)
	}
}

func TestEscInSearchRestoresPreviousQuery(t *testing.T) {
	m, _ := searchModel(t, "/", "C", "L", "I", "enter", "/", "x", "esc")
	if m.mode != modeBoard || m.filter != "CLI" {
		t.Fatalf("mode %v filter %q, want board with CLI", m.mode, m.filter)
	}
	m, _ = searchModel(t, "l", "j", "/", "C", "L", "I", "esc")
	if m.filter != "" || m.boardSelectedTask().ID != sample.DetailTaskID {
		t.Fatalf("esc from a first search kept %q or lost the selection", m.filter)
	}
}

func TestApplyingBlankQueryClears(t *testing.T) {
	m, _ := searchModel(t, "/", "C", "L", "I", "enter", "/", "backspace", "backspace", "backspace", "space", "enter")
	if m.filter != "" || m.lastStatus != "filter cleared" {
		t.Fatalf("filter %q status %q, want cleared", m.filter, m.lastStatus)
	}
}

func TestEscOnBoardClearsFilterAndKeepsPlaceAfterMoving(t *testing.T) {
	m, _ := searchModel(t, "l", "j", "/", "C", "L", "I", "enter", "l", "esc")
	if m.filter != "" || m.boardSelectedTask().Title != "Agent skill for CLI" {
		t.Fatalf("filter %q selection %q", m.filter, m.boardSelectedTask().Title)
	}
}

func TestFilterIsNeverSaved(t *testing.T) {
	m, s := searchModel(t, "/", "C", "L", "I", "enter", "esc", "/", "x", "enter")
	if s.saves != 0 {
		t.Fatalf("filtering saved the store %d times", s.saves)
	}
	_ = m
}
