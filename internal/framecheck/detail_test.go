package framecheck

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
	"github.com/nikhilsharma/kanban-tui/internal/sample"
	"github.com/nikhilsharma/kanban-tui/internal/ui"
	"github.com/nikhilsharma/kanban-tui/internal/ui/dusk"
)

func press(m tea.Model, keys ...string) tea.Model {
	for _, key := range keys {
		m, _ = m.Update(keyMsg(key))
	}
	return m
}

func TestDetailSizeDoesNotDependOnDescription(t *testing.T) {
	for _, desc := range []string{"", "one line", strings.Repeat("Long description.\n", 100)} {
		scn := Scenarios["03/view"]
		scn.Workspace = func(now time.Time) *domain.Workspace {
			w := sample.TaskDetail(now)
			w.ActiveProject().Board.Tasks[sample.DetailTaskID].Description = desc
			return w
		}
		g := ReadScreen(scn.Run(), 120, 36, dusk.Dark)
		for _, corner := range []struct {
			x, y int
			ch   rune
		}{{30, 8, '╭'}, {89, 8, '╮'}, {30, 27, '╰'}, {89, 27, '╯'}} {
			if c := g.At(corner.x, corner.y); c.Ch != corner.ch {
				t.Errorf("description length %d: (%d,%d) got %q, want %q", len(desc), corner.x, corner.y, c.Ch, corner.ch)
			}
		}
	}
}

func TestCopyTaskIDUsesFullIDAndFeedbackClears(t *testing.T) {
	for _, tc := range []struct {
		name, message string
		err           error
	}{
		{"success", "✓ copied a3f91c07", nil},
		{"error", "✗ could not copy ID", errors.New("clipboard unavailable")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var copied string
			m := ui.NewWithOptions(sample.TaskDetail(sample.Now), nil, "", ui.Options{
				WriteClipboard: func(text string) error { copied = text; return tc.err },
			})
			m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
			m = press(m, "l", "j", "enter", "y")
			footer := func(m tea.Model) string { return ReadScreen(m.View(), 120, 36, dusk.Dark).Rows()[26] }
			if got := footer(m); !strings.Contains(got, tc.message) || strings.Contains(got, "esc close") {
				t.Fatalf("copy footer = %q", got)
			}
			if copied != sample.DetailTaskID {
				t.Errorf("clipboard received %q, want full task ID %s", copied, sample.DetailTaskID)
			}
			if got := footer(press(m, "K")); !strings.Contains(got, "esc close") || strings.Contains(got, tc.message) {
				t.Errorf("footer after next key = %q", got)
			}
		})
	}
}

func TestTaskDeletionRequiresY(t *testing.T) {
	for _, keys := range [][]string{{"enter", "esc"}, {"enter", "y"}} {
		w := sample.Workspace(sample.Now)
		board := w.ActiveProject().Board
		m := ui.New(w, nil, "")
		m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
		m = press(m, "l", "j", "space", "t", "x", keys[0])
		if board.Tasks[sample.DetailTaskID] == nil {
			t.Fatal("Enter deleted the task")
		}
		press(m, keys[1])
		if deleted := board.Tasks[sample.DetailTaskID] == nil; deleted != (keys[1] == "y") {
			t.Errorf("after %s: deleted = %t", keys[1], deleted)
		}
	}
}
