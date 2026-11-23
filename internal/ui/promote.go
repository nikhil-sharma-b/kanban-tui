package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
	"github.com/nikhilsharma/kanban-tui/internal/termgrid"
	"github.com/nikhilsharma/kanban-tui/internal/ui/dusk"
)

// Promote (SPC d p on the daily board) moves the selected daily task to a
// project column in two steps inside one fixed-size Dialog (page 11 Daily):
// pick a project from a live filter, then pick one of its columns.

type promoteStepKind int

const (
	promoteStepProject promoteStepKind = iota
	promoteStepColumn
)

// promoteListRows is the fixed height of the dialog's list area; the dialog
// never changes size between the two steps.
const promoteListRows = 4

// promoteDialogHeight is borders, the title row, a blank, the list area, a
// blank and the footer.
const promoteDialogHeight = 6 + promoteListRows

type promotion struct {
	promoteStep          promoteStepKind
	promoteProjectCursor int
	promoteColumnCursor  int
	promoteProjectID     string // the project picked in step 1
	promoteMore          bool
	promoteFilter        textinput.Model
}

type promoteGroup struct {
	project *domain.Project
}

func (m *model) openPromote() (tea.Model, tea.Cmd) {
	m.mode = modeDailyPromote
	m.promoteStep = promoteStepProject
	m.promoteMore = false
	m.promoteFilter = textinput.New()
	m.promoteFilter.Prompt = ""
	m.promoteFilter.Focus()
	m.promoteProjectCursor = 0
	for i, group := range m.promoteProjects() {
		if group.project.ID == m.prevProjectID {
			m.promoteProjectCursor = i
		}
	}
	m.lastErr = nil
	return m, textinput.Blink
}

// promoteProjects are the regular projects matching the filter: the same
// targets the daily board always offered, grouped by project.
func (m *model) promoteProjects() []promoteGroup {
	if m.workspace == nil {
		return nil
	}
	query := strings.TrimSpace(m.promoteFilter.Value())
	var groups []promoteGroup
	for _, project := range m.workspace.RegularProjects() {
		if project.Board == nil {
			continue
		}
		if query == "" || fuzzyMatch(query, project.Name) {
			groups = append(groups, promoteGroup{project: project})
		}
	}
	return groups
}

// promoteHasColumns reports whether any project can take the task.
func (m *model) promoteHasColumns() bool {
	for _, project := range m.workspace.RegularProjects() {
		if project.Board != nil && len(project.Board.Statuses()) > 0 {
			return true
		}
	}
	return false
}

func (m *model) promoteProject() *domain.Project {
	return m.workspace.ProjectByID(m.promoteProjectID)
}

func (m *model) promoteColumns() []domain.Status {
	if project := m.promoteProject(); project != nil && project.Board != nil {
		return project.Board.Statuses()
	}
	return nil
}

func (m *model) updateDailyPromote(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.promotionPending {
		return m, nil
	}
	key := msg.String()
	if key == "?" {
		m.promoteMore = !m.promoteMore
		return m, nil
	}
	if m.promoteStep == promoteStepColumn {
		return m.updatePromoteColumn(key)
	}

	groups := m.promoteProjects()
	m.promoteProjectCursor = min(m.promoteProjectCursor, max(0, len(groups)-1))
	switch key {
	case "esc":
		m.mode = modeBoard
		m.lastErr = nil
		return m, nil
	case "up":
		m.promoteProjectCursor = max(0, m.promoteProjectCursor-1)
		return m, nil
	case "down":
		m.promoteProjectCursor = min(max(0, len(groups)-1), m.promoteProjectCursor+1)
		return m, nil
	case "enter":
		if len(groups) == 0 {
			return m, nil
		}
		project := groups[m.promoteProjectCursor].project
		if len(project.Board.Statuses()) == 0 {
			return m, nil
		}
		m.promoteProjectID = project.ID
		m.promoteStep = promoteStepColumn
		m.promoteColumnCursor = 0
		m.promoteMore = false
		m.lastErr = nil
		return m, nil
	}
	if !m.promoteHasColumns() {
		return m, nil
	}
	before := m.promoteFilter.Value()
	var cmd tea.Cmd
	m.promoteFilter, cmd = m.promoteFilter.Update(msg)
	if m.promoteFilter.Value() != before {
		m.promoteProjectCursor = 0
	}
	return m, cmd
}

func (m *model) updatePromoteColumn(key string) (tea.Model, tea.Cmd) {
	columns := m.promoteColumns()
	switch key {
	case "esc":
		m.promoteStep = promoteStepProject
		m.promoteMore = false
		m.lastErr = nil
	case "k", "up":
		m.promoteColumnCursor = max(0, m.promoteColumnCursor-1)
	case "j", "down":
		m.promoteColumnCursor = min(max(0, len(columns)-1), m.promoteColumnCursor+1)
	case "enter":
		if len(columns) == 0 {
			return m, nil
		}
		return m.promote(m.promoteProject(), columns[min(m.promoteColumnCursor, len(columns)-1)])
	}
	return m, nil
}

// promote saves the move; the daily board only changes once the save lands.
func (m *model) promote(target *domain.Project, status domain.Status) (tea.Model, tea.Cmd) {
	task := m.selectedTask()
	if task == nil {
		m.mode = modeBoard
		m.lastErr = fmt.Errorf("daily task not found")
		return m, nil
	}
	selectID := m.neighbourTaskID(task)
	workspace := m.workspace.Clone()
	promoted, err := workspace.PromoteDailyTask(task.ID, target.ID, status)
	if err != nil {
		m.lastErr = err
		return m, nil
	}
	m.promotionPending = true
	m.lastErr = nil
	m.lastStatus = ""
	dailyID, title, name := m.project.ID, task.Title, target.Name
	return m, m.queueSave(func() tea.Msg {
		msg := promoteDailyTask(m.store, workspace, promoted, m.dataPath, dailyID, name, status)
		if done, ok := msg.(dailyPromotionFinishedMsg); ok {
			done.title, done.selectID = title, selectID
			return done
		}
		return msg
	})
}

// neighbourTaskID is the card selected once task leaves its column: the next
// one, or the one before when it was last.
func (m *model) neighbourTaskID(task *domain.Task) string {
	ids := m.visible[task.Status]
	for i, id := range ids {
		if id != task.ID {
			continue
		}
		if i+1 < len(ids) {
			return ids[i+1]
		}
		if i > 0 {
			return ids[i-1]
		}
	}
	return ""
}

func (m *model) promoteHints() []KeyHint {
	switch {
	case !m.promoteHasColumns():
		if m.promoteMore {
			return []KeyHint{{"esc", "cancel"}, {"?", "less"}}
		}
		return []KeyHint{{"esc", "cancel"}, {"?", "more"}}
	case m.promoteStep == promoteStepColumn:
		if m.promoteMore {
			return []KeyHint{{"j k", "move"}, {"↵", "promote"}, {"esc", "back to projects"}, {"?", "less"}}
		}
		return []KeyHint{{"j k", "move"}, {"↵", "promote"}, {"esc", "back"}, {"?", "more"}}
	case m.promoteMore:
		return []KeyHint{{"↑↓", "move"}, {"↵", "next"}, {"esc", "cancel"}, {"backspace", "edit filter"}, {"?", "less"}}
	}
	return []KeyHint{{"↑↓", "move"}, {"↵", "next"}, {"esc", "cancel"}, {"?", "more"}}
}

// promoteDialog draws 11/promote-project, 11/promote-column and
// 11/no-columns-fallback.
func (m *model) promoteDialog() *termgrid.Grid {
	p := m.palette
	w, h := m.dialogW(), promoteDialogHeight
	if w < 24 || h > m.height {
		return nil
	}
	g := newDialog(p, w, h)
	width := w - 4
	task := m.selectedTask()

	title := ""
	if task != nil {
		title = "promote “" + truncateCell(singleLine(task.Title), width/2) + "”"
	}
	titleX := w - 2 - ansi.StringWidth(title)
	g.Text(titleX, 1, title, tokenFG(p, dusk.TextDim, false))
	left := titleX - 2

	listY := 3
	statusY := listY + promoteListRows - 1
	if m.promoteStep == promoteStepColumn {
		project := m.promoteProject()
		if project != nil {
			g.Text(2, 1, truncateCell(project.Name+" ›", left-3), tokenFG(p, dusk.Text, false))
		}
		columns := m.promoteColumns()
		if project == nil {
			columns = nil
		}
		first := max(0, min(m.promoteColumnCursor-promoteListRows+1, len(columns)-promoteListRows))
		for i := first; i < len(columns) && i < first+promoteListRows; i++ {
			state := ListRowDefault
			if i == m.promoteColumnCursor {
				state = ListRowSelected
			}
			drawPromoteRow(g, p, w, listY+i-first, string(columns[i]), plural(len(project.Board.Order[columns[i]]), "task"), state)
		}
	} else {
		hasColumns := m.promoteHasColumns()
		if hasColumns {
			lines, col := cutInput(m.promoteFilter.Value(), m.promoteFilter.Position(), left-3)
			drawFocusedInput(g, p, 0, 1, left-1, 1, lines, IdleInputField, "Filter projects…", &inputPoint{col: col})
		}
		groups := m.promoteProjects()
		rows := promoteListRows
		if !hasColumns {
			rows-- // the last row carries the status message
		}
		if len(groups) == 0 {
			g.Text(2, listY, "No matching projects", tokenFG(p, dusk.TextMuted, false))
		}
		first := max(0, min(m.promoteProjectCursor-rows+1, len(groups)-rows))
		for i := first; i < len(groups) && i < first+rows; i++ {
			project := groups[i].project
			n := len(project.Board.Statuses())
			hint := plural(n, "column")
			if n == 0 {
				hint = "no columns"
			}
			if project.ID == m.prevProjectID && hasColumns {
				hint = "last board · " + hint
			}
			state := ListRowDefault
			if n == 0 {
				state = ListRowDisabled
			} else if i == m.promoteProjectCursor {
				state = ListRowSelected
			}
			drawPromoteRow(g, p, w, listY+i-first, project.Name, hint, state)
		}
		if !hasColumns {
			drawStatusMessage(g, p, 2, statusY, width, StatusInfo, "no project has a column · add one with SPC c n")
		}
	}
	if m.lastErr != nil {
		drawStatusMessage(g, p, 2, h-3, width, StatusDanger, m.lastErr.Error())
	}
	if m.promotionPending {
		drawStatusMessage(g, p, 2, h-2, width, StatusInfo, "promoting…")
	} else {
		drawDialogHints(g, p, 2, h-2, width, m.promoteHints())
	}
	return g
}

// drawPromoteRow is a List row whose hint reads text-muted, like the
// projects switcher (11/promote-project); a project without columns is
// disabled (11/no-columns-fallback).
func drawPromoteRow(g *termgrid.Grid, p dusk.Palette, w, y int, label, hint string, state ListRowState) {
	drawListRow(g, p, 0, y, w-2, label, hint, state)
	g.Text(w-2-ansi.StringWidth(hint), y, hint, tokenFG(p, dusk.TextMuted, false))
}
