package ui

import (
	"errors"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
	"github.com/nikhilsharma/kanban-tui/internal/store"
)

// externalCheckInterval is how often the TUI looks for saves made by another
// process, such as the CLI driven by an agent.
var externalCheckInterval = time.Second

// changeWatcher is implemented by stores that can tell when another process
// saved the workspace.
type changeWatcher interface {
	Changed() (bool, error)
}

type externalCheckMsg struct{}

type externalChangeMsg struct {
	changed bool
	err     error
}

type workspaceReloadedMsg struct {
	workspace *domain.Workspace
	err       error
	// conflict is set when the reload follows a rejected save, so the status
	// line can say the local change was dropped.
	conflict bool
}

func (m *model) scheduleExternalCheck() tea.Cmd {
	if _, ok := m.store.(changeWatcher); !ok {
		return nil
	}
	return tea.Tick(externalCheckInterval, func(time.Time) tea.Msg { return externalCheckMsg{} })
}

func (m *model) checkExternalChange() tea.Cmd {
	watcher, ok := m.store.(changeWatcher)
	if !ok {
		return nil
	}
	return func() tea.Msg {
		changed, err := watcher.Changed()
		return externalChangeMsg{changed: changed, err: err}
	}
}

func (m *model) reloadWorkspaceCmd(conflict bool) tea.Cmd {
	m.reloading = true
	workspaceStore := m.store
	return func() tea.Msg {
		workspace, err := workspaceStore.Load()
		return workspaceReloadedMsg{workspace: workspace, err: err, conflict: conflict}
	}
}

// canReload reports whether the workspace can be swapped without disturbing
// in-flight work: no queued saves, and no dialog holding task or project
// references.
func (m *model) canReload() bool {
	return !m.reloading && m.mode == modeBoard && !m.promotionPending && m.saveSequence == m.latestSaveResult
}

func (m *model) handleExternalCheck() (tea.Model, tea.Cmd) {
	if !m.canReload() {
		return m, m.scheduleExternalCheck()
	}
	return m, m.checkExternalChange()
}

func (m *model) handleExternalChange(msg externalChangeMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil || !msg.changed || !m.canReload() {
		return m, m.scheduleExternalCheck()
	}
	return m, m.reloadWorkspaceCmd(false)
}

func (m *model) handleWorkspaceReloaded(msg workspaceReloadedMsg) (tea.Model, tea.Cmd) {
	m.reloading = false
	if msg.err != nil {
		m.lastErr = fmt.Errorf("reload workspace: %w", msg.err)
	} else {
		m.replaceWorkspace(msg.workspace)
		if msg.conflict {
			m.lastErr = errors.New("board changed elsewhere; reloaded and dropped your last change")
		} else {
			m.lastStatus = "reloaded external changes"
		}
	}
	if msg.conflict {
		// The conflict path does not own the polling loop.
		return m, nil
	}
	return m, m.scheduleExternalCheck()
}

// isSaveConflict reports whether a save failed because another process wrote
// first.
func isSaveConflict(err error) bool {
	return errors.Is(err, store.ErrConflict)
}

// replaceWorkspace swaps in a freshly loaded workspace while keeping the
// user's place: the open project, column, selection and filter.
func (m *model) replaceWorkspace(workspace *domain.Workspace) {
	if workspace == nil {
		return
	}
	assignWorkspaceWhiteboardPaths(workspace, m.dataPath)

	projectID := ""
	if m.project != nil {
		projectID = m.project.ID
	}
	m.workspace = workspace
	if workspace.ProjectByID(projectID) == nil {
		projectID = workspace.ActiveProjectID
	}
	workspace.SetActiveProject(projectID)
	m.project = workspace.ActiveProject()
	if m.project == nil {
		return
	}
	if m.prevProjectID != "" && workspace.ProjectByID(m.prevProjectID) == nil {
		m.prevProjectID = ""
	}
	if !m.project.Daily {
		m.prevProjectID = m.project.ID
	}
	m.board = m.project.Board
	if columns := len(m.board.Statuses()); m.activeColumn >= columns {
		m.activeColumn = max(columns-1, 0)
	}
	m.ensureColumnState()
	m.recalculateVisible()
	m.syncAllScroll()
}
