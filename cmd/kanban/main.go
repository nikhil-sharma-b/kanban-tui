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
	"github.com/nikhilsharma/kanban-tui/internal/themeconf"
	"github.com/nikhilsharma/kanban-tui/internal/ui"
	"github.com/nikhilsharma/kanban-tui/internal/ui/dusk"
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

	opts := ui.Options{Theme: themeSetting(dataPath)}

	// A board that can't be read gets its own screen instead of exiting;
	// it never writes to the store (page 15).
	var model tea.Model
	if boardStore, workspace, err := open(); err != nil {
		model = ui.NewUnreadable(dataPath, shortenHome(dataPath), err, open, opts)
	} else {
		model = ui.NewWithOptions(workspace, boardStore, dataPath, opts)
	}

	program := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "run tui: %v\n", err)
		os.Exit(1)
	}
}

// themeSetting reads config.toml beside the data file (page 17). The
// terminal is asked for its background once, before the TUI owns the input.
func themeSetting(dataPath string) *ui.ThemeSetting {
	path := themeconf.Path(dataPath)
	setting, _ := themeconf.Read(path)
	env := themeconf.Detect()
	dark, ok := env.Background()
	env.Background = func() (bool, bool) { return dark, ok }
	return &ui.ThemeSetting{
		Setting: setting,
		Palette: func(n themeconf.Name) dusk.Palette { return themeconf.Palette(n, env) },
		Save:    func(n themeconf.Name) error { return themeconf.Write(path, n) },
		Load:    func() (themeconf.Setting, error) { return themeconf.Read(path) },
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
