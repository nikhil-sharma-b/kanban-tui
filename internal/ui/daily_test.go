package ui

import (
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
)

func newDailyTestModel(t *testing.T) *model {
	t.Helper()

	workspace := domain.NewWorkspace()
	m := New(workspace, &stubWorkspaceStore{}, filepath.Join(t.TempDir(), "board.db")).(*model)
	m.width = 120
	m.height = 40
	m.recalculateVisible()
	return m
}

func runeKey(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func TestDailyKeyTogglesBoard(t *testing.T) {
	m := newDailyTestModel(t)
	start := m.project.ID

	next, _ := m.updateBoard(runeKey('D'))
	got := next.(*model)
	if !got.onDailyBoard() {
		t.Fatal("expected daily board to be active")
	}
	if cols := got.board.Statuses(); len(cols) != 3 || cols[0] != domain.StatusWaiting {
		t.Fatalf("unexpected daily columns: %v", cols)
	}

	next, _ = got.updateBoard(runeKey('D'))
	got = next.(*model)
	if got.onDailyBoard() {
		t.Fatal("expected to leave the daily board")
	}
	if got.project.ID != start {
		t.Fatalf("returned to %s, want %s", got.project.ID, start)
	}
}

func TestDailyDoneArchivesTaskAndHidesIt(t *testing.T) {
	m := newDailyTestModel(t)
	next, _ := m.updateBoard(runeKey('D'))
	m = next.(*model)

	task, err := m.board.AddTask("Reply to mail", "")
	if err != nil {
		t.Fatalf("add task: %v", err)
	}
	m.recalculateVisible()
	m.selectTask(task.ID)

	next, cmd := m.updateBoard(tea.KeyMsg{Type: tea.KeySpace})
	m = next.(*model)
	if cmd == nil {
		t.Fatal("expected a save command")
	}
	if !task.Archived() {
		t.Fatal("expected task to be archived")
	}
	if len(m.visible[domain.StatusWaiting]) != 0 {
		t.Fatalf("task still visible: %v", m.visible[domain.StatusWaiting])
	}
	if len(m.board.ArchivedTasks()) != 1 {
		t.Fatal("expected task to stay in the archive")
	}
}

func TestDailyClearAsksConfirmationThenDeletesEverything(t *testing.T) {
	m := newDailyTestModel(t)
	next, _ := m.updateBoard(runeKey('D'))
	m = next.(*model)

	if _, err := m.board.AddTask("one", ""); err != nil {
		t.Fatalf("add task: %v", err)
	}
	if _, err := m.board.AddTask("two", ""); err != nil {
		t.Fatalf("add task: %v", err)
	}
	m.recalculateVisible()

	next, _ = m.updateBoard(runeKey('X'))
	m = next.(*model)
	if m.mode != modeConfirm {
		t.Fatalf("mode = %v, want %v", m.mode, modeConfirm)
	}
	if !strings.Contains(m.confirmMsg, "Clear the daily board") {
		t.Fatalf("unexpected confirm message: %q", m.confirmMsg)
	}

	next, cmd := m.updateConfirm(runeKey('y'))
	m = next.(*model)
	if cmd == nil {
		t.Fatal("expected a save command")
	}
	if m.mode != modeBoard {
		t.Fatalf("mode = %v, want %v", m.mode, modeBoard)
	}
	if len(m.board.Tasks) != 0 {
		t.Fatalf("tasks left: %d", len(m.board.Tasks))
	}
}

func TestDailyClearDoesNotTouchOtherProjects(t *testing.T) {
	m := newDailyTestModel(t)
	regular := m.project
	if _, err := regular.Board.AddTask("keep me", ""); err != nil {
		t.Fatalf("add task: %v", err)
	}

	next, _ := m.updateBoard(runeKey('D'))
	m = next.(*model)
	if _, err := m.board.AddTask("daily", ""); err != nil {
		t.Fatalf("add task: %v", err)
	}
	m.recalculateVisible()

	next, _ = m.updateBoard(runeKey('X'))
	m = next.(*model)
	next, _ = m.updateConfirm(runeKey('y'))
	m = next.(*model)

	if len(regular.Board.Tasks) != 1 {
		t.Fatalf("regular project tasks = %d, want 1", len(regular.Board.Tasks))
	}
}

func TestDailyBoardRejectsColumnEdits(t *testing.T) {
	m := newDailyTestModel(t)
	next, _ := m.updateBoard(runeKey('D'))
	m = next.(*model)

	for _, r := range []rune{'c', 'r', 'd', 'H', 'L'} {
		next, _ = m.updateBoard(runeKey(r))
		m = next.(*model)
		if m.mode != modeBoard {
			t.Fatalf("key %q changed mode to %v", r, m.mode)
		}
		if cols := m.board.Statuses(); len(cols) != 3 {
			t.Fatalf("key %q changed columns: %v", r, cols)
		}
	}
	if m.lastStatus != "daily board columns are fixed" {
		t.Fatalf("unexpected status: %q", m.lastStatus)
	}
}

func TestProjectManagerHidesDailyBoard(t *testing.T) {
	m := newDailyTestModel(t)
	for _, project := range m.filteredProjects() {
		if project.Daily {
			t.Fatal("daily board should not be listed in the project manager")
		}
	}
}
