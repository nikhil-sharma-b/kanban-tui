package ui

// The theme setting (page 17): the palette's theme step applies a theme
// live and writes config.toml; a hand edit is picked up by live reload.

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhilsharma/kanban-tui/internal/themeconf"
	"github.com/nikhilsharma/kanban-tui/internal/ui/dusk"
)

// ThemeSetting connects the app to config.toml.
type ThemeSetting struct {
	// Setting is what config.toml asked for at start.
	Setting themeconf.Setting
	// Palette turns a theme into colours for this terminal.
	Palette func(themeconf.Name) dusk.Palette
	// Save writes the theme to config.toml (nil: nothing is written).
	Save func(themeconf.Name) error
	// Load rereads config.toml for live reload (nil: never reread).
	Load func() (themeconf.Setting, error)
}

func defaultThemeSetting() ThemeSetting {
	return ThemeSetting{
		Setting: themeconf.Setting{Name: themeconf.Dusk},
		Palette: func(n themeconf.Name) dusk.Palette { return themeconf.Palette(n, themeconf.Env{}) },
	}
}

// themePaletteItem opens the theme step. It is found by typing, so the
// unfiltered action list stays as page 12 shows it.
func (m *model) themePaletteItem() paletteItem {
	return paletteItem{label: "theme…", run: func() (tea.Model, tea.Cmd) {
		m.enterPaletteStep(paletteThemes)
		return m, nil
	}}
}

func (m *model) themeStepItems() []paletteItem {
	query := strings.TrimSpace(m.pal.input.Value())
	var items []paletteItem
	for _, n := range themeconf.Names {
		if query != "" && !fuzzyMatch(query, string(n)) {
			continue
		}
		hint := n.Hint()
		current := n == m.theme.Setting.Name
		if current {
			hint = "current · " + hint
		}
		name := n
		items = append(items, paletteItem{label: string(n), hint: hint, current: current, run: func() (tea.Model, tea.Cmd) {
			m.leavePalette()
			return m, m.setTheme(name)
		}})
	}
	return items
}

// setTheme applies a theme and writes it to config.toml.
func (m *model) setTheme(n themeconf.Name) tea.Cmd {
	if n == m.theme.Setting.Name {
		return nil
	}
	m.applyTheme(themeconf.Setting{Name: n})
	if m.theme.Save != nil {
		if err := m.theme.Save(n); err != nil {
			return m.showToast(StatusDanger, fmt.Sprintf("Could not save theme: %v", err), KeyHint{})
		}
	}
	return nil
}

func (m *model) applyTheme(s themeconf.Setting) {
	m.theme.Setting = s
	m.palette = m.theme.Palette(s.Name)
}

// themeWarning toasts an unknown theme value.
func (m *model) themeWarning() tea.Cmd {
	if w := m.theme.Setting.Warning(); w != "" {
		return m.showToast(StatusInfo, w, KeyHint{})
	}
	return nil
}

// reloadTheme picks up a hand edit of config.toml.
func (m *model) reloadTheme() tea.Cmd {
	if m.theme.Load == nil {
		return nil
	}
	s, err := m.theme.Load()
	if err != nil || s == m.theme.Setting {
		return nil
	}
	m.applyTheme(s)
	return m.themeWarning()
}
