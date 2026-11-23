package ui

import (
	"github.com/charmbracelet/x/ansi"

	"github.com/nikhilsharma/kanban-tui/internal/termgrid"
	"github.com/nikhilsharma/kanban-tui/internal/ui/dusk"
)

type inputPoint struct{ col, row int }

func drawCaret(g *termgrid.Grid, p dusk.Palette, x, y int) {
	c := g.At(x, y)
	if c.Ch != 0 && c.Ch != ' ' {
		c.FG = c.BG
	}
	c.BG = tokenBG(p, dusk.Text)
	g.Set(x, y, c)
}

func drawFocusedInput(g *termgrid.Grid, p dusk.Palette, x, y, w, h int, lines []string, kind IdleInputKind, placeholder string, caret *inputPoint) {
	for i := 0; i < h; i++ {
		g.Text(x, y+i, "▐", tokenFG(p, dusk.Accent, false))
	}
	if len(lines) == 0 && placeholder != "" && w > 3 && h > 0 {
		g.Text(x+3, y, truncateCell(placeholder, w-3), tokenFG(p, dusk.TextFaint, false))
	}
	drawIdleInput(g, p, x, y, w, lines[:min(h, len(lines))], kind, "")
	if caret != nil && caret.col >= 0 && caret.col < w-2 && caret.row >= 0 && caret.row < h {
		drawCaret(g, p, x+2+caret.col, y+caret.row)
	}
}

// RenderFocusedInput draws a Foundations input with an optional full-cell caret.
// caretCol is relative to the text, or negative when another control owns it.
func RenderFocusedInput(p dusk.Palette, width, height int, lines []string, kind IdleInputKind, placeholder string, caretCol, caretRow int) string {
	g := termgrid.New(width, height, tokenBG(p, dusk.Bg))
	drawFocusedInput(g, p, 0, 0, width, height, lines, kind, placeholder, &inputPoint{caretCol, caretRow})
	return renderGrid(g)
}

func drawCommandLine(g *termgrid.Grid, p dusk.Palette, x, y, width int, command string) {
	if width < 1 {
		return
	}
	text := ansi.Cut(command, max(0, ansi.StringWidth(command)-width+1), ansi.StringWidth(command))
	end := g.Text(x, y, text, tokenFG(p, dusk.Text, false))
	drawCaret(g, p, end, y)
}

// RenderCommandLine draws the Foundations command and its full-cell caret.
func RenderCommandLine(p dusk.Palette, width int, command string) string {
	g := termgrid.New(width, 1, tokenBG(p, dusk.Bg))
	drawCommandLine(g, p, 0, 0, width, command)
	return renderGrid(g)
}

type MenuItem struct {
	Label, Hint string
	State       ListRowState
}

func menuGrid(p dusk.Palette, width int, items []MenuItem) *termgrid.Grid {
	g := panelBox(p, width, len(items)+2, dusk.Surface)
	for i, item := range items {
		drawListRow(g, p, 0, i+1, width-1, item.Label, item.Hint, item.State)
	}
	return g
}

// RenderMenu draws the Foundations Menu around a list of rows.
func RenderMenu(p dusk.Palette, width int, items []MenuItem) string {
	return renderGrid(onBg(p, menuGrid(p, width, items)))
}
