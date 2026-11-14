# 00 · Foundations: tokens and component kit

Build the system every other ticket uses.

## Deliverables
- Colour tokens: Dusk dark, Dusk light, a 16-colour ANSI mapping, and a **terminal theme** mode (inherits the user's colours; only bold/dim/reverse for emphasis).
- Cell grid: spacing in character cells, Menlo at 14px = 1 cell; corner/border glyph rules (what a 'card' is in real terminal characters).
- Components: column header, card, selected card, done card, footer keybar, dialog shell, text input, list row (normal/selected/disabled), badge/count, toast, scroll indicators.

## Done when
- Components exist as FigCraft components with variants and are used by ticket 01.

## Constraints (all tickets)
- Direction: **05 · Dusk** in `design.figcraft` (card-per-task, plum-dark, rose accent `#D4A5C7`, Menlo).
- Design only in FigCraft; one page per ticket, frames named `<ticket>/<state>`.
- Every frame at **120×36 cells (1280×760)** plus the **80×24** variant where layout changes.
- **No data model changes.** Use only fields that exist today: task id/title/description/status/whiteboards/created/updated/archived, columns, projects, daily board. Anything new (theme, prefs) lives in a separate optional config file, never in `board.db`/`board.json`.
- Show keybinds for every action on the screen.
