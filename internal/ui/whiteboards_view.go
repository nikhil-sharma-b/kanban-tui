package ui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/nikhilsharma/kanban-tui/internal/termgrid"
	"github.com/nikhilsharma/kanban-tui/internal/ui/dusk"
)

const whiteboardRenameDialogHeight = 5

func (m *model) whiteboardHints(empty bool) []KeyHint {
	if empty {
		return []KeyHint{{"n", "new"}, {"esc", "close"}}
	}
	return []KeyHint{{"j k", "move"}, {"↵", "open"}, {"n", "new"}, {"r", "rename"}, {"x", "delete"}, {"esc", "close"}}
}

// whiteboardRow is the list row label and hint shared with the detail view.
func (m *model) whiteboardRow(i int) (string, string) {
	task := m.selectedTask()
	w := task.Whiteboards[i]
	format := w.Format
	if format == "" {
		format = strings.TrimPrefix(w.Extension(), ".")
	}
	return "◇ " + w.Name, format + " · " + relativeTime(m.clock(), w.UpdatedAt)
}

// whiteboardsDialog is the task's whiteboard list (08/list, 08/empty,
// 08/open-in-editor).
func (m *model) whiteboardsDialog() *termgrid.Grid {
	task := m.selectedTask()
	if task == nil {
		return nil
	}
	p := m.palette
	w := m.dialogW()
	n := len(task.Whiteboards)
	status := m.whiteboardStatusLine()
	fixed := 6 // borders, title, blanks around the list, hints
	if status != "" {
		fixed += 2
	}
	listRows := min(max(1, n), m.height-fixed-1)
	if w < 24 || listRows < 1 {
		return nil
	}
	h := fixed + listRows
	g := newDialog(p, w, h)
	width := w - 4

	label := "whiteboards"
	g.Text(w-2-ansi.StringWidth(label), 1, label, tokenFG(p, dusk.TextDim, false))
	g.Text(2, 1, truncateCell(task.Title, width-ansi.StringWidth(label)-1), tokenFG(p, dusk.TextCard, false))

	y := 3
	if n == 0 {
		g.Text(2, y, "No whiteboards yet", tokenFG(p, dusk.TextMuted, false))
	}
	first := max(0, min(m.whiteboardCursor-listRows+1, n-listRows))
	for i := first; i < n && i < first+listRows; i++ {
		state := ListRowDefault
		if i == m.whiteboardCursor {
			state = ListRowSelected
		}
		name, hint := m.whiteboardRow(i)
		drawListRow(g, p, 0, y+i-first, w-2, name, hint, state)
	}
	if status != "" {
		if kind := m.whiteboardStatusKind(); kind == StatusSuccess {
			// The editor-opened line has no glyph (08/open-in-editor).
			g.Text(2, h-4, truncateCell(status, width), tokenFG(p, dusk.Success, false))
		} else {
			drawStatusMessage(g, p, 2, h-4, width, kind, status)
		}
	}
	drawDialogHints(g, p, 2, h-2, width, m.whiteboardHints(n == 0))
	return g
}

func (m *model) whiteboardStatusLine() string {
	if m.lastErr != nil {
		return m.lastErr.Error()
	}
	return m.whiteboardStatus
}

func (m *model) whiteboardStatusKind() StatusKind {
	if m.lastErr != nil {
		return StatusDanger
	}
	return m.whiteboardKind
}

// whiteboardRenameDialog is the name dialog for rename (08/rename).
func (m *model) whiteboardRenameDialog() *termgrid.Grid {
	p := m.palette
	w := m.dialogW()
	h := whiteboardRenameDialogHeight
	if m.lastErr != nil {
		h++
	}
	if w < 24 || h > m.height {
		return nil
	}
	g := newDialog(p, w, h)
	width := w - 4
	label := "rename whiteboard"
	g.Text(w-2-ansi.StringWidth(label), 1, label, tokenFG(p, dusk.TextDim, false))
	inputWidth := width - ansi.StringWidth(label) - 1
	lines, col := cutInput(m.whiteboardInput.Value(), m.whiteboardInput.Position(), inputWidth)
	drawFocusedInput(g, p, 0, 1, inputWidth+2, 1, lines, IdleInputField, "", &inputPoint{col: col})
	if m.lastErr != nil {
		drawStatusMessage(g, p, 2, 2, width, StatusDanger, m.lastErr.Error())
	}
	drawDialogHints(g, p, 2, h-2, width, []KeyHint{{"↵", "rename"}, {"esc", "cancel"}, {"backspace", "delete character"}})
	return g
}

// placeWhiteboardRenameDialog keeps the dialog still when an error adds a row.
func (m *model) placeWhiteboardRenameDialog(board *termgrid.Grid) string {
	ApplyScrim(board, m.palette)
	if d := m.whiteboardRenameDialog(); d != nil {
		board.Blit((board.W-d.W)/2, (board.H-whiteboardRenameDialogHeight)/2, d)
	}
	return renderGrid(board)
}

func (m *model) launch(path string) error {
	if m.launchWhiteboard != nil {
		return m.launchWhiteboard(path)
	}
	return launchWhiteboard(path)
}
