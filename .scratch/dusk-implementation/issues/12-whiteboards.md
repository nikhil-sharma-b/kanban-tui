# 12 — Whiteboards

**What to build:** From a task (SPC t w, or w in detail) the whiteboards list matches the mockup: open in the editor, rename, delete through the confirm pattern, and the empty state. Source of truth: the frames on page 08 Whiteboards of design.figcraft, then DESIGN.md. The page's DEV NOTE frame describes behaviour; where it disagrees with a frame or DESIGN.md, the frame wins.

**Blocked by:** 09 — Confirm pattern

**Status:** ready-for-agent

- [ ] Every frame listed here is enforced by the frame-matching check: zero cell differences (glyph, foreground token, background token, bold) at the frame's size in Dusk dark: 08/list, 08/empty, 08/delete, 08/rename, 08/open-in-editor, 08/list-80x24, 08/open-in-editor-80x24, 08/rename-80x24, 08/delete-80x24, 08/empty-80x24
- [ ] Key hints in every keybar and dialog footer follow the Fixed Order Rule in DESIGN.md
- [ ] Existing tests still pass, updated only where the design deliberately changes behaviour; board.db / board.json format unchanged
- [ ] Opening a whiteboard still launches the external editor as it does today
