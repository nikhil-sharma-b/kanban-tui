# 02 — Frame-matching test and the empty board

**What to build:** Build the check every later ticket is measured by, and prove it end to end on the simplest screen. The check reads design.figcraft directly (no FigCraft app needed), turns a named frame into the cell grid it expects (glyph, foreground token, background token, bold per cell, with colours resolved through the Dusk variable bindings, not raw hex, and shapes mapped through the DESIGN.md shape→glyph rules). It then drives the real app into the same state through its normal update loop (a fixture workspace, the frame's terminal size, a theme, a fixed clock and a sequence of key presses), reads the rendered screen back into the same grid and compares every cell. A ledger in the repo lists each frame as pending or enforced: enforced frames must match exactly, pending frames are reported but don't fail, and each later ticket moves its frames to enforced. The same check can compare a single 00 Foundations component variant, in dark against the Components frame and in light against the Components · light preview. Ship with it the Dusk palette (dark, light and ansi16 values from DESIGN.md "Tokens → lipgloss"), the shared sample workspace that reproduces the data shown in 01/default (projects, columns, tasks, IDs and timestamps relative to the fixed clock), and the board chrome needed for 01/empty-board: App header (Sync=live), Column header (normal, active), Keybar (wide) and Key hint.

**Blocked by:** 01 — Put the mockup on the cell grid

**Status:** ready-for-agent

- [ ] One command runs the check and writes, for each failing frame, a readable report: expected rows, actual rows and the list of differing cells with what each expected
- [ ] The ledger lists every screen frame on pages 01–18 (not 02/breakpoint-rule or DEV NOTE frames), all pending except 01/empty-board
- [ ] 01/empty-board is enforced with zero cell differences at 120×36 in Dusk dark
- [ ] App header (Sync=live), Column header (normal, active), Keybar (wide) and Key hint match their 00 Foundations variants in dark and in the light preview
- [ ] Colours come from the Dusk palette everywhere this ticket touches; no raw colour values in rendering code
- [ ] A deliberate one-cell change to the render makes the check fail with a report naming that cell
- [ ] Existing tests still pass; board.db / board.json format unchanged
