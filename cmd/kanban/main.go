package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhilsharma/kanban-tui/internal/store"
	"github.com/nikhilsharma/kanban-tui/internal/ui"
)

func main() {
	dataPath, err := store.DefaultPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve data path: %v\n", err)
		os.Exit(1)
	}

	boardStore := store.New(dataPath)
	board, err := boardStore.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load board: %v\n", err)
		os.Exit(1)
	}

	program := tea.NewProgram(ui.New(board, boardStore, dataPath), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "run tui: %v\n", err)
		os.Exit(1)
	}
}
