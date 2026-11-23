package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
	"github.com/nikhilsharma/kanban-tui/internal/termgrid"
	"github.com/nikhilsharma/kanban-tui/internal/ui/dusk"
)

func (m *model) resizeDetailEditor() {
	w, _ := m.dialogSize(dialogMaxHeight)
	m.descInput.SetWidth(max(1, w-4))
}

func (m *model) updateDetailEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.editDiscard {
		switch msg.String() {
		case "y":
			m.closeDetailEditor()
		case "n", "esc":
			m.editDiscard = false
		}
		return m, nil
	}
	m.detailStatus = ""
	// A literal '?' remains insertable, including as a Vim operator argument.
	if msg.String() == "?" && m.vimNormal && !m.vim.pending() && m.vimVisual == nil && m.vimCommand == "" {
		m.editHelp = !m.editHelp
		return m, nil
	}
	if m.editHelp {
		m.editHelp = false
		if msg.String() == "esc" {
			return m, nil
		}
	}
	if t := m.board.Tasks[m.editingTaskID]; t != nil {
		return m.updateTaskEditor(msg, m.detailLayout(len(t.Whiteboards)))
	}
	return m.updateEditorInput(msg)
}

func (m *model) updateTaskEditor(msg tea.KeyMsg, l detailLayout) (tea.Model, tea.Cmd) {
	if l.descRows > 0 && m.descInput.Focused() && m.vimCommand == "" && m.vimVisual == nil && !m.vimReplace {
		if delta := m.detailEditVerticalMotion(msg); delta != 0 {
			m.moveDetailEditCursor(l, delta)
			return m, nil
		}
	}
	previousMode := m.mode
	next, cmd := m.updateEditorInput(msg)
	if m.mode == previousMode && m.descInput.Focused() && l.descRows > 0 {
		lines := wrapEditText(m.descInput.Value(), l.width-4)
		m.followDetailEditCursor(lines, l.descRows)
	}
	return next, cmd
}

func (m *model) followDetailEditCursor(lines []editLine, rows int) {
	caret := editPosition(lines, descBuffer{m}.Cursor())
	m.editDescTop = followEditCursor(m.editDescTop, rows, len(lines), caret.row)
}

func (m *model) detailEditVerticalMotion(msg tea.KeyMsg) int {
	if m.vimNormal {
		if !m.vim.pending() {
			switch msg.String() {
			case "j":
				return 1
			case "k":
				return -1
			}
		}
	} else if key.Matches(msg, m.descInput.KeyMap.LineNext) {
		return 1
	} else if key.Matches(msg, m.descInput.KeyMap.LinePrevious) {
		return -1
	}
	return 0
}

func (m *model) moveDetailEditCursor(l detailLayout, delta int) {
	buf := descBuffer{m}
	lines := wrapEditText(buf.Text(), l.width-4)
	caret := editPosition(lines, buf.Cursor())
	m.vimStatus = ""
	if next := caret.row + delta; next >= 0 && next < len(lines) {
		line := lines[next]
		prefix := ansi.Cut(line.text, 0, caret.col)
		buf.SetCursor(line.start + len([]rune(prefix)))
		m.followDetailEditCursor(lines, l.descRows)
	}
}

func (m *model) requestDetailDiscard() (tea.Model, tea.Cmd) {
	t := m.board.Tasks[m.editingTaskID]
	if t != nil && (m.titleInput.Value() != t.Title || m.descInput.Value() != t.Description) {
		m.editDiscard = true
	} else {
		m.closeDetailEditor()
	}
	return m, nil
}

func (m *model) closeDetailEditor() {
	m.closeCreatePanel()
	m.mode = m.editReturn
	m.editDiscard, m.editHelp = false, false
	m.detailStatus = ""
}

type editLine struct {
	text  string
	start int // absolute rune offset, including skipped wrapping spaces
}

func wrapEditText(value string, width int) []editLine {
	if width < 1 {
		return nil
	}
	var lines []editLine
	base := 0
	for _, logical := range strings.Split(value, "\n") {
		rest, offset := logical, base
		for _, text := range strings.Split(ansi.Wrap(logical, width, ""), "\n") {
			if i := strings.Index(rest, text); i >= 0 {
				offset += len([]rune(rest[:i]))
				rest = rest[i+len(text):]
			}
			lines = append(lines, editLine{text, offset})
			offset += len([]rune(text))
		}
		// A cursor after an exactly full line needs its own cell. It
		// occupies the start of the next visual row, not the final glyph.
		if last := lines[len(lines)-1]; ansi.StringWidth(last.text) == width {
			lines = append(lines, editLine{start: base + len([]rune(logical))})
		}
		base += len([]rune(logical)) + 1
	}
	return lines
}

func editPosition(lines []editLine, pos int) inputPoint {
	caret := inputPoint{}
	for i, line := range lines {
		if line.start > pos {
			break
		}
		caret.row = i
	}
	if len(lines) > 0 {
		line := lines[caret.row]
		runes := []rune(line.text)
		caret.col = ansi.StringWidth(string(runes[:clampInt(pos-line.start, 0, len(runes))]))
	}
	return caret
}

func followEditCursor(top, height, count, row int) int {
	return clampInt(max(min(top, row), row-height+1), 0, max(0, count-height))
}

func drawEditSelection(g *termgrid.Grid, p dusk.Palette, x, y int, lines []editLine, selection *vimSelection, length int) {
	if selection == nil {
		return
	}
	start, end := selection.rangeBounds(length)
	for i, line := range lines {
		runes := []rune(line.text)
		first, last := max(0, start-line.start), min(len(runes), end-line.start)
		if first < last {
			col := x + ansi.StringWidth(string(runes[:first]))
			for _, r := range runes[first:last] {
				c := g.At(col, y+i)
				c.BG = tokenBG(p, dusk.Accent)
				if c.Ch != ' ' {
					c.FG = tokenBG(p, dusk.SurfaceSelected)
				}
				g.Set(col, y+i, c)
				col += ansi.StringWidth(string(r))
			}
		}
	}
}

func (m *model) drawEditTitle(g *termgrid.Grid, width int, placeholder string) {
	value := m.titleInput.Value()
	pos := m.titleInput.Position()
	runes := []rune(value)
	before := ansi.StringWidth(string(runes[:pos]))
	left := max(0, before-width+1)
	text := ansi.Cut(value, left, left+width)
	var caret *inputPoint
	if m.vimCommand == "" {
		caret = &inputPoint{col: before - left}
	}
	var lines []string
	if value != "" {
		lines = []string{text}
	}
	drawFocusedInput(g, m.palette, 0, 1, width+2, 1, lines, IdleInputTitle, placeholder, caret)
	if m.vimVisual != nil {
		offset := len([]rune(ansi.Cut(value, 0, left)))
		drawEditSelection(g, m.palette, 2, 1, []editLine{{text, offset}}, m.vimVisual, len(runes))
	}
}

func (m *model) drawEditDescription(g *termgrid.Grid, l detailLayout, width int, showCaret bool) (remaining int) {
	value := m.descInput.Value()
	lines := wrapEditText(value, width)
	caret := editPosition(lines, descBuffer{m}.Cursor())
	top := followEditCursor(m.editDescTop, l.descRows, len(lines), caret.row)
	end := min(top+l.descRows, len(lines))
	caret.row -= top
	caret.col = min(caret.col, width-1)
	var cursor *inputPoint
	if showCaret && m.vimCommand == "" {
		cursor = &caret
	}
	var visible []string
	if value != "" {
		for _, line := range lines[top:end] {
			visible = append(visible, line.text)
		}
	}
	drawFocusedInput(g, m.palette, 0, detailDescRow, width+2, l.descRows, visible, IdleInputBody, "description (optional)", cursor)
	drawEditSelection(g, m.palette, 2, detailDescRow, lines[top:end], m.vimVisual, len([]rune(value)))
	return len(lines) - end
}

func (m *model) drawEditFooter(g *termgrid.Grid, l detailLayout, width, remaining int) {
	p := m.palette
	if m.vimCommand != "" {
		drawCommandLine(g, p, 2, l.footer, width, m.vimCommand)
		return
	}
	if m.vimStatus != "" && m.detailStatus == "" && !m.editDiscard {
		drawStatusMessage(g, p, 2, l.footer, width, StatusInfo, m.vimStatus)
		return
	}
	hints := []KeyHint{{":wq", "save"}, {"tab", "next field"}, {"esc", "normal"}, {"?", "more"}}
	if m.descInput.Focused() {
		hints[1] = KeyHint{"ctrl+e", "$EDITOR"}
	}
	if m.vimNormal {
		hints[2] = KeyHint{"esc", "back"}
	}
	if m.editHelp {
		hints = []KeyHint{{":w", "save"}, {"i", "insert"}, {"tab", "next field"}, {"ctrl+e", "$EDITOR"}, {"?", "less"}}
	}
	if remaining > 0 && !m.editHelp {
		label := fmt.Sprintf("↓ %d more", remaining)
		slot := ansi.StringWidth(label) + 3
		if width >= dialogHintsWidth(hints)+slot {
			g.Text(2+width-ansi.StringWidth(label)-2, l.footer, label, tokenFG(p, dusk.TextFaint, false))
			width -= slot
		}
	}
	drawDialogHints(g, p, 2, l.footer, width, hints)
}

func (m *model) updateDetailStatusMenu(key string) (tea.Model, tea.Cmd) {
	t := m.selectedTask()
	if t == nil {
		m.detailStatusMenu = false
		return m, nil
	}
	return m.updateStatusMenu(key, t.Status, func(cursor int) tea.Cmd {
		spot := m.spotOf(t)
		m.board.MoveTask(t.ID, m.board.Statuses()[cursor], -1)
		m.recalculateVisible()
		m.selectTask(t.ID)
		return tea.Batch(m.saveWorkspaceCmd(), m.showUndoToast(StatusSuccess, "Moved to "+columnName(t.Status), moveUndo(spot)))
	})
}

func (m *model) updateStatusMenu(key string, current domain.Status, set func(int) tea.Cmd) (tea.Model, tea.Cmd) {
	statuses := m.board.Statuses()
	switch key {
	case "esc":
		m.detailStatusMenu = false
	case "j", "down", "k", "up":
		delta := 1
		if key == "k" || key == "up" {
			delta = -1
		}
		for i := m.detailStatusCursor + delta; i >= 0 && i < len(statuses); i += delta {
			if statuses[i] != current {
				m.detailStatusCursor = i
				break
			}
		}
	case "enter":
		m.detailStatusMenu = false
		if i := m.detailStatusCursor; i >= 0 && i < len(statuses) && statuses[i] != current {
			return m, set(i)
		}
	}
	return m, nil
}

func (m *model) drawDetailStatusMenu(g *termgrid.Grid, l detailLayout, current domain.Status) {
	statuses := m.board.Statuses()
	rows := min(len(statuses), l.footer-4)
	width := min(21, l.width-4)
	if rows < 1 || width < 5 {
		return
	}
	first := clampInt(m.detailStatusCursor-rows+1, 0, len(statuses)-rows)
	items := make([]MenuItem, 0, rows)
	for i := first; i < first+rows; i++ {
		item := MenuItem{Label: fmt.Sprintf("%d %s", i+1, strings.ToLower(statuses[i].Title()))}
		switch {
		case statuses[i] == current:
			item.Hint, item.State = "now", ListRowDisabled
		case i == m.detailStatusCursor:
			item.State = ListRowSelected
		}
		items = append(items, item)
	}
	g.Blit(l.width-width-4, 2, menuGrid(m.palette, width, items))
}
