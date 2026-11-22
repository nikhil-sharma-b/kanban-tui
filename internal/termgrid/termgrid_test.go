package termgrid

import (
	"testing"

	"github.com/muesli/termenv"
)

func TestRenderParseRoundTrip(t *testing.T) {
	g := New(6, 2, "#17151A")
	g.Text(1, 0, "ab", Style{FG: "#D4A5C7", Bold: true})
	g.Fill(0, 1, 3, 1, "#1F1C23")
	g.Text(4, 1, "▌", Style{FG: "5", Faint: true})

	got := Parse(g.Render(termenv.TrueColor), 6, 2)
	for i := range g.Cells {
		if got.Cells[i] != g.Cells[i] {
			t.Fatalf("cell %d: got %+v, want %+v", i, got.Cells[i], g.Cells[i])
		}
	}
}

func TestParseSkipsNonSGR(t *testing.T) {
	g := Parse("\x1b[2Kx\x1b]0;title\x07y", 3, 1)
	if rows := g.Rows(); rows[0] != "xy " {
		t.Fatalf("rows = %q", rows)
	}
}
