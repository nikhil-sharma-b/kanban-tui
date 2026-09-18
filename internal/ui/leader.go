package ui

// Board keys resolve through one table, the way Neovim resolves mappings: bare
// keys only navigate, and anything that creates, mutates or opens something
// sits behind the leader (space). While a sequence is unfinished, a which-key
// float lists what may follow it. The float, the help overlay and the
// resolver all read boardBindings, so a key cannot do one thing and be
// documented as another.

import (
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
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

	actTaskNew
	actTaskEdit
	actTaskDelete
	actTaskArchive
	actTaskMoveLeft
	actTaskMoveRight
	actTaskReorderUp
	actTaskReorderDown
	actTaskWhiteboards

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
}

var boardBindings = []boardBinding{
	{"h", actColumnLeft, "column left"},
	{"l", actColumnRight, "column right"},
	{"k", actTaskUp, "prev task"},
	{"j", actTaskDown, "next task"},
	{"left", actColumnLeft, "column left"},
	{"right", actColumnRight, "column right"},
	{"up", actTaskUp, "prev task"},
	{"down", actTaskDown, "next task"},
	{"g g", actTaskTop, "first task"},
	{"G", actTaskBottom, "last task"},
	{"/", actSearch, "search"},
	{"enter", actOpen, "details"},
	{"?", actHelp, "help"},
	{"q", actQuit, "quit"},
	{"ctrl+c", actQuit, "quit"},

	{"space t n", actTaskNew, "new"},
	{"space t e", actTaskEdit, "edit"},
	{"space t x", actTaskDelete, "delete"},
	{"space t a", actTaskArchive, "archive"},
	{"space t h", actTaskMoveLeft, "move left"},
	{"space t l", actTaskMoveRight, "move right"},
	{"space t k", actTaskReorderUp, "reorder up"},
	{"space t j", actTaskReorderDown, "reorder down"},
	{"space t w", actTaskWhiteboards, "whiteboards"},

	{"space c n", actColumnNew, "new"},
	{"space c r", actColumnRename, "rename"},
	{"space c d", actColumnDelete, "delete"},
	{"space c h", actColumnMoveLeft, "move left"},
	{"space c l", actColumnMoveRight, "move right"},

	{"space p p", actProjects, "list / switch"},
	{"space p n", actProjectNew, "new"},
	{"space p r", actProjectRename, "rename current"},
	{"space p x", actProjectDelete, "delete current"},

	{"space a v", actArchiveView, "view"},
	{"space a o", actArchiveOld, "archive old done"},

	{"space d d", actDailyToggle, "daily board / back"},
	{"space d m", actDailyDone, "mark done"},
	{"space d p", actDailyPromote, "promote to project"},
	{"space d X", actDailyClear, "clear board"},
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

// continuations lists the keys that may follow prefix, one entry per key.
func continuations(prefix string) []whichKeyEntry {
	seen := map[string]bool{}
	var out []whichKeyEntry
	add := func(full string, desc string, group bool) {
		rest, ok := strings.CutPrefix(full, prefix+" ")
		if !ok || strings.Contains(rest, " ") || seen[rest] {
			return
		}
		seen[rest] = true
		out = append(out, whichKeyEntry{key: rest, desc: desc, group: group})
	}
	for p, name := range boardGroups {
		add(p, "+"+name, true)
	}
	for _, b := range boardBindings {
		add(b.keys, b.desc, false)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].group != out[j].group {
			return out[i].group
		}
		return out[i].key < out[j].key
	})
	return out
}

func displayKeys(seq string) string {
	return strings.ReplaceAll(seq, leaderKey, "SPC")
}

// renderWhichKey draws the float for the pending sequence, or "" when none.
func (m *model) renderWhichKey() string {
	if len(m.pendingKeys) == 0 {
		return ""
	}
	prefix := strings.Join(m.pendingKeys, " ")
	entries := continuations(prefix)
	if len(entries) == 0 {
		return ""
	}
	keyStyle := lipgloss.NewStyle().Foreground(theme.Mauve).Bold(true)
	arrowStyle := lipgloss.NewStyle().Foreground(theme.Overlay0)
	groupStyle := lipgloss.NewStyle().Foreground(theme.Blue)
	descStyle := lipgloss.NewStyle().Foreground(theme.Text)

	keyWidth := 0
	for _, e := range entries {
		keyWidth = max(keyWidth, ansi.StringWidth(e.key))
	}
	lines := make([]string, 0, len(entries)+1)
	lines = append(lines, lipgloss.NewStyle().Foreground(theme.Subtext0).Bold(true).Render(displayKeys(prefix)))
	for _, e := range entries {
		style := descStyle
		if e.group {
			style = groupStyle
		}
		lines = append(lines, keyStyle.Render(e.key+strings.Repeat(" ", keyWidth-ansi.StringWidth(e.key)))+
			arrowStyle.Render(" → ")+style.Render(e.desc))
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Mauve).
		Padding(0, 1).
		Render(strings.Join(lines, "\n"))
}

// placeFloatBottomRight composites float over base without dimming it, just
// above bottomMargin rows, so the board stays readable while a sequence is
// finished.
func (m *model) placeFloatBottomRight(base, float string, bottomMargin int) string {
	baseLines := strings.Split(base, "\n")
	floatLines := strings.Split(float, "\n")
	width := lipgloss.Width(float)
	left := max(m.width-width-2, 0)
	top := max(len(baseLines)-bottomMargin-len(floatLines), 0)
	for i, fl := range floatLines {
		row := top + i
		if row >= len(baseLines) {
			break
		}
		line := baseLines[row]
		if w := ansi.StringWidth(line); w < m.width {
			line += strings.Repeat(" ", m.width-w)
		}
		head := ansi.Truncate(line, left, "")
		if pad := left - ansi.StringWidth(head); pad > 0 {
			head += strings.Repeat(" ", pad)
		}
		tail := ansi.TruncateLeft(line, left+ansi.StringWidth(fl), "")
		baseLines[row] = head + fl + tail
	}
	return strings.Join(baseLines, "\n")
}

// renderHelpOverlay lists every board binding, grouped the way the leader
// tree is.
func (m *model) renderHelpOverlay() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(theme.Mauve).Render("◆  Keys")
	keyStyle := lipgloss.NewStyle().Foreground(theme.Mauve).Bold(true)
	headStyle := lipgloss.NewStyle().Foreground(theme.Blue).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(theme.Text)

	// One row per action: arrow keys and ctrl+c are aliases of keys already
	// listed, and repeating them only makes the overlay taller.
	listed := map[boardAction]bool{}
	section := func(name, prefix string) string {
		lines := []string{headStyle.Render(name)}
		for _, b := range boardBindings {
			if listed[b.action] || !bindingInSection(b.keys, prefix) {
				continue
			}
			listed[b.action] = true
			k := displayKeys(b.keys)
			lines = append(lines, keyStyle.Render(k+strings.Repeat(" ", max(10-len(k), 1)))+descStyle.Render(b.desc))
		}
		return strings.Join(lines, "\n")
	}
	gap := "    "
	left := section("Navigate", "")
	middle := lipgloss.JoinVertical(lipgloss.Left, section("+task", "space t"), "", section("+column", "space c"))
	right := lipgloss.JoinVertical(lipgloss.Left, section("+project", "space p"), "", section("+daily", "space d"), "", section("+archive", "space a"))
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, gap, middle, gap, right)
	hint := lipgloss.NewStyle().Foreground(theme.Overlay0).Render("esc / ? close")
	lines := []string{title, "", body, "", hint}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Mauve).
		Padding(1, 3).
		Render(strings.Join(lines, "\n"))
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
