# 01 — Put the mockup on the cell grid

**What to build:** Make design.figcraft draw exactly what a terminal can draw, so "pixel to pixel" has one meaning. Today the mockup is close but not cell-true: card titles sit on half rows, the 13px Label style makes key-hint text drift off the 8.4 px column grid, dialog text starts half a column in, carets are 0.89 rows tall, and some toasts, chips and pickers have fractional widths or positions. Fix this in the 00 Foundations components first so the change flows into every screen frame through the instances, then sweep what's left by hand. Half rows stay possible only as block-glyph rows (▄ ▀ with ▗ ▖ ▝ ▘ corners), which is how a card's inset and rounded corners reach the terminal. Every cell the app will paint must be readable from the mockup: either a text node holding the glyph, or one of a short list of shape→glyph rules written into DESIGN.md (for example card corner radius → ▗▖▝▘, dialog/menu stroke → ╭╮╰╯│─, the 4.2 px accent bar → ▌, the scrim → board repainted in text-faint). Read DESIGN.md and AGENTS.md before editing: instances only, override content only.

**Blocked by:** None — can start immediately

**Status:** done

- [x] One text size: the Label style uses the same Menlo size as Body, so every character is exactly one cell wide; DESIGN.md Typography updated
- [x] Card text starts on a whole row and at col 2 in every state; card insets and corners are block-glyph half rows; at compact density (H < 30) the insets drop as DESIGN.md says
- [x] Dialog and menu borders and padding are whole cells, so all dialog text starts on a whole column
- [x] Carets (field caret, command-line caret) are a full cell; toasts, the filter chip and the status/column pickers have whole-cell sizes and positions
- [x] An off-grid scan of every screen frame on pages 01–18 (excluding 02/breakpoint-rule and DEV NOTE frames) and of the 00 Foundations components finds zero nodes off the 8.4 × 18 grid, apart from shapes covered by a documented shape→glyph rule
- [x] DESIGN.md lists every shape→glyph rule and states how a card, dialog, menu, caret and scrim map to cells
- [x] The Components · light preview still shows every variant, with no clipped or overlapping text
- [x] Page overviews of 01–18 reviewed: nothing moved except the snapping; frame sizes unchanged (1008×648 and 672×432)
