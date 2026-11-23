package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
	"github.com/nikhilsharma/kanban-tui/internal/store"
	"github.com/nikhilsharma/kanban-tui/internal/termgrid"
	"github.com/nikhilsharma/kanban-tui/internal/ui/dusk"
)

// Reopen opens the store and loads the workspace again.
type Reopen func() (store.WorkspaceStore, *domain.Workspace, error)

// unreadableModel is the store-unreadable screen (15/store-unreadable). It
// holds no store, so nothing can ever be written to the unreadable board.
type unreadableModel struct {
	dataPath  string
	shownPath string
	err       error
	reopen    Reopen
	opts      Options
	palette   dusk.Palette
	width     int
	height    int
	more      bool
}

// NewUnreadable shows why the board at shownPath could not be read: r runs
// reopen and, once it succeeds, swaps in the board; q quits.
func NewUnreadable(dataPath, shownPath string, err error, reopen Reopen, opts Options) tea.Model {
	m := &unreadableModel{dataPath: dataPath, shownPath: shownPath, err: err, reopen: reopen, opts: opts, palette: dusk.Dark}
	if opts.Palette != nil {
		m.palette = *opts.Palette
	}
	return m
}

func (m *unreadableModel) Init() tea.Cmd { return nil }

func (m *unreadableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "?":
			m.more = !m.more
		case "r":
			if m.reopen == nil {
				return m, nil
			}
			st, workspace, err := m.reopen()
			if err != nil {
				m.err = err
				return m, nil
			}
			board := NewWithOptions(workspace, st, m.dataPath, m.opts)
			board, _ = board.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
			return board, board.Init()
		}
	}
	return m, nil
}

var unreadableHints = []KeyHint{{Key: "r", Label: "retry"}, {Key: "q", Label: "quit"}, {Key: "?", Label: "more"}}

const unreadableDialogWidth = 60

func (m *unreadableModel) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}
	p := m.palette
	g := termgrid.New(m.width, m.height, tokenBG(p, dusk.Bg))
	drawAppHeader(g, p, screenGutter, 1, m.width-2*screenGutter, AppHeader{Sync: SyncUnreadable})

	w := min(unreadableDialogWidth, m.width)
	textWidth := w - 5
	reason := strings.TrimSuffix(m.err.Error(), ".")
	lines := wrapDialogText(m.shownPath+": "+reason+".", textWidth)
	if m.more {
		lines = append(lines, "")
		lines = append(lines, wrapDialogText("kanban-tui never writes to a board it can't read. Restore a backup or move the file aside, then press r to retry.", textWidth)...)
	}
	h := 8 + len(lines)
	d := newDialog(p, w, h)
	d.Text(3, 1, "Can't read the board", tokenFG(p, dusk.Text, true))
	for i, line := range lines {
		d.Text(3, 3+i, line, tokenFG(p, dusk.TextMuted, false))
	}
	drawStatusMessage(d, p, 3, 4+len(lines), textWidth, StatusInfo, "nothing was changed or overwritten")
	drawDialogHints(d, p, 3, h-2, textWidth, unreadableHints)
	g.Blit((m.width-w)/2, (m.height-h)/2, d)
	return renderGrid(g)
}
