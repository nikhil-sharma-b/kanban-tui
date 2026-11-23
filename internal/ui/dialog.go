package ui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/nikhilsharma/kanban-tui/internal/termgrid"
	"github.com/nikhilsharma/kanban-tui/internal/ui/dusk"
)

// newDialog draws the Foundations shell. Callers lay out content inside it.
func newDialog(p dusk.Palette, w, h int) *termgrid.Grid {
	return roundedBox(p, w, h, dusk.SurfaceSelected)
}

// roundedBox is a panel whose fill stops half a cell inside its outer ring:
// half blocks in the surface colour draw the edge, so there is no stroke and
// the ring's other half shows whatever the box is blitted onto.
func roundedBox(p dusk.Palette, w, h int, surface dusk.Token) *termgrid.Grid {
	g := termgrid.New(w, h, tokenBG(p, surface))
	if w < 2 || h < 2 {
		return g
	}
	edge := tokenFG(p, surface, false)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			ch := ' '
			switch {
			case y == 0 && x == 0:
				ch = '▗'
			case y == 0 && x == w-1:
				ch = '▖'
			case y == h-1 && x == 0:
				ch = '▝'
			case y == h-1 && x == w-1:
				ch = '▘'
			case y == 0:
				ch = '▄'
			case y == h-1:
				ch = '▀'
			case x == 0:
				ch = '▐'
			case x == w-1:
				ch = '▌'
			default:
				continue
			}
			if edge.FG == "" {
				// An unpainted surface (ansi16) has no edge to draw.
				ch = ' '
			}
			g.Set(x, y, termgrid.Cell{Ch: ch, FG: edge.FG, Faint: edge.Faint})
		}
	}
	return g
}

// RenderDialog draws the Foundations Dialog: a tier-1 confirmation without a
// note, through the same layout as the live confirm dialogs (page 09).
func RenderDialog(p dusk.Palette, w, h int, title, body, verb string) string {
	return renderGrid(onBg(p, confirmationGrid(p, w, h, confirmation{title: title, body: body, verb: verb})))
}

// onBg sets a standalone panel on the page background, which its edge's
// outer half shows.
func onBg(p dusk.Palette, panel *termgrid.Grid) *termgrid.Grid {
	g := termgrid.New(panel.W, panel.H, tokenBG(p, dusk.Bg))
	g.Blit(0, 0, panel)
	return g
}

// ApplyScrim repaints the visible backdrop: every fill flattens to surface,
// block glyphs drawing a fill's edge follow it, and every other glyph
// (accent bars included) turns text-faint.
func ApplyScrim(g *termgrid.Grid, p dusk.Palette) {
	fg, bg := tokenFG(p, dusk.TextFaint, false), tokenBG(p, dusk.Bg)
	surface := tokenFG(p, dusk.Surface, false)
	fills := map[string]bool{}
	for _, t := range []dusk.Token{dusk.Surface, dusk.SurfaceDone, dusk.SurfaceSelected} {
		if c := tokenBG(p, t); c != "" && c != bg {
			fills[c] = true
		}
	}
	for i := range g.Cells {
		c := &g.Cells[i]
		if fills[c.BG] {
			c.BG = surface.FG
		} else {
			c.BG = bg
		}
		if isFillGlyph(c.Ch) {
			if fills[c.FG] {
				c.FG, c.Faint, c.Bold = surface.FG, surface.Faint, false
				continue
			}
		}
		if c.Ch != ' ' && c.Ch != 0 {
			c.FG, c.Faint = fg.FG, fg.Faint
		} else {
			c.FG, c.Faint, c.Bold = "", false, false
		}
	}
}

// isFillGlyph reports block elements that draw part of a fill.
func isFillGlyph(r rune) bool {
	return strings.ContainsRune("▄▀▗▖▝▘▐▌", r)
}

func (m *model) placeDialog(board, dialog *termgrid.Grid) string {
	ApplyScrim(board, m.palette)
	if dialog != nil {
		board.Blit((board.W-dialog.W)/2, (board.H-dialog.H)/2, dialog)
	}
	if m.toast.text != "" {
		// A toast raised inside a dialog (a status move in detail) stays lit
		// above the scrim.
		toast := toastGrid(m.palette, m.toast.kind, truncateCell(m.toast.text, board.W-2*screenGutter-6), m.toast.hint)
		board.Blit(board.W-screenGutter-toast.W, board.H-4, toast)
	}
	return renderGrid(board)
}

// drawDialogHints draws only whole hints, in the supplied order.
func drawDialogHints(g *termgrid.Grid, p dusk.Palette, x, y, width int, hints []KeyHint) {
	for _, hint := range hints {
		if hint.width() > width {
			break
		}
		x = drawKeyHint(g, p, x, y, hint) + 2
		width -= hint.width() + 2
	}
}

func dialogHintsWidth(hints []KeyHint) int {
	width := 0
	for i, hint := range hints {
		if i > 0 {
			width += 2
		}
		width += hint.width()
	}
	return width
}

func wrapDialogText(s string, width int) []string {
	if width <= 0 || s == "" {
		return nil
	}
	return strings.Split(ansi.Wrap(s, width, ""), "\n")
}

type ListRowState int

const (
	ListRowDefault ListRowState = iota
	ListRowSelected
	ListRowDisabled
)

func drawListRow(g *termgrid.Grid, p dusk.Palette, x, y, width int, label, hint string, state ListRowState) {
	labelToken, hintToken := dusk.TextCard, dusk.TextDim
	switch state {
	case ListRowSelected:
		g.Text(x, y, "▌", tokenFG(p, dusk.Accent, false))
		labelToken = dusk.Accent
	case ListRowDisabled:
		labelToken, hintToken = dusk.TextFaint, dusk.TextFaint
	}
	labelWidth := width - 2
	if hint != "" && ansi.StringWidth(hint)+4 <= width {
		hintWidth := ansi.StringWidth(hint)
		labelWidth -= hintWidth + 2
		g.Text(x+width-hintWidth, y, hint, tokenFG(p, hintToken, false))
	}
	g.Text(x+2, y, truncateCell(label, labelWidth), tokenFG(p, labelToken, false))
}

// RenderListRow draws one Foundations list row.
func RenderListRow(p dusk.Palette, width int, label, hint string, state ListRowState) string {
	g := termgrid.New(width, 1, tokenBG(p, dusk.Bg))
	drawListRow(g, p, 0, 0, width, label, hint, state)
	return renderGrid(g)
}

type IdleInputKind int

const (
	IdleInputField IdleInputKind = iota
	IdleInputTitle
	IdleInputBody
)

func drawIdleInput(g *termgrid.Grid, p dusk.Palette, x, y, width int, lines []string, kind IdleInputKind, placeholder string) {
	token, bold := dusk.Text, kind == IdleInputTitle
	if kind == IdleInputBody {
		token = dusk.TextCard
	}
	if len(lines) == 0 && placeholder != "" {
		lines, token, bold = []string{placeholder}, dusk.TextFaint, false
	}
	for i, line := range lines {
		g.Text(x+2, y+i, truncateCell(line, width-2), tokenFG(p, token, bold))
	}
}

// RenderIdleInput draws the unfocused Foundations text input.
func RenderIdleInput(p dusk.Palette, width int, value string, kind IdleInputKind, placeholder string) string {
	g := termgrid.New(width, 1, tokenBG(p, dusk.Bg))
	var lines []string
	if value != "" {
		lines = []string{value}
	}
	drawIdleInput(g, p, 0, 0, width, lines, kind, placeholder)
	return renderGrid(g)
}

type StatusKind int

const (
	StatusSuccess StatusKind = iota
	StatusDanger
	StatusInfo
	StatusPrompt
)

func drawStatusMessage(g *termgrid.Grid, p dusk.Palette, x, y, width int, kind StatusKind, text string) {
	glyph, color, label := "✓", dusk.Success, dusk.Success
	switch kind {
	case StatusDanger:
		glyph, color, label = "✗", dusk.Danger, dusk.Danger
	case StatusInfo:
		glyph, color, label = "●", dusk.Accent, dusk.TextCard
	case StatusPrompt:
		glyph, label = "", dusk.Text
	}
	if glyph != "" && width > 0 {
		g.Text(x, y, glyph, tokenFG(p, color, false))
		x, width = x+2, width-2
	}
	g.Text(x, y, truncateCell(text, width), tokenFG(p, label, kind == StatusPrompt))
}

// RenderStatusMessage draws one Foundations status message.
func RenderStatusMessage(p dusk.Palette, width int, kind StatusKind, text string) string {
	g := termgrid.New(width, 1, tokenBG(p, dusk.Bg))
	drawStatusMessage(g, p, 0, 0, width, kind, text)
	return renderGrid(g)
}

// RenderToast draws the Foundations toast with an optional action hint.
// Only success and danger have their own dot color; other kinds use accent.
func RenderToast(p dusk.Palette, kind StatusKind, text string, hint KeyHint) string {
	return renderGrid(toastGrid(p, kind, text, hint))
}

func toastGrid(p dusk.Palette, kind StatusKind, text string, hint KeyHint) *termgrid.Grid {
	width := 6 + ansi.StringWidth(text)
	if hint.Key != "" {
		width += 3 + hint.width()
	}
	g := termgrid.New(width, 1, tokenBG(p, dusk.SurfaceSelected))
	dot := dusk.Accent
	if kind == StatusSuccess {
		dot = dusk.Success
	} else if kind == StatusDanger {
		dot = dusk.Danger
	}
	g.Text(2, 0, "●", tokenFG(p, dot, false))
	end := g.Text(4, 0, text, tokenFG(p, dusk.TextCard, false))
	if hint.Key != "" {
		g.Text(end+1, 0, "·", tokenFG(p, dusk.TextDim, false))
		drawKeyHint(g, p, end+3, 0, hint)
	}
	return g
}
