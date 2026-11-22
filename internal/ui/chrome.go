package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
	"github.com/nikhilsharma/kanban-tui/internal/store"
	"github.com/nikhilsharma/kanban-tui/internal/termgrid"
	"github.com/nikhilsharma/kanban-tui/internal/ui/dusk"
)

// Options configure a model beyond its data. Zero values pick the defaults.
type Options struct {
	// Palette is the Dusk theme to draw with (default dusk.Dark).
	Palette *dusk.Palette
	// Now is the clock used for relative times (default time.Now).
	Now func() time.Time
}

// NewWithOptions is New with a theme and clock.
func NewWithOptions(workspace *domain.Workspace, boardStore store.WorkspaceStore, dataPath string, opts Options) tea.Model {
	m := New(workspace, boardStore, dataPath).(*model)
	if opts.Palette != nil {
		m.palette = *opts.Palette
	}
	if opts.Now != nil {
		m.now = opts.Now
	}
	return m
}

// Board layout at keybar width and above, in cells (DESIGN.md "Layout").
const (
	screenGutter    = 2
	duskColumnGap   = 4
	duskMaxColumns  = 4
	duskColumnPitch = 28
	keybarWideGap   = 3
	keybarNarrowGap = 2
	keybarWideMin   = 100
)

// clock is the model's now, defaulting to time.Now for hand-built models.
func (m *model) clock() time.Time {
	if m.now == nil {
		return time.Now()
	}
	return m.now()
}

// colorString turns a palette colour into the string termgrid stores.
func colorString(c lipgloss.TerminalColor) string {
	if s, ok := c.(lipgloss.Color); ok {
		return string(s)
	}
	return ""
}

func tokenBG(p dusk.Palette, t dusk.Token) string { return colorString(p.Get(t).Color) }

func tokenFG(p dusk.Palette, t dusk.Token, bold bool) termgrid.Style {
	c := p.Get(t)
	return termgrid.Style{FG: colorString(c.Color), Bold: bold, Faint: c.Faint}
}

func renderGrid(g *termgrid.Grid) string {
	return g.Render(lipgloss.ColorProfile())
}

// ─── App header ──────────────────────────────────────────────────────────────

// Sync is the App header's sync state.
type Sync int

const (
	SyncLive Sync = iota
	SyncError
	SyncReloaded
)

// AppHeader is the content of the App header component.
type AppHeader struct {
	Project string
	Counts  string
	Sync    Sync
}

func drawAppHeader(g *termgrid.Grid, p dusk.Palette, x, y, width int, h AppHeader) {
	col := g.Text(x, y, "kanban-tui", tokenFG(p, dusk.Text, true))
	col = g.Text(col+1, y, "›", tokenFG(p, dusk.TextFaint, false))
	g.Text(col+1, y, h.Project, tokenFG(p, dusk.Accent, false))

	dot, text, label := dusk.Success, dusk.TextDim, "live"
	switch h.Sync {
	case SyncError:
		dot, text, label = dusk.Danger, dusk.Danger, "save failed · retrying"
	case SyncReloaded:
		dot, text, label = dusk.Accent, dusk.TextDim, "reloaded · changed by CLI"
	}
	rightWidth := 2 + len([]rune(label)) + 3 + len([]rune(h.Counts))
	col = x + width - rightWidth
	col = g.Text(col, y, "●", tokenFG(p, dot, false))
	col = g.Text(col+1, y, label, tokenFG(p, text, false))
	col = g.Text(col+1, y, "·", tokenFG(p, dusk.TextDim, false))
	g.Text(col+1, y, h.Counts, tokenFG(p, dusk.TextDim, false))
}

// RenderAppHeader draws the App header component on its own, width cells wide.
func RenderAppHeader(p dusk.Palette, width int, h AppHeader) string {
	g := termgrid.New(width, 1, tokenBG(p, dusk.Bg))
	drawAppHeader(g, p, 0, 0, width, h)
	return renderGrid(g)
}

// ─── Column header ───────────────────────────────────────────────────────────

func columnLabel(status domain.Status, count int) string {
	return fmt.Sprintf("%s · %d", strings.ToLower(status.Title()), count)
}

func drawColumnHeader(g *termgrid.Grid, p dusk.Palette, x, y int, label string, active bool) {
	if active {
		g.Text(x, y, label, tokenFG(p, dusk.Accent, true))
		return
	}
	g.Text(x, y, label, tokenFG(p, dusk.TextDim, false))
}

// RenderColumnHeader draws the Column header component (State=normal or active).
func RenderColumnHeader(p dusk.Palette, label string, active bool) string {
	g := termgrid.New(len([]rune(label)), 1, tokenBG(p, dusk.Bg))
	drawColumnHeader(g, p, 0, 0, label, active)
	return renderGrid(g)
}

// ─── Key hint and keybar ─────────────────────────────────────────────────────

// KeyHint is one "key label" pair.
type KeyHint struct {
	Key   string
	Label string
}

func (h KeyHint) width() int { return len([]rune(h.Key)) + 1 + len([]rune(h.Label)) }

func drawKeyHint(g *termgrid.Grid, p dusk.Palette, x, y int, h KeyHint) int {
	col := g.Text(x, y, h.Key, tokenFG(p, dusk.TextCard, false))
	return g.Text(col+1, y, h.Label, tokenFG(p, dusk.TextDim, false))
}

// RenderKeyHint draws the Key hint component.
func RenderKeyHint(p dusk.Palette, h KeyHint) string {
	g := termgrid.New(h.width(), 1, tokenBG(p, dusk.Bg))
	drawKeyHint(g, p, 0, 0, h)
	return renderGrid(g)
}

// KeybarSize is the Keybar component's Size variant.
type KeybarSize int

const (
	KeybarWide KeybarSize = iota
	KeybarNarrow
)

// KeybarStatus is the optional right-aligned message in the keybar's status slot.
type KeybarStatus struct {
	Text  string
	Token dusk.Token
}

func drawKeybar(g *termgrid.Grid, p dusk.Palette, x, y, width int, size KeybarSize, hints []KeyHint, status KeybarStatus) {
	gap := keybarWideGap
	if size == KeybarNarrow {
		gap = keybarNarrowGap
	}
	col := x
	for i, h := range hints {
		if i > 0 {
			col += gap
		}
		col = drawKeyHint(g, p, col, y, h)
	}
	if status.Text != "" {
		g.Text(x+width-len([]rune(status.Text)), y, status.Text, tokenFG(p, status.Token, false))
	}
}

// RenderKeybar draws the Keybar component, width cells wide.
func RenderKeybar(p dusk.Palette, size KeybarSize, width int, hints []KeyHint) string {
	g := termgrid.New(width, 1, tokenBG(p, dusk.Bg))
	drawKeybar(g, p, 0, 0, width, size, hints, KeybarStatus{})
	return renderGrid(g)
}

// ─── Board ───────────────────────────────────────────────────────────────────

func (m *model) appHeader() AppHeader {
	h := AppHeader{Sync: SyncLive}
	if m.project != nil {
		h.Project = m.project.Name
	}
	open := 0
	for _, status := range m.board.Statuses() {
		if status == domain.StatusDone {
			continue
		}
		open += len(m.board.Order[status])
	}
	h.Counts = fmt.Sprintf("%d open", open)
	if !m.onDailyBoard() && m.workspace != nil {
		if daily := m.workspace.DailyProject(); daily != nil && daily.Board != nil {
			if n := daily.Board.ActiveTaskCount(); n > 0 {
				h.Counts += fmt.Sprintf(" · %d on daily", n)
			}
		}
	}
	return h
}

// boardHints is the board keybar (DESIGN.md "The Board Keybar Rule"). Hints
// that need a selected task are left out when there is none.
func (m *model) boardHints(size KeybarSize) []KeyHint {
	hasTask := m.selectedTask() != nil
	filtered := m.filter != ""
	var hints []KeyHint
	add := func(ok bool, key, label string) {
		if ok {
			hints = append(hints, KeyHint{key, label})
		}
	}
	wide := size == KeybarWide
	add(true, "h j k l", "select")
	add(hasTask, "↵", "open")
	if m.onDailyBoard() {
		add(wide, "SPC t n", "new")
		add(wide && hasTask, "SPC d m", "done")
		add(hasTask, "SPC d p", "promote")
		add(wide, "SPC d d", "back")
		add(true, "SPC", "more")
		add(true, "?", "help")
		return hints
	}
	add(filtered, "/", "refine")
	add(!filtered && wide, "/", "search")
	add(!filtered || wide, "SPC t n", "new")
	add(wide && hasTask, "SPC t h l", "move")
	add(filtered, "esc", "clear filter")
	add(!filtered, "SPC", "more")
	add(true, "?", "help")
	return hints
}

func (m *model) keybarStatus() KeybarStatus {
	switch {
	case len(m.pendingKeys) > 0:
		return KeybarStatus{Text: displayKeys(strings.Join(m.pendingKeys, " ")) + " …", Token: dusk.Accent}
	case m.lastErr != nil:
		return KeybarStatus{Text: "✗ " + m.lastErr.Error(), Token: dusk.Danger}
	case m.lastStatus != "":
		return KeybarStatus{Text: "✓ " + m.lastStatus, Token: dusk.Success}
	}
	return KeybarStatus{}
}

// boardWindow returns the first visible column and how many are shown:
// N = clamp(floor(W/28), 1, 4), scrolled so the active column is visible.
func (m *model) boardWindow() (int, int) {
	total := len(m.board.Statuses())
	n := min(total, max(1, min(duskMaxColumns, m.width/duskColumnPitch)))
	first := 0
	if m.activeColumn >= n {
		first = m.activeColumn - n + 1
	}
	return first, n
}

// renderDuskBoard draws the whole board screen on the cell grid.
func (m *model) renderDuskBoard() string {
	p := m.palette
	g := termgrid.New(m.width, m.height, tokenBG(p, dusk.Bg))
	inner := m.width - 2*screenGutter

	drawAppHeader(g, p, screenGutter, 1, inner, m.appHeader())

	statuses := m.board.Statuses()
	first, n := m.boardWindow()
	if n > 0 {
		colWidth := (inner - duskColumnGap*(n-1)) / n
		bodyTop, bodyBottom := 4, m.height-3 // rows 5..H-3, 1-based
		for i := 0; i < n; i++ {
			idx := first + i
			status := statuses[idx]
			x := screenGutter + i*(colWidth+duskColumnGap)
			active := idx == m.activeColumn
			drawColumnHeader(g, p, x, 3, columnLabel(status, len(m.visible[status])), active)
			m.drawColumnBody(g, x, bodyTop, colWidth, bodyBottom-bodyTop, status, active)
		}
	}

	size := KeybarWide
	if m.width < keybarWideMin {
		size = KeybarNarrow
	}
	drawKeybar(g, p, screenGutter, m.height-2, inner, size, m.boardHints(size), m.keybarStatus())
	return renderGrid(g)
}

func (m *model) drawColumnBody(g *termgrid.Grid, x, y, width, height int, status domain.Status, active bool) {
	ids := m.visible[status]
	if len(ids) == 0 {
		if active {
			drawKeyHint(g, m.palette, x, y+1, KeyHint{"SPC t n", "add a task"})
		}
		return
	}
	// Cards keep their pre-Dusk look until the board ticket redraws them.
	row := y
	for i := m.scroll[status]; i < len(ids) && row < y+height; i++ {
		task := m.board.Tasks[ids[i]]
		if task == nil {
			continue
		}
		card := m.renderTaskCard(task, width, active && i == m.selected[status], theme.Mauve)
		cg := termgrid.Parse(card, width, lipgloss.Height(card))
		if row+cg.H > y+height {
			break
		}
		g.Blit(x, row, cg)
		row += cg.H
	}
}
