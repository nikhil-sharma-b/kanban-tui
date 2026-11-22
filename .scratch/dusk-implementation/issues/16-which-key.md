# 16 — Which-key

**What to build:** After SPC (300 ms) the which-key popover appears as a Menu bottom-right, one row above the keybar, only as tall as its entries and growing upward from a fixed bottom edge (root 6 rows, a group up to 8); it flips to bottom-left when it would cover the selected card, and at 80×24 it sits on the keybar. Groups come first, then actions in table order, all read from the leader bindings table so new bindings appear automatically. The typed path echoes in the keybar as a prompt Status message, esc cancels silently, backspace drops the last key, and an unbound key cancels with an info message in the keybar. No scrim. Source of truth: the frames on page 14 Which-key of design.figcraft, then DESIGN.md. The page's DEV NOTE frame describes behaviour; where it disagrees with a frame or DESIGN.md, the frame wins.

**Blocked by:** 06 — Task editing; 15 — Feedback: toasts, undo, sync states

**Status:** ready-for-agent

- [ ] Every frame listed here is enforced by the frame-matching check: zero cell differences (glyph, foreground token, background token, bold) at the frame's size in Dusk dark: 14/root, 14/unbound, 14/pending, 14/group-task, 14/group-task-80x24, 14/root-80x24, 14/placement-flip
- [ ] Key hints in every keybar and dialog footer follow the Fixed Order Rule in DESIGN.md
- [ ] Existing tests still pass, updated only where the design deliberately changes behaviour; board.db / board.json format unchanged
- [ ] Popover placement is computed from the selected card position, covered by tests for the normal, flipped and 80×24 cases
