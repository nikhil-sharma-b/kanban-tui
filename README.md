# kanban-tui

Keyboard-first kanban task management TUI written in Go.

Projects are first-class: each project owns its own kanban board, and you can create, rename, switch, and delete projects from inside the TUI.

## Why this shape

The app is built around a normalized board model:

- tasks are stored once in a `map[id]*Task`
- each column keeps an ordered slice of task IDs
- the UI filters into visible ID slices and only renders the current window of cards

That keeps state updates cheap and avoids repainting entire task sets when boards get large.

## Install

```bash
make install      # builds and installs to ~/.local/bin/kanban
```

Make sure `~/.local/bin` is on your `PATH`, then run `kanban` from anywhere. Install elsewhere with `make install PREFIX=/usr/local` (may need `sudo`), and remove it with `make uninstall`. Rerun `make install` after pulling changes.

### Agent skill

`skills/kanban/SKILL.md` teaches AI agents (Claude Code and others that read `~/.claude/skills`) to drive the board through the CLI. Install it with:

```bash
make install-skill   # symlinks ~/.claude/skills/kanban to this repo
```

Because it is a symlink, pulling the repo updates the skill too. Point it elsewhere with `make install-skill SKILLDIR=/path/to/skills`.

## Run

```bash
go run ./cmd/kanban
```

The default database file is:

```text
$XDG_CONFIG_HOME/kanban-tui/board.db
```

On macOS that resolves to:

```text
~/Library/Application Support/kanban-tui/board.db
```

Override it with:

```bash
KANBAN_TUI_DATA_FILE=/path/to/board.db go run ./cmd/kanban
```

Whiteboards are stored under `whiteboards/` next to the active database by default:

```text
$XDG_CONFIG_HOME/kanban-tui/whiteboards/<project>/<task-id>/
```

Override the whiteboard root or launcher with:

```bash
KANBAN_TUI_WHITEBOARD_DIR=/path/to/whiteboards
KANBAN_TUI_WHITEBOARD_CMD=xournalpp
```

Existing JSON data is migrated automatically on first run:

- the default legacy file is `board.json` in the same app config directory
- if `KANBAN_TUI_DATA_FILE` points to a legacy `.json` file, the app imports it into a sibling `.db` file with the same base name

## CLI

Any arguments run a single command instead of the TUI, which makes the board scriptable and usable by AI agents:

```bash
kanban task add "Write release notes" -p Work --column doing
kanban task list -p Work --json
kanban task move 3f2a --column done        # unique ID prefix is enough
kanban daily add "Call Bob"
kanban daily promote 9c1e --to Work
```

Run `kanban help` for every command. Notes:

- every command accepts `--json` and prints the affected task, project or board
- projects are referenced by name (case-insensitive) or ID; `-p` defaults to the active project
- new projects start with `backlog`, `todo`, `doing`, and `done`; existing projects keep their columns
- tasks are referenced by full ID or any unique prefix, across all projects
- errors go to stderr; exit code `1` for failures, `2` for bad usage

The CLI and TUI can run at the same time. The TUI picks up CLI changes within a second. If both write at once, the store rejects the stale save: the CLI retries on fresh data, and the TUI reloads and reports that its last change was dropped.

## Keys

Board keys follow Neovim: bare keys only navigate, and everything that creates, changes or opens something sits behind the leader, `space` (shown as `SPC`). Press `space` and a which-key popover lists what can follow; groups like `+task` open a submenu, `backspace` steps back, `?` opens the cheat sheet and `esc` cancels. Every screen lists its keys in the bottom bar or dialog footer, and `?` shows the rest.

Board:

- `h` / `l` (or arrows): select column
- `j` / `k` (or arrows): select task
- `gg` / `G`: first / last task in the column
- `enter`: open task details
- `/`: search tasks; `esc` on the board clears the filter
- `:`: command palette, every action by name
- `z`: focus mode on the selected task: `J` / `K` scroll its description, `SPC t l` moves it right and focuses the next task, `j` / `k` pick a card when the column is empty, `z` or `esc` returns
- `u`: undo the last move or archive while its toast is showing
- `?`: key cheat sheet (type to filter, `enter` to run, `esc` to close)
- `q` / `ctrl+c`: quit

`SPC t` (+task):

- `n` new · `e` edit · `x` delete · `a` archive (delete and archive confirm first)
- `h` / `l`: move task to the previous / next column
- `k` / `j`: reorder within the column
- `w`: whiteboards · `y`: copy the task ID

`SPC c` (+column):

- `n` new · `r` rename in place · `d` delete (its tasks move to the adjacent column; at least one column is kept)
- `h` / `l`: move the column left / right

`SPC d` (+daily):

- `d`: jump into the daily board (press again to jump back)
- `m` mark done · `p` promote · `X` clear (daily board only; see below)

`SPC a` (+archive):

- `v`: open the archive
- `o`: archive Done tasks not updated in more than 30 days (with confirm)

`SPC p` (+project):

- `p`: switch project
- `n`: new project
- `r`: rename current project
- `x`: delete current project (with confirm)

Task details:

- `e`: edit · `s`: set status · `w`: whiteboards · `y`: copy ID
- `J` / `K`: scroll the description
- `:`: command palette
- `esc` / `enter` / `q`: close · `?`: more keys
- in the whiteboards list (`w` when the task has some): `j` / `k` move, `enter` / `o` open, `esc` back

Editing a task and the new-task dialog edit Vim-style:

- type to insert; `esc` for normal mode, `i` to insert again
- `tab`: next field · `ctrl+e`: write the description in `$EDITOR`
- `:w` saves (editing), `:wq` saves and closes (or creates the new task)
- new task only: `enter` in the title creates it, `s` (normal mode) picks its column
- `esc` in normal mode closes, asking first when there are unsaved changes

Search (`/`):

- type to filter the board live
- `enter`: keep the filter · `esc`: restore the previous one

Command palette (`:`):

- type to filter · `↑` / `↓` (or `ctrl+p` / `ctrl+n`): move · `enter`: run
- `esc`: close, or back from a second step
- with no match, `enter` searches tasks for the typed text
- `theme…` switches the colour theme (see below)

Project switcher (`SPC p p`):

- type to filter · `↑` / `↓`: move · `enter`: switch
- `esc`: manage projects: `j` / `k` move, `enter` switch, `/` filter, `n` new, `e` rename, `x` delete, `esc` close

Whiteboard manager (`SPC t w`):

- `j` / `k`: move between whiteboards
- `enter`: open the selected whiteboard
- `n`: create a new whiteboard and open it immediately
- `r`: rename · `x`: delete it and its file (with confirm)
- `esc`: close

Archive (`SPC a v`):

- `j` / `k`: move between archived tasks
- `/`: filter archived tasks (title + description)
- `enter`: open read-only task details (`J` / `K` scroll, `r` restore, `esc` back)
- `r`: restore the selected task to its original column (or the first column if it was deleted)
- `a`: archive old Done tasks
- `esc`: close

Promote (`SPC d p` on the daily board):

- pick a project: type to filter, `↑` / `↓` move, `enter` next, `esc` cancel
- pick a column: `j` / `k` move, `enter` promote, `esc` back to projects

Confirm dialogs:

- `y` confirms, `esc` keeps
- deleting a column with tasks, deleting a project and clearing the daily board ask you to type its name first, then `enter`

If the board can't be read, the app opens on a screen that never writes to it: `r` retries, `q` quits.

## Daily board

The daily board is a single, always-present board for the small tasks that come up during the day. It sits next to your projects but never shows up in the project manager, and it cannot be renamed or deleted.

It has three fixed columns:

```text
Waiting   Active   Next
```

Columns cannot be added, renamed, reordered, or deleted there.

Keys inside the daily board:

- `SPC d d`: leave the daily board and go back to the last project you were on
- `SPC t n`: capture a new task in Waiting
- `SPC t h` / `SPC t l`: move a task between Waiting, Active and Next
- `SPC d m`: mark the selected task done - it disappears from the board and moves to the archive
- `SPC d p`: promote the selected task into a project column
- `SPC a v`: view the tasks you already marked done (`r` restores one to the board)
- `SPC d X`: clear the board, deleting every daily task including the done ones (with confirm)

## Whiteboards

Each task can own multiple named whiteboards. New whiteboards are created with automatic names like `Whiteboard 1`, `Whiteboard 2`, and so on, but you can rename them later from the whiteboard manager.

New whiteboards are Xournal++ (`.xopp`) files and open in Xournal++; older `.rnote` whiteboards still open in Rnote. Set `KANBAN_TUI_WHITEBOARD_CMD` to use another launcher.

Whiteboard files use stable paths:

```text
whiteboards/<project-slug>/<task-id>/<whiteboard-slug>.xopp
```

Renaming a whiteboard updates the display name in the TUI only. Deleting a whiteboard removes both the task link and the underlying file.

## Theme

The TUI uses the Dusk palette. Pick a theme from the command palette (`:` then `theme…`) or set it in `config.toml` beside `board.db`:

```toml
theme = "dusk"   # or "dusk-light", or "terminal" to follow the terminal's background
```

Hand edits apply while the app runs. Terminals with fewer than 256 colours get a 16-colour version of the palette.
