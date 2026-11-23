---
name: kanban-tui · Dusk
description: Keyboard-first kanban TUI. Plum-dark cards, a single rose accent, one Menlo size, every action shown as a key hint.
colors:
  bg: "#17151A"
  surface: "#1F1C23"
  surface-done: "#1B191E"
  surface-selected: "#2A2430"
  border: "#2E2A33"
  accent: "#D4A5C7"
  text: "#E2DBE6"
  text-card: "#C9C2CD"
  text-muted: "#A39BA8"
  text-dim: "#6E6675"
  text-faint: "#4A4550"
  text-done: "#4F4955"
  meta: "#9A8296"
  success: "#9FC6A6"
  danger: "#E28C8C"
typography:
  title:
    fontFamily: "Menlo, monospace"
    fontSize: "13.94px"
    fontWeight: 700
    lineHeight: "18px"
  body:
    fontFamily: "Menlo, monospace"
    fontSize: "13.94px"
    fontWeight: 400
    lineHeight: "18px"
  label:
    fontFamily: "Menlo, monospace"
    fontSize: "13.94px"
    fontWeight: 400
    lineHeight: "18px"
rounded:
  card: "4px"
  dialog: "6px"
spacing:
  cell-w: "8.4px"
  cell-h: "18px"
components:
  card:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.text-card}"
    rounded: "{rounded.card}"
  card-selected:
    backgroundColor: "{colors.surface-selected}"
    textColor: "{colors.text}"
    rounded: "{rounded.card}"
  card-done:
    backgroundColor: "{colors.surface-done}"
    textColor: "{colors.text-done}"
    rounded: "{rounded.card}"
  dialog:
    backgroundColor: "{colors.surface-selected}"
    textColor: "{colors.text}"
    rounded: "{rounded.dialog}"
  badge:
    backgroundColor: "{colors.accent}"
    textColor: "{colors.bg}"
  key-hint:
    textColor: "{colors.text-dim}"
    typography: "{typography.label}"
---

# Design System: kanban-tui · Dusk

> **For agents: read this before touching `design.figcraft`.** It is the rulebook for every page and frame. The source of truth for tokens and components is the **00 Foundations** page; this file says how to use them. If a frame disagrees with this file, fix the frame (or ask the user), never quietly invent a third way.

## Overview

**Creative North Star: "Dusk"**

A plum-dark terminal at the end of the day: quiet surfaces, one rose accent, and nothing on screen that the keyboard can't reach. It is an Operate surface. Precision and sameness beat expression; the user should learn a key once and find it in the same place on every screen.

Everything lives on a character grid. There is one type size, hierarchy comes from colour tokens and bold, and every visual effect must be something a real terminal can draw with text, block glyphs and colour.

**Key Characteristics:**
- Card-per-task board, 4 columns at 120×36, 2 at 80×24.
- Single rose accent (`accent`) used only for focus, selection and a few fixed markers (see **The One Rose Rule**).
- Every action on screen has a visible key hint, always in the same order.
- Every screen is assembled from 00 Foundations components. No one-off look-alikes.

## Colors

Plum-tinted neutrals with one rose accent. All colours are FigCraft variables in the **Dusk** collection, with three modes: `dark` (the frontmatter values), `light`, and `ansi16`. Always bind the variable; never paste a hex.

### Primary
- **Dusk Rose** (`accent`): active column header, selection bar `▌`, selected list row text, focused-field bar, current project name in the App header, active filter chip, the `●` of info messages, badge fill. Nothing else.

### Neutral
- **Night Plum** (`bg`): screen background.
- **Card Plum** (`surface`): card and menu fill.
- **Spent Plum** (`surface-done`): done-card fill.
- **Lit Plum** (`surface-selected`): selected card/row fill and dialog fill.
- **Rule Plum** (`border`): defined in every mode, but no frame paints it today. Dialog and menu borders use `text-faint`.
- **Text ladder** (`text` → `text-card` → `text-muted` → `text-dim` → `text-faint`, plus `text-done`, `meta`): see each variable's description in FigCraft for its exact role. Every Key hint, in the keybar and in dialog footers, draws its key in `text-card` and its label in `text-dim`.

### Status
- **Sage** (`success`): saved, created, copied, restored.
- **Clay Red** (`danger`): errors and destructive confirmations.

### Named Rules
**The One Rose Rule.** `accent` marks focus or selection: the active column header, `▌` bars and the text of the selected list row (a selected card keeps `text`). Four fixed markers also use it: the current project name in the App header, the active filter chip (`/ query` in `bg` on an `accent` fill), the `●` of an info Status message, Toast or `Sync=reloaded` header, and the badge fill. Nothing else is rose: no headings, no decoration.

**The Variables-Only Rule.** Every fill and text colour is bound to a Dusk variable so the `light` and `ansi16` modes work. A raw hex on a screen frame is a bug.

**The Terminal-Theme Rule.** The setting is `theme = dusk | dusk-light | terminal` in the optional `config.toml` beside `board.db` (default `dusk`). `terminal` is auto: it reads the terminal background and uses `dark` or `light` (no answer → `dark`). Separately, a terminal without truecolor/256 colours gets the `ansi16` mode: bg never painted, selected → colour 8 fill, accent → magenta, dim/faint/done → colour 8 / faint attribute, danger → ANSI 1, success → ANSI 2. Don't design anything that only works through subtle fill differences. See page `17 Theme setting`.

## Typography

**Font:** Menlo (monospace), one size: 13.94 px on an 18 px line. That is the size at which FigCraft advances every Menlo glyph, regular or bold, by exactly 8.4 px, so one character is one cell wide.

### Hierarchy
- **Title** (Menlo Bold 13.94/18, text style `Dusk/Title`): dialog titles, the selected card title, and the bold chrome: app name, active column header, badge, prompt.
- **Body** (Menlo Regular 13.94/18, `Dusk/Body`): card titles, dialog body, field labels, inline meta.
- **Label** (Menlo Regular 13.94/18, `Dusk/Hint`): key hints, scroll hints, chrome. Same size as Body; the separate style only names the role.

### Text boxes
FigCraft rounds an auto-width text box up to a whole pixel, so a run of hugging texts drifts off the 8.4 px columns. Every single-line text box is therefore fixed width = glyph count × 8.4 px (`textAutoResize: height`); on an instance that width is the one non-content override the Override-Only-Content Rule allows. Card titles, descriptions and meta fill the card's text area instead. A box that is too narrow wraps onto a second line and turns 36 px tall, which is easy to spot.

### Named Rules
**The One Size Rule.** Hierarchy is colour token + bold. No italics, no underline, no new sizes. Use the three text styles; never set a font by hand.

**The Glyph Whitelist Rule.** Allowed glyphs: `▌ ▀ ▄ ▗ ▖ ▝ ▘ › ‹ · ↑ ↓ → ● ╭ ╮ ╰ ╯ │ ─ ▏ ◇ ▾ ↵ ✓ ✗ … – “ ” ’`. `…` (U+2026, one cell) only ends truncated text. `›` separates the app name from the project and, with `‹`, points to hidden columns in the Column rail; `▾` marks a field whose picker is open (`todo ▾`); `→` means moved to (`→ doing`); `–` joins a range (`columns 2–5 of 6`); `“ ”` quote names in titles and messages, and `’` is the apostrophe in copy. No Nerd Font icons, no emoji.

## Layout

- **Cell grid:** 1 cell = 1 col × 1 row = 8.4 × 18 px. Every frame, instance and text box sits on whole cells relative to its screen frame and is whole cells in size; half rows exist only through block glyphs. The only exceptions are the shapes in **Shape → glyph rules** below.
- **Frames:** each state at 120×36 cells (1008×648), plus an 80×24 variant whenever layout changes. Name frames `<ticket>/<state>` and `<ticket>/<state>-80x24`, one page per ticket.
- **120×36 rows:** row 1 blank · row 2 header · row 4 column headers · rows 5–33 cards · row 35 keybar · row 36 blank. At 80×24 the same 1-row margin: header row 2, column rail row 21, keybar row 23. Screen gutter is 2 cols at every size (16.8 px, close to the 18 px row, so the margin looks even on all sides). Column gap is 4 cols at 120×36 and 2 at 80×24; columns are 26 cols wide at 120×36. The keybar sits on the same gutters, with hints 3 cols apart.
- **Breakpoint rule** (frame `02/breakpoint-rule`): visible columns N = clamp(floor(W/28), 1, 4). Hidden columns are named in the **Column rail**.
- **Dialogs:** always min(60, W−8) cells wide and centred on whole rows. Full-height dialogs (task detail, new task, archive detail, command palette, cheat sheet) are min(20, H−4) rows tall and never change size while open. Every other dialog takes the height its frame shows: a confirm fits its body, a project or whiteboard list fits its rows, a name dialog fits its input, and the archive list keeps one height in every state. A content-sized dialog may grow or shrink by whole rows as its content changes (a filter drops list rows, an error adds its Status message row) and is re-centred.
- **Truncation:** card titles and descriptions are one line each. Text wider than the card's text area (card width − 3 cols: 23 at 120×36, 34 at 80×24) is cut to width − 1 cols, trailing spaces dropped, and ends in `…`. lipgloss: `strings.TrimRight(ansi.Truncate(s, w-1, ""), " ") + "…"`.
- **Cards in a column:** at regular density a one-line card is 3 rows (inset row, title row, inset row) and cards stack with no gap; the half-row insets make the visible 1-row gap. The first card's top inset row sits directly under the column header (row 5 at 120×36).
- **Compact density** below 30 rows (every 80×24 frame): `Density=compact` cards, no inset rows, 1 blank row between cards and below the column header.

## Elevation & Depth

Flat. Terminals have no shadows or transparency. Depth comes from tonal layering only: `bg` → `surface` → `surface-selected`. When a dialog is open, the board behind it is repainted in `text-faint` using the **Scrim** component.

## Shapes

- **Cards:** quarter-block glyphs cut the corners (reads as 4px radius). Text always starts at col 2 in every state, so titles never shift when the selection moves. One col of right padding, so the text area is card width − 3 cols.
- **Dialogs and menus:** `╭╮╰╯` rounded box drawn in `text-faint`. The border takes the outer ring of cells: 1 row top and bottom, 1 col left and right. Inside a dialog the content starts at col 1; a row with a focus bar puts `▌` in col 1 and its text from col 3, and all other dialog text (title, body, labels, hints) also starts at col 3. One blank col stays inside the right border. A menu's rows start at col 1, their `▌` in col 1 and text from col 3.
- **Focus:** a `▌` accent bar down every text row of the focused item, never an outline (an outline costs 2 rows and 2 cols). Rows with a bar (list rows, text inputs) put the bar in their col 0 and their text at col 2, like cards.
- **Carets:** one full cell (8.4 × 18), in field inputs and in the command line.

### Shape → glyph rules

Every cell the app paints can be read from a frame: a text node gives the glyph, foreground (its fill variable) and bold (`Dusk/Title`) for one cell per character, and a filled frame or rectangle on whole cells gives the background of the cells it covers. The shapes below are the only nodes allowed off the cell grid, and each one maps to cells like this:

| Shape in FigCraft | Cells in the terminal |
|---|---|
| Card `surface` rectangle: 4 px radius, inset 9 px from the top and bottom of its whole-row card frame (`Density=regular`) | Top inset row `▗`, then `▄` up to the last col, then `▖`; bottom inset row `▝`, `▀` repeated, `▘`; across the card's width, foreground = the surface colour, background = what is behind the card. The rows between are the surface colour as background. |
| Card `surface` rectangle at `Density=compact` (square, full height) | Plain background cells, no corner or inset glyphs. |
| Accent bar 4.2 px wide at col 0: Card `bar`, List row `focus bar`, Text input `Input focus bar` | `▌` in `accent` on the row's background, in every row the bar spans. A card's bar spans its text rows only, never the inset rows. |
| Dialog and Menu frame stroke (1 px inside, 6 px / 4 px radius) | `╭─╮` in the top row, `│` in the left and right cols, `╰─╯` in the bottom row, foreground = the stroke colour, on the frame's fill. |
| Caret rectangle (one full cell, `text` fill) | That cell with `text` as background (reverse video), not the terminal cursor. |
| Scrim instance (full-frame `scrim` overlay) | No overlay: the board behind the dialog is repainted with every foreground set to `text-faint` and its fills dropped. |

## Components

All components live on **00 Foundations → Components**, with a light-mode preview under **Components · light preview**. On screen frames, every one of these must be an **instance**.

| Component | Variants | Use it for |
|---|---|---|
| Card | `State=default / selected / done` × `Density=regular / compact` | Every task on a board or list of tasks (`compact` below 30 rows) |
| Column header | `State=normal / active / editing` | Column titles, including in-place rename |
| Column rail | — | Names of columns hidden at narrow widths |
| App header | `Sync=live / error / reloaded` | Row 1 of every board screen |
| Keybar | `Size=wide / narrow` | Bottom-row board key hints (120 vs 80 cols) |
| Key hint | — | One `key label` pair, anywhere a key is shown |
| Dialog | — | Every modal: detail, create, confirm, projects, whiteboards, columns |
| Scrim | — | The faded board behind an open dialog |
| Text input | `Kind=field / title / body` × `State=focused / idle / empty / placeholder` | Every editable text field |
| List row | `State=default / selected / disabled` | Every pickable row: projects, whiteboards, status/column pickers |
| Menu | — | Popovers (status picker, column picker) |
| Badge | — | Board tags such as `daily` |
| Toast | `Kind=success / danger / info` × `Hint` (boolean: `· key label`, e.g. `u undo`) | Transient feedback, bottom-right one row above the keybar |
| Status message | `Kind=success / danger / info / prompt` | One-row messages in a dialog footer or keybar |
| Command line | — | Typed `:` commands (`:wq` and a full-cell caret) |
| Scroll hint | `Dir=up / down` | `↑ N more` / `↓ N more` overflow markers |

### Named Rules
**The Reuse-First Rule.** A new screen is built only from instances of the components above. Before drawing anything, find the matching component in the table. Don't copy an instance and detach it, and don't build a frame that looks like a component (a hand-made row of key/label texts, a fake card, a homemade dialog box).

**The Extend-Don't-Fork Rule.** If no component fits, the fix goes into 00 Foundations: add a variant or property to the closest component (and to the light preview), then use it on the screen. Only create a new component when nothing is close, ask the user first, and add it to the table above in the same change.

**The Override-Only-Content Rule.** On an instance, override text and the variant property, plus the fixed width of a single-line text box when its text changes (glyph count × 8.4 px, see **Typography → Text boxes**). Don't override fills, fonts, padding or spacing. If you need to, the component needs a new variant.

### Key hints

The key hint system is what users rely on most, so it has the strictest rules.

**The Fixed Order Rule.** Within any hint row, hints always appear in this order:

1. **Navigate:** `h j k l select`, `j k move`, `↑↓ move`
2. **Confirm / primary:** `↵ open`, `↵ set`, `↵ create`, `o open`, `:wq save`
3. **Actions** for the current context: `n new`, `e edit`, `s status`, `r rename`, `x delete`, `tab next field`, `ctrl+e $EDITOR`, `SPC t …`
4. **Search / filter:** `/ search`, `/ refine`, `/ filter`
5. **Leave:** `esc close`, `esc cancel`, `esc back`, `esc normal`, `esc clear filter`, and `backspace …` after it in text inputs
6. **Help:** `? help` (board) or `? more` (dialogs), always last
7. Overflow markers such as `↓ 1 more line` sit after the hints, aligned right

**The Board Keybar Rule.** The board keybar is exactly: `h j k l select · ↵ open · / search · SPC t n new · SPC t h l move · SPC more · ? help` (85 cols; edit and delete live under `SPC t`, shown by which-key, so every keybar fits the 116 cols between the gutters). With a filter applied, `/ search` becomes `/ refine` and `SPC more` becomes `esc clear filter`. At 80×24 (`Size=narrow`) it keeps `h j k l select · ↵ open · SPC t n new · SPC more · ? help` (filter: `/ refine · esc clear filter` replace the middle). On the daily board the same slots carry the daily actions: `h j k l select · ↵ open · SPC t n new · SPC d m done · SPC d p promote · SPC d d back · SPC more · ? help` (narrow: `h j k l select · ↵ open · SPC d p promote · SPC more · ? help`). Don't reword or reorder it per screen.

**The Safe-Key-First Rule.** Tier 1 confirmations (one task, one whiteboard, archive) are always `esc keep · y <verb>` (`delete`, `archive`), with the safe key first. `↵` is inert in them: only `y` acts and only `esc` keeps, so a reflex enter can't delete and `↵` never means cancel. Tier 2 (project, non-empty column, clear daily) asks the user to type the name: footer `esc keep` until it matches, then `↵ delete · esc keep`. This is the one exception to "confirm before leave". The discard-changes prompt while editing (`03/unsaved`) is not a confirmation dialog: it reads `discard unsaved changes?  y yes  n no`, and `esc` answers no.

**The Confirm Pattern Rule** (page `09 Confirm`). Every destructive or bulk action uses one Dialog: title = verb + quoted object, body = what will be lost with real counts, then one Status message (`danger` `can't be undone` for permanent loss, `info` for archive), then the footer above. Tier 2 adds a focused Text input above the Status message, which reads `info` `type “name” to delete` until it matches, then `danger` `↵ deletes N tasks for good`.

**The Same-Key Rule.** A key means the same thing everywhere it appears. `↵` confirms, `esc` backs out one level, `?` opens help. Lists use `j k`; use `↑↓` only where the list has a live text filter that would swallow `j k`.

**The Short Footer Rule.** A dialog footer is one row of hints or one Status message. Keys that don't fit go behind `? more`, which ends the row only when something is hidden: a footer that shows every key (`08/list`, `05/typing`) has none. Two exceptions come from the frames: a list dialog with its own actions puts them on a second hint row (`n new  e rename  x delete` under `j k move  ↵ open  / filter  esc close` in `07/actions`; `r restore  a archive old done` in `10/list`), and the discard prompt puts its prompt Status message and its two keys on one row (`03/unsaved`).

## Screens

Every page in `design.figcraft`, with its ticket and frames (120×36 = 1008×648, 80×24 = 672×432, both whole cells). Every screen frame is refit to that grid, with one blank row above the header and below the keybar. `02/breakpoint-rule` is a reference sheet, not a screen, and stays 1280×760. Every frame on pages 01–18 is named `<ticket>/<state>`.

| Page | Ticket | Frames |
|---|---|---|
| 01 Board | [ticket](.scratch/design-tickets/01-board.md) | `01/default`, `01/default-80x24`, `01/sync-error`, `01/sync-reloaded`, `01/filter-active`, `01/more-columns`, `01/overflow-column`, `01/empty-board`, `01/empty-column` |
| 02 Narrow | [ticket](.scratch/design-tickets/02-narrow-80x24.md) | `02/dialog-80x24`, `02/move-right-80x24`, `02/board-80x24`, `02/breakpoint-rule` |
| 03 Task detail | [ticket](.scratch/design-tickets/03-task-detail.md) | `03/view`, `03/no-whiteboards`, `03/empty-title`, `03/unsaved`, `03/copied`, `03/whiteboards`, `03/status-picker`, `03/command-line`, `03/edit-description`, `03/edit-title`, `03/command-line-80x24`, `03/view-80x24` |
| 04 New task | [ticket](.scratch/design-tickets/04-create-task.md) | `04/empty`, `04/created`, `04/empty-title`, `04/command-line`, `04/description`, `04/typing`, `04/column-picker`, `04/typing-80x24`, `04/created-80x24` |
| 05 Search and filter | [ticket](.scratch/design-tickets/05-search-filter.md) | `05/typing`, `05/live-filtered`, `05/no-results`, `05/filter-persisting`, `05/cleared`, `05/typing-80x24`, `05/no-results-80x24`, `05/filter-persisting-80x24`, `05/live-filtered-80x24`, `05/cleared-80x24` |
| 06 Columns | [ticket](.scratch/design-tickets/06-columns.md) | `06/add`, `06/rename-in-place`, `06/delete-non-empty`, `06/duplicate-name`, `06/add-80x24`, `06/rename-in-place-80x24`, `06/delete-non-empty-80x24`, `06/duplicate-name-80x24` |
| 07 Projects | [ticket](.scratch/design-tickets/07-projects.md) | `07/switcher`, `07/single-project`, `07/no-matches`, `07/filtering`, `07/actions`, `07/new`, `07/rename`, `07/delete`, `07/switcher-80x24`, `07/actions-80x24`, `07/filtering-80x24`, `07/no-matches-80x24`, `07/single-project-80x24`, `07/new-80x24`, `07/rename-80x24`, `07/delete-80x24` |
| 08 Whiteboards | [ticket](.scratch/design-tickets/08-whiteboards.md) | `08/list`, `08/empty`, `08/delete`, `08/rename`, `08/open-in-editor`, `08/list-80x24`, `08/open-in-editor-80x24`, `08/rename-80x24`, `08/delete-80x24`, `08/empty-80x24` |
| 09 Confirm | [ticket](.scratch/design-tickets/09-confirm.md) | `09/delete-task`, `09/delete-column`, `09/delete-column-matched`, `09/delete-project`, `09/delete-project-matched`, `09/delete-project-typing`, `09/delete-whiteboard`, `09/clear-daily`, `09/clear-daily-matched`, `09/archive-task`, `09/archive-old`, `09/delete-task-80x24`, `09/delete-project-80x24`, `09/archive-old-80x24` |
| 10 Archive | [ticket](.scratch/design-tickets/10-archive.md) | `10/list`, `10/searching`, `10/no-match`, `10/empty`, `10/detail`, `10/restored`, `10/restore-fallback`, `10/archive-old`, `10/archived-old`, `10/archive-old-none`, `10/list-80x24`, `10/searching-80x24`, `10/detail-80x24`, `10/archive-old-80x24`, `10/empty-80x24` |
| 11 Daily | [ticket](.scratch/design-tickets/11-daily.md) | `11/daily`, `11/empty`, `11/promoted`, `11/promote-project`, `11/no-columns-fallback`, `11/promote-column`, `11/daily-80x24`, `11/empty-80x24`, `11/promote-column-80x24`, `11/promote-project-80x24` |
| 12 Command palette | [ticket](.scratch/design-tickets/12-command-palette.md) | `12/empty`, `12/second-step`, `12/no-match`, `12/typing`, `12/from-dialog`, `12/empty-80x24`, `12/second-step-80x24` |
| 13 Focus mode | [ticket](.scratch/design-tickets/13-focus-mode.md) | `13/doing`, `13/doing-80x24`, `13/advanced`, `13/no-doing` |
| 14 Which-key | [ticket](.scratch/design-tickets/14-which-key.md) | `14/root`, `14/unbound`, `14/pending`, `14/group-task`, `14/group-task-80x24`, `14/root-80x24`, `14/placement-flip` |
| 15 Feedback | [ticket](.scratch/design-tickets/15-feedback.md) | `15/saved`, `15/archived-undo`, `15/moved`, `15/copied-id`, `15/archived-undo-80x24`, `15/store-unreadable-80x24`, `15/save-error`, `15/external-reload`, `15/store-unreadable` |
| 16 Light Dusk | [ticket](.scratch/design-tickets/16-light-dusk.md) | `16/board`, `16/board-80x24`, `16/task-detail`, `16/task-detail-80x24`, `16/projects`, `16/projects-80x24`, `16/palette`, `16/palette-80x24`, `16/focus`, `16/focus-80x24` |
| 17 Theme setting | [ticket](.scratch/design-tickets/17-theme-setting.md) | `17/dusk-ansi16`, `17/terminal`, `17/dusk-light`, `17/dusk`, `17/palette-theme`, `17/palette-theme-80x24` |
| 18 First run | [ticket](.scratch/design-tickets/18-first-run.md) | `18/first-task`, `18/no-projects-80x24`, `18/first-project`, `18/no-projects`, `18/cheat-sheet`, `18/cheat-sheet-80x24` |

## Tokens → lipgloss

One `Palette` struct per theme, one field per token, built with `lipgloss.Color`. `dusk` uses the dark column, `dusk-light` the light column, `terminal` picks one of the two from the detected background (the terminal's OSC 11 answer, else `COLORFGBG`, else dark). When `termenv` reports fewer than 256 colours, use the ansi16 column whatever the theme. `NoColor{}` means leave the attribute unset (`lipgloss.NoColor{}`).

| Token | dark (`dusk`) | light (`dusk-light`) | ansi16 |
|---|---|---|---|
| `bg` | `#17151A` | `#F7F3F6` | `NoColor{}` (never painted) |
| `surface` | `#1F1C23` | `#EEE8ED` | `"0"` |
| `surface-done` | `#1B191E` | `#F3EFF2` | `NoColor{}` |
| `surface-selected` | `#2A2430` | `#E8D9E4` | `"8"` |
| `border` | `#2E2A33` | `#DDD5DC` | `"8"` |
| `accent` | `#D4A5C7` | `#8E4F7E` | `"5"` |
| `text` | `#E2DBE6` | `#221E26` | `"15"` |
| `text-card` | `#C9C2CD` | `#3A333F` | `"7"` |
| `text-muted` | `#A39BA8` | `#5F5765` | `"7"` |
| `text-dim` | `#6E6675` | `#756D7A` | `"8"` |
| `text-faint` | `#4A4550` | `#928B96` | `"8" + Faint(true)` |
| `text-done` | `#4F4955` | `#8F8892` | `"8" + Faint(true)` |
| `meta` | `#9A8296` | `#745A6F` | `"5" + Faint(true)` |
| `success` | `#9FC6A6` | `#3F7D4E` | `"2"` |
| `danger` | `#E28C8C` | `#B44A4A` | `"1"` |

- **Typography:** `Dusk/Title` → `.Bold(true)`; `Dusk/Body` and `Dusk/Hint` → plain. The 13.94 px size is a mockup detail that makes one glyph exactly one 8.4 px cell; terminals have one size.
- **Rounded:** card corners → quarter-block glyphs `▗▖▝▘` in `surface` on `bg`, inside the inset rows `▄`/`▀` (see **Shape → glyph rules**); dialog and menu → `╭─╮ │ ╰─╯` glyphs in `text-faint` on the box fill (see **Shape → glyph rules**).
- **Spacing:** `cell-w`/`cell-h` are mockup units only; in code every size is whole cells.
- **Scrim:** the FigCraft `scrim` variable (70% black overlay) is a mockup aid. In code there is no transparency: re-render the board behind a dialog with every foreground set to `text-faint` and fills dropped.

## Breakpoints

| Width W / height H | Rule |
|---|---|
| Visible columns | N = clamp(floor(W/28), 1, 4); hidden columns listed in the Column rail |
| Dialog size | min(60, W−8) cols; full-height dialogs min(20, H−4) rows, fixed while open; the rest fit their content (see **Layout → Dialogs**) |
| Keybar | `Size=wide` at W ≥ 100, else `Size=narrow` |
| Density | H < 30 → compact: no card insets, 1 blank row between cards |
| Which-key | bottom-right popover one row above the keybar; moves to bottom-left when it would cover the selected card (`14/placement-flip`). At 80×24 both columns sit under it, so it drops onto the keybar instead, leaving a blank row below the selected card. Menus and dialogs are whole rows: the border is drawn inside the padding. The popover is only as tall as its entries plus its 1-row borders (root 7 rows, a group up to 9) and grows upward from a fixed bottom edge |

## Keybind map

Board keys, all built. The bindings table in `internal/ui/leader.go` holds every row except `esc`, `z` and `u`, which the board handles beside it. Board screens may only use keys listed here; dialog keys follow the Fixed Order Rule.

| Keys | Action |
|---|---|
| `h j k l` / arrows | select column / task |
| `g g` · `G` | first · last task |
| `↵` | open task detail |
| `/` | search / refine filter |
| `esc` | back one level: cancels a pending leader, then clears the filter |
| `:` | command palette (page 12) |
| `z` | focus mode (page 13); in it `J K` scroll the description, `z` · `esc` back to the board |
| `?` | help / cheat sheet |
| `q` · `ctrl+c` | quit |
| `SPC t` `n e x a` | task: new, edit, delete, archive |
| `SPC t` `h l` · `k j` | task: move column · reorder |
| `SPC t` `w y` | task: whiteboards, copy id |
| `SPC c` `n r d h l` | column: new, rename, delete, move left/right |
| `SPC p` `p n r x` | project: switch, new, rename, delete |
| `SPC a` `v o` | archive: view, archive old done |
| `SPC d` `d m p X` | daily: back (toggle), done, promote, clear |
| `u` | undo the last move or archive, only while its toast shows (page 15) |

Dialogs: `j k` (or `↑↓` beside a live filter) move, `↵` confirm, `tab` next field, `ctrl+e` $EDITOR, `:wq` save, `y` confirm (Safe-Key-First Rule), `esc` back.

## Do's and Don'ts

### Do:
- **Do** start every new screen by duplicating the closest existing frame on its ticket page, or by placing 00 Foundations instances. Never from an empty frame with hand-drawn parts.
- **Do** bind every colour to a Dusk variable and every text to a `Dusk/*` text style.
- **Do** make every size and position whole cells (multiples of 8.4 px wide, 18 px tall), including single-line text boxes (glyph count × 8.4 px). Only the shapes in **Shape → glyph rules** may sit on half cells.
- **Do** ship the 80×24 variant whenever the layout changes, and check it against `02/breakpoint-rule`.
- **Do** show a key hint for every action available on screen, in the Fixed Order.
- **Do** run a consistency pass across all pages after changing a shared pattern: the same component, key and order everywhere.
- **Do** keep the stored data format untouched: designs may only use fields that exist today (task id/title/description/status/whiteboards/created/updated/archived, columns, projects, daily board).

### Don't:
- **Don't** create a new component, or a frame that imitates one, when a 00 Foundations component exists.
- **Don't** detach instances or override their fills, fonts or spacing.
- **Don't** reorder, rename or reword key hints on one screen only.
- **Don't** use `accent` for decoration, headings or anything **The One Rose Rule** doesn't list.
- **Don't** use shadows, transparency, gradients, icons, emoji, italics, underline or a second font size.
- **Don't** let a full-height dialog change size while it is open, or let a content-sized one leave whole rows or the centre when its content changes.
