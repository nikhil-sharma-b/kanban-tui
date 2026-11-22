# 18 — Focus mode

**What to build:** z on the board opens focus mode on the selected task, and z or esc returns with the same task selected. A row of Column headers with counts replaces the columns (the task's column active); the task sheet is 84 cells wide and centred (full width at 80×24) with the detail title row, a fixed scrolling description region (J/K) and whiteboards when present, plus an "up next" preview. j k step through the column, ↵ opens detail in place, SPC t l moves right (labelled done next to the last column), then focus advances and a success toast names the moved task. With nothing in doing, the sheet lists the left column's cards. Purely a view: nothing new is stored. Source of truth: the frames on page 13 Focus mode of design.figcraft, then DESIGN.md. The page's DEV NOTE frame describes behaviour; where it disagrees with a frame or DESIGN.md, the frame wins.

**Blocked by:** 15 — Feedback: toasts, undo, sync states

**Status:** ready-for-agent

- [ ] Every frame listed here is enforced by the frame-matching check: zero cell differences (glyph, foreground token, background token, bold) at the frame's size in Dusk dark: 13/doing, 13/doing-80x24, 13/advanced, 13/no-doing
- [ ] Key hints in every keybar and dialog footer follow the Fixed Order Rule in DESIGN.md
- [ ] Existing tests still pass, updated only where the design deliberately changes behaviour; board.db / board.json format unchanged
