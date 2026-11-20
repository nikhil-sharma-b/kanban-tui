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
    fontSize: "14px"
    fontWeight: 700
    lineHeight: "18px"
  body:
    fontFamily: "Menlo, monospace"
    fontSize: "14px"
    fontWeight: 400
    lineHeight: "18px"
  label:
    fontFamily: "Menlo, monospace"
    fontSize: "13px"
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
- Single rose accent (`accent`) used only for focus, selection and prompts.
- Every action on screen has a visible key hint, always in the same order.
- Every screen is assembled from 00 Foundations components. No one-off look-alikes.

## Colors

Plum-tinted neutrals with one rose accent. All colours are FigCraft variables in the **Dusk** collection, with three modes: `dark` (the frontmatter values), `light`, and `ansi16`. Always bind the variable; never paste a hex.

### Primary
- **Dusk Rose** (`accent`): active column header, selection bar `▌`, focused-field bar, prompts, badge fill. Nothing else.

### Neutral
- **Night Plum** (`bg`): screen background.
- **Card Plum** (`surface`): card and menu fill.
- **Spent Plum** (`surface-done`): done-card fill.
- **Lit Plum** (`surface-selected`): selected card/row fill and dialog fill.
- **Rule Plum** (`border`): dialog and menu borders.
- **Text ladder** (`text` → `text-card` → `text-muted` → `text-dim` → `text-faint`, plus `text-done`, `meta`): see each variable's description in FigCraft for its exact role. Keybar keys use `text-muted`, keybar labels use `text-faint`.

### Status
- **Sage** (`success`): saved, created, copied, restored.
- **Clay Red** (`danger`): errors and destructive confirmations.

### Named Rules
**The One Rose Rule.** `accent` marks focus or selection only. If nothing is focused or selected, no rose is on screen apart from the badge.

**The Variables-Only Rule.** Every fill and text colour is bound to a Dusk variable so the `light` and `ansi16` modes work. A raw hex on a screen frame is a bug.

**The Terminal-Theme Rule.** In `theme = terminal` nothing is painted: selected → reverse video, accent → bold, dim/faint/done → faint attribute, danger → ANSI 1, success → ANSI 2. Don't design anything that only works through fill colour.

## Typography

**Font:** Menlo (monospace), one size.

### Hierarchy
- **Title** (Menlo Bold 14/18, text style `Dusk/Title`): dialog titles and the selected card title only.
- **Body** (Menlo Regular 14/18, `Dusk/Body`): card titles, dialog body, field labels, inline meta.
- **Label** (Menlo Regular 13/18, `Dusk/Hint`): key hints, scroll hints, chrome.

### Named Rules
**The One Size Rule.** Hierarchy is colour token + bold. No italics, no underline, no new sizes. Use the three text styles; never set a font by hand.

**The Glyph Whitelist Rule.** Allowed glyphs: `▌ ▀ ▄ ▗ ▖ ▝ ▘ › · ↑ ↓ ● ╭ ╮ ╰ ╯ │ ─ ▏ ◇ ↵ ✓ ✗`. No Nerd Font icons, no emoji.

## Layout

- **Cell grid:** 1 cell = 1 col × 1 row = 8.4 × 18 px. Every size is whole cells; half rows exist only through block glyphs.
- **Frames:** each state at 120×36 cells (1280×760), plus an 80×24 variant whenever layout changes. Name frames `<ticket>/<state>` and `<ticket>/<state>-80x24`, one page per ticket.
- **120×36 rows:** row 1 header · row 3 column headers · rows 4–34 cards · row 36 keybar. Screen gutter 5 cols, column gap 3 cols.
- **Breakpoint rule** (frame `02/breakpoint-rule`): visible columns N = clamp(floor(W/28), 1, 4). Hidden columns are named in the **Column rail**.
- **Dialogs:** min(60, W−8) cells wide, min(20, H−4) rows tall, fixed size. Nothing changes height while the user interacts.
- **Compact density** below 30 rows: card insets dropped, 1 blank row between cards.

## Elevation & Depth

Flat. Terminals have no shadows or transparency. Depth comes from tonal layering only: `bg` → `surface` → `surface-selected`. When a dialog is open, the board behind it is repainted in `text-faint` using the **Scrim** component.

## Shapes

- **Cards:** quarter-block glyphs cut the corners (reads as 4px radius). Text always starts at col 2 in every state, so titles never shift when the selection moves.
- **Dialogs and menus:** `╭╮╰╯` rounded box drawn in `border`.
- **Focus:** a full-height `▌` accent bar, never an outline (an outline costs 2 rows and 2 cols).

## Components

All components live on **00 Foundations → Components**, with a light-mode preview under **Components · light preview**. On screen frames, every one of these must be an **instance**.

| Component | Variants | Use it for |
|---|---|---|
| Card | `State=default / selected / done` | Every task on a board or list of tasks |
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
| Toast | `Kind=success / danger / info` | Transient feedback over the board |
| Status message | `Kind=success / danger / info / prompt` | One-row messages in a dialog footer or keybar |
| Command line | — | Typed `:` commands (`:wq▏`) |
| Scroll hint | `Dir=up / down` | `↑ N more` / `↓ N more` overflow markers |

### Named Rules
**The Reuse-First Rule.** A new screen is built only from instances of the components above. Before drawing anything, find the matching component in the table. Don't copy an instance and detach it, and don't build a frame that looks like a component (a hand-made row of key/label texts, a fake card, a homemade dialog box).

**The Extend-Don't-Fork Rule.** If no component fits, the fix goes into 00 Foundations: add a variant or property to the closest component (and to the light preview), then use it on the screen. Only create a new component when nothing is close, ask the user first, and add it to the table above in the same change.

**The Override-Only-Content Rule.** On an instance, override text and the variant property. Don't override fills, fonts, padding or spacing. If you need to, the component needs a new variant.

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

**The Board Keybar Rule.** The board keybar is exactly: `h j k l select · ↵ open · / search · SPC t n new · SPC t e edit · SPC t h l move · SPC t x delete · SPC more · ? help`. With a filter applied, `/ search` becomes `/ refine` and `SPC more` becomes `esc clear filter`. At 80×24 (`Size=narrow`) it keeps `h j k l select · ↵ open · SPC t n new · SPC more · ? help` (filter: `/ refine · esc clear filter` replace the middle). Don't reword or reorder it per screen.

**The Safe-Key-First Rule.** Destructive confirmations are always `esc keep · y delete`, with the safe key first. This is the one exception to "confirm before leave".

**The Same-Key Rule.** A key means the same thing everywhere it appears. `↵` confirms, `esc` backs out one level, `?` opens help. Lists use `j k`; use `↑↓` only where the list has a live text filter that would swallow `j k`.

**The Short Footer Rule.** A dialog footer is one row: either a Status message or 3–4 hints plus `? more`, never both. Extra keys go behind `?`.

## Do's and Don'ts

### Do:
- **Do** start every new screen by duplicating the closest existing frame on its ticket page, or by placing 00 Foundations instances. Never from an empty frame with hand-drawn parts.
- **Do** bind every colour to a Dusk variable and every text to a `Dusk/*` text style.
- **Do** make every size whole cells (multiples of 8.4 px wide, 18 px tall).
- **Do** ship the 80×24 variant whenever the layout changes, and check it against `02/breakpoint-rule`.
- **Do** show a key hint for every action available on screen, in the Fixed Order.
- **Do** run a consistency pass across all pages after changing a shared pattern: the same component, key and order everywhere.
- **Do** keep the stored data format untouched: designs may only use fields that exist today (task id/title/description/status/whiteboards/created/updated/archived, columns, projects, daily board).

### Don't:
- **Don't** create a new component, or a frame that imitates one, when a 00 Foundations component exists.
- **Don't** detach instances or override their fills, fonts or spacing.
- **Don't** reorder, rename or reword key hints on one screen only.
- **Don't** use `accent` for decoration, headings or anything that isn't focus, selection or a prompt.
- **Don't** use shadows, transparency, gradients, icons, emoji, italics, underline or a second font size.
- **Don't** let a dialog change size while the user interacts with it.
