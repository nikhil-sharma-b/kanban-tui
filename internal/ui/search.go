package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/nikhilsharma/kanban-tui/internal/termgrid"
	"github.com/nikhilsharma/kanban-tui/internal/ui/dusk"
)

// boardPlace is a board selection that survives filtering: the column, the
// selected task and where the column window started.
type boardPlace struct {
	column      int
	taskID      string
	windowFirst int
}

// searchDialogWidth is min(60, W−8) cells (page 05 DEV NOTE).
const searchDialogWidth = 60

// filtering reports whether a query is narrowing the board, live or applied.
func (m *model) filtering() bool { return strings.TrimSpace(m.filter) != "" }

func (m *model) openSearch() (tea.Model, tea.Cmd) {
	m.mode = modeSearch
	m.filterDraft = m.filter
	if !m.filtering() {
		m.rememberFilterOrigin()
	}
	m.searchInput.SetValue(m.filter)
	m.searchInput.CursorEnd()
	m.searchInput.Focus()
	return m, textinput.Blink
}

func (m *model) rememberFilterOrigin() {
	place := &boardPlace{column: m.activeColumn, windowFirst: m.windowFirst}
	if task := m.boardSelectedTask(); task != nil {
		place.taskID = task.ID
	}
	m.filterOrigin = place
}

func (m *model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.closeSearch()
		m.setFilter(m.filterDraft)
		return m, nil
	case "enter":
		m.closeSearch()
		query := strings.TrimSpace(m.searchInput.Value())
		if query == "" {
			if m.filtering() || m.filterDraft != "" {
				m.clearFilter()
			} else {
				m.setFilter("")
			}
			return m, nil
		}
		m.setFilter(query)
		return m, nil
	}
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	m.setFilter(m.searchInput.Value())
	return m, cmd
}

func (m *model) closeSearch() {
	m.mode = modeBoard
	m.searchInput.Blur()
}

// setFilter narrows the board to query. When the selection is hidden, the
// first match is selected; an empty query puts the pre-filter selection back.
func (m *model) setFilter(query string) {
	var keep string
	if task := m.boardSelectedTask(); task != nil {
		keep = task.ID
	}
	m.filter = query
	if strings.TrimSpace(query) == "" {
		m.filter = ""
	}
	m.recalculateVisible()
	if keep != "" {
		m.selectTask(keep)
	}
	if !m.filtering() {
		m.restoreFilterOrigin()
		return
	}
	if m.boardSelectedTask() != nil {
		return
	}
	for i, status := range m.board.Statuses() {
		if len(m.visible[status]) > 0 {
			m.activeColumn, m.windowFirst = i, i
			m.selected[status] = 0
			m.syncScroll(status)
			return
		}
	}
}

// clearFilter drops the applied query (esc on the board).
func (m *model) clearFilter() {
	m.filterDraft = ""
	m.searchInput.SetValue("")
	m.setFilter("")
	m.lastErr = nil
	m.lastStatus = "filter cleared"
}

func (m *model) restoreFilterOrigin() {
	place := m.filterOrigin
	m.filterOrigin = nil
	if place == nil || place.column >= len(m.board.Statuses()) {
		return
	}
	m.activeColumn = place.column
	m.windowFirst = place.windowFirst
	if place.taskID != "" {
		m.selectTask(place.taskID)
	}
}

// filterCounts returns the matching and total task counts over every column,
// hidden ones included.
func (m *model) filterCounts() (matches, total int) {
	for _, status := range m.board.Statuses() {
		matches += len(m.visible[status])
		total += len(m.board.Order[status])
	}
	return matches, total
}

func (m *model) searchDialog() *termgrid.Grid {
	p := m.palette
	w := min(searchDialogWidth, m.width-8)
	h := 8
	if m.compact() {
		h = 7
	}
	if w < 8 || h > m.height {
		return nil
	}
	g := newDialog(p, w, h)
	width := w - 5
	label := "filter tasks"
	g.Text(w-2-ansi.StringWidth(label), 1, label, tokenFG(p, dusk.TextDim, false))
	inputWidth := width - ansi.StringWidth(label) - 1

	value := m.searchInput.Value()
	runes := []rune(value)
	pos := min(m.searchInput.Position(), len(runes))
	before := ansi.StringWidth(string(runes[:pos]))
	left := max(0, before-inputWidth+1)
	var lines []string
	if value != "" {
		lines = []string{ansi.Cut(value, left, left+inputWidth)}
	}
	drawFocusedInput(g, p, 1, 1, inputWidth+2, 1, lines, IdleInputField, "", &inputPoint{col: before - left})

	matches, total := m.filterCounts()
	result := fmt.Sprintf("%d of %d tasks match", matches, total)
	if matches == 0 {
		result = "No tasks match"
	}
	g.Text(3, 3, truncateCell(result, width), tokenFG(p, dusk.TextMuted, false))
	drawDialogHints(g, p, 3, h-2, width, []KeyHint{{"↵", "apply"}, {"esc", "restore"}, {"backspace", "delete character"}})
	return g
}
