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

func roundedBox(p dusk.Palette, w, h int, surface dusk.Token) *termgrid.Grid {
	g := termgrid.New(w, h, tokenBG(p, surface))
	if w < 2 || h < 2 {
		return g
	}
	border := tokenFG(p, dusk.TextFaint, false)
	rule := strings.Repeat("─", w-2)
	g.Text(0, 0, "╭"+rule+"╮", border)
	g.Text(0, h-1, "╰"+rule+"╯", border)
	for y := 1; y < h-1; y++ {
		g.Text(0, y, "│", border)
		g.Text(w-1, y, "│", border)
	}
	return g
}

// RenderDialog draws the Foundations shell with a confirmation body.
// The body is shared by the live task-delete dialog.
func RenderDialog(p dusk.Palette, w, h int, title, body string, warning bool, hints []KeyHint) string {
	return renderGrid(confirmationDialog(p, w, h, title, body, warning, hints))
}

func confirmationDialog(p dusk.Palette, w, h int, title, body string, warning bool, hints []KeyHint) *termgrid.Grid {
	g := newDialog(p, w, h)
	if w < 6 || h < 4 {
		return g
	}
	width := w - 5
	g.Text(3, 1, truncateCell(title, width), tokenFG(p, dusk.Text, true))
	end := h - 2
	if warning && h >= 8 {
		end -= 2
		drawStatusMessage(g, p, 3, end, width, StatusDanger, "can't be undone")
	}
	for i, line := range wrapDialogText(body, width) {
		if i+3 >= end {
			break
		}
		g.Text(3, i+3, line, tokenFG(p, dusk.TextMuted, false))
	}
	drawDialogHints(g, p, 3, h-2, width, hints)
	return g
}

// ApplyScrim repaints the visible backdrop without changing its glyphs or bold.
func ApplyScrim(g *termgrid.Grid, p dusk.Palette) {
	fg, bg := tokenFG(p, dusk.TextFaint, false), tokenBG(p, dusk.Bg)
	for i := range g.Cells {
		c := &g.Cells[i]
		c.BG = bg
		if c.Ch != ' ' && c.Ch != 0 {
			c.FG, c.Faint = fg.FG, fg.Faint
		} else {
			c.FG, c.Faint, c.Bold = "", false, false
		}
	}
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
