package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
	"github.com/nikhilsharma/kanban-tui/internal/store"
)

const cardSlotHeight = 6

var theme = struct {
	Rosewater lipgloss.Color
	Flamingo  lipgloss.Color
	Pink      lipgloss.Color
	Mauve     lipgloss.Color
	Red       lipgloss.Color
	Peach     lipgloss.Color
	Yellow    lipgloss.Color
	Green     lipgloss.Color
	Teal      lipgloss.Color
	Blue      lipgloss.Color
	Lavender  lipgloss.Color
	Text      lipgloss.Color
	Subtext1  lipgloss.Color
	Subtext0  lipgloss.Color
	Overlay0  lipgloss.Color
	Surface2  lipgloss.Color
	Surface1  lipgloss.Color
	Surface0  lipgloss.Color
	Base      lipgloss.Color
	Mantle    lipgloss.Color
	Crust     lipgloss.Color
}{
	Rosewater: lipgloss.Color("#F5E0DC"),
	Flamingo:  lipgloss.Color("#F2CDCD"),
	Pink:      lipgloss.Color("#F5C2E7"),
	Mauve:     lipgloss.Color("#CBA6F7"),
	Red:       lipgloss.Color("#F38BA8"),
	Peach:     lipgloss.Color("#FAB387"),
	Yellow:    lipgloss.Color("#F9E2AF"),
	Green:     lipgloss.Color("#A6E3A1"),
	Teal:      lipgloss.Color("#94E2D5"),
	Blue:      lipgloss.Color("#89B4FA"),
	Lavender:  lipgloss.Color("#B4BEFE"),
	Text:      lipgloss.Color("#CDD6F4"),
	Subtext1:  lipgloss.Color("#BAC2DE"),
	Subtext0:  lipgloss.Color("#A6ADC8"),
	Overlay0:  lipgloss.Color("#6C7086"),
	Surface2:  lipgloss.Color("#585B70"),
	Surface1:  lipgloss.Color("#45475A"),
	Surface0:  lipgloss.Color("#313244"),
	Base:      lipgloss.Color("#1E1E2E"),
	Mantle:    lipgloss.Color("#181825"),
	Crust:     lipgloss.Color("#11111B"),
}

type mode int

const (
	modeBoard mode = iota
	modeCreate
	modeSearch
	modeDetail
)

type saveFinishedMsg struct {
	err error
}

type keyMap struct {
	Left        key.Binding
	Right       key.Binding
	Up          key.Binding
	Down        key.Binding
	MoveLeft    key.Binding
	MoveRight   key.Binding
	ReorderUp   key.Binding
	ReorderDown key.Binding
	NewTask     key.Binding
	Search      key.Binding
	Open        key.Binding
	Delete      key.Binding
	Help        key.Binding
	Quit        key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Left, k.Right, k.Up, k.Down, k.NewTask, k.Search, k.Open, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Left, k.Right, k.Up, k.Down},
		{k.MoveLeft, k.MoveRight, k.ReorderUp, k.ReorderDown},
		{k.NewTask, k.Search, k.Open, k.Delete},
		{k.Help, k.Quit},
	}
}

type model struct {
	board        *domain.Board
	store        store.BoardStore
	dataPath     string
	width        int
	height       int
	activeColumn int
	selected     map[domain.Status]int
	scroll       map[domain.Status]int
	visible      map[domain.Status][]string
	filter       string
	filterDraft  string
	mode         mode
	titleInput   textinput.Model
	descInput    textarea.Model
	searchInput  textinput.Model
	help         help.Model
	keys         keyMap
	showHelp     bool
	lastStatus   string
	lastErr      error
}

func New(board *domain.Board, boardStore store.BoardStore, dataPath string) tea.Model {
	titleInput := textinput.New()
	titleInput.Placeholder = "Task title"
	titleInput.CharLimit = 120
	titleInput.Width = 48
	titleInput.PromptStyle = lipgloss.NewStyle().Foreground(theme.Mauve)
	titleInput.TextStyle = lipgloss.NewStyle().Foreground(theme.Text)
	titleInput.PlaceholderStyle = lipgloss.NewStyle().Foreground(theme.Overlay0)

	descInput := textarea.New()
	descInput.Placeholder = "Description"
	descInput.SetWidth(48)
	descInput.SetHeight(6)
	descInput.ShowLineNumbers = false
	descInput.FocusedStyle.Base = lipgloss.NewStyle().Foreground(theme.Text).BorderForeground(theme.Mauve)
	descInput.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(theme.Overlay0)
	descInput.BlurredStyle.Base = lipgloss.NewStyle().Foreground(theme.Subtext1).BorderForeground(theme.Surface1)
	descInput.BlurredStyle.Placeholder = lipgloss.NewStyle().Foreground(theme.Overlay0)

	searchInput := textinput.New()
	searchInput.Placeholder = "Search title or description"
	searchInput.Width = 42
	searchInput.PromptStyle = lipgloss.NewStyle().Foreground(theme.Blue)
	searchInput.TextStyle = lipgloss.NewStyle().Foreground(theme.Text)
	searchInput.PlaceholderStyle = lipgloss.NewStyle().Foreground(theme.Overlay0)

	m := &model{
		board:    board,
		store:    boardStore,
		dataPath: dataPath,
		selected: map[domain.Status]int{
			domain.StatusBacklog:    0,
			domain.StatusInProgress: 0,
			domain.StatusDone:       0,
		},
		scroll: map[domain.Status]int{
			domain.StatusBacklog:    0,
			domain.StatusInProgress: 0,
			domain.StatusDone:       0,
		},
		visible: map[domain.Status][]string{
			domain.StatusBacklog:    {},
			domain.StatusInProgress: {},
			domain.StatusDone:       {},
		},
		titleInput:  titleInput,
		descInput:   descInput,
		searchInput: searchInput,
		help:        help.New(),
		showHelp:    true,
		keys: keyMap{
			Left:        key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("h/left", "column left")),
			Right:       key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("l/right", "column right")),
			Up:          key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("k/up", "previous task")),
			Down:        key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("j/down", "next task")),
			MoveLeft:    key.NewBinding(key.WithKeys("["), key.WithHelp("[", "move task left")),
			MoveRight:   key.NewBinding(key.WithKeys("]"), key.WithHelp("]", "move task right")),
			ReorderUp:   key.NewBinding(key.WithKeys("K"), key.WithHelp("K", "reorder up")),
			ReorderDown: key.NewBinding(key.WithKeys("J"), key.WithHelp("J", "reorder down")),
			NewTask:     key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new task")),
			Search:      key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
			Open:        key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "details")),
			Delete:      key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "delete")),
			Help:        key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "toggle help")),
			Quit:        key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		},
	}

	m.recalculateVisible()
	return m
}

func (m *model) Init() tea.Cmd {
	return textinput.Blink
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.syncAllScroll()
		return m, nil
	case saveFinishedMsg:
		m.lastErr = msg.err
		if msg.err != nil {
			m.lastStatus = "save failed"
		} else if m.lastStatus == "" {
			m.lastStatus = "saved"
		}
		return m, nil
	case tea.KeyMsg:
		switch m.mode {
		case modeCreate:
			return m.updateCreate(msg)
		case modeSearch:
			return m.updateSearch(msg)
		case modeDetail:
			return m.updateDetail(msg)
		default:
			return m.updateBoard(msg)
		}
	}

	return m, nil
}

func (m *model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	header := m.renderHeader()
	board := m.renderBoard()
	footer := m.renderFooter()
	view := lipgloss.JoinVertical(lipgloss.Left, header, board, footer)

	switch m.mode {
	case modeCreate:
		return placeOverlay(view, m.width, m.height, m.renderCreateDialog())
	case modeSearch:
		return placeOverlay(view, m.width, m.height, m.renderSearchDialog())
	case modeDetail:
		return placeOverlay(view, m.width, m.height, m.renderDetailDialog())
	default:
		return view
	}
}

func (m *model) updateBoard(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Left):
		if m.activeColumn > 0 {
			m.activeColumn--
		}
		m.syncScroll(domain.StatusOrder[m.activeColumn])
	case key.Matches(msg, m.keys.Right):
		if m.activeColumn < len(domain.StatusOrder)-1 {
			m.activeColumn++
		}
		m.syncScroll(domain.StatusOrder[m.activeColumn])
	case key.Matches(msg, m.keys.Up):
		m.moveSelection(-1)
	case key.Matches(msg, m.keys.Down):
		m.moveSelection(1)
	case key.Matches(msg, m.keys.MoveLeft):
		return m.shiftSelected(-1)
	case key.Matches(msg, m.keys.MoveRight):
		return m.shiftSelected(1)
	case key.Matches(msg, m.keys.ReorderUp):
		return m.reorderSelected(-1)
	case key.Matches(msg, m.keys.ReorderDown):
		return m.reorderSelected(1)
	case key.Matches(msg, m.keys.NewTask):
		m.mode = modeCreate
		m.titleInput.SetValue("")
		m.descInput.SetValue("")
		m.titleInput.Focus()
		m.descInput.Blur()
		m.lastErr = nil
		return m, textinput.Blink
	case key.Matches(msg, m.keys.Search):
		m.mode = modeSearch
		m.filterDraft = m.filter
		m.searchInput.SetValue(m.filter)
		m.searchInput.CursorEnd()
		m.searchInput.Focus()
		return m, textinput.Blink
	case key.Matches(msg, m.keys.Open):
		if m.selectedTask() != nil {
			m.mode = modeDetail
		}
	case key.Matches(msg, m.keys.Delete):
		return m.deleteSelected()
	case key.Matches(msg, m.keys.Help):
		m.showHelp = !m.showHelp
	}

	return m, nil
}

func (m *model) updateCreate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeBoard
		m.titleInput.Blur()
		m.descInput.Blur()
		return m, nil
	case "tab", "shift+tab":
		if m.titleInput.Focused() {
			m.titleInput.Blur()
			m.descInput.Focus()
		} else {
			m.descInput.Blur()
			m.titleInput.Focus()
		}
		return m, nil
	case "ctrl+s":
		return m.createTask()
	}

	var cmd tea.Cmd
	if m.titleInput.Focused() {
		m.titleInput, cmd = m.titleInput.Update(msg)
		return m, cmd
	}

	m.descInput, cmd = m.descInput.Update(msg)
	return m, cmd
}

func (m *model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.filter = m.filterDraft
		m.mode = modeBoard
		m.searchInput.Blur()
		m.recalculateVisible()
		return m, nil
	case "enter":
		m.filter = strings.TrimSpace(m.searchInput.Value())
		m.mode = modeBoard
		m.searchInput.Blur()
		m.recalculateVisible()
		return m, nil
	}

	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	m.filter = strings.TrimSpace(m.searchInput.Value())
	m.recalculateVisible()
	return m, cmd
}

func (m *model) updateDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter", "q":
		m.mode = modeBoard
	}

	return m, nil
}

func (m *model) createTask() (tea.Model, tea.Cmd) {
	task, err := m.board.AddTask(m.titleInput.Value(), m.descInput.Value())
	if err != nil {
		m.lastErr = err
		return m, nil
	}

	m.mode = modeBoard
	m.lastStatus = fmt.Sprintf("created %s", task.ID)
	m.lastErr = nil
	m.filter = ""
	m.searchInput.SetValue("")
	m.activeColumn = 0
	m.recalculateVisible()
	m.selected[task.Status] = len(m.visible[task.Status]) - 1
	m.syncScroll(task.Status)

	return m, saveBoardCmd(m.store, m.board.Clone())
}

func (m *model) shiftSelected(delta int) (tea.Model, tea.Cmd) {
	task := m.selectedTask()
	if task == nil {
		return m, nil
	}
	if !m.board.ShiftTask(task.ID, delta) {
		return m, nil
	}

	m.lastStatus = fmt.Sprintf("moved %s to %s", task.ID, task.Status.Title())
	m.recalculateVisible()
	m.selectTask(task.ID)
	return m, saveBoardCmd(m.store, m.board.Clone())
}

func (m *model) reorderSelected(delta int) (tea.Model, tea.Cmd) {
	if m.filter != "" {
		m.lastStatus = "clear search before reordering"
		return m, nil
	}

	status := domain.StatusOrder[m.activeColumn]
	index := m.selected[status]
	target := index + delta
	if !m.board.MoveWithin(status, index, target) {
		return m, nil
	}

	m.selected[status] = target
	m.lastStatus = "task reordered"
	m.recalculateVisible()
	m.syncScroll(status)
	return m, saveBoardCmd(m.store, m.board.Clone())
}

func (m *model) deleteSelected() (tea.Model, tea.Cmd) {
	task := m.selectedTask()
	if task == nil {
		return m, nil
	}

	if !m.board.DeleteTask(task.ID) {
		return m, nil
	}

	m.lastStatus = fmt.Sprintf("deleted %s", task.ID)
	m.lastErr = nil
	m.recalculateVisible()
	return m, saveBoardCmd(m.store, m.board.Clone())
}

func (m *model) moveSelection(delta int) {
	status := domain.StatusOrder[m.activeColumn]
	visible := m.visible[status]
	if len(visible) == 0 {
		m.selected[status] = 0
		m.scroll[status] = 0
		return
	}

	next := m.selected[status] + delta
	if next < 0 {
		next = 0
	}
	if next >= len(visible) {
		next = len(visible) - 1
	}
	m.selected[status] = next
	m.syncScroll(status)
}

func (m *model) recalculateVisible() {
	query := strings.ToLower(strings.TrimSpace(m.filter))
	for _, status := range domain.StatusOrder {
		tasks := m.board.Order[status]
		visible := make([]string, 0, len(tasks))
		for _, id := range tasks {
			task := m.board.Tasks[id]
			if task == nil {
				continue
			}
			if query == "" || strings.Contains(task.SearchText(), query) {
				visible = append(visible, id)
			}
		}
		m.visible[status] = visible
		if len(visible) == 0 {
			m.selected[status] = 0
			m.scroll[status] = 0
			continue
		}
		if m.selected[status] >= len(visible) {
			m.selected[status] = len(visible) - 1
		}
		m.syncScroll(status)
	}
}

func (m *model) syncAllScroll() {
	for _, status := range domain.StatusOrder {
		m.syncScroll(status)
	}
}

func (m *model) syncScroll(status domain.Status) {
	rows := m.taskRows()
	if rows <= 0 {
		rows = 1
	}

	selected := m.selected[status]
	scroll := m.scroll[status]
	if selected < scroll {
		scroll = selected
	}
	if selected >= scroll+rows {
		scroll = selected - rows + 1
	}
	if scroll < 0 {
		scroll = 0
	}

	maxScroll := max(0, len(m.visible[status])-rows)
	if scroll > maxScroll {
		scroll = maxScroll
	}
	m.scroll[status] = scroll
}

func (m *model) selectTask(id string) {
	for columnIndex, status := range domain.StatusOrder {
		for i, candidate := range m.visible[status] {
			if candidate != id {
				continue
			}
			m.activeColumn = columnIndex
			m.selected[status] = i
			m.syncScroll(status)
			return
		}
	}
}

func (m *model) selectedTask() *domain.Task {
	status := domain.StatusOrder[m.activeColumn]
	visible := m.visible[status]
	if len(visible) == 0 {
		return nil
	}

	index := m.selected[status]
	if index < 0 || index >= len(visible) {
		return nil
	}

	return m.board.Tasks[visible[index]]
}

func (m *model) renderHeader() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Text).
		Render("kanban-tui")

	subtitle := lipgloss.NewStyle().
		Foreground(theme.Subtext0).
		Render("fast keyboard-first task board")

	filter := "filter: all"
	if m.filter != "" {
		filter = "filter: " + m.filter
	}

	statusText := ""
	switch {
	case m.lastErr != nil:
		statusText = lipgloss.NewStyle().Foreground(theme.Red).Render(m.lastErr.Error())
	case m.lastStatus != "":
		statusText = lipgloss.NewStyle().Foreground(theme.Green).Render(m.lastStatus)
	}

	left := lipgloss.JoinVertical(lipgloss.Left, title, subtitle)
	right := lipgloss.JoinVertical(
		lipgloss.Right,
		lipgloss.NewStyle().Foreground(theme.Subtext1).Render(filter),
		lipgloss.NewStyle().Foreground(theme.Subtext0).Render(statusText),
	)

	style := lipgloss.NewStyle().
		Width(m.width).
		Padding(1, 2).
		Background(theme.Mantle).
		Foreground(theme.Text)

	return style.Render(lipgloss.JoinHorizontal(lipgloss.Top, left, spacer(max(2, m.width-lipgloss.Width(left)-lipgloss.Width(right)-8)), right))
}

func (m *model) renderBoard() string {
	gap := 2
	columnWidth := max(24, (m.width-8-(gap*2))/3)
	columns := make([]string, 0, len(domain.StatusOrder))

	for i, status := range domain.StatusOrder {
		columns = append(columns, m.renderColumn(status, i == m.activeColumn, columnWidth))
	}

	return lipgloss.NewStyle().
		Padding(1, 2).
		Render(joinHorizontal(columns, gap))
}

func (m *model) renderColumn(status domain.Status, active bool, width int) string {
	ids := m.visible[status]
	columnStyle := lipgloss.NewStyle().
		Width(width).
		Height(max(12, m.height-11)).
		Padding(1).
		Border(lipgloss.NormalBorder()).
		BorderForeground(theme.Surface1).
		Background(theme.Crust)

	if active {
		columnStyle = columnStyle.BorderForeground(statusAccent(status))
	}

	label := lipgloss.NewStyle().Bold(true).Foreground(statusAccent(status)).Render(status.Title())
	count := lipgloss.NewStyle().Foreground(theme.Subtext0).Render(fmt.Sprintf("%d tasks", len(ids)))
	header := lipgloss.JoinHorizontal(lipgloss.Center, label, " ", count)

	bodyHeight := max(6, m.height-17)
	scroll := m.scroll[status]
	rows := m.taskRows()
	end := min(len(ids), scroll+rows)

	body := make([]string, 0, rows)
	if len(ids) == 0 {
		body = append(body, lipgloss.NewStyle().Foreground(theme.Overlay0).Render("No tasks"))
	}

	for i := scroll; i < end; i++ {
		task := m.board.Tasks[ids[i]]
		if task == nil {
			continue
		}
		body = append(body, m.renderTaskCard(task, width-4, active && i == m.selected[status]))
	}

	if hidden := len(ids) - end; hidden > 0 {
		body = append(body, lipgloss.NewStyle().Foreground(theme.Overlay0).Render(fmt.Sprintf("+%d more", hidden)))
	}

	bodyView := lipgloss.NewStyle().Height(bodyHeight).Render(strings.Join(body, "\n"))
	return columnStyle.Render(lipgloss.JoinVertical(lipgloss.Left, header, "", bodyView))
}

func (m *model) renderTaskCard(task *domain.Task, width int, selected bool) string {
	title := truncate(task.Title, width-4)
	desc := truncate(singleLine(task.Description), width-4)
	if desc == "" {
		desc = "No description"
	}

	meta := fmt.Sprintf("%s  %s", shortID(task.ID), task.UpdatedAt.Local().Format("02 Jan 15:04"))
	card := lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Render(title),
		lipgloss.NewStyle().Foreground(theme.Subtext0).Render(meta),
		lipgloss.NewStyle().Foreground(theme.Subtext1).Render(desc),
	)

	style := lipgloss.NewStyle().
		Width(width).
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Surface1).
		Background(theme.Base).
		Foreground(theme.Text)

	if selected {
		style = style.
			BorderForeground(theme.Mauve).
			Background(theme.Surface0)
	}

	return style.Render(card)
}

func (m *model) renderFooter() string {
	info := lipgloss.NewStyle().Foreground(theme.Overlay0).Render("data: " + m.dataPath)
	helpView := ""
	if m.showHelp {
		helpView = m.help.View(m.keys)
	} else {
		helpView = "press ? for help"
	}

	return lipgloss.NewStyle().
		Width(m.width).
		Padding(0, 2, 1, 2).
		Foreground(theme.Subtext1).
		Render(lipgloss.JoinVertical(lipgloss.Left, helpView, info))
}

func (m *model) renderCreateDialog() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(theme.Text).Render("New task")
	hint := lipgloss.NewStyle().Foreground(theme.Subtext0).Render("tab switches fields, ctrl+s saves, esc cancels")
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		m.titleInput.View(),
		"",
		m.descInput.View(),
		"",
		hint,
	)

	return lipgloss.NewStyle().
		Width(56).
		Padding(1, 2).
		Border(lipgloss.ThickBorder()).
		BorderForeground(theme.Mauve).
		Background(theme.Base).
		Render(content)
}

func (m *model) renderSearchDialog() string {
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Foreground(theme.Text).Render("Search"),
		"",
		m.searchInput.View(),
		"",
		lipgloss.NewStyle().Foreground(theme.Subtext0).Render("type to filter, enter applies, esc restores"),
	)

	return lipgloss.NewStyle().
		Width(50).
		Padding(1, 2).
		Border(lipgloss.ThickBorder()).
		BorderForeground(theme.Blue).
		Background(theme.Base).
		Render(content)
}

func (m *model) renderDetailDialog() string {
	task := m.selectedTask()
	if task == nil {
		return ""
	}

	description := task.Description
	if strings.TrimSpace(description) == "" {
		description = "No description"
	}

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Foreground(theme.Text).Render(task.Title),
		"",
		lipgloss.NewStyle().Foreground(theme.Subtext0).Render("ID: "+task.ID),
		lipgloss.NewStyle().Foreground(theme.Subtext0).Render("Status: "+task.Status.Title()),
		lipgloss.NewStyle().Foreground(theme.Subtext0).Render("Created: "+task.CreatedAt.Local().Format(time.RFC822)),
		lipgloss.NewStyle().Foreground(theme.Subtext0).Render("Updated: "+task.UpdatedAt.Local().Format(time.RFC822)),
		"",
		lipgloss.NewStyle().Width(64).Foreground(theme.Subtext1).Render(description),
		"",
		lipgloss.NewStyle().Foreground(theme.Subtext0).Render("esc closes"),
	)

	return lipgloss.NewStyle().
		Width(72).
		Padding(1, 2).
		Border(lipgloss.ThickBorder()).
		BorderForeground(statusAccent(task.Status)).
		Background(theme.Base).
		Render(content)
}

func (m *model) taskRows() int {
	height := max(1, m.height-18)
	rows := height / cardSlotHeight
	if rows < 1 {
		return 1
	}
	return rows
}

func saveBoardCmd(boardStore store.BoardStore, board *domain.Board) tea.Cmd {
	return func() tea.Msg {
		return saveFinishedMsg{err: boardStore.Save(board)}
	}
}

func placeOverlay(base string, width, height int, overlay string) string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		base,
		"",
		lipgloss.Place(width, 0, lipgloss.Center, lipgloss.Center, overlay),
	)
}

func shortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

func truncate(value string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	if width <= 1 {
		return string(runes[:width])
	}
	return string(runes[:width-1]) + "…"
}

func singleLine(value string) string {
	fields := strings.Fields(value)
	return strings.Join(fields, " ")
}

func spacer(width int) string {
	if width <= 0 {
		return ""
	}
	return strings.Repeat(" ", width)
}

func joinHorizontal(parts []string, gap int) string {
	if len(parts) == 0 {
		return ""
	}

	withGaps := make([]string, 0, len(parts)*2-1)
	for i, part := range parts {
		if i > 0 {
			withGaps = append(withGaps, spacer(gap))
		}
		withGaps = append(withGaps, part)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, withGaps...)
}

func statusAccent(status domain.Status) lipgloss.Color {
	switch status {
	case domain.StatusBacklog:
		return theme.Blue
	case domain.StatusInProgress:
		return theme.Peach
	case domain.StatusDone:
		return theme.Green
	default:
		return theme.Lavender
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
