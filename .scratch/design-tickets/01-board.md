# 01 · Board (main screen)

Refine variant 05 into the final board.

## States
- Default (current mock), empty column ('Press n to add a task'), empty board, overflowing column with scroll indicators, more columns than fit (horizontal scroll), active filter shown, live-reload/sync status in header, project name in header.

## Done when
- All states built from 00 components; header and footer content final.

## Constraints (all tickets)
- Direction: **05 · Dusk** in `design.figcraft` (card-per-task, plum-dark, rose accent `#D4A5C7`, Menlo).
- Design only in FigCraft; one page per ticket, frames named `<ticket>/<state>`.
- Every frame at **120×36 cells (1280×760)** plus the **80×24** variant where layout changes.
- **No data model changes.** Use only fields that exist today: task id/title/description/status/whiteboards/created/updated/archived, columns, projects, daily board. Anything new (theme, prefs) lives in a separate optional config file, never in `board.db`/`board.json`.
- Show keybinds for every action on the screen.
