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
	// WriteClipboard copies text (default: the system clipboard).
	WriteClipboard func(string) error
	// NewTask constructs a new task before it is appended (default domain.NewTask).
	NewTask func(title, description string) (*domain.Task, error)
	// LaunchWhiteboard opens a whiteboard file (default: the external editor).
	LaunchWhiteboard func(path string) error
	// ToastDuration is how long a toast shows (default 3 s).
	ToastDuration time.Duration
	// SaveRetryInterval is the wait before retrying a failed save (default 2 s).
	SaveRetryInterval time.Duration
	// ExternalCheckInterval is how often to look for writes by another
	// process (default 1 s).
	ExternalCheckInterval time.Duration
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
	if opts.WriteClipboard != nil {
		m.writeClipboard = opts.WriteClipboard
	}
	if opts.NewTask != nil {
		m.newTask = opts.NewTask
	}
	if opts.LaunchWhiteboard != nil {
		m.launchWhiteboard = opts.LaunchWhiteboard
	}
	if opts.ToastDuration > 0 {
		m.toastDuration = opts.ToastDuration
	}
	if opts.SaveRetryInterval > 0 {
		m.saveRetryInterval = opts.SaveRetryInterval
	}
	if opts.ExternalCheckInterval > 0 {
		m.externalInterval = opts.ExternalCheckInterval
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
	// Below narrowBelowCols the board is narrow: columns 2 cols apart and no
	// daily count in the header (80×24).
	narrowBelowCols = keybarWideMin
	narrowColumnGap = 2
	// Below compactBelowRows cards use Density=compact.
	compactBelowRows = 30
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
	// SyncUnreadable is the store-unreadable screen's header (page 15).
	SyncUnreadable
)

// AppHeader is the content of the App header component.
type AppHeader struct {
	Project string
	Counts  string
	Sync    Sync
	// Filter is the applied query, shown as a Badge chip after the project.
	Filter string
}

func drawAppHeader(g *termgrid.Grid, p dusk.Palette, x, y, width int, h AppHeader) {
	col := g.Text(x, y, "kanban-tui", tokenFG(p, dusk.Text, true))
	if h.Project != "" {
		col = g.Text(col+1, y, "›", tokenFG(p, dusk.TextFaint, false))
		col = g.Text(col+1, y, h.Project, tokenFG(p, dusk.Accent, false))
	}
	if h.Filter != "" {
		drawBadge(g, p, col+1, y, "/ "+h.Filter)
	}

	dot, text, label := dusk.Success, dusk.TextDim, "live"
	switch h.Sync {
	case SyncError:
		dot, text, label = dusk.Danger, dusk.Danger, "save failed · retrying"
	case SyncReloaded:
		dot, text, label = dusk.Accent, dusk.TextDim, "reloaded · changed by CLI"
	case SyncUnreadable:
		dot, text, label = dusk.Danger, dusk.Danger, "can't read board"
	}
	rightWidth := 2 + len([]rune(label))
	if h.Counts != "" {
		rightWidth += 3 + len([]rune(h.Counts))
	}
	col = x + width - rightWidth
	col = g.Text(col, y, "●", tokenFG(p, dot, false))
	col = g.Text(col+1, y, label, tokenFG(p, text, false))
	if h.Counts != "" {
		col = g.Text(col+1, y, "·", tokenFG(p, dusk.TextDim, false))
		g.Text(col+1, y, h.Counts, tokenFG(p, dusk.TextDim, false))
	}
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

func drawColumnHeader(g *termgrid.Grid, p dusk.Palette, x, y int, label string, active bool) int {
	if active {
		return g.Text(x, y, label, tokenFG(p, dusk.Accent, true))
	}
	return g.Text(x, y, label, tokenFG(p, dusk.TextDim, false))
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
	// Glyph, when set, leads Text in its own colour (an info "●").
	Glyph      string
	GlyphToken dusk.Token
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
		end := x + width - len([]rune(status.Text))
		g.Text(end, y, status.Text, tokenFG(p, status.Token, false))
		if status.Glyph != "" {
			g.Text(end-1-len([]rune(status.Glyph)), y, status.Glyph, tokenFG(p, status.GlyphToken, false))
		}
	}
}

// RenderKeybar draws the Keybar component, width cells wide.
func RenderKeybar(p dusk.Palette, size KeybarSize, width int, hints []KeyHint) string {
	g := termgrid.New(width, 1, tokenBG(p, dusk.Bg))
	drawKeybar(g, p, 0, 0, width, size, hints, KeybarStatus{})
	return renderGrid(g)
}

// ─── Column rail ─────────────────────────────────────────────────────────────

// railSide names the hidden columns on one side, nearest to the window first
// in reading order: "a · b", or with 3+ "doing +2" (right) / "+2 todo" (left).
func railSide(names []string, left bool) string {
	switch {
	case len(names) == 0:
		return ""
	case len(names) <= 2:
		return strings.Join(names, " · ")
	case left:
		return fmt.Sprintf("+%d %s", len(names)-1, names[len(names)-1])
	default:
		return fmt.Sprintf("%s +%d", names[0], len(names)-1)
	}
}

func drawColumnRail(g *termgrid.Grid, p dusk.Palette, x, y, width int, left, right []string) {
	st := tokenFG(p, dusk.TextDim, false)
	if s := railSide(left, true); s != "" {
		g.Text(x, y, "‹ "+s, st)
	}
	if s := railSide(right, false); s != "" {
		s += " ›"
		g.Text(x+width-len([]rune(s)), y, s, st)
	}
}

// RenderColumnRail draws the Column rail component: hidden columns left and
// right of the window, width cells wide.
func RenderColumnRail(p dusk.Palette, width int, left, right []string) string {
	g := termgrid.New(width, 1, tokenBG(p, dusk.Bg))
	drawColumnRail(g, p, 0, 0, width, left, right)
	return renderGrid(g)
}

// showRail reports whether hidden columns go in the Column rail. At four
// visible columns they are marked on the column headers instead (01/more-columns).
func (m *model) showRail() bool {
	_, n := m.boardWindow()
	return n < len(m.board.Statuses()) && n < duskMaxColumns
}

// ─── Board ───────────────────────────────────────────────────────────────────

func (m *model) appHeader() AppHeader {
	h := AppHeader{Sync: m.syncState()}
	if m.project != nil {
		h.Project = m.project.Name
	}
	if m.onDailyBoard() && m.filter == "" {
		// 11/daily: lowercase name and a count per column.
		h.Project = strings.ToLower(m.project.Name)
		var counts []string
		for _, status := range m.board.Statuses() {
			counts = append(counts, fmt.Sprintf("%d %s", len(m.board.Order[status]), status))
		}
		h.Counts = strings.Join(counts, " · ")
		total := m.board.ActiveTaskCount()
		if m.width < narrowBelowCols {
			h.Counts = plural(total, "task") // 11/daily-80x24
		} else if total == 0 {
			h.Counts = "0 on daily" // 11/empty
		}
		return h
	}
	open := 0
	for _, status := range m.board.Statuses() {
		if status == domain.StatusDone {
			continue
		}
		open += len(m.board.Order[status])
	}
	h.Counts = fmt.Sprintf("%d open", open)
	if m.filter != "" {
		matches, total := m.filterCounts()
		h.Counts = fmt.Sprintf("%d of %d match", matches, total)
		if m.mode != modeSearch {
			h.Filter = m.filter
		}
		return h
	}
	if first, n := m.boardWindow(); n < len(m.board.Statuses()) && !m.showRail() {
		h.Counts = fmt.Sprintf("columns %d–%d of %d", first+1, first+n, len(m.board.Statuses()))
		return h
	}
	// The daily count is left out at narrow widths (01/default-80x24).
	if !m.onDailyBoard() && m.workspace != nil && m.width >= narrowBelowCols {
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
	filtered := m.filter != ""
	// A filter keeps its keybar steady, matches or not (05/no-results).
	hasTask := filtered || m.boardSelectedTask() != nil
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
		add(wide || !hasTask, "SPC t n", "new") // 11/empty-80x24
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

func (m *model) keybarStatus(width int, size KeybarSize, hints []KeyHint) KeybarStatus {
	switch {
	case len(m.pendingKeys) > 0:
		return KeybarStatus{Text: displayKeys(strings.Join(m.pendingKeys, " ")) + " …", Token: dusk.Accent}
	case m.lastErr != nil:
		return KeybarStatus{Text: "✗ " + m.lastErr.Error(), Token: dusk.Danger}
	case m.createFeedback != "":
		text := "✓ " + m.createFeedback
		gap := keybarWideGap
		if size == KeybarNarrow {
			gap = keybarNarrowGap
		}
		for _, hint := range hints {
			width -= hint.width() + gap
		}
		if size == KeybarNarrow || len([]rune(text)) > width {
			text, _, _ = strings.Cut(text, " in ")
		}
		return KeybarStatus{Text: truncateCell(text, width), Token: dusk.Success}
	case m.lastInfo != "":
		return KeybarStatus{Text: m.lastInfo, Token: dusk.TextCard, Glyph: "●", GlyphToken: dusk.Accent}
	case m.lastStatus != "":
		return KeybarStatus{Text: "✓ " + m.lastStatus, Token: dusk.Success}
	}
	return KeybarStatus{}
}

// boardWindow returns the first visible column and how many are shown:
// N = clamp(floor(W/28), 1, 4). The window slides only as far as it must to
// keep the active column visible, and remembers where it was.
func (m *model) boardWindow() (int, int) {
	total := len(m.board.Statuses())
	n := min(total, max(1, min(duskMaxColumns, m.width/duskColumnPitch)))
	first := max(0, min(m.windowFirst, total-n))
	if m.activeColumn < first {
		first = m.activeColumn
	}
	if m.activeColumn >= first+n {
		first = m.activeColumn - n + 1
	}
	m.windowFirst = first
	return first, n
}

// renderDuskBoard draws the whole board screen on the cell grid.
func (m *model) duskBoard() *termgrid.Grid {
	p := m.palette
	g := termgrid.New(m.width, m.height, tokenBG(p, dusk.Bg))
	inner := m.width - 2*screenGutter

	drawAppHeader(g, p, screenGutter, 1, inner, m.appHeader())

	statuses := m.board.Statuses()
	first, n := m.boardWindow()
	rail := m.showRail()
	gap := duskColumnGap
	if m.width < narrowBelowCols {
		gap = narrowColumnGap
	}
	if n > 0 {
		colWidth := (inner - gap*(n-1)) / n
		// An empty board shows the add hint in the active column only; once
		// the board has tasks, every empty column shows it.
		boardEmpty := true
		for _, status := range statuses {
			if len(m.visible[status]) > 0 {
				boardEmpty = false
			}
		}
		for i := 0; i < n; i++ {
			idx := first + i
			status := statuses[idx]
			x := screenGutter + i*(colWidth+gap)
			active := idx == m.activeColumn
			hx := x
			if i == 0 && first > 0 && !rail {
				hx = g.Text(x, 3, "‹", tokenFG(p, dusk.TextDim, false)) + 1
			}
			label := columnLabel(status, len(m.visible[status]))
			if m.filter != "" {
				label = fmt.Sprintf("%s/%d", label, len(m.board.Order[status]))
			}
			if active && m.mode == modeRenameColumn {
				drawColumnHeaderEditing(g, p, hx, 3, m.columnInput.Value(), m.columnInput.Position(), len(m.visible[status]))
			} else {
				hx = drawColumnHeader(g, p, hx, 3, label, active)
			}
			if i == n-1 && first+n < len(statuses) && !rail {
				g.Text(hx+1, 3, "›", tokenFG(p, dusk.TextDim, false))
			}
			m.drawColumnBody(g, x, 4, colWidth, m.duskBodyHeight(), status, active && (m.mode != modeCreate || m.compact()), m.filter == "" && (active || !boardEmpty))
		}
	}

	if rail {
		var left, right []string
		for i, status := range statuses {
			name := strings.ToLower(status.Title())
			if i < first {
				left = append(left, name)
			} else if i >= first+n {
				right = append(right, name)
			}
		}
		drawColumnRail(g, p, screenGutter, m.height-4, inner, left, right)
	}

	size := KeybarWide
	if m.width < keybarWideMin {
		size = KeybarNarrow
	}
	hints := m.boardHints(size)
	status := m.keybarStatus(inner, size, hints)
	if m.mode == modeRenameColumn {
		// Rename edits the header in place; the keybar carries its keys
		// and a duplicate name (06/rename-in-place).
		hints, status = renameColumnHints, KeybarStatus{}
		if m.columnErr != nil {
			status = KeybarStatus{Text: "✗ " + m.columnErr.Error(), Token: dusk.Danger}
		}
	}
	if m.toast.text != "" {
		// The toast sits right-aligned one blank row above the keybar, and
		// above the Column rail when it shows (page 15).
		y := m.height - 4
		if rail {
			y--
		}
		toast := toastGrid(p, m.toast.kind, truncateCell(m.toast.text, inner-6), m.toast.hint)
		g.Blit(screenGutter+inner-toast.W, y, toast)
	}
	drawKeybar(g, p, screenGutter, m.height-2, inner, size, hints, status)
	return g
}
