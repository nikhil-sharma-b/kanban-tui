package framecheck

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/nikhilsharma/kanban-tui/internal/termgrid"
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
			if scn.Palette.Name == dusk.ANSI16.Name {
				// ansi16 tokens share colours; compare what the terminal shows.
				want.Grid = Canonical(want.Grid, scn.Palette)
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
		{"App header Sync=error", "Components/App header/Sync=error", "Components · light preview/Chrome/App header#1",
			func(p dusk.Palette, w int) string {
				return ui.RenderAppHeader(p, w, ui.AppHeader{Project: "home", Counts: counts, Sync: ui.SyncError})
			}},
		{"App header Sync=reloaded", "Components/App header/Sync=reloaded", "Components · light preview/Chrome/App header#2",
			func(p dusk.Palette, w int) string {
				return ui.RenderAppHeader(p, w, ui.AppHeader{Project: "home", Counts: counts, Sync: ui.SyncReloaded})
			}},
		{"Column header State=normal", "Components/Column header/State=normal", "Components · light preview/Column header/Column header#0",
			func(p dusk.Palette, _ int) string { return ui.RenderColumnHeader(p, "todo · 3", false) }},
		{"Column header State=active", "Components/Column header/State=active", "Components · light preview/Column header/Column header#1",
			func(p dusk.Palette, _ int) string { return ui.RenderColumnHeader(p, "doing · 2", true) }},
		{"Column header State=editing", "Components/Column header/State=editing", "Components · light preview/Column header/Column header#2",
			func(p dusk.Palette, _ int) string { return ui.RenderColumnHeaderEditing(p, "ready", 3) }},
		{"Keybar Size=wide", "Components/Keybar/Size=wide", "Components · light preview/Chrome/Keybar#0",
			func(p dusk.Palette, w int) string { return ui.RenderKeybar(p, ui.KeybarWide, w, keybarHints) }},
		{"Keybar Size=narrow", "Components/Keybar/Size=narrow", "Components · light preview/Chrome/Keybar#1",
			func(p dusk.Palette, w int) string { return ui.RenderKeybar(p, ui.KeybarNarrow, w, keybarHints) }},
		{"Column rail", "Components/Column rail", "Components · light preview/Chrome/Column rail#0",
			func(p dusk.Palette, w int) string { return ui.RenderColumnRail(p, w, nil, []string{"doing", "done"}) }},
		{"Key hint", "Components/Key hint", "Components · light preview/Status/Key hint#0",
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

func TestComponentsDialogs(t *testing.T) {
	type fixture struct {
		name, dark, light string
		render            func(dusk.Palette, int, int) string
	}
	cases := []fixture{
		{"Dialog", "Dialog", "Dialog", func(p dusk.Palette, w, h int) string {
			return ui.RenderDialog(p, w, h, "Delete column “backlog”?", "5 tasks will move to the archive. You can restore them later.", "delete")
		}},
		{"Scrim", "Scrim", "Scrim", func(p dusk.Palette, w, h int) string {
			g := termgrid.New(w, h, "")
			if p.Name == "light" {
				g.Blit(0, 0, termgrid.Parse(ui.RenderCard(p, 26, ui.Card{Title: "Archive search by date"}), 26, 3))
				g.Blit(0, 3, termgrid.Parse(ui.RenderCard(p, 26, ui.Card{Title: "Copy task ID keybind", State: ui.CardDone}), 26, 3))
			}
			ui.ApplyScrim(g, p)
			return g.Render(termenv.TrueColor)
		}},
		{"Command line", "Command line", "Status/Command line#0", func(p dusk.Palette, w, _ int) string {
			return ui.RenderCommandLine(p, w, ":wq")
		}},
		{"Menu", "Menu", "List row + Menu/Menu#0", func(p dusk.Palette, w, _ int) string {
			return ui.RenderMenu(p, w, []ui.MenuItem{
				{Label: "1 backlog"},
				{Label: "3 doing", State: ui.ListRowSelected},
				{Label: "2 todo", Hint: "now", State: ui.ListRowDisabled},
			})
		}},
	}
	for i, row := range []struct {
		name, label, hint string
		state             ui.ListRowState
	}{
		{"default", "Move to done", "SPC t d", ui.ListRowDefault},
		{"selected", "Move to doing", "L", ui.ListRowSelected},
		{"disabled", "Restore from archive", "not archived", ui.ListRowDisabled},
	} {
		cases = append(cases, fixture{"List row " + row.name, "List row/State=" + row.name, fmt.Sprintf("List row + Menu/List row#%d", i),
			func(p dusk.Palette, w, _ int) string { return ui.RenderListRow(p, w, row.label, row.hint, row.state) }})
	}
	for _, input := range []struct {
		name, variant, value, placeholder string
		kind                              ui.IdleInputKind
		index                             int
	}{
		{"field", "idle", "Undo for move", "", ui.IdleInputField, 1},
		{"title", "idle", "Undo for move", "", ui.IdleInputTitle, 4},
		{"body", "idle", "Undo for move", "", ui.IdleInputBody, 7},
		{"body", "placeholder", "", "description (optional)", ui.IdleInputBody, 9},
	} {
		cases = append(cases, fixture{"Text input " + input.name + " " + input.variant, "Text input/Kind=" + input.name + ", State=" + input.variant,
			fmt.Sprintf("Text input/Text input#%d", input.index), func(p dusk.Palette, w, _ int) string {
				return ui.RenderIdleInput(p, w, input.value, input.kind, input.placeholder)
			}})
	}
	for _, input := range []struct {
		name, placeholder string
		kind              ui.IdleInputKind
		index, caret      int
	}{
		{"field", "", ui.IdleInputField, 0, 13},
		{"field", "name", ui.IdleInputField, 2, 0},
		{"title", "", ui.IdleInputTitle, 3, 13},
		{"title", "title", ui.IdleInputTitle, 5, 0},
		{"body", "", ui.IdleInputBody, 6, -1},
		{"body", "description (optional)", ui.IdleInputBody, 8, 0},
	} {
		variant := "focused"
		var lines []string
		if input.placeholder != "" {
			variant = "empty"
		} else {
			lines = []string{"Undo for move"}
		}
		cases = append(cases, fixture{"Text input " + input.name + " " + variant,
			"Text input/Kind=" + input.name + ", State=" + variant, fmt.Sprintf("Text input/Text input#%d", input.index),
			func(p dusk.Palette, w, h int) string {
				return ui.RenderFocusedInput(p, w, h, lines, input.kind, input.placeholder, input.caret, 0)
			}})
	}
	for i, status := range []struct {
		name, status, toast string
		kind                ui.StatusKind
	}{
		{"success", "created 7c2e0b14 in todo", "Copied k3f9", ui.StatusSuccess},
		{"danger", "title can't be empty", "Could not save — retrying", ui.StatusDanger},
		{"info", "board reloaded · changed by CLI", "Board reloaded · changed by CLI", ui.StatusInfo},
		{"prompt", "discard unsaved changes?", "", ui.StatusPrompt},
	} {
		cases = append(cases, fixture{"Status message " + status.name, "Status message/Kind=" + status.name, fmt.Sprintf("Status/Status message#%d", i),
			func(p dusk.Palette, w, _ int) string { return ui.RenderStatusMessage(p, w, status.kind, status.status) }})
		if status.toast != "" {
			cases = append(cases, fixture{"Toast " + status.name, "Toast/Kind=" + status.name, fmt.Sprintf("Toast + Scroll hint/Toast#%d", i),
				func(p dusk.Palette, _, _ int) string {
					return ui.RenderToast(p, status.kind, status.toast, ui.KeyHint{})
				}})
		}
	}
	cases = append(cases, fixture{"Toast with hint", "", "Toast + Scroll hint/Toast#3", func(p dusk.Palette, _, _ int) string {
		return ui.RenderToast(p, ui.StatusInfo, "Archived “Theme config file”", ui.KeyHint{Key: "u", Label: "undo"})
	}})
	for _, c := range cases {
		for _, variant := range []struct{ mode, prefix, path string }{
			{"dark", "Components/", c.dark},
			{"light", "Components · light preview/", c.light},
		} {
			if variant.path == "" {
				continue
			}
			t.Run(c.name+"/"+variant.mode, func(t *testing.T) {
				want, err := design.Expect(variant.prefix + variant.path)
				if err != nil {
					t.Fatal(err)
				}
				p, ok := dusk.ByMode(want.Mode)
				if !ok || want.Mode != variant.mode {
					t.Fatalf("%s is in mode %q, want %q", variant.path, want.Mode, variant.mode)
				}
				got := ReadScreen(c.render(p, want.Grid.W, want.Grid.H), want.Grid.W, want.Grid.H, p)
				if path, diffs := checkGrid(t, "component "+c.name+" "+variant.mode, want, got); path != "" {
					t.Errorf("%s: %d cells differ, %d design problems; report: %s\n%s",
						variant.path, len(diffs), len(want.Problems), path, summary(diffs))
				}
			})
		}
	}
}

func TestBoardHelpUsesActivePalette(t *testing.T) {
	for _, source := range []string{"01/default", "01/default-80x24"} {
		t.Run(source, func(t *testing.T) {
			h := Scenarios[source]
			h.Keys = append(append([]string(nil), h.Keys...), "?")
			dark := ReadScreen(h.Run(), h.W, h.H, h.Palette)
			h.Palette = dusk.Light
			light := ReadScreen(h.Run(), h.W, h.H, h.Palette)
			rows := strings.Join(light.Rows(), "\n")
			for _, binding := range []string{"start here", "everything else · : lists all", "SPC t h l"} {
				if !strings.Contains(rows, binding) {
					t.Fatalf("help did not show %q", binding)
				}
			}
			for i, c := range light.Cells {
				if strings.HasPrefix(c.FG, "raw:") || strings.HasPrefix(c.BG, "raw:") {
					t.Fatalf("non-palette colour at row %d col %d: %+v", i/light.W+1, i%light.W+1, c)
				}
			}
			if diffs := Compare(dark, light); len(diffs) != 0 {
				t.Errorf("changing theme changed help cells beyond their palette values:\n%s", summary(diffs))
			}
		})
	}
}

func TestCheatSheetFiltersKeysAndRunsTheSelectedAction(t *testing.T) {
	s := firstRunScenario(120, 36, "?", "SPC p")
	screen := func(s Scenario) string {
		return strings.Join(ReadScreen(s.Run(), s.W, s.H, s.Palette).Rows(), "\n")
	}
	if rows := screen(s); !strings.Contains(rows, "matches · 1") || !strings.Contains(rows, "projects") || strings.Contains(rows, "start here") {
		t.Fatal("filtering the cheat sheet by displayed keys should show the matching project action")
	}
	s.After = []string{"enter"}
	if rows := screen(s); !strings.Contains(rows, "Filter projects…") || strings.Contains(rows, "start here") {
		t.Fatal("enter should run the matching project action")
	}
	s.After = []string{"esc"}
	if rows := screen(s); strings.Contains(rows, "matches · 1") || !strings.Contains(rows, "add the first task") {
		t.Fatal("escape should close the cheat sheet without running an action")
	}
}

func TestFirstRunGuideFollowsActiveTaskCount(t *testing.T) {
	s := firstRunScenario(120, 36, "space", "t", "n", "First", "enter")
	screen := func(s Scenario) string {
		return strings.Join(ReadScreen(s.Run(), s.W, s.H, s.Palette).Rows(), "\n")
	}
	if rows := screen(s); !strings.Contains(rows, "open it") || strings.Contains(rows, "add the first task") {
		t.Fatal("first saved task should replace the empty-board guide with card hints")
	}
	s.After = []string{"space", "t", "n", "Second", "enter"}
	if rows := screen(s); strings.Contains(rows, "open it") || strings.Contains(rows, "add the first task") || !strings.Contains(rows, "Second") {
		t.Fatal("second saved task should remove the first-run guide")
	}
}

func TestFirstRunGuideStaysUsableBetweenFrameSizes(t *testing.T) {
	s := firstRunScenario(112, 36)
	rows := strings.Join(ReadScreen(s.Run(), s.W, s.H, s.Palette).Rows(), "\n")
	if !strings.Contains(rows, "SPC t n add the first") {
		t.Fatal("the primary guide should still show in a narrower four-column layout")
	}
}

func TestFirstTaskGuideOnlyShowsAvailableActions(t *testing.T) {
	s := firstTaskScenario()
	s.Keys = []string{"l"}
	rows := strings.Join(ReadScreen(s.Run(), s.W, s.H, s.Palette).Rows(), "\n")
	if strings.Contains(rows, "open it") || strings.Contains(rows, "move it along") {
		t.Fatal("the lone card's shortcuts should not show while another column is active")
	}
	s.Keys = []string{"space", "t", "l", "space", "t", "l", "space", "t", "l"}
	rows = strings.Join(ReadScreen(s.Run(), s.W, s.H, s.Palette).Rows(), "\n")
	if !strings.Contains(rows, "open it") || strings.Contains(rows, "move it along") {
		t.Fatal("a card in the final column should show open, but not move right")
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
