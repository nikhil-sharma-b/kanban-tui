// Package termgrid is a terminal screen as a grid of cells: draw into it and
// serialise it to ANSI, or parse ANSI output back into it.
package termgrid

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/muesli/termenv"
)

// Cell is one terminal cell. FG and BG are colour strings as lipgloss spells
// them ("#RRGGBB" or an ANSI index such as "5"); "" means unset.
type Cell struct {
	Ch    rune
	FG    string
	BG    string
	Bold  bool
	Faint bool
}

// Grid is W×H cells, row-major.
type Grid struct {
	W, H  int
	Cells []Cell
}

// New returns a grid of blank cells with the given background.
func New(w, h int, bg string) *Grid {
	g := &Grid{W: w, H: h, Cells: make([]Cell, w*h)}
	for i := range g.Cells {
		g.Cells[i] = Cell{Ch: ' ', BG: bg}
	}
	return g
}

// At returns the cell at col x, row y (zero value when out of range).
func (g *Grid) At(x, y int) Cell {
	if x < 0 || y < 0 || x >= g.W || y >= g.H {
		return Cell{}
	}
	return g.Cells[y*g.W+x]
}

// Set replaces the cell at col x, row y; out-of-range writes are dropped.
func (g *Grid) Set(x, y int, c Cell) {
	if x < 0 || y < 0 || x >= g.W || y >= g.H {
		return
	}
	g.Cells[y*g.W+x] = c
}

// Style is the foreground part of a text run.
type Style struct {
	FG    string
	Bold  bool
	Faint bool
}

// Text writes s from col x on row y, one rune per cell, keeping each cell's
// background. It returns the col after the last rune.
func (g *Grid) Text(x, y int, s string, st Style) int {
	for _, r := range s {
		bg := g.At(x, y).BG
		g.Set(x, y, Cell{Ch: r, FG: st.FG, BG: bg, Bold: st.Bold, Faint: st.Faint})
		x++
	}
	return x
}

// Fill paints the background of a w×h block and clears its glyphs.
func (g *Grid) Fill(x, y, w, h int, bg string) {
	for row := y; row < y+h; row++ {
		for col := x; col < x+w; col++ {
			g.Set(col, row, Cell{Ch: ' ', BG: bg})
		}
	}
}

// Blit copies src onto g with its top-left corner at col x, row y.
func (g *Grid) Blit(x, y int, src *Grid) {
	for row := 0; row < src.H; row++ {
		for col := 0; col < src.W; col++ {
			g.Set(x+col, y+row, src.At(col, row))
		}
	}
}

// Rows returns the glyphs of each row as plain text.
func (g *Grid) Rows() []string {
	rows := make([]string, g.H)
	for y := 0; y < g.H; y++ {
		var b strings.Builder
		for x := 0; x < g.W; x++ {
			ch := g.At(x, y).Ch
			if ch == 0 {
				ch = ' '
			}
			b.WriteRune(ch)
		}
		rows[y] = b.String()
	}
	return rows
}

// Render serialises the grid to ANSI in the given colour profile, one line per row.
func (g *Grid) Render(profile termenv.Profile) string {
	var b strings.Builder
	for y := 0; y < g.H; y++ {
		if y > 0 {
			b.WriteByte('\n')
		}
		var prev Cell
		open := false
		for x := 0; x < g.W; x++ {
			c := g.At(x, y)
			if !open || c.FG != prev.FG || c.BG != prev.BG || c.Bold != prev.Bold || c.Faint != prev.Faint {
				if open {
					b.WriteString("\x1b[0m")
				}
				b.WriteString(sgr(c, profile))
				open = true
				prev = c
			}
			ch := c.Ch
			if ch == 0 {
				ch = ' '
			}
			b.WriteRune(ch)
		}
		if open {
			b.WriteString("\x1b[0m")
		}
	}
	return b.String()
}

func sgr(c Cell, profile termenv.Profile) string {
	var parts []string
	if c.Bold {
		parts = append(parts, "1")
	}
	if c.Faint {
		parts = append(parts, "2")
	}
	if s := colorSeq(c.FG, false, profile); s != "" {
		parts = append(parts, s)
	}
	if s := colorSeq(c.BG, true, profile); s != "" {
		parts = append(parts, s)
	}
	if len(parts) == 0 {
		return ""
	}
	return "\x1b[" + strings.Join(parts, ";") + "m"
}

// colorSeq writes truecolor hex exactly (termenv's own conversion rounds
// through floats and can shift a channel by one) and defers to termenv otherwise.
func colorSeq(col string, bg bool, profile termenv.Profile) string {
	if col == "" {
		return ""
	}
	if profile == termenv.TrueColor && len(col) == 7 && col[0] == '#' {
		r, errR := strconv.ParseUint(col[1:3], 16, 8)
		gr, errG := strconv.ParseUint(col[3:5], 16, 8)
		b, errB := strconv.ParseUint(col[5:7], 16, 8)
		if errR == nil && errG == nil && errB == nil {
			lead := "38"
			if bg {
				lead = "48"
			}
			return fmt.Sprintf("%s;2;%d;%d;%d", lead, r, gr, b)
		}
	}
	seq := profile.Color(col)
	if seq == nil {
		return ""
	}
	return seq.Sequence(bg)
}

// Parse reads ANSI text into a w×h grid. Only SGR sequences are interpreted;
// other escape sequences are skipped. Every rune takes one cell.
func Parse(s string, w, h int) *Grid {
	g := New(w, h, "")
	lines := strings.Split(s, "\n")
	for y, line := range lines {
		if y >= h {
			break
		}
		var cur Cell
		x := 0
		for i := 0; i < len(line); {
			if line[i] == 0x1b {
				n, params, final := scanEscape(line[i:])
				if final == 'm' {
					applySGR(&cur, params)
				}
				i += n
				continue
			}
			r, size := utf8.DecodeRuneInString(line[i:])
			i += size
			if r == '\r' {
				continue
			}
			c := cur
			c.Ch = r
			g.Set(x, y, c)
			x++
		}
	}
	return g
}

// scanEscape returns the length of the escape sequence at the start of s, its
// CSI parameters and its final byte (0 for non-CSI sequences).
func scanEscape(s string) (int, string, byte) {
	if len(s) < 2 {
		return len(s), "", 0
	}
	switch s[1] {
	case '[':
		for j := 2; j < len(s); j++ {
			if s[j] >= 0x40 && s[j] <= 0x7e {
				return j + 1, s[2:j], s[j]
			}
		}
		return len(s), "", 0
	case ']':
		for j := 2; j < len(s); j++ {
			if s[j] == 0x07 {
				return j + 1, "", 0
			}
			if s[j] == 0x1b && j+1 < len(s) && s[j+1] == '\\' {
				return j + 2, "", 0
			}
		}
		return len(s), "", 0
	}
	return 2, "", 0
}

func applySGR(c *Cell, params string) {
	if params == "" {
		*c = Cell{}
		return
	}
	fields := strings.Split(params, ";")
	num := func(i int) int {
		if i >= len(fields) {
			return 0
		}
		n, _ := strconv.Atoi(fields[i])
		return n
	}
	for i := 0; i < len(fields); i++ {
		n := num(i)
		switch {
		case n == 0:
			*c = Cell{}
		case n == 1:
			c.Bold = true
		case n == 2:
			c.Faint = true
		case n == 22:
			c.Bold, c.Faint = false, false
		case n >= 30 && n <= 37:
			c.FG = strconv.Itoa(n - 30)
		case n >= 90 && n <= 97:
			c.FG = strconv.Itoa(n - 90 + 8)
		case n >= 40 && n <= 47:
			c.BG = strconv.Itoa(n - 40)
		case n >= 100 && n <= 107:
			c.BG = strconv.Itoa(n - 100 + 8)
		case n == 39:
			c.FG = ""
		case n == 49:
			c.BG = ""
		case n == 38 || n == 48:
			var col string
			switch num(i + 1) {
			case 5:
				col = strconv.Itoa(num(i + 2))
				i += 2
			case 2:
				col = fmt.Sprintf("#%02X%02X%02X", num(i+2), num(i+3), num(i+4))
				i += 4
			}
			if n == 38 {
				c.FG = col
			} else {
				c.BG = col
			}
		}
	}
}
