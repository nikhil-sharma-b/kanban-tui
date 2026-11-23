package framecheck

import (
	"errors"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
	"github.com/nikhilsharma/kanban-tui/internal/sample"
	"github.com/nikhilsharma/kanban-tui/internal/store"
	"github.com/nikhilsharma/kanban-tui/internal/themeconf"
	"github.com/nikhilsharma/kanban-tui/internal/ui"
	"github.com/nikhilsharma/kanban-tui/internal/ui/dusk"
)

// Scenario drives the real app into the state a frame shows.
type Scenario struct {
	Workspace func(now time.Time) *domain.Workspace
	W, H      int
	Palette   dusk.Palette
	// Keys are pressed in order after the first resize, spelled as
	// tea.KeyMsg.String() spells them ("space", "enter", "esc", "j").
	Keys []string
	// WriteClipboard stands in for the desktop clipboard when a key copies text.
	WriteClipboard func(string) error
	NewTask        func(string, string) (*domain.Task, error)
	Blur           bool // capture the cursor's hidden phase after typing
	// Settle runs the command the last key returned against a store that
	// discards writes and feeds its result back, so a save can finish.
	Settle bool
	// Loaded changes the workspace after the app has loaded it, for states
	// loading would repair.
	Loaded func(*domain.Workspace)
	// Store replaces the store that discards writes, e.g. one that fails.
	Store store.WorkspaceStore
	// Reload lets the app notice an external write and reload the board.
	Reload bool
	// After are keys pressed once saves and reloads have settled.
	After []string
	// WaitWhichKey runs the timer the last key started, so the which-key
	// popover shows.
	WaitWhichKey bool
	// Unreadable opens the store-unreadable screen with this error.
	Unreadable error
	// PaletteRecent seeds the command palette's recent actions.
	PaletteRecent []string
}

// Scenarios maps a frame name to how to reach it. Frames without one are
// reported as missing a scenario.
var Scenarios = withLightScenarios(map[string]Scenario{
	"01/default":     {Workspace: sample.Workspace, W: 120, H: 36, Palette: dusk.Dark, Keys: []string{"l", "j"}},
	"01/empty-board": {Workspace: sample.EmptyBoard, W: 120, H: 36, Palette: dusk.Dark},
	// The doing column is empty; todo stays active.
	"01/empty-column": {Workspace: sample.EmptyColumn, W: 120, H: 36, Palette: dusk.Dark, Keys: []string{"l", "j"}},
	// Selecting the 9th of 16 backlog tasks scrolls the column by 2.
	"01/overflow-column": {Workspace: sample.OverflowColumn, W: 120, H: 36, Palette: dusk.Dark, Keys: []string{"j", "j", "j", "j", "j", "j", "j", "j"}},
	// Out to done slides the window to columns 2–5; back to todo keeps it.
	"01/more-columns": {Workspace: sample.MoreColumns, W: 120, H: 36, Palette: dusk.Dark, Keys: []string{"l", "l", "l", "l", "h", "h", "j"}},
	// 80×24: two columns, compact cards, the rest named in the Column rail.
	"01/default-80x24": {Workspace: sample.Workspace, W: 80, H: 24, Palette: dusk.Dark, Keys: []string{"l", "j"}},
	"02/board-80x24":   {Workspace: sample.Workspace, W: 80, H: 24, Palette: dusk.Dark, Keys: []string{"l", "j"}},
	// Out to doing slides the window one column right.
	"02/move-right-80x24": {Workspace: sample.Workspace, W: 80, H: 24, Palette: dusk.Dark, Keys: []string{"l", "j", "l"}},
	"03/view":             {Workspace: sample.TaskDetail, W: 120, H: 36, Palette: dusk.Dark, Keys: []string{"l", "j", "enter"}},
	"02/dialog-80x24":     {Workspace: sample.Workspace, W: 80, H: 24, Palette: dusk.Dark, Keys: []string{"l", "j", "space", "t", "x"}},
	"03/no-whiteboards":   {Workspace: sample.TaskDetailWithoutWhiteboards, W: 120, H: 36, Palette: dusk.Dark, Keys: []string{"l", "j", "enter"}},
	"03/whiteboards":      {Workspace: sample.TaskDetail, W: 120, H: 36, Palette: dusk.Dark, Keys: []string{"l", "j", "enter", "w"}},
	"03/copied":           {Workspace: sample.TaskDetail, W: 120, H: 36, Palette: dusk.Dark, Keys: []string{"l", "j", "enter", "y"}, WriteClipboard: func(string) error { return nil }},
	"03/view-80x24":       {Workspace: sample.TaskDetail, W: 80, H: 24, Palette: dusk.Dark, Keys: []string{"l", "j", "enter"}},
	"03/edit-title":       {Workspace: sample.TaskDetail, W: 120, H: 36, Palette: dusk.Dark, Keys: []string{"l", "j", "enter", "e"}},
	// Insert the extra space after the third period, then return to the top.
	"03/edit-description":   {Workspace: sample.TaskDetail, W: 120, H: 36, Palette: dusk.Dark, Keys: []string{"l", "j", "enter", "e", "tab", "esc", "f", ".", "f", ".", "f", ".", "a", "space", "esc", "g", "g", "i"}},
	"03/empty-title":        {Workspace: sample.TaskDetail, W: 120, H: 36, Palette: dusk.Dark, Keys: []string{"l", "j", "enter", "e", "esc", "d", "d", ":", "w", "q", "enter"}},
	"03/unsaved":            {Workspace: sample.TaskDetail, W: 120, H: 36, Palette: dusk.Dark, Keys: []string{"l", "j", "enter", "e", "tab", "!", "esc", "esc"}},
	"03/status-picker":      {Workspace: sample.TaskDetail, W: 120, H: 36, Palette: dusk.Dark, Keys: []string{"l", "j", "enter", "s", "j"}},
	"03/command-line":       {Workspace: sample.TaskDetail, W: 120, H: 36, Palette: dusk.Dark, Keys: []string{"l", "j", "enter", "e", "tab", "esc", ":", "w", "q"}},
	"03/command-line-80x24": {Workspace: sample.TaskDetail, W: 80, H: 24, Palette: dusk.Dark, Keys: []string{"l", "j", "enter", "e", "tab", "esc", ":", "w", "q"}},
	"04/empty":              {Workspace: sample.Workspace, W: 120, H: 36, Palette: dusk.Dark, Keys: []string{"l", "space", "t", "n"}},
	"04/typing":             newTaskScenario(120, 36, newTaskTitle),
	"04/typing-80x24":       narrowNewTaskScenario(newTaskTitle),
	"04/empty-title":        newTaskScenario(120, 36, "enter"),
	"04/description":        newTaskDescriptionScenario(),
	"04/column-picker":      newTaskScenario(120, 36, newTaskTitle, "tab", newTaskLine1, "enter", newTaskLine2, "esc", "s", "j"),
	"04/command-line":       newTaskScenario(120, 36, newTaskTitle, "tab", newTaskLine1, "enter", newTaskLine2, "esc", ":", "w", "q"),
	"04/created":            newTaskCreatedScenario(120, 36),
	"04/created-80x24":      newTaskCreatedScenario(80, 24),
	"01/filter-active":      searchScenario(120, 36, "enter"),
	// Typing "CLI" over the todo selection previews the two matches.
	"05/typing":                  searchScenario(120, 36),
	"05/live-filtered":           searchScenario(120, 36, "enter"),
	"05/no-results":              searchScenario(120, 36, "x"),
	"05/filter-persisting":       searchScenario(120, 36, "enter", "l"),
	"05/cleared":                 searchScenario(120, 36, "enter", "esc"),
	"05/typing-80x24":            searchScenario(80, 24),
	"05/live-filtered-80x24":     searchScenario(80, 24, "enter"),
	"05/no-results-80x24":        searchScenario(80, 24, "x"),
	"05/filter-persisting-80x24": searchScenario(80, 24, "enter", "l"),
	"05/cleared-80x24":           searchScenario(80, 24, "enter", "esc"),
	"09/delete-task":             confirmScenario(sample.Workspace, 120, 36, "space", "t", "x"),
	"09/delete-task-80x24":       confirmScenario(sample.Workspace, 80, 24, "space", "t", "x"),
	"09/archive-task":            confirmScenario(sample.Workspace, 120, 36, "space", "t", "a"),
	"09/archive-old":             confirmScenario(sample.ArchiveOld, 120, 36, "space", "a", "o"),
	"09/archive-old-80x24":       confirmScenario(sample.ArchiveOld, 80, 24, "space", "a", "o"),
	"09/delete-whiteboard":       confirmScenario(sample.TaskDetail, 120, 36, "space", "t", "w", "x"),
	"09/delete-column":           confirmScenario(sample.Workspace, 120, 36, "space", "c", "d"),
	"09/delete-column-matched":   confirmScenario(sample.Workspace, 120, 36, "space", "c", "d", "t", "o", "d", "o"),
	"09/clear-daily":             clearDailyScenario(),
	"09/clear-daily-matched":     clearDailyScenario("d", "a", "i", "l", "y"),
	"06/add":                     confirmScenario(sample.Workspace, 120, 36, "space", "c", "n", "r", "e", "v", "i", "e", "w"),
	"06/add-80x24":               confirmScenario(sample.Workspace, 80, 24, "space", "c", "n", "r", "e", "v", "i", "e", "w"),
	"06/duplicate-name":          confirmScenario(sample.Workspace, 120, 36, "space", "c", "n", "t", "o", "d", "o", "enter"),
	"06/duplicate-name-80x24":    confirmScenario(sample.Workspace, 80, 24, "space", "c", "n", "t", "o", "d", "o", "enter"),
	"06/rename-in-place":         confirmScenario(sample.Workspace, 120, 36, "space", "c", "r", "backspace", "backspace", "backspace", "backspace", "r", "e", "a", "d", "y"),
	"06/rename-in-place-80x24":   confirmScenario(sample.Workspace, 80, 24, "space", "c", "r", "backspace", "backspace", "backspace", "backspace", "r", "e", "a", "d", "y"),
	"06/delete-non-empty":        confirmScenario(sample.Workspace, 120, 36, "space", "c", "d"),
	"06/delete-non-empty-80x24":  confirmScenario(sample.Workspace, 80, 24, "space", "c", "d"),
	"07/switcher":                projectsScenario(120, 36),
	"07/switcher-80x24":          projectsScenario(80, 24),
	"07/filtering":               projectsScenario(120, 36, "w", "o", "r"),
	"07/filtering-80x24":         projectsScenario(80, 24, "w", "o", "r"),
	"07/no-matches":              projectsScenario(120, 36, "z", "z"),
	"07/no-matches-80x24":        projectsScenario(80, 24, "z", "z"),
	"07/actions":                 projectsScenario(120, 36, "esc", "j"),
	"07/actions-80x24":           projectsScenario(80, 24, "esc", "j"),
	"07/single-project":          confirmScenario(sample.Workspace, 120, 36, "space", "p", "p", "esc", "x"),
	"07/single-project-80x24":    confirmScenario(sample.Workspace, 80, 24, "space", "p", "p", "esc", "x"),
	"07/new":                     confirmScenario(sample.Projects, 120, 36, "space", "p", "n"),
	"07/new-80x24":               confirmScenario(sample.Projects, 80, 24, "space", "p", "n"),
	"07/rename":                  projectsScenario(120, 36, "esc", "j", "e"),
	"07/rename-80x24":            projectsScenario(80, 24, "esc", "j", "e"),
	"07/delete":                  projectsScenario(120, 36, "esc", "j", "x"),
	"07/delete-80x24":            projectsScenario(80, 24, "esc", "j", "x"),
	"08/list":                    whiteboardsScenario(sample.Whiteboards, 120, 36),
	"08/list-80x24":              whiteboardsScenario(sample.Whiteboards, 80, 24),
	"08/empty":                   whiteboardsScenario(sample.WhiteboardsEmpty, 120, 36),
	"08/empty-80x24":             whiteboardsScenario(sample.WhiteboardsEmpty, 80, 24),
	"08/delete":                  whiteboardsScenario(sample.Whiteboards, 120, 36, "x"),
	"08/delete-80x24":            whiteboardsScenario(sample.Whiteboards, 80, 24, "x"),
	"08/rename":                  whiteboardsScenario(sample.Whiteboards, 120, 36, "r"),
	"08/rename-80x24":            whiteboardsScenario(sample.Whiteboards, 80, 24, "r"),
	"08/open-in-editor":          whiteboardsScenario(sample.Whiteboards, 120, 36, "enter"),
	"08/open-in-editor-80x24":    whiteboardsScenario(sample.Whiteboards, 80, 24, "enter"),
	"10/list":                    archiveScenario(sample.Archive, 120, 36, "j"),
	"10/list-80x24":              archiveScenario(sample.Archive, 80, 24, "j"),
	"10/searching":               archiveScenario(sample.Archive, 120, 36, "/", "dialog"),
	"10/searching-80x24":         archiveScenario(sample.Archive, 80, 24, "/", "dialog"),
	"10/no-match":                archiveScenario(sample.Archive, 120, 36, "/", "invoice"),
	"10/empty":                   archiveScenario(sample.Workspace, 120, 36),
	"10/empty-80x24":             archiveScenario(sample.Workspace, 80, 24),
	"10/detail":                  archiveScenario(sample.Archive, 120, 36, "j", "enter"),
	"10/detail-80x24":            archiveScenario(sample.Archive, 80, 24, "j", "enter"),
	"10/restored":                archiveScenario(sample.ArchiveRestore, 120, 36, "j", "r"),
	// After restoring “Copy task ID keybind”, open the task whose column is gone.
	"10/restore-fallback":  archiveScenario(sample.ArchiveRestore, 120, 36, "j", "r", "j", "j", "j", "j", "enter"),
	"10/archive-old":       confirmScenario(sample.ArchiveOld, 120, 36, "space", "a", "o"),
	"10/archive-old-80x24": confirmScenario(sample.ArchiveOld, 80, 24, "space", "a", "o"),
	// The cursor sits on done, which the bulk archive empties.
	"10/archived-old":           {Workspace: sample.ArchiveOld, W: 120, H: 36, Palette: dusk.Dark, Keys: []string{"l", "l", "l", "space", "a", "o", "y"}},
	"10/archive-old-none":       confirmScenario(sample.Workspace, 120, 36, "space", "a", "o"),
	"11/daily":                  dailyScenario(sample.DailyBoard, 120, 36),
	"11/daily-80x24":            dailyScenario(sample.DailyBoard, 80, 24),
	"11/empty":                  {Workspace: sample.DailyEmpty, W: 120, H: 36, Palette: dusk.Dark, Keys: []string{"space", "d", "d"}},
	"11/empty-80x24":            {Workspace: sample.DailyEmpty, W: 80, H: 24, Palette: dusk.Dark, Keys: []string{"space", "d", "d"}},
	"11/promote-project":        dailyScenario(sample.DailyPromote, 120, 36, "space", "d", "p"),
	"11/promote-project-80x24":  dailyScenario(sample.DailyPromote, 80, 24, "space", "d", "p"),
	"11/promote-column":         dailyScenario(sample.DailyPromote, 120, 36, "space", "d", "p", "enter", "j"),
	"11/promote-column-80x24":   dailyScenario(sample.DailyPromote, 80, 24, "space", "d", "p", "enter", "j"),
	"11/no-columns-fallback":    noColumnsScenario(),
	"11/promoted":               promotedScenario(),
	"15/saved":                  savedScenario(nil),
	"15/save-error":             savedScenario(failingStore{}),
	"01/sync-error":             withAfter(savedScenario(failingStore{}), "esc"),
	"15/moved":                  {Workspace: sample.Workspace, W: 120, H: 36, Palette: dusk.Dark, Keys: []string{"l", "j", "space", "t", "l"}},
	"15/archived-undo":          confirmScenario(sample.Workspace, 120, 36, "space", "t", "a", "y"),
	"15/archived-undo-80x24":    confirmScenario(sample.Workspace, 80, 24, "space", "t", "a", "y"),
	"15/copied-id":              {Workspace: sample.Workspace, W: 120, H: 36, Palette: dusk.Dark, Keys: []string{"l", "j", "space", "t", "y"}, WriteClipboard: func(string) error { return nil }},
	"15/external-reload":        {Workspace: sample.Workspace, W: 120, H: 36, Palette: dusk.Dark, Keys: []string{"l", "j"}, Reload: true},
	"01/sync-reloaded":          {Workspace: sample.Workspace, W: 120, H: 36, Palette: dusk.Dark, Keys: []string{"l", "j"}, Reload: true, After: []string{"esc"}},
	"15/store-unreadable":       {W: 120, H: 36, Palette: dusk.Dark, Unreadable: errors.New("file is not a database")},
	"15/store-unreadable-80x24": {W: 80, H: 24, Palette: dusk.Dark, Unreadable: errors.New("file is not a database")},
	"14/root":                   whichKeyScenario(sample.Workspace, 120, 36, true, "l", "j", "space"),
	"14/root-80x24":             whichKeyScenario(sample.Workspace, 80, 24, true, "l", "j", "space"),
	"14/pending":                whichKeyScenario(sample.Workspace, 120, 36, false, "l", "j", "space", "t"),
	"14/group-task":             whichKeyScenario(sample.Workspace, 120, 36, true, "l", "j", "space", "t"),
	"14/group-task-80x24":       whichKeyScenario(sample.Workspace, 80, 24, true, "l", "j", "space", "t"),
	"14/unbound":                whichKeyScenario(sample.Workspace, 120, 36, false, "l", "j", "space", "t", "q"),
	"14/placement-flip":         whichKeyScenario(sample.PlacementFlip, 120, 36, true, "l", "l", "l", "j", "j", "j", "j", "j", "j", "j", "j", "j", "j", "space"),
	"12/empty":                  paletteScenario(sample.Workspace, 120, 36, true),
	"12/empty-80x24":            paletteScenario(sample.Workspace, 80, 24, true),
	"12/typing":                 paletteScenario(sample.Workspace, 120, 36, false, "a", "r", "c", "h"),
	"12/no-match":               paletteScenario(sample.Workspace, 120, 36, false, "u", "n", "d", "o"),
	"12/second-step":            paletteScenario(sample.FourProjects, 120, 36, false, "s", "w", "i", "t", "c", "h", "enter"),
	"12/second-step-80x24":      paletteScenario(sample.FourProjects, 80, 24, false, "s", "w", "i", "t", "c", "h", "enter"),
	"17/dusk":                   {Workspace: sample.Workspace, W: 120, H: 36, Palette: dusk.Dark, Keys: []string{"l", "j"}},
	"17/dusk-light":             {Workspace: sample.Workspace, W: 120, H: 36, Palette: dusk.Light, Keys: []string{"l", "j"}},
	// theme = "terminal" on a terminal that answers with a light background.
	"17/terminal": {Workspace: sample.Workspace, W: 120, H: 36, Keys: []string{"l", "j"},
		Palette: themeconf.Palette(themeconf.Terminal, themeconf.Env{Background: func() (bool, bool) { return false, true }})},
	// A 16-colour terminal gets ansi16 whatever the theme.
	"17/dusk-ansi16": {Workspace: sample.Workspace, W: 120, H: 36, Keys: []string{"l", "j"},
		Palette: themeconf.Palette(themeconf.Dusk, themeconf.Env{Profile: termenv.ANSI})},
	"17/palette-theme":          paletteScenario(sample.Workspace, 120, 36, false, "t", "h", "e", "m", "e", "enter"),
	"17/palette-theme-80x24":    paletteScenario(sample.Workspace, 80, 24, false, "t", "h", "e", "m", "e", "enter"),
	"12/from-dialog":            {Workspace: sample.Workspace, W: 120, H: 36, Palette: dusk.Dark, Keys: []string{"l", "j", "enter", ":"}},
	"13/doing":                  focusScenario(sample.Focus, 120, 36),
	"13/doing-80x24":            focusScenario(sample.Focus, 80, 24),
	"13/advanced":               focusScenario(sample.Focus, 120, 36, "space", "t", "l"),
	"13/no-doing":               focusScenario(sample.FocusNoDoing, 120, 36),
	"09/delete-project":         deleteProjectScenario(120, 36),
	"09/delete-project-80x24":   deleteProjectScenario(80, 24),
	"09/delete-project-typing":  deleteProjectScenario(120, 36, "w", "o", "r"),
	"09/delete-project-matched": deleteProjectScenario(120, 36, "w", "o", "r", "k"),
})

func withLightScenarios(scenarios map[string]Scenario) map[string]Scenario {
	// Page 16 repeats source scenarios in light mode. The wide palette
	// shows the filtered state; its narrow counterpart shows the empty state.
	for light, source := range map[string]string{
		"16/board":             "01/default",
		"16/board-80x24":       "01/default-80x24",
		"16/task-detail":       "03/view",
		"16/task-detail-80x24": "03/view-80x24",
		"16/projects":          "07/switcher",
		"16/projects-80x24":    "07/switcher-80x24",
		"16/palette":           "12/typing",
		"16/palette-80x24":     "12/empty-80x24",
		"16/focus":             "13/doing",
		"16/focus-80x24":       "13/doing-80x24",
	} {
		scenario, ok := scenarios[source]
		if !ok {
			panic("missing source scenario for " + light + ": " + source)
		}
		scenario.Palette = dusk.Light
		scenarios[light] = scenario
	}
	return scenarios
}

// whichKeyScenario presses keys; wait lets the which-key delay pass after the
// last one, so the popover shows.
func whichKeyScenario(ws func(time.Time) *domain.Workspace, w, h int, wait bool, keys ...string) Scenario {
	return Scenario{Workspace: ws, W: w, H: h, Palette: dusk.Dark, Keys: keys, WaitWhichKey: wait}
}

// paletteScenario opens the palette on “Undo for move and delete” in todo,
// then types keys. recent seeds the actions 12/empty lists as recent.
func paletteScenario(ws func(time.Time) *domain.Workspace, w, h int, recent bool, keys ...string) Scenario {
	s := Scenario{Workspace: ws, W: w, H: h, Palette: dusk.Dark, Keys: append([]string{"l", "j", ":"}, keys...)}
	if recent {
		s.PaletteRecent = []string{"space t l", "space t y", "space t w"}
	}
	return s
}

// focusScenario opens focus mode on the doing column, then presses keys.
func focusScenario(ws func(time.Time) *domain.Workspace, w, h int, keys ...string) Scenario {
	return Scenario{Workspace: ws, W: w, H: h, Palette: dusk.Dark, Keys: append([]string{"l", "l", "z"}, keys...)}
}

// confirmScenario selects “Undo for move and delete” in todo (the board every
// 09 frame shows behind its dialog), then presses keys.
func confirmScenario(ws func(time.Time) *domain.Workspace, w, h int, keys ...string) Scenario {
	return Scenario{Workspace: ws, W: w, H: h, Palette: dusk.Dark, Keys: append([]string{"l", "j"}, keys...)}
}

// clearDailyScenario opens the daily board and asks to clear it.
func clearDailyScenario(keys ...string) Scenario {
	return Scenario{Workspace: sample.DailyBoard, W: 120, H: 36, Palette: dusk.Dark, Keys: append([]string{"space", "d", "d", "l", "j", "space", "d", "X"}, keys...)}
}

// dailyScenario opens the daily board from home, selects “Fix flaky CI test”
// in active, then presses keys.
func dailyScenario(ws func(time.Time) *domain.Workspace, w, h int, keys ...string) Scenario {
	return Scenario{Workspace: ws, W: w, H: h, Palette: dusk.Dark, Keys: append([]string{"space", "d", "d", "l", "j"}, keys...)}
}

// noColumnsScenario opens promote once no project has a column left.
func noColumnsScenario() Scenario {
	// Start on the daily board: the home board can't run without columns.
	ws := func(now time.Time) *domain.Workspace {
		w := sample.DailyTwoProjects(now)
		w.ActiveProjectID = w.DailyProject().ID
		return w
	}
	return Scenario{Workspace: ws, W: 120, H: 36, Palette: dusk.Dark, Keys: []string{"l", "j", "space", "d", "p"}, Loaded: sample.DropColumns}
}

// promotedScenario promotes “Fix flaky CI test” to home / todo and lets the
// save finish.
func promotedScenario() Scenario {
	s := dailyScenario(sample.DailyPromote, 120, 36, "space", "d", "p", "enter", "j", "enter")
	s.Settle = true
	return s
}

// savedScenario edits “Undo for move and delete”, saves it with :wq and lets
// the save land in st (nil: a store that discards writes).
func savedScenario(st store.WorkspaceStore) Scenario {
	return Scenario{Workspace: sample.Workspace, W: 120, H: 36, Palette: dusk.Dark,
		Keys: []string{"l", "j", "space", "t", "e", "esc", ":", "w", "q", "enter"}, Settle: true, Store: st}
}

func withAfter(s Scenario, keys ...string) Scenario {
	s.After = keys
	return s
}

// deleteProjectScenario deletes “work” from the projects list, over the home board.
func deleteProjectScenario(w, h int, keys ...string) Scenario {
	return confirmScenario(sample.WithWorkProject, w, h, append([]string{"space", "p", "p", "esc", "j", "x"}, keys...)...)
}

// whiteboardsScenario opens the selected task's whiteboards with SPC t w.
func whiteboardsScenario(ws func(time.Time) *domain.Workspace, w, h int, keys ...string) Scenario {
	return confirmScenario(ws, w, h, append([]string{"space", "t", "w"}, keys...)...)
}

// archiveScenario opens the archive with SPC a v over the home board.
func archiveScenario(ws func(time.Time) *domain.Workspace, w, h int, keys ...string) Scenario {
	return confirmScenario(ws, w, h, append([]string{"space", "a", "v"}, keys...)...)
}

// projectsScenario opens the project switcher over the home board.
func projectsScenario(w, h int, keys ...string) Scenario {
	return confirmScenario(sample.Projects, w, h, append([]string{"space", "p", "p"}, keys...)...)
}

func searchScenario(w, h int, keys ...string) Scenario {
	return Scenario{Workspace: sample.SearchFilter, W: w, H: h, Palette: dusk.Dark, Keys: append([]string{"l", "j", "/", "C", "L", "I"}, keys...)}
}

const (
	newTaskTitle = "Export board as CSV"
	newTaskLine1 = "Same columns as the markdown export, one row per task."
	newTaskLine2 = "Quote titles that contain commas."
)

func newTaskScenario(w, h int, keys ...string) Scenario {
	return Scenario{Workspace: sample.Workspace, W: w, H: h, Palette: dusk.Dark, Keys: append([]string{"l", "space", "t", "n"}, keys...)}
}

func narrowNewTaskScenario(keys ...string) Scenario {
	s := newTaskScenario(80, 24, keys...)
	s.Keys = append([]string{"l", "j"}, s.Keys[1:]...)
	return s
}

func newTaskDescriptionScenario() Scenario {
	s := newTaskScenario(120, 36, newTaskTitle, "tab", newTaskLine1, "enter", newTaskLine2)
	s.Blur = true
	return s
}

func newTaskCreatedScenario(w, h int) Scenario {
	s := newTaskScenario(w, h, newTaskTitle, "tab", newTaskLine1, "enter", newTaskLine2, "esc", ":", "w", "q", "enter")
	s.NewTask = func(title, description string) (*domain.Task, error) {
		task, err := domain.NewTask(title, description)
		if err == nil {
			task.ID = "7c2e0b1401234567"
			task.CreatedAt, task.UpdatedAt = sample.Now, sample.Now
		}
		return task, err
	}
	return s
}

// Run renders the scenario through the app's normal update loop and returns
// the final screen as ANSI.
func (s Scenario) Run() string {
	palette := s.Palette
	if s.Unreadable != nil {
		m := ui.NewUnreadable("", unreadablePath, s.Unreadable, nil, ui.Options{Palette: &palette})
		m, _ = m.Update(tea.WindowSizeMsg{Width: s.W, Height: s.H})
		return m.View()
	}
	var st store.WorkspaceStore
	if s.Settle {
		st = noopStore{}
	}
	if s.Store != nil {
		st = s.Store
	}
	if s.Reload {
		st = reloadStore{}
	}
	workspace := s.Workspace(sample.Now)
	m := ui.NewWithOptions(workspace, st, "", ui.Options{
		Palette:               &palette,
		Now:                   func() time.Time { return sample.Now },
		WriteClipboard:        s.WriteClipboard,
		ExternalCheckInterval: time.Nanosecond,
		NewTask:               s.NewTask,
		// Frames never launch a real editor.
		LaunchWhiteboard: func(string) error { return nil },
		WhichKeyDelay:    time.Millisecond,
		PaletteRecent:    s.PaletteRecent,
	})
	if s.Loaded != nil {
		s.Loaded(workspace)
	}
	m, _ = m.Update(tea.WindowSizeMsg{Width: s.W, Height: s.H})
	var cmd tea.Cmd
	for _, k := range s.Keys {
		m, cmd = m.Update(keyMsg(k))
	}
	if (s.Settle || s.WaitWhichKey) && cmd != nil {
		for _, msg := range runCmd(cmd) {
			m, _ = m.Update(msg)
		}
	}
	if s.Reload {
		// Init's poll notices the write, checks, then reloads: three hops.
		cmd := m.Init()
		for hop := 0; hop < 3 && cmd != nil; hop++ {
			var next []tea.Cmd
			for _, msg := range runCmd(cmd) {
				var c tea.Cmd
				m, c = m.Update(msg)
				next = append(next, c)
			}
			cmd = tea.Batch(next...)
		}
	}
	for _, k := range s.After {
		m, _ = m.Update(keyMsg(k))
	}
	if s.Blur {
		m, _ = m.Update(tea.BlurMsg{})
	}
	return m.View()
}

func keyMsg(k string) tea.KeyMsg {
	named := map[string]tea.KeyType{
		"enter": tea.KeyEnter, "esc": tea.KeyEsc, "tab": tea.KeyTab, "backspace": tea.KeyBackspace,
		"up": tea.KeyUp, "down": tea.KeyDown, "left": tea.KeyLeft, "right": tea.KeyRight,
		"ctrl+c": tea.KeyCtrlC, "ctrl+e": tea.KeyCtrlE,
	}
	if t, ok := named[k]; ok {
		return tea.KeyMsg{Type: t}
	}
	if k == "space" {
		return tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
}

// unreadablePath is the board path page 15 shows.
const unreadablePath = "~/Library/Application Support/kanban-tui/board.db"

// runCmd runs cmd and any batch inside it, keeping the messages that arrive
// promptly; timers still waiting are dropped.
func runCmd(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	ch := make(chan tea.Msg, 1)
	go func() { ch <- cmd() }()
	var msg tea.Msg
	select {
	case msg = <-ch:
	case <-time.After(100 * time.Millisecond):
		return nil
	}
	if batch, ok := msg.(tea.BatchMsg); ok {
		var out []tea.Msg
		for _, c := range batch {
			out = append(out, runCmd(c)...)
		}
		return out
	}
	if msg == nil {
		return nil
	}
	return []tea.Msg{msg}
}

type failingStore struct{}

func (failingStore) Load() (*domain.Workspace, error) { return nil, errors.New("disk full") }
func (failingStore) Save(*domain.Workspace) error     { return errors.New("disk full") }

// reloadStore reports one external write, of the same board.
type reloadStore struct{}

func (reloadStore) Load() (*domain.Workspace, error) { return sample.Workspace(sample.Now), nil }
func (reloadStore) Save(*domain.Workspace) error     { return nil }
func (reloadStore) Changed() (bool, error)           { return true, nil }

type noopStore struct{}

func (noopStore) Load() (*domain.Workspace, error) { return nil, nil }
func (noopStore) Save(*domain.Workspace) error     { return nil }
