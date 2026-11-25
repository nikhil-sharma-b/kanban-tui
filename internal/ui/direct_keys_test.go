package ui

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/nikhilsharma/kanban-tui/internal/sample"
)

// newSampleBoard is the 01/default workspace at 120×36 with backlog active.
func newSampleBoard(t *testing.T) *model {
	t.Helper()
	m := New(sample.Workspace(sample.Now), &stubWorkspaceStore{}, filepath.Join(t.TempDir(), "board.db")).(*model)
	m.width, m.height = 120, 36
	m.recalculateVisible()
	return m
}

func TestDirectKeysOpenNewAndEdit(t *testing.T) {
	m := newSampleBoard(t)
	if m = pressBoardKeys(m, "n"); m.mode != modeCreate {
		t.Fatalf("n opened mode %v, want create", m.mode)
	}

	m = newSampleBoard(t)
	if m = pressBoardKeys(m, "e"); m.mode != modeDetailEdit {
		t.Fatalf("e opened mode %v, want detail edit", m.mode)
	}
}

func TestDirectKeysMoveAndReorder(t *testing.T) {
	m := newSampleBoard(t)
	task := m.boardSelectedTask()
	from := task.Status

	m = pressBoardKeys(m, "L")
	if task.Status == from {
		t.Fatalf("L left the task in %s", from)
	}
	if m.boardSelectedTask() != task {
		t.Fatal("the selection did not follow the moved task")
	}
	m = pressBoardKeys(m, "H")
	if task.Status != from {
		t.Fatalf("H put the task in %s, want %s", task.Status, from)
	}

	index := func() int {
		for i, id := range m.board.Order[from] {
			if id == task.ID {
				return i
			}
		}
		return -1
	}
	at := index()
	if at < 1 {
		t.Fatalf("task back at %d, want below the first card", at)
	}
	if m = pressBoardKeys(m, "K"); index() != at-1 {
		t.Fatalf("K left the task at %d, want %d", index(), at-1)
	}
	if m = pressBoardKeys(m, "J"); index() != at {
		t.Fatalf("J left the task at %d, want %d", index(), at)
	}
}

// Every card is one height, so j and k never shift the cards around them.
func TestSelectionKeepsCardRowsStill(t *testing.T) {
	m := newSampleBoard(t)
	status := m.board.Statuses()[0]
	rows := func() []int {
		var out []int
		for _, p := range layoutCards(m.columnCards(status, true), 0, 0, m.duskBodyHeight()) {
			out = append(out, p.top)
		}
		return out
	}
	before := rows()
	m = pressBoardKeys(m, "j", "j")
	after := rows()
	for i := range before {
		if before[i] != after[i] {
			t.Fatalf("card %d moved from row %d to %d when the selection moved", i, before[i], after[i])
		}
	}
}

func TestColumnKeysLandOnTheNearestRow(t *testing.T) {
	m := newSampleBoard(t)
	statuses := m.board.Statuses()
	// The third backlog card sits beside the third todo card.
	m = pressBoardKeys(m, "j", "j", "l")
	if got := m.selected[statuses[1]]; got != 2 {
		t.Fatalf("l selected todo card %d, want 2", got)
	}
	// Further down than the next column reaches: its last card.
	m = pressBoardKeys(m, "h", "j", "j", "l")
	last := len(m.visible[statuses[1]]) - 1
	if got := m.selected[statuses[1]]; got != min(4, last) {
		t.Fatalf("l selected todo card %d, want %d", got, min(4, last))
	}
}

func TestPaletteSearchHandoffFiltersAtOnce(t *testing.T) {
	m := newSampleBoard(t)
	m = pressBoardKeys(m, ":")
	for _, r := range "zzqx" {
		next, _ := m.Update(keyMsgFor(string(r)))
		m = next.(*model)
	}
	next, _ := m.Update(keyMsgFor("enter"))
	m = next.(*model)
	if m.mode != modeSearch {
		t.Fatalf("mode = %v, want search", m.mode)
	}
	if matches, _ := m.filterCounts(); matches != 0 {
		t.Fatalf("%d tasks match “zzqx”, want 0: the board was not filtered", matches)
	}
}

// A half-width split pane (under 100 cols) shows one full-width column.
func TestNarrowPanesShowOneColumn(t *testing.T) {
	for _, c := range []struct{ width, want int }{{80, 1}, {92, 1}, {99, 1}, {100, 3}, {120, 4}} {
		m := newSampleBoard(t)
		m.width = c.width
		if _, n := m.boardWindow(); n != c.want {
			t.Errorf("%d cols show %d columns, want %d", c.width, n, c.want)
		}
	}
}

func TestMoveFollowsTheTaskInOneColumn(t *testing.T) {
	m := newSampleBoard(t)
	m.width = 92
	task := m.boardSelectedTask()
	m = pressBoardKeys(m, "L")
	first, _ := m.boardWindow()
	if first != m.activeColumn || m.boardSelectedTask() != task {
		t.Fatalf("after L the window shows column %d with %v selected, want the moved task's column", first, m.boardSelectedTask())
	}
}

// The one-column board names every column as a tab instead of a rail.
func TestOneColumnShowsColumnTabs(t *testing.T) {
	m := newSampleBoard(t)
	m.width, m.height = 92, 48
	if !m.columnTabs() || m.showRail() {
		t.Fatalf("92 cols: tabs %v, rail %v; want tabs and no rail", m.columnTabs(), m.showRail())
	}
	row := m.duskBoard().Rows()[3]
	for _, s := range m.board.Statuses() {
		if !strings.Contains(row, strings.ToLower(s.Title())) {
			t.Errorf("tab row %q is missing %s", row, s)
		}
	}
	m.width = 120
	if m.columnTabs() {
		t.Fatal("120 cols shows four columns, not tabs")
	}
}

// Tabs that would not fit give way to the single header and the rail.
func TestColumnTabsFallBackToTheRail(t *testing.T) {
	m := newSampleBoard(t)
	m.width, m.height = 40, 24
	if m.columnTabs() || !m.showRail() {
		t.Fatalf("40 cols: tabs %v, rail %v; want the rail", m.columnTabs(), m.showRail())
	}
}
