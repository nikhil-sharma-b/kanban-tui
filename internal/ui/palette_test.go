package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func palettePress(t *testing.T, m *model, keys ...string) *model {
	t.Helper()
	named := map[string]tea.KeyType{"enter": tea.KeyEnter, "esc": tea.KeyEsc, "down": tea.KeyDown, "up": tea.KeyUp, "tab": tea.KeyTab, "backspace": tea.KeyBackspace}
	for _, k := range keys {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		if typ, ok := named[k]; ok {
			msg = tea.KeyMsg{Type: typ}
		}
		next, _ := m.Update(msg)
		m = next.(*model)
	}
	return m
}

func paletteLabels(m *model) map[string]bool {
	out := map[string]bool{}
	for _, item := range m.paletteItems() {
		out[item.label] = true
	}
	return out
}

func TestPaletteListsNewBindingWithNoOtherChange(t *testing.T) {
	saved := boardBindings
	t.Cleanup(func() { boardBindings = saved })
	boardBindings = append(append([]boardBinding(nil), saved...), boardBinding{"space t z", actTaskCopyID + 100, "zap", ""})

	m, _ := boardAt(t, 120, 36, false, "l", "j", ":")
	if m.mode != modePalette {
		t.Fatalf("mode = %v, want palette", m.mode)
	}
	if !paletteLabels(m)["zap"] {
		t.Fatal("a binding added to the leader table is missing from the palette")
	}
	m = palettePress(t, m, "z", "a", "p")
	items := m.paletteItems()
	if len(items) != 1 || items[0].hint != "SPC t z" {
		t.Fatalf("matches for zap = %+v, want the new binding with its keys", items)
	}
}

func TestPaletteColonInDetailEditStaysVimCommandLine(t *testing.T) {
	m, _ := boardAt(t, 120, 36, false, "l", "j")
	m = palettePress(t, m, "enter", "e", "esc", ":")
	if m.mode != modeDetailEdit || m.vimCommand != ":" {
		t.Fatalf("mode = %v, vimCommand = %q; want the vim command line", m.mode, m.vimCommand)
	}
}

func TestPaletteFromDetailSwapsInPlace(t *testing.T) {
	m, _ := boardAt(t, 120, 36, false, "l", "j")
	m = palettePress(t, m, "enter", ":")
	if m.mode != modePalette || !m.pal.fromDetail {
		t.Fatalf("mode = %v, fromDetail = %v", m.mode, m.pal.fromDetail)
	}
	if paletteLabels(m)["open details"] {
		t.Fatal("open details listed while details are open")
	}
	m = palettePress(t, m, "esc")
	if m.mode != modeDetail {
		t.Fatalf("esc from palette over details: mode = %v, want details", m.mode)
	}
}

func TestPaletteBoardActionFromDetailClosesDetailsFirst(t *testing.T) {
	m, _ := boardAt(t, 120, 36, false, "l", "j")
	task := m.selectedTask()
	m = palettePress(t, m, "enter", ":", "m", "o", "v", "e", " ", "t", "a", "s", "k", " ", "r", "enter")
	if m.mode != modeBoard {
		t.Fatalf("mode = %v, want board", m.mode)
	}
	if got := m.board.Tasks[task.ID].Status; got != "doing" {
		t.Fatalf("status = %q, want doing", got)
	}
}

func TestPaletteRecentKeepsLastThreeInMemory(t *testing.T) {
	m, _ := boardAt(t, 120, 36, false, "l", "j")
	run := func(query string) {
		m = palettePress(t, m, ":")
		for _, r := range query {
			m = palettePress(t, m, string(r))
		}
		m = palettePress(t, m, "enter")
		if m.mode != modeBoard {
			m = palettePress(t, m, "esc")
		}
	}
	run("first task")
	run("last task")
	run("select next task")
	run("first task")
	run("select previous task")
	want := []boardAction{actTaskUp, actTaskTop, actTaskDown}
	if len(m.pal.recent) != len(want) {
		t.Fatalf("recent = %v, want %v", m.pal.recent, want)
	}
	for i := range want {
		if m.pal.recent[i] != want[i] {
			t.Fatalf("recent = %v, want %v", m.pal.recent, want)
		}
	}
	m = palettePress(t, m, ":")
	if s := m.paletteSections(); len(s) != 2 || s[0].title != "recent" || len(s[0].items) != 3 {
		t.Fatalf("sections = %+v", s)
	}
}

func TestPaletteSecondStepEscGoesBack(t *testing.T) {
	m, _ := boardAt(t, 120, 36, false, ":", "s", "w", "i", "t", "c", "h")
	m = palettePress(t, m, "enter")
	if m.mode != modePalette || m.pal.step != paletteProjects {
		t.Fatalf("mode = %v step = %v, want the project step", m.mode, m.pal.step)
	}
	m = palettePress(t, m, "esc")
	if m.mode != modePalette || m.pal.step != paletteActions || m.pal.input.Value() != "" {
		t.Fatalf("esc: mode = %v step = %v query = %q, want the action list", m.mode, m.pal.step, m.pal.input.Value())
	}
	m = palettePress(t, m, "esc")
	if m.mode != modeBoard {
		t.Fatalf("second esc: mode = %v, want board", m.mode)
	}
}

func TestPaletteSetStatusMovesTask(t *testing.T) {
	m, _ := boardAt(t, 120, 36, false, "l", "j")
	task := m.selectedTask()
	m = palettePress(t, m, "enter", ":", "s", "e", "t", " ", "s", "enter")
	if m.pal.step != paletteStatuses {
		t.Fatalf("step = %v, want statuses", m.pal.step)
	}
	m = palettePress(t, m, "d", "o", "n", "e", "enter")
	if m.mode != modeDetail || m.board.Tasks[task.ID].Status != "done" {
		t.Fatalf("mode = %v status = %q, want details with done", m.mode, m.board.Tasks[task.ID].Status)
	}
}

func TestPaletteNoMatchCarriesQueryToSearch(t *testing.T) {
	m, _ := boardAt(t, 120, 36, false, ":", "u", "n", "d", "o")
	if len(m.paletteItems()) != 0 {
		t.Fatalf("undo matched %+v", m.paletteItems())
	}
	m = palettePress(t, m, "enter")
	if m.mode != modeSearch || m.searchInput.Value() != "undo" {
		t.Fatalf("mode = %v query = %q, want search for undo", m.mode, m.searchInput.Value())
	}
}

func TestPaletteMatchesWordStartsOnly(t *testing.T) {
	m, _ := boardAt(t, 120, 36, false, ":", "a", "r", "c", "h")
	var got []string
	for _, item := range m.paletteItems() {
		got = append(got, item.label)
	}
	want := []string{"archive task", "archive old done", "view archive"}
	if len(got) != len(want) {
		t.Fatalf("matches = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("matches = %v, want %v", got, want)
		}
	}
}

func TestPaletteListScrollsUnderFixedFooter(t *testing.T) {
	m, _ := boardAt(t, 120, 36, false, ":")
	before := m.paletteDialog()
	for i := 0; i < 20; i++ {
		m = palettePress(t, m, "down")
	}
	after := m.paletteDialog()
	if before.H != after.H || before.W != after.W {
		t.Fatalf("dialog resized from %dx%d to %dx%d", before.W, before.H, after.W, after.H)
	}
	if m.pal.cursor != 20 {
		t.Fatalf("cursor = %d, want 20", m.pal.cursor)
	}
}
