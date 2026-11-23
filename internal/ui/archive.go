package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
	"github.com/nikhilsharma/kanban-tui/internal/termgrid"
	"github.com/nikhilsharma/kanban-tui/internal/ui/dusk"
)

// archiveListRows keeps the archive dialog one height whatever it holds.
const archiveListRows = 7

// openArchive opens the archive list (SPC a v, page 10).
func (m *model) openArchive() (tea.Model, tea.Cmd) {
	m.mode = modeArchive
	m.archiveCursor = 0
	m.archiveFiltering = false
	m.archiveFilterInput.SetValue("")
	m.archiveFilterInput.Blur()
	m.archiveStatus, m.lastErr = "", nil
	return m, nil
}

// filteredArchivedTasks returns archived tasks, newest first, whose title or
// description contains the search query, ignoring case.
func (m *model) filteredArchivedTasks() []*domain.Task {
	archived := m.board.ArchivedTasks()
	query := strings.ToLower(strings.TrimSpace(m.archiveFilterInput.Value()))
	if query == "" {
		return archived
	}
	matches := make([]*domain.Task, 0, len(archived))
	for _, task := range archived {
		if strings.Contains(task.SearchText(), query) {
			matches = append(matches, task)
		}
	}
	return matches
}

func (m *model) selectedArchivedTask() *domain.Task {
	archived := m.filteredArchivedTasks()
	if len(archived) == 0 {
		return nil
	}
	m.archiveCursor = max(0, min(m.archiveCursor, len(archived)-1))
	return archived[m.archiveCursor]
}

// archiveWhen is relative under 7 days, else a date ("12 Nov").
func archiveWhen(now, t time.Time) string {
	d := now.Sub(t)
	switch {
	case d < 24*time.Hour:
		return "today"
	case d < 48*time.Hour:
		return "yesterday"
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
	return t.Local().Format("2 Jan")
}

// restoreTarget is the column a restore lands in, and whether archived_from
// is gone (RestoreTask then falls back to the first column).
func (m *model) restoreTarget(task *domain.Task) (domain.Status, bool) {
	if m.board.StatusIndex(task.ArchivedFrom) >= 0 || len(m.board.Columns) == 0 {
		return task.ArchivedFrom, false
	}
	return m.board.Columns[0], true
}

// say shows a message where the user is: the archive footer or a board toast.
func (m *model) say(where mode, kind StatusKind, text string) {
	m.mode = where
	if where == modeArchive {
		m.archiveStatus, m.archiveStatusKind = text, kind
		return
	}
	if kind == StatusInfo {
		m.lastInfo = text
	} else {
		m.lastStatus = text
	}
}

func (m *model) moveArchiveCursor(down bool, n int) {
	if down && m.archiveCursor < n-1 {
		m.archiveCursor++
	} else if !down && m.archiveCursor > 0 {
		m.archiveCursor--
	}
}

func (m *model) openArchiveDetail() {
	if m.selectedArchivedTask() != nil {
		m.mode = modeArchiveDetail
		m.detailScroll, m.detailHelp = 0, false
	}
}

func (m *model) updateArchive(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.archiveStatus = ""
	key := msg.String()
	archived := m.filteredArchivedTasks()
	m.archiveCursor = max(0, min(m.archiveCursor, len(archived)-1))

	if m.archiveFiltering {
		switch key {
		case "esc":
			m.archiveFiltering = false
			m.archiveFilterInput.SetValue("")
			m.archiveFilterInput.Blur()
			m.archiveCursor = 0
		case "up", "down":
			m.moveArchiveCursor(key == "down", len(archived))
		case "enter":
			m.openArchiveDetail()
		default:
			before := m.archiveFilterInput.Value()
			var cmd tea.Cmd
			m.archiveFilterInput, cmd = m.archiveFilterInput.Update(msg)
			if m.archiveFilterInput.Value() != before {
				m.archiveCursor = 0
			}
			return m, cmd
		}
		return m, nil
	}

	switch key {
	case "esc":
		m.mode = modeBoard
		m.lastErr = nil
	case "/":
		m.archiveFiltering = true
		m.archiveFilterInput.CursorEnd()
		m.archiveFilterInput.Focus()
		return m, textinput.Blink
	case "k", "up":
		m.moveArchiveCursor(false, len(archived))
	case "j", "down":
		m.moveArchiveCursor(true, len(archived))
	case "enter":
		m.openArchiveDetail()
	case "r":
		return m.restoreArchived()
	case "a":
		return m.confirmArchiveOld(modeArchive)
	}
	return m, nil
}

func (m *model) updateArchiveDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	task := m.selectedArchivedTask()
	if task == nil {
		m.mode = modeArchive
		return m, nil
	}
	switch msg.String() {
	case "esc":
		m.mode = modeArchive
	case "r":
		m.mode = modeArchive
		return m.restoreArchived()
	case "?":
		m.detailHelp = !m.detailHelp
	case "J", "K":
		l := m.detailLayout(max(1, len(task.Whiteboards)))
		lines := wrapDialogText(task.Description, l.width-5)
		if msg.String() == "J" {
			m.detailScroll = min(max(0, len(lines)-l.descRows), m.detailScroll+1)
		} else {
			m.detailScroll = max(0, m.detailScroll-1)
		}
	}
	return m, nil
}

// restoreArchived restores the selected task at once (it is reversible): the
// row leaves the list, the cursor keeps its index, and the footer names
// where it went.
func (m *model) restoreArchived() (tea.Model, tea.Cmd) {
	task := m.selectedArchivedTask()
	if task == nil {
		return m, nil
	}
	restored, err := m.board.RestoreTask(task.ID)
	if err != nil {
		m.lastErr = err
		return m, nil
	}
	// RestoreTask appends; the board shows the restored task on top of its
	// column instead, where it is seen first (10/restored).
	order := m.board.Order[restored.Status]
	if n := len(order); n > 0 && order[n-1] == restored.ID {
		m.board.Order[restored.Status] = append([]string{restored.ID}, order[:n-1]...)
	}
	m.lastErr = nil
	m.archiveStatus = fmt.Sprintf("restored “%s” to %s", restored.Title, restored.Status)
	m.archiveStatusKind = StatusSuccess
	m.ensureColumnState()
	m.recalculateVisible()
	m.syncScroll(restored.Status)
	m.archiveCursor = max(0, min(m.archiveCursor, len(m.filteredArchivedTasks())-1))
	return m, m.saveWorkspaceCmd()
}

// oldDoneTasks are the done tasks not updated in over bulkArchiveDays.
func (m *model) oldDoneTasks() []string {
	cutoff := m.clock().AddDate(0, 0, -bulkArchiveDays)
	var ids []string
	for _, id := range m.board.Order[domain.StatusDone] {
		if t := m.board.Tasks[id]; t != nil && !t.Archived() && t.UpdatedAt.Before(cutoff) {
			ids = append(ids, id)
		}
	}
	return ids
}

func (m *model) archiveOldDone(back mode) (tea.Model, tea.Cmd) {
	ids := m.oldDoneTasks()
	var spots []undoSpot
	for _, id := range ids {
		spot := m.spotOf(m.board.Tasks[id])
		spot.archived = true
		spots = append(spots, spot)
	}
	// Restore in column order so each task lands back at its own index.
	sort.Slice(spots, func(i, j int) bool { return spots[i].index < spots[j].index })
	for _, id := range ids {
		if _, err := m.board.ArchiveTask(id); err != nil {
			m.mode = back
			m.lastErr = err
			return m, nil
		}
	}
	m.lastErr = nil
	text := fmt.Sprintf("archived %s", plural(len(ids), "done task"))
	m.recalculateVisible()
	if back != modeBoard || len(spots) == 0 {
		m.say(back, StatusSuccess, text)
		return m, m.saveWorkspaceCmd()
	}
	m.mode = back
	return m, tea.Batch(m.saveWorkspaceCmd(), m.showUndoToast(StatusInfo, "A"+text[1:],
		undoEntry{spots: spots, undone: "Restored " + plural(len(spots), "task")}))
}

func (m *model) archiveHints(shown, total int) [][]KeyHint {
	query := strings.TrimSpace(m.archiveFilterInput.Value()) != ""
	switch {
	case m.archiveFiltering && shown == 0 && query:
		return [][]KeyHint{{{"esc", "clear filter"}}}
	case m.archiveFiltering:
		return [][]KeyHint{{{"↑↓", "move"}, {"↵", "open"}, {"esc", "clear filter"}}}
	case total == 0:
		return [][]KeyHint{{{"a", "archive old done"}, {"esc", "close"}}}
	}
	return [][]KeyHint{
		{{"j k", "move"}, {"↵", "open"}, {"/", "search"}, {"esc", "close"}},
		{{"r", "restore"}, {"a", "archive old done"}},
	}
}

// archiveDialog is the archive list (10/list, 10/searching, 10/no-match,
// 10/empty, 10/restored): the search field, seven rows, and the footer.
func (m *model) archiveDialog() *termgrid.Grid {
	p := m.palette
	w := min(dialogMaxWidth, m.width-8)
	rows := min(archiveListRows, m.height-8)
	if w < 24 || rows < 1 {
		return nil
	}
	h := rows + 7 // borders, search, blanks around the list, two footer rows
	g := newDialog(p, w, h)
	width := w - 5

	total := len(m.board.ArchivedTasks())
	archived := m.filteredArchivedTasks()
	query := strings.TrimSpace(m.archiveFilterInput.Value())
	label := fmt.Sprintf("archive · %d", total)
	if query != "" {
		label = fmt.Sprintf("archive · %d of %d", len(archived), total)
	}
	g.Text(w-2-ansi.StringWidth(label), 1, label, tokenFG(p, dusk.TextDim, false))
	inputWidth := width - ansi.StringWidth(label) - 1
	placeholder := "Search archive…"
	if m.archiveFiltering {
		lines, col := cutInput(m.archiveFilterInput.Value(), m.archiveFilterInput.Position(), inputWidth)
		drawFocusedInput(g, p, 1, 1, inputWidth+2, 1, lines, IdleInputField, placeholder, &inputPoint{col: col})
	} else {
		var lines []string
		if query != "" {
			lines = []string{query}
		}
		drawIdleInput(g, p, 1, 1, inputWidth+2, lines, IdleInputField, placeholder)
	}

	const top = 3
	switch {
	case total == 0:
		for i, line := range []string{"nothing archived yet", "SPC t a on the board archives one task", "a archives done tasks older than 30 days"} {
			if i < rows {
				drawListRow(g, p, 1, top+i, w-3, line, "", ListRowDisabled)
			}
		}
	case len(archived) == 0:
		drawListRow(g, p, 1, top, w-3, fmt.Sprintf("no archived task matches “%s”", query), "", ListRowDisabled)
	}
	now := m.clock()
	first := max(0, min(m.archiveCursor-rows+1, len(archived)-rows))
	for i := first; i < len(archived) && i < first+rows; i++ {
		task := archived[i]
		state := ListRowDefault
		if i == m.archiveCursor {
			state = ListRowSelected
		}
		hint := fmt.Sprintf("from %s · %s", task.ArchivedFrom, archiveWhen(now, *task.ArchivedAt))
		drawListRow(g, p, 1, top+i-first, w-3, task.Title, hint, state)
		// The hint reads text-muted, as in the projects list (10/list).
		g.Text(w-2-ansi.StringWidth(hint), top+i-first, hint, tokenFG(p, dusk.TextMuted, false))
	}

	switch {
	case m.lastErr != nil:
		drawStatusMessage(g, p, 3, h-3, width, StatusDanger, m.lastErr.Error())
	case m.archiveStatus != "":
		drawStatusMessage(g, p, 3, h-3, width, m.archiveStatusKind, m.archiveStatus)
	default:
		for i, row := range m.archiveHints(len(archived), total) {
			drawDialogHints(g, p, 3, h-3+i, width, row)
		}
	}
	return g
}

// archiveDetail is the 03 detail dialog, read-only, for an archived task
// (10/detail, 10/restore-fallback).
func (m *model) archiveDetail() *termgrid.Grid {
	task := m.selectedArchivedTask()
	if task == nil {
		return nil
	}
	p := m.palette
	l := m.detailLayout(max(1, len(task.Whiteboards)))
	if l.width < 6 || l.height < 5 || l.footer < 2 {
		return nil
	}
	g := newDialog(p, l.width, l.height)
	const x = 3
	right := l.width - 2
	textWidth := right - x
	now := m.clock()

	drawIdleInput(g, p, 1, 1, textWidth+2, []string{task.Title}, IdleInputTitle, "")
	target, deleted := m.restoreTarget(task)
	meta := fmt.Sprintf("archived %s · from %s", archiveWhen(now, *task.ArchivedAt), task.ArchivedFrom)
	if deleted {
		meta += " (deleted)"
	}
	// The meta wins over a long title (10/restore-fallback).
	meta = truncateCell(meta, textWidth)
	g.Text(right-ansi.StringWidth(meta), 1, meta, tokenFG(p, dusk.TextDim, false))

	lines := wrapDialogText(task.Description, textWidth)
	start := min(m.detailScroll, max(0, len(lines)-l.descRows))
	end := min(len(lines), start+l.descRows)
	drawIdleInput(g, p, 1, detailDescRow, textWidth+2, lines[start:end], IdleInputBody, "")

	if l.listRows > 0 {
		y := detailDescRow + l.descRows + 1
		g.Text(x, y, truncateCell(fmt.Sprintf("whiteboards · %d", len(task.Whiteboards)), textWidth), tokenFG(p, dusk.TextDim, false))
		for i := 0; i < min(len(task.Whiteboards), l.listRows); i++ {
			wb := task.Whiteboards[i]
			format := wb.Format
			if format == "" {
				format = strings.TrimPrefix(wb.Extension(), ".")
			}
			drawListRow(g, p, 1, y+1+i, right-1, "◇ "+wb.Name, format+" · "+relativeTime(now, wb.UpdatedAt), ListRowDefault)
		}
	}

	restore := KeyHint{"r", "restore"}
	if deleted {
		restore.Label = "restore to " + string(target)
	}
	hints := []KeyHint{restore, {"esc", "back"}, {"?", "more"}}
	if m.detailHelp {
		hints = []KeyHint{{"J K", "scroll"}, {"esc", "back"}, {"?", "less"}}
	}
	drawDialogHints(g, p, x, l.footer, textWidth, hints)
	return g
}
