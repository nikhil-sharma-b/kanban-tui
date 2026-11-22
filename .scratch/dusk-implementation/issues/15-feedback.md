# 15 — Feedback: toasts, undo, sync states

**What to build:** Feedback reaches the user the way page 15 shows it. Toasts sit bottom-right one blank row above the keybar (above the column rail at 80×24), clear after 3 s or on the next key, and a newer one replaces the old; copy-ID, save and live-reload messages route to them. New: u undo (single level, the last move or archive, only while its toast shows), automatic retry of failed saves with the header reading "save failed · retrying" until a save succeeds (see the save-retry DEV NOTE on page 01), the reloaded header state after an external write (selection kept by task id), and a store-unreadable screen where r retries the open, q quits and nothing is ever written to the unreadable store. Source of truth: the frames on page 15 Feedback of design.figcraft, then DESIGN.md. The page's DEV NOTE frame describes behaviour; where it disagrees with a frame or DESIGN.md, the frame wins.

**Blocked by:** 05 — Dialog shell and task detail view

**Status:** ready-for-agent

- [ ] Every frame listed here is enforced by the frame-matching check: zero cell differences (glyph, foreground token, background token, bold) at the frame's size in Dusk dark: 15/saved, 15/archived-undo, 15/moved, 15/copied-id, 15/archived-undo-80x24, 15/store-unreadable-80x24, 15/save-error, 15/external-reload, 15/store-unreadable, 01/sync-error, 01/sync-reloaded
- [ ] Key hints in every keybar and dialog footer follow the Fixed Order Rule in DESIGN.md
- [ ] Existing tests still pass, updated only where the design deliberately changes behaviour; board.db / board.json format unchanged
- [ ] App header (Sync=error, Sync=reloaded) and Toast with its key hint match their 00 Foundations variants in dark and in the light preview
- [ ] Tests cover undo of a move and of an archive, undo being unavailable once the toast is gone, save retry until success, and that an unreadable store is never written
