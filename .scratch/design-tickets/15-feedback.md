# 15 · Feedback: toasts, saving, live reload, errors

## States
- Saved, copied ID, moved, archived with undo hint, external change reloaded (CLI write), save error, store unreadable.

## Constraints (all tickets)
- Direction: **05 · Dusk** in `design.figcraft` (card-per-task, plum-dark, rose accent `#D4A5C7`, Menlo).
- Design only in FigCraft; one page per ticket, frames named `<ticket>/<state>`.
- Every frame at **120×36 cells (1280×760)** plus the **80×24** variant where layout changes.
- Narrow layout follows the **breakpoint rule** (frame `02/breakpoint-rule`): N = clamp(floor(W/28), 1, 4) visible columns; dialogs min(60, W−8) cells wide.
- **No data model changes.** Use only fields that exist today: task id/title/description/status/whiteboards/created/updated/archived, columns, projects, daily board. Anything new (theme, prefs) lives in a separate optional config file, never in `board.db`/`board.json`.
- Show keybinds for every action on the screen.
