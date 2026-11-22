# 13 — Archive

**What to build:** SPC a v opens the archive Dialog fixed at 7 rows, newest first, each row titled with "from <column> · <when>" (relative under 7 days, else a date) and the title reading archive · N (or M of N while searching). j k move, ↵ opens the read-only detail, r restores at once (✓ restored “title” to <column>, falling back to the first column with "(deleted)" when the source column is gone), / searches, a archives old done tasks through the confirm pattern (or shows "no done tasks past 30 days" without asking), esc closes. ↵ no longer closes detail; esc backs out. Source of truth: the frames on page 10 Archive of design.figcraft, then DESIGN.md. The page's DEV NOTE frame describes behaviour; where it disagrees with a frame or DESIGN.md, the frame wins.

**Blocked by:** 09 — Confirm pattern

**Status:** ready-for-agent

- [ ] Every frame listed here is enforced by the frame-matching check: zero cell differences (glyph, foreground token, background token, bold) at the frame's size in Dusk dark: 10/list, 10/searching, 10/no-match, 10/empty, 10/detail, 10/restored, 10/restore-fallback, 10/archive-old, 10/archived-old, 10/archive-old-none, 10/list-80x24, 10/searching-80x24, 10/detail-80x24, 10/archive-old-80x24, 10/empty-80x24
- [ ] Key hints in every keybar and dialog footer follow the Fixed Order Rule in DESIGN.md
- [ ] Existing tests still pass, updated only where the design deliberately changes behaviour; board.db / board.json format unchanged
- [ ] Uses only archived_at, archived_from and updated_at; no new stored fields
