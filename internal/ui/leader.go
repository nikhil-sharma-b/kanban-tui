package ui

// Board keys resolve through one table, the way Neovim resolves mappings: bare
// keys only navigate, and anything that creates, mutates or opens something
// sits behind the leader (space). While a sequence is unfinished, a which-key
// float lists what may follow it. The float, the help overlay and the
// resolver all read boardBindings, so a key cannot do one thing and be
// documented as another.

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/nikhilsharma/kanban-tui/internal/termgrid"
	"github.com/nikhilsharma/kanban-tui/internal/ui/dusk"
)

type boardAction int

const (
	actNone boardAction = iota
	actQuit
	actHelp
	actColumnLeft
	actColumnRight
	actTaskUp
	actTaskDown
	actTaskTop
	actTaskBottom
	actSearch
	actOpen
	actPalette

	actTaskNew
	actTaskEdit
	actTaskDelete
	actTaskArchive
	actTaskMoveLeft
	actTaskMoveRight
	actTaskReorderUp
	actTaskReorderDown
	actTaskWhiteboards
	actTaskCopyID

	actColumnNew
	actColumnRename
	actColumnDelete
	actColumnMoveLeft
	actColumnMoveRight

	actProjects
	actProjectNew
	actProjectRename
	actProjectDelete
	actArchiveView
	actArchiveOld

	actDailyToggle
	actDailyDone
	actDailyPromote
	actDailyClear
)

// leaderKey is the token the space bar normalizes to.
const leaderKey = "space"

type boardBinding struct {
	keys   string // space-separated tokens, e.g. "space t n"
	action boardAction
	desc   string
	// label names the action in the command palette, where it is read
	// without its group; empty falls back to desc.
	label string
}

var boardBindings = []boardBinding{
	{"h", actColumnLeft, "column left", "select column left"},
	{"l", actColumnRight, "column right", "select column right"},
	{"k", actTaskUp, "prev task", "select previous task"},
	{"j", actTaskDown, "next task", "select next task"},
	{"left", actColumnLeft, "column left", "select column left"},
	{"right", actColumnRight, "column right", "select column right"},
	{"up", actTaskUp, "prev task", "select previous task"},
	{"down", actTaskDown, "next task", "select next task"},
	{"g g", actTaskTop, "first task", ""},
	{"G", actTaskBottom, "last task", ""},
	{"/", actSearch, "search", "search tasks"},
	{"enter", actOpen, "details", "open details"},
	{":", actPalette, "command palette", ""},
	{"?", actHelp, "help", ""},
	{"q", actQuit, "quit", ""},
	{"ctrl+c", actQuit, "quit", ""},

	{"space t n", actTaskNew, "new", "new task"},
	{"space t e", actTaskEdit, "edit", "edit task"},
	{"space t x", actTaskDelete, "delete", "delete task"},
	{"space t a", actTaskArchive, "archive", "archive task"},
	{"space t h", actTaskMoveLeft, "move left", "move task left"},
	{"space t l", actTaskMoveRight, "move right", "move task right"},
	{"space t k", actTaskReorderUp, "reorder up", "move task up"},
	{"space t j", actTaskReorderDown, "reorder down", "move task down"},
	{"space t w", actTaskWhiteboards, "whiteboards", "whiteboards…"},
	{"space t y", actTaskCopyID, "copy ID", "copy task ID"},

	{"space c n", actColumnNew, "new", "new column"},
	{"space c r", actColumnRename, "rename", "rename column"},
	{"space c d", actColumnDelete, "delete", "delete column"},
	{"space c h", actColumnMoveLeft, "move left", "move column left"},
	{"space c l", actColumnMoveRight, "move right", "move column right"},

	{"space p p", actProjects, "list / switch", "switch project…"},
	{"space p n", actProjectNew, "new", "new project"},
	{"space p r", actProjectRename, "rename current", "rename project"},
	{"space p x", actProjectDelete, "delete current", "delete project"},

	{"space a v", actArchiveView, "view", "view archive"},
	{"space a o", actArchiveOld, "archive old done", ""},

	{"space d d", actDailyToggle, "daily board / back", "toggle daily board"},
	{"space d m", actDailyDone, "mark done", "mark daily task done"},
	{"space d p", actDailyPromote, "promote to project", "promote daily task"},
	{"space d X", actDailyClear, "clear board", "clear daily board"},
}

// boardGroups names the prefixes, drawn as "+name" like which-key.nvim.
var boardGroups = map[string]string{
	"g":       "goto",
	"space":   "leader",
	"space t": "task",
	"space c": "column",
	"space a": "archive",
	"space p": "project",
	"space d": "daily",
}

// normalizeKey turns a keypress into a binding token.
func normalizeKey(msg tea.KeyMsg) string {
	s := msg.String()
	if s == " " {
		return leaderKey
	}
	return s
}

// resolveBoardKey feeds one key into the pending sequence. It returns the
// action fired, or actNone while the sequence is unfinished or abandoned.
func (m *model) resolveBoardKey(key string) boardAction {
	if key == "esc" {
		m.pendingKeys = nil
		return actNone
	}
	seq := strings.Join(append(append([]string(nil), m.pendingKeys...), key), " ")
	for _, b := range boardBindings {
		if b.keys == seq {
			m.pendingKeys = nil
			return b.action
		}
	}
	if _, ok := boardGroups[seq]; ok {
		m.pendingKeys = strings.Fields(seq)
		return actNone
	}
	m.pendingKeys = nil
	return actNone
}

// peekBoardAction reports what key would fire without consuming it, so a
// board can refuse an action before the sequence resolves.
func (m *model) peekBoardAction(key string) boardAction {
	seq := strings.Join(append(append([]string(nil), m.pendingKeys...), key), " ")
	for _, b := range boardBindings {
		if b.keys == seq {
			return b.action
		}
	}
	return actNone
}

type whichKeyEntry struct {
	key   string
	desc  string
	group bool
}

// continuations lists the keys that may follow prefix, one entry per key:
// groups first, then actions, each in the order the bindings table lists them.
func continuations(prefix string) []whichKeyEntry {
	seen := map[string]bool{}
	var groups, actions []whichKeyEntry
	for _, b := range boardBindings {
		rest, ok := strings.CutPrefix(b.keys, prefix+" ")
		if !ok {
			continue
		}
		key, _, nested := strings.Cut(rest, " ")
		if seen[key] {
			continue
		}
		seen[key] = true
		if nested {
			if name, ok := boardGroups[prefix+" "+key]; ok {
				groups = append(groups, whichKeyEntry{key: key, desc: "+" + name, group: true})
			}
			continue
		}
		actions = append(actions, whichKeyEntry{key: key, desc: b.desc})
	}
	return append(groups, actions...)
}

func displayKeys(seq string) string {
	return strings.ReplaceAll(seq, leaderKey, "SPC")
}

// helpOverlay lists every board binding, grouped the way the leader tree is.
func (m *model) helpOverlay() *termgrid.Grid {
	// One row per action: arrow keys and ctrl+c are aliases of keys already
	// listed, and repeating them only makes the overlay taller.
	type row struct{ key, label string }
	type section struct{ title, prefix string }
	listed := map[boardAction]bool{}
	var columns [][]row
	for _, sections := range [][]section{
		{{"Navigate", ""}},
		{{"+task", "space t"}, {"+column", "space c"}},
		{{"+project", "space p"}, {"+daily", "space d"}, {"+archive", "space a"}},
	} {
		var rows []row
		for i, s := range sections {
			if i > 0 {
				rows = append(rows, row{})
			}
			rows = append(rows, row{label: s.title})
			for _, b := range boardBindings {
				if listed[b.action] || !bindingInSection(b.keys, s.prefix) {
					continue
				}
				listed[b.action] = true
				rows = append(rows, row{displayKeys(b.keys), b.desc})
			}
		}
		columns = append(columns, rows)
	}
	keyWidth := func(key string) int {
		if key == "" {
			return 0
		}
		return max(10, ansi.StringWidth(key)+1)
	}
	widths := make([]int, len(columns))
	bodyW, bodyH := 0, 0
	for i, rows := range columns {
		bodyH = max(bodyH, len(rows))
		for _, r := range rows {
			widths[i] = max(widths[i], keyWidth(r.key)+ansi.StringWidth(r.label))
		}
		bodyW += widths[i]
	}
	pad, gap := 4, 4
	if bodyW+2*pad+(len(columns)-1)*gap > m.width {
		pad, gap = 2, 1
	}
	w := bodyW + 2*pad + (len(columns)-1)*gap
	p := m.palette
	g := newDialog(p, min(w, m.width), min(bodyH+8, m.height))
	if g.W < 2*pad || g.H < 8 {
		return g
	}
	draw := func(x, y int, s string, token dusk.Token, bold bool) {
		g.Text(x, y, truncateCell(s, g.W-1-x), tokenFG(p, token, bold))
	}
	draw(pad, 2, "Keys", dusk.Text, true)
	x := pad
	for i, rows := range columns {
		for j, r := range rows {
			y := 4 + j
			if y >= g.H-3 {
				break
			}
			if r.key == "" {
				draw(x, y, r.label, dusk.Meta, true)
			} else {
				draw(x, y, r.key, dusk.TextMuted, true)
				draw(x+keyWidth(r.key), y, r.label, dusk.Text, false)
			}
		}
		x += widths[i] + gap
	}
	drawDialogHints(g, p, pad, g.H-3, g.W-pad-1, []KeyHint{{"esc / ?", "close"}})
	return g
}

func bindingInSection(keys, prefix string) bool {
	switch prefix {
	case "":
		return !strings.HasPrefix(keys, leaderKey+" ")
	case leaderKey:
		// Leader keys that belong to no named group.
		rest, ok := strings.CutPrefix(keys, leaderKey+" ")
		return ok && !strings.Contains(rest, " ")
	default:
		return strings.HasPrefix(keys, prefix+" ")
	}
}
