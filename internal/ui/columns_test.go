package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
)

var backspaceKey = tea.KeyMsg{Type: tea.KeyBackspace}

func TestAddDuplicateColumnKeepsDraft(t *testing.T) {
	m := confirmModel(t)
	before := len(m.board.Columns)
	m = pressKeys(t, m, typed(" cntodo")...)
	m = pressKeys(t, m, enterKey)
	if m.mode != modeAddColumn || m.columnInput.Value() != "todo" {
		t.Fatalf("mode %v draft %q: duplicate should keep the draft open", m.mode, m.columnInput.Value())
	}
	if len(m.board.Columns) != before {
		t.Fatal("duplicate changed the board")
	}
	if !strings.Contains(ansi.Strip(m.View()), "column “todo” already exists") {
		t.Fatal("duplicate error not shown")
	}
	m = pressKeys(t, m, backspaceKey)
	if strings.Contains(ansi.Strip(m.View()), "already exists") {
		t.Fatal("editing the draft should clear the error")
	}
	m = pressKeys(t, m, typed("x")...)
	m = pressKeys(t, m, enterKey)
	if m.mode != modeBoard || len(m.board.Columns) != before+1 {
		t.Fatal("unique name not added")
	}
}

func TestRenameColumnInPlace(t *testing.T) {
	m := confirmModel(t)
	m = pressKeys(t, m, typed("l c")...)
	m = pressKeys(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if m.mode != modeRenameColumn {
		t.Fatalf("mode = %v, want rename", m.mode)
	}
	m = pressKeys(t, m, backspaceKey, backspaceKey, backspaceKey, backspaceKey)
	m = pressKeys(t, m, typed("ready")...)
	if !strings.Contains(ansi.Strip(m.View()), "↵ rename") {
		t.Fatal("rename keybar not shown")
	}
	m = pressKeys(t, m, enterKey)
	if m.mode != modeBoard || m.board.StatusIndex(domain.Status("ready")) != 1 {
		t.Fatalf("rename did not keep the column in place: %v", m.board.Columns)
	}
}

func TestRenameColumnEscCancels(t *testing.T) {
	m := confirmModel(t)
	m = pressKeys(t, m, typed("l crx")...)
	m = pressKeys(t, m, escKey)
	if m.mode != modeBoard || m.board.StatusIndex(domain.Status("todo")) != 1 {
		t.Fatal("esc should cancel the rename")
	}
}
