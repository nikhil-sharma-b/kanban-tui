// Package themeconf reads and writes the theme setting (page 17): one key,
// theme, in an optional config.toml beside the data file. Nothing about the
// theme is ever stored in board.db or board.json.
package themeconf

import (
	"bufio"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/muesli/termenv"

	"github.com/nikhilsharma/kanban-tui/internal/ui/dusk"
)

// Name is a theme setting value.
type Name string

const (
	Dusk      Name = "dusk"
	DuskLight Name = "dusk-light"
	// Terminal picks dusk or dusk-light from the terminal's background.
	Terminal Name = "terminal"
)

// Names lists every theme in the order the picker shows them.
var Names = []Name{Dusk, DuskLight, Terminal}

// Hint is what the picker shows beside the theme.
func (n Name) Hint() string {
	switch n {
	case DuskLight:
		return "light"
	case Terminal:
		return "auto · from terminal bg"
	}
	return "dark"
}

func valid(n Name) bool {
	for _, v := range Names {
		if v == n {
			return true
		}
	}
	return false
}

// Path is config.toml in the data file's directory, so it follows an
// overridden data file.
func Path(dataPath string) string {
	return filepath.Join(filepath.Dir(dataPath), "config.toml")
}

// Setting is what config.toml asks for. Unknown holds a theme value the file
// names that does not exist; Name is then Dusk.
type Setting struct {
	Name    Name
	Unknown string
}

// Warning is the info toast for an unknown value, or "".
func (s Setting) Warning() string {
	if s.Unknown == "" {
		return ""
	}
	return "unknown theme “" + s.Unknown + "”, using dusk"
}

// Read reads the setting. A missing file or key means dusk.
func Read(path string) (Setting, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Setting{Name: Dusk}, nil
	}
	if err != nil {
		return Setting{Name: Dusk}, err
	}
	sc := bufio.NewScanner(strings.NewReader(string(b)))
	for sc.Scan() {
		value, ok := themeValue(sc.Text())
		if !ok {
			continue
		}
		if n := Name(value); valid(n) {
			return Setting{Name: n}, nil
		}
		return Setting{Name: Dusk, Unknown: value}, nil
	}
	return Setting{Name: Dusk}, nil
}

// themeValue returns the value of a top-level theme = "…" line.
func themeValue(line string) (string, bool) {
	key, value, ok := strings.Cut(line, "=")
	if !ok || strings.TrimSpace(key) != "theme" {
		return "", false
	}
	value = strings.TrimSpace(value)
	if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') {
		if end := strings.IndexByte(value[1:], value[0]); end >= 0 {
			if value[0] == '"' {
				if s, err := strconv.Unquote(value[:end+2]); err == nil {
					return s, true
				}
			}
			return value[1 : end+1], true
		}
	}
	if i := strings.IndexByte(value, '#'); i >= 0 {
		value = strings.TrimSpace(value[:i])
	}
	return value, true
}

// Write sets the theme, replacing an existing theme line and keeping every
// other line, or creating the file.
func Write(path string, n Name) error {
	line := "theme = " + strconv.Quote(string(n))
	b, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	var lines []string
	if len(b) > 0 {
		lines = strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	}
	replaced := false
	for i, l := range lines {
		if _, ok := themeValue(l); ok {
			lines[i], replaced = line, true
			break
		}
	}
	if !replaced {
		lines = append(lines, line)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}

// Env is what the terminal tells us.
type Env struct {
	// Profile is the colour depth; below 256 colours means the ansi16 mode.
	Profile termenv.Profile
	// Background asks the terminal for its background (OSC 11); ok is false
	// when it gives no answer.
	Background func() (dark, ok bool)
	// Getenv reads COLORFGBG, the fallback when the terminal gives no answer.
	Getenv func(string) string
}

// Detect reads the real terminal. Background asks the terminal each time
// it is called.
func Detect() Env {
	out := termenv.NewOutput(os.Stdout)
	return Env{
		Profile: out.EnvColorProfile(),
		Background: func() (bool, bool) {
			// termenv answers an RGB colour only from OSC 11; an ANSI colour
			// comes from COLORFGBG or its own default, which we read ourselves.
			rgb, ok := out.BackgroundColor().(termenv.RGBColor)
			if !ok {
				return false, false
			}
			_, _, l := termenv.ConvertToRGB(rgb).Hsl()
			return l < 0.5, true
		},
		Getenv: os.Getenv,
	}
}

// DarkBackground reports whether the terminal is dark: its own answer,
// else COLORFGBG, else dark.
func (e Env) DarkBackground() bool {
	if e.Background != nil {
		if dark, ok := e.Background(); ok {
			return dark
		}
	}
	if e.Getenv != nil {
		fields := strings.Split(e.Getenv("COLORFGBG"), ";")
		if bg, err := strconv.Atoi(fields[len(fields)-1]); err == nil {
			return !(bg == 7 || (bg >= 9 && bg <= 15))
		}
	}
	return true
}

// Palette is the Dusk palette for a theme in this terminal.
func Palette(n Name, e Env) dusk.Palette {
	if e.Profile > termenv.ANSI256 { // termenv orders TrueColor < ANSI256 < ANSI < Ascii
		return dusk.ANSI16
	}
	switch n {
	case DuskLight:
		return dusk.Light
	case Terminal:
		if !e.DarkBackground() {
			return dusk.Light
		}
	}
	return dusk.Dark
}
