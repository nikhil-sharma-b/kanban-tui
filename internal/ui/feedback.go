package ui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
)

// Feedback timings (page 15). Tests shorten them through Options.
const (
	defaultToastDuration     = 3 * time.Second
	defaultSaveRetryInterval = 2 * time.Second
	defaultReloadedDuration  = 2 * time.Second
)

// toast is the transient message bottom-right above the keybar. One shows at
// a time; a newer one replaces it.
type toast struct {
	kind StatusKind
	text string
	hint KeyHint
}

// undoEntry puts one moved or archived task back where it was. It lives only
// while the toast that offers it shows; nothing about it is stored.
type undoEntry struct {
	taskID   string
	status   domain.Status
	index    int
	archived bool
}

type toastExpiredMsg struct{ seq uint64 }

type saveRetryMsg struct{}

type reloadedExpiredMsg struct{ seq uint64 }

// showToast replaces the current toast and drops any undo it offered.
func (m *model) showToast(kind StatusKind, text string, hint KeyHint) tea.Cmd {
	m.toastSeq++
	m.toast = toast{kind: kind, text: text, hint: hint}
	m.undo = nil
	seq := m.toastSeq
	return tea.Tick(m.toastDuration, func(time.Time) tea.Msg { return toastExpiredMsg{seq: seq} })
}

func (m *model) showUndoToast(kind StatusKind, text string, entry undoEntry) tea.Cmd {
	cmd := m.showToast(kind, text, undoHint)
	m.undo = &entry
	return cmd
}

var undoHint = KeyHint{Key: "u", Label: "undo"}

func (m *model) clearToast() {
	m.toast = toast{}
	m.undo = nil
}

// toastRoom is the widest a toast may be: half the screen.
func (m *model) toastRoom(hint KeyHint) int {
	room := m.width/2 - 6
	if hint.Key != "" {
		room -= 3 + hint.width()
	}
	return room
}

// quotedToast fits “title” into the toast room, shortening only the title.
func (m *model) quotedToast(prefix, title string, hint KeyHint) string {
	title = singleLine(title)
	room := m.toastRoom(hint) - ansi.StringWidth(prefix) - 2
	return prefix + "“" + truncateCell(title, room) + "”"
}

// undoSpot records where a task sits now, so undo can put it back.
func (m *model) undoSpot(task *domain.Task) undoEntry {
	index := 0
	for i, id := range m.board.Order[task.Status] {
		if id == task.ID {
			index = i
			break
		}
	}
	return undoEntry{taskID: task.ID, status: task.Status, index: index}
}

func (m *model) applyUndo(e undoEntry) (tea.Model, tea.Cmd) {
	task := m.board.Tasks[e.taskID]
	if task == nil {
		return m, m.showToast(StatusDanger, "Nothing to undo · task is gone", KeyHint{})
	}
	text := "Moved back to " + columnName(e.status)
	if e.archived {
		if _, err := m.board.RestoreTask(e.taskID); err != nil {
			return m, m.showToast(StatusDanger, "Could not undo · "+err.Error(), KeyHint{})
		}
		text = m.quotedToast("Restored ", task.Title, KeyHint{})
	}
	m.board.MoveTask(e.taskID, e.status, e.index)
	m.recalculateVisible()
	m.selectTask(e.taskID)
	return m, tea.Batch(m.saveWorkspaceCmd(), m.showToast(StatusSuccess, text, KeyHint{}))
}

func columnName(status domain.Status) string {
	return strings.ToLower(status.Title())
}

// handleSaveFinished keeps retrying a failed save until one succeeds; the
// header reads “save failed · retrying” meanwhile (page 01 DEV NOTE).
func (m *model) handleSaveFinished(err error) (tea.Model, tea.Cmd) {
	if err != nil {
		var cmds []tea.Cmd
		if !m.saveFailing {
			cmds = append(cmds, m.showToast(StatusDanger, "Could not save · retrying", KeyHint{}))
		}
		m.saveFailing = true
		m.savedToast = ""
		if !m.retryPending {
			m.retryPending = true
			cmds = append(cmds, tea.Tick(m.saveRetryInterval, func(time.Time) tea.Msg { return saveRetryMsg{} }))
		}
		return m, tea.Batch(cmds...)
	}
	m.saveFailing = false
	if id := m.savedToast; id != "" {
		m.savedToast = ""
		if m.mode == modeBoard {
			return m, m.showToast(StatusSuccess, "Saved "+id, KeyHint{})
		}
	}
	return m, nil
}

func (m *model) handleSaveRetry() (tea.Model, tea.Cmd) {
	m.retryPending = false
	if !m.saveFailing {
		return m, nil
	}
	// Saving the whole workspace again carries every change made since.
	return m, m.saveWorkspaceCmd()
}

// showReloaded marks the header and toasts an external write.
func (m *model) showReloaded() tea.Cmd {
	m.reloadedSeq++
	m.reloaded = true
	seq := m.reloadedSeq
	return tea.Batch(
		m.showToast(StatusInfo, "Board reloaded · changed by CLI", KeyHint{}),
		tea.Tick(m.reloadedDuration, func(time.Time) tea.Msg { return reloadedExpiredMsg{seq: seq} }),
	)
}

func (m *model) syncState() Sync {
	switch {
	case m.saveFailing:
		return SyncError
	case m.reloaded:
		return SyncReloaded
	}
	return SyncLive
}
