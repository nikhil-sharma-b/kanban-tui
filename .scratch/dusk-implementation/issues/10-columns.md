# 10 — Columns

**What to build:** SPC c n adds a column through the Dialog, SPC c r renames the selected column in place in its header (Column header editing state, ↵ rename · esc cancel), a duplicate name keeps the draft editable and shows ✗ column “name” already exists under the field without changing the board, and SPC c d deletes through the confirm pattern (tasks move left, or right for the first column). Source of truth: the frames on page 06 Columns of design.figcraft, then DESIGN.md. The page's DEV NOTE frame describes behaviour; where it disagrees with a frame or DESIGN.md, the frame wins.

**Blocked by:** 09 — Confirm pattern

**Status:** ready-for-agent

- [ ] Every frame listed here is enforced by the frame-matching check: zero cell differences (glyph, foreground token, background token, bold) at the frame's size in Dusk dark: 06/add, 06/rename-in-place, 06/delete-non-empty, 06/duplicate-name, 06/add-80x24, 06/rename-in-place-80x24, 06/delete-non-empty-80x24, 06/duplicate-name-80x24
- [ ] Key hints in every keybar and dialog footer follow the Fixed Order Rule in DESIGN.md
- [ ] Existing tests still pass, updated only where the design deliberately changes behaviour; board.db / board.json format unchanged
- [ ] Column header (editing) matches its 00 Foundations variant in dark and in the light preview
