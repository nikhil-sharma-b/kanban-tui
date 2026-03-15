# kanban-tui

Keyboard-first kanban task management TUI written in Go.

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

The default data file is:

```text
$XDG_CONFIG_HOME/kanban-tui/board.json
```

On macOS that resolves to:

```text
~/Library/Application Support/kanban-tui/board.json
```

Override it with:

```bash
KANBAN_TUI_DATA_FILE=/path/to/board.json go run ./cmd/kanban
```

## Keys

- `h` / `l`: switch columns
- `j` / `k`: move selection
- `n`: create task
- `/`: search tasks
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
