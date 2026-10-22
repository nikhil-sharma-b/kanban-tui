package ui

import (
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
	"github.com/nikhilsharma/kanban-tui/internal/store"
)

func openTestStore(t *testing.T, dbPath string) *store.SQLiteStore {
	t.Helper()
	s, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// runUntilIdle feeds msg to the model and keeps running returned commands
// until one yields nothing or schedules the next poll.
func runUntilIdle(t *testing.T, m *model, msg tea.Msg) {
	t.Helper()
	for i := 0; i < 10 && msg != nil; i++ {
		_, cmd := m.Update(msg)
		if cmd == nil {
			return
		}
		msg = cmd()
		if _, ok := msg.(externalCheckMsg); ok {
			return
		}
	}
}

// addFromOtherProcess adds a task through a second store on the same file, as
// the CLI would.
func addFromOtherProcess(t *testing.T, dbPath, title string) *domain.Task {
	t.Helper()
	other := openTestStore(t, dbPath)
	workspace, err := other.Load()
	if err != nil {
		t.Fatalf("other load: %v", err)
	}
	task, err := workspace.ActiveProject().Board.AddTask(title, "")
	if err != nil {
		t.Fatalf("other add: %v", err)
	}
	if err := other.Save(workspace); err != nil {
		t.Fatalf("other save: %v", err)
	}
	return task
}

func newLiveReloadModel(t *testing.T) (*model, string) {
	t.Helper()
	previous := externalCheckInterval
	externalCheckInterval = time.Millisecond
	t.Cleanup(func() { externalCheckInterval = previous })

	dbPath := filepath.Join(t.TempDir(), "board.db")
	tuiStore := openTestStore(t, dbPath)
	if err := tuiStore.Save(domain.NewWorkspace()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	workspace, err := tuiStore.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	m := New(workspace, tuiStore, dbPath).(*model)
	m.width, m.height = 120, 40
	return m, dbPath
}

func TestTUIReloadsExternalChanges(t *testing.T) {
	m, dbPath := newLiveReloadModel(t)
	projectID := m.project.ID

	task := addFromOtherProcess(t, dbPath, "from agent")
	runUntilIdle(t, m, externalCheckMsg{})

	if m.board.Tasks[task.ID] == nil {
		t.Fatal("expected external task after reload")
	}
	if m.project.ID != projectID {
		t.Fatalf("expected to stay on project %s, got %s", projectID, m.project.ID)
	}
	if m.lastStatus != "reloaded external changes" {
		t.Fatalf("unexpected status %q", m.lastStatus)
	}
}

func TestTUIDefersReloadWhileDialogOpen(t *testing.T) {
	m, dbPath := newLiveReloadModel(t)
	m.mode = modeCreate

	task := addFromOtherProcess(t, dbPath, "from agent")
	runUntilIdle(t, m, externalCheckMsg{})
	if m.board.Tasks[task.ID] != nil {
		t.Fatal("must not reload while a dialog is open")
	}

	m.mode = modeBoard
	runUntilIdle(t, m, externalCheckMsg{})
	if m.board.Tasks[task.ID] == nil {
		t.Fatal("expected reload once back on the board")
	}
}

func TestTUIConflictingSaveReloadsInsteadOfOverwriting(t *testing.T) {
	m, dbPath := newLiveReloadModel(t)

	external := addFromOtherProcess(t, dbPath, "from agent")
	local, err := m.board.AddTask("from tui", "")
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	runUntilIdle(t, m, m.saveWorkspaceCmd()())

	if m.board.Tasks[external.ID] == nil {
		t.Fatal("external task must survive the conflicting save")
	}
	if m.board.Tasks[local.ID] != nil {
		t.Fatal("expected the conflicting local change to be dropped")
	}
	if m.lastErr == nil {
		t.Fatal("expected a conflict message")
	}
}
