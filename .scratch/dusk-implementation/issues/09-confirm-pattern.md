# 09 — Confirm pattern

**What to build:** Every destructive or bulk action goes through one confirm Dialog in two tiers. Tier 1 (delete task, delete whiteboard, delete an empty column, archive task, archive old done tasks): footer esc keep · y <verb>, only y acts, only esc keeps, ↵ does nothing (today ↵ confirms: change it). Tier 2 (delete project, delete non-empty column, clear daily board): a focused field under the body, info "type “name” to delete" and footer esc keep until the typed text matches exactly (case-sensitive, trimmed), then danger "↵ deletes N tasks for good" and footer ↵ delete · esc keep. The body always states what is lost with real counts. Source of truth: the frames on page 09 Confirm of design.figcraft, then DESIGN.md. The page's DEV NOTE frame describes behaviour; where it disagrees with a frame or DESIGN.md, the frame wins.

**Blocked by:** 06 — Task editing

**Status:** ready-for-agent

- [ ] Every frame listed here is enforced by the frame-matching check: zero cell differences (glyph, foreground token, background token, bold) at the frame's size in Dusk dark: 09/delete-task, 09/delete-column, 09/delete-column-matched, 09/delete-project, 09/delete-project-matched, 09/delete-project-typing, 09/delete-whiteboard, 09/clear-daily, 09/clear-daily-matched, 09/archive-task, 09/archive-old, 09/delete-task-80x24, 09/delete-project-80x24, 09/archive-old-80x24
- [x] Key hints in every keybar and dialog footer follow the Fixed Order Rule in DESIGN.md
- [x] Existing tests still pass, updated only where the design deliberately changes behaviour; board.db / board.json format unchanged
- [x] Tests cover: ↵ is inert in tier 1; tier 2 acts only after an exact match; esc keeps in both tiers

**Open:** 12 of 14 frames enforced. 09/clear-daily and 09/clear-daily-matched now show the 11/daily board behind the dialog, so they match the app once ticket 14 brings the daily board in line with 11/daily. Until then only their dialog is enforced (TestClearDailyDialog); ticket 14 should move them to enforced.
