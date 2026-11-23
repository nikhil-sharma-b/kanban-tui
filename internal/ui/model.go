package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
	"github.com/nikhilsharma/kanban-tui/internal/store"
	"github.com/nikhilsharma/kanban-tui/internal/ui/dusk"
)

const (
	projectDialogMaxWidth = 60
	createDialogMaxWidth  = 92
	searchDialogMaxWidth  = 50
	detailDialogMaxWidth  = 72
	columnDialogMaxWidth  = 56
	defaultDialogPadding  = 2
	searchDialogPadding   = 2
	createDialogPadding   = 3
	maxCreateInputWidth   = 84
	taskTitleCharLimit    = 120
	maxModalInputWidth    = 42
	maxDescriptionHeight  = 14
	minDescriptionHeight  = 4
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
	// Pre-Dusk names, mapped onto Dusk tokens until each screen is redrawn.
	Rosewater: dusk.Dark.TextCard.Color,
	Flamingo:  dusk.Dark.Danger.Color,
	Pink:      dusk.Dark.Accent.Color,
	Mauve:     dusk.Dark.Accent.Color,
	Red:       dusk.Dark.Danger.Color,
	Peach:     dusk.Dark.Meta.Color,
	Yellow:    dusk.Dark.Meta.Color,
	Green:     dusk.Dark.Success.Color,
	Teal:      dusk.Dark.Success.Color,
	Blue:      dusk.Dark.Meta.Color,
	Lavender:  dusk.Dark.Accent.Color,
	Text:      dusk.Dark.Text.Color,
	Subtext1:  dusk.Dark.TextCard.Color,
	Subtext0:  dusk.Dark.TextMuted.Color,
	Overlay0:  dusk.Dark.TextDim.Color,
	Surface2:  dusk.Dark.TextFaint.Color,
	Surface1:  dusk.Dark.Border.Color,
	Surface0:  dusk.Dark.Border.Color,
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
	modeDetailEdit
	modeAddColumn
	modeRenameColumn
	modeProjects
	modeProjectEdit
	modeWhiteboards
	modeWhiteboardRename
	modeConfirm
	modeArchive
	modeArchiveDetail
	modeDailyPromote
	modePalette
)

// bulkArchiveDays is the fixed v1 threshold for archiving old Done tasks.
const bulkArchiveDays = 30

type saveFinishedMsg struct {
	err error
}

type queuedSaveFinishedMsg struct {
	sequence uint64
	msg      tea.Msg
}

type dailyPromotionFinishedMsg struct {
	title      string
	selectID   string // the card selected after the promoted one leaves
	workspace  *domain.Workspace
	dailyID    string
	targetName string
	status     domain.Status
	warning    error
	err        error
}

type editorFinishedMsg struct {
	err  error
	path string
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
	windowFirst        int // first column of the Dusk board window
	selected           map[domain.Status]int
	scroll             map[domain.Status]int
	visible            map[domain.Status][]string
	filter             string
	filterDraft        string
	filterOrigin       *boardPlace // selection before the filter, restored on clear
	columnInput        textinput.Model
	projectInput       textinput.Model
	mode               mode
	columnRename       domain.Status
	columnErr          error // duplicate or empty name, shown with the draft
	projectDraft       string
	projectCursor      int
	projectFilterInput textinput.Model
	projectFiltering   bool
	projectErr         error // shown in the projects dialogs, not the keybar
	whiteboardInput    textinput.Model
	whiteboardCursor   int
	whiteboardRenameID string
	whiteboardBack     mode // where esc from the list returns
	whiteboardStatus   string
	whiteboardKind     StatusKind
	launchWhiteboard   func(path string) error // nil: the package launcher
	archiveCursor      int
	archiveFilterInput textinput.Model
	archiveFiltering   bool
	promotion
	promotionPending   bool
	toast              toast // bottom-right above the keybar until the next key or its timer
	toastSeq           uint64
	undo               *undoEntry // offered by the showing toast only
	saveFailing        bool       // a save failed; retrying until one succeeds
	retryPending       bool
	savedToast         string // short ID to toast once the edit save lands
	reloaded           bool   // header reads “reloaded” after an external write
	reloadedSeq        uint64
	toastDuration      time.Duration
	saveRetryInterval  time.Duration
	reloadedDuration   time.Duration
	externalInterval   time.Duration
	reloading          bool
	saveBarrier        <-chan struct{}
	saveSequence       uint64
	latestSaveResult   uint64
	titleInput         textinput.Model
	descInput          textarea.Model
	searchInput        textinput.Model
	pendingKeys        []string // unfinished key sequence, e.g. ["space", "t"]
	whichKeyShown      bool     // the popover is up for pendingKeys
	whichKeySeq        uint64
	whichKeyDelay      time.Duration
	projectEditPrev    mode // where the project name dialog returns on cancel/rename
	editingTaskID      string
	editReturn         mode
	editDiscard        bool
	editHelp           bool
	editDescTop        int
	detailTaskID       string
	detailStatusMenu   bool
	detailStatusCursor int
	createColumn       domain.Status
	createProjectID    string
	createFeedback     string
	createSaving       bool
	newTask            func(title, description string) (*domain.Task, error)
	vimNormal          bool
	vimVisual          *vimSelection
	vim                vimEngine
	vimReplace         bool   // R – replace mode (overtype)
	vimStatus          string // transient feedback shown in dialog (e.g. yanked "foo")
	vimCommand         string // command-line input in normal mode, including leading ':'
	showHelp           bool
	lastStatus         string
	// lastInfo is an info toast on the keybar until the next key.
	lastInfo string
	// archiveStatus is the archive dialog's footer message until the next key.
	archiveStatus     string
	archiveStatusKind StatusKind
	lastErr           error
	confirm           confirmation
	confirmPrev       mode
	confirmInput      textinput.Model
	detailScroll      int
	detailWhiteboards bool
	detailHelp        bool
	detailStatus      string
	detailStatusKind  StatusKind
	writeClipboard    func(string) error
	// prevProjectID is the project to return to when leaving the daily board.
	prevProjectID string
	palette       dusk.Palette
	theme         ThemeSetting
	now           func() time.Time
	pal           paletteState
	focus         focusState
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
	titleInput.CharLimit = taskTitleCharLimit
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

	initialSaveBarrier := make(chan struct{})
	close(initialSaveBarrier)
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
		confirmInput:       textinput.New(),
		projectInput:       projectInput,
		projectFilterInput: projectFilterInput,
		whiteboardInput:    whiteboardInput,
		archiveFilterInput: archiveFilterInput,
		saveBarrier:        initialSaveBarrier,
		palette:            dusk.Dark,
		now:                time.Now,
		newTask:            domain.NewTask,
		toastDuration:      defaultToastDuration,
		theme:              defaultThemeSetting(),
		whichKeyDelay:      defaultWhichKeyDelay,
		saveRetryInterval:  defaultSaveRetryInterval,
		reloadedDuration:   defaultReloadedDuration,
		externalInterval:   externalCheckInterval,
	}

	if !project.Daily {
		m.prevProjectID = project.ID
	}

	m.recalculateVisible()
	m.syncResponsiveLayout()
	return m
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.scheduleExternalCheck(), m.themeWarning())
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := m.update(msg)
	// Slide the board window after every change of active column, so it
	// follows the selection step by step rather than only when drawn.
	if mm, ok := next.(*model); ok && mm.board != nil && mm.width > 0 {
		mm.boardWindow()
	}
	return next, cmd
}

func (m *model) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.syncResponsiveLayout()
		m.syncAllScroll()
		if m.mode == modeDetailEdit || m.mode == modeCreate {
			m.resizeDetailEditor()
		}
		return m, nil
	case queuedSaveFinishedMsg:
		if msg.sequence < m.latestSaveResult {
			return m, nil
		}
		m.latestSaveResult = msg.sequence
		return m.Update(msg.msg)
	case externalCheckMsg:
		return m.handleExternalCheck()
	case externalChangeMsg:
		return m.handleExternalChange(msg)
	case workspaceReloadedMsg:
		return m.handleWorkspaceReloaded(msg)
	case saveFinishedMsg:
		if isSaveConflict(msg.err) {
			if m.reloading {
				return m, nil
			}
			return m, m.reloadWorkspaceCmd(true)
		}
		return m.handleSaveFinished(msg.err)
	case saveRetryMsg:
		return m.handleSaveRetry()
	case whichKeyMsg:
		if msg.seq == m.whichKeySeq && len(m.pendingKeys) > 0 {
			m.whichKeyShown = true
		}
		return m, nil
	case toastExpiredMsg:
		if msg.seq == m.toastSeq {
			m.clearToast()
		}
		return m, nil
	case reloadedExpiredMsg:
		if msg.seq == m.reloadedSeq {
			m.reloaded = false
		}
		return m, nil
	case dailyPromotionFinishedMsg:
		m.promotionPending = false
		if isSaveConflict(msg.err) {
			m.mode = modeBoard
			if m.reloading {
				return m, nil
			}
			return m, m.reloadWorkspaceCmd(true)
		}
		if msg.err != nil {
			m.lastErr = msg.err
			m.lastStatus = "promotion failed"
			return m, nil
		}
		column := m.activeColumn
		m.workspace = msg.workspace
		m.activateProject(msg.dailyID)
		m.activeColumn = min(column, max(0, len(m.board.Statuses())-1))
		m.selectTask(msg.selectID)
		m.mode = modeBoard
		m.lastErr = msg.warning
		return m, m.showToast(StatusSuccess, fmt.Sprintf("promoted “%s” to %s / %s", singleLine(msg.title), msg.targetName, msg.status), KeyHint{})
	case editorFinishedMsg:
		return m.handleEditorResult(msg)
	case taskCreatedMsg:
		return m.finishCreateTask(msg)
	case cursor.BlinkMsg, tea.FocusMsg, tea.BlurMsg:
		if m.mode == modeCreate && m.descInput.Focused() {
			var cmd tea.Cmd
			m.descInput.Cursor, cmd = m.descInput.Cursor.Update(msg)
			return m, cmd
		}
	case tea.KeyMsg:
		undo := m.undo
		m.createFeedback, m.lastInfo = "", ""
		m.clearToast()
		if undo != nil && (m.mode == modeBoard || m.mode == modeDetail) && len(m.pendingKeys) == 0 && msg.String() == "u" {
			return m.applyUndo(*undo)
		}
		switch m.mode {
		case modeCreate:
			return m.updateNewTask(msg)
		case modeDetailEdit:
			return m.updateDetailEdit(msg)
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
		case modeDailyPromote:
			return m.updateDailyPromote(msg)
		case modePalette:
			return m.updatePalette(msg)
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

	board := m.duskBoard()
	if m.mode == modeDetail || m.mode == modeDetailEdit || m.mode == modeCreate {
		return m.placeDialog(board, m.taskDetail())
	}
	if m.mode == modeConfirm {
		return m.placeDialog(board, m.confirmDialog())
	}
	if m.mode == modeSearch {
		return m.placeDialog(board, m.searchDialog())
	}
	if m.mode == modeAddColumn {
		return m.placeAddColumnDialog(board)
	}
	if m.mode == modeProjects {
		return m.placeDialog(board, m.projectsDialog())
	}
	if m.mode == modeProjectEdit {
		return m.placeProjectEditDialog(board)
	}
	if m.mode == modeWhiteboards {
		return m.placeDialog(board, m.whiteboardsDialog())
	}
	if m.mode == modeArchive {
		return m.placeDialog(board, m.archiveDialog())
	}
	if m.mode == modeArchiveDetail {
		return m.placeDialog(board, m.archiveDetail())
	}
	if m.mode == modeWhiteboardRename {
		return m.placeWhiteboardRenameDialog(board)
	}
	if m.mode == modeDailyPromote {
		return m.placeDialog(board, m.promoteDialog())
	}
	if m.mode == modePalette {
		return m.placeDialog(board, m.paletteDialog())
	}
	view := renderGrid(board)

	switch m.mode {
	default:
		if m.showHelp {
			return m.placeOverlayCenter(view, m.renderHelpOverlay())
		}
		return view
	}
}

func (m *model) updateBoard(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	statuses := m.board.Statuses()
	if len(statuses) == 0 {
		return m, nil
	}
	if m.showHelp {
		if k := msg.String(); k == "esc" || k == "?" || k == "q" {
			m.showHelp = false
		}
		return m, nil
	}

	if m.onDailyBoard() {
		switch m.peekBoardAction(normalizeKey(msg)) {
		case actColumnNew, actColumnRename, actColumnDelete, actColumnMoveLeft, actColumnMoveRight:
			m.pendingKeys = nil
			m.lastErr = nil
			m.lastStatus = "daily board columns are fixed"
			return m, nil
		}
	}

	if m.focus.on {
		if handled, next, cmd := m.updateFocus(normalizeKey(msg)); handled {
			return next, cmd
		}
	} else if msg.String() == "z" && len(m.pendingKeys) == 0 {
		m.focus = focusState{on: true}
		return m, nil
	}
	if handled, cmd := m.leaderKeyStep(normalizeKey(msg)); handled {
		return m, cmd
	}
	if msg.String() == "esc" && len(m.pendingKeys) == 0 && m.filter != "" {
		m.clearFilter()
		return m, nil
	}
	action := m.resolveBoardKey(normalizeKey(msg))
	if len(m.pendingKeys) == 0 {
		m.whichKeyShown = false
	} else if !m.whichKeyShown {
		return m, m.armWhichKey()
	}
	switch action {
	case actColumnLeft, actColumnRight, actTaskUp, actTaskDown:
		// Moving while filtered keeps the new place when the filter clears.
		m.filterOrigin = nil
	}
	return m.runBoardAction(action)
}

// runBoardAction does what a board binding does; the keys and the command
// palette both come through here.
func (m *model) runBoardAction(action boardAction) (tea.Model, tea.Cmd) {
	statuses := m.board.Statuses()
	if len(statuses) == 0 {
		return m, nil
	}
	switch action {
	case actQuit:
		return m, tea.Quit
	case actPalette:
		return m.openPalette(false)
	case actDailyToggle:
		return m.toggleDaily()
	case actDailyDone:
		if m.onDailyBoard() {
			return m.markDailyDone()
		}
		m.lastStatus = "only on the daily board"
	case actDailyPromote:
		if !m.onDailyBoard() {
			m.lastStatus = "only on the daily board"
			return m, nil
		}
		if m.selectedTask() == nil {
			m.lastErr = nil
			m.lastStatus = "select a daily task to promote"
			return m, nil
		}
		return m.openPromote()
	case actDailyClear:
		if !m.onDailyBoard() {
			m.lastStatus = "only on the daily board"
			return m, nil
		}
		if len(m.board.Tasks) == 0 {
			m.lastErr = nil
			m.lastStatus = "daily board is already empty"
			return m, nil
		}
		return m.confirmClearDaily()
	case actColumnLeft:
		if m.activeColumn > 0 {
			m.activeColumn--
		}
		m.syncScroll(statuses[m.activeColumn])
	case actColumnRight:
		if m.activeColumn < len(statuses)-1 {
			m.activeColumn++
		}
		m.syncScroll(statuses[m.activeColumn])
	case actTaskUp:
		m.moveSelection(-1)
	case actTaskDown:
		m.moveSelection(1)
	case actTaskMoveLeft:
		return m.shiftSelected(-1)
	case actTaskMoveRight:
		if m.focus.on {
			return m, m.focusMoveRight()
		}
		return m.shiftSelected(1)
	case actColumnMoveLeft:
		return m.moveColumn(-1)
	case actColumnMoveRight:
		return m.moveColumn(1)
	case actTaskReorderUp:
		return m.reorderSelected(-1)
	case actTaskReorderDown:
		return m.reorderSelected(1)
	case actTaskNew:
		m.createColumn = statuses[m.activeColumn]
		m.createProjectID = m.project.ID
		m.editingTaskID = ""
		m.mode = modeCreate
		m.detailStatusMenu, m.editHelp, m.editDiscard = false, false, false
		m.editDescTop, m.detailStatus = 0, ""
		m.vimCommand, m.vimStatus = "", ""
		m.resizeDetailEditor()
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
	case actTaskEdit:
		return m.beginEditSelected()
	case actColumnNew:
		m.mode = modeAddColumn
		m.columnErr = nil
		m.columnInput.SetValue("")
		m.columnInput.Focus()
		m.lastErr = nil
		return m, textinput.Blink
	case actProjects:
		return m.openProjects()
	case actColumnRename:
		return m.beginRenameColumn()
	case actColumnDelete:
		statuses := m.board.Statuses()
		if len(statuses) > 0 && m.activeColumn < len(statuses) {
			return m.confirmDeleteColumn(statuses[m.activeColumn])
		}
	case actSearch:
		return m.openSearch()
	case actOpen:
		if task := m.selectedTask(); task != nil {
			m.mode = modeDetail
			m.detailTaskID = task.ID
			m.detailScroll, m.detailWhiteboards, m.detailHelp, m.detailStatus = 0, false, false, ""
			m.detailStatusMenu = false
		}
	case actTaskDelete:
		task := m.selectedTask()
		if task != nil {
			return m.confirmDeleteTask(task, modeBoard)
		}
	case actTaskArchive:
		task := m.selectedTask()
		if task != nil {
			return m.confirmArchiveTask(task)
		}
	case actArchiveOld:
		return m.confirmArchiveOld(modeBoard)
	case actArchiveView:
		return m.openArchive()
	case actHelp:
		m.showHelp = !m.showHelp
	case actTaskTop:
		m.moveSelection(-len(m.visible[statuses[m.activeColumn]]))
	case actTaskBottom:
		m.moveSelection(len(m.visible[statuses[m.activeColumn]]))
	case actTaskCopyID:
		return m.copySelectedTaskID()
	case actTaskWhiteboards:
		return m.openWhiteboards()
	case actProjectNew:
		return m.beginProjectEdit("", "", modeBoard)
	case actProjectRename:
		if m.project != nil {
			return m.beginProjectEdit(m.project.ID, m.project.Name, modeBoard)
		}
	case actProjectDelete:
		if m.project != nil {
			if err := m.deletableProject(m.project); err != nil {
				m.lastErr = err
				return m, nil
			}
			return m.confirmDeleteProject(m.project, modeBoard)
		}
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
		if m.mode == modeCreate {
			m.detailStatus = "editor: " + msg.err.Error()
			return m, nil
		}
		m.lastErr = fmt.Errorf("editor: %w", msg.err)
		return m, nil
	}

	content, err := os.ReadFile(msg.path)
	if err != nil {
		if m.mode == modeCreate {
			m.detailStatus = "read file: " + err.Error()
			return m, nil
		}
		m.lastErr = fmt.Errorf("read file: %w", err)
		return m, nil
	}

	title, description := parseEditorContent(string(content))
	if m.mode == modeCreate && m.editingTaskID == "" {
		if title == "" {
			m.closeCreatePanel()
			return m, nil
		}
		m.titleInput.CharLimit = max(taskTitleCharLimit, len([]rune(title)))
		m.titleInput.SetValue(title)
		m.descInput.SetValue(description)
		return m.createTask(title, description)
	}
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
		m.savedToast = shortID(task.ID)
		m.lastErr = nil
		m.recalculateVisible()
		m.selectTask(task.ID)

		return m, m.saveWorkspaceCmd()
	}

	return m, nil
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

func (m *model) updateEditorInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	model, cmd := m.updateCreateInner(msg)
	return model, tea.Batch(cmd, m.syncVimCursor())
}

func (m *model) updateCreateInner(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Always handle ctrl+e regardless of vim mode.
	switch msg.String() {
	case "ctrl+e":
		if m.mode == modeCreate && m.editingTaskID == "" {
			next, cmd := m.openEditorWithDraft()
			if m.lastErr != nil {
				m.detailStatus = m.lastErr.Error()
				m.lastErr = nil
			}
			return next, cmd
		}
		if m.mode == modeDetailEdit {
			m.titleInput.CharLimit = taskTitleCharLimit
			m.syncResponsiveLayout()
		}
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
	case "enter":
		if m.mode == modeCreate && m.titleInput.Focused() {
			return m.createTask(m.titleInput.Value(), "")
		}
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
	case "enter":
		if m.mode == modeCreate && m.titleInput.Focused() && !m.vim.pending() {
			return m.createTask(m.titleInput.Value(), "")
		}
	case "esc":
		if m.vim.pending() {
			m.vim.reset()
			return m, nil
		}
		if m.mode == modeDetailEdit {
			return m.requestDetailDiscard()
		}
		if m.titleInput.Value() != "" || m.descInput.Value() != "" {
			m.editDiscard = true
		} else {
			m.closeCreatePanel()
		}
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
			if m.mode == modeCreate {
				m.detailStatus = "use :wq to create"
				return m, nil
			}
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
	titleCmd := m.titleInput.Cursor.SetMode(mode)
	descCmd := m.descInput.Cursor.SetMode(mode)
	if m.mode == modeCreate && m.descInput.Focused() && mode == cursor.CursorBlink {
		descCmd = m.descInput.Cursor.BlinkCmd()
	}
	return tea.Batch(
		titleCmd,
		descCmd,
	)
}

func (m *model) saveTask(closePanel bool) (tea.Model, tea.Cmd) {
	title := m.titleInput.Value()
	description := m.descInput.Value()

	if m.editingTaskID != "" {
		task, err := m.board.UpdateTask(m.editingTaskID, title, description)
		if err != nil {
			if m.mode == modeDetailEdit {
				m.detailStatusKind, m.detailStatus = StatusDanger, err.Error()
				if strings.TrimSpace(title) == "" {
					m.detailStatus = "title can't be empty"
					m.descInput.Blur()
					m.titleInput.Focus()
					m.vimNormal = false
				}
				return m, nil
			}
			m.lastErr = err
			return m, nil
		}

		m.savedToast = shortID(task.ID)
		m.lastErr = nil
		m.recalculateVisible()
		m.selectTask(task.ID)
		if closePanel {
			m.closeCreatePanel()
		} else if m.mode == modeDetailEdit {
			// Compare future changes to the normalized values that were saved.
			posTitle := m.titleInput.Position()
			m.titleInput.SetValue(task.Title)
			m.titleInput.SetCursor(posTitle)
			pos := descBuffer{m}.Cursor()
			descBuffer{m}.SetText(task.Description, pos)
		}

		return m, m.saveWorkspaceCmd()
	}

	return m.createTask(title, description)
}

func (m *model) closeCreatePanel() {
	m.titleInput.CharLimit = taskTitleCharLimit
	if m.mode == modeDetailEdit {
		m.syncResponsiveLayout()
	}
	m.mode = modeBoard
	m.editingTaskID = ""
	m.vimNormal = false
	m.vimVisual = nil
	m.vimReplace = false
	m.vimCommand = ""
	m.editDiscard = false
	m.titleInput.Blur()
	m.descInput.Blur()
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

	m.editReturn = m.mode
	m.editingTaskID = task.ID
	m.mode = modeDetailEdit
	m.editDiscard, m.editHelp = false, false
	m.editDescTop, m.detailStatus = 0, ""
	m.vimNormal = false
	m.vimVisual = nil
	m.vimReplace = false
	m.vimCommand, m.vimStatus = "", ""
	m.vim.reset()
	m.resizeDetailEditor()
	// Preserve existing titles over the creation limit (e.g. from the CLI).
	m.titleInput.CharLimit = max(taskTitleCharLimit, len([]rune(task.Title)))
	m.titleInput.SetValue(task.Title)
	m.titleInput.CursorEnd()
	m.descInput.SetValue(task.Description)
	descBuffer{m}.SetCursor(0)
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
	m.columnErr = nil
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
	m.detailStatus = ""
	if m.detailStatusMenu {
		return m.updateDetailStatusMenu(msg.String())
	}
	if msg.String() == "?" {
		m.detailHelp = !m.detailHelp
		return m, nil
	}
	if m.detailHelp && msg.String() == "esc" {
		m.detailHelp = false
		return m, nil
	}
	m.detailHelp = false
	if m.detailWhiteboards {
		switch msg.String() {
		case "esc", "q":
			m.detailWhiteboards = false
		case "up", "k":
			m.whiteboardCursor = max(0, m.whiteboardCursor-1)
		case "down", "j":
			if task := m.selectedTask(); task != nil && m.whiteboardCursor+1 < len(task.Whiteboards) {
				m.whiteboardCursor++
			}
		case "enter", "o":
			return m.openSelectedWhiteboard()
		}
		return m, nil
	}
	switch msg.String() {
	case "esc", "enter", "q":
		m.mode = modeBoard
	case ":":
		// View mode only: while a field is edited, : is the vim command line.
		return m.openPalette(true)
	case "e":
		return m.beginEditSelected()
	case "s":
		if task := m.selectedTask(); task != nil {
			m.detailStatusMenu = true
			m.detailStatusCursor = max(0, m.board.StatusIndex(task.Status))
		}
	case "w":
		if task := m.selectedTask(); task != nil && m.detailLayout(len(task.Whiteboards)).listRows > 0 {
			m.detailWhiteboards, m.whiteboardCursor = true, 0
		} else {
			return m.openWhiteboards()
		}
	case "J", "K":
		m.scrollDetail(msg.String() == "J")
	case "y":
		return m.copySelectedTaskID()
	}

	return m, nil
}

// copySelectedTaskID puts the full task ID on the system clipboard so it can be
// handed to the kanban CLI or an agent.
func (m *model) copySelectedTaskID() (tea.Model, tea.Cmd) {
	task := m.selectedTask()
	if task == nil {
		return m, nil
	}
	write := m.writeClipboard
	if write == nil {
		write = clipboard.WriteAll
	}
	if err := write(task.ID); err != nil {
		if m.mode == modeDetail {
			m.detailStatusKind, m.detailStatus = StatusDanger, "could not copy ID"
			return m, nil
		}
		return m, m.showToast(StatusDanger, "Could not copy ID", KeyHint{})
	}
	if m.mode == modeDetail {
		m.detailStatusKind, m.detailStatus = StatusSuccess, "copied "+shortID(task.ID)
		return m, nil
	}
	m.lastErr = nil
	return m, m.showToast(StatusSuccess, "Copied "+shortID(task.ID), KeyHint{})
}

func (m *model) updateWhiteboards(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	task := m.selectedTask()
	if task == nil {
		m.mode = modeBoard
		return m, nil
	}
	// A status line answers only the key that produced it.
	m.whiteboardStatus = ""

	switch msg.String() {
	case "esc":
		m.mode = m.whiteboardBack
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
	case "enter":
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
	m.detailTaskID = task.ID
	if len(task.Whiteboards) == 0 {
		m.whiteboardCursor = 0
	} else if m.whiteboardCursor >= len(task.Whiteboards) {
		m.whiteboardCursor = len(task.Whiteboards) - 1
	}
	if m.mode != modeWhiteboards {
		m.whiteboardBack = m.mode
	}
	m.mode = modeWhiteboards
	m.lastErr, m.whiteboardStatus = nil, ""
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
	launchErr := m.launch(whiteboard.Path)
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
	if err := m.launch(path); err != nil {
		if m.mode == modeDetail {
			m.detailStatusKind, m.detailStatus = StatusDanger, "could not open whiteboard"
			return m, nil
		}
		m.lastErr = fmt.Errorf("launch whiteboard: %w", err)
		return m, nil
	}
	if m.mode == modeDetail {
		m.detailStatusKind, m.detailStatus = StatusSuccess, "opened "+whiteboard.Name
		return m, nil
	}
	m.lastErr = nil
	m.whiteboardKind, m.whiteboardStatus = StatusSuccess, "opened "+whiteboard.Name
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
	spot := m.spotOf(task)
	if !m.board.ShiftTask(task.ID, delta) {
		return m, nil
	}

	m.recalculateVisible()
	m.selectTask(task.ID)
	return m, tea.Batch(m.saveWorkspaceCmd(), m.showUndoToast(StatusSuccess, "Moved to "+columnName(task.Status), moveUndo(spot)))
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
	task := m.boardSelectedTask()
	if task == nil {
		return m, nil
	}
	spot := m.spotOf(task)
	if !m.board.MoveWithin(status, index, target) {
		return m, nil
	}

	m.selected[status] = target
	m.recalculateVisible()
	m.syncScroll(status)
	verb := "Moved down"
	if delta < 0 {
		verb = "Moved up"
	}
	return m, tea.Batch(m.saveWorkspaceCmd(), m.showUndoToast(StatusSuccess, verb, undoEntry{spots: []undoSpot{spot}, undone: "Order restored"}))
}

func (m *model) deleteTask(task *domain.Task) (tea.Model, tea.Cmd) {
	m.mode = modeBoard
	// A reload already in flight can replace the board while the dialog is open.
	if m.board.Tasks[task.ID] != task {
		m.lastErr = fmt.Errorf("task changed; reopen the confirmation")
		return m, nil
	}

	if !m.board.DeleteTask(task.ID) {
		return m, nil
	}

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

	spot := m.spotOf(task)
	spot.archived = true
	if _, err := m.board.ArchiveTask(task.ID); err != nil {
		m.mode = modeBoard
		m.lastErr = err
		return m, nil
	}

	m.mode = modeBoard
	m.lastErr = nil
	m.recalculateVisible()
	return m, tea.Batch(m.saveWorkspaceCmd(), m.showUndoToast(StatusInfo, m.quotedToast("Archived ", task.Title, undoHint),
		undoEntry{spots: []undoSpot{spot}, undone: m.quotedToast("Restored ", task.Title, KeyHint{})}))
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

// visibleTaskCount returns how many task cards fit in the column body starting
// from the given scroll offset, by measuring actual rendered card heights.
func (m *model) visibleTaskCount(status domain.Status, scroll int) int {
	statuses := m.board.Statuses()
	active := m.activeColumn < len(statuses) && statuses[m.activeColumn] == status
	return max(1, fitCards(m.columnCards(status, active), scroll, m.duskBodyHeight()))
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
	switch {
	case m.mode == modeDetailEdit:
		return m.board.Tasks[m.editingTaskID]
	case m.mode == modeDetail || m.mode == modeWhiteboards || m.mode == modeWhiteboardRename ||
		(m.mode == modePalette && m.pal.fromDetail) ||
		(m.mode == modeConfirm && m.confirmPrev == modeWhiteboards):
		if m.detailTaskID != "" {
			return m.board.Tasks[m.detailTaskID]
		}
	}
	return m.boardSelectedTask()
}

func (m *model) boardSelectedTask() *domain.Task {
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

// ─── Board ───────────────────────────────────────────────────────────────────

// ─── Footer ──────────────────────────────────────────────────────────────────

// ─── Dialogs ─────────────────────────────────────────────────────────────────

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
	m.focus = focusState{} // focus mode belongs to the board it opened on
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
	return m, nil
}

func promoteDailyTask(workspaceStore store.WorkspaceStore, workspace *domain.Workspace, task *domain.Task, dataPath, dailyID, targetName string, status domain.Status) tea.Msg {
	commitFiles, rollbackFiles, err := stageTaskWhiteboardFiles(task, dataPath, targetName)
	if err != nil {
		return dailyPromotionFinishedMsg{err: fmt.Errorf("stage whiteboards: %w", err)}
	}
	if err := workspaceStore.Save(workspace); err != nil {
		if cleanupErr := rollbackFiles(); cleanupErr != nil {
			err = fmt.Errorf("save promotion: %v; clean staged whiteboards: %w", err, cleanupErr)
		}
		return dailyPromotionFinishedMsg{err: err}
	}
	return dailyPromotionFinishedMsg{
		workspace:  workspace,
		dailyID:    dailyID,
		targetName: targetName,
		status:     status,
		warning:    commitFiles(),
	}
}

// markDailyDone archives the selected daily task so it stops showing on the
// board but stays available in the archive view.
func (m *model) markDailyDone() (tea.Model, tea.Cmd) {
	task := m.selectedTask()
	if task == nil {
		return m, nil
	}

	spot := m.spotOf(task)
	spot.archived = true
	if _, err := m.board.ArchiveTask(task.ID); err != nil {
		m.lastErr = err
		return m, nil
	}

	m.lastErr = nil
	m.recalculateVisible()
	return m, tea.Batch(m.saveWorkspaceCmd(), m.showUndoToast(StatusSuccess, m.quotedToast("Done ", task.Title, undoHint),
		undoEntry{spots: []undoSpot{spot}, undone: m.quotedToast("Restored ", task.Title, KeyHint{})}))
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
	workspace := m.workspace.Clone()
	return m.queueSave(func() tea.Msg {
		return saveFinishedMsg{err: m.store.Save(workspace)}
	})
}

func (m *model) queueSave(action func() tea.Msg) tea.Cmd {
	previous := m.saveBarrier
	done := make(chan struct{})
	m.saveBarrier = done
	m.saveSequence++
	sequence := m.saveSequence
	return func() tea.Msg {
		<-previous
		defer close(done)
		return queuedSaveFinishedMsg{sequence: sequence, msg: action()}
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

func relativeTime(now, t time.Time) string {
	d := now.Sub(t)
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
		return t.Local().Format("2 Jan")
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
