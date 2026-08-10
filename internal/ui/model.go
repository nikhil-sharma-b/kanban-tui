package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/cursor"
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

const (
	compactBoardBreakpoint = 100
	boardGap               = 2
	createDialogMaxWidth   = 92
	searchDialogMaxWidth   = 50
	detailDialogMaxWidth   = 72
	columnDialogMaxWidth   = 56
	projectDialogMaxWidth  = 60
	defaultDialogPadding   = 2
	searchDialogPadding    = 2
	createDialogPadding    = 3
	maxCreateInputWidth    = 84
	maxModalInputWidth     = 42
	maxDescriptionHeight   = 14
	minDescriptionHeight   = 4
	columnMinWidth         = 18
	compactColumnWidth     = 24
)

// leftAccentBorder defines a border with only a left accent bar for modern card styling.
var leftAccentBorder = lipgloss.Border{
	Left: "┃",
}

var theme = struct {
	Rosewater lipgloss.TerminalColor
	Flamingo  lipgloss.TerminalColor
	Pink      lipgloss.TerminalColor
	Mauve     lipgloss.TerminalColor
	Red       lipgloss.TerminalColor
	Peach     lipgloss.TerminalColor
	Yellow    lipgloss.TerminalColor
	Green     lipgloss.TerminalColor
	Teal      lipgloss.TerminalColor
	Blue      lipgloss.TerminalColor
	Lavender  lipgloss.TerminalColor
	Text      lipgloss.TerminalColor
	Subtext1  lipgloss.TerminalColor
	Subtext0  lipgloss.TerminalColor
	Overlay0  lipgloss.TerminalColor
	Surface2  lipgloss.TerminalColor
	Surface1  lipgloss.TerminalColor
	Surface0  lipgloss.TerminalColor
	Base      lipgloss.TerminalColor
	Mantle    lipgloss.TerminalColor
	Crust     lipgloss.TerminalColor
}{
	// ANSI colors inherit their RGB values from the user's terminal theme.
	Rosewater: lipgloss.Color("7"),
	Flamingo:  lipgloss.Color("1"),
	Pink:      lipgloss.Color("5"),
	Mauve:     lipgloss.Color("5"),
	Red:       lipgloss.Color("1"),
	Peach:     lipgloss.Color("3"),
	Yellow:    lipgloss.Color("3"),
	Green:     lipgloss.Color("2"),
	Teal:      lipgloss.Color("6"),
	Blue:      lipgloss.Color("4"),
	Lavender:  lipgloss.Color("6"),
	Text:      lipgloss.NoColor{},
	Subtext1:  lipgloss.NoColor{},
	Subtext0:  lipgloss.Color("8"),
	Overlay0:  lipgloss.Color("8"),
	Surface2:  lipgloss.Color("8"),
	Surface1:  lipgloss.Color("8"),
	Surface0:  lipgloss.Color("8"),
	Base:      lipgloss.NoColor{},
	Mantle:    lipgloss.NoColor{},
	Crust:     lipgloss.NoColor{},
}

type mode int

const (
	modeBoard mode = iota
	modeCreate
	modeSearch
	modeDetail
	modeAddColumn
	modeRenameColumn
	modeProjects
	modeProjectEdit
	modeWhiteboards
	modeWhiteboardRename
	modeConfirm
	modeArchive
	modeArchiveDetail
)

// bulkArchiveAge is the fixed v1 threshold for archiving old Done tasks.
const bulkArchiveAge = 30 * 24 * time.Hour

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
	Projects     key.Binding
	Whiteboards  key.Binding
	Search       key.Binding
	Edit         key.Binding
	Open         key.Binding
	Delete       key.Binding
	Archive      key.Binding
	ArchiveOld   key.Binding
	ArchiveView  key.Binding
	Daily        key.Binding
	DailyDone    key.Binding
	DailyClear   key.Binding
	Help         key.Binding
	Quit         key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Left, k.Right, k.Up, k.Down, k.NewTask, k.NewColumn, k.Projects, k.Daily, k.Open, k.Edit, k.Archive, k.ArchiveView, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Left, k.Right, k.Up, k.Down},
		{k.MoveColLeft, k.MoveColRight, k.MoveLeft, k.MoveRight},
		{k.ReorderUp, k.ReorderDown, k.NewTask, k.NewColumn, k.Projects, k.RenameCol, k.DeleteCol, k.Search, k.Open, k.Edit, k.Delete},
		{k.Archive, k.ArchiveOld, k.ArchiveView},
		{k.Daily, k.DailyDone, k.DailyClear},
		{k.Help, k.Quit},
	}
}

type model struct {
	workspace          *domain.Workspace
	project            *domain.Project
	board              *domain.Board
	store              store.WorkspaceStore
	dataPath           string
	width              int
	height             int
	activeColumn       int
	selected           map[domain.Status]int
	scroll             map[domain.Status]int
	visible            map[domain.Status][]string
	filter             string
	filterDraft        string
	columnInput        textinput.Model
	projectInput       textinput.Model
	mode               mode
	columnRename       domain.Status
	projectDraft       string
	projectCursor      int
	projectFilterInput textinput.Model
	projectFiltering   bool
	whiteboardInput    textinput.Model
	whiteboardCursor   int
	whiteboardRenameID string
	archiveCursor      int
	archiveFilterInput textinput.Model
	archiveFiltering   bool
	titleInput         textinput.Model
	descInput          textarea.Model
	searchInput        textinput.Model
	help               help.Model
	keys               keyMap
	editingTaskID      string
	vimNormal          bool
	vimVisual          *vimSelection
	vim                vimEngine
	vimReplace         bool   // R – replace mode (overtype)
	vimStatus          string // transient feedback shown in dialog (e.g. yanked "foo")
	vimCommand         string // command-line input in normal mode, including leading ':'
	showHelp           bool
	lastStatus         string
	lastErr            error
	confirmMsg         string
	confirmPrev        mode
	confirmAction      func() (tea.Model, tea.Cmd)
	// prevProjectID is the project to return to when leaving the daily board.
	prevProjectID string
}

// ansiStripRe matches ANSI escape sequences for the dim/blur effect.
var ansiStripRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func New(workspace *domain.Workspace, boardStore store.WorkspaceStore, dataPath string) tea.Model {
	if workspace == nil {
		workspace = domain.NewWorkspace()
	}
	if err := workspace.Normalize(); err != nil {
		workspace = domain.NewWorkspace()
	}
	assignWorkspaceWhiteboardPaths(workspace, dataPath)
	project := workspace.ActiveProject()
	if project == nil {
		project, _ = workspace.CreateProject(domain.DefaultProjectName)
	}
	board := project.Board

	titleInput := textinput.New()
	titleInput.Prompt = ""
	titleInput.Placeholder = "What needs to be done?"
	titleInput.CharLimit = 120
	titleInput.Width = maxCreateInputWidth
	titleInput.TextStyle = lipgloss.NewStyle().Foreground(theme.Text)
	titleInput.PlaceholderStyle = lipgloss.NewStyle().Foreground(theme.Overlay0)

	descInput := textarea.New()
	descInput.Prompt = ""
	descInput.Placeholder = "Add details (optional)"
	descInput.SetWidth(maxCreateInputWidth)
	descInput.SetHeight(maxDescriptionHeight)
	descInput.ShowLineNumbers = false
	descInput.FocusedStyle.Base = lipgloss.NewStyle().Foreground(theme.Text)
	descInput.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(theme.Overlay0)
	descInput.FocusedStyle.CursorLine = lipgloss.NewStyle()
	descInput.BlurredStyle.Base = lipgloss.NewStyle().Foreground(theme.Subtext1)
	descInput.BlurredStyle.Placeholder = lipgloss.NewStyle().Foreground(theme.Overlay0)

	searchInput := textinput.New()
	searchInput.Prompt = ""
	searchInput.Placeholder = "Type to filter tasks..."
	searchInput.Width = maxModalInputWidth
	searchInput.TextStyle = lipgloss.NewStyle().Foreground(theme.Text)
	searchInput.PlaceholderStyle = lipgloss.NewStyle().Foreground(theme.Overlay0)

	columnInput := textinput.New()
	columnInput.Placeholder = "Column name"
	columnInput.Width = maxModalInputWidth
	columnInput.TextStyle = lipgloss.NewStyle().Foreground(theme.Text)
	columnInput.PlaceholderStyle = lipgloss.NewStyle().Foreground(theme.Overlay0)

	projectInput := textinput.New()
	projectInput.Placeholder = "Project name"
	projectInput.Width = maxModalInputWidth
	projectInput.TextStyle = lipgloss.NewStyle().Foreground(theme.Text)
	projectInput.PlaceholderStyle = lipgloss.NewStyle().Foreground(theme.Overlay0)

	projectFilterInput := textinput.New()
	projectFilterInput.Prompt = "/ "
	projectFilterInput.Placeholder = "Fuzzy filter projects..."
	projectFilterInput.Width = maxModalInputWidth
	projectFilterInput.TextStyle = lipgloss.NewStyle().Foreground(theme.Text)
	projectFilterInput.PlaceholderStyle = lipgloss.NewStyle().Foreground(theme.Overlay0)

	whiteboardInput := textinput.New()
	whiteboardInput.Placeholder = "Whiteboard name"
	whiteboardInput.Width = maxModalInputWidth
	whiteboardInput.TextStyle = lipgloss.NewStyle().Foreground(theme.Text)
	whiteboardInput.PlaceholderStyle = lipgloss.NewStyle().Foreground(theme.Overlay0)

	archiveFilterInput := textinput.New()
	archiveFilterInput.Prompt = "/ "
	archiveFilterInput.Placeholder = "Filter archived tasks..."
	archiveFilterInput.Width = maxModalInputWidth
	archiveFilterInput.TextStyle = lipgloss.NewStyle().Foreground(theme.Text)
	archiveFilterInput.PlaceholderStyle = lipgloss.NewStyle().Foreground(theme.Overlay0)

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
		workspace:          workspace,
		project:            project,
		board:              board,
		store:              boardStore,
		dataPath:           dataPath,
		selected:           selected,
		scroll:             scroll,
		visible:            visible,
		titleInput:         titleInput,
		descInput:          descInput,
		searchInput:        searchInput,
		columnInput:        columnInput,
		projectInput:       projectInput,
		projectFilterInput: projectFilterInput,
		whiteboardInput:    whiteboardInput,
		archiveFilterInput: archiveFilterInput,
		help:               help.New(),
		showHelp:           true,
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
			Projects:     key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "projects")),
			Whiteboards:  key.NewBinding(key.WithKeys("w"), key.WithHelp("w", "whiteboards")),
			MoveColLeft:  key.NewBinding(key.WithKeys("H"), key.WithHelp("H", "move column left")),
			MoveColRight: key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "move column right")),
			RenameCol:    key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "rename column")),
			DeleteCol:    key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete column")),
			Search:       key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
			Edit:         key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit selected")),
			Open:         key.NewBinding(key.WithKeys("enter"), key.WithHelp("\u23ce", "details")),
			Delete:       key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "delete")),
			Archive:      key.NewBinding(key.WithKeys("A"), key.WithHelp("A", "archive task")),
			ArchiveOld:   key.NewBinding(key.WithKeys("ctrl+a"), key.WithHelp("ctrl+a", "archive old done")),
			ArchiveView:  key.NewBinding(key.WithKeys("z"), key.WithHelp("z", "archive view")),
			Daily:        key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "daily board")),
			DailyDone:    key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "daily: mark done")),
			DailyClear:   key.NewBinding(key.WithKeys("X"), key.WithHelp("X", "daily: clear board")),
			Help:         key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "toggle help")),
			Quit:         key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		},
	}

	if !project.Daily {
		m.prevProjectID = project.ID
	}

	m.recalculateVisible()
	m.syncResponsiveLayout()
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
		m.syncResponsiveLayout()
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
		case modeProjects:
			return m.updateProjects(msg)
		case modeProjectEdit:
			return m.updateProjectEdit(msg)
		case modeWhiteboards:
			return m.updateWhiteboards(msg)
		case modeWhiteboardRename:
			return m.updateWhiteboardRename(msg)
		case modeConfirm:
			return m.updateConfirm(msg)
		case modeArchive:
			return m.updateArchive(msg)
		case modeArchiveDetail:
			return m.updateArchiveDetail(msg)
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
	case modeProjects:
		return m.placeOverlayCenter(view, m.renderProjectsDialog())
	case modeProjectEdit:
		return m.placeOverlayCenter(view, m.renderProjectEditDialog())
	case modeWhiteboards:
		return m.placeOverlayCenter(view, m.renderWhiteboardsDialog())
	case modeWhiteboardRename:
		return m.placeOverlayCenter(view, m.renderWhiteboardRenameDialog())
	case modeConfirm:
		return m.placeOverlayCenter(view, m.renderConfirmDialog())
	case modeArchive:
		return m.placeOverlayCenter(view, m.renderArchiveDialog())
	case modeArchiveDetail:
		return m.placeOverlayCenter(view, m.renderTaskDetail(m.selectedArchivedTask(), true))
	default:
		return view
	}
}

func (m *model) updateBoard(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	statuses := m.board.Statuses()
	if len(statuses) == 0 {
		return m, nil
	}

	if m.onDailyBoard() {
		switch {
		case key.Matches(msg, m.keys.DailyDone):
			return m.markDailyDone()
		case key.Matches(msg, m.keys.DailyClear):
			if len(m.board.Tasks) == 0 {
				m.lastErr = nil
				m.lastStatus = "daily board is already empty"
				return m, nil
			}
			return m.askConfirm(
				fmt.Sprintf("Clear the daily board? %d task(s) will be deleted.", len(m.board.Tasks)),
				modeBoard,
				func() (tea.Model, tea.Cmd) { return m.clearDailyBoard() },
			)
		case key.Matches(msg, m.keys.NewColumn),
			key.Matches(msg, m.keys.RenameCol),
			key.Matches(msg, m.keys.DeleteCol),
			key.Matches(msg, m.keys.MoveColLeft),
			key.Matches(msg, m.keys.MoveColRight):
			m.lastErr = nil
			m.lastStatus = "daily board columns are fixed"
			return m, nil
		}
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Daily):
		return m.toggleDaily()
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
		m.editingTaskID = ""
		m.mode = modeCreate
		m.vimNormal = false
		m.vimVisual = nil
		m.vimReplace = false
		m.vim.reset()
		m.titleInput.SetValue("")
		m.descInput.SetValue("")
		m.titleInput.Focus()
		m.descInput.Blur()
		m.lastErr = nil
		return m, tea.Batch(textinput.Blink, m.syncVimCursor())
	case key.Matches(msg, m.keys.Edit):
		return m.beginEditSelected()
	case key.Matches(msg, m.keys.NewColumn):
		m.mode = modeAddColumn
		m.columnInput.SetValue("")
		m.columnInput.Focus()
		m.lastErr = nil
		return m, textinput.Blink
	case key.Matches(msg, m.keys.Projects):
		m.mode = modeProjects
		m.projectCursor = m.activeProjectIndex()
		m.projectInput.Blur()
		m.projectFiltering = true
		m.projectFilterInput.SetValue("")
		m.projectFilterInput.Focus()
		m.lastErr = nil
		return m, textinput.Blink
	case key.Matches(msg, m.keys.RenameCol):
		return m.beginRenameColumn()
	case key.Matches(msg, m.keys.DeleteCol):
		statuses := m.board.Statuses()
		if len(statuses) > 0 && m.activeColumn < len(statuses) {
			col := statuses[m.activeColumn]
			taskCount := len(m.board.Order[col])
			msg := fmt.Sprintf("Delete column %q?", col.Title())
			if taskCount > 0 {
				msg = fmt.Sprintf("Delete column %q? Its %d task(s) will move to an adjacent column.", col.Title(), taskCount)
			}
			return m.askConfirm(msg, modeBoard, func() (tea.Model, tea.Cmd) { return m.deleteColumn() })
		}
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
		task := m.selectedTask()
		if task != nil {
			title := task.Title
			if len(title) > 20 {
				title = title[:20] + "…"
			}
			return m.askConfirm(
				fmt.Sprintf("Delete task %q?", title),
				modeBoard,
				func() (tea.Model, tea.Cmd) { return m.deleteSelected() },
			)
		}
	case key.Matches(msg, m.keys.Archive):
		task := m.selectedTask()
		if task != nil {
			title := task.Title
			if len(title) > 20 {
				title = title[:20] + "…"
			}
			return m.askConfirm(
				fmt.Sprintf("Archive task %q?", title),
				modeBoard,
				func() (tea.Model, tea.Cmd) { return m.archiveSelected() },
			)
		}
	case key.Matches(msg, m.keys.ArchiveOld):
		return m.askConfirm(
			"Archive Done tasks not updated in more than 30 days?",
			modeBoard,
			func() (tea.Model, tea.Cmd) { return m.archiveOldDone() },
		)
	case key.Matches(msg, m.keys.ArchiveView):
		m.mode = modeArchive
		m.archiveCursor = 0
		m.archiveFiltering = false
		m.archiveFilterInput.SetValue("")
		m.archiveFilterInput.Blur()
		m.lastErr = nil
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
	content += "# Save and quit to apply, empty to cancel\n"

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
		if m.editingTaskID == "" {
			m.lastStatus = "task creation cancelled"
		} else {
			m.lastStatus = "task edit cancelled"
			m.editingTaskID = ""
		}
		m.lastErr = nil
		m.mode = modeBoard
		return m, nil
	}

	if m.editingTaskID != "" {
		task, err := m.board.UpdateTask(m.editingTaskID, title, description)
		m.editingTaskID = ""
		if err != nil {
			m.lastErr = err
			return m, nil
		}

		m.mode = modeBoard
		m.lastStatus = fmt.Sprintf("updated %s", shortID(task.ID))
		m.lastErr = nil
		m.recalculateVisible()
		m.selectTask(task.ID)

		return m, m.saveWorkspaceCmd()
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

	return m, m.saveWorkspaceCmd()
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
	model, cmd := m.updateCreateInner(msg)
	return model, tea.Batch(cmd, m.syncVimCursor())
}

func (m *model) updateCreateInner(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Always handle ctrl+e regardless of vim mode.
	switch msg.String() {
	case "ctrl+e":
		m.mode = modeBoard
		m.titleInput.Blur()
		m.descInput.Blur()
		return m.openEditorWithDraft()
	}

	key := msg.String()
	if len(msg.Runes) == 1 {
		key = string(msg.Runes)
	}

	m.vimStatus = "" // any keypress clears transient yank/paste feedback

	if m.vimNormal {
		return m.updateCreateVimNormal(msg, key)
	}

	if m.vimReplace {
		return m.updateCreateReplace(msg)
	}

	// Insert mode: esc enters vim normal mode.
	switch msg.String() {
	case "esc":
		m.vimNormal = true
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
	}

	var cmd tea.Cmd
	if m.titleInput.Focused() {
		m.titleInput, cmd = m.titleInput.Update(msg)
		return m, cmd
	}

	m.descInput, cmd = m.descInput.Update(msg)
	return m, cmd
}

func (m *model) updateCreateVimNormal(msg tea.KeyMsg, key string) (tea.Model, tea.Cmd) {
	if key == "" && len(msg.Runes) == 1 {
		key = string(msg.Runes)
	}
	if m.vimCommand != "" {
		return m.updateCreateVimCommand(msg)
	}

	if m.vimVisual != nil {
		return m.updateCreateVimVisual(key)
	}
	switch key {
	case ":":
		m.vim.reset()
		m.vimCommand = ":"
		m.vimStatus = m.vimCommand
		return m, nil
	case "esc":
		if m.vim.pending() {
			m.vim.reset()
			return m, nil
		}
		m.vimNormal = false
		m.mode = modeBoard
		m.titleInput.Blur()
		m.descInput.Blur()
		m.editingTaskID = ""
		return m, nil
	case "v":
		m.vim.reset()
		pos := m.focusedVimBuffer().Cursor()
		m.vimVisual = &vimSelection{anchor: pos, cursor: pos}
		return m, nil
	case "tab", "shift+tab":
		m.vim.reset()
		if m.titleInput.Focused() {
			m.titleInput.Blur()
			m.descInput.Focus()
		} else {
			m.descInput.Blur()
			m.titleInput.Focus()
		}
		return m, nil
	case "j", "k":
		// Plain j/k use the textarea's wrap-aware vertical movement;
		// with a pending operator they fall through to the engine.
		if !m.vim.pending() {
			if !m.titleInput.Focused() {
				if key == "j" {
					m.descInput.CursorDown()
				} else {
					m.descInput.CursorUp()
				}
			}
			return m, nil
		}
	}

	if key == "R" && !m.vim.pending() {
		m.vimNormal = false
		m.vimReplace = true
		return m, nil
	}

	res := m.vim.HandleKey(m.focusedVimBuffer(), key)
	if res.enterInsert {
		m.vimNormal = false
	}
	m.vimStatus = m.vim.status
	return m, nil
}

func (m *model) updateCreateVimVisual(key string) (tea.Model, tea.Cmd) {
	buf := m.focusedVimBuffer()
	selection := m.vimVisual
	switch key {
	case "esc", "v":
		m.vimVisual = nil
		return m, nil
	case "d", "x", "c", "y":
		text := []rune(buf.Text())
		start, end := selection.rangeBounds(len(text))
		op := []rune(key)[0]
		if op == 'x' {
			op = 'd'
		}
		res := m.vim.opRange(buf, text, start, end, false, op)
		m.vimStatus = m.vim.status
		m.vimVisual = nil
		if res.enterInsert {
			m.vimNormal = false
		}
		return m, nil
	}

	text := []rune(buf.Text())
	pos := clampInt(selection.cursor, 0, len(text))
	if len(text) > 0 && pos == len(text) {
		pos--
	}
	if mr, ok := visualMotionTarget(key, text, pos); ok {
		selection.cursor = clampInt(mr.pos, 0, max(len(text)-1, 0))
		buf.SetCursor(selection.cursor)
	}
	return m, nil
}

func visualMotionTarget(key string, text []rune, pos int) (motionResult, bool) {
	if key == "G" {
		return motionResult{pos: max(len(text)-1, 0)}, true
	}
	runes := []rune(key)
	if len(runes) != 1 {
		return motionResult{}, false
	}
	return motionTarget(runes[0], text, pos, 1)
}

func (m *model) updateCreateVimCommand(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.vimCommand = ""
		m.vimStatus = ""
		return m, nil
	case "backspace":
		runes := []rune(m.vimCommand)
		if len(runes) <= 1 {
			m.vimCommand = ""
			m.vimStatus = ""
		} else {
			m.vimCommand = string(runes[:len(runes)-1])
			m.vimStatus = m.vimCommand
		}
		return m, nil
	case "enter":
		command := m.vimCommand
		m.vimCommand = ""
		m.vimStatus = ""
		switch command {
		case ":w":
			return m.saveTask(false)
		case ":wq":
			return m.saveTask(true)
		default:
			m.vimStatus = fmt.Sprintf("not an editor command: %s", command)
			return m, nil
		}
	}

	if msg.Type == tea.KeyRunes {
		m.vimCommand += string(msg.Runes)
		m.vimStatus = m.vimCommand
	}
	return m, nil
}

func (m *model) focusedVimBuffer() vimBuffer {
	if m.titleInput.Focused() {
		return titleBuffer{m}
	}
	return descBuffer{m}
}

// updateCreateReplace implements vim's R (overtype) mode.
func (m *model) updateCreateReplace(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	buf := m.focusedVimBuffer()
	text := []rune(buf.Text())
	pos := clampInt(buf.Cursor(), 0, len(text))

	switch msg.String() {
	case "esc":
		m.vimReplace = false
		m.vimNormal = true
		return m, nil
	case "backspace":
		if pos > lineStart(text, pos) {
			buf.SetCursor(pos - 1)
		}
		return m, nil
	case "enter":
		if !buf.SingleLine() {
			buf.SetText(string(text[:pos])+"\n"+string(text[pos:]), pos+1)
		}
		return m, nil
	}

	if msg.Type != tea.KeyRunes {
		return m, nil
	}
	for _, r := range msg.Runes {
		if pos < len(text) && text[pos] != '\n' {
			text[pos] = r
		} else {
			text = append(text[:pos:pos], append([]rune{r}, text[pos:]...)...)
		}
		pos++
	}
	buf.SetText(string(text), pos)
	return m, nil
}

// syncVimCursor recolors the input cursors to reflect the current vim mode:
// green block in normal, yellow while an operator/prefix is pending, red in
// replace mode, default blinking cursor in insert.
func (m *model) syncVimCursor() tea.Cmd {
	style := lipgloss.NewStyle()
	mode := cursor.CursorBlink
	switch {
	case m.vimReplace:
		style = style.Foreground(theme.Red)
		mode = cursor.CursorStatic
	case m.vimNormal && m.vim.pending():
		style = style.Foreground(theme.Yellow)
		mode = cursor.CursorStatic
	case m.vimNormal:
		style = style.Foreground(theme.Green)
		mode = cursor.CursorStatic
	}
	m.titleInput.Cursor.Style = style
	m.descInput.Cursor.Style = style
	return tea.Batch(
		m.titleInput.Cursor.SetMode(mode),
		m.descInput.Cursor.SetMode(mode),
	)
}

func (m *model) saveTask(closePanel bool) (tea.Model, tea.Cmd) {
	title := m.titleInput.Value()
	description := m.descInput.Value()

	if m.editingTaskID != "" {
		task, err := m.board.UpdateTask(m.editingTaskID, title, description)
		if err != nil {
			m.lastErr = err
			return m, nil
		}

		m.lastStatus = fmt.Sprintf("updated %s", shortID(task.ID))
		m.lastErr = nil
		m.recalculateVisible()
		m.selectTask(task.ID)
		if closePanel {
			m.closeCreatePanel()
		}

		return m, m.saveWorkspaceCmd()
	}

	task, err := m.board.AddTask(title, description)
	if err != nil {
		m.lastErr = err
		return m, nil
	}

	m.lastStatus = fmt.Sprintf("created %s", shortID(task.ID))
	m.lastErr = nil
	m.editingTaskID = task.ID
	m.filter = ""
	m.searchInput.SetValue("")
	m.activeColumn = 0
	m.recalculateVisible()
	m.selected[task.Status] = len(m.visible[task.Status]) - 1
	m.syncScroll(task.Status)
	if closePanel {
		m.closeCreatePanel()
	}

	return m, m.saveWorkspaceCmd()
}

func (m *model) closeCreatePanel() {
	m.mode = modeBoard
	m.editingTaskID = ""
	m.vimNormal = false
	m.vimVisual = nil
	m.vimReplace = false
	m.vimCommand = ""
	m.titleInput.Blur()
	m.descInput.Blur()
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
			return m, m.saveWorkspaceCmd()
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

		return m, m.saveWorkspaceCmd()
	}

	var cmd tea.Cmd
	m.columnInput, cmd = m.columnInput.Update(msg)
	return m, cmd
}

// fuzzyMatch reports whether every rune of query appears in target in order
// (case-insensitive subsequence match).
func fuzzyMatch(query, target string) bool {
	query = strings.ToLower(query)
	target = strings.ToLower(target)
	runes := []rune(target)
	i := 0
	for _, q := range query {
		found := false
		for ; i < len(runes); i++ {
			if runes[i] == q {
				found = true
				i++
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (m *model) filteredProjects() []*domain.Project {
	projects := m.workspace.RegularProjects()
	query := strings.TrimSpace(m.projectFilterInput.Value())
	if query == "" {
		return projects
	}
	matches := make([]*domain.Project, 0, len(projects))
	for _, project := range projects {
		if fuzzyMatch(query, project.Name) {
			matches = append(matches, project)
		}
	}
	return matches
}

func (m *model) updateProjects(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.projectFiltering {
		switch msg.String() {
		case "esc":
			m.projectFiltering = false
			m.projectFilterInput.SetValue("")
			m.projectFilterInput.Blur()
			m.projectCursor = 0
			return m, nil
		case "enter":
			m.projectFiltering = false
			m.projectFilterInput.Blur()
			if projects := m.filteredProjects(); len(projects) > 0 {
				if m.projectCursor >= len(projects) {
					m.projectCursor = len(projects) - 1
				}
				return m.switchProject(projects[m.projectCursor].ID)
			}
			return m, nil
		case "up", "down":
			// fall through to list navigation below
		default:
			var cmd tea.Cmd
			m.projectFilterInput, cmd = m.projectFilterInput.Update(msg)
			m.projectCursor = 0
			return m, cmd
		}
	}

	projects := m.filteredProjects()
	if m.projectCursor >= len(projects) {
		m.projectCursor = max(0, len(projects)-1)
	}

	switch msg.String() {
	case "esc":
		if strings.TrimSpace(m.projectFilterInput.Value()) != "" {
			m.projectFilterInput.SetValue("")
			m.projectCursor = 0
			return m, nil
		}
		m.mode = modeBoard
		m.lastErr = nil
		return m, nil
	case "/":
		m.projectFiltering = true
		m.projectFilterInput.CursorEnd()
		m.projectFilterInput.Focus()
		m.lastErr = nil
		return m, textinput.Blink
	}

	if len(projects) == 0 {
		return m, nil
	}

	switch msg.String() {
	case "up", "k":
		if m.projectCursor > 0 {
			m.projectCursor--
		}
		return m, nil
	case "down", "j":
		if m.projectCursor < len(projects)-1 {
			m.projectCursor++
		}
		return m, nil
	case "enter":
		return m.switchProject(projects[m.projectCursor].ID)
	case "n":
		m.mode = modeProjectEdit
		m.projectDraft = ""
		m.projectInput.SetValue("")
		m.projectInput.Focus()
		m.lastErr = nil
		return m, textinput.Blink
	case "e":
		m.mode = modeProjectEdit
		m.projectDraft = projects[m.projectCursor].ID
		m.projectInput.SetValue(projects[m.projectCursor].Name)
		m.projectInput.Focus()
		m.lastErr = nil
		return m, textinput.Blink
	case "x":
		project := projects[m.projectCursor]
		return m.askConfirm(
			fmt.Sprintf("Delete project %q and all its tasks?", project.Name),
			modeProjects,
			func() (tea.Model, tea.Cmd) {
				if err := m.workspace.DeleteProject(project.ID); err != nil {
					m.lastErr = err
					m.mode = modeProjects
					return m, nil
				}
				m.activateProject(m.workspace.ActiveProjectID)
				if remaining := len(m.workspace.RegularProjects()); m.projectCursor >= remaining {
					m.projectCursor = remaining - 1
				}
				m.lastErr = nil
				m.lastStatus = fmt.Sprintf("deleted project %s", project.Name)
				m.mode = modeProjects
				return m, m.saveWorkspaceCmd()
			},
		)
	}

	return m, nil
}

func (m *model) updateProjectEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeProjects
		m.projectInput.Blur()
		m.lastErr = nil
		m.projectDraft = ""
		return m, nil
	case "enter":
		name := m.projectInput.Value()
		if m.projectDraft == "" {
			project, err := m.workspace.CreateProject(name)
			if err != nil {
				m.lastErr = err
				return m, nil
			}
			m.activateProject(project.ID)
			m.mode = modeBoard
			m.projectInput.Blur()
			m.lastErr = nil
			m.lastStatus = fmt.Sprintf("created project %s", project.Name)
			return m, m.saveWorkspaceCmd()
		}

		previousPaths := snapshotProjectWhiteboardPaths(m.workspace.ProjectByID(m.projectDraft), m.dataPath)
		project, err := m.workspace.RenameProject(m.projectDraft, name)
		if err != nil {
			m.lastErr = err
			return m, nil
		}
		if err := relocateProjectWhiteboardFiles(project, m.dataPath, previousPaths); err != nil {
			m.lastErr = fmt.Errorf("rename project whiteboards: %w", err)
			return m, nil
		}
		m.mode = modeProjects
		m.projectInput.Blur()
		m.projectDraft = ""
		m.projectCursor = m.regularProjectIndex(project.ID)
		m.lastErr = nil
		m.lastStatus = fmt.Sprintf("renamed project %s", project.Name)
		return m, m.saveWorkspaceCmd()
	}

	var cmd tea.Cmd
	m.projectInput, cmd = m.projectInput.Update(msg)
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

	return m, m.saveWorkspaceCmd()
}

func (m *model) beginEditSelected() (tea.Model, tea.Cmd) {
	task := m.selectedTask()
	if task == nil {
		return m, nil
	}

	m.editingTaskID = task.ID
	m.mode = modeCreate
	m.vimNormal = false
	m.vimVisual = nil
	m.vimReplace = false
	m.vim.reset()
	m.titleInput.SetValue(task.Title)
	m.descInput.SetValue(task.Description)
	m.titleInput.Focus()
	m.descInput.Blur()
	m.lastErr = nil
	m.lastStatus = ""

	return m, tea.Batch(textinput.Blink, m.syncVimCursor())
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

	m.mode = modeBoard
	m.ensureColumnState()
	m.recalculateVisible()
	m.syncAllScroll()

	return m, m.saveWorkspaceCmd()
}

func (m *model) updateDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter", "q":
		m.mode = modeBoard
	case "e":
		return m.beginEditSelected()
	case "w":
		return m.openWhiteboards()
	}

	return m, nil
}

func (m *model) updateWhiteboards(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	task := m.selectedTask()
	if task == nil {
		m.mode = modeBoard
		return m, nil
	}

	switch msg.String() {
	case "esc", "q":
		m.mode = modeDetail
		m.lastErr = nil
		return m, nil
	case "up", "k":
		if m.whiteboardCursor > 0 {
			m.whiteboardCursor--
		}
		return m, nil
	case "down", "j":
		if m.whiteboardCursor < len(task.Whiteboards)-1 {
			m.whiteboardCursor++
		}
		return m, nil
	case "n":
		return m.createWhiteboard()
	case "enter", "o":
		return m.openSelectedWhiteboard()
	case "r":
		return m.beginRenameWhiteboard()
	case "x":
		return m.confirmDeleteWhiteboard()
	}

	return m, nil
}

func (m *model) updateWhiteboardRename(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeWhiteboards
		m.whiteboardInput.Blur()
		m.whiteboardRenameID = ""
		m.lastErr = nil
		return m, nil
	case "enter":
		return m.renameSelectedWhiteboard()
	}

	var cmd tea.Cmd
	m.whiteboardInput, cmd = m.whiteboardInput.Update(msg)
	return m, cmd
}

func (m *model) openWhiteboards() (tea.Model, tea.Cmd) {
	task := m.selectedTask()
	if task == nil {
		return m, nil
	}
	if len(task.Whiteboards) == 0 {
		m.whiteboardCursor = 0
	} else if m.whiteboardCursor >= len(task.Whiteboards) {
		m.whiteboardCursor = len(task.Whiteboards) - 1
	}
	m.mode = modeWhiteboards
	m.lastErr = nil
	return m, nil
}

func (m *model) createWhiteboard() (tea.Model, tea.Cmd) {
	task := m.selectedTask()
	if task == nil {
		return m, nil
	}

	name := m.board.NextWhiteboardName(task.ID)
	const newFormat = "xopp"
	ext := ".xopp"
	path := resolveWhiteboardPath(m.dataPath, m.project.Name, task.ID, name, ext)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		m.lastErr = fmt.Errorf("create whiteboard dir: %w", err)
		return m, nil
	}
	if err := createWhiteboardFile(path); err != nil {
		m.lastErr = fmt.Errorf("create whiteboard file: %w", err)
		return m, nil
	}

	whiteboard, err := m.board.AddWhiteboard(task.ID, name, path)
	if err == nil {
		whiteboard.Format = newFormat
	}
	if err != nil {
		_ = removeWhiteboardFile(path)
		m.lastErr = err
		return m, nil
	}
	m.whiteboardCursor = len(task.Whiteboards) - 1
	m.lastErr = nil
	m.lastStatus = fmt.Sprintf("created %s", whiteboard.Name)

	saveCmd := m.saveWorkspaceCmd()
	launchErr := launchWhiteboard(whiteboard.Path)
	if launchErr != nil {
		m.lastErr = fmt.Errorf("launch whiteboard: %w", launchErr)
		m.lastStatus = fmt.Sprintf("created %s", whiteboard.Name)
	}
	return m, saveCmd
}

func (m *model) openSelectedWhiteboard() (tea.Model, tea.Cmd) {
	task := m.selectedTask()
	whiteboard := m.selectedWhiteboard()
	if task == nil || whiteboard == nil {
		m.lastErr = fmt.Errorf("no whiteboard selected")
		return m, nil
	}
	path := resolveWhiteboardPath(m.dataPath, m.project.Name, task.ID, whiteboard.Name, whiteboard.Extension())
	whiteboard.Path = path
	if err := launchWhiteboard(path); err != nil {
		m.lastErr = fmt.Errorf("launch whiteboard: %w", err)
		return m, nil
	}
	m.lastErr = nil
	m.lastStatus = fmt.Sprintf("opened %s", whiteboard.Name)
	return m, nil
}

func (m *model) beginRenameWhiteboard() (tea.Model, tea.Cmd) {
	whiteboard := m.selectedWhiteboard()
	if whiteboard == nil {
		return m, nil
	}
	m.mode = modeWhiteboardRename
	m.whiteboardRenameID = whiteboard.ID
	m.whiteboardInput.SetValue(whiteboard.Name)
	m.whiteboardInput.Focus()
	m.lastErr = nil
	return m, textinput.Blink
}

func (m *model) renameSelectedWhiteboard() (tea.Model, tea.Cmd) {
	task := m.selectedTask()
	if task == nil {
		return m, nil
	}
	current, err := m.board.Whiteboard(task.ID, m.whiteboardRenameID)
	if err != nil {
		m.lastErr = err
		return m, nil
	}
	oldPath := resolveWhiteboardPath(m.dataPath, m.project.Name, task.ID, current.Name, current.Extension())
	whiteboard, err := m.board.RenameWhiteboard(task.ID, m.whiteboardRenameID, m.whiteboardInput.Value())
	if err != nil {
		m.lastErr = err
		return m, nil
	}
	newPath := resolveWhiteboardPath(m.dataPath, m.project.Name, task.ID, whiteboard.Name, whiteboard.Extension())
	if oldPath != newPath {
		if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
			m.lastErr = fmt.Errorf("prepare whiteboard dir: %w", err)
			return m, nil
		}
		if err := moveWhiteboardFile(oldPath, newPath); err != nil && !os.IsNotExist(err) {
			m.lastErr = fmt.Errorf("rename whiteboard file: %w", err)
			return m, nil
		}
	}
	whiteboard.Path = newPath
	m.mode = modeWhiteboards
	m.whiteboardRenameID = ""
	m.whiteboardInput.Blur()
	m.lastErr = nil
	m.lastStatus = fmt.Sprintf("renamed %s", whiteboard.Name)
	return m, m.saveWorkspaceCmd()
}

func (m *model) confirmDeleteWhiteboard() (tea.Model, tea.Cmd) {
	whiteboard := m.selectedWhiteboard()
	if whiteboard == nil {
		return m, nil
	}
	return m.askConfirm(
		fmt.Sprintf("Delete whiteboard %q and its file?", whiteboard.Name),
		modeWhiteboards,
		func() (tea.Model, tea.Cmd) { return m.deleteSelectedWhiteboard() },
	)
}

func (m *model) deleteSelectedWhiteboard() (tea.Model, tea.Cmd) {
	task := m.selectedTask()
	whiteboard := m.selectedWhiteboard()
	if task == nil || whiteboard == nil {
		m.mode = modeWhiteboards
		return m, nil
	}
	path := resolveWhiteboardPath(m.dataPath, m.project.Name, task.ID, whiteboard.Name, whiteboard.Extension())
	whiteboard.Path = path

	if err := removeWhiteboardFile(path); err != nil && !os.IsNotExist(err) {
		m.mode = modeWhiteboards
		m.lastErr = fmt.Errorf("delete whiteboard file: %w", err)
		return m, nil
	}

	removed, err := m.board.DeleteWhiteboard(task.ID, whiteboard.ID)
	if err != nil {
		m.mode = modeWhiteboards
		m.lastErr = err
		return m, nil
	}

	if m.whiteboardCursor >= len(task.Whiteboards) && len(task.Whiteboards) > 0 {
		m.whiteboardCursor = len(task.Whiteboards) - 1
	}
	if len(task.Whiteboards) == 0 {
		m.whiteboardCursor = 0
	}
	m.mode = modeWhiteboards
	m.lastErr = nil
	m.lastStatus = fmt.Sprintf("deleted %s", removed.Name)
	return m, m.saveWorkspaceCmd()
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
	return m, m.saveWorkspaceCmd()
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
	return m, m.saveWorkspaceCmd()
}

func (m *model) deleteSelected() (tea.Model, tea.Cmd) {
	task := m.selectedTask()
	if task == nil {
		return m, nil
	}

	if !m.board.DeleteTask(task.ID) {
		return m, nil
	}

	m.mode = modeBoard
	m.lastStatus = fmt.Sprintf("deleted %s", shortID(task.ID))
	m.lastErr = nil
	m.recalculateVisible()
	return m, m.saveWorkspaceCmd()
}

func (m *model) archiveSelected() (tea.Model, tea.Cmd) {
	task := m.selectedTask()
	if task == nil {
		m.mode = modeBoard
		return m, nil
	}

	if _, err := m.board.ArchiveTask(task.ID); err != nil {
		m.mode = modeBoard
		m.lastErr = err
		return m, nil
	}

	m.mode = modeBoard
	m.lastStatus = "archived task"
	m.lastErr = nil
	m.recalculateVisible()
	return m, m.saveWorkspaceCmd()
}

func (m *model) archiveOldDone() (tea.Model, tea.Cmd) {
	count, err := m.board.ArchiveDoneOlderThan(bulkArchiveAge)
	m.mode = modeBoard
	if err != nil {
		m.lastErr = err
		return m, nil
	}

	m.lastErr = nil
	m.lastStatus = fmt.Sprintf("archived %d done tasks", count)
	if count == 0 {
		m.lastStatus = "no done tasks old enough to archive"
		return m, nil
	}
	m.recalculateVisible()
	return m, m.saveWorkspaceCmd()
}

// filteredArchivedTasks returns archived tasks (newest first) matching the
// archive filter against title + description.
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
	if m.archiveCursor < 0 {
		m.archiveCursor = 0
	}
	if m.archiveCursor >= len(archived) {
		m.archiveCursor = len(archived) - 1
	}
	return archived[m.archiveCursor]
}

func (m *model) updateArchive(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.archiveFiltering {
		switch msg.String() {
		case "esc":
			m.archiveFiltering = false
			m.archiveFilterInput.SetValue("")
			m.archiveFilterInput.Blur()
			m.archiveCursor = 0
			return m, nil
		case "enter":
			m.archiveFiltering = false
			m.archiveFilterInput.Blur()
			return m, nil
		case "up", "down":
			// fall through to list navigation below
		default:
			var cmd tea.Cmd
			m.archiveFilterInput, cmd = m.archiveFilterInput.Update(msg)
			m.archiveCursor = 0
			return m, cmd
		}
	}

	archived := m.filteredArchivedTasks()
	if m.archiveCursor >= len(archived) {
		m.archiveCursor = max(0, len(archived)-1)
	}

	switch msg.String() {
	case "esc", "q":
		if !m.archiveFiltering && strings.TrimSpace(m.archiveFilterInput.Value()) != "" {
			m.archiveFilterInput.SetValue("")
			m.archiveCursor = 0
			return m, nil
		}
		m.mode = modeBoard
		m.lastErr = nil
		return m, nil
	case "/":
		m.archiveFiltering = true
		m.archiveFilterInput.CursorEnd()
		m.archiveFilterInput.Focus()
		return m, textinput.Blink
	case "up", "k":
		if m.archiveCursor > 0 {
			m.archiveCursor--
		}
		return m, nil
	case "down", "j":
		if m.archiveCursor < len(archived)-1 {
			m.archiveCursor++
		}
		return m, nil
	case "enter":
		if m.selectedArchivedTask() != nil {
			m.mode = modeArchiveDetail
		}
		return m, nil
	case "r":
		return m.restoreSelectedArchived()
	}

	return m, nil
}

func (m *model) restoreSelectedArchived() (tea.Model, tea.Cmd) {
	task := m.selectedArchivedTask()
	if task == nil {
		return m, nil
	}

	restored, err := m.board.RestoreTask(task.ID)
	if err != nil {
		m.lastErr = err
		return m, nil
	}

	m.lastErr = nil
	m.lastStatus = "restored task"
	m.ensureColumnState()
	m.recalculateVisible()
	m.syncScroll(restored.Status)
	if m.archiveCursor >= len(m.filteredArchivedTasks()) {
		m.archiveCursor = max(0, len(m.filteredArchivedTasks())-1)
	}
	return m, m.saveWorkspaceCmd()
}

func (m *model) updateArchiveDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter", "q":
		m.mode = modeArchive
	case "r":
		model, cmd := m.restoreSelectedArchived()
		m.mode = modeArchive
		return model, cmd
	}
	return m, nil
}

func (m *model) askConfirm(msg string, prev mode, action func() (tea.Model, tea.Cmd)) (tea.Model, tea.Cmd) {
	m.confirmMsg = msg
	m.confirmPrev = prev
	m.confirmAction = action
	m.mode = modeConfirm
	return m, nil
}

func (m *model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		return m.confirmAction()
	case "n", "N", "esc":
		m.mode = m.confirmPrev
		return m, nil
	}
	return m, nil
}

func (m *model) renderConfirmDialog() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Red).
		Render("⚠  Confirm")
	dialogWidth := m.dialogWidth(40)
	contentWidth := m.dialogContentWidth(dialogWidth, 2)

	separator := lipgloss.NewStyle().
		Foreground(theme.Red).
		Render(strings.Repeat("━", contentWidth))

	message := lipgloss.NewStyle().
		Foreground(theme.Text).
		Width(contentWidth).
		Render(m.confirmMsg)

	keyStyle := lipgloss.NewStyle().Foreground(theme.Subtext0)
	hintStyle := lipgloss.NewStyle().Foreground(theme.Surface2)
	hint := keyStyle.Render("y/enter") + hintStyle.Render(" confirm  ") +
		keyStyle.Render("n/esc") + hintStyle.Render(" cancel")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		separator,
		"",
		message,
		"",
		hint,
	)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Red).
		Padding(1, 2).
		Width(dialogWidth).
		Render(content)
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

// cardColumnWidth returns the width parameter passed to renderColumn for the
// given column index. It mirrors the width calculation in renderBoard.
func (m *model) cardColumnWidth(colIndex int) int {
	statuses := m.board.Statuses()
	if len(statuses) == 0 {
		return max(1, m.width-4)
	}
	if m.useCompactBoardLayout() {
		return max(1, m.width-4)
	}
	availableWidth := max(0, m.width-4-(boardGap*(len(statuses)-1)))
	if availableWidth <= 0 {
		return max(1, m.width-4)
	}
	columnWidth := max(1, availableWidth/len(statuses))
	extraWidth := max(0, availableWidth-(columnWidth*len(statuses)))
	if colIndex < extraWidth {
		columnWidth++
	}
	return columnWidth
}

// visibleTaskCount returns how many task cards fit in the column body starting
// from the given scroll offset, by measuring actual rendered card heights.
func (m *model) visibleTaskCount(status domain.Status, scroll int) int {
	bodyHeight := max(1, m.columnHeight()-5)
	ids := m.visible[status]
	if len(ids) == 0 {
		return 1
	}
	available := bodyHeight
	if scroll > 0 {
		available-- // "▲ more" indicator
	}
	innerWidth := max(1, m.cardColumnWidth(m.activeColumn)-4)
	count := 0
	for i := scroll; i < len(ids); i++ {
		task := m.board.Tasks[ids[i]]
		if task == nil {
			continue
		}
		card := m.renderTaskCard(task, innerWidth, false, theme.Mauve)
		h := lipgloss.Height(card)
		// Reserve 1 line for "▼ N more" if there are more tasks after this.
		reserve := 0
		if i+1 < len(ids) {
			reserve = 1
		}
		if available-h-reserve < 0 && count > 0 {
			break
		}
		available -= h
		count++
	}
	return max(1, count)
}

func (m *model) syncScroll(status domain.Status) {
	selected := m.selected[status]
	scroll := m.scroll[status]
	if selected < scroll {
		scroll = selected
	}

	// Advance scroll until the selected task is visible.
	for scroll <= selected {
		rows := m.visibleTaskCount(status, scroll)
		if selected < scroll+rows {
			break
		}
		scroll++
	}

	if scroll < 0 {
		scroll = 0
	}
	maxScroll := max(0, len(m.visible[status])-1)
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

func (m *model) selectedWhiteboard() *domain.Whiteboard {
	task := m.selectedTask()
	if task == nil || len(task.Whiteboards) == 0 {
		return nil
	}
	if m.whiteboardCursor < 0 {
		m.whiteboardCursor = 0
	}
	if m.whiteboardCursor >= len(task.Whiteboards) {
		m.whiteboardCursor = len(task.Whiteboards) - 1
	}
	return &task.Whiteboards[m.whiteboardCursor]
}

// ─── Header ──────────────────────────────────────────────────────────────────

func (m *model) renderHeader() string {
	logo := lipgloss.NewStyle().Bold(true).Foreground(theme.Mauve).Render("\u25c6")
	titleText := " kanban"
	if m.project != nil {
		titleText += " / " + m.project.Name
	}
	title := lipgloss.NewStyle().Bold(true).Foreground(theme.Text).Render(titleText)
	compact := m.useCompactBoardLayout()

	total := m.board.ActiveTaskCount()
	done := m.board.Count(domain.StatusDone)
	inProgress := m.board.Count(domain.StatusInProgress)
	if m.onDailyBoard() {
		inProgress = m.board.Count(domain.StatusActive)
		done = 0
	}

	// Visual progress bar
	barWidth := min(20, max(4, m.width/4))
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
	if m.onDailyBoard() && total > 0 {
		dot := lipgloss.NewStyle().Foreground(theme.Surface2).Render(" · ")
		stats = lipgloss.NewStyle().Foreground(theme.Blue).Render(fmt.Sprintf("%d waiting", m.board.Count(domain.StatusWaiting))) + dot +
			lipgloss.NewStyle().Foreground(theme.Peach).Render(fmt.Sprintf("%d active", m.board.Count(domain.StatusActive))) + dot +
			lipgloss.NewStyle().Foreground(theme.Mauve).Render(fmt.Sprintf("%d next", m.board.Count(domain.StatusNext)))
	} else if total > 0 {
		stats = lipgloss.NewStyle().Foreground(theme.Peach).Render(fmt.Sprintf("%d active", inProgress)) +
			lipgloss.NewStyle().Foreground(theme.Surface2).Render(" \u00b7 ") +
			lipgloss.NewStyle().Foreground(theme.Green).Render(fmt.Sprintf("%d done", done)) +
			lipgloss.NewStyle().Foreground(theme.Surface2).Render(" \u00b7 ") +
			lipgloss.NewStyle().Foreground(theme.Subtext0).Render(fmt.Sprintf("%d total", total))
	}
	if compact {
		stats = lipgloss.NewStyle().Foreground(theme.Peach).Render(m.compactColumnIndicator())
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
	gap := boardGap
	statuses := m.board.Statuses()
	if len(statuses) == 0 {
		return ""
	}

	if m.useCompactBoardLayout() {
		if m.activeColumn < 0 || m.activeColumn >= len(statuses) {
			m.activeColumn = 0
		}
		width := max(1, m.width-4)
		return lipgloss.NewStyle().
			Padding(1, 2, 0, 2).
			Render(m.renderColumn(statuses[m.activeColumn], true, width))
	}

	availableWidth := max(0, m.width-4-(gap*(len(statuses)-1)))
	if availableWidth <= 0 {
		width := max(1, m.width-4)
		return lipgloss.NewStyle().
			Padding(1, 2, 0, 2).
			Render(m.renderColumn(statuses[m.activeColumn], true, width))
	}

	columnWidth := max(1, availableWidth/len(statuses))
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

	colHeight := m.columnHeight()
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
		Foreground(theme.Text).
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

	body := make([]string, 0)
	usedHeight := 0

	if scroll > 0 {
		indicator := lipgloss.NewStyle().Foreground(theme.Overlay0).Align(lipgloss.Center).Width(innerWidth).Render("\u25b2 more")
		body = append(body, indicator)
		usedHeight += lipgloss.Height(indicator)
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

	end := scroll
	for i := scroll; i < len(ids); i++ {
		task := m.board.Tasks[ids[i]]
		if task == nil {
			end = i + 1
			continue
		}
		card := m.renderTaskCard(task, innerWidth, active && i == m.selected[status], accent)
		cardH := lipgloss.Height(card)
		// Reserve 1 line for "▼ N more" if there are more tasks after this one.
		reserve := 0
		if i+1 < len(ids) {
			reserve = 1
		}
		if usedHeight+cardH+reserve > bodyHeight && end > scroll {
			break
		}
		body = append(body, card)
		usedHeight += cardH
		end = i + 1
	}

	if hidden := len(ids) - end; hidden > 0 {
		body = append(body,
			lipgloss.NewStyle().Foreground(theme.Overlay0).Align(lipgloss.Center).Width(innerWidth).Render(fmt.Sprintf("\u25bc %d more", hidden)),
		)
	}

	bodyView := lipgloss.NewStyle().MaxHeight(bodyHeight).Render(strings.Join(body, "\n"))
	return columnStyle.Render(lipgloss.JoinVertical(lipgloss.Left, header, separator, bodyView))
}

func (m *model) renderTaskCard(task *domain.Task, width int, selected bool, accent lipgloss.TerminalColor) string {
	cardWidth := width
	if cardWidth < 1 {
		cardWidth = 1
	}
	innerWidth := cardWidth - 4
	if innerWidth < 1 {
		innerWidth = 1
	}

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
	compact := m.useCompactBoardLayout()

	if m.onDailyBoard() {
		keyStyle := lipgloss.NewStyle().Foreground(theme.Subtext1).Bold(true)
		descStyle := lipgloss.NewStyle().Foreground(theme.Overlay0)
		sep := lipgloss.NewStyle().Foreground(theme.Surface1).Render("  │  ")

		content = keyStyle.Render("h/l") + descStyle.Render(" column") + sep +
			keyStyle.Render("j/k") + descStyle.Render(" task") + sep +
			keyStyle.Render("n") + descStyle.Render(" new") + sep +
			keyStyle.Render("[/]") + descStyle.Render(" move") + sep +
			keyStyle.Render("space") + descStyle.Render(" done") + sep +
			keyStyle.Render("D") + descStyle.Render(" back")
		if !compact {
			content += sep +
				keyStyle.Render("X") + descStyle.Render(" clear board") + sep +
				keyStyle.Render("z") + descStyle.Render(" done items") + sep +
				keyStyle.Render("q") + descStyle.Render(" quit")
		}

		return lipgloss.NewStyle().
			Width(m.width).
			Padding(0, 2, 1, 2).
			Foreground(theme.Subtext0).
			Render(content)
	}

	if compact {
		keyStyle := lipgloss.NewStyle().Foreground(theme.Subtext1).Bold(true)
		descStyle := lipgloss.NewStyle().Foreground(theme.Overlay0)
		sepStyle := lipgloss.NewStyle().Foreground(theme.Surface1)
		sep := sepStyle.Render("  \u2502  ")

		content = keyStyle.Render("h/l") + descStyle.Render(" column") + sep +
			keyStyle.Render("j/k") + descStyle.Render(" task") + sep +
			keyStyle.Render("n") + descStyle.Render(" new") + sep +
			keyStyle.Render("p") + descStyle.Render(" projects") + sep +
			keyStyle.Render("D") + descStyle.Render(" daily") + sep +
			keyStyle.Render("/") + descStyle.Render(" search") + sep +
			keyStyle.Render("\u23ce") + descStyle.Render(" open") + sep +
			keyStyle.Render("?") + descStyle.Render(" help") + sep +
			keyStyle.Render("q") + descStyle.Render(" quit")

		if m.showHelp {
			content += sep +
				keyStyle.Render("[/]") + descStyle.Render(" move") + sep +
				keyStyle.Render("H/L") + descStyle.Render(" reorder col") + sep +
				keyStyle.Render("e") + descStyle.Render(" edit") + sep +
				keyStyle.Render("x") + descStyle.Render(" delete")
		}
	} else if m.showHelp {
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
			keyStyle.Render("e") + descStyle.Render(" edit") + sep +
			keyStyle.Render("c") + descStyle.Render(" column") + sep +
			keyStyle.Render("p") + descStyle.Render(" projects") + sep +
			keyStyle.Render("D") + descStyle.Render(" daily") + sep +
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
	isEditing := m.editingTaskID != ""
	titleText := "New Task"
	saveHint := "save"
	if isEditing {
		titleText = "Edit Task"
		saveHint = "update"
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Mauve).
		Render(fmt.Sprintf("\u25c6  %s", titleText))

	titleLabel := lipgloss.NewStyle().
		Foreground(theme.Overlay0).
		Bold(true).
		Render("TITLE")
	descLabel := lipgloss.NewStyle().
		Foreground(theme.Overlay0).
		Bold(true).
		Render("DESCRIPTION")
	dialogWidth := m.dialogWidth(createDialogMaxWidth)
	contentWidth := m.dialogContentWidth(dialogWidth, createDialogPadding)

	separator := lipgloss.NewStyle().
		Foreground(theme.Mauve).
		Render(strings.Repeat("\u2501", contentWidth))

	hintStyle := lipgloss.NewStyle().Foreground(theme.Surface2)
	keyStyle := lipgloss.NewStyle().Foreground(theme.Subtext0)
	var modeHint string
	switch {
	case m.vimReplace:
		modeHint = lipgloss.NewStyle().Bold(true).Foreground(theme.Red).Render("REPLACE") + "  "
	case m.vimVisual != nil:
		modeHint = lipgloss.NewStyle().Bold(true).Foreground(theme.Pink).Render("VISUAL") + "  "
	case m.vimNormal && m.vim.pending():
		modeHint = lipgloss.NewStyle().Bold(true).Foreground(theme.Yellow).Render("NORMAL·") + "  "
	case m.vimNormal:
		modeHint = lipgloss.NewStyle().Bold(true).Foreground(theme.Green).Render("NORMAL") + "  "
	default:
		modeHint = lipgloss.NewStyle().Bold(true).Foreground(theme.Blue).Render("INSERT") + "  "
	}
	var hint string
	if m.vimStatus != "" {
		// Transient feedback takes over the hint line (vim-style) so the
		// dialog height never changes; hints return on next keypress.
		hint = modeHint + lipgloss.NewStyle().Foreground(theme.Yellow).Render(m.vimStatus)
	} else {
		hint = modeHint +
			keyStyle.Render("tab") + hintStyle.Render(" switch  ") +
			keyStyle.Render(":w") + hintStyle.Render(" "+saveHint+"  ") +
			keyStyle.Render(":wq") + hintStyle.Render(" "+saveHint+" & close  ") +
			keyStyle.Render("ctrl+e") + hintStyle.Render(" editor  ") +
			keyStyle.Render("esc") + hintStyle.Render(func() string {
			if m.vimNormal {
				return " close"
			}
			return " normal"
		}())
	}
	hint = lipgloss.NewStyle().MaxWidth(contentWidth).Render(hint)

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
		m.renderCreateInput(m.titleInput.View(), m.titleInput.Value(), m.titleInput.Focused()),
		"",
		descLabel,
		m.renderCreateInput(m.descInput.View(), m.descInput.Value(), m.descInput.Focused()),
	)
	if errView != "" {
		content = lipgloss.JoinVertical(lipgloss.Left, content, "", errView)
	}
	content = lipgloss.JoinVertical(lipgloss.Left, content, "", hint)

	return lipgloss.NewStyle().
		Width(dialogWidth).
		Padding(1, 3).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Mauve).
		Background(theme.Base).
		Render(content)
}

func (m *model) renderCreateInput(view, value string, focused bool) string {
	if m.vimVisual == nil || !focused {
		return view
	}
	start, end := m.vimVisual.rangeBounds(len([]rune(value)))
	selected := string([]rune(value)[start:end])
	if selected == "" {
		return view
	}
	return highlightVisibleText(view, selected)
}

func highlightVisibleText(view, selected string) string {
	visible := ansiStripRe.ReplaceAllString(view, "")
	start := strings.Index(visible, selected)
	if start < 0 {
		return view
	}
	end := start + len(selected)
	var out strings.Builder
	visibleByte := 0
	for i := 0; i < len(view); {
		if view[i] == '\x1b' {
			if match := ansiStripRe.FindString(view[i:]); strings.HasPrefix(view[i:], match) {
				out.WriteString(match)
				i += len(match)
				continue
			}
		}
		if visibleByte == start {
			out.WriteString("\x1b[7m")
		}
		if visibleByte == end {
			out.WriteString("\x1b[27m")
		}
		_, size := utf8.DecodeRuneInString(view[i:])
		out.WriteString(view[i : i+size])
		i += size
		visibleByte += size
	}
	if visibleByte == end {
		out.WriteString("\x1b[27m")
	}
	return out.String()
}

func (m *model) renderSearchDialog() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Blue).
		Render("\u2315  Search")
	dialogWidth := m.dialogWidth(searchDialogMaxWidth)
	contentWidth := m.dialogContentWidth(dialogWidth, searchDialogPadding)

	separator := lipgloss.NewStyle().
		Foreground(theme.Blue).
		Render(strings.Repeat("\u2501", contentWidth))

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
		Width(dialogWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Blue).
		Background(theme.Base).
		Render(content)
}

func (m *model) renderDetailDialog() string {
	return m.renderTaskDetail(m.selectedTask(), false)
}

// renderTaskDetail renders the detail dialog for an explicit task so archived
// tasks (which are never the active-board selection) can reuse it. archived
// selects the read-only hint set.
func (m *model) renderTaskDetail(task *domain.Task, archived bool) string {
	if task == nil {
		return ""
	}

	accent := statusAccent(task.Status)

	titleView := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Text).
		Render(task.Title)

	// Outline badge rather than a filled one: the palette is pure ANSI, so the
	// RGB behind accent/Mantle comes from the user's terminal theme and a
	// filled badge can land text on a near-matching background. An accent
	// foreground on the dialog background is legible in every theme.
	statusBadge := lipgloss.NewStyle().
		Foreground(accent).
		Bold(true).
		Render(statusIcon(task.Status) + " " + task.Status.Title())
	dialogWidth := m.dialogWidth(detailDialogMaxWidth)
	contentWidth := m.dialogContentWidth(dialogWidth, defaultDialogPadding)

	separator := lipgloss.NewStyle().
		Foreground(accent).
		Render(strings.Repeat("\u2501", contentWidth))

	labelWidth := 12
	if contentWidth < 24 {
		labelWidth = max(1, contentWidth/3)
	}
	if labelWidth > contentWidth {
		labelWidth = contentWidth
	}
	labelStyle := lipgloss.NewStyle().Foreground(theme.Overlay0).Width(labelWidth)
	valueStyle := lipgloss.NewStyle().Foreground(theme.Subtext1)

	metaRows := []string{
		lipgloss.JoinHorizontal(lipgloss.Top, labelStyle.Render("ID"), valueStyle.Render(task.ID)),
		lipgloss.JoinHorizontal(lipgloss.Top, labelStyle.Render("Status"), statusBadge),
		lipgloss.JoinHorizontal(lipgloss.Top, labelStyle.Render("Whiteboards"), valueStyle.Render(fmt.Sprintf("%d", len(task.Whiteboards)))),
		lipgloss.JoinHorizontal(lipgloss.Top, labelStyle.Render("Created"), valueStyle.Render(task.CreatedAt.Local().Format("02 Jan 2006 15:04"))),
		lipgloss.JoinHorizontal(lipgloss.Top, labelStyle.Render("Updated"), valueStyle.Render(task.UpdatedAt.Local().Format("02 Jan 2006 15:04"))),
	}
	if task.Archived() {
		metaRows = append(metaRows,
			lipgloss.JoinHorizontal(lipgloss.Top, labelStyle.Render("Archived"), valueStyle.Render(task.ArchivedAt.Local().Format("02 Jan 2006 15:04"))),
		)
	}

	description := strings.TrimSpace(task.Description)
	descView := ""
	if description != "" {
		descView = lipgloss.NewStyle().
			Width(max(1, contentWidth)).
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
	hint := keyStyle.Render("e") + hintStyle.Render(" edit  ") +
		keyStyle.Render("w") + hintStyle.Render(" whiteboards  ") +
		keyStyle.Render("esc") + hintStyle.Render(" close")
	if archived {
		hint = keyStyle.Render("r") + hintStyle.Render(" restore  ") +
			keyStyle.Render("esc") + hintStyle.Render(" back")
	}

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
		Width(dialogWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accent).
		Background(theme.Base).
		Render(content)
}

func (m *model) renderWhiteboardsDialog() string {
	task := m.selectedTask()
	if task == nil {
		return ""
	}

	dialogWidth := m.dialogWidth(projectDialogMaxWidth + 8)
	contentWidth := m.dialogContentWidth(dialogWidth, defaultDialogPadding)
	title := lipgloss.NewStyle().Bold(true).Foreground(theme.Blue).Render("Whiteboards")
	subtitle := lipgloss.NewStyle().Foreground(theme.Subtext0).Render(truncate(task.Title, max(1, contentWidth)))
	separator := lipgloss.NewStyle().Foreground(theme.Blue).Render(strings.Repeat("\u2501", contentWidth))

	rows := make([]string, 0, len(task.Whiteboards))
	for i, whiteboard := range task.Whiteboards {
		prefix := "  "
		if i == m.whiteboardCursor {
			prefix = lipgloss.NewStyle().Foreground(theme.Mauve).Render("\u25b8 ")
		}
		name := lipgloss.NewStyle().Foreground(theme.Text).Render(truncate(whiteboard.Name, max(1, contentWidth/2)))
		path := lipgloss.NewStyle().Foreground(theme.Subtext0).Render(truncate(whiteboard.Path, max(1, contentWidth-lipgloss.Width(prefix)-6)))
		meta := lipgloss.NewStyle().Foreground(theme.Overlay0).Render(relativeTime(whiteboard.UpdatedAt))
		row := lipgloss.JoinVertical(lipgloss.Left, lipgloss.JoinHorizontal(lipgloss.Center, prefix, name, spacer(max(1, contentWidth-lipgloss.Width(prefix)-lipgloss.Width(name)-lipgloss.Width(meta))), meta), spacer(0)+path)
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		rows = append(rows, lipgloss.NewStyle().Foreground(theme.Surface2).Italic(true).Render("No whiteboards yet"))
	}

	errView := ""
	if m.lastErr != nil {
		errView = lipgloss.NewStyle().Foreground(theme.Red).Render("\u2717 " + m.lastErr.Error())
	}
	keyStyle := lipgloss.NewStyle().Foreground(theme.Subtext0)
	hintStyle := lipgloss.NewStyle().Foreground(theme.Surface2)
	hint := keyStyle.Render("enter/o") + hintStyle.Render(" open  ") +
		keyStyle.Render("n") + hintStyle.Render(" new  ") +
		keyStyle.Render("r") + hintStyle.Render(" rename  ") +
		keyStyle.Render("x") + hintStyle.Render(" delete  ") +
		keyStyle.Render("esc") + hintStyle.Render(" close")

	content := lipgloss.JoinVertical(lipgloss.Left, title, subtitle, separator, "", strings.Join(rows, "\n\n"))
	if errView != "" {
		content = lipgloss.JoinVertical(lipgloss.Left, content, "", errView)
	}
	content = lipgloss.JoinVertical(lipgloss.Left, content, "", hint)

	return lipgloss.NewStyle().
		Width(dialogWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Blue).
		Background(theme.Base).
		Render(content)
}

func (m *model) renderWhiteboardRenameDialog() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(theme.Mauve).Render("Rename Whiteboard")
	dialogWidth := m.dialogWidth(projectDialogMaxWidth)
	contentWidth := m.dialogContentWidth(dialogWidth, defaultDialogPadding)
	separator := lipgloss.NewStyle().Foreground(theme.Mauve).Render(strings.Repeat("\u2501", contentWidth))
	label := lipgloss.NewStyle().Foreground(theme.Overlay0).Bold(true).Render("NAME")

	errView := ""
	if m.lastErr != nil {
		errView = lipgloss.NewStyle().Foreground(theme.Red).Render("\u2717 " + m.lastErr.Error())
	}
	keyStyle := lipgloss.NewStyle().Foreground(theme.Subtext0)
	hintStyle := lipgloss.NewStyle().Foreground(theme.Surface2)
	hint := keyStyle.Render("enter") + hintStyle.Render(" rename  ") +
		keyStyle.Render("esc") + hintStyle.Render(" cancel")

	content := lipgloss.JoinVertical(lipgloss.Left, title, separator, "", label, m.whiteboardInput.View())
	if errView != "" {
		content = lipgloss.JoinVertical(lipgloss.Left, content, "", errView)
	}
	content = lipgloss.JoinVertical(lipgloss.Left, content, "", hint)

	return lipgloss.NewStyle().
		Width(dialogWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Mauve).
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
	dialogWidth := m.dialogWidth(columnDialogMaxWidth)
	contentWidth := m.dialogContentWidth(dialogWidth, defaultDialogPadding)

	separator := lipgloss.NewStyle().
		Foreground(theme.Mauve).
		Render(strings.Repeat("\u2501", contentWidth))

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
		Width(dialogWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Mauve).
		Background(theme.Base).
		Render(content)
}

func (m *model) renderProjectsDialog() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(theme.Blue).Render("Projects")
	dialogWidth := m.dialogWidth(projectDialogMaxWidth)
	contentWidth := m.dialogContentWidth(dialogWidth, defaultDialogPadding)
	separator := lipgloss.NewStyle().Foreground(theme.Blue).Render(strings.Repeat("\u2501", contentWidth))

	projects := m.filteredProjects()
	cursor := min(m.projectCursor, max(0, len(projects)-1))

	rows := make([]string, 0, len(projects))
	for i, project := range projects {
		prefix := "  "
		if i == cursor {
			prefix = lipgloss.NewStyle().Foreground(theme.Mauve).Render("\u25b8 ")
		}

		name := project.Name
		if project.ID == m.workspace.ActiveProjectID {
			name += lipgloss.NewStyle().Foreground(theme.Green).Render("  active")
		}
		count := lipgloss.NewStyle().Foreground(theme.Subtext0).Render(fmt.Sprintf("%d tasks", project.Board.ActiveTaskCount()))
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Center, prefix, truncate(name, max(1, contentWidth-12)), spacer(max(1, contentWidth-lipgloss.Width(prefix)-lipgloss.Width(name)-lipgloss.Width(count))), count))
	}

	if len(rows) == 0 {
		emptyText := "No projects"
		if strings.TrimSpace(m.projectFilterInput.Value()) != "" {
			emptyText = "No matching projects"
		}
		rows = append(rows, lipgloss.NewStyle().Foreground(theme.Surface2).Italic(true).Render(emptyText))
	}

	filterView := m.projectFilterInput.View()

	errView := ""
	if m.lastErr != nil {
		errView = lipgloss.NewStyle().Foreground(theme.Red).Render("\u2717 " + m.lastErr.Error())
	}

	keyStyle := lipgloss.NewStyle().Foreground(theme.Subtext0)
	hintStyle := lipgloss.NewStyle().Foreground(theme.Surface2)
	hint := keyStyle.Render("enter") + hintStyle.Render(" open  ") +
		keyStyle.Render("n") + hintStyle.Render(" new  ") +
		keyStyle.Render("e") + hintStyle.Render(" rename  ") +
		keyStyle.Render("x") + hintStyle.Render(" delete  ") +
		keyStyle.Render("/") + hintStyle.Render(" filter  ") +
		keyStyle.Render("esc") + hintStyle.Render(" close")
	if m.projectFiltering {
		hint = keyStyle.Render("enter") + hintStyle.Render(" open  ") +
			keyStyle.Render("↑↓") + hintStyle.Render(" move  ") +
			keyStyle.Render("esc") + hintStyle.Render(" clear")
	}

	content := lipgloss.JoinVertical(lipgloss.Left, title, separator, "", filterView, "", strings.Join(rows, "\n"))
	if errView != "" {
		content = lipgloss.JoinVertical(lipgloss.Left, content, "", errView)
	}
	content = lipgloss.JoinVertical(lipgloss.Left, content, "", hint)

	return lipgloss.NewStyle().Width(dialogWidth).Padding(1, 2).Border(lipgloss.RoundedBorder()).BorderForeground(theme.Blue).Background(theme.Base).Render(content)
}

func (m *model) renderArchiveDialog() string {
	projectName := ""
	if m.project != nil {
		projectName = m.project.Name
	}
	title := lipgloss.NewStyle().Bold(true).Foreground(theme.Lavender).Render("Archive - " + projectName)
	dialogWidth := m.dialogWidth(projectDialogMaxWidth + 8)
	contentWidth := m.dialogContentWidth(dialogWidth, defaultDialogPadding)
	separator := lipgloss.NewStyle().Foreground(theme.Lavender).Render(strings.Repeat("━", contentWidth))

	archived := m.filteredArchivedTasks()
	cursor := min(m.archiveCursor, max(0, len(archived)-1))

	rows := make([]string, 0, len(archived))
	for i, task := range archived {
		prefix := "  "
		if i == cursor {
			prefix = lipgloss.NewStyle().Foreground(theme.Mauve).Render("▸ ")
		}
		name := lipgloss.NewStyle().Foreground(theme.Text).Render(truncate(task.Title, max(1, contentWidth-24)))
		meta := lipgloss.NewStyle().Foreground(theme.Overlay0).Render(
			task.ArchivedAt.Local().Format("02 Jan 2006") + " · " + task.ArchivedFrom.Title(),
		)
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Center, prefix, name,
			spacer(max(1, contentWidth-lipgloss.Width(prefix)-lipgloss.Width(name)-lipgloss.Width(meta))), meta))
	}
	if len(rows) == 0 {
		emptyText := "No archived tasks"
		if strings.TrimSpace(m.archiveFilterInput.Value()) != "" {
			emptyText = "No matching archived tasks"
		}
		rows = append(rows, lipgloss.NewStyle().Foreground(theme.Surface2).Italic(true).Render(emptyText))
	}

	filterView := m.archiveFilterInput.View()

	errView := ""
	if m.lastErr != nil {
		errView = lipgloss.NewStyle().Foreground(theme.Red).Render("✗ " + m.lastErr.Error())
	}

	keyStyle := lipgloss.NewStyle().Foreground(theme.Subtext0)
	hintStyle := lipgloss.NewStyle().Foreground(theme.Surface2)
	hint := keyStyle.Render("enter") + hintStyle.Render(" details  ") +
		keyStyle.Render("r") + hintStyle.Render(" restore  ") +
		keyStyle.Render("/") + hintStyle.Render(" filter  ") +
		keyStyle.Render("esc") + hintStyle.Render(" close")
	if m.archiveFiltering {
		hint = keyStyle.Render("enter") + hintStyle.Render(" apply  ") +
			keyStyle.Render("↑↓") + hintStyle.Render(" move  ") +
			keyStyle.Render("esc") + hintStyle.Render(" clear")
	}

	content := lipgloss.JoinVertical(lipgloss.Left, title, separator, "", filterView, "", strings.Join(rows, "\n"))
	if errView != "" {
		content = lipgloss.JoinVertical(lipgloss.Left, content, "", errView)
	}
	content = lipgloss.JoinVertical(lipgloss.Left, content, "", hint)

	return lipgloss.NewStyle().Width(dialogWidth).Padding(1, 2).Border(lipgloss.RoundedBorder()).BorderForeground(theme.Lavender).Background(theme.Base).Render(content)
}

func (m *model) renderProjectEditDialog() string {
	isRename := m.projectDraft != ""
	titleText := "New Project"
	actionText := "create"
	if isRename {
		titleText = "Rename Project"
		actionText = "rename"
	}

	title := lipgloss.NewStyle().Bold(true).Foreground(theme.Mauve).Render(titleText)
	dialogWidth := m.dialogWidth(projectDialogMaxWidth)
	contentWidth := m.dialogContentWidth(dialogWidth, defaultDialogPadding)
	separator := lipgloss.NewStyle().Foreground(theme.Mauve).Render(strings.Repeat("\u2501", contentWidth))
	label := lipgloss.NewStyle().Foreground(theme.Overlay0).Bold(true).Render("NAME")
	errView := ""
	if m.lastErr != nil {
		errView = lipgloss.NewStyle().Foreground(theme.Red).Render("\u2717 " + m.lastErr.Error())
	}
	keyStyle := lipgloss.NewStyle().Foreground(theme.Subtext0)
	hintStyle := lipgloss.NewStyle().Foreground(theme.Surface2)
	hint := keyStyle.Render("enter") + hintStyle.Render(" "+actionText+"  ") + keyStyle.Render("esc") + hintStyle.Render(" cancel")

	content := lipgloss.JoinVertical(lipgloss.Left, title, separator, "", label, m.projectInput.View())
	if errView != "" {
		content = lipgloss.JoinVertical(lipgloss.Left, content, "", errView)
	}
	content = lipgloss.JoinVertical(lipgloss.Left, content, "", hint)

	return lipgloss.NewStyle().Width(dialogWidth).Padding(1, 2).Border(lipgloss.RoundedBorder()).BorderForeground(theme.Mauve).Background(theme.Base).Render(content)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func (m *model) syncResponsiveLayout() {
	if m.width <= 0 || m.height <= 0 {
		return
	}

	createContentWidth := m.dialogContentWidth(
		m.dialogWidth(createDialogMaxWidth),
		createDialogPadding,
	)
	if createContentWidth > 0 {
		m.titleInput.Width = createContentWidth
		m.descInput.SetWidth(createContentWidth)
	}

	searchContentWidth := m.dialogContentWidth(
		m.dialogWidth(searchDialogMaxWidth),
		searchDialogPadding,
	)
	m.searchInput.Width = min(max(1, searchContentWidth), maxModalInputWidth)
	m.columnInput.Width = min(max(1, searchContentWidth), maxModalInputWidth)
	m.projectInput.Width = min(max(1, searchContentWidth), maxModalInputWidth)
	m.whiteboardInput.Width = min(max(1, searchContentWidth), maxModalInputWidth)

	if createContentWidth > 0 {
		height := m.height - 16
		if height < minDescriptionHeight {
			height = minDescriptionHeight
		}
		if height > maxDescriptionHeight {
			height = maxDescriptionHeight
		}
		m.descInput.SetHeight(height)
	}
}

func (m *model) useCompactBoardLayout() bool {
	statuses := m.board.Statuses()
	if len(statuses) <= 1 || m.width <= 0 {
		return false
	}

	availableWidth := max(0, m.width-4-(boardGap*(len(statuses)-1)))
	if availableWidth <= 0 {
		return true
	}

	if availableWidth/len(statuses) < compactColumnWidth {
		return true
	}

	if m.width < compactBoardBreakpoint {
		return true
	}

	return false
}

func (m *model) dialogWidth(maxWidth int) int {
	if m.width <= 0 {
		return maxWidth
	}

	available := max(1, m.width-4)
	if available > maxWidth {
		return maxWidth
	}
	return available
}

func (m *model) dialogContentWidth(dialogWidth, padding int) int {
	content := dialogWidth - 2*(padding+1)
	if content < 1 {
		return 1
	}
	return content
}

func (m *model) columnHeight() int {
	return max(6, m.height-10)
}

func (m *model) compactColumnIndicator() string {
	statuses := m.board.Statuses()
	if len(statuses) == 0 {
		return "0/0"
	}
	if m.activeColumn < 0 || m.activeColumn >= len(statuses) {
		m.activeColumn = 0
	}
	return fmt.Sprintf("%d/%d %s", m.activeColumn+1, len(statuses), statuses[m.activeColumn].Title())
}

func (m *model) taskRows() int {
	bodyHeight := max(1, m.columnHeight()-5)
	rows := bodyHeight / cardSlotHeight
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

func (m *model) activeProjectIndex() int {
	if m.workspace == nil {
		return 0
	}
	return m.regularProjectIndex(m.workspace.ActiveProjectID)
}

// regularProjectIndex returns the position of id in the project list shown by
// the project manager, which excludes the daily board.
func (m *model) regularProjectIndex(id string) int {
	for i, project := range m.workspace.RegularProjects() {
		if project.ID == id {
			return i
		}
	}
	return 0
}

func (m *model) activateProject(id string) {
	if m.workspace == nil || !m.workspace.SetActiveProject(id) {
		return
	}

	m.project = m.workspace.ActiveProject()
	if m.project == nil {
		return
	}
	if !m.project.Daily {
		m.prevProjectID = m.project.ID
	}
	m.board = m.project.Board
	m.activeColumn = 0
	m.filter = ""
	m.filterDraft = ""
	m.searchInput.SetValue("")
	m.ensureColumnState()
	m.recalculateVisible()
	m.syncAllScroll()
}

// onDailyBoard reports whether the daily board is the active project.
func (m *model) onDailyBoard() bool {
	return m.project != nil && m.project.Daily
}

// toggleDaily jumps into the daily board, or back to the last regular project
// when it is already open.
func (m *model) toggleDaily() (tea.Model, tea.Cmd) {
	if m.workspace == nil {
		return m, nil
	}

	if m.onDailyBoard() {
		target := m.prevProjectID
		if project := m.workspace.ProjectByID(target); project == nil || project.Daily {
			target = ""
			if regular := m.workspace.RegularProjects(); len(regular) > 0 {
				target = regular[0].ID
			}
		}
		if target == "" {
			return m, nil
		}
		m.activateProject(target)
		m.lastErr = nil
		m.lastStatus = fmt.Sprintf("opened project %s", m.project.Name)
		return m, nil
	}

	daily := m.workspace.DailyProject()
	if daily == nil {
		m.lastErr = fmt.Errorf("daily board not found")
		return m, nil
	}

	m.activateProject(daily.ID)
	m.lastErr = nil
	m.lastStatus = "daily board"
	return m, nil
}

// markDailyDone archives the selected daily task so it stops showing on the
// board but stays available in the archive view.
func (m *model) markDailyDone() (tea.Model, tea.Cmd) {
	task := m.selectedTask()
	if task == nil {
		return m, nil
	}

	if _, err := m.board.ArchiveTask(task.ID); err != nil {
		m.lastErr = err
		return m, nil
	}

	m.lastErr = nil
	m.lastStatus = fmt.Sprintf("done: %s", singleLine(truncate(task.Title, 30)))
	m.recalculateVisible()
	return m, m.saveWorkspaceCmd()
}

// clearDailyBoard deletes every task on the daily board, archived ones
// included.
func (m *model) clearDailyBoard() (tea.Model, tea.Cmd) {
	m.mode = modeBoard
	if !m.onDailyBoard() {
		return m, nil
	}

	count := m.board.Clear()
	m.lastErr = nil
	m.lastStatus = fmt.Sprintf("cleared %d task(s)", count)
	m.recalculateVisible()
	return m, m.saveWorkspaceCmd()
}

func (m *model) switchProject(id string) (tea.Model, tea.Cmd) {
	project := m.workspace.ProjectByID(id)
	if project == nil {
		m.lastErr = fmt.Errorf("project not found")
		return m, nil
	}

	m.activateProject(id)
	m.mode = modeBoard
	m.lastErr = nil
	m.lastStatus = fmt.Sprintf("opened project %s", project.Name)
	return m, nil
}

func (m *model) saveWorkspaceCmd() tea.Cmd {
	if m.project != nil {
		m.project.Touch()
	}
	return saveWorkspaceCmd(m.store, m.workspace.Clone())
}

func saveWorkspaceCmd(workspaceStore store.WorkspaceStore, workspace *domain.Workspace) tea.Cmd {
	return func() tea.Msg {
		return saveFinishedMsg{err: workspaceStore.Save(workspace)}
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

func statusAccent(status domain.Status) lipgloss.TerminalColor {
	switch status {
	case domain.StatusBacklog:
		return theme.Blue
	case domain.StatusInProgress:
		return theme.Peach
	case domain.StatusDone:
		return theme.Green
	case domain.StatusWaiting:
		return theme.Blue
	case domain.StatusActive:
		return theme.Peach
	case domain.StatusNext:
		return theme.Mauve
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
	case domain.StatusWaiting:
		return "\u25cb" // ○
	case domain.StatusActive:
		return "\u25d0" // ◐
	case domain.StatusNext:
		return "\u25d1" // ◑
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
	case domain.StatusWaiting:
		return "Press n to capture something"
	case domain.StatusActive:
		return "What you are on right now"
	case domain.StatusNext:
		return "Queue up what comes after"
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
