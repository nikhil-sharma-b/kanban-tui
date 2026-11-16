# 03 · Task detail / edit (modeDetail)

## States
- View, editing title, editing description (multi-line), status picker, whiteboards list, created/updated, copy-ID confirmation, unsaved changes, validation (empty title).

## Constraints (all tickets)
- Direction: **05 · Dusk** in `design.figcraft` (card-per-task, plum-dark, rose accent `#D4A5C7`, Menlo).
- Design only in FigCraft; one page per ticket, frames named `<ticket>/<state>`.
- Every frame at **120×36 cells (1280×760)** plus the **80×24** variant where layout changes.
- Narrow layout follows the **breakpoint rule** (frame `02/breakpoint-rule`): N = clamp(floor(W/28), 1, 4) visible columns; dialogs min(60, W−8) cells wide.
- **No data model changes.** Use only fields that exist today: task id/title/description/status/whiteboards/created/updated/archived, columns, projects, daily board. Anything new (theme, prefs) lives in a separate optional config file, never in `board.db`/`board.json`.
- Show keybinds for every action on the screen.

## Decision
- Page `03 Task detail`: `03/view`, `03/edit-title`, `03/edit-description`, `03/command-line`, `03/status-picker`, `03/whiteboards`, `03/copied`, `03/unsaved`, `03/empty-title`, `03/no-whiteboards`, plus `03/view-80x24`, `03/command-line-80x24`.
- Fixed-size dialog: min(60, W−8) × min(20, H−4) cells; no layout shift while interacting.
- No input boxes: focused field = rose accent bar + cursor. Edit in place (no hop to the create dialog); vim modes kept.
- Title row carries status + updated age in text-dim (`todo · 2d ago`); no meta line, no description label. ID only via `y`.
- Description: fixed 8-row region (7 at 80×24), scrolls (J/K, cursor-follow); `↓ N more lines` in text-faint only on overflow. Whiteboards: fixed 3-row region, hidden when empty.
- Footer is one row, always: either a message/prompt (`:wq▏`, errors, `copied a3f91c07`, discard y/n) or 3–4 mode hints + `? more`, never both.
- Rose marks focus only. Focus bar spacing matches the selected card: ½-cell bar, 1½-cell gap, text in column 2 — bar lives in a reserved gutter so text never shifts on focus.
- `s` opens a column popover; `w` focuses the whiteboard list; `y` copies the full ID.
- All colours bound to Dusk variables.
