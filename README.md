# kanban-tui

Keyboard-first kanban task management TUI written in Go.

Projects are first-class: each project owns its own kanban board, and you can create, rename, switch, and delete projects from inside the TUI.

## Why this shape

The app is built around a normalized board model:

- tasks are stored once in a `map[id]*Task`
- each column keeps an ordered slice of task IDs
- the UI filters into visible ID slices and only renders the current window of cards

That keeps state updates cheap and avoids repainting entire task sets when boards get large.

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
KANBAN_TUI_WHITEBOARD_CMD=rnote
```

Existing JSON data is migrated automatically on first run:

- the default legacy file is `board.json` in the same app config directory
- if `KANBAN_TUI_DATA_FILE` points to a legacy `.json` file, the app imports it into a sibling `.db` file with the same base name

## Keys

- `h` / `l`: switch columns
- `j` / `k`: move selection
- `H` / `L`: move active column left/right
- `c`: add custom column
- `p`: open project manager
- `D`: jump into the daily board (press again to jump back)
- `r`: rename active column
- `d`: delete active column (at least one column always kept; tasks are moved to the nearest adjacent column)
- `n`: create task
- `/`: search tasks
- `e`: edit selected task
- `[` / `]`: move task left or right across columns
- `J` / `K`: reorder inside a column
- `enter`: open task details
- `x`: delete selected task
- `A`: archive selected task (with confirm)
- `ctrl+a`: archive Done tasks not updated in more than 30 days (with confirm)
- `z`: open archive view
- `?`: toggle help
- `q`: quit

Inside task details:

- `w`: open the whiteboard manager for the selected task
- `e`: edit task
- `esc`: close

Inside the whiteboard manager:

- `j` / `k`: move between whiteboards
- `n`: create a new whiteboard and open it immediately
- `enter` / `o`: open the selected whiteboard
- `r`: rename the selected whiteboard
- `x`: delete the selected whiteboard and its `.rnote` file
- `esc`: close

Inside the create dialog:

- `tab`: switch fields
- `:w`: save
- `:wq`: save and close
- `esc`: cancel

Inside the archive view:

- `j` / `k`: move between archived tasks
- `/`: filter archived tasks (title + description)
- `enter`: open read-only task details
- `r`: restore selected task to its original column (or the first column if it was deleted)
- `esc`: close

Inside the project manager:

- `j` / `k`: move between projects
- `enter`: switch to selected project
- `n`: create project
- `e`: rename selected project
- `x`: delete selected project
- `esc`: close

## Daily board

The daily board is a single, always-present board for the small tasks that come up during the day. It sits next to your projects but never shows up in the project manager, and it cannot be renamed or deleted.

It has three fixed columns:

```text
Waiting   Active   Next
```

Columns cannot be added, renamed, reordered, or deleted there.

Keys inside the daily board:

- `D`: leave the daily board and go back to the last project you were on
- `n`: capture a new task in Waiting
- `[` / `]`: move a task between Waiting, Active and Next
- `space`: mark the selected task done - it disappears from the board and moves to the archive
- `P`: promote the selected task into a project column
- `z`: view the tasks you already marked done (`r` restores one to the board)
- `X`: clear the board, deleting every daily task including the done ones (with confirm)

## Whiteboards

Each task can own multiple named whiteboards. New whiteboards are created with automatic names like `Whiteboard 1`, `Whiteboard 2`, and so on, but you can rename them later from the whiteboard manager.

Creating a whiteboard requires `rnote-cli` to be available on `PATH`, because the app uses it to generate a valid `.rnote` file before opening it in Rnote.

Whiteboard files use stable paths:

```text
whiteboards/<project-slug>/<task-id>/<whiteboard-slug>.rnote
```

Renaming a whiteboard updates the display name in the TUI only. Deleting a whiteboard removes both the task link and the underlying `.rnote` file.
