package ui

import (
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
)

func newArchiveTestModel(t *testing.T) (*model, *domain.Task) {
	t.Helper()

	workspace := domain.NewWorkspace()
	project := workspace.ActiveProject()
	if project == nil {
		t.Fatal("expected active project")
	}
	task, err := project.Board.AddTask("Ship feature", "Details")
	if err != nil {
		t.Fatalf("add task: %v", err)
	}

	store := &stubWorkspaceStore{}
	m := New(workspace, store, filepath.Join(t.TempDir(), "board.db")).(*model)
	m.width = 120
	m.height = 40
	m.project = project
	m.board = project.Board
	m.recalculateVisible()
	m.selectTask(task.ID)
	return m, task
}

func TestArchiveKeyOpensConfirmation(t *testing.T) {
	m, _ := newArchiveTestModel(t)

	next, _ := m.updateBoard(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	got := next.(*model)
	if got.mode != modeConfirm {
		t.Fatalf("mode = %v, want %v", got.mode, modeConfirm)
	}
	if !strings.Contains(got.confirmMsg, "Archive task") {
		t.Fatalf("unexpected confirm message: %q", got.confirmMsg)
	}
}

func TestConfirmArchivesSelectedTaskAndSaves(t *testing.T) {
	m, task := newArchiveTestModel(t)

	next, _ := m.updateBoard(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	got := next.(*model)
	next, cmd := got.updateConfirm(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	got = next.(*model)

	if got.mode != modeBoard {
		t.Fatalf("mode = %v, want %v", got.mode, modeBoard)
	}
	if !got.board.Tasks[task.ID].Archived() {
		t.Fatal("expected task to be archived")
	}
	if cmd == nil {
		t.Fatal("expected save cmd after archive")
	}
	if got.lastStatus != "archived task" {
		t.Fatalf("unexpected status: %q", got.lastStatus)
	}
}

func TestArchiveViewListsArchivedTask(t *testing.T) {
	m, task := newArchiveTestModel(t)
	if _, err := m.board.ArchiveTask(task.ID); err != nil {
		t.Fatalf("archive: %v", err)
	}
	m.recalculateVisible()

	next, _ := m.updateBoard(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	got := next.(*model)
	if got.mode != modeArchive {
		t.Fatalf("mode = %v, want %v", got.mode, modeArchive)
	}

	view := got.renderArchiveDialog()
	if !strings.Contains(view, "Ship feature") {
		t.Fatalf("expected archived task in view, got:\n%s", view)
	}
	if !strings.Contains(view, "Backlog") {
		t.Fatalf("expected original column in view, got:\n%s", view)
	}
}

func TestArchiveViewRestoreReturnsTaskToBoard(t *testing.T) {
	m, task := newArchiveTestModel(t)
	if _, err := m.board.ArchiveTask(task.ID); err != nil {
		t.Fatalf("archive: %v", err)
	}
	m.recalculateVisible()
	m.mode = modeArchive

	next, cmd := m.updateArchive(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	got := next.(*model)

	if got.board.Tasks[task.ID].Archived() {
		t.Fatal("expected task restored")
	}
	if order := got.board.Order[domain.StatusBacklog]; len(order) != 1 || order[0] != task.ID {
		t.Fatalf("unexpected backlog order after restore: %v", order)
	}
	if cmd == nil {
		t.Fatal("expected save cmd after restore")
	}
	if got.lastStatus != "restored task" {
		t.Fatalf("unexpected status: %q", got.lastStatus)
	}
}

func TestArchiveViewFilterMatchesTitleAndDescription(t *testing.T) {
	m, task := newArchiveTestModel(t)
	other, err := m.board.AddTask("Other work", "unrelated")
	if err != nil {
		t.Fatalf("add task: %v", err)
	}
	if _, err := m.board.ArchiveTask(task.ID); err != nil {
		t.Fatalf("archive: %v", err)
	}
	if _, err := m.board.ArchiveTask(other.ID); err != nil {
		t.Fatalf("archive other: %v", err)
	}

	m.archiveFilterInput.SetValue("details")
	archived := m.filteredArchivedTasks()
	if len(archived) != 1 || archived[0].ID != task.ID {
		t.Fatalf("expected description match only, got %d results", len(archived))
	}
}

func TestBulkArchiveConfirmMessage(t *testing.T) {
	m, _ := newArchiveTestModel(t)

	next, _ := m.updateBoard(tea.KeyMsg{Type: tea.KeyCtrlA})
	got := next.(*model)
	if got.mode != modeConfirm {
		t.Fatalf("mode = %v, want %v", got.mode, modeConfirm)
	}
	if got.confirmMsg != "Archive Done tasks not updated in more than 30 days?" {
		t.Fatalf("unexpected confirm message: %q", got.confirmMsg)
	}
}

func TestBoardSearchExcludesArchivedTasks(t *testing.T) {
	m, task := newArchiveTestModel(t)
	if _, err := m.board.ArchiveTask(task.ID); err != nil {
		t.Fatalf("archive: %v", err)
	}

	m.filter = "ship"
	m.recalculateVisible()
	for _, status := range m.board.Statuses() {
		if len(m.visible[status]) != 0 {
			t.Fatalf("expected no visible tasks in %s, got %v", status, m.visible[status])
		}
	}
}
