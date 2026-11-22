# 08 — Search and filter

**What to build:** / opens the search dialog with live, case-insensitive title and description matches previewed and highlighted wherever visible; ↵ applies the filter (even with no matches), esc in search restores the previous query, and on the board the filter persists as a Badge chip in the header, the keybar swaps / search for / refine and SPC more for esc clear filter, and esc clears the applied filter (a planned key in the DESIGN.md keybind map). Counts include hidden columns. At 80×24 the footer compacts without dropping / refine, esc clear or ? help. Source of truth: the frames on page 05 Search and filter of design.figcraft, then DESIGN.md. The page's DEV NOTE frame describes behaviour; where it disagrees with a frame or DESIGN.md, the frame wins.

**Blocked by:** 06 — Task editing

**Status:** ready-for-agent

- [ ] Every frame listed here is enforced by the frame-matching check: zero cell differences (glyph, foreground token, background token, bold) at the frame's size in Dusk dark: 05/typing, 05/live-filtered, 05/no-results, 05/filter-persisting, 05/cleared, 05/typing-80x24, 05/no-results-80x24, 05/filter-persisting-80x24, 05/live-filtered-80x24, 05/cleared-80x24, 01/filter-active
- [ ] Key hints in every keybar and dialog footer follow the Fixed Order Rule in DESIGN.md
- [ ] Existing tests still pass, updated only where the design deliberately changes behaviour; board.db / board.json format unchanged
- [ ] The filter lives in memory only; nothing is written to the store
