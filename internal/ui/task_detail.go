package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/nikhilsharma/kanban-tui/internal/termgrid"
	"github.com/nikhilsharma/kanban-tui/internal/ui/dusk"
)

const (
	dialogMaxWidth  = 60
	dialogMaxHeight = 20
	detailDescRow   = 3
)

// dialogW is a dialog's width: half the screen, held between 60 and 96
// cols, and never closer than 4 cols to either side.
func (m *model) dialogW() int {
	return min(max(dialogMaxWidth, min(m.width/2, 96)), m.width-8)
}

// dialogH is a full-height dialog's height: 5/9 of the screen, held between
// 20 and 32 rows, and never closer than 2 rows to the top or bottom.
func (m *model) dialogH() int {
	return min(max(dialogMaxHeight, min(m.height*5/9, 32)), m.height-4)
}

func (m *model) dialogSize(int) (int, int) {
	return m.dialogW(), m.dialogH()
}

type detailLayout struct {
	width, height, descRows, listRows, footer int
}

func (m *model) detailLayout(whiteboards int) detailLayout {
	w, h := m.dialogSize(dialogMaxHeight)
	l := detailLayout{width: w, height: h, footer: h - 2}
	if m.height < compactBelowRows {
		l.footer--
	}
	l.descRows = max(0, l.footer-detailDescRow-2)
	if whiteboards > 0 && l.descRows >= 4 {
		l.listRows = min(3, l.descRows-3)
		l.descRows -= l.listRows + 2 // label and blank row
	}
	return l
}

func (m *model) scrollDetail(down bool) {
	task := m.selectedTask()
	if task == nil {
		return
	}
	l := m.detailLayout(len(task.Whiteboards))
	lines := wrapDialogText(task.Description, l.width-4)
	if down {
		m.detailScroll = min(max(0, len(lines)-l.descRows), m.detailScroll+1)
	} else {
		m.detailScroll = max(0, m.detailScroll-1)
	}
}

func (m *model) taskDetail() *termgrid.Grid {
	if m.mode == modeCreate {
		return m.taskCreate()
	}
	task := m.selectedTask()
	editing := m.mode == modeDetailEdit
	if task == nil {
		return nil
	}
	p := m.palette
	l := m.detailLayout(len(task.Whiteboards))
	if l.width < 6 || l.height < 5 || l.footer < 2 {
		return nil
	}
	g := newDialog(p, l.width, l.height)
	const x = 2
	right := l.width - 2 // exclusive right edge of content
	textWidth := right - x
	status := strings.ToLower(task.Status.Title())
	switch {
	case m.editDiscard && editing:
		status = "modified"
	case m.detailStatusMenu:
		status += " ▾"
	}
	status += " · " + relativeTime(m.clock(), task.UpdatedAt)
	titleWidth := textWidth
	if status = truncateCell(status, textWidth/2); status != "" {
		titleWidth -= ansi.StringWidth(status) + 1
		g.Text(right-ansi.StringWidth(status), 1, status, tokenFG(p, dusk.TextDim, false))
	}
	title, description := task.Title, task.Description
	if editing && !m.editDiscard {
		title, description = m.titleInput.Value(), m.descInput.Value()
	}
	if editing && !m.editDiscard && m.titleInput.Focused() {
		m.drawEditTitle(g, titleWidth, "")
	} else {
		drawIdleInput(g, p, 0, 1, titleWidth+2, []string{title}, IdleInputTitle, "")
	}

	lines := wrapDialogText(description, textWidth)
	start := min(m.detailScroll, max(0, len(lines)-l.descRows))
	if editing {
		start = 0
		if !m.editDiscard {
			start = min(m.editDescTop, max(0, len(lines)-l.descRows))
		}
	}
	end := min(len(lines), start+l.descRows)
	editRemaining := 0
	if editing && !m.editDiscard && m.descInput.Focused() {
		editRemaining = m.drawEditDescription(g, l, textWidth, true)
	} else {
		drawIdleInput(g, p, 0, detailDescRow, textWidth+2, lines[start:end], IdleInputBody, "")
	}

	if l.listRows > 0 {
		y := detailDescRow + l.descRows + 1
		label := dusk.TextDim
		if m.detailWhiteboards && !editing {
			label = dusk.Accent
		}
		g.Text(x, y, truncateCell(fmt.Sprintf("whiteboards · %d", len(task.Whiteboards)), textWidth), tokenFG(p, label, false))
		first := 0
		if m.detailWhiteboards && !editing {
			first = max(0, m.whiteboardCursor-l.listRows+1)
		}
		for i := first; i < min(len(task.Whiteboards), first+l.listRows); i++ {
			w := task.Whiteboards[i]
			state := ListRowDefault
			if m.detailWhiteboards && !editing && i == m.whiteboardCursor {
				state = ListRowSelected
			}
			format := w.Format
			if format == "" {
				format = strings.TrimPrefix(w.Extension(), ".")
			}
			hint := format + " · " + relativeTime(m.clock(), w.UpdatedAt)
			drawListRow(g, p, 0, y+1+i-first, right, "◇ "+w.Name, hint, state)
		}
	}

	hints := []KeyHint{{"e", "edit"}, {"s", "status"}, {"esc", "close"}, {"?", "more"}}
	switch {
	case editing && m.editDiscard:
		drawDiscardFooter(g, p, l.footer, textWidth)
		return g
	case m.detailStatus != "":
		drawStatusMessage(g, p, x, l.footer, textWidth, m.detailStatusKind, m.detailStatus)
		return g
	case editing:
		m.drawEditFooter(g, l, textWidth, editRemaining)
		return g
	case m.detailStatusMenu:
		m.drawDetailStatusMenu(g, l, task.Status)
		drawDialogHints(g, p, x, l.footer, textWidth, []KeyHint{{"j k", "move"}, {"↵", "set"}, {"esc", "cancel"}})
		return g
	case m.detailWhiteboards && m.detailHelp:
		hints = []KeyHint{{"j k", "move"}, {"o", "open"}, {"esc", "back"}, {"?", "less"}}
	case m.detailWhiteboards:
		hints = []KeyHint{{"j k", "move"}, {"o", "open"}, {"esc", "back"}, {"?", "more"}}
	case m.detailHelp:
		hints = []KeyHint{{"J K", "scroll"}, {"y", "copy ID"}, {"w", "whiteboards"}, {"esc", "back"}, {"?", "less"}}
	}
	if end < len(lines) && l.descRows > 0 && !m.detailWhiteboards && !m.detailHelp {
		text := fmt.Sprintf("↓ %d more line", len(lines)-end)
		if len(lines)-end != 1 {
			text += "s"
		}
		// Keep the painted frames' 18-cell slot at full dialog width.
		slot := ansi.StringWidth(text) + 2
		if l.width == dialogMaxWidth {
			slot = max(slot, 19)
		}
		if slot <= textWidth {
			g.Text(right-slot, l.footer, text, tokenFG(p, dusk.TextFaint, false))
			textWidth -= slot
			// Prefer keeping Close and More visible when actions don't fit.
			for len(hints) > 2 && dialogHintsWidth(hints) > textWidth {
				hints = hints[1:]
			}
		}
	}
	drawDialogHints(g, p, x, l.footer, textWidth, hints)
	return g
}

func drawDiscardFooter(g *termgrid.Grid, p dusk.Palette, row, width int) {
	const x = 2
	prompt := "discard unsaved changes?"
	hints := []KeyHint{{"y", "yes"}, {"n", "no"}}
	if hintWidth := dialogHintsWidth(hints) + 2; width > hintWidth {
		prompt = truncateCell(prompt, width-hintWidth)
		hintX := ansi.StringWidth(prompt) + 2
		drawDialogHints(g, p, x+hintX, row, width-hintX, hints)
	}
	drawStatusMessage(g, p, x, row, width, StatusPrompt, prompt)
}
