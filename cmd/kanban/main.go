package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhilsharma/kanban-tui/internal/cli"
	"github.com/nikhilsharma/kanban-tui/internal/domain"
	"github.com/nikhilsharma/kanban-tui/internal/store"
	"github.com/nikhilsharma/kanban-tui/internal/ui"
)

func main() {
	dataPath, legacyPath, err := store.ResolvePaths()
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve storage paths: %v\n", err)
		os.Exit(1)
	}

	if args := os.Args[1:]; cli.IsCommand(args) {
		os.Exit(cli.Run(args, os.Stdout, os.Stderr, dataPath, legacyPath))
	}

	var opened store.WorkspaceStore
	defer func() { closeStore(opened) }()
	open := func() (store.WorkspaceStore, *domain.Workspace, error) {
		boardStore, err := store.Open(dataPath, legacyPath)
		if err != nil {
			return nil, nil, err
		}
		workspace, err := boardStore.Load()
		if err != nil {
			closeStore(boardStore)
			return nil, nil, err
		}
		opened = boardStore
		return boardStore, workspace, nil
	}

	// A board that can't be read gets its own screen instead of exiting;
	// it never writes to the store (page 15).
	var model tea.Model
	if boardStore, workspace, err := open(); err != nil {
		model = ui.NewUnreadable(dataPath, shortenHome(dataPath), err, open, ui.Options{})
	} else {
		model = ui.New(workspace, boardStore, dataPath)
	}

	program := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "run tui: %v\n", err)
		os.Exit(1)
	}
}

func closeStore(s store.WorkspaceStore) {
	if closer, ok := s.(interface{ Close() error }); ok {
		closer.Close()
	}
}

// shortenHome writes the home directory as ~.
func shortenHome(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if rel, err := filepath.Rel(home, path); err == nil && !strings.HasPrefix(rel, "..") {
		return "~/" + rel
	}
	return path
}
