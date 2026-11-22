package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
	"github.com/nikhilsharma/kanban-tui/internal/termgrid"
	"github.com/nikhilsharma/kanban-tui/internal/ui/dusk"
)

// truncateCell fits s into width cells by the DESIGN.md Truncation rule: text
// wider than width is cut to width−1 cols, trailing spaces dropped, then "…".
func truncateCell(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if ansi.StringWidth(s) <= width {
		return s
	}
	return strings.TrimRight(ansi.Truncate(s, width-1, ""), " ") + "…"
}

// ─── Card ────────────────────────────────────────────────────────────────────

// CardState is the Card component's State variant.
type CardState int

const (
	CardDefault CardState = iota
	CardSelected
	CardDone
)

// Card is what one task card shows. Desc and Meta show on a selected card only.
type Card struct {
	Title string
	Desc  string
	Meta  string
	State CardState
	// Compact is Density=compact: no inset rows, square full-height surface.
	Compact bool
}

// cardHeight is the card's height in rows: its text rows, plus an inset row
// above and below at regular density.
func cardHeight(c Card) int {
	rows := 1
	if c.State == CardSelected {
		rows = 3 // title, blank, meta
		if c.Desc != "" {
			rows++
		}
	}
	if c.Compact {
		return rows
	}
	return rows + 2
}

// cardSlot is the rows a card takes in a column: compact cards sit one blank
// row below the header or the card above.
func cardSlot(c Card) int {
	if c.Compact {
		return cardHeight(c) + 1
	}
	return cardHeight(c)
}

// drawCard draws a card with its top-left corner at x, y and
// returns its height.
func drawCard(g *termgrid.Grid, p dusk.Palette, x, y, width int, c Card) int {
	surface, title := dusk.Surface, tokenFG(p, dusk.TextCard, false)
	switch c.State {
	case CardSelected:
		surface, title = dusk.SurfaceSelected, tokenFG(p, dusk.Text, true)
	case CardDone:
		surface, title = dusk.SurfaceDone, tokenFG(p, dusk.TextDone, false)
	}
	bg := tokenBG(p, surface)
	textWidth := width - 3
	h := cardHeight(c)

	inset := func(row int, left, mid, right rune) {
		fg := tokenFG(p, surface, false)
		for col := 0; col < width; col++ {
			ch := mid
			switch col {
			case 0:
				ch = left
			case width - 1:
				ch = right
			}
			g.Set(x+col, row, termgrid.Cell{Ch: ch, FG: fg.FG, Faint: fg.Faint, BG: g.At(x+col, row).BG})
		}
	}
	top := y + 1
	if c.Compact {
		g.Fill(x, y, width, h, bg)
		top = y
	} else {
		inset(y, '▗', '▄', '▖')
		inset(y+h-1, '▝', '▀', '▘')
		g.Fill(x, y+1, width, h-2, bg)
	}

	row := top
	g.Text(x+2, row, truncateCell(c.Title, textWidth), title)
	if c.State == CardSelected {
		row++
		if c.Desc != "" {
			desc := singleLine(c.Desc)
			if c.Compact && strings.Contains(c.Desc, "\n") {
				// Compact cards preview the first wrapped line of multiline descriptions.
				if lines := wrapDialogText(c.Desc, textWidth); len(lines) > 0 {
					desc = lines[0]
				}
			}
			g.Text(x+2, row, truncateCell(desc, textWidth), tokenFG(p, dusk.TextMuted, false))
			row++
		}
		row++ // one blank row before the meta
		g.Text(x+2, row, truncateCell(c.Meta, textWidth), tokenFG(p, dusk.TextDim, false))
		for r := top; r <= row; r++ {
			g.Text(x, r, "▌", tokenFG(p, dusk.Accent, false))
		}
	}
	return h
}

// RenderCard draws the Card component on its own, width cells wide.
func RenderCard(p dusk.Palette, width int, c Card) string {
	g := termgrid.New(width, cardHeight(c), tokenBG(p, dusk.Bg))
	drawCard(g, p, 0, 0, width, c)
	return renderGrid(g)
}

// taskCard is the card a task shows on the board.
func (m *model) taskCard(task *domain.Task, selected bool) Card {
	c := Card{Title: singleLine(task.Title), Compact: m.compact()}
	switch {
	case selected:
		c.State = CardSelected
		c.Desc = task.Description
		c.Meta = cardMeta(task, m.clock(), m.onDailyBoard())
	case task.Status == domain.StatusDone:
		c.State = CardDone
	}
	return c
}

// cardMeta is "short ID · relative age · ◇ whiteboard count", the count
// hidden at 0. The daily board says "today" for anything touched today (11/daily).
func cardMeta(task *domain.Task, now time.Time, daily bool) string {
	age := relativeTime(now, task.UpdatedAt)
	if daily && sameDay(now, task.UpdatedAt) {
		age = "today"
	}
	meta := shortID(task.ID) + " · " + age
	if n := len(task.Whiteboards); n > 0 {
		meta += fmt.Sprintf(" · ◇%d", n)
	}
	return meta
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Local().Date()
	by, bm, bd := b.Local().Date()
	return ay == by && am == bm && ad == bd
}

// ─── Scroll hint ─────────────────────────────────────────────────────────────

// ScrollDir is the Scroll hint component's Dir variant.
type ScrollDir int

const (
	ScrollUp ScrollDir = iota
	ScrollDown
)

func scrollHintText(dir ScrollDir, n int) string {
	arrow := "↑"
	if dir == ScrollDown {
		arrow = "↓"
	}
	return fmt.Sprintf("%s %d more", arrow, n)
}

func drawScrollHint(g *termgrid.Grid, p dusk.Palette, x, y int, dir ScrollDir, n int) {
	g.Text(x, y, scrollHintText(dir, n), tokenFG(p, dusk.TextDim, false))
}

// RenderScrollHint draws the Scroll hint component.
func RenderScrollHint(p dusk.Palette, dir ScrollDir, n int) string {
	text := scrollHintText(dir, n)
	g := termgrid.New(len([]rune(text)), 1, tokenBG(p, dusk.Bg))
	drawScrollHint(g, p, 0, 0, dir, n)
	return renderGrid(g)
}

// ─── Badge ───────────────────────────────────────────────────────────────────

func drawBadge(g *termgrid.Grid, p dusk.Palette, x, y int, label string) int {
	w := len([]rune(label)) + 2
	g.Fill(x, y, w, 1, tokenBG(p, dusk.Accent))
	g.Text(x+1, y, label, tokenFG(p, dusk.Bg, true))
	return x + w
}

// RenderBadge draws the Badge component: the label on an accent fill with one
// cell of padding each side.
func RenderBadge(p dusk.Palette, label string) string {
	g := termgrid.New(len([]rune(label))+2, 1, tokenBG(p, dusk.Bg))
	drawBadge(g, p, 0, 0, label)
	return renderGrid(g)
}

// ─── Column body ─────────────────────────────────────────────────────────────

// Rows the up scroll hint takes at the top of a column body: a blank row, the
// hint, a blank row, so it reads like a card.
const scrollUpRows = 3

// compact is Density=compact, used below 30 rows.
func (m *model) compact() bool { return m.height < compactBelowRows }

// duskBodyHeight is the card area: rows 5 to H−3 (1-based), ending above the
// Column rail and its blank row when the rail shows.
func (m *model) duskBodyHeight() int {
	if m.showRail() {
		return max(1, m.height-9)
	}
	return max(1, m.height-7)
}

// columnCards returns the cards of a column with the selection marked.
func (m *model) columnCards(status domain.Status, active bool) []Card {
	ids := m.visible[status]
	cards := make([]Card, 0, len(ids))
	for i, id := range ids {
		task := m.board.Tasks[id]
		if task == nil {
			task = &domain.Task{ID: id}
		}
		cards = append(cards, m.taskCard(task, active && i == m.selected[status]))
	}
	return cards
}

// fitCards returns how many cards from scroll fit in height rows, leaving room
// for the up hint when scrolled and a down-hint row when cards remain. At
// least one card is always shown.
func fitCards(cards []Card, scroll, height int) int {
	avail := height
	if scroll > 0 {
		avail -= scrollUpRows
	}
	count := 0
	for i := scroll; i < len(cards); i++ {
		h := cardSlot(cards[i])
		reserve := 0
		if i+1 < len(cards) {
			reserve = 1
		}
		if h+reserve > avail && count > 0 {
			break
		}
		avail -= h
		count++
	}
	return count
}

func (m *model) drawColumnBody(g *termgrid.Grid, x, y, width, height int, status domain.Status, active, hintIfEmpty bool) {
	p := m.palette
	cards := m.columnCards(status, active)
	if len(cards) == 0 {
		if hintIfEmpty {
			drawKeyHint(g, p, x, y+1, KeyHint{"SPC t n", "add a task"})
		}
		return
	}
	scroll := min(m.scroll[status], len(cards)-1)
	row := y
	if scroll > 0 {
		drawScrollHint(g, p, x, y+1, ScrollUp, scroll)
		row += scrollUpRows
	}
	end := scroll + fitCards(cards, scroll, height)
	for i := scroll; i < end; i++ {
		drawCard(g, p, x, row+cardSlot(cards[i])-cardHeight(cards[i]), width, cards[i])
		row += cardSlot(cards[i])
	}
	if hidden := len(cards) - end; hidden > 0 {
		drawScrollHint(g, p, x, row, ScrollDown, hidden)
	}
}
