package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/nikhilsharma/kanban-tui/internal/sample"
)

func projectsModel(t *testing.T) *model {
	t.Helper()
	m := NewWithOptions(sample.Projects(sample.Now), nil, "", Options{}).(*model)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	return next.(*model)
}

func TestSwitcherFiltersAndOpens(t *testing.T) {
	m := projectsModel(t)
	m = pressKeys(t, m, typed(" ppbench")...)
	m = pressKeys(t, m, enterKey)
	if m.mode != modeBoard || m.project.Name != "workbench" {
		t.Fatalf("mode %v project %q, want workbench board", m.mode, m.project.Name)
	}
}

func TestSwitcherEscClearsQueryIntoManagementKeepingSelection(t *testing.T) {
	m := projectsModel(t)
	m = pressKeys(t, m, typed(" ppwor")...)
	m = pressKeys(t, m, tea.KeyMsg{Type: tea.KeyDown}, escKey)
	if m.mode != modeProjects || m.projectFiltering || m.projectFilterInput.Value() != "" {
		t.Fatalf("esc should clear the query and return to management")
	}
	if got := m.filteredProjects()[m.projectCursor].Name; got != "workbench" {
		t.Fatalf("selection = %q, want workbench", got)
	}
	m = pressKeys(t, m, escKey)
	if m.mode != modeBoard {
		t.Fatalf("esc in management should close, mode %v", m.mode)
	}
}

func TestSwitcherListsCountsIncludingDoneAndCurrentMarker(t *testing.T) {
	m := projectsModel(t)
	m = pressKeys(t, m, typed(" pp")...)
	view := ansi.Strip(m.View())
	if !strings.Contains(view, "current    14 tasks") || !strings.Contains(view, "8 tasks") {
		t.Fatalf("counts or current marker missing:\n%s", view)
	}
	if strings.Contains(strings.ToLower(view[strings.Index(view, "╭"):]), "daily") {
		t.Fatal("daily board listed")
	}
	m = pressKeys(t, m, escKey)
	m = pressKeys(t, m, typed("j")...)
	m = pressKeys(t, m, enterKey)
	m = pressKeys(t, m, typed(" pp")...)
	for _, line := range strings.Split(ansi.Strip(m.View()), "\n") {
		if strings.Contains(line, "current") && !strings.Contains(line, "work ") {
			t.Fatalf("current marker should follow the active project: %q", line)
		}
	}
}

func TestLastProjectCannotBeDeleted(t *testing.T) {
	m := confirmModel(t)
	m = pressKeys(t, m, typed(" pp")...)
	m = pressKeys(t, m, escKey)
	m = pressKeys(t, m, typed("x")...)
	if m.mode != modeProjects {
		t.Fatalf("x on the last project opened mode %v", m.mode)
	}
	if !strings.Contains(ansi.Strip(m.View()), "The last project can’t be deleted.") {
		t.Fatal("last-project error not shown")
	}
	if strings.Contains(ansi.Strip(m.View()), "x delete") {
		t.Fatal("last project should not offer delete")
	}
	m = pressKeys(t, m, escKey)
	m = pressKeys(t, m, typed(" px")...)
	if m.mode != modeBoard || len(m.workspace.RegularProjects()) != 1 {
		t.Fatalf("SPC p x on the last project opened mode %v", m.mode)
	}
}
