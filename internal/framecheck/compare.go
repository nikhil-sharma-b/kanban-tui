package framecheck

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/nikhilsharma/kanban-tui/internal/termgrid"
	"github.com/nikhilsharma/kanban-tui/internal/ui/dusk"
)

// ReadScreen parses rendered ANSI into token cells, naming each colour by the
// palette token it came from. A colour no token has reads as "raw:<colour>".
func ReadScreen(ansi string, w, h int, p dusk.Palette) *Grid {
	tg := termgrid.Parse(ansi, w, h)
	names := map[string]string{}
	for i := len(dusk.Tokens) - 1; i >= 0; i-- { // earlier tokens win on shared values
		t := dusk.Tokens[i]
		if c, ok := p.Get(t).Color.(lipgloss.Color); ok {
			names[strings.ToUpper(string(c))] = string(t)
		}
	}
	unsetBG := ""
	if _, ok := p.Bg.Color.(lipgloss.NoColor); ok {
		unsetBG = string(dusk.Bg)
	}
	name := func(col string) string {
		if col == "" {
			return ""
		}
		if t, ok := names[strings.ToUpper(col)]; ok {
			return t
		}
		return "raw:" + col
	}
	g := newGrid(w, h, "")
	for i, c := range tg.Cells {
		out := Cell{Ch: c.Ch, FG: name(c.FG), BG: name(c.BG), Bold: c.Bold}
		if c.BG == "" {
			out.BG = unsetBG
		}
		if out.Ch == 0 {
			out.Ch = ' '
		}
		if out.Ch == ' ' {
			out.FG, out.Bold = "", false
		}
		g.Cells[i] = out
	}
	return g
}

// Diff is one cell that differs.
type Diff struct {
	Row, Col  int // 1-based, as in DESIGN.md
	Want, Got Cell
}

func (d Diff) String() string {
	return fmt.Sprintf("row %d col %d: want %s, got %s", d.Row, d.Col, describe(d.Want), describe(d.Got))
}

func describe(c Cell) string {
	s := fmt.Sprintf("%q bg=%s", string(c.Ch), orNone(c.BG))
	if c.Ch != ' ' {
		s += " fg=" + orNone(c.FG)
		if c.Bold {
			s += " bold"
		}
	}
	return s
}

func orNone(s string) string {
	if s == "" {
		return "none"
	}
	return s
}

// Compare lists every cell where got differs from want. Grids of different
// sizes compare over the larger size, so missing cells show up as diffs.
func Compare(want, got *Grid) []Diff {
	var diffs []Diff
	for y := 0; y < max(want.H, got.H); y++ {
		for x := 0; x < max(want.W, got.W); x++ {
			w, g := want.At(x, y), got.At(x, y)
			if w != g {
				diffs = append(diffs, Diff{Row: y + 1, Col: x + 1, Want: w, Got: g})
			}
		}
	}
	return diffs
}

// Report is the readable failure text for one frame.
func Report(name string, want, got *Grid, diffs []Diff, problems []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s: %d differing cells (rows and cols are 1-based)\n", name, len(diffs))
	if len(problems) > 0 {
		b.WriteString("\ndesign problems:\n")
		for _, p := range problems {
			b.WriteString("  " + p + "\n")
		}
	}
	writeRows := func(title string, g *Grid) {
		fmt.Fprintf(&b, "\n%s (%d×%d):\n", title, g.W, g.H)
		for i, row := range g.Rows() {
			fmt.Fprintf(&b, "%3d |%s|\n", i+1, row)
		}
	}
	writeRows("expected", want)
	writeRows("actual", got)
	b.WriteString("\ndiffering cells:\n")
	for _, d := range diffs {
		b.WriteString("  " + d.String() + "\n")
	}
	return b.String()
}
