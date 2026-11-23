# 21 — First run and cheat sheet

**What to build:** A fresh workspace opens on the empty Personal board with three hints in the active column (SPC t n add your first task · SPC p projects · ? all keys) and ↵ open dropped from the keybar while there is nothing to open. After creating a project the app switches to it with the same hints and a success toast. While the board holds exactly one task, the hints move under its card (↵ open it · SPC t l move it along); from the second task on they disappear. ? opens a short cheat sheet in the palette dialog ("start here" first, type to filter, ↵ runs, esc closes). Hints come from board state only; nothing about onboarding is stored. Source of truth: the frames on page 18 First run of design.figcraft, then DESIGN.md. The page's DEV NOTE frame describes behaviour; where it disagrees with a frame or DESIGN.md, the frame wins.

**Blocked by:** 11 — Projects

**Status:** done

- [x] Every frame listed here is enforced by the frame-matching check: zero cell differences (glyph, foreground token, background token, bold) at the frame's size in Dusk dark: 18/no-projects, 18/no-projects-80x24, 18/first-project, 18/first-task, 18/cheat-sheet, 18/cheat-sheet-80x24
- [x] Key hints in every keybar and dialog footer follow the Fixed Order Rule in DESIGN.md
- [x] Existing tests still pass, updated only where the design deliberately changes behaviour; board.db / board.json format unchanged
