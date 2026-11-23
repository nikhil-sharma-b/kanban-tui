package ui

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
	"github.com/nikhilsharma/kanban-tui/internal/store"
)

// newFeedbackModel has “first” and “second” in todo with “first” selected.
func newFeedbackModel(t *testing.T, st store.WorkspaceStore) (*model, *domain.Task) {
	t.Helper()
	workspace := domain.NewWorkspace()
	board := workspace.ActiveProject().Board
	first, err := board.AddTask("first", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := board.AddTask("second", ""); err != nil {
		t.Fatal(err)
	}
	if st == nil {
		st = &stubWorkspaceStore{}
	}
	m := New(workspace, st, filepath.Join(t.TempDir(), "board.db")).(*model)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	m.selectTask(first.ID)
	return m, first
}

// press sends keys through Update, as the program does.
func press(m *model, keys ...string) {
	for _, k := range keys {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		if k == "esc" {
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		}
		m.Update(msg)
	}
}

func spot(m *model, id string) (domain.Status, int) {
	task := m.board.Tasks[id]
	for i, candidate := range m.board.Order[task.Status] {
		if candidate == id {
			return task.Status, i
		}
	}
	return task.Status, -1
}

func TestUndoPutsMovedTaskBack(t *testing.T) {
	m, task := newFeedbackModel(t, nil)
	status, index := spot(m, task.ID)

	press(m, " ", "t", "l")
	if moved, _ := spot(m, task.ID); moved == status {
		t.Fatal("task did not move")
	}
	if m.toast.hint != undoHint {
		t.Fatalf("move toast %q offers no undo", m.toast.text)
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	if s, i := spot(m, task.ID); s != status || i != index {
		t.Fatalf("after undo task is at %s/%d, want %s/%d", s, i, status, index)
	}
	if cmd == nil {
		t.Fatal("undo did not save")
	}
	if sel := m.boardSelectedTask(); sel == nil || sel.ID != task.ID {
		t.Fatal("undo did not keep the task selected")
	}
}

func TestUndoRestoresArchivedTask(t *testing.T) {
	m, task := newFeedbackModel(t, nil)
	status, index := spot(m, task.ID)

	press(m, " ", "t", "a", "y")
	if !m.board.Tasks[task.ID].Archived() {
		t.Fatal("task was not archived")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	if m.board.Tasks[task.ID].Archived() {
		t.Fatal("undo did not restore the task")
	}
	if s, i := spot(m, task.ID); s != status || i != index {
		t.Fatalf("restored to %s/%d, want %s/%d", s, i, status, index)
	}
}

func TestUndoIsGoneWithItsToast(t *testing.T) {
	for _, tc := range []struct {
		name   string
		expire func(m *model)
	}{
		{"timer", func(m *model) { m.Update(toastExpiredMsg{seq: m.toastSeq}) }},
		{"next key", func(m *model) { press(m, "esc") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, task := newFeedbackModel(t, nil)
			press(m, " ", "t", "l")
			moved, index := spot(m, task.ID)
			tc.expire(m)
			if m.toast.text != "" {
				t.Fatalf("toast %q still shows", m.toast.text)
			}
			m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
			if s, i := spot(m, task.ID); s != moved || i != index {
				t.Fatalf("u without its toast moved the task to %s/%d", s, i)
			}
		})
	}
}

// flakyStore fails the first failures saves.
type flakyStore struct {
	failures int
	attempts int
	saved    *domain.Workspace
}

func (s *flakyStore) Load() (*domain.Workspace, error) { return s.saved, nil }
func (s *flakyStore) Save(w *domain.Workspace) error {
	s.attempts++
	if s.attempts <= s.failures {
		return errors.New("disk full")
	}
	s.saved = w.Clone()
	return nil
}

// settle runs cmd and what it leads to, dropping timers that don't fire soon.
func settle(m *model, cmd tea.Cmd, steps int) {
	for ; steps > 0 && cmd != nil; steps-- {
		var next []tea.Cmd
		for _, msg := range collect(cmd) {
			_, c := m.Update(msg)
			next = append(next, c)
		}
		cmd = tea.Batch(next...)
	}
}

func collect(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	ch := make(chan tea.Msg, 1)
	go func() { ch <- cmd() }()
	select {
	case msg := <-ch:
		if batch, ok := msg.(tea.BatchMsg); ok {
			var out []tea.Msg
			for _, c := range batch {
				out = append(out, collect(c)...)
			}
			return out
		}
		if msg == nil {
			return nil
		}
		return []tea.Msg{msg}
	case <-time.After(200 * time.Millisecond):
		return nil
	}
}

func TestFailedSaveRetriesUntilItSucceeds(t *testing.T) {
	st := &flakyStore{failures: 2}
	m, task := newFeedbackModel(t, st)
	m.saveRetryInterval = time.Millisecond
	m.toastDuration = time.Hour
	from := task.Status

	press(m, " ", "t")
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	settle(m, cmd, 1)
	if m.syncState() != SyncError || m.toast.text != "Could not save · retrying" {
		t.Fatalf("after a failed save: sync %v, toast %q", m.syncState(), m.toast.text)
	}
	if got := RenderAppHeader(m.palette, 116, m.appHeader()); !bytes.Contains([]byte(got), []byte("save failed · retrying")) {
		t.Fatal("header does not read save failed · retrying")
	}

	settle(m, tea.Tick(time.Millisecond, func(time.Time) tea.Msg { return saveRetryMsg{} }), 8)
	if st.attempts != 3 {
		t.Fatalf("save attempts = %d, want 3", st.attempts)
	}
	if m.syncState() != SyncLive {
		t.Fatalf("sync = %v after a save succeeded, want live", m.syncState())
	}
	if st.saved == nil || st.saved.ActiveProject().Board.Tasks[task.ID].Status == from {
		t.Fatal("the retried save did not carry the move")
	}
}

func TestUnreadableStoreIsNeverWritten(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "board.db")
	garbage := bytes.Repeat([]byte("not a database\n"), 512)
	if err := os.WriteFile(dbPath, garbage, 0o644); err != nil {
		t.Fatal(err)
	}
	reopen := func() (store.WorkspaceStore, *domain.Workspace, error) {
		st, err := store.Open(dbPath, filepath.Join(dir, "board.json"))
		if err != nil {
			return nil, nil, err
		}
		workspace, err := st.Load()
		if err != nil {
			if c, ok := st.(interface{ Close() error }); ok {
				c.Close()
			}
			return nil, nil, err
		}
		return st, workspace, nil
	}
	_, _, err := reopen()
	if err == nil {
		t.Fatal("garbage file opened")
	}

	var m tea.Model = NewUnreadable(dbPath, dbPath, err, reopen, Options{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	for _, k := range []string{"r", "?", "r", "j", "enter"} {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
	}
	if _, ok := m.(*unreadableModel); !ok {
		t.Fatalf("left the unreadable screen for %T", m)
	}
	if got, _ := os.ReadFile(dbPath); !bytes.Equal(got, garbage) {
		t.Fatal("the unreadable store was written")
	}
	if _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}); cmd == nil || cmd() != tea.Quit() {
		t.Fatal("q does not quit")
	}

	// Once the file is fixed, r opens the board.
	if err := os.Remove(dbPath); err != nil {
		t.Fatal(err)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	board, ok := m.(*model)
	if !ok {
		t.Fatalf("r after the fix shows %T, want the board", m)
	}
	if board.width != 120 || board.height != 36 {
		t.Fatalf("board size %dx%d, want 120x36", board.width, board.height)
	}
}
