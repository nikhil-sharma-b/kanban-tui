package ui

import "github.com/nikhilsharma/kanban-tui/internal/termgrid"

var firstRunEmptyHints = []KeyHint{
	{"SPC t n", "add the first task"},
	{"SPC p", "projects"},
	{"?", "all keys"},
}

var firstRunTaskHints = []KeyHint{
	{"↵", "open it"},
	{"SPC t l", "move it along"},
}

func (m *model) drawBoardGuide(g *termgrid.Grid, x, y, width, height int, hints []KeyHint) {
	for i, hint := range hints {
		labelWidth := width - len([]rune(hint.Key)) - 1
		if row := 2 * i; row < height && labelWidth > 0 {
			hint.Label = truncateCell(hint.Label, labelWidth)
			drawKeyHint(g, m.palette, x, y+row, hint)
		}
	}
}
