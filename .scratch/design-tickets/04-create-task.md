# 04 · New task (modeCreate)

## States
- Quick-add inline in a column vs dialog (pick one and justify), with description, target column, error, success feedback.

## Constraints (all tickets)
- Direction: **05 · Dusk** in `design.figcraft` (card-per-task, plum-dark, rose accent `#D4A5C7`, Menlo).
- Design only in FigCraft; one page per ticket, frames named `<ticket>/<state>`.
- Every frame at **120×36 cells (1280×760)** plus the **80×24** variant where layout changes.
- Narrow layout follows the **breakpoint rule** (frame `02/breakpoint-rule`): N = clamp(floor(W/28), 1, 4) visible columns; dialogs min(60, W−8) cells wide.
- **No data model changes.** Use only fields that exist today: task id/title/description/status/whiteboards/created/updated/archived, columns, projects, daily board. Anything new (theme, prefs) lives in a separate optional config file, never in `board.db`/`board.json`.
- Show keybinds for every action on the screen.

## Decision
- Page `04 New task`: `04/empty`, `04/typing`, `04/description`, `04/column-picker`, `04/command-line`, `04/empty-title`, `04/created`, plus `04/typing-80x24`, `04/created-80x24`.
- Dialog, not inline quick-add: reuses the 03 detail dialog (same size, focus bar, footer rules). A 28-cell card can't hold a description or column choice, and one dialog for create + detail means one set of keys.
- Quick-add speed via `↵` in the title field: creates right away. `tab` → description (`↵` = newline, `:wq` creates, `ctrl+e` $EDITOR).
- Proposal: target column defaults to the focused column (today: always the first). Shown as `new · todo` in the title row; `s` (normal mode) opens the 03 column popover.
- No whiteboards region on create; text-faint placeholders (`title`, `description (optional)`) after the cursor.
- Error `✗ title can't be empty` replaces the footer hints; dialog stays open.
- Success: dialog closes, new card appended to the target column and selected, count updates, keybar shows `✓ created 7c2e0b14 in todo` until the next key.
- No data model changes: title, description, status only.
