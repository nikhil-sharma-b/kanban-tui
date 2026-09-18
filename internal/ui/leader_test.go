package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// pressBoardKeys feeds keys to the board one at a time, " " being the leader.
func pressBoardKeys(m *model, keys ...string) *model {
	for _, k := range keys {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		if k == " " {
			msg = tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
		}
		next, _ := m.updateBoard(msg)
		m = next.(*model)
	}
	return m
}

func TestLeaderSequenceStaysPendingUntilComplete(t *testing.T) {
	m, _ := newArchiveTestModel(t)

	m = pressBoardKeys(m, " ", "t")
	if got := strings.Join(m.pendingKeys, " "); got != "space t" {
		t.Fatalf("pendingKeys = %q, want %q", got, "space t")
	}
	if m.mode != modeBoard {
		t.Fatalf("mode = %v, want board while pending", m.mode)
	}

	m = pressBoardKeys(m, "n")
	if m.mode != modeCreate {
		t.Fatalf("mode = %v, want %v", m.mode, modeCreate)
	}
	if len(m.pendingKeys) != 0 {
		t.Fatalf("pendingKeys not cleared: %v", m.pendingKeys)
	}
}

func TestBareMutatingKeysNoLongerFire(t *testing.T) {
	m, task := newArchiveTestModel(t)

	for _, k := range []string{"x", "A", "n", "d"} {
		m = pressBoardKeys(m, k)
		if m.mode != modeBoard {
			t.Fatalf("bare %q changed mode to %v", k, m.mode)
		}
	}
	if m.board.Tasks[task.ID] == nil {
		t.Fatal("task removed by a bare key")
	}
}

func TestUnknownKeyAbandonsSequence(t *testing.T) {
	m, _ := newArchiveTestModel(t)

	m = pressBoardKeys(m, " ", "z")
	if len(m.pendingKeys) != 0 {
		t.Fatalf("pendingKeys = %v, want empty", m.pendingKeys)
	}
}

func TestWhichKeyListsContinuations(t *testing.T) {
	m, _ := newArchiveTestModel(t)

	m = pressBoardKeys(m, " ")
	float := m.renderWhichKey()
	for _, want := range []string{"SPC", "+task", "+column", "+archive", "+project"} {
		if !strings.Contains(float, want) {
			t.Fatalf("which-key float missing %q:\n%s", want, float)
		}
	}
	if !strings.Contains(m.View(), "+task") {
		t.Fatal("which-key float not composited into view")
	}

	m = pressBoardKeys(m, "t")
	if float := m.renderWhichKey(); !strings.Contains(float, "archive") || strings.Contains(float, "+column") {
		t.Fatalf("task submenu wrong:\n%s", float)
	}
}

func TestEveryBindingIsReachable(t *testing.T) {
	for _, b := range boardBindings {
		tokens := strings.Fields(b.keys)
		for i := 1; i < len(tokens); i++ {
			if _, ok := boardGroups[strings.Join(tokens[:i], " ")]; !ok {
				t.Errorf("binding %q: prefix %q is not a group", b.keys, strings.Join(tokens[:i], " "))
			}
		}
	}
}

func TestLeaderNewProjectCreatesAndReturnsToBoard(t *testing.T) {
	m, _ := newArchiveTestModel(t)

	m = pressBoardKeys(m, " ", "p", "n")
	if m.mode != modeProjectEdit || m.projectDraft != "" {
		t.Fatalf("mode = %v draft = %q, want new-project dialog", m.mode, m.projectDraft)
	}
	for _, r := range "Side quest" {
		next, _ := m.updateProjectEdit(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = next.(*model)
	}
	next, _ := m.updateProjectEdit(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(*model)

	if m.mode != modeBoard {
		t.Fatalf("mode = %v, want board", m.mode)
	}
	if m.project == nil || m.project.Name != "Side quest" {
		t.Fatalf("active project = %+v, want Side quest", m.project)
	}
}

func TestLeaderRenameProjectCancelReturnsToBoard(t *testing.T) {
	m, _ := newArchiveTestModel(t)

	m = pressBoardKeys(m, " ", "p", "r")
	if m.mode != modeProjectEdit || m.projectInput.Value() != m.project.Name {
		t.Fatalf("rename dialog not prefilled: mode=%v value=%q", m.mode, m.projectInput.Value())
	}
	next, _ := m.updateProjectEdit(tea.KeyMsg{Type: tea.KeyEsc})
	if got := next.(*model).mode; got != modeBoard {
		t.Fatalf("mode after esc = %v, want board", got)
	}
}
