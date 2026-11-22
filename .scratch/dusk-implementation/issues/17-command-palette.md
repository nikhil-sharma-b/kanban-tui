# 17 — Command palette

**What to build:** : on the board opens the command palette for the selected task: a fixed-size Dialog listing every action with its keys from the leader bindings table, fuzzy-matched on label and key, with the last 3 actions run this session (in memory only) as recent. Actions ending in … open a second step in the same dialog (esc goes back a step), the list scrolls with ↓ N more while the footer stays put, no match offers / search tasks with the query carried over, and from task detail (view mode only) the palette takes over the detail dialog in place and esc swaps detail back. Source of truth: the frames on page 12 Command palette of design.figcraft, then DESIGN.md. The page's DEV NOTE frame describes behaviour; where it disagrees with a frame or DESIGN.md, the frame wins.

**Blocked by:** 06 — Task editing

**Status:** ready-for-agent

- [ ] Every frame listed here is enforced by the frame-matching check: zero cell differences (glyph, foreground token, background token, bold) at the frame's size in Dusk dark: 12/empty, 12/second-step, 12/no-match, 12/typing, 12/from-dialog, 12/empty-80x24, 12/second-step-80x24
- [ ] Key hints in every keybar and dialog footer follow the Fixed Order Rule in DESIGN.md
- [ ] Existing tests still pass, updated only where the design deliberately changes behaviour; board.db / board.json format unchanged
- [ ] A binding added to the leader table appears in the palette with no other change
- [ ] While a detail field is being edited, : stays the vim command line
