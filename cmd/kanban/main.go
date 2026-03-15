package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhilsharma/kanban-tui/internal/store"
	"github.com/nikhilsharma/kanban-tui/internal/ui"
)

func main() {
	dataPath, legacyPath, err := store.ResolvePaths()
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve storage paths: %v\n", err)
		os.Exit(1)
	}

	boardStore, err := store.Open(dataPath, legacyPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open store: %v\n", err)
		os.Exit(1)
	}
	if closer, ok := boardStore.(interface{ Close() error }); ok {
		defer closer.Close()
	}

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
