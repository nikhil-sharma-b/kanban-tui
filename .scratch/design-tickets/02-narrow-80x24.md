# 02 · Narrow layout (80×24)

Decide how the board behaves when 4 columns do not fit: e.g. a single focused column with column tabs, or 2 columns plus counts.

## States
- Board at 80×24, moving between columns, selected task with meta, dialogs at 80×24.

## Done when
- Breakpoint rule is written (at which width the layout switches) and every later ticket references it.

## Constraints (all tickets)
- Direction: **05 · Dusk** in `design.figcraft` (card-per-task, plum-dark, rose accent `#D4A5C7`, Menlo).
- Design only in FigCraft; one page per ticket, frames named `<ticket>/<state>`.
- Every frame at **120×36 cells (1280×760)** plus the **80×24** variant where layout changes.
- **No data model changes.** Use only fields that exist today: task id/title/description/status/whiteboards/created/updated/archived, columns, projects, daily board. Anything new (theme, prefs) lives in a separate optional config file, never in `board.db`/`board.json`.
- Show keybinds for every action on the screen.

## Decision
- 2 columns + edge counts at 80×24 (not tabs). Rule lives in frame `02/breakpoint-rule` on page `02 Narrow`:
  - N = clamp(floor(W/28), 1, 4) visible columns; window slides with h/l; hidden columns are named at the edges in the faint tone (`‹ backlog │ todo · 3`, `doing · done ›`; 3+ hidden → `doing +2 ›`).
  - Dialogs min(60, W−8) cells, centred over a scrim.
