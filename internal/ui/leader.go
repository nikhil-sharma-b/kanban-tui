package ui

// Board keys resolve through one table, the way Neovim resolves mappings: bare
// keys only navigate, and anything that creates, mutates or opens something
// sits behind the leader (space). While a sequence is unfinished, a which-key
// float lists what may follow it. The float, the command palette and the
// resolver all read boardBindings.

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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
