package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) openCheatSheet() (tea.Model, tea.Cmd) {
	return m.openPaletteAt(paletteKeys, false)
}

func (m *model) cheatSheetSections() []paletteSection {
	entry := func(label, keys string, run func() (tea.Model, tea.Cmd)) paletteItem {
		return paletteItem{label: label, hint: paletteKeyHint(keys), keys: keys, run: func() (tea.Model, tea.Cmd) {
			m.leavePalette()
			return run()
		}}
	}
	action := func(label string, action boardAction) paletteItem {
		for _, binding := range boardBindings {
			if binding.action == action {
				return entry(label, binding.keys, func() (tea.Model, tea.Cmd) { return m.runBoardAction(action) })
			}
		}
		panic("missing cheat sheet binding: " + label)
	}
	start := []paletteItem{
		entry("select", "h j k l", func() (tea.Model, tea.Cmd) {
			if m.activeColumn == len(m.board.Statuses())-1 {
				return m.runBoardAction(actColumnLeft)
			}
			return m.runBoardAction(actColumnRight)
		}),
		action("open task", actOpen),
		action("new task", actTaskNew),
	}
	rest := []paletteItem{
		action("edit task", actTaskEdit),
		entry("move task", "space t h l", func() (tea.Model, tea.Cmd) {
			move := actTaskMoveRight
			if m.activeColumn == len(m.board.Statuses())-1 {
				move = actTaskMoveLeft
			}
			return m.runBoardAction(move)
		}),
		action("search", actSearch),
		// Page 18 shows the project group prefix; Enter opens its switcher.
		entry("projects", "space p", func() (tea.Model, tea.Cmd) { return m.runBoardAction(actProjects) }),
		action("command palette", actPalette),
		entry("focus mode", "z", func() (tea.Model, tea.Cmd) { return m.updateBoard(keyMsgFor("z")) }),
		entry("leader menu", "space", func() (tea.Model, tea.Cmd) {
			return m.updateBoard(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
		}),
		action("archive task", actTaskArchive),
		action("delete task", actTaskDelete),
		action("whiteboards", actTaskWhiteboards),
		action("daily board", actDailyToggle),
		action("view archive", actArchiveView),
	}
	if query := strings.TrimSpace(m.pal.input.Value()); query != "" {
		items := rankPaletteItems(query, append(start, rest...))
		if len(items) == 0 {
			return nil
		}
		return []paletteSection{{title: fmt.Sprintf("matches · %d", len(items)), items: items}}
	}
	return []paletteSection{{title: "start here", items: start}, {title: "everything else · : lists all", items: rest}}
}
