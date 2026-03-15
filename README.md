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

Existing JSON data is migrated automatically on first run:

- the default legacy file is `board.json` in the same app config directory
- if `KANBAN_TUI_DATA_FILE` points to a legacy `.json` file, the app imports it into a sibling `.db` file with the same base name

## Keys

- `h` / `l`: switch columns
- `j` / `k`: move selection
- `H` / `L`: move active column left/right
- `c`: add custom column
- `p`: open project manager
- `r`: rename active column
- `d`: delete active column (at least one column always kept; tasks are moved to the nearest adjacent column)
- `n`: create task
- `/`: search tasks
- `e`: edit selected task
- `[` / `]`: move task left or right across columns
- `J` / `K`: reorder inside a column
- `enter`: open task details
- `x`: delete selected task
- `?`: toggle help
- `q`: quit

Inside the create dialog:

- `tab`: switch fields
- `ctrl+s`: save
- `esc`: cancel

Inside the project manager:

- `j` / `k`: move between projects
- `enter`: switch to selected project
- `n`: create project
- `e`: rename selected project
- `x`: delete selected project
- `esc`: close
