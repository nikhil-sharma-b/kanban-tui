# 07 — New task

**What to build:** SPC t n opens the same dialog as detail in create mode: ↵ in the title creates a title-only task at once, tab moves to the description (↵ is a newline there, :wq creates), s picks the target column (shown as new · todo in the title row), empty fields show text-faint placeholders, errors replace the footer hints, and on success the dialog closes, the new card is appended to the target column and selected, and the keybar shows ✓ created <id> in <column> until the next key. Source of truth: the frames on page 04 New task of design.figcraft, then DESIGN.md. The page's DEV NOTE frame describes behaviour; where it disagrees with a frame or DESIGN.md, the frame wins.

**Blocked by:** 06 — Task editing

**Status:** done

- [x] Every frame listed here is enforced by the frame-matching check: zero cell differences (glyph, foreground token, background token, bold) at the frame's size in Dusk dark: 04/empty, 04/typing, 04/description, 04/column-picker, 04/command-line, 04/empty-title, 04/created, 04/typing-80x24, 04/created-80x24
- [x] Key hints in every keybar and dialog footer follow the Fixed Order Rule in DESIGN.md
- [x] Existing tests still pass, updated only where the design deliberately changes behaviour; board.db / board.json format unchanged
- [x] Creating a task stores only title, description and status
