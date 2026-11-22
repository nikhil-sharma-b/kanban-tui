# 05 — Dialog shell and task detail view

**What to build:** Opening a task (↵) shows the fixed-size detail Dialog over the Scrim: min(60, W−8) × min(20, H−4) cells, centred on whole rows, with the board behind repainted in text-faint and fills dropped. The view shows the title row (title left, status · age right in text-dim), the fixed description region, the whiteboards region (hidden when the task has none, the description takes the space) and a one-row footer of hints or a single Status message (e.g. ✓ copied a3f91c07 after y). Build Dialog, Scrim, List row, the idle Text input states, Status message and Toast here; the Toast is checked against its component now and first appears on screens in later tickets. Source of truth: the frames on page 03 Task detail of design.figcraft, then DESIGN.md. The page's DEV NOTE frame describes behaviour; where it disagrees with a frame or DESIGN.md, the frame wins.

**Blocked by:** 04 — Narrow board at 80×24

**Status:** ready-for-agent

- [ ] Every frame listed here is enforced by the frame-matching check: zero cell differences (glyph, foreground token, background token, bold) at the frame's size in Dusk dark: 03/view, 03/no-whiteboards, 03/whiteboards, 03/copied, 03/view-80x24, 02/dialog-80x24
- [ ] Key hints in every keybar and dialog footer follow the Fixed Order Rule in DESIGN.md
- [ ] Existing tests still pass, updated only where the design deliberately changes behaviour; board.db / board.json format unchanged
- [ ] Dialog, Scrim, List row (default, selected, disabled), Text input (idle states), Status message (all kinds) and Toast (all kinds) match their 00 Foundations variants in dark and in the light preview
- [ ] The dialog never changes size while it is open, whatever the description length
