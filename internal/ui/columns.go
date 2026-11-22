package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/nikhilsharma/kanban-tui/internal/termgrid"
	"github.com/nikhilsharma/kanban-tui/internal/ui/dusk"
)

// columnHeaderFieldWidth is the cells the editing header gives its field
// (focus bar, gap and value) before the count (00 Column header, editing).
const columnHeaderFieldWidth = 22

// addColumnDialogHeight is the add dialog without its error row; the dialog
// stays anchored at this height when the error row appears.
const addColumnDialogHeight = 5

var (
	addColumnHints    = []KeyHint{{"↵", "add"}, {"esc", "cancel"}, {"backspace", "delete character"}}
	renameColumnHints = []KeyHint{{"↵", "rename"}, {"esc", "cancel"}, {"backspace", "delete character"}}
)

// cutInput returns the visible slice of a one-line value and the caret column.
func cutInput(value string, pos, width int) ([]string, int) {
	runes := []rune(value)
	pos = min(pos, len(runes))
	before := ansi.StringWidth(string(runes[:pos]))
	left := max(0, before-width+1)
	if value == "" {
		return nil, before - left
	}
	return []string{ansi.Cut(value, left, left+width)}, before - left
}

// drawColumnHeaderEditing draws the Column header in its editing state: the
// draft in a focused field, then the count.
func drawColumnHeaderEditing(g *termgrid.Grid, p dusk.Palette, x, y int, value string, pos, count int) {
	lines, col := cutInput(value, pos, columnHeaderFieldWidth-3)
	drawFocusedInput(g, p, x, y, columnHeaderFieldWidth, 1, lines, IdleInputField, "", &inputPoint{col: col})
	g.Text(x+columnHeaderFieldWidth, y, fmt.Sprintf("· %d", count), tokenFG(p, dusk.Accent, true))
}

// RenderColumnHeaderEditing draws the Column header component (State=editing).
func RenderColumnHeaderEditing(p dusk.Palette, value string, count int) string {
	label := fmt.Sprintf("· %d", count)
	g := termgrid.New(columnHeaderFieldWidth+len([]rune(label)), 1, tokenBG(p, dusk.Bg))
	drawColumnHeaderEditing(g, p, 0, 0, value, len([]rune(value)), count)
	return renderGrid(g)
}

// addColumnDialog is the 00 Dialog for SPC c n: the name field, a duplicate
// error under it, and the footer.
func (m *model) addColumnDialog() *termgrid.Grid {
	p := m.palette
	w := min(dialogMaxWidth, m.width-8)
	h := addColumnDialogHeight
	if m.columnErr != nil {
		h++
	}
	if w < 16 || h > m.height {
		return nil
	}
	g := newDialog(p, w, h)
	width := w - 5
	label := "new column"
	g.Text(w-2-ansi.StringWidth(label), 1, label, tokenFG(p, dusk.TextDim, false))
	inputWidth := width - ansi.StringWidth(label) - 1
	lines, col := cutInput(m.columnInput.Value(), m.columnInput.Position(), inputWidth)
	drawFocusedInput(g, p, 1, 1, inputWidth+2, 1, lines, IdleInputField, "", &inputPoint{col: col})
	if m.columnErr != nil {
		drawStatusMessage(g, p, 3, 2, width, StatusDanger, m.columnErr.Error())
	}
	drawDialogHints(g, p, 3, h-2, width, addColumnHints)
	return g
}

// placeAddColumnDialog keeps the field on the row 06/add puts it, with or
// without the error row.
func (m *model) placeAddColumnDialog(board *termgrid.Grid) string {
	ApplyScrim(board, m.palette)
	if d := m.addColumnDialog(); d != nil {
		board.Blit((board.W-d.W)/2, (board.H-addColumnDialogHeight)/2, d)
	}
	return renderGrid(board)
}

func (m *model) updateColumnDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.closeColumnEdit()
		return m, nil
	case "enter":
		if m.mode == modeRenameColumn {
			renamed, err := m.board.RenameColumn(string(m.columnRename), m.columnInput.Value())
			if err != nil {
				m.columnErr = err
				return m, nil
			}
			m.closeColumnEdit()
			m.lastStatus = fmt.Sprintf("renamed %s", renamed.Title())
			m.activeColumn = m.board.StatusIndex(renamed)
		} else {
			status, err := m.board.AddColumn(m.columnInput.Value())
			if err != nil {
				m.columnErr = err
				return m, nil
			}
			m.closeColumnEdit()
			m.lastStatus = fmt.Sprintf("added %s", status.Title())
			m.activeColumn = m.board.StatusIndex(status)
		}
		m.ensureColumnState()
		m.recalculateVisible()
		m.syncAllScroll()
		return m, m.saveWorkspaceCmd()
	}
	before := m.columnInput.Value()
	var cmd tea.Cmd
	m.columnInput, cmd = m.columnInput.Update(msg)
	if m.columnInput.Value() != before {
		m.columnErr = nil
	}
	return m, cmd
}

func (m *model) closeColumnEdit() {
	m.mode = modeBoard
	m.columnInput.Blur()
	m.columnErr, m.lastErr = nil, nil
	m.columnRename = ""
}
