# 06 — Task editing

**What to build:** Editing a task in the detail dialog matches the mockup in every mode: focused title and description fields with the full-height accent bar and a full-cell caret, the description scrolling with the cursor, the :wq command line, the s status popover (Menu with j/k, ↵ set), the empty-title error and the unsaved-changes y/n prompt, each shown as one footer row. Complete the Text input states (focused, empty, placeholder) and build Command line and Menu. Source of truth: the frames on page 03 Task detail of design.figcraft, then DESIGN.md. The page's DEV NOTE frame describes behaviour; where it disagrees with a frame or DESIGN.md, the frame wins.

**Blocked by:** 05 — Dialog shell and task detail view

**Status:** ready-for-agent

- [ ] Every frame listed here is enforced by the frame-matching check: zero cell differences (glyph, foreground token, background token, bold) at the frame's size in Dusk dark: 03/edit-title, 03/edit-description, 03/empty-title, 03/unsaved, 03/status-picker, 03/command-line, 03/command-line-80x24
- [ ] Key hints in every keybar and dialog footer follow the Fixed Order Rule in DESIGN.md
- [ ] Existing tests still pass, updated only where the design deliberately changes behaviour; board.db / board.json format unchanged
- [ ] Every Text input variant, Command line and Menu match their 00 Foundations variants in dark and in the light preview
- [ ] Vim editing behaviour that exists today keeps working in both fields
