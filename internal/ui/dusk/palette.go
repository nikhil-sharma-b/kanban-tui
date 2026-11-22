// Package dusk holds the Dusk colour tokens from DESIGN.md "Tokens → lipgloss".
// Rendering code asks a Palette for a token; it never names a colour itself.
package dusk

import "github.com/charmbracelet/lipgloss"

// Token names one Dusk colour variable, spelled as in design.figcraft.
type Token string

const (
	Bg              Token = "bg"
	Surface         Token = "surface"
	SurfaceDone     Token = "surface-done"
	SurfaceSelected Token = "surface-selected"
	Border          Token = "border"
	Accent          Token = "accent"
	Text            Token = "text"
	TextCard        Token = "text-card"
	TextMuted       Token = "text-muted"
	TextDim         Token = "text-dim"
	TextFaint       Token = "text-faint"
	TextDone        Token = "text-done"
	Meta            Token = "meta"
	Success         Token = "success"
	Danger          Token = "danger"
)

// Tokens lists every token in DESIGN.md order.
var Tokens = []Token{
	Bg, Surface, SurfaceDone, SurfaceSelected, Border, Accent,
	Text, TextCard, TextMuted, TextDim, TextFaint, TextDone, Meta,
	Success, Danger,
}

// Color is one token's value in one theme. Faint is set where the ansi16
// column asks for the faint attribute on top of the colour.
type Color struct {
	Color lipgloss.TerminalColor
	Faint bool
}

// Palette is one theme: one field per token.
type Palette struct {
	Name            string
	Bg              Color
	Surface         Color
	SurfaceDone     Color
	SurfaceSelected Color
	Border          Color
	Accent          Color
	Text            Color
	TextCard        Color
	TextMuted       Color
	TextDim         Color
	TextFaint       Color
	TextDone        Color
	Meta            Color
	Success         Color
	Danger          Color
}

// Get returns the value of a token.
func (p Palette) Get(t Token) Color {
	switch t {
	case Bg:
		return p.Bg
	case Surface:
		return p.Surface
	case SurfaceDone:
		return p.SurfaceDone
	case SurfaceSelected:
		return p.SurfaceSelected
	case Border:
		return p.Border
	case Accent:
		return p.Accent
	case Text:
		return p.Text
	case TextCard:
		return p.TextCard
	case TextMuted:
		return p.TextMuted
	case TextDim:
		return p.TextDim
	case TextFaint:
		return p.TextFaint
	case TextDone:
		return p.TextDone
	case Meta:
		return p.Meta
	case Success:
		return p.Success
	case Danger:
		return p.Danger
	}
	return Color{Color: lipgloss.NoColor{}}
}

func hex(s string) Color       { return Color{Color: lipgloss.Color(s)} }
func ansi(s string) Color      { return Color{Color: lipgloss.Color(s)} }
func ansiFaint(s string) Color { return Color{Color: lipgloss.Color(s), Faint: true} }
func none() Color              { return Color{Color: lipgloss.NoColor{}} }

// Dark is the "dusk" theme.
var Dark = Palette{
	Name:            "dark",
	Bg:              hex("#17151A"),
	Surface:         hex("#1F1C23"),
	SurfaceDone:     hex("#1B191E"),
	SurfaceSelected: hex("#2A2430"),
	Border:          hex("#2E2A33"),
	Accent:          hex("#D4A5C7"),
	Text:            hex("#E2DBE6"),
	TextCard:        hex("#C9C2CD"),
	TextMuted:       hex("#A39BA8"),
	TextDim:         hex("#6E6675"),
	TextFaint:       hex("#4A4550"),
	TextDone:        hex("#4F4955"),
	Meta:            hex("#9A8296"),
	Success:         hex("#9FC6A6"),
	Danger:          hex("#E28C8C"),
}

// Light is the "dusk-light" theme.
var Light = Palette{
	Name:            "light",
	Bg:              hex("#F7F3F6"),
	Surface:         hex("#EEE8ED"),
	SurfaceDone:     hex("#F3EFF2"),
	SurfaceSelected: hex("#E8D9E4"),
	Border:          hex("#DDD5DC"),
	Accent:          hex("#8E4F7E"),
	Text:            hex("#221E26"),
	TextCard:        hex("#3A333F"),
	TextMuted:       hex("#5F5765"),
	TextDim:         hex("#756D7A"),
	TextFaint:       hex("#928B96"),
	TextDone:        hex("#8F8892"),
	Meta:            hex("#745A6F"),
	Success:         hex("#3F7D4E"),
	Danger:          hex("#B44A4A"),
}

// ANSI16 is used whatever the theme when the terminal has fewer than 256 colours.
var ANSI16 = Palette{
	Name:            "ansi16",
	Bg:              none(),
	Surface:         ansi("0"),
	SurfaceDone:     none(),
	SurfaceSelected: ansi("8"),
	Border:          ansi("8"),
	Accent:          ansi("5"),
	Text:            ansi("15"),
	TextCard:        ansi("7"),
	TextMuted:       ansi("7"),
	TextDim:         ansi("8"),
	TextFaint:       ansiFaint("8"),
	TextDone:        ansiFaint("8"),
	Meta:            ansiFaint("5"),
	Success:         ansi("2"),
	Danger:          ansi("1"),
}

// ByMode returns the palette for a design.figcraft mode name (dark, light, ansi16).
func ByMode(mode string) (Palette, bool) {
	switch mode {
	case "dark":
		return Dark, true
	case "light":
		return Light, true
	case "ansi16":
		return ANSI16, true
	}
	return Palette{}, false
}
