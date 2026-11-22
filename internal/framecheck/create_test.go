package framecheck

import (
	"errors"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
	"github.com/nikhilsharma/kanban-tui/internal/sample"
	"github.com/nikhilsharma/kanban-tui/internal/ui"
	"github.com/nikhilsharma/kanban-tui/internal/ui/dusk"
)

type createStore struct {
	saved *domain.Workspace
	err   error
}

func (s *createStore) Load() (*domain.Workspace, error) { return s.saved, nil }

func (s *createStore) Save(w *domain.Workspace) error {
	if s.err == nil {
		s.saved = w
	}
	return s.err
}

func submitCreation(m tea.Model, key string) tea.Model {
	m, cmd := m.Update(keyMsg(key))
	return applyCreationCommand(m, cmd)
}

func applyCreationCommand(m tea.Model, cmd tea.Cmd) tea.Model {
	if cmd == nil {
		return m
	}
	if msg := cmd(); msg != nil {
		if batch, ok := msg.(tea.BatchMsg); ok {
			for _, run := range batch {
				m, _ = m.Update(run())
			}
		} else {
			m, _ = m.Update(msg)
		}
	}
	return m
}

func TestNewTaskRetriesSaveAndSelectsInPickedColumn(t *testing.T) {
	s := &createStore{err: errors.New("disk full")}
	w := sample.Workspace(sample.Now)
	previous := slices.Clone(w.ActiveProject().Board.Order["doing"])
	m := ui.New(w, s, "")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	m = press(m, "l", "space", "t", "n", newTaskTitle, "tab", newTaskLine1, "enter", newTaskLine2, "esc", "s", "j", "enter", ":", "w", "q")
	m = submitCreation(m, "enter")
	screen := func(m tea.Model) []string { return ReadScreen(m.View(), 120, 36, dusk.Dark).Rows() }
	if rows := screen(m); !strings.Contains(rows[26], "✗ save failed: disk full") || !strings.Contains(rows[9], "new · doing") || s.saved != nil {
		t.Fatalf("failed save did not retain draft and target: %q / %q", rows[9], rows[26])
	}
	s.err = nil
	m = submitCreation(press(m, ":", "w", "q"), "enter")
	if s.saved == nil {
		t.Fatal("retry did not save the task")
	}
	board := s.saved.ActiveProject().Board
	order := board.Order["doing"]
	if len(order) != len(previous)+1 || !slices.Equal(order[:len(previous)], previous) {
		t.Fatalf("task was not appended: %v", order)
	}
	id := order[len(previous)]
	task := board.Tasks[id]
	if task.Title != newTaskTitle || task.Description != newTaskLine1+"\n"+newTaskLine2 || task.Status != "doing" || len(task.Whiteboards) != 0 {
		t.Fatalf("saved task = %+v", task)
	}
	feedback := "✓ created " + id[:8] + " in doing"
	if row := screen(m)[34]; !strings.Contains(row, feedback) {
		t.Fatalf("keybar after save: %q", row)
	}
	rows := screen(press(m, "enter"))
	if !strings.Contains(rows[9], newTaskTitle) || strings.Contains(rows[34], feedback) {
		t.Fatalf("next key did not open the new task and clear feedback: %q / %q", rows[9], rows[34])
	}
}

func TestNewTaskQuickAddsTitleToFocusedColumn(t *testing.T) {
	s := &createStore{}
	m := ui.New(sample.Workspace(sample.Now), s, "")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	submitCreation(press(m, "l", "space", "t", "n", newTaskTitle, "tab", "draft description", "tab"), "enter")
	if s.saved == nil {
		t.Fatal("title Enter did not save")
	}
	board := s.saved.ActiveProject().Board
	order := board.Order["todo"]
	if len(order) != 4 {
		t.Fatalf("todo order = %v, want four tasks", order)
	}
	task := board.Tasks[order[3]]
	if task.Title != newTaskTitle || task.Description != "" || task.Status != "todo" {
		t.Fatalf("quick-added task = %+v", task)
	}
}

func TestNewTaskAsksBeforeDiscardingDraft(t *testing.T) {
	m := ui.New(sample.Workspace(sample.Now), nil, "")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	m = press(m, "l", "space", "t", "n", newTaskTitle, "esc", "esc")
	rows := func(m tea.Model) []string { return ReadScreen(m.View(), 120, 36, dusk.Dark).Rows() }
	if footer := rows(m)[26]; !strings.Contains(footer, "discard unsaved changes?") {
		t.Fatalf("discard prompt missing: %q", footer)
	}
	m = press(m, "n")
	if row := rows(m)[9]; !strings.Contains(row, newTaskTitle) {
		t.Fatalf("declining discard lost the draft: %q", row)
	}
	m = press(m, "esc", "y")
	if row := rows(m)[9]; strings.Contains(row, "new · todo") {
		t.Fatalf("confirming discard did not close the dialog: %q", row)
	}
}

func TestNewTaskDoesNotSaveWhileEarlierWriteIsUnresolved(t *testing.T) {
	s := &createStore{}
	m := ui.New(sample.Workspace(sample.Now), s, "")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	m = press(m, "space", "t")
	m, earlierSave := m.Update(keyMsg("l"))
	if earlierSave == nil {
		t.Fatal("moving an existing task did not queue a save")
	}

	m = press(m, "space", "t", "n", newTaskTitle)
	m, cmd := m.Update(keyMsg("enter"))
	// A rejected attempt may start a cursor command, but must not queue a save.
	// Resolve the earlier command before draining the rejected attempt.
	m, _ = m.Update(earlierSave())
	m = applyCreationCommand(m, cmd)
	rows := ReadScreen(m.View(), 120, 36, dusk.Dark).Rows()
	if footer := rows[26]; !strings.Contains(footer, "board is still saving; retry") {
		t.Fatalf("unresolved write did not protect the draft: %q", footer)
	}
	m = submitCreation(m, "enter")
	if rows := ReadScreen(m.View(), 120, 36, dusk.Dark).Rows(); !strings.Contains(rows[34], "✓ created ") {
		t.Fatalf("create did not succeed after earlier save resolved: %q", rows[34])
	}
}
