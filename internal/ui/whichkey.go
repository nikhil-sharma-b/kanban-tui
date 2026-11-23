package ui

// The which-key popover (page 14): a Menu listing what may follow the typed
// leader path. It appears whichKeyDelay after the sequence starts, unless the
// sequence finishes first, and stays up until it ends. No scrim: the board
// stays readable around it.

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhilsharma/kanban-tui/internal/termgrid"
	"github.com/nikhilsharma/kanban-tui/internal/ui/dusk"
)

const (
	defaultWhichKeyDelay = 100 * time.Millisecond
	whichKeyWideWidth    = 44
	whichKeyNarrowWidth  = 40
)

type whichKeyMsg struct{ seq uint64 }

// box is a screen rectangle in cells.
type box struct{ x, y, w, h int }

func (a box) overlaps(b box) bool {
	return a.x < b.x+b.w && b.x < a.x+a.w && a.y < b.y+b.h && b.y < a.y+a.h
}

// armWhichKey schedules the popover for a sequence that has just started.
func (m *model) armWhichKey() tea.Cmd {
	m.whichKeySeq++
	seq := m.whichKeySeq
	return tea.Tick(m.whichKeyDelay, func(time.Time) tea.Msg { return whichKeyMsg{seq: seq} })
}

func (m *model) cancelPending() {
	m.pendingKeys = nil
	m.whichKeyShown = false
	m.whichKeySeq++
}

// leaderKeyStep handles the keys that edit a pending sequence rather than
// extend it. It reports whether key was handled.
func (m *model) leaderKeyStep(key string) (bool, tea.Cmd) {
	if len(m.pendingKeys) == 0 {
		return false, nil
	}
	switch key {
	case "esc":
		m.cancelPending()
		return true, nil
	case "backspace":
		m.pendingKeys = m.pendingKeys[:len(m.pendingKeys)-1]
		if len(m.pendingKeys) == 0 {
			m.cancelPending()
		}
		return true, nil
	case "?":
		_, cmd := m.openCheatSheet()
		return true, cmd
	}
	seq := strings.Join(append(append([]string(nil), m.pendingKeys...), key), " ")
	if m.peekBoardAction(key) != actNone {
		return false, nil
	}
	if _, ok := boardGroups[seq]; ok {
		return false, nil
	}
	m.cancelPending()
	return true, m.showToast(StatusDanger, fmt.Sprintf("%s isn’t bound", displayKeys(seq)), KeyHint{})
}

// whichKeySize is the popover's size for entries at the current width: two
// columns of entries, a blank row and the key hints, inside the border.
func (m *model) whichKeySize(entries int) (int, int) {
	w := whichKeyWideWidth
	if m.width < keybarWideMin {
		w = whichKeyNarrowWidth
	}
	return w, (entries+1)/2 + 4
}

// whichKeyBox places the popover: bottom-right one row above the keybar,
// bottom-left when that would cover the selected card. Below 100 cols it sits
// on the keybar, since both columns lie under it anyway.
func (m *model) whichKeyBox(entries int) box {
	w, h := m.whichKeySize(entries)
	bottom := m.height - 4 // last row of the popover
	if m.width < keybarWideMin {
		bottom = m.height - 3 // over the Column rail
	} else if m.showRail() {
		bottom--
	}
	inner := m.width - 2*screenGutter
	right := box{x: screenGutter + inner - w, y: bottom - h + 1, w: w, h: h}
	if card, ok := m.selectedCardBox(); ok && card.overlaps(right) {
		left := right
		left.x = screenGutter
		if !card.overlaps(left) {
			return left
		}
	}
	return right
}

// selectedCardBox is where the selected card is drawn on the board.
func (m *model) selectedCardBox() (box, bool) {
	statuses := m.board.Statuses()
	first, n := m.boardWindow()
	i := m.activeColumn - first
	if n == 0 || i < 0 || i >= n {
		return box{}, false
	}
	status := statuses[m.activeColumn]
	cards := m.columnCards(status, true)
	sel := m.selected[status]
	if sel < 0 || sel >= len(cards) {
		return box{}, false
	}
	gap := duskColumnGap
	if m.width < narrowBelowCols {
		gap = narrowColumnGap
	}
	inner := m.width - 2*screenGutter
	colWidth := (inner - gap*(n-1)) / n
	x := screenGutter + i*(colWidth+gap)
	for _, c := range layoutCards(cards, m.scroll[status], 4, m.duskBodyHeight()) {
		if c.index == sel {
			return box{x: x, y: c.top, w: colWidth, h: cardHeight(cards[sel])}, true
		}
	}
	return box{}, false
}

// drawWhichKey blits the popover over the board grid when it shows.
func (m *model) drawWhichKey(g *termgrid.Grid) {
	if !m.whichKeyShown || len(m.pendingKeys) == 0 || m.mode != modeBoard {
		return
	}
	entries := continuations(strings.Join(m.pendingKeys, " "))
	if len(entries) == 0 {
		return
	}
	b := m.whichKeyBox(len(entries))
	g.Blit(b.x, b.y, whichKeyGrid(m.palette, b.w, entries, len(m.pendingKeys) > 1))
}

func whichKeyGrid(p dusk.Palette, w int, entries []whichKeyEntry, nested bool) *termgrid.Grid {
	rows := (len(entries) + 1) / 2
	g := panelBox(p, w, rows+4, dusk.Surface)
	colWidth := (w - 4) / 2
	for i, e := range entries {
		x := 2 + (i%2)*colWidth
		y := 1 + i/2
		end := g.Text(x, y, e.key, tokenFG(p, dusk.TextCard, false))
		g.Text(end+1, y, e.desc, tokenFG(p, dusk.TextDim, false))
	}
	hints := []KeyHint{{"esc", "cancel"}}
	if nested {
		hints = append(hints, KeyHint{"backspace", "back"})
	}
	hints = append(hints, KeyHint{"?", "help"})
	x := 2
	for i, h := range hints {
		if i > 0 {
			x += 2
		}
		x = drawKeyHint(g, p, x, rows+2, h)
	}
	return g
}
