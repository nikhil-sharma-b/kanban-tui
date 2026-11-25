package ui

import (
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

// searchHints are the keys the search prompt shows at the right of the
// keybar row while a query is typed.
var searchHints = []KeyHint{{"↵", "apply"}, {"esc", "restore"}}

// drawSearchLine turns the keybar row into the search prompt, the way vim
// types a search on its command line: the Command line component reads
// "/query" with a full-cell caret at the cursor, and the prompt's keys sit at
// the right. The board above stays fully visible and live-filters as you type.
func (m *model) drawSearchLine(g *termgrid.Grid, x, y, width int, size KeybarSize) {
	p := m.palette
	gap := keybarWideGap
	if size == KeybarNarrow {
		gap = keybarNarrowGap
	}
	hintsWidth := 0
	for i, h := range searchHints {
		if i > 0 {
			hintsWidth += gap
		}
		hintsWidth += h.width()
	}
	col := x + width - hintsWidth
	for i, h := range searchHints {
		if i > 0 {
			col += gap
		}
		col = drawKeyHint(g, p, col, y, h)
	}

	// "/" plus as much of the query as fits before the hints, scrolled so the
	// caret stays in view.
	room := width - hintsWidth - gap - 1
	if room < 2 {
		return
	}
	value := m.searchInput.Value()
	runes := []rune(value)
	pos := min(m.searchInput.Position(), len(runes))
	before := ansi.StringWidth(string(runes[:pos]))
	left := max(0, before-(room-1)+1)
	end := g.Text(x, y, "/", tokenFG(p, dusk.Text, false))
	g.Text(end, y, ansi.Cut(value, left, left+room-1), tokenFG(p, dusk.Text, false))
	drawCaret(g, p, end+before-left, y)
}
