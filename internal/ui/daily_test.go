package ui

import (
	"errors"
	"os"
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

func TestDailyPromoteOpensDestinationDialogAndCancels(t *testing.T) {
	m := newDailyTestModel(t)
	next, _ := m.updateBoard(runeKey('D'))
	m = next.(*model)
	task, err := m.board.AddTask("Promote me", "")
	if err != nil {
		t.Fatalf("add task: %v", err)
	}
	m.recalculateVisible()
	m.selectTask(task.ID)

	next, _ = m.updateBoard(runeKey('P'))
	m = next.(*model)
	if m.mode != modeDailyPromote {
		t.Fatalf("mode = %v, want %v", m.mode, modeDailyPromote)
	}
	if len(m.dailyPromotionTargets()) != len(m.workspace.ActiveProject().Board.Statuses()) {
		t.Fatalf("unexpected promotion targets: %v", m.dailyPromotionTargets())
	}

	next, _ = m.updateDailyPromote(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(*model)
	if m.mode != modeBoard || m.board.Tasks[task.ID] == nil {
		t.Fatal("cancel should return to daily board without moving task")
	}
}

func TestDailyPromoteMovesTaskToChosenProjectColumn(t *testing.T) {
	m := newDailyTestModel(t)
	target, err := m.workspace.CreateProject("Work")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	m.activateProject(target.ID)
	next, _ := m.updateBoard(runeKey('D'))
	m = next.(*model)
	task, err := m.board.AddTask("Ship release", "Keep details")
	if err != nil {
		t.Fatalf("add task: %v", err)
	}
	daily := m.project
	m.recalculateVisible()
	m.selectTask(task.ID)

	next, _ = m.updateBoard(runeKey('P'))
	m = next.(*model)
	targets := m.dailyPromotionTargets()
	wantCursor := -1
	for i, destination := range targets {
		if destination.project.ID == target.ID && destination.status == domain.StatusInProgress {
			wantCursor = i
			break
		}
	}
	if wantCursor < 0 {
		t.Fatal("missing Work / In Progress destination")
	}
	m.promotionCursor = wantCursor

	next, cmd := m.updateDailyPromote(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(*model)
	if cmd == nil {
		t.Fatal("expected save command")
	}
	if daily.Board.Tasks[task.ID] == nil {
		t.Fatal("task should remain on daily board until save succeeds")
	}
	next, _ = m.Update(cmd())
	m = next.(*model)
	target = m.workspace.ProjectByID(target.ID)
	promoted := target.Board.Tasks[task.ID]
	if m.mode != modeBoard || !m.onDailyBoard() {
		t.Fatal("promotion should return to daily board")
	}
	if m.board.Tasks[task.ID] != nil {
		t.Fatal("task remained on daily board")
	}
	if promoted == nil || promoted.Status != domain.StatusInProgress || promoted.Title != task.Title || promoted.Description != task.Description {
		t.Fatalf("task not promoted to target column: %+v", promoted)
	}
	if !strings.Contains(m.lastStatus, "Work") || !strings.Contains(m.lastStatus, "In Progress") {
		t.Fatalf("unexpected status: %q", m.lastStatus)
	}
}

func TestDailyPromoteRelocatesWhiteboardFiles(t *testing.T) {
	m := newDailyTestModel(t)
	target := m.workspace.ActiveProject()
	next, _ := m.updateBoard(runeKey('D'))
	m = next.(*model)
	task, err := m.board.AddTask("Promote drawing", "")
	if err != nil {
		t.Fatalf("add task: %v", err)
	}
	oldPath := resolveWhiteboardPath(m.dataPath, m.project.Name, task.ID, "Sketch", ".xopp")
	if err := os.MkdirAll(filepath.Dir(oldPath), 0o755); err != nil {
		t.Fatalf("create whiteboard directory: %v", err)
	}
	if err := os.WriteFile(oldPath, []byte("drawing"), 0o644); err != nil {
		t.Fatalf("create whiteboard: %v", err)
	}
	task.Whiteboards = []domain.Whiteboard{{ID: "wb-1", Name: "Sketch", Path: oldPath, Format: "xopp"}}
	m.recalculateVisible()
	m.selectTask(task.ID)

	next, _ = m.updateBoard(runeKey('P'))
	m = next.(*model)
	next, cmd := m.updateDailyPromote(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(*model)
	next, _ = m.Update(cmd())
	m = next.(*model)
	task = m.workspace.ProjectByID(target.ID).Board.Tasks[task.ID]

	newPath := resolveWhiteboardPath(m.dataPath, target.Name, task.ID, "Sketch", ".xopp")
	if task.Whiteboards[0].Path != newPath {
		t.Fatalf("whiteboard path = %q, want %q", task.Whiteboards[0].Path, newPath)
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("promoted whiteboard missing: %v", err)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("old whiteboard still exists or stat failed: %v", err)
	}
}

func TestDailyPromoteSaveFailureLeavesTaskOnDailyBoard(t *testing.T) {
	m := newDailyTestModel(t)
	m.store.(*stubWorkspaceStore).err = errors.New("disk full")
	next, _ := m.updateBoard(runeKey('D'))
	m = next.(*model)
	task, err := m.board.AddTask("Stay daily", "")
	if err != nil {
		t.Fatalf("add task: %v", err)
	}
	oldPath := resolveWhiteboardPath(m.dataPath, m.project.Name, task.ID, "Notes", ".xopp")
	if err := os.MkdirAll(filepath.Dir(oldPath), 0o755); err != nil {
		t.Fatalf("create whiteboard directory: %v", err)
	}
	if err := os.WriteFile(oldPath, []byte("notes"), 0o644); err != nil {
		t.Fatalf("create whiteboard: %v", err)
	}
	task.Whiteboards = []domain.Whiteboard{{ID: "wb-1", Name: "Notes", Path: oldPath, Format: "xopp"}}
	m.recalculateVisible()
	m.selectTask(task.ID)

	next, _ = m.updateBoard(runeKey('P'))
	m = next.(*model)
	next, cmd := m.updateDailyPromote(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(*model)
	if !m.promotionPending {
		t.Fatal("expected pending promotion")
	}
	next, _ = m.Update(cmd())
	m = next.(*model)

	if m.promotionPending || m.mode != modeDailyPromote {
		t.Fatalf("failed promotion state: pending=%v mode=%v", m.promotionPending, m.mode)
	}
	if m.board.Tasks[task.ID] != task {
		t.Fatal("save failure removed task from daily board")
	}
	if task.Whiteboards[0].Path != oldPath {
		t.Fatalf("whiteboard path = %q, want rollback to %q", task.Whiteboards[0].Path, oldPath)
	}
	if _, err := os.Stat(oldPath); err != nil {
		t.Fatalf("daily whiteboard was not restored: %v", err)
	}
	if m.lastErr == nil || !strings.Contains(m.lastErr.Error(), "disk full") {
		t.Fatalf("unexpected error: %v", m.lastErr)
	}
}
