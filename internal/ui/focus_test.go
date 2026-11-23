package ui

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
	"github.com/nikhilsharma/kanban-tui/internal/sample"
)

func newFocusModel(t *testing.T, ws func(time.Time) *domain.Workspace) *model {
	t.Helper()
	m := New(ws(time.Now()), &stubWorkspaceStore{}, filepath.Join(t.TempDir(), "board.db")).(*model)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	return m
}

func TestFocusZAndEscReturnWithSameTask(t *testing.T) {
	for _, exit := range []string{"z", "esc"} {
		m := newFocusModel(t, sample.Focus)
		press(m, "l", "l", "z")
		if !m.focus.on {
			t.Fatalf("z did not open focus mode")
		}
		press(m, "j", exit)
		if m.focus.on {
			t.Fatalf("%s did not leave focus mode", exit)
		}
		if got := m.boardSelectedTask(); got == nil || got.Title != "Revamp the TUI" {
			t.Fatalf("after %s selected %v, want Revamp the TUI", exit, got)
		}
	}
}

func TestFocusMoveRightAdvancesAndNamesTask(t *testing.T) {
	m := newFocusModel(t, sample.Focus)
	press(m, "l", "l", "z", " ", "t", "l")
	if !m.focus.on {
		t.Fatal("moving right left focus mode")
	}
	moved := m.board.Tasks["e2d81b4a6c094f1e8b3d5a7c9e1f2468"]
	if moved.Status != domain.StatusDone {
		t.Fatalf("moved task is in %s, want done", moved.Status)
	}
	if got := m.boardSelectedTask(); got == nil || got.Title != "Revamp the TUI" {
		t.Fatalf("focus is on %v, want Revamp the TUI", got)
	}
	if want := "“Live reload from CLI writes” → done"; m.toast.text != want || m.toast.kind != StatusSuccess {
		t.Fatalf("toast = %q (%v), want %q", m.toast.text, m.toast.kind, want)
	}
}

func TestFocusStartsTaskFromLeftColumn(t *testing.T) {
	m := newFocusModel(t, sample.FocusNoDoing)
	press(m, "l", "l", "z", "j", " ", "t", "l")
	got := m.boardSelectedTask()
	if got == nil || got.Title != "Undo for move and delete" || got.Status != "doing" {
		t.Fatalf("focus is on %v, want Undo for move and delete in doing", got)
	}
	if !m.focus.on {
		t.Fatal("starting a task left focus mode")
	}
}

func TestFocusChangesNothingOnEnterAndExit(t *testing.T) {
	m := newFocusModel(t, sample.Focus)
	before := m.workspace.Clone()
	press(m, "l", "l", "z", "j", "k", "J", "z")
	if !reflect.DeepEqual(before.ActiveProject().Board.Order, m.board.Order) {
		t.Fatal("focus mode changed the board")
	}
}

func TestFocusEndsOnProjectOrDailySwitch(t *testing.T) {
	for _, keys := range [][]string{{" ", "d", "d"}, {" ", "p", "p"}} {
		m := newFocusModel(t, sample.FourProjects)
		press(m, "l", "l", "z")
		press(m, keys...)
		if keys[1] == "p" {
			m.Update(tea.KeyMsg{Type: tea.KeyDown})
			m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		}
		if m.focus.on {
			t.Fatalf("%v kept focus mode on", keys)
		}
	}
}
