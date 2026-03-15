package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
)

func TestDialogWidths(t *testing.T) {
	m := &model{width: 120}
	if got, want := m.dialogWidth(createDialogMaxWidth), createDialogMaxWidth; got != want {
		t.Fatalf("dialogWidth() = %d, want %d", got, want)
	}

	m.width = 50
	if got, want := m.dialogWidth(createDialogMaxWidth), 46; got != want {
		t.Fatalf("dialogWidth() = %d, want %d", got, want)
	}

	if got, want := m.dialogContentWidth(46, createDialogPadding), 38; got != want {
		t.Fatalf("dialogContentWidth() = %d, want %d", got, want)
	}

	if got, want := m.dialogContentWidth(4, createDialogPadding), 1; got != want {
		t.Fatalf("dialogContentWidth() = %d, want %d", got, want)
	}
}

func TestCompactBoardDecision(t *testing.T) {
	testCases := []struct {
		name  string
		width int
		cols  int
		want  bool
	}{
		{name: "single column never compact", width: 40, cols: 1, want: false},
		{name: "medium width still compact for four columns", width: 100, cols: 4, want: true},
		{name: "narrow width compact", width: 70, cols: 3, want: true},
		{name: "very narrow width compact", width: 40, cols: 2, want: true},
		{name: "wide width non-compact", width: 200, cols: 3, want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := responsiveTestModel(tc.width, 24, tc.cols)
			if got := m.useCompactBoardLayout(); got != tc.want {
				t.Fatalf("useCompactBoardLayout() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSyncResponsiveLayout(t *testing.T) {
	m := responsiveTestModel(40, 14, 3)
	m.syncResponsiveLayout()

	if got, want := m.titleInput.Width, 28; got != want {
		t.Fatalf("titleInput.Width = %d, want %d", got, want)
	}

	if got, want := m.searchInput.Width, 30; got != want {
		t.Fatalf("searchInput.Width = %d, want %d", got, want)
	}

	if got, want := m.columnInput.Width, 30; got != want {
		t.Fatalf("columnInput.Width = %d, want %d", got, want)
	}
}

func TestHeightDerivedSizes(t *testing.T) {
	m := &model{height: 8}
	if got, want := m.columnHeight(), 6; got != want {
		t.Fatalf("columnHeight() = %d, want %d", got, want)
	}
	if got, want := m.taskRows(), 1; got != want {
		t.Fatalf("taskRows() = %d, want %d", got, want)
	}

	m.height = 24
	if got, want := m.columnHeight(), 14; got != want {
		t.Fatalf("columnHeight() = %d, want %d", got, want)
	}
	if got, want := m.taskRows(), 2; got != want {
		t.Fatalf("taskRows() = %d, want %d", got, want)
	}
}

func TestCompactColumnIndicator(t *testing.T) {
	m := responsiveTestModel(100, 24, 4)
	m.activeColumn = 1

	if got, want := m.compactColumnIndicator(), "2/4 status-1"; got != want {
		t.Fatalf("compactColumnIndicator() = %q, want %q", got, want)
	}
}

func TestRenderFooterCompactUsesShorterHelp(t *testing.T) {
	m := responsiveTestModel(100, 24, 4)
	m.showHelp = false

	footer := m.renderFooter()
	if !strings.Contains(footer, "column") {
		t.Fatalf("renderFooter() missing compact navigation hint: %q", footer)
	}
	if strings.Contains(footer, "navigate") {
		t.Fatalf("renderFooter() should use compact footer copy: %q", footer)
	}
	if strings.Contains(footer, "rename") {
		t.Fatalf("renderFooter() should omit long hints in compact collapsed mode: %q", footer)
	}
}

func responsiveTestModel(width, height int, cols int) *model {
	board := &domain.Board{
		Columns: make([]domain.Status, cols),
		Tasks:   map[string]*domain.Task{},
		Order:   map[domain.Status][]string{},
	}

	for i := 0; i < cols; i++ {
		status := domain.Status(fmt.Sprintf("status-%d", i))
		board.Columns[i] = status
		board.Order[status] = []string{}
	}

	return &model{
		board:       board,
		width:       width,
		height:      height,
		titleInput:  textinput.New(),
		descInput:   textarea.New(),
		searchInput: textinput.New(),
		columnInput: textinput.New(),
	}
}
