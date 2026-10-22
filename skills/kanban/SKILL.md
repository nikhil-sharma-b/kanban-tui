---
name: kanban
description: Read and change the user's kanban board through the `kanban` CLI. Use when the user asks to add, find, move, or finish tasks, check what is on their board or daily board, or manage kanban projects and columns.
---

# kanban

The user's kanban board is a TUI that also runs as a CLI: any arguments run one command and exit. `kanban help` lists every command and how tasks, projects and columns are referenced; read it before your first command in a session.

## Working with the board

1. **Read before you write.** Run `kanban project list --json`, then `kanban task list -p PROJECT --json` (or `kanban daily show --json`) so you work from real IDs, project names and column names.
2. **Pass `--json` on every command** and read IDs and fields from the output.
3. **Name the project with `-p`** on `task add`, `task list` and `column` commands. The active project is whatever the user last opened in the TUI, so it shifts under you.
4. **Report back** each change with the task title, project and column it landed in.

## The daily board

The daily board is a separate project (`Daily`) for things that come up during the day. Its columns are fixed: `waiting`, `active`, `next`. Use `kanban daily add` for quick, same-day items, and `kanban daily promote TASK --to PROJECT` when one turns into real project work. Promoting keeps the task's ID and whiteboards.

## Destructive commands

`task delete` and `project delete` cannot be undone. Use them only when the user explicitly asks for that exact deletion. To clear a task off the board otherwise, use `task archive`: it stays restorable with `task restore`.

## Running alongside the TUI

The user may have the TUI open while you work. It reloads your changes within a second, so they need no restart. If a command fails with "workspace was changed by another process", rerun it.
