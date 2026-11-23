package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/nikhilsharma/kanban-tui/internal/sample"
)

func whiteboardsModel(t *testing.T, launched *[]string) *model {
	t.Helper()
	m := NewWithOptions(sample.Whiteboards(sample.Now), nil, "", Options{
		LaunchWhiteboard: func(path string) error { *launched = append(*launched, path); return nil },
	}).(*model)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	return pressKeys(t, next.(*model), typed("lj")...)
}

func TestWhiteboardListEscReturnsToOpener(t *testing.T) {
	var launched []string
	m := whiteboardsModel(t, &launched)
	m = pressKeys(t, m, typed(" tw")...)
	if m.mode != modeWhiteboards {
		t.Fatalf("SPC t w opened mode %v", m.mode)
	}
	m = pressKeys(t, m, escKey)
	if m.mode != modeBoard {
		t.Fatalf("esc from SPC t w returned to mode %v, want board", m.mode)
	}

	empty := NewWithOptions(sample.WhiteboardsEmpty(sample.Now), nil, "", Options{}).(*model)
	next, _ := empty.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	m = pressKeys(t, next.(*model), typed("lj")...)
	m = pressKeys(t, m, enterKey)
	m = pressKeys(t, m, typed("w")...)
	if m.mode != modeWhiteboards {
		t.Fatalf("w in detail without whiteboards opened mode %v", m.mode)
	}
	m = pressKeys(t, m, escKey)
	if m.mode != modeDetail {
		t.Fatalf("esc returned to mode %v, want detail", m.mode)
	}
}

func TestWhiteboardOpenLaunchesEditorAndConfirms(t *testing.T) {
	var launched []string
	m := whiteboardsModel(t, &launched)
	m = pressKeys(t, m, typed(" tw")...)
	m = pressKeys(t, m, enterKey)
	if len(launched) != 1 || !strings.Contains(launched[0], "undo-flow") {
		t.Fatalf("launched %v, want undo-flow", launched)
	}
	if !strings.Contains(ansi.Strip(m.View()), "opened undo-flow") {
		t.Fatal("open confirmation missing")
	}
	m = pressKeys(t, m, typed("j")...)
	if strings.Contains(ansi.Strip(m.View()), "opened undo-flow") {
		t.Fatal("open confirmation should clear on the next key")
	}
}
