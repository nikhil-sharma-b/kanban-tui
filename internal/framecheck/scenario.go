package framecheck

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
	"github.com/nikhilsharma/kanban-tui/internal/sample"
	"github.com/nikhilsharma/kanban-tui/internal/ui"
	"github.com/nikhilsharma/kanban-tui/internal/ui/dusk"
)

// Scenario drives the real app into the state a frame shows.
type Scenario struct {
	Workspace func(now time.Time) *domain.Workspace
	W, H      int
	Palette   dusk.Palette
	// Keys are pressed in order after the first resize, spelled as
	// tea.KeyMsg.String() spells them ("space", "enter", "esc", "j").
	Keys []string
}

// Scenarios maps a frame name to how to reach it. Frames without one are
// reported as missing a scenario.
var Scenarios = map[string]Scenario{
	"01/default":     {Workspace: sample.Workspace, W: 120, H: 36, Palette: dusk.Dark, Keys: []string{"l", "j"}},
	"01/empty-board": {Workspace: sample.EmptyBoard, W: 120, H: 36, Palette: dusk.Dark},
}

// Run renders the scenario through the app's normal update loop and returns
// the final screen as ANSI.
func (s Scenario) Run() string {
	palette := s.Palette
	m := ui.NewWithOptions(s.Workspace(sample.Now), nil, "", ui.Options{
		Palette: &palette,
		Now:     func() time.Time { return sample.Now },
	})
	m, _ = m.Update(tea.WindowSizeMsg{Width: s.W, Height: s.H})
	for _, k := range s.Keys {
		m, _ = m.Update(keyMsg(k))
	}
	return m.View()
}

func keyMsg(k string) tea.KeyMsg {
	named := map[string]tea.KeyType{
		"enter": tea.KeyEnter, "esc": tea.KeyEsc, "tab": tea.KeyTab, "backspace": tea.KeyBackspace,
		"up": tea.KeyUp, "down": tea.KeyDown, "left": tea.KeyLeft, "right": tea.KeyRight,
		"ctrl+c": tea.KeyCtrlC, "ctrl+e": tea.KeyCtrlE,
	}
	if t, ok := named[k]; ok {
		return tea.KeyMsg{Type: t}
	}
	if k == "space" {
		return tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
}
