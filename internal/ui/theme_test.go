package ui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhilsharma/kanban-tui/internal/sample"
	"github.com/nikhilsharma/kanban-tui/internal/themeconf"
	"github.com/nikhilsharma/kanban-tui/internal/ui/dusk"
)

type fakeConfig struct {
	setting themeconf.Setting
	saved   []themeconf.Name
}

func themedModel(t *testing.T, cfg *fakeConfig) *model {
	t.Helper()
	m := NewWithOptions(sample.Workspace(sample.Now), nil, "", Options{
		Now: func() time.Time { return sample.Now },
		Theme: &ThemeSetting{
			Setting: cfg.setting,
			Palette: func(n themeconf.Name) dusk.Palette { return themeconf.Palette(n, themeconf.Env{}) },
			Save: func(n themeconf.Name) error {
				cfg.saved = append(cfg.saved, n)
				cfg.setting = themeconf.Setting{Name: n}
				return nil
			},
			Load: func() (themeconf.Setting, error) { return cfg.setting, nil },
		},
	}).(*model)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	return next.(*model)
}

func TestThemeStartsFromConfig(t *testing.T) {
	m := themedModel(t, &fakeConfig{setting: themeconf.Setting{Name: themeconf.DuskLight}})
	if m.palette.Name != "light" {
		t.Fatalf("palette = %s, want light", m.palette.Name)
	}
}

func TestUnknownThemeToastsOnStart(t *testing.T) {
	m := themedModel(t, &fakeConfig{setting: themeconf.Setting{Name: themeconf.Dusk, Unknown: "neon"}})
	m.Init()
	if m.palette.Name != "dark" || m.toast.text != "unknown theme “neon”, using dusk" || m.toast.kind != StatusInfo {
		t.Fatalf("palette = %s, toast = %+v", m.palette.Name, m.toast)
	}
}

func TestThemeIsFoundByTypingNotListed(t *testing.T) {
	m := palettePress(t, themedModel(t, &fakeConfig{setting: themeconf.Setting{Name: themeconf.Dusk}}), ":")
	if paletteLabels(m)["theme…"] {
		t.Fatal("theme… is in the unfiltered list; page 12 lists 31 actions")
	}
	m = palettePress(t, m, "t", "h", "e", "m", "e")
	if items := m.paletteItems(); len(items) == 0 || items[0].label != "theme…" {
		t.Fatalf("matches for theme = %+v", items)
	}
}

func TestPaletteThemeAppliesLiveAndWritesConfig(t *testing.T) {
	cfg := &fakeConfig{setting: themeconf.Setting{Name: themeconf.Dusk}}
	m := palettePress(t, themedModel(t, cfg), ":", "t", "h", "e", "m", "e", "enter")
	items := m.paletteItems()
	if len(items) != 3 || !items[0].current || items[0].hint != "current · dark" || m.pal.cursor != 1 {
		t.Fatalf("theme step = %+v cursor %d; want dusk current and the cursor on dusk-light", items, m.pal.cursor)
	}
	m = palettePress(t, m, "enter")
	if m.mode != modeBoard || m.palette.Name != "light" || len(cfg.saved) != 1 || cfg.saved[0] != themeconf.DuskLight {
		t.Fatalf("mode %v palette %s saved %v", m.mode, m.palette.Name, cfg.saved)
	}
	// The current theme is disabled: picking it again writes nothing.
	m = palettePress(t, m, ":", "t", "h", "e", "m", "e", "enter")
	m.pal.cursor = 1
	m = palettePress(t, m, "enter")
	if len(cfg.saved) != 1 {
		t.Fatalf("picking the current theme saved again: %v", cfg.saved)
	}
}

func TestHandEditedThemeIsPickedUpByLiveReload(t *testing.T) {
	cfg := &fakeConfig{setting: themeconf.Setting{Name: themeconf.Dusk}}
	m := themedModel(t, cfg)
	cfg.setting = themeconf.Setting{Name: themeconf.DuskLight}
	next, _ := m.Update(externalCheckMsg{})
	m = next.(*model)
	if m.palette.Name != "light" {
		t.Fatalf("palette = %s after a hand edit, want light", m.palette.Name)
	}
	cfg.setting = themeconf.Setting{Name: themeconf.Dusk, Unknown: "x"}
	next, _ = m.Update(externalCheckMsg{})
	m = next.(*model)
	if m.palette.Name != "dark" || m.toast.text != "unknown theme “x”, using dusk" {
		t.Fatalf("palette = %s toast = %q", m.palette.Name, m.toast.text)
	}
}
