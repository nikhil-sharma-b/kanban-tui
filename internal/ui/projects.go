package ui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
	"github.com/nikhilsharma/kanban-tui/internal/termgrid"
	"github.com/nikhilsharma/kanban-tui/internal/ui/dusk"
)

// projectEditDialogHeight is the name dialog without its error row; it stays
// anchored at this height when the error row appears (07/new).
const projectEditDialogHeight = 5

var errLastProject = errors.New("The last project can’t be deleted.")

// deletableProject refuses the daily board and the last regular project, so
// no entry point can open a delete confirm for them.
func (m *model) deletableProject(project *domain.Project) error {
	if project.Daily {
		return errors.New("The daily board can’t be deleted.")
	}
	if len(m.workspace.RegularProjects()) <= 1 {
		return errLastProject
	}
	return nil
}

// openProjects opens the switcher (SPC p p): the live name filter is focused
// and the current project is selected.
func (m *model) openProjects() (tea.Model, tea.Cmd) {
	m.mode = modeProjects
	m.projectCursor = m.activeProjectIndex()
	m.projectInput.Blur()
	m.projectFiltering = true
	m.projectFilterInput.SetValue("")
	m.projectFilterInput.Focus()
	m.projectErr, m.lastErr = nil, nil
	return m, textinput.Blink
}

func (m *model) filteredProjects() []*domain.Project {
	projects := m.workspace.RegularProjects()
	query := strings.TrimSpace(m.projectFilterInput.Value())
	if query == "" {
		return projects
	}
	matches := make([]*domain.Project, 0, len(projects))
	for _, project := range projects {
		if fuzzyMatch(query, project.Name) {
			matches = append(matches, project)
		}
	}
	return matches
}

func (m *model) updateProjects(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	projects := m.filteredProjects()
	m.projectCursor = min(m.projectCursor, max(0, len(projects)-1))
	key := msg.String()
	switch key {
	case "up", "down":
		m.moveProjectCursor(key == "down", len(projects))
		return m, nil
	case "enter":
		if len(projects) > 0 {
			return m.switchProject(projects[m.projectCursor].ID)
		}
		return m, nil
	}

	if m.projectFiltering {
		if key == "esc" {
			// Clear the query and manage, keeping the selected project.
			selected := ""
			if len(projects) > 0 {
				selected = projects[m.projectCursor].ID
			}
			m.projectFiltering = false
			m.projectFilterInput.SetValue("")
			m.projectFilterInput.Blur()
			m.projectCursor = m.regularProjectIndex(selected)
			if selected == "" {
				m.projectCursor = m.activeProjectIndex()
			}
			return m, nil
		}
		before := m.projectFilterInput.Value()
		var cmd tea.Cmd
		m.projectFilterInput, cmd = m.projectFilterInput.Update(msg)
		if m.projectFilterInput.Value() != before {
			m.projectCursor = 0
		}
		return m, cmd
	}

	m.projectErr = nil
	switch key {
	case "esc":
		m.mode = modeBoard
		return m, nil
	case "/":
		m.projectFiltering = true
		m.projectFilterInput.Focus()
		return m, textinput.Blink
	case "k":
		m.moveProjectCursor(false, len(projects))
	case "j":
		m.moveProjectCursor(true, len(projects))
	case "n":
		return m.beginProjectEdit("", "", modeProjects)
	case "e":
		if len(projects) > 0 {
			project := projects[m.projectCursor]
			return m.beginProjectEdit(project.ID, project.Name, modeProjects)
		}
	case "x":
		if len(projects) > 0 {
			if err := m.deletableProject(projects[m.projectCursor]); err != nil {
				m.projectErr = err
				return m, nil
			}
			return m.confirmDeleteProject(projects[m.projectCursor], modeProjects)
		}
	}
	return m, nil
}

func (m *model) moveProjectCursor(down bool, n int) {
	if down && m.projectCursor < n-1 {
		m.projectCursor++
	} else if !down && m.projectCursor > 0 {
		m.projectCursor--
	}
}

// beginProjectEdit opens the project name dialog: a new project when id is
// empty, otherwise a rename. back is where cancelling or renaming returns.
func (m *model) beginProjectEdit(id, name string, back mode) (tea.Model, tea.Cmd) {
	m.mode = modeProjectEdit
	m.projectEditPrev = back
	m.projectDraft = id
	m.projectInput.SetValue(name)
	m.projectInput.CursorEnd()
	m.projectInput.Focus()
	m.projectErr, m.lastErr = nil, nil
	return m, textinput.Blink
}

func (m *model) closeProjectEdit(next mode) {
	m.mode = next
	m.projectInput.Blur()
	m.projectDraft = ""
	m.projectErr = nil
}

func (m *model) updateProjectEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.closeProjectEdit(m.projectEditPrev)
		return m, nil
	case "enter":
		name := m.projectInput.Value()
		if m.projectDraft == "" {
			project, err := m.workspace.CreateProject(name)
			if err != nil {
				m.projectErr = err
				return m, nil
			}
			m.activateProject(project.ID)
			m.closeProjectEdit(modeBoard)
			text := m.quotedToast("Created project ", project.Name, KeyHint{})
			return m, tea.Batch(m.saveWorkspaceCmd(), m.showToast(StatusSuccess, text, KeyHint{}))
		}

		previousPaths := snapshotProjectWhiteboardPaths(m.workspace.ProjectByID(m.projectDraft), m.dataPath)
		project, err := m.workspace.RenameProject(m.projectDraft, name)
		if err != nil {
			m.projectErr = err
			return m, nil
		}
		if err := relocateProjectWhiteboardFiles(project, m.dataPath, previousPaths); err != nil {
			m.projectErr = fmt.Errorf("rename project whiteboards: %w", err)
			return m, nil
		}
		m.closeProjectEdit(m.projectEditPrev)
		m.projectCursor = m.regularProjectIndex(project.ID)
		m.lastStatus = fmt.Sprintf("renamed project %s", project.Name)
		return m, m.saveWorkspaceCmd()
	}

	before := m.projectInput.Value()
	var cmd tea.Cmd
	m.projectInput, cmd = m.projectInput.Update(msg)
	if m.projectInput.Value() != before {
		m.projectErr = nil
	}
	return m, cmd
}

// projectHints are the projects dialog footer rows, in the Fixed Order.
func (m *model) projectHints(shown int) [][]KeyHint {
	if m.projectFiltering {
		query := strings.TrimSpace(m.projectFilterInput.Value()) != ""
		if shown == 0 && query {
			return [][]KeyHint{{{"esc", "clear filter"}, {"backspace", "edit query"}}}
		}
		leave := KeyHint{"esc", "manage"}
		if query {
			leave = KeyHint{"esc", "clear filter"}
		}
		return [][]KeyHint{{{"↑↓", "move"}, {"↵", "open"}, leave}}
	}
	var nav, actions []KeyHint
	if shown > 1 {
		nav = append(nav, KeyHint{"j k", "move"})
	}
	nav = append(nav, KeyHint{"↵", "open"}, KeyHint{"/", "filter"}, KeyHint{"esc", "close"})
	actions = []KeyHint{{"n", "new"}, {"e", "rename"}}
	if len(m.workspace.RegularProjects()) > 1 {
		actions = append(actions, KeyHint{"x", "delete"})
	}
	return [][]KeyHint{nav, actions}
}

// projectsDialog is the 00 Dialog for SPC p p (page 07): the name filter,
// one List row per regular project with its task count, and the footer.
func (m *model) projectsDialog() *termgrid.Grid {
	p := m.palette
	w := m.dialogW()
	projects := m.filteredProjects()
	hints := m.projectHints(len(projects))
	listRows := max(1, len(projects))
	fixed := 4 + len(hints) // borders, filter row and the blanks around the list
	if m.projectErr != nil {
		fixed++
	}
	listRows = min(listRows, m.height-fixed-1)
	if w < 24 || listRows < 1 {
		return nil
	}
	h := fixed + listRows + 1
	g := newDialog(p, w, h)
	width := w - 4

	label := "projects"
	g.Text(w-2-ansi.StringWidth(label), 1, label, tokenFG(p, dusk.TextDim, false))
	inputWidth := width - ansi.StringWidth(label) - 1
	placeholder := "Filter projects…"
	if m.projectFiltering {
		lines, col := cutInput(m.projectFilterInput.Value(), m.projectFilterInput.Position(), inputWidth)
		drawFocusedInput(g, p, 0, 1, inputWidth+2, 1, lines, IdleInputField, placeholder, &inputPoint{col: col})
	} else {
		drawIdleInput(g, p, 0, 1, inputWidth+2, nil, IdleInputField, placeholder)
	}

	y := 3
	if len(projects) == 0 {
		g.Text(2, y, "No matching projects", tokenFG(p, dusk.TextMuted, false))
		y++
	}
	first := max(0, min(m.projectCursor-listRows+1, len(projects)-listRows))
	for i := first; i < len(projects) && i < first+listRows; i++ {
		project := projects[i]
		state := ListRowDefault
		if i == m.projectCursor {
			state = ListRowSelected
		}
		count := plural(project.Board.ActiveTaskCount(), "task")
		// The count and the current marker read text-muted (07/switcher).
		label := project.Name
		marker := ""
		if project.ID == m.workspace.ActiveProjectID {
			marker = "current    "
		}
		drawListRow(g, p, 0, y, w-2, label, marker+count, state)
		g.Text(w-2-ansi.StringWidth(marker+count), y, marker+count, tokenFG(p, dusk.TextMuted, false))
		y++
	}
	if m.projectErr != nil {
		drawStatusMessage(g, p, 2, y, width, StatusDanger, m.projectErr.Error())
	}
	for i, row := range hints {
		drawDialogHints(g, p, 2, h-1-len(hints)+i, width, row)
	}
	return g
}

// projectEditDialog is the name dialog for new and rename (07/new, 07/rename).
func (m *model) projectEditDialog() *termgrid.Grid {
	p := m.palette
	w := m.dialogW()
	h := projectEditDialogHeight
	if m.projectErr != nil {
		h++
	}
	if w < 24 || h > m.height {
		return nil
	}
	g := newDialog(p, w, h)
	width := w - 4
	label, verb, placeholder := "new project", "create", "Project name…"
	if m.projectDraft != "" {
		label, verb, placeholder = "rename project", "rename", ""
	}
	g.Text(w-2-ansi.StringWidth(label), 1, label, tokenFG(p, dusk.TextDim, false))
	inputWidth := width - ansi.StringWidth(label) - 1
	lines, col := cutInput(m.projectInput.Value(), m.projectInput.Position(), inputWidth)
	drawFocusedInput(g, p, 0, 1, inputWidth+2, 1, lines, IdleInputField, placeholder, &inputPoint{col: col})
	if m.projectErr != nil {
		drawStatusMessage(g, p, 2, 2, width, StatusDanger, m.projectErr.Error())
	}
	drawDialogHints(g, p, 2, h-2, width, []KeyHint{{"↵", verb}, {"esc", "cancel"}, {"backspace", "delete character"}})
	return g
}

func (m *model) placeProjectEditDialog(board *termgrid.Grid) string {
	ApplyScrim(board, m.palette)
	if d := m.projectEditDialog(); d != nil {
		board.Blit((board.W-d.W)/2, (board.H-projectEditDialogHeight)/2, d)
	}
	return renderGrid(board)
}
