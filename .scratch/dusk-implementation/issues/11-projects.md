# 11 — Projects

**What to build:** SPC p p opens the project switcher with a focused live name filter (↑↓ move, ↵ open), task counts including done, and the current marker beside the count following the active project. Esc clears the query and returns to management, where n creates, e renames, x deletes (through the confirm pattern) and esc closes; from the board SPC p n / r / x act on the current project. The daily board is not listed, and the last regular project offers no delete. Source of truth: the frames on page 07 Projects of design.figcraft, then DESIGN.md. The page's DEV NOTE frame describes behaviour; where it disagrees with a frame or DESIGN.md, the frame wins.

**Blocked by:** 09 — Confirm pattern

**Status:** done

- [x] Every frame listed here is enforced by the frame-matching check: zero cell differences (glyph, foreground token, background token, bold) at the frame's size in Dusk dark: 07/switcher, 07/single-project, 07/no-matches, 07/filtering, 07/actions, 07/new, 07/rename, 07/delete, 07/switcher-80x24, 07/actions-80x24, 07/filtering-80x24, 07/no-matches-80x24, 07/single-project-80x24, 07/new-80x24, 07/rename-80x24, 07/delete-80x24
- [x] Key hints in every keybar and dialog footer follow the Fixed Order Rule in DESIGN.md
- [x] Existing tests still pass, updated only where the design deliberately changes behaviour; board.db / board.json format unchanged
- [x] Deleting the last regular project is impossible from every entry point
