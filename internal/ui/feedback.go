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

// undoSpot is where one task sat before a move or archive.
type undoSpot struct {
	taskID   string
	status   domain.Status
	index    int
	archived bool
}

// undoEntry puts the tasks of the last move or archive back where they were.
// It lives only while the toast that offers it shows; nothing is stored.
type undoEntry struct {
	spots  []undoSpot // in board order, so restoring by index works forward
	undone string     // toast after undoing
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

// spotOf records where a task sits now, so undo can put it back.
func (m *model) spotOf(task *domain.Task) undoSpot {
	index := 0
	for i, id := range m.board.Order[task.Status] {
		if id == task.ID {
			index = i
			break
		}
	}
	return undoSpot{taskID: task.ID, status: task.Status, index: index}
}

// moveUndo is the entry for moving one task out of spot.
func moveUndo(spot undoSpot) undoEntry {
	return undoEntry{spots: []undoSpot{spot}, undone: "Moved back to " + columnName(spot.status)}
}

func (m *model) applyUndo(e undoEntry) (tea.Model, tea.Cmd) {
	for _, s := range e.spots {
		if m.board.Tasks[s.taskID] == nil {
			return m, m.showToast(StatusDanger, "Nothing to undo · task is gone", KeyHint{})
		}
	}
	for _, s := range e.spots {
		task := m.board.Tasks[s.taskID]
		if s.archived {
			if _, err := m.board.RestoreTask(s.taskID); err != nil {
				return m, m.showToast(StatusDanger, "Could not undo · "+err.Error(), KeyHint{})
			}
		}
		if task.Status == s.status {
			// MoveTask would land one short when moving down a column.
			m.board.MoveWithin(s.status, indexIn(m.board.Order[s.status], s.taskID), s.index)
		} else {
			m.board.MoveTask(s.taskID, s.status, s.index)
		}
	}
	m.recalculateVisible()
	m.selectTask(e.spots[0].taskID)
	return m, tea.Batch(m.saveWorkspaceCmd(), m.showToast(StatusSuccess, e.undone, KeyHint{}))
}

func indexIn(ids []string, id string) int {
	for i, candidate := range ids {
		if candidate == id {
			return i
		}
	}
	return -1
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
