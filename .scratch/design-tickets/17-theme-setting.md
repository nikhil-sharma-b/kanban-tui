# 17 · Theme setting and fallbacks

A simple config switch: theme = dusk | dusk-light | terminal (auto picks dusk or dusk-light from terminal background). Lives in a new optional config file next to the data; missing file means defaults.

## Deliverables
- Board rendered in all three modes, plus the 16-colour fallback of Dusk.
- Where the setting is shown or changed in the app (palette action and/or config file only).

## Constraints (all tickets)
- Direction: **05 · Dusk** in `design.figcraft` (card-per-task, plum-dark, rose accent `#D4A5C7`, Menlo).
- Design only in FigCraft; one page per ticket, frames named `<ticket>/<state>`.
- Every frame at **120×36 cells (1280×760)** plus the **80×24** variant where layout changes.
- **No data model changes.** Use only fields that exist today: task id/title/description/status/whiteboards/created/updated/archived, columns, projects, daily board. Anything new (theme, prefs) lives in a separate optional config file, never in `board.db`/`board.json`.
- Show keybinds for every action on the screen.
