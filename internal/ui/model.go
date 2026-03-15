package ui

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
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

const cardSlotHeight = 4

// leftAccentBorder defines a border with only a left accent bar for modern card styling.
var leftAccentBorder = lipgloss.Border{
	Left: "┃",
}

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
	modeAddColumn
	modeRenameColumn
)

type saveFinishedMsg struct {
	err error
}

type editorFinishedMsg struct {
	err  error
	path string
}

type keyMap struct {
	Left         key.Binding
	Right        key.Binding
	Up           key.Binding
	Down         key.Binding
	MoveLeft     key.Binding
	MoveRight    key.Binding
	ReorderUp    key.Binding
	ReorderDown  key.Binding
	MoveColLeft  key.Binding
	MoveColRight key.Binding
	RenameCol    key.Binding
	DeleteCol    key.Binding
	NewTask      key.Binding
	NewColumn    key.Binding
	Search       key.Binding
	Open         key.Binding
	Delete       key.Binding
	Help         key.Binding
	Quit         key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Left, k.Right, k.Up, k.Down, k.NewTask, k.NewColumn, k.RenameCol, k.DeleteCol, k.Search, k.Open, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Left, k.Right, k.Up, k.Down},
		{k.MoveColLeft, k.MoveColRight, k.MoveLeft, k.MoveRight},
		{k.ReorderUp, k.ReorderDown, k.NewTask, k.NewColumn, k.RenameCol, k.DeleteCol},
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
	columnInput  textinput.Model
	mode         mode
	columnRename domain.Status
	titleInput   textinput.Model
	descInput    textarea.Model
	searchInput  textinput.Model
	help         help.Model
	keys         keyMap
	showHelp     bool
	lastStatus   string
	lastErr      error
}

// ansiStripRe matches ANSI escape sequences for the dim/blur effect.
var ansiStripRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func New(board *domain.Board, boardStore store.BoardStore, dataPath string) tea.Model {
	titleInput := textinput.New()
	titleInput.Prompt = ""
	titleInput.Placeholder = "What needs to be done?"
	titleInput.CharLimit = 120
	titleInput.Width = 84
	titleInput.TextStyle = lipgloss.NewStyle().Foreground(theme.Text)
	titleInput.PlaceholderStyle = lipgloss.NewStyle().Foreground(theme.Overlay0)

	descInput := textarea.New()
	descInput.Placeholder = "Add details (optional)"
	descInput.SetWidth(84)
	descInput.SetHeight(12)
	descInput.ShowLineNumbers = false
	descInput.FocusedStyle.Base = lipgloss.NewStyle().Foreground(theme.Text).BorderForeground(theme.Mauve)
	descInput.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(theme.Overlay0)
	descInput.FocusedStyle.CursorLine = lipgloss.NewStyle()
	descInput.BlurredStyle.Base = lipgloss.NewStyle().Foreground(theme.Subtext1).BorderForeground(theme.Surface1)
	descInput.BlurredStyle.Placeholder = lipgloss.NewStyle().Foreground(theme.Overlay0)

	searchInput := textinput.New()
	searchInput.Prompt = ""
	searchInput.Placeholder = "Type to filter tasks..."
	searchInput.Width = 42
	searchInput.TextStyle = lipgloss.NewStyle().Foreground(theme.Text)
	searchInput.PlaceholderStyle = lipgloss.NewStyle().Foreground(theme.Overlay0)

	columnInput := textinput.New()
	columnInput.Placeholder = "Column name"
	columnInput.Width = 42
	columnInput.TextStyle = lipgloss.NewStyle().Foreground(theme.Text)
	columnInput.PlaceholderStyle = lipgloss.NewStyle().Foreground(theme.Overlay0)

	columns := board.Statuses()
	selected := make(map[domain.Status]int, len(columns))
	scroll := make(map[domain.Status]int, len(columns))
	visible := make(map[domain.Status][]string, len(columns))
	for _, status := range columns {
		selected[status] = 0
		scroll[status] = 0
		visible[status] = []string{}
	}

	m := &model{
		board:       board,
		store:       boardStore,
		dataPath:    dataPath,
		selected:    selected,
		scroll:      scroll,
		visible:     visible,
		titleInput:  titleInput,
		descInput:   descInput,
		searchInput: searchInput,
		columnInput: columnInput,
		help:        help.New(),
		showHelp:    true,
		keys: keyMap{
			Left:         key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("h/\u2190", "column left")),
			Right:        key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("l/\u2192", "column right")),
			Up:           key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("k/\u2191", "prev task")),
			Down:         key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("j/\u2193", "next task")),
			MoveLeft:     key.NewBinding(key.WithKeys("["), key.WithHelp("[", "move left")),
			MoveRight:    key.NewBinding(key.WithKeys("]"), key.WithHelp("]", "move right")),
			ReorderUp:    key.NewBinding(key.WithKeys("K"), key.WithHelp("K", "reorder up")),
			ReorderDown:  key.NewBinding(key.WithKeys("J"), key.WithHelp("J", "reorder down")),
			NewTask:      key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new task")),
			NewColumn:    key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "new column")),
			MoveColLeft:  key.NewBinding(key.WithKeys("H"), key.WithHelp("H", "move column left")),
			MoveColRight: key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "move column right")),
			RenameCol:    key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "rename column")),
			DeleteCol:    key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete column")),
			Search:       key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
			Open:         key.NewBinding(key.WithKeys("enter"), key.WithHelp("\u23ce", "details")),
			Delete:       key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "delete")),
			Help:         key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "toggle help")),
			Quit:         key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
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
	case editorFinishedMsg:
		return m.handleEditorResult(msg)
	case tea.KeyMsg:
		switch m.mode {
		case modeCreate:
			return m.updateCreate(msg)
		case modeSearch:
			return m.updateSearch(msg)
		case modeDetail:
			return m.updateDetail(msg)
		case modeAddColumn:
			return m.updateColumnDialog(msg)
		case modeRenameColumn:
			return m.updateColumnDialog(msg)
		default:
			return m.updateBoard(msg)
		}
	}

	return m, nil
}

func (m *model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	header := m.renderHeader()
	board := m.renderBoard()
	footer := m.renderFooter()
	view := lipgloss.JoinVertical(lipgloss.Left, header, board, footer)

	switch m.mode {
	case modeCreate:
		return m.placeOverlayCenter(view, m.renderCreateDialog())
	case modeSearch:
		return m.placeOverlayCenter(view, m.renderSearchDialog())
	case modeDetail:
		return m.placeOverlayCenter(view, m.renderDetailDialog())
	case modeAddColumn:
		return m.placeOverlayCenter(view, m.renderAddColumnDialog())
	case modeRenameColumn:
		return m.placeOverlayCenter(view, m.renderAddColumnDialog())
	default:
		return view
	}
}

func (m *model) updateBoard(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	statuses := m.board.Statuses()
	if len(statuses) == 0 {
		return m, nil
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Left):
		if m.activeColumn > 0 {
			m.activeColumn--
		}
		m.syncScroll(statuses[m.activeColumn])
	case key.Matches(msg, m.keys.Right):
		if m.activeColumn < len(statuses)-1 {
			m.activeColumn++
		}
		m.syncScroll(statuses[m.activeColumn])
	case key.Matches(msg, m.keys.Up):
		m.moveSelection(-1)
	case key.Matches(msg, m.keys.Down):
		m.moveSelection(1)
	case key.Matches(msg, m.keys.MoveLeft):
		return m.shiftSelected(-1)
	case key.Matches(msg, m.keys.MoveRight):
		return m.shiftSelected(1)
	case key.Matches(msg, m.keys.MoveColLeft):
		return m.moveColumn(-1)
	case key.Matches(msg, m.keys.MoveColRight):
		return m.moveColumn(1)
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
	case key.Matches(msg, m.keys.NewColumn):
		m.mode = modeAddColumn
		m.columnInput.SetValue("")
		m.columnInput.Focus()
		m.lastErr = nil
		return m, textinput.Blink
	case key.Matches(msg, m.keys.RenameCol):
		return m.beginRenameColumn()
	case key.Matches(msg, m.keys.DeleteCol):
		return m.deleteColumn()
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

func (m *model) openEditorWithDraft() (tea.Model, tea.Cmd) {
	tmpFile, err := os.CreateTemp("", "kanban-*.md")
	if err != nil {
		m.lastErr = fmt.Errorf("create temp file: %w", err)
		return m, nil
	}

	// Carry over any draft content from the dialog
	draftTitle := strings.TrimSpace(m.titleInput.Value())
	draftDesc := strings.TrimSpace(m.descInput.Value())

	var content string
	if draftTitle != "" || draftDesc != "" {
		content = draftTitle + "\n\n" + draftDesc + "\n"
	}
	content += "\n# ─── kanban-tui ──────────────────────────────\n"
	content += "# First line = task title\n"
	content += "# Everything after = description\n"
	content += "# Lines starting with # are ignored\n"
	content += "# Save and quit to create, empty to cancel\n"

	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		m.lastErr = fmt.Errorf("write template: %w", err)
		return m, nil
	}
	tmpFile.Close()

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nvim"
	}

	path := tmpFile.Name()
	c := exec.Command(editor, path)
	return m, tea.ExecProcess(c, func(err error) tea.Msg {
		return editorFinishedMsg{err: err, path: path}
	})
}

func (m *model) handleEditorResult(msg editorFinishedMsg) (tea.Model, tea.Cmd) {
	defer os.Remove(msg.path)

	if msg.err != nil {
		m.lastErr = fmt.Errorf("editor: %w", msg.err)
		return m, nil
	}

	content, err := os.ReadFile(msg.path)
	if err != nil {
		m.lastErr = fmt.Errorf("read file: %w", err)
		return m, nil
	}

	title, description := parseEditorContent(string(content))
	if title == "" {
		m.lastStatus = "task creation cancelled"
		m.lastErr = nil
		return m, nil
	}

	task, err := m.board.AddTask(title, description)
	if err != nil {
		m.lastErr = err
		return m, nil
	}

	m.lastStatus = fmt.Sprintf("created %s", shortID(task.ID))
	m.lastErr = nil
	m.filter = ""
	m.searchInput.SetValue("")
	m.activeColumn = 0
	m.recalculateVisible()
	m.selected[task.Status] = len(m.visible[task.Status]) - 1
	m.syncScroll(task.Status)

	return m, saveBoardCmd(m.store, m.board.Clone())
}

func parseEditorContent(content string) (title, description string) {
	lines := strings.Split(content, "\n")

	var nonComment []string
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		nonComment = append(nonComment, line)
	}

	// First non-empty line is the title
	titleFound := false
	var descLines []string
	for _, line := range nonComment {
		if !titleFound {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				title = trimmed
				titleFound = true
			}
			continue
		}
		descLines = append(descLines, line)
	}

	description = strings.TrimSpace(strings.Join(descLines, "\n"))
	return title, description
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
	case "ctrl+e":
		m.mode = modeBoard
		m.titleInput.Blur()
		m.descInput.Blur()
		return m.openEditorWithDraft()
	}

	var cmd tea.Cmd
	if m.titleInput.Focused() {
		m.titleInput, cmd = m.titleInput.Update(msg)
		return m, cmd
	}

	m.descInput, cmd = m.descInput.Update(msg)
	return m, cmd
}

func (m *model) createTask() (tea.Model, tea.Cmd) {
	task, err := m.board.AddTask(m.titleInput.Value(), m.descInput.Value())
	if err != nil {
		m.lastErr = err
		return m, nil
	}

	m.mode = modeBoard
	m.lastStatus = fmt.Sprintf("created %s", shortID(task.ID))
	m.lastErr = nil
	m.filter = ""
	m.searchInput.SetValue("")
	m.activeColumn = 0
	m.recalculateVisible()
	m.selected[task.Status] = len(m.visible[task.Status]) - 1
	m.syncScroll(task.Status)

	return m, saveBoardCmd(m.store, m.board.Clone())
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

func (m *model) updateColumnDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeBoard
		m.columnInput.Blur()
		m.lastErr = nil
		m.columnRename = ""
		return m, nil
	case "enter":
		if m.mode == modeRenameColumn {
			if m.columnRename == "" {
				m.lastErr = fmt.Errorf("column target missing")
				return m, nil
			}

			renamed, err := m.board.RenameColumn(string(m.columnRename), m.columnInput.Value())
			if err != nil {
				m.lastErr = err
				return m, nil
			}

			m.mode = modeBoard
			m.columnInput.Blur()
			m.lastErr = nil
			m.lastStatus = fmt.Sprintf("renamed %s", renamed.Title())
			m.columnRename = ""
			m.activeColumn = m.board.StatusIndex(renamed)
			m.ensureColumnState()
			m.recalculateVisible()
			m.syncAllScroll()
			return m, saveBoardCmd(m.store, m.board.Clone())
		}

		status, err := m.board.AddColumn(m.columnInput.Value())
		if err != nil {
			m.lastErr = err
			return m, nil
		}

		m.mode = modeBoard
		m.columnInput.Blur()
		m.lastErr = nil
		m.lastStatus = fmt.Sprintf("added %s", status.Title())
		m.ensureColumnState()
		m.activeColumn = m.board.StatusIndex(status)
		m.recalculateVisible()
		m.syncAllScroll()

		return m, saveBoardCmd(m.store, m.board.Clone())
	}

	var cmd tea.Cmd
	m.columnInput, cmd = m.columnInput.Update(msg)
	return m, cmd
}

func (m *model) moveColumn(delta int) (tea.Model, tea.Cmd) {
	statuses := m.board.Statuses()
	if len(statuses) == 0 {
		return m, nil
	}

	target := m.activeColumn + delta
	if !m.board.MoveColumn(m.activeColumn, target) {
		return m, nil
	}

	m.activeColumn = target
	m.lastStatus = fmt.Sprintf("moved column %s", m.board.Columns[m.activeColumn].Title())
	m.lastErr = nil
	m.ensureColumnState()
	m.recalculateVisible()
	m.syncAllScroll()

	return m, saveBoardCmd(m.store, m.board.Clone())
}

func (m *model) beginRenameColumn() (tea.Model, tea.Cmd) {
	statuses := m.board.Statuses()
	if len(statuses) == 0 {
		return m, nil
	}
	if m.activeColumn < 0 || m.activeColumn >= len(statuses) {
		return m, nil
	}

	status := statuses[m.activeColumn]
	m.columnRename = status
	m.mode = modeRenameColumn
	m.columnInput.SetValue(string(status))
	m.columnInput.Focus()
	m.lastErr = nil

	return m, textinput.Blink
}

func (m *model) deleteColumn() (tea.Model, tea.Cmd) {
	statuses := m.board.Statuses()
	if len(statuses) == 0 || m.activeColumn >= len(statuses) {
		return m, nil
	}

	status := statuses[m.activeColumn]
	if err := m.board.DeleteColumn(status); err != nil {
		m.lastErr = err
		return m, nil
	}

	m.lastErr = nil
	m.lastStatus = fmt.Sprintf("deleted %s", status.Title())
	if m.activeColumn >= len(m.board.Columns) {
		m.activeColumn = len(m.board.Columns) - 1
	}
	if m.activeColumn < 0 {
		m.activeColumn = 0
	}

	m.ensureColumnState()
	m.recalculateVisible()
	m.syncAllScroll()

	return m, saveBoardCmd(m.store, m.board.Clone())
}

func (m *model) updateDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter", "q":
		m.mode = modeBoard
	}

	return m, nil
}

func (m *model) shiftSelected(delta int) (tea.Model, tea.Cmd) {
	task := m.selectedTask()
	if task == nil {
		return m, nil
	}
	if !m.board.ShiftTask(task.ID, delta) {
		return m, nil
	}

	m.lastStatus = fmt.Sprintf("moved %s \u2192 %s", shortID(task.ID), task.Status.Title())
	m.recalculateVisible()
	m.selectTask(task.ID)
	return m, saveBoardCmd(m.store, m.board.Clone())
}

func (m *model) reorderSelected(delta int) (tea.Model, tea.Cmd) {
	if m.filter != "" {
		m.lastStatus = "clear search before reordering"
		return m, nil
	}

	statuses := m.board.Statuses()
	if len(statuses) == 0 || m.activeColumn >= len(statuses) {
		return m, nil
	}
	status := statuses[m.activeColumn]
	index := m.selected[status]
	target := index + delta
	if !m.board.MoveWithin(status, index, target) {
		return m, nil
	}

	m.selected[status] = target
	m.lastStatus = "reordered"
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

	m.lastStatus = fmt.Sprintf("deleted %s", shortID(task.ID))
	m.lastErr = nil
	m.recalculateVisible()
	return m, saveBoardCmd(m.store, m.board.Clone())
}

func (m *model) moveSelection(delta int) {
	statuses := m.board.Statuses()
	if len(statuses) == 0 || m.activeColumn >= len(statuses) {
		m.activeColumn = 0
		return
	}

	status := statuses[m.activeColumn]
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
	statusList := m.board.Statuses()
	for _, status := range statusList {
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
	m.ensureColumnState()
	for _, status := range m.board.Statuses() {
		m.syncScroll(status)
	}
}

func (m *model) ensureColumnState() {
	if m.selected == nil {
		m.selected = make(map[domain.Status]int)
	}
	if m.scroll == nil {
		m.scroll = make(map[domain.Status]int)
	}
	if m.visible == nil {
		m.visible = make(map[domain.Status][]string)
	}

	has := map[domain.Status]struct{}{}
	for _, status := range m.board.Statuses() {
		has[status] = struct{}{}
		if _, ok := m.selected[status]; !ok {
			m.selected[status] = 0
		}
		if _, ok := m.scroll[status]; !ok {
			m.scroll[status] = 0
		}
		if _, ok := m.visible[status]; !ok {
			m.visible[status] = []string{}
		}
	}

	for status := range m.selected {
		if _, ok := has[status]; !ok {
			delete(m.selected, status)
		}
	}
	for status := range m.scroll {
		if _, ok := has[status]; !ok {
			delete(m.scroll, status)
		}
	}
	for status := range m.visible {
		if _, ok := has[status]; !ok {
			delete(m.visible, status)
		}
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
	for columnIndex, status := range m.board.Statuses() {
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
	if len(m.board.Statuses()) == 0 {
		return nil
	}
	if m.activeColumn < 0 || m.activeColumn >= len(m.board.Statuses()) {
		m.activeColumn = 0
	}

	status := m.board.Statuses()[m.activeColumn]
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

// ─── Header ──────────────────────────────────────────────────────────────────

func (m *model) renderHeader() string {
	logo := lipgloss.NewStyle().Bold(true).Foreground(theme.Mauve).Render("\u25c6")
	title := lipgloss.NewStyle().Bold(true).Foreground(theme.Text).Render(" kanban")

	total := len(m.board.Tasks)
	done := m.board.Count(domain.StatusDone)
	inProgress := m.board.Count(domain.StatusInProgress)

	// Visual progress bar
	barWidth := 20
	var progressBar string
	if total > 0 {
		doneW := (done * barWidth) / total
		activeW := (inProgress * barWidth) / total
		if done > 0 && doneW == 0 {
			doneW = 1
		}
		if inProgress > 0 && activeW == 0 {
			activeW = 1
		}
		if doneW+activeW > barWidth {
			activeW = barWidth - doneW
		}
		remainW := barWidth - doneW - activeW
		progressBar = lipgloss.NewStyle().Foreground(theme.Green).Render(strings.Repeat("\u2501", doneW)) +
			lipgloss.NewStyle().Foreground(theme.Peach).Render(strings.Repeat("\u2501", activeW)) +
			lipgloss.NewStyle().Foreground(theme.Surface1).Render(strings.Repeat("\u2501", remainW))
	} else {
		progressBar = lipgloss.NewStyle().Foreground(theme.Surface1).Render(strings.Repeat("\u2501", barWidth))
	}

	// Compact stats
	var stats string
	if total > 0 {
		stats = lipgloss.NewStyle().Foreground(theme.Peach).Render(fmt.Sprintf("%d active", inProgress)) +
			lipgloss.NewStyle().Foreground(theme.Surface2).Render(" \u00b7 ") +
			lipgloss.NewStyle().Foreground(theme.Green).Render(fmt.Sprintf("%d done", done)) +
			lipgloss.NewStyle().Foreground(theme.Surface2).Render(" \u00b7 ") +
			lipgloss.NewStyle().Foreground(theme.Subtext0).Render(fmt.Sprintf("%d total", total))
	}

	// Right side: filter + status
	var rightParts []string
	if m.filter != "" {
		rightParts = append(rightParts,
			lipgloss.NewStyle().Foreground(theme.Blue).Render("\u2315 "+m.filter))
	}
	switch {
	case m.lastErr != nil:
		rightParts = append(rightParts,
			lipgloss.NewStyle().Foreground(theme.Red).Render("\u2717 "+m.lastErr.Error()))
	case m.lastStatus != "":
		rightParts = append(rightParts,
			lipgloss.NewStyle().Foreground(theme.Green).Render("\u2713 "+m.lastStatus))
	}

	left := lipgloss.JoinHorizontal(lipgloss.Center, logo, title)
	if total > 0 {
		left = lipgloss.JoinHorizontal(lipgloss.Center, left, "  ", progressBar, "  ", stats)
	}

	right := strings.Join(rightParts, "  ")
	gap := max(2, m.width-lipgloss.Width(left)-lipgloss.Width(right)-6)

	headerBar := lipgloss.JoinHorizontal(lipgloss.Center, left, spacer(gap), right)

	// Thin separator line
	sepWidth := max(0, m.width-4)
	sep := lipgloss.NewStyle().Foreground(theme.Surface0).Render(strings.Repeat("\u2500", sepWidth))
	content := headerBar + "\n" + sep

	return lipgloss.NewStyle().
		Width(m.width).
		Padding(0, 2).
		PaddingTop(1).
		Background(theme.Mantle).
		Foreground(theme.Text).
		Render(content)
}

// ─── Board ───────────────────────────────────────────────────────────────────

func (m *model) renderBoard() string {
	gap := 2
	statuses := m.board.Statuses()
	if len(statuses) == 0 {
		return ""
	}
	availableWidth := max(0, m.width-4-(gap*(len(statuses)-1)))
	columnWidth := max(24, availableWidth/len(statuses))
	extraWidth := max(0, availableWidth-(columnWidth*len(statuses)))
	columns := make([]string, 0, len(statuses))

	for i, status := range statuses {
		width := columnWidth
		if extraWidth > 0 {
			width++
			extraWidth--
		}
		columns = append(columns, m.renderColumn(status, i == m.activeColumn, width))
	}

	return lipgloss.NewStyle().
		Padding(1, 2, 0, 2).
		Render(joinHorizontal(columns, gap))
}

func (m *model) renderColumn(status domain.Status, active bool, width int) string {
	ids := m.visible[status]
	accent := statusAccent(status)
	innerWidth := max(1, width-4)

	colHeight := max(12, m.height-10)
	columnStyle := lipgloss.NewStyle().
		Width(innerWidth).
		Height(colHeight).
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Surface0)

	if active {
		columnStyle = columnStyle.BorderForeground(accent)
	}

	// Column header: icon + title + pill badge
	icon := statusIcon(status)
	label := lipgloss.NewStyle().Bold(true).Foreground(accent).Render(icon + " " + status.Title())

	countBadge := lipgloss.NewStyle().
		Foreground(theme.Subtext0).
		Background(theme.Surface0).
		PaddingLeft(1).
		PaddingRight(1).
		Render(fmt.Sprintf("%d", len(ids)))
	header := lipgloss.JoinHorizontal(lipgloss.Center, label, " ", countBadge)

	// Accent separator tracks the header width so it stays tidy on narrow columns.
	separatorWidth := min(innerWidth, lipgloss.Width(header))
	if separatorWidth < 1 {
		separatorWidth = 1
	}
	sepChar := "\u2500"
	sepColor := theme.Surface1
	if active {
		sepChar = "\u2501"
		sepColor = accent
	}
	separator := lipgloss.NewStyle().
		Foreground(sepColor).
		Render(strings.Repeat(sepChar, separatorWidth))

	// Task body
	bodyHeight := colHeight - 5
	scroll := m.scroll[status]
	rows := m.taskRows()
	end := min(len(ids), scroll+rows)

	body := make([]string, 0, rows)

	if scroll > 0 {
		body = append(body,
			lipgloss.NewStyle().Foreground(theme.Overlay0).Align(lipgloss.Center).Width(innerWidth).Render("\u25b2 more"),
		)
	}

	if len(ids) == 0 {
		emptyMsg := statusEmptyMessage(status)
		body = append(body,
			lipgloss.NewStyle().
				Foreground(theme.Surface2).
				Italic(true).
				Align(lipgloss.Center).
				Width(innerWidth).
				PaddingTop(2).
				Render(emptyMsg),
		)
	}

	for i := scroll; i < end; i++ {
		task := m.board.Tasks[ids[i]]
		if task == nil {
			continue
		}
		body = append(body, m.renderTaskCard(task, innerWidth, active && i == m.selected[status], accent))
	}

	if hidden := len(ids) - end; hidden > 0 {
		body = append(body,
			lipgloss.NewStyle().Foreground(theme.Overlay0).Align(lipgloss.Center).Width(innerWidth).Render(fmt.Sprintf("\u25bc %d more", hidden)),
		)
	}

	bodyView := lipgloss.NewStyle().Height(bodyHeight).Render(strings.Join(body, "\n"))
	return columnStyle.Render(lipgloss.JoinVertical(lipgloss.Left, header, separator, bodyView))
}

func (m *model) renderTaskCard(task *domain.Task, width int, selected bool, accent lipgloss.Color) string {
	cardWidth := width
	if cardWidth < 6 {
		cardWidth = 6
	}
	innerWidth := cardWidth - 4

	title := truncate(task.Title, innerWidth)
	desc := truncate(singleLine(task.Description), innerWidth)

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Text).Width(innerWidth)
	descStyle := lipgloss.NewStyle().Foreground(theme.Subtext0).Width(innerWidth)
	metaStyle := lipgloss.NewStyle().Foreground(theme.Overlay0).Width(innerWidth)

	var cardParts []string
	cardParts = append(cardParts, titleStyle.Render(title))
	if desc != "" {
		cardParts = append(cardParts, descStyle.Render(desc))
	}
	cardParts = append(cardParts, metaStyle.Render(
		shortID(task.ID)+" \u00b7 "+relativeTime(task.UpdatedAt),
	))

	card := lipgloss.JoinVertical(lipgloss.Left, cardParts...)

	borderColor := theme.Surface1
	if selected {
		borderColor = theme.Mauve
	}

	style := lipgloss.NewStyle().
		Width(innerWidth).
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor)

	return style.Render(card)
}

// ─── Footer ──────────────────────────────────────────────────────────────────

func (m *model) renderFooter() string {
	var content string

	if m.showHelp {
		content = m.help.View(m.keys)
	} else {
		keyStyle := lipgloss.NewStyle().Foreground(theme.Subtext1).Bold(true)
		descStyle := lipgloss.NewStyle().Foreground(theme.Overlay0)
		sepStyle := lipgloss.NewStyle().Foreground(theme.Surface1)
		sep := sepStyle.Render("  \u2502  ")

		content = keyStyle.Render("h/l") + descStyle.Render(" navigate") + sep +
			keyStyle.Render("j/k") + descStyle.Render(" select") + sep +
			keyStyle.Render("H") + descStyle.Render(" move col") + sep +
			keyStyle.Render("L") + descStyle.Render(" move col") + sep +
			keyStyle.Render("c") + descStyle.Render(" column") + sep +
			keyStyle.Render("r") + descStyle.Render(" rename") + sep +
			keyStyle.Render("d") + descStyle.Render(" delete") + sep +
			keyStyle.Render("n") + descStyle.Render(" new") + sep +
			keyStyle.Render("/") + descStyle.Render(" search") + sep +
			keyStyle.Render("\u23ce") + descStyle.Render(" details") + sep +
			keyStyle.Render("q") + descStyle.Render(" quit") + sep +
			keyStyle.Render("?") + descStyle.Render(" more")
	}

	return lipgloss.NewStyle().
		Width(m.width).
		Padding(0, 2, 1, 2).
		Foreground(theme.Subtext0).
		Render(content)
}

// ─── Dialogs ─────────────────────────────────────────────────────────────────

func (m *model) renderCreateDialog() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Mauve).
		Render("\u25c6  New Task")

	titleLabel := lipgloss.NewStyle().
		Foreground(theme.Overlay0).
		Bold(true).
		Render("TITLE")
	descLabel := lipgloss.NewStyle().
		Foreground(theme.Overlay0).
		Bold(true).
		Render("DESCRIPTION")

	separator := lipgloss.NewStyle().
		Foreground(theme.Mauve).
		Render(strings.Repeat("\u2501", 84))

	hintStyle := lipgloss.NewStyle().Foreground(theme.Surface2)
	keyStyle := lipgloss.NewStyle().Foreground(theme.Subtext0)
	hint := keyStyle.Render("tab") + hintStyle.Render(" switch  ") +
		keyStyle.Render("ctrl+s") + hintStyle.Render(" save  ") +
		keyStyle.Render("ctrl+e") + hintStyle.Render(" editor  ") +
		keyStyle.Render("esc") + hintStyle.Render(" cancel")

	errView := ""
	if m.lastErr != nil {
		errView = lipgloss.NewStyle().Foreground(theme.Red).Render("\u2717 " + m.lastErr.Error())
	}

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		separator,
		"",
		titleLabel,
		m.titleInput.View(),
		"",
		descLabel,
		m.descInput.View(),
	)
	if errView != "" {
		content = lipgloss.JoinVertical(lipgloss.Left, content, "", errView)
	}
	content = lipgloss.JoinVertical(lipgloss.Left, content, "", hint)

	return lipgloss.NewStyle().
		Width(92).
		Padding(1, 3).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Mauve).
		Background(theme.Base).
		Render(content)
}

func (m *model) renderSearchDialog() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Blue).
		Render("\u2315  Search")

	separator := lipgloss.NewStyle().
		Foreground(theme.Blue).
		Render(strings.Repeat("\u2501", 46))

	totalVisible := 0
	for _, status := range m.board.Statuses() {
		totalVisible += len(m.visible[status])
	}
	resultText := lipgloss.NewStyle().Foreground(theme.Subtext0).
		Render(fmt.Sprintf("%d tasks matching", totalVisible))

	keyStyle := lipgloss.NewStyle().Foreground(theme.Subtext0)
	hintStyle := lipgloss.NewStyle().Foreground(theme.Surface2)
	hint := keyStyle.Render("enter") + hintStyle.Render(" apply  ") +
		keyStyle.Render("esc") + hintStyle.Render(" restore")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		separator,
		"",
		m.searchInput.View(),
		"",
		resultText,
		"",
		hint,
	)

	return lipgloss.NewStyle().
		Width(50).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Blue).
		Background(theme.Base).
		Render(content)
}

func (m *model) renderDetailDialog() string {
	task := m.selectedTask()
	if task == nil {
		return ""
	}

	accent := statusAccent(task.Status)

	titleView := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Text).
		Render(task.Title)

	statusBadge := lipgloss.NewStyle().
		Foreground(theme.Mantle).
		Background(accent).
		Bold(true).
		PaddingLeft(1).
		PaddingRight(1).
		Render(statusIcon(task.Status) + " " + task.Status.Title())

	separator := lipgloss.NewStyle().
		Foreground(accent).
		Render(strings.Repeat("\u2501", 68))

	labelStyle := lipgloss.NewStyle().Foreground(theme.Overlay0).Width(12)
	valueStyle := lipgloss.NewStyle().Foreground(theme.Subtext1)

	metaRows := []string{
		lipgloss.JoinHorizontal(lipgloss.Top, labelStyle.Render("ID"), valueStyle.Render(task.ID)),
		lipgloss.JoinHorizontal(lipgloss.Top, labelStyle.Render("Status"), statusBadge),
		lipgloss.JoinHorizontal(lipgloss.Top, labelStyle.Render("Created"), valueStyle.Render(task.CreatedAt.Local().Format("02 Jan 2006 15:04"))),
		lipgloss.JoinHorizontal(lipgloss.Top, labelStyle.Render("Updated"), valueStyle.Render(task.UpdatedAt.Local().Format("02 Jan 2006 15:04"))),
	}

	description := strings.TrimSpace(task.Description)
	descView := ""
	if description != "" {
		descView = lipgloss.NewStyle().
			Width(64).
			Foreground(theme.Subtext1).
			PaddingTop(1).
			Render(description)
	} else {
		descView = lipgloss.NewStyle().
			Foreground(theme.Surface2).
			Italic(true).
			PaddingTop(1).
			Render("No description")
	}

	keyStyle := lipgloss.NewStyle().Foreground(theme.Subtext0)
	hintStyle := lipgloss.NewStyle().Foreground(theme.Surface2)
	hint := keyStyle.Render("esc") + hintStyle.Render(" close")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		titleView,
		separator,
		"",
		strings.Join(metaRows, "\n"),
		descView,
		"",
		hint,
	)

	return lipgloss.NewStyle().
		Width(72).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accent).
		Background(theme.Base).
		Render(content)
}

func (m *model) renderAddColumnDialog() string {
	isRename := m.mode == modeRenameColumn

	titleText := "New Column"
	saveHint := "save"
	if isRename {
		titleText = "Rename Column"
		saveHint = "rename"
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Mauve).
		Render("\u25c6  " + titleText)

	label := lipgloss.NewStyle().
		Foreground(theme.Overlay0).
		Bold(true).
		Render("NAME")

	separator := lipgloss.NewStyle().
		Foreground(theme.Mauve).
		Render(strings.Repeat("\u2501", 42))

	errView := ""
	if m.lastErr != nil {
		errView = lipgloss.NewStyle().Foreground(theme.Red).Render("\u2717 " + m.lastErr.Error())
	}

	hintStyle := lipgloss.NewStyle().Foreground(theme.Surface2)
	keyStyle := lipgloss.NewStyle().Foreground(theme.Subtext0)
	hint := keyStyle.Render("enter") + hintStyle.Render(" "+saveHint+"  ") +
		keyStyle.Render("esc") + hintStyle.Render(" cancel")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		separator,
		"",
		label,
		m.columnInput.View(),
	)
	if errView != "" {
		content = lipgloss.JoinVertical(lipgloss.Left, content, "", errView)
	}
	content = lipgloss.JoinVertical(lipgloss.Left, content, "", hint)

	return lipgloss.NewStyle().
		Width(56).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Mauve).
		Background(theme.Base).
		Render(content)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func (m *model) taskRows() int {
	height := max(1, m.height-16)
	rows := height / cardSlotHeight
	if rows < 1 {
		return 1
	}
	return rows
}

func (m *model) placeOverlayCenter(base string, overlay string) string {
	// Dim the base to create a frosted backdrop
	bg := dimContent(base)
	bgLines := strings.Split(bg, "\n")
	overlayLines := strings.Split(overlay, "\n")

	// Pad background to fill terminal height
	for len(bgLines) < m.height {
		bgLines = append(bgLines, "")
	}

	// Center the overlay on top of the dimmed background
	startRow := max(0, (m.height-len(overlayLines))/2)
	for i, oLine := range overlayLines {
		row := startRow + i
		if row >= len(bgLines) {
			break
		}
		oWidth := lipgloss.Width(oLine)
		padLeft := max(0, (m.width-oWidth)/2)
		bgLines[row] = spacer(padLeft) + oLine
	}

	return strings.Join(bgLines, "\n")
}

// dimContent strips ANSI codes and re-renders all visible characters
// in a muted color to create a frosted/blurred backdrop effect.
func dimContent(s string) string {
	dimStyle := lipgloss.NewStyle().Foreground(theme.Surface0)
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		plain := ansiStripRe.ReplaceAllString(line, "")
		lines[i] = dimStyle.Render(plain)
	}
	return strings.Join(lines, "\n")
}

func saveBoardCmd(boardStore store.BoardStore, board *domain.Board) tea.Cmd {
	return func() tea.Msg {
		return saveFinishedMsg{err: boardStore.Save(board)}
	}
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
	return string(runes[:width-1]) + "\u2026"
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

func statusIcon(status domain.Status) string {
	switch status {
	case domain.StatusBacklog:
		return "\u25cb" // ○
	case domain.StatusInProgress:
		return "\u25d0" // ◐
	case domain.StatusDone:
		return "\u25cf" // ●
	default:
		return "\u25cb"
	}
}

func statusEmptyMessage(status domain.Status) string {
	switch status {
	case domain.StatusBacklog:
		return "Press n to add a task"
	case domain.StatusInProgress:
		return "Move tasks here with ]"
	case domain.StatusDone:
		return "Completed tasks appear here"
	default:
		return "No tasks"
	}
}

func relativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		mins := int(d.Minutes())
		if mins == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", mins)
	case d < 24*time.Hour:
		hours := int(d.Hours())
		if hours == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", hours)
	case d < 7*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "yesterday"
		}
		return fmt.Sprintf("%dd ago", days)
	default:
		return t.Local().Format("02 Jan")
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
