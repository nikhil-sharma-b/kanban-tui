# 14 — Daily board and promote

**What to build:** The daily board matches the mockup with its own keybar (SPC d m done · SPC d p promote · SPC d d back; narrow keeps SPC d p promote). Promote is daily → project only, in two steps inside one fixed-size Dialog: pick a project (filter, ↑↓, cursor starting on the last board, hinted "last board"), then pick a column (j k); esc in step 2 goes back to step 1. After promoting, the app stays on daily, selection moves to the next card, and a success Toast names the task and its new home. The empty daily board and the defensive no-columns fallback match too. Source of truth: the frames on page 11 Daily of design.figcraft, then DESIGN.md. The page's DEV NOTE frame describes behaviour; where it disagrees with a frame or DESIGN.md, the frame wins.

**Blocked by:** 06 — Task editing

**Status:** ready-for-agent

- [ ] Every frame listed here is enforced by the frame-matching check: zero cell differences (glyph, foreground token, background token, bold) at the frame's size in Dusk dark: 11/daily, 11/empty, 11/promoted, 11/promote-project, 11/no-columns-fallback, 11/promote-column, 11/daily-80x24, 11/empty-80x24, 11/promote-column-80x24, 11/promote-project-80x24
- [ ] Key hints in every keybar and dialog footer follow the Fixed Order Rule in DESIGN.md
- [ ] Existing tests still pass, updated only where the design deliberately changes behaviour; board.db / board.json format unchanged
- [ ] Promote targets are the same set the app offers today, only grouped by project
