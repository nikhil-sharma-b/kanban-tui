# 19 — Theme setting and fallbacks

**What to build:** The theme comes from an optional config.toml beside the data (following the data file's directory when it is overridden), with one key: theme = "dusk" | "dusk-light" | "terminal". A missing file or key means dusk; an unknown value means dusk plus an info toast. terminal asks the terminal for its background (falling back to COLORFGBG) and picks dusk or dusk-light, defaulting to dusk. Independently, a terminal without 256-colour or truecolor support gets the Dusk ansi16 mode from DESIGN.md. The palette gains a theme action (: → theme) that applies live, marks the current theme disabled with "current" and writes config.toml; hand edits are picked up on start and by live reload. Source of truth: the frames on page 17 Theme setting of design.figcraft, then DESIGN.md. The page's DEV NOTE frame describes behaviour; where it disagrees with a frame or DESIGN.md, the frame wins.

**Blocked by:** 17 — Command palette

**Status:** done

- [x] Every frame listed here is enforced by the frame-matching check: zero cell differences (glyph, foreground token, background token, bold) at the frame's size in Dusk dark: 17/dusk, 17/dusk-light, 17/terminal, 17/dusk-ansi16, 17/palette-theme, 17/palette-theme-80x24 (each in the theme it shows)
- [x] Key hints in every keybar and dialog footer follow the Fixed Order Rule in DESIGN.md
- [x] Existing tests still pass, updated only where the design deliberately changes behaviour; board.db / board.json format unchanged
- [x] Nothing theme-related is written to board.db / board.json
- [x] Tests cover missing file, missing key, unknown value, each valid value, terminal detection with and without an answer, and the ansi16 fallback
