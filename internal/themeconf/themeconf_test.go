package themeconf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/muesli/termenv"

	"github.com/nikhilsharma/kanban-tui/internal/ui/dusk"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPathSitsBesideTheDataFile(t *testing.T) {
	if got := Path("/data/kanban-tui/board.db"); got != "/data/kanban-tui/config.toml" {
		t.Fatalf("Path = %q", got)
	}
	if got := Path("/elsewhere/mine.json"); got != "/elsewhere/config.toml" {
		t.Fatalf("Path follows an overridden data file: got %q", got)
	}
}

func TestReadMissingFileIsDusk(t *testing.T) {
	s, err := Read(filepath.Join(t.TempDir(), "config.toml"))
	if err != nil || s != (Setting{Name: Dusk}) {
		t.Fatalf("Read = %+v, %v; want dusk", s, err)
	}
}

func TestReadMissingKeyIsDusk(t *testing.T) {
	s, err := Read(writeConfig(t, "# nothing here\nother = \"x\"\n"))
	if err != nil || s != (Setting{Name: Dusk}) {
		t.Fatalf("Read = %+v, %v; want dusk", s, err)
	}
}

func TestReadUnknownValueIsDuskAndNamesIt(t *testing.T) {
	s, err := Read(writeConfig(t, "theme = \"solarized\"\n"))
	if err != nil || s != (Setting{Name: Dusk, Unknown: "solarized"}) {
		t.Fatalf("Read = %+v, %v; want dusk with solarized unknown", s, err)
	}
	if got := s.Warning(); got != "unknown theme “solarized”, using dusk" {
		t.Fatalf("Warning = %q", got)
	}
}

func TestReadEachValidValue(t *testing.T) {
	for _, n := range Names {
		for _, body := range []string{"theme = \"" + string(n) + "\"", "  theme=\"" + string(n) + "\"  # mine", "theme = '" + string(n) + "'"} {
			s, err := Read(writeConfig(t, body+"\n"))
			if err != nil || s != (Setting{Name: n}) {
				t.Fatalf("Read(%q) = %+v, %v; want %s", body, s, err, n)
			}
		}
	}
}

func TestWriteCreatesThenReplacesKeepingOtherLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "config.toml")
	if err := Write(path, DuskLight); err != nil {
		t.Fatal(err)
	}
	if s, _ := Read(path); s.Name != DuskLight {
		t.Fatalf("after write, Read = %+v", s)
	}
	os.WriteFile(path, []byte("# mine\ntheme = \"dusk-light\"\nother = 1\n"), 0o644)
	if err := Write(path, Terminal); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	if string(b) != "# mine\ntheme = \"terminal\"\nother = 1\n" {
		t.Fatalf("file = %q", b)
	}
}

func answer(dark bool) func() (bool, bool) { return func() (bool, bool) { return dark, true } }
func noAnswer() (bool, bool)               { return false, false }

func env(bg func() (bool, bool), colorfgbg string) Env {
	return Env{Background: bg, Getenv: func(k string) string {
		if k == "COLORFGBG" {
			return colorfgbg
		}
		return ""
	}}
}

func TestPaletteForEachTheme(t *testing.T) {
	cases := []struct {
		name Name
		env  Env
		want string
	}{
		{Dusk, env(answer(false), ""), "dark"},
		{DuskLight, env(answer(true), ""), "light"},
		{Terminal, env(answer(false), ""), "light"},
		{Terminal, env(answer(true), ""), "dark"},
		// No answer: COLORFGBG decides, then dark.
		{Terminal, env(noAnswer, "0;15"), "light"},
		{Terminal, env(noAnswer, "15;0"), "dark"},
		{Terminal, env(noAnswer, "12;default;7"), "light"},
		{Terminal, env(noAnswer, ""), "dark"},
		{Terminal, env(nil, "junk"), "dark"},
	}
	for _, c := range cases {
		if got := Palette(c.name, c.env).Name; got != c.want {
			t.Errorf("Palette(%s, COLORFGBG=%q) = %s, want %s", c.name, c.env.Getenv("COLORFGBG"), got, c.want)
		}
	}
}

func TestFewColoursUseANSI16WhateverTheTheme(t *testing.T) {
	for _, n := range Names {
		for _, p := range []termenv.Profile{termenv.ANSI, termenv.Ascii} {
			e := env(answer(false), "")
			e.Profile = p
			if got := Palette(n, e); got.Name != dusk.ANSI16.Name {
				t.Errorf("Palette(%s, profile %v) = %s, want ansi16", n, p, got.Name)
			}
		}
		e := env(answer(true), "")
		e.Profile = termenv.ANSI256
		if got := Palette(n, e).Name; got == "ansi16" {
			t.Errorf("Palette(%s, 256 colours) = ansi16", n)
		}
	}
}

func TestHintNamesWhatEachThemeDoes(t *testing.T) {
	var got []string
	for _, n := range Names {
		got = append(got, string(n)+":"+n.Hint())
	}
	if s := strings.Join(got, ","); s != "dusk:dark,dusk-light:light,terminal:auto · from terminal bg" {
		t.Fatalf("hints = %s", s)
	}
}
