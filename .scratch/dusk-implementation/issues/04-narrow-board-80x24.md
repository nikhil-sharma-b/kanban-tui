# 04 — Narrow board at 80×24

**What to build:** Below full width the board follows the breakpoint rule: N = clamp(floor(W/28), 1, 4) visible columns, a window that slides with h/l, hidden columns named in the Column rail (mirrored: ‹ backlog on the left edge, doing · done › on the right, 3+ on a side collapse to doing +2 ›), the narrow keybar below 100 cols, and compact density below 30 rows (card insets dropped, one blank row between cards). Source of truth: the frames on page 02 Narrow of design.figcraft, then DESIGN.md. The page's DEV NOTE frame describes behaviour; where it disagrees with a frame or DESIGN.md, the frame wins. 02/breakpoint-rule is the reference sheet for the rule.

**Blocked by:** 03 — Board at 120×36

**Status:** done

- [x] Every frame listed here is enforced by the frame-matching check: zero cell differences (glyph, foreground token, background token, bold) at the frame's size in Dusk dark: 01/default-80x24, 02/board-80x24, 02/move-right-80x24
- [x] Key hints in every keybar and dialog footer follow the Fixed Order Rule in DESIGN.md
- [x] Existing tests still pass, updated only where the design deliberately changes behaviour; board.db / board.json format unchanged
- [x] Column rail and Keybar (narrow) match their 00 Foundations variants in dark and in the light preview
- [x] Resizing the terminal between 80×24 and 120×36 switches layout with no stale rows, and the selected task stays selected
