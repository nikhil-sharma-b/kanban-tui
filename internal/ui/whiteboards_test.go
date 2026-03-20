package ui

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
)

type stubWorkspaceStore struct {
	saved *domain.Workspace
	err   error
}

func (s *stubWorkspaceStore) Load() (*domain.Workspace, error) { return s.saved, s.err }
func (s *stubWorkspaceStore) Save(workspace *domain.Workspace) error {
	s.saved = workspace.Clone()
	return s.err
}

func TestDetailModeOpensWhiteboards(t *testing.T) {
	m := newWhiteboardTestModel(t)
	m.mode = modeDetail

	next, _ := m.updateDetail(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	got := next.(*model)
	if got.mode != modeWhiteboards {
		t.Fatalf("mode = %v, want %v", got.mode, modeWhiteboards)
	}
}

func TestCreateWhiteboardsUsesIncrementingNames(t *testing.T) {
	m := newWhiteboardTestModel(t)
	var launched []string
	var created []string
	origLaunch := launchWhiteboard
	origCreate := createWhiteboardFile
	launchWhiteboard = func(path string) error {
		launched = append(launched, path)
		return nil
	}
	createWhiteboardFile = func(path string) error {
		created = append(created, path)
		return os.WriteFile(path, []byte("stub"), 0o644)
	}
	defer func() {
		launchWhiteboard = origLaunch
		createWhiteboardFile = origCreate
	}()

	if _, cmd := m.createWhiteboard(); cmd == nil {
		t.Fatal("expected save cmd on first create")
	}
	if _, cmd := m.createWhiteboard(); cmd == nil {
		t.Fatal("expected save cmd on second create")
	}

	task := m.selectedTask()
	if len(task.Whiteboards) != 2 {
		t.Fatalf("unexpected whiteboard count: got %d want 2", len(task.Whiteboards))
	}
	if task.Whiteboards[0].Name != "Whiteboard 1" || task.Whiteboards[1].Name != "Whiteboard 2" {
		t.Fatalf("unexpected whiteboard names: %+v", task.Whiteboards)
	}
	if len(launched) != 2 {
		t.Fatalf("unexpected launch count: got %d want 2", len(launched))
	}
	if len(created) != 2 {
		t.Fatalf("unexpected create count: got %d want 2", len(created))
	}
}

func TestRenameWhiteboardFlow(t *testing.T) {
	m := newWhiteboardTestModel(t)
	origLaunch := launchWhiteboard
	origCreate := createWhiteboardFile
	launchWhiteboard = func(path string) error { return nil }
	createWhiteboardFile = func(path string) error { return os.WriteFile(path, []byte("stub"), 0o644) }
	defer func() {
		launchWhiteboard = origLaunch
		createWhiteboardFile = origCreate
	}()

	m.createWhiteboard()
	m.mode = modeWhiteboards

	next, cmd := m.beginRenameWhiteboard()
	got := next.(*model)
	if got.mode != modeWhiteboardRename || cmd == nil {
		t.Fatalf("expected rename mode with blink command")
	}

	got.whiteboardInput.SetValue("Sketches")
	next, cmd = got.renameSelectedWhiteboard()
	got = next.(*model)
	if got.mode != modeWhiteboards || cmd == nil {
		t.Fatalf("expected return to manager with save command")
	}
	if got.selectedTask().Whiteboards[0].Name != "Sketches" {
		t.Fatalf("unexpected whiteboard name: %q", got.selectedTask().Whiteboards[0].Name)
	}
}

func TestOpenSelectedWhiteboardLaunchesPath(t *testing.T) {
	m := newWhiteboardTestModel(t)
	origLaunch := launchWhiteboard
	defer func() { launchWhiteboard = origLaunch }()

	m.selectedTask().Whiteboards = append(m.selectedTask().Whiteboards, domain.Whiteboard{
		ID:   "wb1",
		Name: "Sketches",
		Path: "/tmp/sketches.rnote",
	})

	var launched string
	launchWhiteboard = func(path string) error {
		launched = path
		return nil
	}

	if _, cmd := m.openSelectedWhiteboard(); cmd != nil {
		t.Fatalf("open should not save workspace")
	}
	if launched != "/tmp/sketches.rnote" {
		t.Fatalf("launched path = %q, want %q", launched, "/tmp/sketches.rnote")
	}
}

func TestCreateWhiteboardRespectsEnvDir(t *testing.T) {
	m := newWhiteboardTestModel(t)
	origLaunch := launchWhiteboard
	origCreate := createWhiteboardFile
	launchWhiteboard = func(path string) error { return nil }
	createWhiteboardFile = func(path string) error { return os.WriteFile(path, []byte("stub"), 0o644) }
	defer func() {
		launchWhiteboard = origLaunch
		createWhiteboardFile = origCreate
	}()

	root := filepath.Join(t.TempDir(), "notes")
	t.Setenv("KANBAN_TUI_WHITEBOARD_DIR", root)

	m.createWhiteboard()
	path := m.selectedTask().Whiteboards[0].Path
	if !strings.HasPrefix(path, root) {
		t.Fatalf("whiteboard path %q does not use override root %q", path, root)
	}
}

func TestCreateWhiteboardKeepsEntryOnLaunchFailure(t *testing.T) {
	m := newWhiteboardTestModel(t)
	origLaunch := launchWhiteboard
	origCreate := createWhiteboardFile
	launchWhiteboard = func(path string) error { return errors.New("boom") }
	createWhiteboardFile = func(path string) error { return os.WriteFile(path, []byte("stub"), 0o644) }
	defer func() {
		launchWhiteboard = origLaunch
		createWhiteboardFile = origCreate
	}()

	_, cmd := m.createWhiteboard()
	if cmd == nil {
		t.Fatal("expected save command despite launch failure")
	}
	if len(m.selectedTask().Whiteboards) != 1 {
		t.Fatalf("expected whiteboard to remain linked")
	}
	if m.lastErr == nil {
		t.Fatal("expected launch error to be surfaced")
	}
}

func TestCreateWhiteboardFileFailurePreventsLinking(t *testing.T) {
	m := newWhiteboardTestModel(t)
	origLaunch := launchWhiteboard
	origCreate := createWhiteboardFile
	launchWhiteboard = func(path string) error { return nil }
	createWhiteboardFile = func(path string) error { return errors.New("missing rnote-cli") }
	defer func() {
		launchWhiteboard = origLaunch
		createWhiteboardFile = origCreate
	}()

	_, cmd := m.createWhiteboard()
	if cmd != nil {
		t.Fatal("did not expect save command when file creation fails")
	}
	if len(m.selectedTask().Whiteboards) != 0 {
		t.Fatalf("whiteboard should not be linked when file creation fails")
	}
	if m.lastErr == nil {
		t.Fatal("expected file creation error")
	}
}

func TestDeleteWhiteboardRemovesFileAndEntry(t *testing.T) {
	m := newWhiteboardTestModel(t)
	filePath := filepath.Join(t.TempDir(), "board.rnote")
	if err := os.WriteFile(filePath, []byte("stub"), 0o644); err != nil {
		t.Fatalf("write temp whiteboard: %v", err)
	}
	m.selectedTask().Whiteboards = []domain.Whiteboard{{ID: "wb1", Name: "Board", Path: filePath}}
	m.mode = modeWhiteboards

	origRemove := removeWhiteboardFile
	defer func() { removeWhiteboardFile = origRemove }()
	removeWhiteboardFile = os.Remove

	_, cmd := m.deleteSelectedWhiteboard()
	if cmd == nil {
		t.Fatal("expected save command after delete")
	}
	if len(m.selectedTask().Whiteboards) != 0 {
		t.Fatalf("expected whiteboard entry removed")
	}
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatalf("expected whiteboard file removed, stat err = %v", err)
	}
}

func TestDeleteWhiteboardMissingFileStillUnlinks(t *testing.T) {
	m := newWhiteboardTestModel(t)
	m.selectedTask().Whiteboards = []domain.Whiteboard{{ID: "wb1", Name: "Board", Path: filepath.Join(t.TempDir(), "missing.rnote")}}
	m.mode = modeWhiteboards

	origRemove := removeWhiteboardFile
	defer func() { removeWhiteboardFile = origRemove }()
	removeWhiteboardFile = os.Remove

	_, cmd := m.deleteSelectedWhiteboard()
	if cmd == nil {
		t.Fatal("expected save command after unlink")
	}
	if len(m.selectedTask().Whiteboards) != 0 {
		t.Fatalf("expected whiteboard entry removed")
	}
}

func TestDeleteWhiteboardFileFailurePreventsUnlink(t *testing.T) {
	m := newWhiteboardTestModel(t)
	m.selectedTask().Whiteboards = []domain.Whiteboard{{ID: "wb1", Name: "Board", Path: "/tmp/board.rnote"}}
	m.mode = modeWhiteboards

	origRemove := removeWhiteboardFile
	defer func() { removeWhiteboardFile = origRemove }()
	removeWhiteboardFile = func(path string) error { return errors.New("permission denied") }

	_, cmd := m.deleteSelectedWhiteboard()
	if cmd != nil {
		t.Fatal("unexpected save command on failed file delete")
	}
	if len(m.selectedTask().Whiteboards) != 1 {
		t.Fatalf("whiteboard should remain linked on file delete failure")
	}
	if m.lastErr == nil {
		t.Fatal("expected delete error")
	}
}

func TestRenderWhiteboardsDialogEmptyState(t *testing.T) {
	m := newWhiteboardTestModel(t)
	m.mode = modeWhiteboards
	m.width = 100
	m.height = 30

	view := m.renderWhiteboardsDialog()
	if !strings.Contains(view, "No whiteboards yet") {
		t.Fatalf("expected empty state in whiteboards dialog: %q", view)
	}
}

func TestWhiteboardLaunchCommandUsesCustomCommandWithArgs(t *testing.T) {
	t.Setenv("KANBAN_TUI_WHITEBOARD_CMD", "open -a Rnote --args")

	command, args, err := whiteboardLaunchCommand("/tmp/board.rnote")
	if err != nil {
		t.Fatalf("whiteboardLaunchCommand() error = %v", err)
	}
	if command != "open" {
		t.Fatalf("command = %q, want %q", command, "open")
	}
	if got, want := strings.Join(args, " "), "-a Rnote --args /tmp/board.rnote"; got != want {
		t.Fatalf("args = %q, want %q", got, want)
	}
}

func TestWhiteboardLaunchCommandFallbacks(t *testing.T) {
	t.Setenv("KANBAN_TUI_WHITEBOARD_CMD", "")

	command, args, err := whiteboardLaunchCommand("/tmp/board.rnote")
	if err != nil {
		t.Fatalf("whiteboardLaunchCommand() error = %v", err)
	}

	switch runtime.GOOS {
	case "darwin":
		if command != "open" {
			t.Fatalf("command = %q, want open", command)
		}
		if got, want := strings.Join(args, " "), "-a Rnote --args /tmp/board.rnote"; got != want {
			t.Fatalf("args = %q, want %q", got, want)
		}
	case "windows":
		if command != "cmd" {
			t.Fatalf("command = %q, want cmd", command)
		}
	default:
		if command == "" || len(args) == 0 {
			t.Fatalf("expected non-empty fallback command and args")
		}
	}
}

func newWhiteboardTestModel(t *testing.T) *model {
	t.Helper()

	workspace := domain.NewWorkspace()
	project := workspace.ActiveProject()
	if project == nil {
		t.Fatal("expected active project")
	}
	task, err := project.Board.AddTask("Task", "Description")
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
	return m
}
