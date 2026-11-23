package ui

// Focus mode (page 13): z on the board shows the selected task alone on a
// centred sheet, with the columns collapsed to their counts. It is only a
// view of the board selection, so every board action keeps working and z or
// esc returns to the board on the same task. Nothing new is stored.

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
	"github.com/nikhilsharma/kanban-tui/internal/termgrid"
	"github.com/nikhilsharma/kanban-tui/internal/ui/dusk"
)

const (
	focusSheetWidth = 60 // centred sheet at W ≥ keybarWideMin; full width below
	focusDescRows   = 12 // description region, 6 rows when compact
)

type focusState struct {
	on bool
	// pick is the selected card of the left column while the focused column
	// is empty (13/no-doing).
	pick int
	// scroll is the description's first line; scrollID the task it is for.
	scroll   int
	scrollID string
}

// focusLeft is the column left of the focused one, or "" when there is none.
func (m *model) focusLeft() domain.Status {
	statuses := m.board.Statuses()
	if m.activeColumn <= 0 || m.activeColumn >= len(statuses) {
		return ""
	}
	return statuses[m.activeColumn-1]
}

// focusPicked is the selected left-column card while the column is empty.
func (m *model) focusPicked() *domain.Task {
	left := m.focusLeft()
	if left == "" || len(m.visible[left]) == 0 {
		return nil
	}
	m.focus.pick = max(0, min(m.focus.pick, len(m.visible[left])-1))
	return m.board.Tasks[m.visible[left][m.focus.pick]]
}

// updateFocus handles the keys focus mode owns; the rest fall through to the
// board (handled false).
func (m *model) updateFocus(key string) (bool, tea.Model, tea.Cmd) {
	if len(m.pendingKeys) > 0 {
		return false, m, nil
	}
	task := m.boardSelectedTask()
	switch key {
	case "z", "esc":
		m.focus = focusState{}
		return true, m, nil
	case "J", "K":
		if task != nil {
			if key == "J" {
				m.focus.scroll++
			} else {
				m.focus.scroll = max(0, m.focus.scroll-1)
			}
		}
		return true, m, nil
	case "h", "l", "left", "right", "/":
		// The columns are collapsed; search belongs to the board.
		return true, m, nil
	case "j", "k", "down", "up", "enter":
		if task == nil {
			if key == "j" || key == "down" {
				m.focus.pick++
			} else if key == "k" || key == "up" {
				m.focus.pick = max(0, m.focus.pick-1)
			}
			m.focusPicked()
			return true, m, nil
		}
	}
	return false, m, nil
}

// focusMoveRight moves the focused task one column right and advances focus
// to the next task in the column (13/advanced). With the column empty it
// starts the picked left-column card instead and focuses on it.
func (m *model) focusMoveRight() tea.Cmd {
	statuses := m.board.Statuses()
	column := m.activeColumn
	if column >= len(statuses) {
		return nil
	}
	status := statuses[column]
	if m.boardSelectedTask() == nil {
		picked := m.focusPicked()
		if picked == nil {
			return nil
		}
		m.selectTask(picked.ID)
		_, cmd := m.shiftSelected(1)
		if picked.Status != status {
			return cmd
		}
		m.focus.pick = 0
		return tea.Batch(cmd, m.movedToast(picked))
	}
	index := m.selected[status]
	task := m.boardSelectedTask()
	_, cmd := m.shiftSelected(1)
	if task.Status == status {
		return cmd
	}
	m.activeColumn = column
	if index >= len(m.visible[status]) {
		index = 0
	}
	m.selected[status] = index
	m.syncScroll(status)
	return tea.Batch(cmd, m.movedToast(task))
}

// movedToast names a task focus mode moved and where it went (13/advanced).
func (m *model) movedToast(task *domain.Task) tea.Cmd {
	return m.showToast(StatusSuccess, m.quotedToast("", task.Title, KeyHint{})+" → "+columnName(task.Status), KeyHint{})
}

// focusUpNext is the card after the focused one in its column; when the
// column is about to empty, the top of the column to the left.
func (m *model) focusUpNext(status domain.Status) (string, *domain.Task) {
	visible := m.visible[status]
	if len(visible) > 1 {
		next := (m.selected[status] + 1) % len(visible)
		return "up next in " + columnName(status), m.board.Tasks[visible[next]]
	}
	if left := m.focusLeft(); left != "" && len(m.visible[left]) > 0 {
		return fmt.Sprintf("%s empties after this · next from %s", columnName(status), columnName(left)), m.board.Tasks[m.visible[left][0]]
	}
	return "", nil
}

func (m *model) focusHints(size KeybarSize, task *domain.Task) []KeyHint {
	statuses := m.board.Statuses()
	right := ""
	switch {
	case m.activeColumn+1 == len(statuses)-1:
		right = "done"
	case m.activeColumn+1 < len(statuses):
		right = "move"
	}
	var hints []KeyHint
	add := func(ok bool, key, label string) {
		if ok {
			hints = append(hints, KeyHint{key, label})
		}
	}
	if task == nil {
		picked := m.focusPicked() != nil
		add(picked, "j k", "select")
		add(true, "SPC t n", "new")
		add(picked, "SPC t l", "start")
	} else {
		wide := size == KeybarWide
		add(true, "j k", "task")
		add(true, "↵", "details")
		add(wide, "SPC t e", "edit")
		add(right != "", "SPC t l", right)
		add(wide, "SPC t w", "whiteboards")
		add(wide, "SPC", "more")
	}
	add(true, "esc", "board")
	add(true, "?", "help")
	return hints
}

// focusView draws focus mode on the cell grid.
func (m *model) focusView() *termgrid.Grid {
	p := m.palette
	g := termgrid.New(m.width, m.height, tokenBG(p, dusk.Bg))
	inner := m.width - 2*screenGutter
	drawAppHeader(g, p, screenGutter, 1, inner, m.appHeader())

	statuses := m.board.Statuses()
	task := m.boardSelectedTask()
	// With the column empty, the pick in the column to its left is active.
	active := m.activeColumn
	if task == nil && m.focusPicked() != nil {
		active--
	}
	x := screenGutter
	for i, status := range statuses {
		x = drawColumnHeader(g, p, x, 3, columnLabel(status, len(m.visible[status])), i == active) + 3
	}
	if m.toast.text != "" && m.mode == modeBoard {
		toast := toastGrid(p, m.toast.kind, truncateCell(m.toast.text, inner-6), m.toast.hint)
		g.Blit(screenGutter+inner-toast.W, 3, toast)
	}

	size, sheetW := KeybarWide, focusSheetWidth
	if m.width < keybarWideMin {
		size, sheetW = KeybarNarrow, inner
	}
	x0 := (m.width - sheetW) / 2
	textWidth := sheetW - 2
	top, descRows := 6, focusDescRows
	if m.compact() {
		top, descRows = 5, focusDescRows/2
	}
	bottom := m.height - 3 // last row above the keybar's blank row

	if task == nil {
		m.drawFocusEmpty(g, x0, top, sheetW, bottom)
	} else {
		if task.ID != m.focus.scrollID {
			m.focus.scroll, m.focus.scrollID = 0, task.ID
		}
		status := columnName(task.Status) + " · " + relativeTime(m.clock(), task.UpdatedAt)
		titleWidth := textWidth
		if status = truncateCell(status, textWidth/2); status != "" {
			titleWidth -= ansi.StringWidth(status) + 1
			g.Text(x0+sheetW-ansi.StringWidth(status), top, status, tokenFG(p, dusk.TextDim, false))
		}
		drawIdleInput(g, p, x0, top, titleWidth+2, []string{task.Title}, IdleInputTitle, "")

		lines := wrapDialogText(task.Description, textWidth)
		m.focus.scroll = min(m.focus.scroll, max(0, len(lines)-descRows))
		end := min(len(lines), m.focus.scroll+descRows)
		drawIdleInput(g, p, x0, top+2, textWidth+2, lines[m.focus.scroll:end], IdleInputBody, "")

		label, next := m.focusUpNext(statuses[m.activeColumn])
		listEnd := bottom
		if next != nil {
			card := Card{Title: singleLine(next.Title), Compact: m.compact()}
			if card.State = CardDefault; next.Status == domain.StatusDone {
				card.State = CardDone
			}
			cardY := bottom - 3
			if card.Compact {
				cardY++
			}
			drawCard(g, p, x0, cardY, sheetW, card)
			g.Text(x0+2, bottom-4, truncateCell(label, textWidth), tokenFG(p, dusk.TextDim, false))
			listEnd = bottom - 5
		}

		if n := len(task.Whiteboards); n > 0 {
			y := top + 2 + descRows + 1
			g.Text(x0+2, y, truncateCell(fmt.Sprintf("whiteboards · %d", n), textWidth), tokenFG(p, dusk.TextDim, false))
			for i, w := range task.Whiteboards {
				if y+1+i > listEnd {
					break
				}
				format := w.Format
				if format == "" {
					format = w.Extension()[1:]
				}
				drawListRow(g, p, x0, y+1+i, sheetW, "◇ "+w.Name, format+" · "+relativeTime(m.clock(), w.UpdatedAt), ListRowDefault)
			}
		}
	}

	hints := m.focusHints(size, task)
	m.drawWhichKey(g)
	drawKeybar(g, p, screenGutter, m.height-2, inner, size, hints, m.keybarStatus(inner, size, hints))
	return g
}

// drawFocusEmpty lists the left column's cards under a note while the
// focused column is empty (13/no-doing).
func (m *model) drawFocusEmpty(g *termgrid.Grid, x0, top, sheetW, bottom int) {
	p := m.palette
	status := m.board.Statuses()[m.activeColumn]
	note := columnName(status) + " is empty"
	picked := m.focusPicked()
	left := m.focusLeft()
	if picked != nil {
		note += " · start one from " + columnName(left)
	}
	g.Text(x0+2, top, truncateCell(note, sheetW-2), tokenFG(p, dusk.TextDim, false))
	if picked == nil {
		return
	}
	var cards []Card
	for _, id := range m.visible[left] {
		task := m.board.Tasks[id]
		cards = append(cards, m.taskCard(task, task == picked))
	}
	// Scroll just far enough to keep the picked card in view.
	first, room := 0, bottom-top
	for {
		used := 0
		for _, c := range cards[first : m.focus.pick+1] {
			used += cardSlot(c)
		}
		if used <= room || first == m.focus.pick {
			break
		}
		first++
	}
	y := top + 1
	if m.compact() {
		y++
	}
	for _, c := range cards[first:] {
		if y+cardHeight(c)-1 > bottom {
			break
		}
		drawCard(g, p, x0, y, sheetW, c)
		y += cardSlot(c)
	}
}
