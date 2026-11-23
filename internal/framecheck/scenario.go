package framecheck

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
	"github.com/nikhilsharma/kanban-tui/internal/sample"
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
}

// Scenarios maps a frame name to how to reach it. Frames without one are
// reported as missing a scenario.
var Scenarios = map[string]Scenario{
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
	"09/delete-project":          deleteProjectScenario(120, 36),
	"09/delete-project-80x24":    deleteProjectScenario(80, 24),
	"09/delete-project-typing":   deleteProjectScenario(120, 36, "w", "o", "r"),
	"09/delete-project-matched":  deleteProjectScenario(120, 36, "w", "o", "r", "k"),
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

// deleteProjectScenario deletes “work” from the projects list, over the home board.
func deleteProjectScenario(w, h int, keys ...string) Scenario {
	return confirmScenario(sample.WithWorkProject, w, h, append([]string{"space", "p", "p", "esc", "j", "x"}, keys...)...)
}

// whiteboardsScenario opens the selected task's whiteboards with SPC t w.
func whiteboardsScenario(ws func(time.Time) *domain.Workspace, w, h int, keys ...string) Scenario {
	return confirmScenario(ws, w, h, append([]string{"space", "t", "w"}, keys...)...)
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
	m := ui.NewWithOptions(s.Workspace(sample.Now), nil, "", ui.Options{
		Palette:        &palette,
		Now:            func() time.Time { return sample.Now },
		WriteClipboard: s.WriteClipboard,
		NewTask:        s.NewTask,
		// Frames never launch a real editor.
		LaunchWhiteboard: func(string) error { return nil },
	})
	m, _ = m.Update(tea.WindowSizeMsg{Width: s.W, Height: s.H})
	for _, k := range s.Keys {
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
