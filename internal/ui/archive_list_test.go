package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
	"github.com/nikhilsharma/kanban-tui/internal/sample"
)

func archiveModel(t *testing.T, ws func(now time.Time) *domain.Workspace) *model {
	t.Helper()
	m := NewWithOptions(ws(sample.Now), nil, "", Options{Now: func() time.Time { return sample.Now }}).(*model)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	return pressKeys(t, next.(*model), typed("lj av")...)
}

func TestArchiveRestoreKeepsCursorIndexAndPutsTaskOnTop(t *testing.T) {
	m := archiveModel(t, sample.ArchiveRestore)
	m = pressKeys(t, m, typed("jr")...)
	if m.mode != modeArchive {
		t.Fatalf("mode = %v, want archive", m.mode)
	}
	if got := m.selectedArchivedTask().Title; got != "Spike: SQLite store" {
		t.Fatalf("cursor on %q, want the row that moved into index 1", got)
	}
	top := m.board.Tasks[m.board.Order[domain.StatusDone][0]]
	if top.Title != "Copy task ID keybind" || top.Archived() {
		t.Fatalf("done starts with %q (archived %v), want the restored task", top.Title, top.Archived())
	}
	if !strings.Contains(ansi.Strip(m.View()), "✓ restored “Copy task ID keybind” to done") {
		t.Fatal("restored message missing")
	}
	m = pressKeys(t, m, typed("j")...)
	if strings.Contains(ansi.Strip(m.View()), "restored") {
		t.Fatal("restored message should clear on the next key")
	}
}

func TestArchiveDetailIgnoresEnterAndRestoresWithR(t *testing.T) {
	m := archiveModel(t, sample.Archive)
	m = pressKeys(t, m, typed("j")...)
	m = pressKeys(t, m, enterKey)
	if m.mode != modeArchiveDetail {
		t.Fatalf("↵ opened mode %v, want detail", m.mode)
	}
	m = pressKeys(t, m, enterKey)
	if m.mode != modeArchiveDetail {
		t.Fatalf("↵ in detail left for mode %v", m.mode)
	}
	m = pressKeys(t, m, escKey)
	if m.mode != modeArchive {
		t.Fatalf("esc from detail went to mode %v, want archive", m.mode)
	}
	m = pressKeys(t, m, enterKey)
	m = pressKeys(t, m, typed("r")...)
	if m.mode != modeArchive || m.archiveStatus == "" {
		t.Fatalf("r in detail: mode %v, status %q", m.mode, m.archiveStatus)
	}
}

func TestArchiveSearchEnterOpensMatchAndEscClearsQuery(t *testing.T) {
	m := archiveModel(t, sample.Archive)
	m = pressKeys(t, m, typed("/tab order")...)
	m = pressKeys(t, m, enterKey)
	if m.mode != modeArchiveDetail || m.selectedArchivedTask().Title != "Tab order in dialogs" {
		t.Fatalf("↵ while searching: mode %v", m.mode)
	}
	if !strings.Contains(ansi.Strip(m.View()), "r restore to backlog") {
		t.Fatal("restore fallback hint missing for a deleted source column")
	}
	m = pressKeys(t, m, escKey, escKey)
	if m.mode != modeArchive || m.archiveFilterInput.Value() != "" || m.archiveFiltering {
		t.Fatalf("esc should clear the query and return to the list: mode %v, query %q", m.mode, m.archiveFilterInput.Value())
	}
}

func TestArchiveOldFromListConfirmsThenReturnsToList(t *testing.T) {
	m := archiveModel(t, sample.ArchiveOld)
	m = pressKeys(t, m, typed("a")...)
	if m.mode != modeConfirm {
		t.Fatalf("a opened mode %v, want confirm", m.mode)
	}
	m = pressKeys(t, m, typed("y")...)
	if m.mode != modeArchive || m.archiveStatus != "archived 4 done tasks" {
		t.Fatalf("after y: mode %v, status %q", m.mode, m.archiveStatus)
	}
	if n := len(m.board.Order[domain.StatusDone]); n != 0 {
		t.Fatalf("done still holds %d tasks", n)
	}

	m = archiveModel(t, sample.Workspace)
	m = pressKeys(t, m, typed("a")...)
	if m.mode != modeArchive || m.archiveStatus != "no done tasks past 30 days" {
		t.Fatalf("a with nothing old: mode %v, status %q", m.mode, m.archiveStatus)
	}
}
