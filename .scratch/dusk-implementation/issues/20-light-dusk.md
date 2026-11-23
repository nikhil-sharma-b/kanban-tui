# 20 — Light Dusk

**What to build:** Every screen that has a light counterpart matches it exactly in dusk-light. Page 16 frames are copies of their 01, 03, 07, 12 and 13 sources with the Dusk collection in light mode, so passing them proves the palette and that no screen paints a colour outside the tokens. Source of truth: the frames on page 16 Light Dusk of design.figcraft, then DESIGN.md. The page's DEV NOTE frame describes behaviour; where it disagrees with a frame or DESIGN.md, the frame wins.

**Blocked by:** 11 — Projects; 17 — Command palette; 18 — Focus mode; 19 — Theme setting and fallbacks

**Status:** done

- [x] Every frame listed here is enforced by the frame-matching check with zero cell differences in Dusk light: 16/board, 16/board-80x24, 16/task-detail, 16/task-detail-80x24, 16/projects, 16/projects-80x24, 16/palette, 16/palette-80x24, 16/focus, 16/focus-80x24
- [x] Every 00 Foundations component matches the Components · light preview
- [x] No rendering code chooses a colour except through the palette

The wide palette frame shows `12/typing`. The light task detail copies' overflow counts and narrow card preview were aligned with their shared description and page 03; page 13 focus timestamps now use page 16's `1d ago` in both themes.
