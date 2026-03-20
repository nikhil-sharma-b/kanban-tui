package store

import (
	"path/filepath"
	"testing"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
)

func TestSQLiteStoreRoundTrip(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "board.db")
	sqliteStore, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("new sqlite store: %v", err)
	}
	defer sqliteStore.Close()

	workspace := domain.NewWorkspace()
	project := workspace.ActiveProject()
	if project == nil {
		t.Fatal("expected default project")
	}

	first, err := project.Board.AddTask("first", "alpha")
	if err != nil {
		t.Fatalf("add first task: %v", err)
	}
	second, err := project.Board.AddTask("second", "beta")
	if err != nil {
		t.Fatalf("add second task: %v", err)
	}
	if _, err := project.Board.AddWhiteboard(first.ID, "Sketches", "/tmp/sketches.rnote"); err != nil {
		t.Fatalf("add whiteboard: %v", err)
	}
	if !project.Board.ShiftTask(second.ID, 1) {
		t.Fatalf("expected second task to shift")
	}
	workspace.Version = 3

	if err := sqliteStore.Save(workspace); err != nil {
		t.Fatalf("save workspace: %v", err)
	}

	loaded, err := sqliteStore.Load()
	if err != nil {
		t.Fatalf("load workspace: %v", err)
	}

	if loaded.Version != 3 {
		t.Fatalf("unexpected version: got %d want %d", loaded.Version, 3)
	}
	loadedProject := loaded.ActiveProject()
	if loadedProject == nil {
		t.Fatal("expected active project after load")
	}
	if len(loadedProject.Board.Tasks) != 2 {
		t.Fatalf("unexpected task count: got %d want %d", len(loadedProject.Board.Tasks), 2)
	}
	if got := loadedProject.Board.Order[domain.StatusBacklog]; len(got) != 1 || got[0] != first.ID {
		t.Fatalf("unexpected backlog order: %v", got)
	}
	if got := loadedProject.Board.Order[domain.StatusInProgress]; len(got) != 1 || got[0] != second.ID {
		t.Fatalf("unexpected in-progress order: %v", got)
	}
	if got := len(loadedProject.Board.Tasks[first.ID].Whiteboards); got != 1 {
		t.Fatalf("unexpected whiteboard count: got %d want 1", got)
	}
	if got := loadedProject.Board.Tasks[first.ID].Whiteboards[0].Path; got != "/tmp/sketches.rnote" {
		t.Fatalf("unexpected whiteboard path: %q", got)
	}
}

func TestOpenImportsLegacyJSONOnFirstRun(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "board.db")
	legacyPath := filepath.Join(tempDir, "board.json")

	legacyBoard := domain.NewBoard()
	task, err := legacyBoard.AddTask("migrated", "from json")
	if err != nil {
		t.Fatalf("add legacy task: %v", err)
	}

	if err := NewJSONStore(legacyPath).Save(domain.WorkspaceFromBoard(legacyBoard)); err != nil {
		t.Fatalf("save legacy workspace: %v", err)
	}

	boardStore, err := Open(dbPath, legacyPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer boardStore.(*SQLiteStore).Close()

	loaded, err := boardStore.Load()
	if err != nil {
		t.Fatalf("load migrated board: %v", err)
	}

	project := loaded.ActiveProject()
	if project == nil {
		t.Fatal("expected active project")
	}
	if len(project.Board.Tasks) != 1 {
		t.Fatalf("unexpected migrated task count: %d", len(project.Board.Tasks))
	}
	if project.Board.Tasks[task.ID] == nil {
		t.Fatalf("expected migrated task %s to exist", task.ID)
	}
	if got := project.Board.Order[domain.StatusBacklog]; len(got) != 1 || got[0] != task.ID {
		t.Fatalf("unexpected migrated order: %v", got)
	}
}

func TestSQLiteStorePersistsProjectsAndColumns(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "board-columns.db")
	sqliteStore, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("new sqlite store: %v", err)
	}
	defer sqliteStore.Close()

	workspace := domain.NewWorkspace()
	project := workspace.ActiveProject()
	if project == nil {
		t.Fatal("expected default project")
	}
	review, err := project.Board.AddColumn("Review")
	if err != nil {
		t.Fatalf("add column: %v", err)
	}

	other, err := workspace.CreateProject("Work")
	if err != nil {
		t.Fatalf("create second project: %v", err)
	}

	task, err := project.Board.AddTask("reviewed", "looks good")
	if err != nil {
		t.Fatalf("add task: %v", err)
	}
	if !project.Board.MoveTask(task.ID, review, 0) {
		t.Fatalf("move task into custom column")
	}
	if _, err := other.Board.AddTask("ship it", "prod"); err != nil {
		t.Fatalf("add second project task: %v", err)
	}

	if err := sqliteStore.Save(workspace); err != nil {
		t.Fatalf("save workspace: %v", err)
	}

	loaded, err := sqliteStore.Load()
	if err != nil {
		t.Fatalf("load workspace: %v", err)
	}

	if len(loaded.Projects) != 2 {
		t.Fatalf("unexpected projects count: got %d want 2", len(loaded.Projects))
	}
	loadedProject := loaded.ProjectByID(project.ID)
	if loadedProject == nil {
		t.Fatal("expected first project to load")
	}
	if len(loadedProject.Board.Columns) != 4 {
		t.Fatalf("unexpected columns count: got %d want 4", len(loadedProject.Board.Columns))
	}
	if loadedProject.Board.Columns[3] != review {
		t.Fatalf("unexpected custom column order: %v", loadedProject.Board.Columns)
	}
	if got := loadedProject.Board.Order[review]; len(got) != 1 || got[0] != task.ID {
		t.Fatalf("unexpected custom column order after load: %v", got)
	}
	if loaded.ProjectByID(other.ID) == nil {
		t.Fatal("expected second project to load")
	}
}

func TestResolvePathsWithLegacyJSONEnv(t *testing.T) {
	t.Setenv("KANBAN_TUI_DATA_FILE", "/tmp/custom-board.json")

	dbPath, legacyPath, err := ResolvePaths()
	if err != nil {
		t.Fatalf("resolve paths: %v", err)
	}

	if dbPath != "/tmp/custom-board.db" {
		t.Fatalf("unexpected db path: %s", dbPath)
	}
	if legacyPath != "/tmp/custom-board.json" {
		t.Fatalf("unexpected legacy path: %s", legacyPath)
	}
}
