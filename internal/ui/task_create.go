package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
	"github.com/nikhilsharma/kanban-tui/internal/termgrid"
	"github.com/nikhilsharma/kanban-tui/internal/ui/dusk"
)

type taskCreatedMsg struct {
	task *domain.Task
	err  error
}

func (m *model) updateNewTask(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.createSaving || m.reloading {
		return m, nil
	}
	if m.editDiscard {
		switch msg.String() {
		case "y":
			m.closeCreatePanel()
		case "n", "esc":
			m.editDiscard = false
		}
		return m, nil
	}
	m.detailStatus = ""
	if m.detailStatusMenu {
		return m.updateStatusMenu(msg.String(), m.createColumn, func(cursor int) tea.Cmd {
			m.createColumn = m.board.Statuses()[cursor]
			return nil
		})
	}
	if !m.vim.pending() && m.vimVisual == nil && !m.vimReplace && m.vimCommand == "" {
		switch msg.String() {
		case "s":
			if m.vimNormal {
				m.editHelp = false
				m.detailStatusMenu = true
				m.detailStatusCursor = m.board.StatusIndex(m.createColumn)
				return m, nil
			}
		case "?":
			// In Insert, a second '?' types a literal question mark.
			if !m.editHelp || m.vimNormal {
				m.editHelp = !m.editHelp
				return m, nil
			}
		}
	}
	if m.editHelp {
		m.editHelp = false
		if msg.String() == "esc" {
			return m, nil
		}
	}
	return m.updateTaskEditor(msg, m.createLayout())
}

func (m *model) createTask(title, description string) (tea.Model, tea.Cmd) {
	if m.createProjectID != "" && m.project.ID != m.createProjectID {
		m.detailStatus = "original project no longer exists"
		return m, nil
	}
	// Never capture a workspace while an earlier save can still trigger a
	// reload and advance the store revision out from under this snapshot.
	if m.reloading || m.saveSequence != m.latestSaveResult {
		m.detailStatus = "board is still saving; retry"
		return m, nil
	}
	task, err := m.newTask(title, description)
	if err != nil {
		m.detailStatus = err.Error()
		if strings.TrimSpace(title) == "" {
			m.detailStatus = "title can't be empty"
			m.descInput.Blur()
			m.titleInput.Focus()
			m.vimNormal = false
		}
		return m, nil
	}
	status := m.createColumn
	if status == "" {
		status = m.board.Statuses()[m.activeColumn]
	}
	if task == nil {
		m.detailStatus = "invalid new task"
		return m, nil
	}
	task.Status = status
	result := taskCreatedMsg{task: task}
	if m.store == nil {
		return m.finishCreateTask(result)
	}

	// Save a candidate first so a failed write never leaves an unsaved card
	// on the board, and the dialog still holds everything needed for retry.
	workspace := m.workspace.Clone()
	project := workspace.ProjectByID(m.project.ID)
	if err := project.Board.AppendTask(task, status); err != nil {
		m.detailStatus = err.Error()
		return m, nil
	}
	project.Touch()
	m.createSaving = true
	return m, m.queueSave(func() tea.Msg {
		result.err = m.store.Save(workspace)
		return result
	})
}

func (m *model) finishCreateTask(msg taskCreatedMsg) (tea.Model, tea.Cmd) {
	m.createSaving = false
	if msg.err != nil {
		if isSaveConflict(msg.err) {
			m.detailStatus = "board changed elsewhere; retry after reload"
			if !m.reloading {
				return m, m.reloadWorkspaceCmd(true)
			}
		} else {
			m.detailStatus = "save failed: " + msg.err.Error()
		}
		return m, nil
	}
	task := msg.task
	if err := m.board.AppendTask(task, task.Status); err != nil {
		m.detailStatus = err.Error()
		return m, nil
	}
	m.project.Touch()

	m.lastStatus, m.lastErr = "", nil
	m.filter = ""
	m.searchInput.SetValue("")
	m.recalculateVisible()
	m.selectTask(task.ID)
	m.closeCreatePanel()
	m.createFeedback = fmt.Sprintf("created %s in %s", shortID(task.ID), strings.ToLower(task.Status.Title()))
	return m, nil
}

func (m *model) createLayout() detailLayout {
	l := m.detailLayout(0)
	// New-task keeps its footer one row above the bottom border at both sizes.
	l.footer = l.height - 2
	l.descRows = max(0, l.footer-detailDescRow-2)
	return l
}

func (m *model) taskCreate() *termgrid.Grid {
	l := m.createLayout()
	if l.width < 6 || l.height < 5 || l.footer < 2 {
		return nil
	}
	p := m.palette
	g := newDialog(p, l.width, l.height)
	width := l.width - 5
	status := "new · " + strings.ToLower(m.createColumn.Title())
	if m.editDiscard {
		status = "modified"
	} else if m.detailStatusMenu {
		status += " ▾"
	}
	status = truncateCell(status, width/2)
	titleWidth := width
	if status != "" {
		titleWidth -= ansi.StringWidth(status) + 1
		g.Text(l.width-2-ansi.StringWidth(status), 1, status, tokenFG(p, dusk.TextDim, false))
	}

	if m.titleInput.Focused() && !m.detailStatusMenu && !m.editDiscard && m.vimCommand == "" {
		m.drawEditTitle(g, titleWidth, "title")
	} else {
		var lines []string
		if title := m.titleInput.Value(); title != "" {
			lines = []string{title}
		}
		drawIdleInput(g, p, 1, 1, titleWidth+2, lines, IdleInputTitle, "title")
	}
	if l.descRows > 0 {
		if m.descInput.Focused() && !m.detailStatusMenu && !m.editDiscard && m.vimCommand == "" {
			m.drawEditDescription(g, l, width, !m.descInput.Cursor.Blink)
		} else {
			lines := wrapDialogText(m.descInput.Value(), width)
			top := min(m.editDescTop, max(0, len(lines)-l.descRows))
			drawIdleInput(g, p, 1, detailDescRow, width+2, lines[top:min(top+l.descRows, len(lines))], IdleInputBody, "description (optional)")
		}
	}

	switch {
	case m.editDiscard:
		drawDiscardFooter(g, p, l.footer, width)
	case m.detailStatusMenu:
		m.drawDetailStatusMenu(g, l, m.createColumn)
		drawDialogHints(g, p, 3, l.footer, width, []KeyHint{{"j k", "move"}, {"↵", "set"}, {"esc", "cancel"}})
	case m.detailStatus != "":
		drawStatusMessage(g, p, 3, l.footer, width, StatusDanger, m.detailStatus)
	case m.createSaving:
		drawStatusMessage(g, p, 3, l.footer, width, StatusInfo, "saving…")
	case m.vimCommand != "":
		drawCommandLine(g, p, 3, l.footer, width, m.vimCommand)
	case m.vimStatus != "":
		drawStatusMessage(g, p, 3, l.footer, width, StatusInfo, m.vimStatus)
	default:
		hints := []KeyHint{{"↵", "create"}, {"tab", "description"}, {"esc", "normal"}, {"?", "more"}}
		if m.descInput.Focused() {
			hints[0], hints[1] = KeyHint{":wq", "create"}, KeyHint{"ctrl+e", "$EDITOR"}
		}
		if m.vimNormal {
			hints[0], hints[1], hints[2] = KeyHint{":wq", "create"}, KeyHint{"s", "column"}, KeyHint{"esc", "close"}
		}
		if m.editHelp {
			if m.vimNormal {
				hints = []KeyHint{{":wq", "create"}, {"i", "insert"}, {"tab", "next field"}, {"?", "less"}}
			} else {
				hints = []KeyHint{{"tab", "next field"}, {"ctrl+e", "$EDITOR"}, {"esc", "back"}, {"?", "insert ?"}}
			}
		}
		drawDialogHints(g, p, 3, l.footer, width, hints)
	}
	return g
}
