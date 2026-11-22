package ui

import (
	"fmt"
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
		board:           board,
		width:           width,
		height:          height,
		titleInput:      textinput.New(),
		descInput:       textarea.New(),
		searchInput:     textinput.New(),
		columnInput:     textinput.New(),
		projectInput:    textinput.New(),
		whiteboardInput: textinput.New(),
	}
}
