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

// confirmation is the one dialog for every destructive or bulk action
// (page 09 Confirm). Tier 1 acts on y; tier 2 (name set) acts on ↵ once the
// typed text matches name exactly.
type confirmation struct {
	title, body string
	// Tier 1: footer esc keep · y verb, above a note status message.
	verb     string
	noteKind StatusKind
	note     string
	// Tier 2: type name to confirm; prompt is the info line, matched the danger line.
	name, prompt, matched string
	action                func() (tea.Model, tea.Cmd)
}

func (c confirmation) typed() bool { return c.name != "" }

func permanent(title, body string, action func() (tea.Model, tea.Cmd)) confirmation {
	return confirmation{title: title, body: body, verb: "delete", noteKind: StatusDanger, note: "can't be undone", action: action}
}

func archival(title, body, note string, action func() (tea.Model, tea.Cmd)) confirmation {
	return confirmation{title: title, body: body, verb: "archive", noteKind: StatusInfo, note: note, action: action}
}

func typedName(title, body, name, verb, matched string, action func() (tea.Model, tea.Cmd)) confirmation {
	return confirmation{title: title, body: body, name: name, prompt: fmt.Sprintf("type “%s” to %s", name, verb), matched: matched, action: action}
}

func (m *model) askConfirm(c confirmation, prev mode) (tea.Model, tea.Cmd) {
	m.confirm = c
	m.confirmPrev = prev
	m.lastStatus, m.lastErr = "", nil
	m.mode = modeConfirm
	m.confirmInput.SetValue("")
	if c.typed() {
		m.confirmInput.Focus()
		return m, textinput.Blink
	}
	m.confirmInput.Blur()
	return m, nil
}

func (m *model) confirmMatched() bool {
	return m.confirm.typed() && strings.TrimSpace(m.confirmInput.Value()) == m.confirm.name
}

func (m *model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "esc" {
		m.confirmInput.Blur()
		m.mode = m.confirmPrev
		return m, nil
	}
	if !m.confirm.typed() {
		if key == "y" {
			return m.confirm.action()
		}
		return m, nil
	}
	if key == "enter" {
		if m.confirmMatched() {
			m.confirmInput.Blur()
			return m.confirm.action()
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.confirmInput, cmd = m.confirmInput.Update(msg)
	return m, cmd
}

func plural(n int, one string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", one)
	}
	return fmt.Sprintf("%d %ss", n, one)
}

// confirmationGrid draws c in a w×h Dialog: title, wrapped body, then for
// tier 1 its note (when set) and esc keep · y verb. For tier 2 the caller
// draws the input, the status line and the footer.
func confirmationGrid(p dusk.Palette, w, h int, c confirmation) *termgrid.Grid {
	g := newDialog(p, w, h)
	width := w - 5
	if width < 1 || h < 4 {
		return g
	}
	g.Text(3, 1, truncateCell(c.title, width), tokenFG(p, dusk.Text, true))
	end := h - 3 // the blank row above the footer
	switch {
	case c.typed():
		end = h - 7
	case c.note != "":
		end = h - 5
	}
	for i, line := range wrapDialogText(c.body, width) {
		if 3+i >= end {
			break
		}
		g.Text(3, 3+i, line, tokenFG(p, dusk.TextMuted, false))
	}
	if !c.typed() {
		if c.note != "" {
			drawStatusMessage(g, p, 3, h-4, width, c.noteKind, c.note)
		}
		drawDialogHints(g, p, 3, h-2, width, []KeyHint{{"esc", "keep"}, {"y", c.verb}})
	}
	return g
}

// confirmationHeight fits c around a body of n wrapped lines.
func confirmationHeight(c confirmation, n int) int {
	h := 6 + n
	switch {
	case c.typed():
		h += 4
	case c.note != "":
		h += 2
	}
	return h
}

func (m *model) confirmDialog() *termgrid.Grid {
	c := m.confirm
	p := m.palette
	w := m.dialogW()
	width := w - 5
	if width < 4 {
		return nil
	}
	h := min(confirmationHeight(c, len(wrapDialogText(c.body, width))), m.height)
	if h < 4 {
		return nil
	}
	g := confirmationGrid(p, w, h, c)
	if !c.typed() {
		return g
	}
	statusRow := h - 4
	m.drawConfirmInput(g, 3, statusRow-2, width)
	if m.confirmMatched() {
		drawStatusMessage(g, p, 3, statusRow, width, StatusDanger, c.matched)
		drawDialogHints(g, p, 3, h-2, width, []KeyHint{{"↵", "delete"}, {"esc", "keep"}})
	} else {
		drawStatusMessage(g, p, 3, statusRow, width, StatusInfo, c.prompt)
		drawDialogHints(g, p, 3, h-2, width, []KeyHint{{"esc", "keep"}})
	}
	return g
}

func (m *model) drawConfirmInput(g *termgrid.Grid, x, y, width int) {
	inputWidth := width - 2
	value := m.confirmInput.Value()
	runes := []rune(value)
	pos := min(m.confirmInput.Position(), len(runes))
	before := ansi.StringWidth(string(runes[:pos]))
	left := max(0, before-inputWidth+1)
	var lines []string
	if value != "" {
		lines = []string{ansi.Cut(value, left, left+inputWidth)}
	}
	drawFocusedInput(g, m.palette, x, y, width, 1, lines, IdleInputField, m.confirm.name, &inputPoint{col: before - left})
}

// Confirmations for each action. Bodies state what is lost with real counts.

func (m *model) confirmDeleteTask(task *domain.Task, prev mode) (tea.Model, tea.Cmd) {
	body := shortID(task.ID)
	var losses []string
	if strings.TrimSpace(task.Description) != "" {
		losses = append(losses, "its description")
	}
	if n := len(task.Whiteboards); n > 0 {
		losses = append(losses, "its "+plural(n, "whiteboard link"))
	}
	if len(losses) == 2 {
		body += ", " + strings.Join(losses, " and ")
	} else if len(losses) == 1 {
		body += " and " + losses[0]
	}
	body += " will be permanently deleted."
	return m.askConfirm(permanent(fmt.Sprintf("Delete task “%s”?", task.Title), body,
		func() (tea.Model, tea.Cmd) { return m.deleteTask(task) }), prev)
}

func (m *model) confirmArchiveTask(task *domain.Task) (tea.Model, tea.Cmd) {
	return m.askConfirm(archival(fmt.Sprintf("Archive task “%s”?", task.Title),
		"It leaves the board and moves to the archive.", "restore it any time from the archive",
		func() (tea.Model, tea.Cmd) { return m.archiveSelected() }), modeBoard)
}

// confirmArchiveOld asks before archiving done tasks untouched for 30 days,
// or says there are none without asking (10/archive-old-none). back is where
// esc and the result land: the board or the archive list.
func (m *model) confirmArchiveOld(back mode) (tea.Model, tea.Cmd) {
	n := len(m.oldDoneTasks())
	if n == 0 {
		m.say(back, StatusInfo, fmt.Sprintf("no done tasks past %d days", bulkArchiveDays))
		return m, nil
	}
	body := fmt.Sprintf("%s in “%s” not updated in over %d days will move to the archive.",
		plural(n, "task"), domain.StatusDone, bulkArchiveDays)
	return m.askConfirm(archival("Archive old done tasks?", body, "restore them any time from the archive",
		func() (tea.Model, tea.Cmd) { return m.archiveOldDone(back) }), back)
}

func (m *model) confirmDeleteColumn(col domain.Status) (tea.Model, tea.Cmd) {
	title := fmt.Sprintf("Delete column “%s”?", string(col))
	action := func() (tea.Model, tea.Cmd) { return m.deleteColumn() }
	n := len(m.board.Order[col])
	if n == 0 {
		return m.askConfirm(permanent(title, "It has no tasks. Only the column itself is removed.", action), modeBoard)
	}
	tasks := "Its " + plural(n, "task")
	if n == 1 {
		tasks = "Its task"
	}
	body := fmt.Sprintf("%s will move to “%s”. Only the column itself is removed.", tasks, m.board.ReplacementColumn(col))
	return m.askConfirm(typedName(title, body, string(col), "delete", "↵ deletes the column for good", action), modeBoard)
}

func (m *model) confirmClearDaily() (tea.Model, tea.Cmd) {
	n := len(m.board.Tasks)
	return m.askConfirm(typedName("Clear the daily board?",
		fmt.Sprintf("All %s on the daily board will be permanently deleted.", plural(n, "task")),
		"daily", "clear", fmt.Sprintf("↵ deletes %s for good", plural(n, "task")),
		func() (tea.Model, tea.Cmd) { return m.clearDailyBoard() }), modeBoard)
}

func projectLoss(project *domain.Project) (total int, body string) {
	archived := 0
	if project.Board != nil {
		for _, t := range project.Board.Tasks {
			if t == nil {
				continue
			}
			total++
			if t.Archived() {
				archived++
			}
		}
	}
	all := "All " + plural(total, "task")
	if total == 1 {
		all = "The only task"
	}
	body = fmt.Sprintf("%s in “%s”", all, project.Name)
	if archived > 0 {
		body += fmt.Sprintf(", including %s,", plural(archived, "archived task"))
	}
	return total, body + " will be permanently deleted."
}

func (m *model) confirmDeleteProject(project *domain.Project, back mode) (tea.Model, tea.Cmd) {
	total, body := projectLoss(project)
	return m.askConfirm(typedName(fmt.Sprintf("Delete project “%s”?", project.Name), body, project.Name, "delete",
		fmt.Sprintf("↵ deletes %s for good", plural(total, "task")),
		func() (tea.Model, tea.Cmd) {
			if err := m.workspace.DeleteProject(project.ID); err != nil {
				m.lastErr = err
				m.mode = back
				return m, nil
			}
			m.activateProject(m.workspace.ActiveProjectID)
			if remaining := len(m.workspace.RegularProjects()); m.projectCursor >= remaining {
				m.projectCursor = remaining - 1
			}
			m.lastErr = nil
			m.lastStatus = fmt.Sprintf("deleted project %s", project.Name)
			m.mode = back
			return m, m.saveWorkspaceCmd()
		}), back)
}

func (m *model) confirmDeleteWhiteboard() (tea.Model, tea.Cmd) {
	whiteboard := m.selectedWhiteboard()
	if whiteboard == nil {
		return m, nil
	}
	body := "The whiteboard and its file will be permanently deleted."
	return m.askConfirm(permanent(fmt.Sprintf("Delete whiteboard “%s”?", whiteboard.Name), body,
		func() (tea.Model, tea.Cmd) { return m.deleteSelectedWhiteboard() }), modeWhiteboards)
}
