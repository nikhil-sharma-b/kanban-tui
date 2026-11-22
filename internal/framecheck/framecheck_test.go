package framecheck

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/nikhilsharma/kanban-tui/internal/ui"
	"github.com/nikhilsharma/kanban-tui/internal/ui/dusk"
)

const designPath = "../../design.figcraft"

var design *Design

func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	var err error
	design, err = Load(designPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "framecheck:", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// reportDir is where failing frames write their reports.
func reportDir(t *testing.T) string {
	dir := os.Getenv("FRAMECHECK_REPORTS")
	if dir == "" {
		dir = filepath.Join("..", "..", ".framecheck")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func loadLedger(t *testing.T) []Entry {
	f, err := os.Open("ledger.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	entries, err := ParseLedger(f)
	if err != nil {
		t.Fatal(err)
	}
	return entries
}

func TestLedgerListsEveryScreenFrame(t *testing.T) {
	inLedger := map[string]bool{}
	for _, e := range loadLedger(t) {
		if inLedger[e.Frame] {
			t.Errorf("ledger lists %s twice", e.Frame)
		}
		inLedger[e.Frame] = true
	}
	inDesign := map[string]bool{}
	for _, f := range design.ScreenFrames() {
		inDesign[f] = true
		if !inLedger[f] {
			t.Errorf("ledger is missing screen frame %s", f)
		}
	}
	for f := range inLedger {
		if !inDesign[f] {
			t.Errorf("ledger lists %s, which is not a screen frame in design.figcraft", f)
		}
	}
}

// checkGrid compares and, on a difference, writes a report and returns its path.
func checkGrid(t *testing.T, name string, want *Expected, got *Grid) (string, []Diff) {
	t.Helper()
	diffs := Compare(want.Grid, got)
	if len(diffs) == 0 && len(want.Problems) == 0 {
		return "", nil
	}
	path := filepath.Join(reportDir(t), strings.NewReplacer("/", "_", " ", "_", "·", "-").Replace(name)+".txt")
	if err := os.WriteFile(path, []byte(Report(name, want.Grid, got, diffs, want.Problems)), 0o644); err != nil {
		t.Fatal(err)
	}
	return path, diffs
}

func summary(diffs []Diff) string {
	var lines []string
	for i, d := range diffs {
		if i == 5 {
			lines = append(lines, fmt.Sprintf("… and %d more", len(diffs)-5))
			break
		}
		lines = append(lines, d.String())
	}
	return strings.Join(lines, "\n")
}

func TestFrames(t *testing.T) {
	for _, e := range loadLedger(t) {
		t.Run(e.Frame, func(t *testing.T) {
			fail := t.Errorf
			if e.Status == Pending {
				fail = t.Logf
			}
			scn, ok := Scenarios[e.Frame]
			if !ok {
				fail("%s: no scenario yet", e.Frame)
				return
			}
			want, err := design.Expect(e.Frame)
			if err != nil {
				t.Fatal(err)
			}
			if want.Mode != scn.Palette.Name {
				fail("%s: frame is in %s mode, scenario renders %s", e.Frame, want.Mode, scn.Palette.Name)
			}
			got := ReadScreen(scn.Run(), want.Grid.W, want.Grid.H, scn.Palette)
			if path, diffs := checkGrid(t, e.Frame, want, got); path != "" {
				fail("%s (%s): %d cells differ, %d design problems; report: %s\n%s",
					e.Frame, e.Status, len(diffs), len(want.Problems), path, summary(diffs))
			}
		})
	}
}

func TestComponents(t *testing.T) {
	type tc struct {
		name   string
		dark   string
		light  string
		render func(p dusk.Palette, width int) string
	}
	counts := "10 open · 3 on daily"
	keybarHints := []ui.KeyHint{{Key: "n", Label: "new"}, {Key: "e", Label: "edit"}, {Key: "H L", Label: "move"}, {Key: ":", Label: "commands"}, {Key: "?", Label: "keys"}}
	cases := []tc{
		{"App header Sync=live", "Components/App header/Sync=live", "Components · light preview/Chrome/App header#0",
			func(p dusk.Palette, w int) string {
				return ui.RenderAppHeader(p, w, ui.AppHeader{Project: "home", Counts: counts, Sync: ui.SyncLive})
			}},
		{"Column header State=normal", "Components/Column header/State=normal", "Components · light preview/Column header/Column header#0",
			func(p dusk.Palette, _ int) string { return ui.RenderColumnHeader(p, "todo · 3", false) }},
		{"Column header State=active", "Components/Column header/State=active", "Components · light preview/Column header/Column header#1",
			func(p dusk.Palette, _ int) string { return ui.RenderColumnHeader(p, "doing · 2", true) }},
		{"Keybar Size=wide", "Components/Keybar/Size=wide", "Components · light preview/Chrome/Keybar#0",
			func(p dusk.Palette, w int) string { return ui.RenderKeybar(p, ui.KeybarWide, w, keybarHints) }},
		{"Keybar Size=narrow", "Components/Keybar/Size=narrow", "Components · light preview/Chrome/Keybar#1",
			func(p dusk.Palette, w int) string { return ui.RenderKeybar(p, ui.KeybarNarrow, w, keybarHints) }},
		{"Column rail", "Components/Column rail", "Components · light preview/Chrome/Column rail#0",
			func(p dusk.Palette, w int) string { return ui.RenderColumnRail(p, w, nil, []string{"doing", "done"}) }},
		{"Key hint", "Components/Key hint", "Components · light preview/Chrome/Keybar#0/hints/Key hint#0",
			func(p dusk.Palette, _ int) string { return ui.RenderKeyHint(p, ui.KeyHint{Key: "n", Label: "new"}) }},
		{"Card State=default", "Components/Card/State=default, Density=regular", "Components · light preview/Card/Card#0",
			func(p dusk.Palette, w int) string {
				return ui.RenderCard(p, w, ui.Card{Title: "Archive search by date", State: ui.CardDefault})
			}},
		{"Card State=selected", "Components/Card/State=selected, Density=regular", "Components · light preview/Card/Card#1",
			func(p dusk.Palette, w int) string {
				return ui.RenderCard(p, w, ui.Card{Title: "Undo for move and delete", Desc: "Undo the last move or delete with u, from a toast",
					Meta: "a3f91c07 · 2d ago · ◇2", State: ui.CardSelected})
			}},
		{"Card State=done", "Components/Card/State=done, Density=regular", "Components · light preview/Card/Card#2",
			func(p dusk.Palette, w int) string {
				return ui.RenderCard(p, w, ui.Card{Title: "Copy task ID keybind", State: ui.CardDone})
			}},
		{"Card State=default compact", "Components/Card/State=default, Density=compact", "Components · light preview/Card/Card#3",
			func(p dusk.Palette, w int) string {
				return ui.RenderCard(p, w, ui.Card{Title: "Archive search by date", State: ui.CardDefault, Compact: true})
			}},
		{"Card State=selected compact", "Components/Card/State=selected, Density=compact", "Components · light preview/Card/Card#4",
			func(p dusk.Palette, w int) string {
				return ui.RenderCard(p, w, ui.Card{Title: "Undo for move and delete", Desc: "Undo the last move or delete with u, from a toast",
					Meta: "a3f91c07 · 2d ago · ◇2", State: ui.CardSelected, Compact: true})
			}},
		{"Card State=done compact", "Components/Card/State=done, Density=compact", "Components · light preview/Card/Card#5",
			func(p dusk.Palette, w int) string {
				return ui.RenderCard(p, w, ui.Card{Title: "Copy task ID keybind", State: ui.CardDone, Compact: true})
			}},
		{"Scroll hint Dir=up", "Components/Scroll hint/Dir=up", "Components · light preview/Toast + Scroll hint/Scroll hint#0",
			func(p dusk.Palette, _ int) string { return ui.RenderScrollHint(p, ui.ScrollUp, 2) }},
		{"Scroll hint Dir=down", "Components/Scroll hint/Dir=down", "Components · light preview/Toast + Scroll hint/Scroll hint#1",
			func(p dusk.Palette, _ int) string { return ui.RenderScrollHint(p, ui.ScrollDown, 3) }},
		{"Badge", "Components/Badge", "Components · light preview/Status/Badge#0",
			func(p dusk.Palette, _ int) string { return ui.RenderBadge(p, "daily") }},
	}
	for _, c := range cases {
		for _, variant := range []struct{ mode, path string }{{"dark", c.dark}, {"light", c.light}} {
			t.Run(c.name+"/"+variant.mode, func(t *testing.T) {
				want, err := design.Expect(variant.path)
				if err != nil {
					t.Fatal(err)
				}
				p, ok := dusk.ByMode(want.Mode)
				if !ok || want.Mode != variant.mode {
					t.Fatalf("%s is in mode %q, want %q", variant.path, want.Mode, variant.mode)
				}
				got := ReadScreen(c.render(p, want.Grid.W), want.Grid.W, want.Grid.H, p)
				if path, diffs := checkGrid(t, "component "+c.name+" "+variant.mode, want, got); path != "" {
					t.Errorf("%s: %d cells differ, %d design problems; report: %s\n%s",
						variant.path, len(diffs), len(want.Problems), path, summary(diffs))
				}
			})
		}
	}
}

// A one-cell change to the render must fail the check and name that cell.
func TestOneCellChangeIsReported(t *testing.T) {
	scn := Scenarios["01/empty-board"]
	want, err := design.Expect("01/empty-board")
	if err != nil {
		t.Fatal(err)
	}
	screen := scn.Run()
	if !strings.Contains(screen, "backlog") {
		t.Fatal("render has no backlog header to alter")
	}
	altered := strings.Replace(screen, "backlog", "backlug", 1)
	got := ReadScreen(altered, want.Grid.W, want.Grid.H, scn.Palette)
	diffs := Compare(want.Grid, got)
	if len(diffs) != 1 {
		t.Fatalf("got %d diffs, want 1:\n%s", len(diffs), summary(diffs))
	}
	report := Report("01/empty-board", want.Grid, got, diffs, nil)
	if d := diffs[0]; d.Row != 4 || d.Col != 8 || d.Want.Ch != 'o' || d.Got.Ch != 'u' {
		t.Fatalf("diff = %s, want row 4 col 8 'o' → 'u'", d)
	}
	if !strings.Contains(report, "row 4 col 8: want \"o\"") {
		t.Fatalf("report does not name the cell:\n%s", report)
	}
}
