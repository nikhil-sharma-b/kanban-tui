package ui

// The command palette (page 12): : on the board, or in task details while
// nothing is being edited, opens a fixed-size Dialog listing every action in
// boardBindings with its keys. Typing fuzzy-matches label and key. Actions
// ending in … open a second step in the same dialog, and esc steps back.
// Opened from details, the palette takes the details' place and esc swaps
// them back, so dialogs never stack.

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/nikhilsharma/kanban-tui/internal/termgrid"
	"github.com/nikhilsharma/kanban-tui/internal/ui/dusk"
)

const paletteRecentMax = 3

type paletteStep int

const (
	paletteActions paletteStep = iota
	paletteProjects
	paletteStatuses
	paletteThemes
	paletteKeys
)

func (s paletteStep) isRoot() bool { return s == paletteActions || s == paletteKeys }

type paletteState struct {
	input      textinput.Model
	cursor     int
	step       paletteStep
	fromDetail bool
	// recent are the last actions run from the palette this session, most
	// recent first. They live in memory only.
	recent []boardAction
}

type paletteItem struct {
	label, hint string
	action      boardAction // a board action, or actNone
	keys        string      // the binding the hint shows, for key matching
	current     bool        // the project or column already in effect: shown disabled
	run         func() (tea.Model, tea.Cmd)
}

type paletteSection struct {
	title string
	items []paletteItem
}

// paletteFirst leads the list in this order; every other action follows in
// the order boardBindings lists it, so a new binding shows up on its own.
var paletteFirst = []boardAction{
	actTaskNew, actOpen, actTaskEdit, actTaskMoveLeft, actTaskMoveRight,
	actTaskArchive, actTaskDelete, actProjects, actTaskCopyID, actTaskWhiteboards,
}

// paletteFirstFromDetail leads the board actions offered over details: the
// ones that act on the open task.
var paletteFirstFromDetail = []boardAction{actTaskArchive, actTaskMoveRight, actTaskDelete, actTaskMoveLeft}

// paletteHidden never appear: help has its own key in the footer, and the
// palette does not list itself.
var paletteHidden = map[boardAction]bool{actHelp: true, actPalette: true}

// paletteDetailCovered are board actions the details section already offers,
// or that only make sense on the board.
var paletteDetailCovered = map[boardAction]bool{
	actColumnLeft: true, actColumnRight: true, actTaskUp: true, actTaskDown: true,
	actTaskTop: true, actTaskBottom: true, actOpen: true, actTaskEdit: true,
	actTaskWhiteboards: true, actTaskCopyID: true,
}

// paletteBindings returns the first binding of each listable action, leading
// actions first.
func paletteBindings(first []boardAction, skip map[boardAction]bool) []boardBinding {
	byAction := map[boardAction]boardBinding{}
	var order []boardAction
	for _, b := range boardBindings {
		if _, ok := byAction[b.action]; ok || paletteHidden[b.action] || skip[b.action] {
			continue
		}
		byAction[b.action] = b
		order = append(order, b.action)
	}
	var out []boardBinding
	seen := map[boardAction]bool{}
	for _, a := range append(append([]boardAction(nil), first...), order...) {
		if b, ok := byAction[a]; ok && !seen[a] {
			seen[a] = true
			out = append(out, b)
		}
	}
	return out
}

func actionsForKeys(keys []string) []boardAction {
	var out []boardAction
	for _, k := range keys {
		for _, b := range boardBindings {
			if b.keys == k {
				out = append(out, b.action)
				break
			}
		}
	}
	return out
}

func paletteLabel(b boardBinding) string {
	if b.label != "" {
		return b.label
	}
	return b.desc
}

// paletteKeyHint spells a binding the way hints do: SPC for the leader, ↵
// for enter.
func paletteKeyHint(keys string) string {
	if keys == "enter" {
		return "↵"
	}
	return displayKeys(keys)
}

func (m *model) openPalette(fromDetail bool) (tea.Model, tea.Cmd) {
	return m.openPaletteAt(paletteActions, fromDetail)
}

func (m *model) openPaletteAt(step paletteStep, fromDetail bool) (tea.Model, tea.Cmd) {
	m.mode = modePalette
	m.pal.fromDetail = fromDetail
	m.resetForPalette()
	m.enterPaletteStep(step)
	return m, textinput.Blink
}

func (m *model) resetForPalette() {
	m.cancelPending()
	m.lastErr, m.lastStatus = nil, ""
}

func (m *model) enterPaletteStep(step paletteStep) {
	m.pal.step = step
	m.pal.input = textinput.New()
	m.pal.input.Prompt = ""
	m.pal.input.Focus()
	m.pal.cursor = 0
	// The current project or status is already what it is: start on the next.
	for i, item := range m.paletteItems() {
		if !item.current {
			m.pal.cursor = i
			break
		}
	}
}

func (m *model) closePalette() {
	m.pal.input.Blur()
	if m.pal.fromDetail {
		m.mode = modeDetail
		return
	}
	m.mode = modeBoard
}

// leavePalette closes the palette and the details under it, for actions
// that run on the board.
func (m *model) leavePalette() {
	m.pal.input.Blur()
	m.mode = modeBoard
}

func (m *model) rememberPaletteAction(a boardAction) {
	recent := []boardAction{a}
	for _, r := range m.pal.recent {
		if r != a && len(recent) < paletteRecentMax {
			recent = append(recent, r)
		}
	}
	m.pal.recent = recent
}

func (m *model) boardPaletteItem(b boardBinding) paletteItem {
	item := paletteItem{label: paletteLabel(b), hint: paletteKeyHint(b.keys), action: b.action, keys: b.keys}
	action := b.action
	item.run = func() (tea.Model, tea.Cmd) {
		if action == actProjects {
			// Remembered once a project is picked, not on entering the step.
			m.enterPaletteStep(paletteProjects)
			return m, nil
		}
		m.rememberPaletteAction(action)
		m.leavePalette()
		return m.runBoardAction(action)
	}
	return item
}

// detailPaletteItems are the keys of task details in view mode.
func (m *model) detailPaletteItems() []paletteItem {
	key := func(k string) func() (tea.Model, tea.Cmd) {
		return func() (tea.Model, tea.Cmd) {
			m.closePalette()
			return m.updateDetail(keyMsgFor(k))
		}
	}
	return []paletteItem{
		{label: "edit title and description", hint: "e", run: key("e")},
		{label: "set status…", hint: "s", run: func() (tea.Model, tea.Cmd) {
			m.enterPaletteStep(paletteStatuses)
			return m, nil
		}},
		{label: "whiteboards…", hint: "w", run: key("w")},
		{label: "copy task ID", hint: "y", run: key("y")},
		{label: "edit description in $EDITOR", hint: "ctrl+e", run: func() (tea.Model, tea.Cmd) {
			m.closePalette()
			if _, cmd := m.beginEditSelected(); m.mode != modeDetailEdit {
				return m, cmd
			}
			return m.openEditorWithDraft()
		}},
		{label: "close details", hint: "esc", run: key("esc")},
	}
}

func keyMsgFor(k string) tea.KeyMsg {
	if k == "esc" {
		return tea.KeyMsg{Type: tea.KeyEsc}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
}

func (m *model) projectPaletteItems() []paletteItem {
	query := strings.TrimSpace(m.pal.input.Value())
	var items []paletteItem
	for _, project := range m.workspace.RegularProjects() {
		if query != "" && !fuzzyMatch(query, project.Name) {
			continue
		}
		hint := plural(len(project.Board.Statuses()), "column")
		if project.ID == m.workspace.ActiveProjectID {
			hint = "current · " + hint
		}
		id := project.ID
		items = append(items, paletteItem{label: project.Name, hint: hint, current: id == m.workspace.ActiveProjectID, run: func() (tea.Model, tea.Cmd) {
			m.rememberPaletteAction(actProjects)
			m.leavePalette()
			return m.switchProject(id)
		}})
	}
	return items
}

func (m *model) statusPaletteItems() []paletteItem {
	task := m.selectedTask()
	if task == nil {
		return nil
	}
	query := strings.TrimSpace(m.pal.input.Value())
	var items []paletteItem
	for i, status := range m.board.Statuses() {
		name := columnName(status)
		if query != "" && !fuzzyMatch(query, name) {
			continue
		}
		hint := plural(m.board.Count(status), "task")
		if status == task.Status {
			hint = "current · " + hint
		}
		cursor := i
		items = append(items, paletteItem{label: name, hint: hint, current: status == task.Status, run: func() (tea.Model, tea.Cmd) {
			m.closePalette()
			if status == task.Status {
				return m, nil
			}
			m.detailStatusMenu, m.detailStatusCursor = true, cursor
			return m.updateDetailStatusMenu("enter")
		}})
	}
	return items
}

// paletteSections is what the list shows for the current step and query.
func (m *model) paletteSections() []paletteSection {
	switch m.pal.step {
	case paletteProjects:
		items := m.projectPaletteItems()
		return []paletteSection{{title: fmt.Sprintf("projects · %d", len(items)), items: items}}
	case paletteStatuses:
		items := m.statusPaletteItems()
		return []paletteSection{{title: fmt.Sprintf("columns · %d", len(items)), items: items}}
	case paletteThemes:
		items := m.themeStepItems()
		return []paletteSection{{title: fmt.Sprintf("themes · %d", len(items)), items: items}}
	case paletteKeys:
		return m.cheatSheetSections()
	}
	var board []paletteItem
	first, skip := paletteFirst, map[boardAction]bool(nil)
	if m.pal.fromDetail {
		first, skip = paletteFirstFromDetail, paletteDetailCovered
	}
	for _, b := range paletteBindings(first, skip) {
		board = append(board, m.boardPaletteItem(b))
	}
	var detail []paletteItem
	if m.pal.fromDetail {
		detail = m.detailPaletteItems()
	}
	if query := strings.TrimSpace(m.pal.input.Value()); query != "" {
		matches := rankPaletteItems(query, append(append(detail, board...), m.themePaletteItem()))
		if len(matches) == 0 {
			return nil
		}
		return []paletteSection{{title: fmt.Sprintf("matches · %d", len(matches)), items: matches}}
	}
	if m.pal.fromDetail {
		return []paletteSection{{title: "this dialog", items: detail}, {title: "board · closes details first", items: board}}
	}
	var recent, rest []paletteItem
	isRecent := map[boardAction]bool{}
	for _, a := range m.pal.recent {
		for _, item := range board {
			if item.action == a {
				recent = append(recent, item)
				isRecent[a] = true
			}
		}
	}
	for _, item := range board {
		if !isRecent[item.action] {
			rest = append(rest, item)
		}
	}
	all := paletteSection{title: fmt.Sprintf("all actions · %d", len(rest)), items: rest}
	if len(recent) == 0 {
		return []paletteSection{all}
	}
	return []paletteSection{{title: "recent", items: recent}, all}
}

func (m *model) paletteItems() []paletteItem {
	var items []paletteItem
	for _, s := range m.paletteSections() {
		items = append(items, s.items...)
	}
	return items
}

// paletteScore ranks a match: 0 for a label or key prefix, 1 for a word
// starting with the query, 2 for the query's letters in order from a word
// start. Matches inside a word (arch in “search”) don't count.
func paletteScore(query string, item paletteItem) (int, bool) {
	q := strings.ToLower(query)
	label := strings.ToLower(item.label)
	keys := strings.ToLower(paletteKeyHint(item.keys))
	compact := strings.ReplaceAll(strings.TrimPrefix(keys, "spc "), " ", "")
	switch {
	case strings.HasPrefix(label, q),
		item.keys != "" && strings.HasPrefix(compact, strings.ReplaceAll(q, " ", "")),
		item.keys != "" && strings.HasPrefix(keys, q):
		return 0, true
	case strings.Contains(" "+label, " "+q):
		return 1, true
	}
	runes := []rune(label)
	first := []rune(q)[0]
	for i, r := range runes {
		if r == first && (i == 0 || runes[i-1] == ' ') && fuzzyMatch(string([]rune(q)[1:]), string(runes[i+1:])) {
			return 2, true
		}
	}
	return 0, false
}

func rankPaletteItems(query string, items []paletteItem) []paletteItem {
	type scored struct {
		item  paletteItem
		score int
	}
	var matches []scored
	for _, item := range items {
		if s, ok := paletteScore(query, item); ok {
			matches = append(matches, scored{item, s})
		}
	}
	sort.SliceStable(matches, func(i, j int) bool { return matches[i].score < matches[j].score })
	out := make([]paletteItem, len(matches))
	for i, s := range matches {
		out[i] = s.item
	}
	return out
}

func (m *model) updatePalette(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	items := m.paletteItems()
	query := strings.TrimSpace(m.pal.input.Value())
	noMatch := m.pal.step == paletteActions && len(items) == 0 && query != ""
	key := msg.String()
	if noMatch && key == "/" {
		key = "enter" // the offered row's own key
	}
	switch key {
	case ":":
		if m.pal.step == paletteKeys && query == "" {
			return m.openPalette(false)
		}
	case "up", "ctrl+p":
		m.pal.cursor = max(0, m.pal.cursor-1)
		return m, nil
	case "down", "ctrl+n":
		m.pal.cursor = max(0, min(m.pal.cursor+1, len(items)-1))
		return m, nil
	case "enter":
		if noMatch {
			m.leavePalette()
			next, cmd := m.openSearch()
			m.searchInput.SetValue(query)
			m.searchInput.CursorEnd()
			return next, cmd
		}
		if m.pal.cursor < len(items) {
			return items[m.pal.cursor].run()
		}
		return m, nil
	case "esc":
		if !m.pal.step.isRoot() {
			m.enterPaletteStep(paletteActions)
			return m, nil
		}
		m.closePalette()
		return m, nil
	case "?":
		if m.pal.step == paletteKeys {
			m.closePalette()
			return m, nil
		}
		m.closePalette()
		if m.mode == modeDetail {
			m.detailHelp = true
		} else {
			return m.openCheatSheet()
		}
		return m, nil
	}
	before := m.pal.input.Value()
	var cmd tea.Cmd
	m.pal.input, cmd = m.pal.input.Update(msg)
	if m.pal.input.Value() != before {
		m.pal.cursor = 0
	}
	return m, cmd
}

// paletteHints are the footer, in the Fixed Order.
func (m *model) paletteHints(noMatch bool) []KeyHint {
	leave := KeyHint{"esc", "close"}
	if m.pal.fromDetail || !m.pal.step.isRoot() {
		leave = KeyHint{"esc", "back"}
	}
	if m.pal.step == paletteKeys {
		return []KeyHint{{"↑↓", "move"}, {"↵", "run"}, leave}
	}
	if noMatch {
		return []KeyHint{{"↵", "search"}, leave, {"backspace", "edit"}, {"?", "more"}}
	}
	run := KeyHint{"↵", "run"}
	switch m.pal.step {
	case paletteProjects:
		run = KeyHint{"↵", "switch"}
	case paletteStatuses:
		run = KeyHint{"↵", "move"}
	case paletteThemes:
		run = KeyHint{"↵", "set"}
	}
	return []KeyHint{{"↑↓", "move"}, run, leave, {"?", "more"}}
}

// paletteContext is the dim label at the right of the input row.
func (m *model) paletteContext(width int) string {
	switch m.pal.step {
	case paletteProjects:
		return "switch project › pick one"
	case paletteStatuses:
		return "set status › pick one"
	case paletteThemes:
		return "theme › pick one"
	case paletteKeys:
		return "keys"
	}
	task := m.selectedTask()
	if task == nil {
		return "board"
	}
	title := singleLine(task.Title)
	if m.pal.fromDetail {
		return truncateCell(title, width-10) + " · actions"
	}
	return "on “" + truncateCell(title, width-5) + "”"
}

type paletteRow struct {
	header string
	item   int // index into the items, or -1
}

// paletteDialog is the 00 Dialog at dialogW × dialogH: the input,
// the scrolling list and the footer, which never moves.
func (m *model) paletteDialog() *termgrid.Grid {
	p := m.palette
	w, h := m.dialogW(), m.dialogH()
	if w < 24 || h < 8 {
		return nil
	}
	g := newDialog(p, w, h)
	width := w - 4

	context := m.paletteContext(width - 16)
	g.Text(w-2-ansi.StringWidth(context), 1, context, tokenFG(p, dusk.TextDim, false))
	inputWidth := width - ansi.StringWidth(context) - 1
	placeholder := map[paletteStep]string{paletteActions: "Type an action…", paletteProjects: "Filter projects…", paletteStatuses: "Filter columns…", paletteThemes: "Filter themes…", paletteKeys: "Filter keys…"}[m.pal.step]
	lines, col := cutInput(m.pal.input.Value(), m.pal.input.Position(), inputWidth)
	drawFocusedInput(g, p, 0, 1, inputWidth+2, 1, lines, IdleInputField, placeholder, &inputPoint{col: col})

	sections := m.paletteSections()
	query := strings.TrimSpace(m.pal.input.Value())
	noMatch := m.pal.step == paletteActions && len(sections) == 0 && query != ""
	if noMatch {
		drawStatusMessage(g, p, 2, 3, width, StatusInfo, "no action matches “"+query+"”")
		drawListRow(g, p, 0, 5, w-2, "search tasks for “"+query+"”", "/", ListRowSelected)
	}
	if m.pal.step == paletteKeys && len(sections) == 0 {
		drawStatusMessage(g, p, 2, 3, width, StatusInfo, "no key matches “"+query+"”")
	}

	var rows []paletteRow
	var items []paletteItem
	for i, s := range sections {
		if i > 0 {
			rows = append(rows, paletteRow{item: -1})
		}
		rows = append(rows, paletteRow{header: s.title, item: -1})
		for _, item := range s.items {
			rows = append(rows, paletteRow{item: len(items)})
			items = append(items, item)
		}
	}
	m.pal.cursor = max(0, min(m.pal.cursor, len(items)-1))

	// From details the list ends a row higher, level with the details body.
	listRows := h - 6
	if m.pal.fromDetail && m.pal.step == paletteActions {
		listRows--
	}
	shown, top := len(rows), 0
	if len(rows) > listRows {
		shown = listRows - 1
		cursorRow := 0
		for i, r := range rows {
			if r.item == m.pal.cursor {
				cursorRow = i
			}
		}
		top = max(0, min(cursorRow-shown+1, len(rows)-shown))
	}
	for i := 0; i < shown && top+i < len(rows); i++ {
		r, y := rows[top+i], 3+i
		switch {
		case r.header != "":
			g.Text(2, y, truncateCell(r.header, width), tokenFG(p, dusk.TextDim, false))
		case r.item >= 0:
			item := items[r.item]
			state := ListRowDefault
			if r.item == m.pal.cursor {
				state = ListRowSelected
			}
			if item.current {
				state = ListRowDisabled
			}
			drawListRow(g, p, 0, y, w-2, item.label, item.hint, state)
		}
	}
	more := 0
	for _, r := range rows[min(top+shown, len(rows)):] {
		if r.item >= 0 {
			more++
		}
	}
	if more > 0 {
		label := fmt.Sprintf("↓ %d more", more)
		g.Text(w-2-ansi.StringWidth(label), 3+shown, label, tokenFG(p, dusk.TextDim, false))
	}
	drawDialogHints(g, p, 2, h-2, width, m.paletteHints(noMatch))
	return g
}
