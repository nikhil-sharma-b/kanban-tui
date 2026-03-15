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

	board := domain.NewBoard()
	first, err := board.AddTask("first", "alpha")
	if err != nil {
		t.Fatalf("add first task: %v", err)
	}
	second, err := board.AddTask("second", "beta")
	if err != nil {
		t.Fatalf("add second task: %v", err)
	}
	if !board.ShiftTask(second.ID, 1) {
		t.Fatalf("expected second task to shift")
	}
	board.Version = 3

	if err := sqliteStore.Save(board); err != nil {
		t.Fatalf("save board: %v", err)
	}

	loaded, err := sqliteStore.Load()
	if err != nil {
		t.Fatalf("load board: %v", err)
	}

	if loaded.Version != 3 {
		t.Fatalf("unexpected version: got %d want %d", loaded.Version, 3)
	}
	if len(loaded.Tasks) != 2 {
		t.Fatalf("unexpected task count: got %d want %d", len(loaded.Tasks), 2)
	}
	if got := loaded.Order[domain.StatusBacklog]; len(got) != 1 || got[0] != first.ID {
		t.Fatalf("unexpected backlog order: %v", got)
	}
	if got := loaded.Order[domain.StatusInProgress]; len(got) != 1 || got[0] != second.ID {
		t.Fatalf("unexpected in-progress order: %v", got)
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

	if err := NewJSONStore(legacyPath).Save(legacyBoard); err != nil {
		t.Fatalf("save legacy board: %v", err)
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

	if len(loaded.Tasks) != 1 {
		t.Fatalf("unexpected migrated task count: %d", len(loaded.Tasks))
	}
	if loaded.Tasks[task.ID] == nil {
		t.Fatalf("expected migrated task %s to exist", task.ID)
	}
	if got := loaded.Order[domain.StatusBacklog]; len(got) != 1 || got[0] != task.ID {
		t.Fatalf("unexpected migrated order: %v", got)
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
