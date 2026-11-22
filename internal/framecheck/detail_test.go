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

func TestDetailEditCanKeepOrDiscardDraft(t *testing.T) {
	for _, decision := range []string{"y", "n"} {
		t.Run(decision, func(t *testing.T) {
			w := sample.TaskDetail(sample.Now)
			task := w.ActiveProject().Board.Tasks[sample.DetailTaskID]
			original := task.Title
			m := ui.New(w, nil, "")
			m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
			m = press(m, "l", "j", "enter", "e", "!", "esc", "esc", decision)
			if task.Title != original {
				t.Fatalf("answering %s unexpectedly saved %q", decision, task.Title)
			}
			if decision == "n" {
				m = press(m, ":", "w", "q", "enter")
				if task.Title != original+"!" {
					t.Errorf("save after keeping draft: title = %q", task.Title)
				}
			} else {
				titleRow := ReadScreen(m.View(), 120, 36, dusk.Dark).Rows()[9]
				if !strings.Contains(titleRow, original) || strings.Contains(titleRow, original+"!") {
					t.Errorf("view after discard: %q", titleRow)
				}
			}
		})
	}
}

func TestDetailEditScrollFollowsVimCursor(t *testing.T) {
	for _, size := range []tea.WindowSizeMsg{{Width: 120, Height: 36}, {Width: 80, Height: 24}} {
		w := sample.TaskDetail(sample.Now)
		task := w.ActiveProject().Board.Tasks[sample.DetailTaskID]
		task.Description = strings.Repeat("A long paragraph that wraps across several terminal lines. ", 13) + "\nlast line"
		m := ui.New(w, nil, "")
		m, _ = m.Update(size)
		m = press(m, "l", "j", "enter", "e", "tab", "esc", "G")
		g := ReadScreen(m.View(), size.Width, size.Height, dusk.Dark)
		rows := 8
		if size.Height == 24 {
			rows = 7
		}
		x, y := (size.Width-60)/2+3, (size.Height-20)/2+3+rows-1
		if c := g.At(x, y); c.Ch != 'l' || c.BG != "text" {
			t.Fatalf("%dx%d: last line caret = %+v, row %q", size.Width, size.Height, c, g.Rows()[y])
		}
		m = press(m, "A", "!", "esc", ":", "w", "q", "enter")
		if !strings.HasSuffix(task.Description, "\nlast line!") {
			t.Errorf("%dx%d: Vim append did not edit the last line", size.Width, size.Height)
		}
	}
}

func TestDetailStatusMenuSetsOnlyOnEnter(t *testing.T) {
	w := sample.TaskDetail(sample.Now)
	task := w.ActiveProject().Board.Tasks[sample.DetailTaskID]
	m := ui.New(w, nil, "")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	m = press(m, "l", "j", "enter", "s", "j", "esc")
	if task.Status != "todo" {
		t.Fatalf("cancel changed status to %s", task.Status)
	}
	m = press(m, "s", "j", "enter")
	if task.Status != "doing" {
		t.Errorf("set: got %s, want doing", task.Status)
	}
	titleRow := ReadScreen(m.View(), 120, 36, dusk.Dark).Rows()[9]
	if !strings.Contains(titleRow, task.Title) || !strings.Contains(titleRow, "doing ·") {
		t.Errorf("detail lost moved task: %q", titleRow)
	}
}

func TestDetailRetainsTaskWhenSaveRemovesItFromFilter(t *testing.T) {
	w := sample.TaskDetail(sample.Now)
	m := ui.New(w, nil, "")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	// The filter selects its first match, so no column move is needed.
	m = press(m, "/", "Undo for move and delete", "enter", "enter", "e", "esc", "S", "Rewritten", "esc", ":", "w", "enter", "esc")
	if titleRow := ReadScreen(m.View(), 120, 36, dusk.Dark).Rows()[9]; !strings.Contains(titleRow, "Rewritten") {
		t.Errorf("detail lost filtered-out task: %q", titleRow)
	}
}

func TestDetailCaretAfterFullWidthDescriptionLine(t *testing.T) {
	w := sample.TaskDetail(sample.Now)
	task := w.ActiveProject().Board.Tasks[sample.DetailTaskID]
	task.Description = strings.Repeat("x", 55) + "\nlast line"
	m := ui.New(w, nil, "")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	m = press(m, "l", "j", "enter", "e", "tab", "esc", "$")
	g := ReadScreen(m.View(), 120, 36, dusk.Dark)
	if end, caret := g.At(87, 11), g.At(33, 12); end.Ch != 'x' || end.BG == "text" || caret.Ch != ' ' || caret.BG != "text" {
		t.Fatalf("full-line end = %+v, wrapped caret = %+v", end, caret)
	}
	m = press(m, "j")
	if caret := ReadScreen(m.View(), 120, 36, dusk.Dark).At(33, 13); caret.Ch != 'l' || caret.BG != "text" {
		t.Fatalf("vertical motion did not reach next visual row: %+v", caret)
	}
}

func TestEditingDescriptionKeepsLongExistingTitle(t *testing.T) {
	w := sample.TaskDetail(sample.Now)
	task := w.ActiveProject().Board.Tasks[sample.DetailTaskID]
	title := strings.Repeat("Long title ", 12) + "end"
	task.Title = title
	m := ui.New(w, nil, "")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	m = press(m, "l", "j", "enter", "e", "tab", "!", "esc", ":", "w", "q", "enter")
	if task.Title != title || !strings.HasPrefix(task.Description, "!") {
		t.Errorf("editing description changed existing title: %q", task.Title)
	}
}
