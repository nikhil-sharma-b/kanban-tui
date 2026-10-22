// Package cli is the non-interactive front end: one command per invocation,
// text or JSON output, built for scripts and AI agents. It shares the domain
// model and store with the TUI, so both see and enforce the same rules.
package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/nikhilsharma/kanban-tui/internal/domain"
	"github.com/nikhilsharma/kanban-tui/internal/store"
	"github.com/nikhilsharma/kanban-tui/internal/whiteboardfs"
)

const (
	exitOK    = 0
	exitError = 1
	exitUsage = 2
)

// saveAttempts bounds retries when another process (usually the TUI) saves
// between our load and save.
const saveAttempts = 3

// usageError marks errors caused by bad arguments; they exit with code 2.
type usageError struct{ msg string }

func (e usageError) Error() string { return e.msg }

func usagef(format string, args ...any) error {
	return usageError{msg: fmt.Sprintf(format, args...)}
}

// Run executes one CLI command and returns the process exit code. dbPath and
// legacyPath locate the store, as resolved by store.ResolvePaths.
func Run(args []string, stdout, stderr io.Writer, dbPath, legacyPath string) int {
	app := &app{stdout: stdout, dbPath: dbPath, legacyPath: legacyPath}
	err := app.dispatch(args)
	if err == nil {
		return exitOK
	}
	if errors.Is(err, flag.ErrHelp) {
		return exitOK
	}
	fmt.Fprintf(stderr, "kanban: %v\n", err)
	var usage usageError
	if errors.As(err, &usage) {
		fmt.Fprintln(stderr, "run 'kanban help' for usage")
		return exitUsage
	}
	return exitError
}

// IsCommand reports whether args should be handled by the CLI rather than
// launching the TUI.
func IsCommand(args []string) bool {
	return len(args) > 0
}

type app struct {
	stdout     io.Writer
	dbPath     string
	legacyPath string
	json       bool
}

type command struct {
	usage string
	run   func(a *app, args []string) error
}

func (a *app) dispatch(args []string) error {
	if len(args) == 0 || isHelp(args[0]) {
		fmt.Fprint(a.stdout, helpText)
		return nil
	}

	group, ok := groups[args[0]]
	if !ok {
		return usagef("unknown command %q", args[0])
	}
	if len(args) < 2 || isHelp(args[1]) {
		a.printGroupHelp(args[0], group)
		return nil
	}
	cmd, ok := group[args[1]]
	if !ok {
		return usagef("unknown command %q %q", args[0], args[1])
	}
	return cmd.run(a, args[2:])
}

func isHelp(arg string) bool {
	return arg == "help" || arg == "-h" || arg == "--help"
}

func (a *app) printGroupHelp(name string, group map[string]command) {
	fmt.Fprintf(a.stdout, "kanban %s commands:\n", name)
	for _, sub := range groupOrder[name] {
		fmt.Fprintf(a.stdout, "  kanban %s %s\n", name, group[sub].usage)
	}
}

// flags builds a flag set with the options every command shares.
func (a *app) flags(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(a.stdout)
	fs.BoolVar(&a.json, "json", false, "print JSON")
	return fs
}

// parse lets flags and positional arguments mix in any order, so
// `task add "Title" -p Work` works as well as `task add -p Work "Title"`.
func parse(fs *flag.FlagSet, args []string, want int, names ...string) ([]string, error) {
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil, err
			}
			return nil, usageError{msg: err.Error()}
		}
		args = fs.Args()
		if len(args) == 0 {
			break
		}
		if args[0] == "--" {
			positional = append(positional, args[1:]...)
			break
		}
		positional = append(positional, args[0])
		args = args[1:]
	}
	if len(positional) != want {
		return nil, usagef("%s expects %d argument(s): %s", fs.Name(), want, strings.Join(names, " "))
	}
	return positional, nil
}

// openStore opens the workspace store; callers must close it.
func (a *app) openStore() (*store.SQLiteStore, error) {
	opened, err := store.Open(a.dbPath, a.legacyPath)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}
	sqliteStore, ok := opened.(*store.SQLiteStore)
	if !ok {
		return nil, fmt.Errorf("unexpected store type %T", opened)
	}
	return sqliteStore, nil
}

func (a *app) load(s store.WorkspaceStore) (*domain.Workspace, error) {
	workspace, err := s.Load()
	if err != nil {
		return nil, fmt.Errorf("load workspace: %w", err)
	}
	whiteboardfs.AssignWorkspacePaths(workspace, a.dbPath)
	return workspace, nil
}

// view runs a read-only command against the current workspace.
func (a *app) view(fn func(*domain.Workspace) (any, error)) error {
	s, err := a.openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	workspace, err := a.load(s)
	if err != nil {
		return err
	}
	result, err := fn(workspace)
	if err != nil {
		return err
	}
	return a.print(result)
}

// change describes the outcome of a mutation. commit runs after the
// workspace is saved (file moves that must not happen for a discarded save),
// rollback when the save fails.
type change struct {
	result   any
	commit   func() error
	rollback func() error
}

// mutate loads the workspace, applies fn, and saves. If another process saved
// in between, it reloads and reapplies fn rather than overwriting that save.
func (a *app) mutate(fn func(*domain.Workspace) (change, error)) error {
	s, err := a.openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	for attempt := 1; ; attempt++ {
		workspace, err := a.load(s)
		if err != nil {
			return err
		}
		c, err := fn(workspace)
		if err != nil {
			return err
		}
		err = s.Save(workspace)
		if err != nil {
			if c.rollback != nil {
				if rollbackErr := c.rollback(); rollbackErr != nil {
					err = fmt.Errorf("%w; roll back files: %v", err, rollbackErr)
				}
			}
			if errors.Is(err, store.ErrConflict) && attempt < saveAttempts {
				continue
			}
			return fmt.Errorf("save workspace: %w", err)
		}
		if c.commit != nil {
			if err := c.commit(); err != nil {
				return fmt.Errorf("saved, but updating whiteboard files failed: %w", err)
			}
		}
		return a.print(c.result)
	}
}

// printer is implemented by results that have a human-readable form.
type printer interface {
	printText(w io.Writer)
}

func (a *app) print(result any) error {
	if a.json {
		encoder := json.NewEncoder(a.stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	}
	if p, ok := result.(printer); ok {
		p.printText(a.stdout)
		return nil
	}
	return fmt.Errorf("no text output for %T", result)
}
